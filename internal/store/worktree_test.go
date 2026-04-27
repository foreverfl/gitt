package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestUpdateWorktree_Happy(t *testing.T) {
	store := openTestStore(t)
	inserted, err := store.InsertWorktree(
		"/repo", "repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo",
	)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	updated, err := store.UpdateWorktree(
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
	store := openTestStore(t)
	_, err := store.UpdateWorktree(
		"/repo", "missing",
		"new", "new", "/repo/.worktrees/new",
	)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateWorktree_BranchNameConflict(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.InsertWorktree(
		"/repo", "repo", "a", "a", "/repo/.worktrees/a",
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := store.InsertWorktree(
		"/repo", "repo", "b", "b", "/repo/.worktrees/b",
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	_, err := store.UpdateWorktree(
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
	store := openTestStore(t)
	if _, err := store.InsertWorktree(
		"/repo", "repo", "a", "a", "/repo/.worktrees/a",
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := store.InsertWorktree(
		"/repo", "repo", "b", "b", "/repo/.worktrees/b",
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	_, err := store.UpdateWorktree(
		"/repo", "a",
		"a-renamed", "a-renamed", "/repo/.worktrees/b",
	)
	if err == nil {
		t.Fatal("expected unique path conflict, got nil")
	}
}

func TestDeleteWorktree_Happy(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.InsertWorktree(
		"/repo", "repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo",
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := store.DeleteWorktree("/repo", "feat/foo"); err != nil {
		t.Fatalf("DeleteWorktree: %v", err)
	}

	if _, err := store.GetWorktree("/repo", "feat/foo"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected row gone (sql.ErrNoRows), got %v", err)
	}
}

func TestDeleteWorktree_NoMatch(t *testing.T) {
	store := openTestStore(t)
	err := store.DeleteWorktree("/repo", "nope")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteWorktree_LeavesOtherRows(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.InsertWorktree(
		"/repo", "repo", "a", "a", "/repo/.worktrees/a",
	); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	if _, err := store.InsertWorktree(
		"/repo", "repo", "b", "b", "/repo/.worktrees/b",
	); err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	if err := store.DeleteWorktree("/repo", "a"); err != nil {
		t.Fatalf("DeleteWorktree a: %v", err)
	}

	if _, err := store.GetWorktree("/repo", "b"); err != nil {
		t.Errorf("row b should still exist, got: %v", err)
	}
}

func TestDeleteWorktree_RestrictsToRepo(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.InsertWorktree(
		"/repoA", "repoA", "shared", "shared", "/repoA/.worktrees/shared",
	); err != nil {
		t.Fatalf("Insert A/shared: %v", err)
	}
	if _, err := store.InsertWorktree(
		"/repoB", "repoB", "shared", "shared", "/repoB/.worktrees/shared",
	); err != nil {
		t.Fatalf("Insert B/shared: %v", err)
	}

	if err := store.DeleteWorktree("/repoA", "shared"); err != nil {
		t.Fatalf("DeleteWorktree A/shared: %v", err)
	}

	if _, err := store.GetWorktree("/repoB", "shared"); err != nil {
		t.Errorf("repoB shared should still exist, got: %v", err)
	}
}
