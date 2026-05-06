package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InitBare runs `git init --bare -b <branch> <dest>`, streaming git's progress
// to stderr. Unlike CloneBare, no remote is configured — callers may add one
// later via `git remote add`.
func InitBare(dest, branch string) error {
	cmd := exec.Command("git", "init", "--bare", "-b", branch, dest)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init --bare: %w", err)
	}
	return nil
}

// WorktreeAddOrphan runs `git worktree add --orphan -b <branch> <target>`,
// creating an empty worktree on an unborn branch. Used by `gitt init` for
// fresh repos with no commits — `git worktree add` without --orphan would
// fail because the branch ref does not yet exist.
//
// Requires git 2.42 or newer.
func WorktreeAddOrphan(workdir, target, branch string) error {
	cmd := exec.Command("git", "worktree", "add", "--orphan", "-b", branch, target)
	cmd.Dir = workdir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add --orphan: %w", err)
	}
	return nil
}

// DefaultInitBranch returns the value of git config `init.defaultBranch` if
// set, falling back to "main". Used by `gitt init` when the user has not
// specified `--initial-branch`.
func DefaultInitBranch() string {
	out, err := exec.Command("git", "config", "--get", "init.defaultBranch").Output()
	if err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	return "main"
}
