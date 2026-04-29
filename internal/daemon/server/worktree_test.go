package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/store"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func setupBareLayout(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	runGit(t, source, "init", "-q", "-b", "main")
	runGit(t, source, "commit", "--allow-empty", "-q", "-m", "init")

	project := t.TempDir()
	runGit(t, project, "clone", "--bare", "-q", source, ".bare")
	if err := os.WriteFile(filepath.Join(project, ".git"), []byte("gitdir: ./.bare\n"), 0o644); err != nil {
		t.Fatalf("write .git pointer: %v", err)
	}
	runGit(t, project, "worktree", "add", "-q", ".worktrees/main", "main")

	resolved, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	return resolved
}

func newTestServer(t *testing.T) *server {
	t.Helper()
	sqliteStore, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })
	return &server{store: sqliteStore}
}

func mustEncodeArgs(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := daemon.EncodeArgs(v)
	if err != nil {
		t.Fatalf("EncodeArgs: %v", err)
	}
	return raw
}

func TestHandleRenameWorktree_RenamesFolderBranchAndRow(t *testing.T) {
	repoRoot := setupBareLayout(t)
	runGit(t, repoRoot, "worktree", "add", "-q", "-b", "feat/foo", ".worktrees/feat-foo")

	srv := newTestServer(t)
	if _, err := srv.store.InsertWorktree(
		repoRoot, filepath.Base(repoRoot),
		"feat/foo", "feat-foo",
		filepath.Join(repoRoot, ".worktrees/feat-foo"),
	); err != nil {
		t.Fatalf("InsertWorktree: %v", err)
	}

	resp := srv.handleRenameWorktree(daemon.Request{
		Op: daemon.OpRenameWorktree,
		Args: mustEncodeArgs(t, daemon.RenameWorktreeArgs{
			RepoRoot:  repoRoot,
			OldBranch: "feat/foo",
			NewBranch: "feat/bar",
		}),
	})
	if !resp.OK {
		t.Fatalf("rename failed: %s", resp.Error)
	}

	newPath := filepath.Join(repoRoot, ".worktrees/feat-bar")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new path missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".worktrees/feat-foo")); !os.IsNotExist(err) {
		t.Errorf("old path still exists: %v", err)
	}

	cmd := exec.Command("git", "branch", "--list", "feat/bar")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if !strings.Contains(string(out), "feat/bar") {
		t.Errorf("branch feat/bar not found, git output: %q", out)
	}

	row, err := srv.store.GetWorktree(repoRoot, "feat/bar")
	if err != nil {
		t.Fatalf("GetWorktree(feat/bar): %v", err)
	}
	if row.SafeBranchName != "feat-bar" {
		t.Errorf("safe_branch_name = %q, want feat-bar", row.SafeBranchName)
	}
	if row.WorktreePath != newPath {
		t.Errorf("worktree_path = %q, want %s", row.WorktreePath, newPath)
	}
}

func TestHandleRenameWorktree_RejectsTargetExists(t *testing.T) {
	repoRoot := setupBareLayout(t)
	runGit(t, repoRoot, "worktree", "add", "-q", "-b", "a", ".worktrees/a")
	runGit(t, repoRoot, "worktree", "add", "-q", "-b", "b", ".worktrees/b")

	srv := newTestServer(t)
	if _, err := srv.store.InsertWorktree(
		repoRoot, filepath.Base(repoRoot),
		"a", "a", filepath.Join(repoRoot, ".worktrees/a"),
	); err != nil {
		t.Fatalf("InsertWorktree a: %v", err)
	}

	resp := srv.handleRenameWorktree(daemon.Request{
		Op: daemon.OpRenameWorktree,
		Args: mustEncodeArgs(t, daemon.RenameWorktreeArgs{
			RepoRoot:  repoRoot,
			OldBranch: "a",
			NewBranch: "b",
		}),
	})
	if resp.OK {
		t.Fatal("expected rejection when target path exists, got OK")
	}
	if !strings.Contains(resp.Error, "already exists") {
		t.Errorf("expected 'already exists' error, got: %s", resp.Error)
	}
}

func TestHandleRelease_DeletesRow(t *testing.T) {
	repoRoot := setupBareLayout(t)
	srv := newTestServer(t)
	if _, err := srv.store.InsertWorktree(
		repoRoot, filepath.Base(repoRoot),
		"feat/foo", "feat-foo",
		filepath.Join(repoRoot, ".worktrees/feat-foo"),
	); err != nil {
		t.Fatalf("InsertWorktree: %v", err)
	}

	resp := srv.handleRelease(daemon.Request{
		Op: daemon.OpRelease,
		Args: mustEncodeArgs(t, daemon.ReleaseArgs{
			RepoRoot:   repoRoot,
			BranchName: "feat/foo",
		}),
	})
	if !resp.OK {
		t.Fatalf("release failed: %s", resp.Error)
	}

	rows, err := srv.store.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, row := range rows {
		if row.BranchName == "feat/foo" {
			t.Errorf("row still present after release: %+v", row)
		}
	}
}

func TestHandleRelease_RejectsMissing(t *testing.T) {
	repoRoot := setupBareLayout(t)
	srv := newTestServer(t)

	resp := srv.handleRelease(daemon.Request{
		Op: daemon.OpRelease,
		Args: mustEncodeArgs(t, daemon.ReleaseArgs{
			RepoRoot:   repoRoot,
			BranchName: "ghost",
		}),
	})
	if resp.OK {
		t.Fatal("expected release of non-existent row to fail, got OK")
	}
}

func TestHandleRenameWorktree_RejectsUnregistered(t *testing.T) {
	repoRoot := setupBareLayout(t)
	srv := newTestServer(t)

	resp := srv.handleRenameWorktree(daemon.Request{
		Op: daemon.OpRenameWorktree,
		Args: mustEncodeArgs(t, daemon.RenameWorktreeArgs{
			RepoRoot:  repoRoot,
			OldBranch: "nope",
			NewBranch: "new",
		}),
	})
	if resp.OK {
		t.Fatal("expected rejection for unregistered branch, got OK")
	}
	if !strings.Contains(resp.Error, "not registered") {
		t.Errorf("expected 'not registered' error, got: %s", resp.Error)
	}
}
