// Package gitx wraps the git CLI calls gitt needs. It shells out rather
// than linking a git library to keep the binary small and stay close to
// observable git behavior.
package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns the absolute path to the enclosing git repository's
// top-level directory. Errors if the current working directory is not inside
// a git repo.
func RepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// MainRepoRoot returns the main repository's top-level directory. When called
// from inside a linked worktree, this differs from RepoRoot, which returns the
// worktree's own toplevel.
//
// Also supports gitt's bare layout (<project>/.bare + <project>/.git pointer):
// --git-common-dir resolves to <project>/.bare from anywhere inside, and its
// parent is the project root we want. See gitx_test.go for the locked-in cases.
func MainRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return filepath.Dir(strings.TrimSpace(string(out))), nil
}

// CurrentBranch returns the short branch name of HEAD, or empty string if
// HEAD is detached.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsClean reports whether the working tree has no staged, unstaged, or
// untracked changes.
func IsClean() (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) == 0, nil
}

// HasConflicts reports whether the working tree has any unmerged paths.
// Conflicts can outlive an ongoing operation (e.g. `git stash pop` may leave
// unmerged files behind without setting MERGE_HEAD), so this is checked
// independently from OngoingOp.
func HasConflicts() (bool, error) {
	out, err := exec.Command("git", "diff", "--name-only", "--diff-filter=U").Output()
	if err != nil {
		return false, fmt.Errorf("git diff --diff-filter=U: %w", err)
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// OngoingOp returns the name of the in-progress git operation (merging,
// rebasing, cherry-picking, reverting, bisecting), or an empty string if no
// operation is in progress.
func OngoingOp() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-dir: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}
	switch {
	case exists("rebase-merge"), exists("rebase-apply"):
		return "rebasing", nil
	case exists("MERGE_HEAD"):
		return "merging", nil
	case exists("CHERRY_PICK_HEAD"):
		return "cherry-picking", nil
	case exists("REVERT_HEAD"):
		return "reverting", nil
	case exists("BISECT_LOG"):
		return "bisecting", nil
	}
	return "", nil
}

// WorktreeForBranch returns the absolute path of the worktree currently
// holding the given branch, or an empty string if the branch is not checked
// out anywhere. Useful for giving a friendly message before `git worktree add`
// would fail with "already used by worktree at ...".
func WorktreeForBranch(branch string) (string, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %w", err)
	}
	target := "refs/heads/" + branch
	var currentPath string
	for line := range strings.SplitSeq(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			currentPath = strings.TrimPrefix(line, "worktree ")
		case line == "branch "+target:
			return currentPath, nil
		}
	}
	return "", nil
}

// BranchExists reports whether a local branch with the given name exists.
func BranchExists(branch string) (bool, error) {
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	switch e := err.(type) {
	case nil:
		return true, nil
	case *exec.ExitError:
		if e.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git show-ref: %w", err)
	default:
		return false, fmt.Errorf("git show-ref: %w", err)
	}
}

// WorktreeAdd runs `git worktree add`, streaming git's progress output to
// stderr. When newBranch is true, a new branch is created via `-b`; otherwise
// the existing ref is checked out into the worktree.
//
// workdir is the directory git resolves the repo from; pass "" to use the
// current working directory. clone callers need to set this because cwd at
// clone time is the user's launch dir, not the new project.
//
// Both git's stdout and stderr are routed to stderr so callers like `gitt add
// --print-path` can keep stdout reserved for a single machine-readable line.
// Without this, git's "HEAD is now at <sha> ..." line would mix with the
// printed worktree path and break shell wrappers that `cd "$(gitt add ...)"`.
//
// For existing branches with an `origin/<branch>` counterpart, upstream
// tracking is wired to it so editors (VS Code, etc.) show ahead/behind out of
// the box. New branches are left untracked — they get an upstream when the
// user first runs `git push -u`.
func WorktreeAdd(workdir, target, branch string, newBranch bool) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, target)
	} else {
		args = append(args, target, branch)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	if !newBranch {
		if err := setUpstreamIfOriginExists(target, branch); err != nil {
			return err
		}
	}
	return nil
}

// setUpstreamIfOriginExists wires `branch` to track `origin/<branch>` when
// that remote ref is present. Bare clones leave local refs/heads/* without
// tracking config, so without this `git worktree add <existing>` produces a
// branch that looks "unpublished" to editors even though origin already has it.
func setUpstreamIfOriginExists(worktreePath, branch string) error {
	check := exec.Command("git", "-C", worktreePath, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	if err := check.Run(); err != nil {
		return nil
	}
	set := exec.Command("git", "-C", worktreePath, "branch", "--set-upstream-to=origin/"+branch, branch)
	if out, err := set.CombinedOutput(); err != nil {
		return fmt.Errorf("set upstream origin/%s: %w: %s", branch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// WorktreeRemove runs `git worktree remove <target>`, streaming git's output
// to the caller's stdout/stderr. Fails if the worktree has uncommitted or
// untracked changes; git's own message explains the cause.
func WorktreeRemove(target string) error {
	cmd := exec.Command("git", "worktree", "remove", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

// WorktreePrune runs `git worktree prune` against the repo at workdir, dropping
// orphaned admin records under .git/worktrees/<name> for worktrees whose
// folders were already removed out-of-band. Pass "" to use the current working
// directory.
func WorktreePrune(workdir string) error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree prune: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// BranchRename runs `git branch -m <old> <new>` against the repo at workdir.
// Pass "" to use the current working directory. Daemon callers pass the main
// repo root so the rename hits the right repo regardless of where the daemon
// was started.
func BranchRename(workdir, oldBranch, newBranch string) error {
	cmd := exec.Command("git", "branch", "-m", oldBranch, newBranch)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch -m %s %s: %w: %s", oldBranch, newBranch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// WorktreeMove runs `git worktree move <old> <new>` against the repo at
// workdir. Git refuses to move a worktree from inside itself, so callers must
// run this from the main repo (or any other worktree). Both paths must be
// absolute or resolvable from workdir.
func WorktreeMove(workdir, oldPath, newPath string) error {
	cmd := exec.Command("git", "worktree", "move", oldPath, newPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree move %s %s: %w: %s", oldPath, newPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}
