package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// canonPath resolves symlinks so comparisons survive macOS's /tmp -> /private/tmp.
func canonPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return resolved
}

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

// setupNormalRepo creates a plain git repo with one commit and returns its root.
func setupNormalRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")
	runGit(t, root, "commit", "--allow-empty", "-q", "-m", "init")
	return canonPath(t, root)
}

// setupBareLayout creates the gitt bare layout: <project>/.bare + <project>/.git
// pointer file + <project>/.worktrees/main checked out from main.
func setupBareLayout(t *testing.T) string {
	t.Helper()
	source := setupNormalRepo(t)

	project := t.TempDir()
	runGit(t, project, "clone", "--bare", "-q", source, ".bare")
	if err := os.WriteFile(filepath.Join(project, ".git"), []byte("gitdir: ./.bare\n"), 0o644); err != nil {
		t.Fatalf("write .git pointer: %v", err)
	}
	runGit(t, project, "worktree", "add", "-q", ".worktrees/main", "main")
	return canonPath(t, project)
}

// setupBareLayoutViaCloneBare uses the production CloneBare path so tests see
// the real fetch refspec and origin refs that gitt clone produces.
func setupBareLayoutViaCloneBare(t *testing.T) string {
	t.Helper()
	source := setupNormalRepo(t)

	project := t.TempDir()
	if err := CloneBare(source, filepath.Join(project, ".bare")); err != nil {
		t.Fatalf("CloneBare: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, ".git"), []byte("gitdir: ./.bare\n"), 0o644); err != nil {
		t.Fatalf("write .git pointer: %v", err)
	}
	return canonPath(t, project)
}

func TestMainRepoRoot(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) string
		cwd   func(root string) string
	}{
		{
			name:  "normal repo from root",
			setup: setupNormalRepo,
			cwd:   func(root string) string { return root },
		},
		{
			name: "normal repo from linked worktree",
			setup: func(t *testing.T) string {
				root := setupNormalRepo(t)
				runGit(t, root, "worktree", "add", "-q", "-b", "feat", ".wt/feat")
				return root
			},
			cwd: func(root string) string { return filepath.Join(root, ".wt/feat") },
		},
		{
			name:  "bare layout from project root",
			setup: setupBareLayout,
			cwd:   func(root string) string { return root },
		},
		{
			name:  "bare layout from .worktrees/main",
			setup: setupBareLayout,
			cwd:   func(root string) string { return filepath.Join(root, ".worktrees/main") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.setup(t)
			t.Chdir(tc.cwd(want))

			got, err := MainRepoRoot()
			if err != nil {
				t.Fatalf("MainRepoRoot: %v", err)
			}
			if canonPath(t, got) != want {
				t.Errorf("MainRepoRoot = %q, want %q", got, want)
			}
		})
	}
}

func TestWorktreeAdd_SetsUpstreamForExistingRemoteBranch(t *testing.T) {
	project := setupBareLayoutViaCloneBare(t)
	target := filepath.Join(project, ".worktrees/main")

	if err := WorktreeAdd(project, target, "main", false); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}

	out, err := exec.Command("git", "-C", target, "rev-parse", "--abbrev-ref", "main@{upstream}").Output()
	if err != nil {
		t.Fatalf("read upstream: %v", err)
	}
	if got, want := strings.TrimSpace(string(out)), "origin/main"; got != want {
		t.Errorf("upstream = %q, want %q", got, want)
	}
}

func TestWorktreeAdd_LeavesNewBranchUntracked(t *testing.T) {
	project := setupBareLayoutViaCloneBare(t)
	target := filepath.Join(project, ".worktrees/feat")

	if err := WorktreeAdd(project, target, "feat", true); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}

	cmd := exec.Command("git", "-C", target, "rev-parse", "--abbrev-ref", "feat@{upstream}")
	if err := cmd.Run(); err == nil {
		t.Errorf("expected new branch %q to have no upstream, but @{upstream} resolved", "feat")
	}
}
