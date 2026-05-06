package repo

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foreverfl/gitt/internal/store"
)

func openTestRepo(t *testing.T) *Repo {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return New(s.DB())
}

// TestInsertWorktree_StampsProtectedFlag verifies that the isProtected
// argument is round-tripped through the INSERT and read back from the
// returned row, so callers can tell at register time whether the worktree
// they just created is governed by the protection policy.
func TestInsertWorktree_StampsProtectedFlag(t *testing.T) {
	r := openTestRepo(t)

	protected, err := r.InsertWorktree(
		"/repo", "main", "main", "/repo/.worktrees/main", true,
	)
	if err != nil {
		t.Fatalf("Insert main: %v", err)
	}
	if !protected.IsProtected {
		t.Errorf("inserted protected row IsProtected = false, want true")
	}

	regular, err := r.InsertWorktree(
		"/repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo", false,
	)
	if err != nil {
		t.Fatalf("Insert feat/foo: %v", err)
	}
	if regular.IsProtected {
		t.Errorf("inserted unprotected row IsProtected = true, want false")
	}

	// Read back through Get to confirm the flag persists past the
	// INSERT ... RETURNING path.
	got, err := r.GetWorktree("/repo", "main")
	if err != nil {
		t.Fatalf("GetWorktree: %v", err)
	}
	if !got.IsProtected {
		t.Errorf("GetWorktree(main).IsProtected = false, want true")
	}
}

func TestUpdateWorktree_Happy(t *testing.T) {
	r := openTestRepo(t)
	inserted, err := r.InsertWorktree(
		"/repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo", false,
	)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	updated, err := r.UpdateWorktree(
		"/repo", "feat/foo",
		"feat/bar", "feat-bar", "/repo/.worktrees/feat-bar",
	)
	if err != nil {
		t.Fatalf("UpdateWorktree: %v", err)
	}

	if updated.ID != inserted.ID {
		t.Errorf("id changed: got %d, want %d", updated.ID, inserted.ID)
	}
	if updated.BranchName != "feat/bar" {
		t.Errorf("branch_name = %q, want feat/bar", updated.BranchName)
	}
	if updated.SafeBranchName != "feat-bar" {
		t.Errorf("safe_branch_name = %q, want feat-bar", updated.SafeBranchName)
	}
	if updated.WorktreePath != "/repo/.worktrees/feat-bar" {
		t.Errorf("worktree_path = %q, want /repo/.worktrees/feat-bar", updated.WorktreePath)
	}
	if updated.RepoRoot != "/repo" || updated.RepoName != "repo" {
		t.Errorf("repo fields drifted: %+v", updated)
	}
	if updated.Status != inserted.Status {
		t.Errorf("status drifted: %q -> %q", inserted.Status, updated.Status)
	}
}

func TestUpdateWorktree_NoMatch(t *testing.T) {
	r := openTestRepo(t)
	_, err := r.UpdateWorktree(
		"/repo", "missing",
		"new", "new", "/repo/.worktrees/new",
	)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateWorktree_BranchNameConflict(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.InsertWorktree(
		"/repo", "a", "a", "/repo/.worktrees/a", false,
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := r.InsertWorktree(
		"/repo", "b", "b", "/repo/.worktrees/b", false,
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	_, err := r.UpdateWorktree(
		"/repo", "a",
		"b", "b", "/repo/.worktrees/b-other",
	)
	if err == nil {
		t.Fatal("expected unique constraint error, got nil")
	}
	if !strings.Contains(err.Error(), "UNIQUE") && !strings.Contains(err.Error(), "constraint") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}
}

func TestUpdateWorktree_PathConflict(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.InsertWorktree(
		"/repo", "a", "a", "/repo/.worktrees/a", false,
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := r.InsertWorktree(
		"/repo", "b", "b", "/repo/.worktrees/b", false,
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	_, err := r.UpdateWorktree(
		"/repo", "a",
		"a-renamed", "a-renamed", "/repo/.worktrees/b",
	)
	if err == nil {
		t.Fatal("expected unique path conflict, got nil")
	}
}

func TestDeleteWorktree_Happy(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.InsertWorktree(
		"/repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo", false,
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := r.DeleteWorktree("/repo", "feat/foo"); err != nil {
		t.Fatalf("DeleteWorktree: %v", err)
	}

	if _, err := r.GetWorktree("/repo", "feat/foo"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected row gone (sql.ErrNoRows), got %v", err)
	}
}

func TestDeleteWorktree_NoMatch(t *testing.T) {
	r := openTestRepo(t)
	err := r.DeleteWorktree("/repo", "nope")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteWorktree_LeavesOtherRows(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.InsertWorktree(
		"/repo", "a", "a", "/repo/.worktrees/a", false,
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := r.InsertWorktree(
		"/repo", "b", "b", "/repo/.worktrees/b", false,
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	if err := r.DeleteWorktree("/repo", "a"); err != nil {
		t.Fatalf("DeleteWorktree a: %v", err)
	}

	if _, err := r.GetWorktree("/repo", "b"); err != nil {
		t.Errorf("row b should still exist, got: %v", err)
	}
}

func TestDeleteWorktree_RestrictsToRepo(t *testing.T) {
	r := openTestRepo(t)
	if _, err := r.InsertWorktree(
		"/repoA", "shared", "shared", "/repoA/.worktrees/shared", false,
	); err != nil {
		t.Fatalf("Insert A/shared: %v", err)
	}
	if _, err := r.InsertWorktree(
		"/repoB", "shared", "shared", "/repoB/.worktrees/shared", false,
	); err != nil {
		t.Fatalf("Insert B/shared: %v", err)
	}

	if err := r.DeleteWorktree("/repoA", "shared"); err != nil {
		t.Fatalf("DeleteWorktree A/shared: %v", err)
	}

	if _, err := r.GetWorktree("/repoB", "shared"); err != nil {
		t.Errorf("repoB shared should still exist, got: %v", err)
	}
}
