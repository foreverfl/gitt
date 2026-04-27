// Package worktreeclient holds caller-side helpers that talk to the gitt
// daemon about worktrees. It is split out from internal/worktree so that
// internal/worktree stays a pure leaf (path/layout helpers only) and the
// daemon package can import it without a cycle.
package worktreeclient

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/paths"
	"github.com/foreverfl/gitt/internal/worktree"
)

// Release tells the daemon to drop the worktree row identified by (mainRoot,
// branch). cmd/remove calls this after `git worktree remove` succeeds so the
// daemon's view stays in sync with the filesystem.
func Release(mainRoot, branch string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	response, err := daemon.Call(sockpath, daemon.Request{
		Op: daemon.OpRelease,
		Args: map[string]any{
			"repo_root":   mainRoot,
			"branch_name": branch,
		},
	})
	if err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}

// Register tells the running daemon about a worktree. Returns an error if the
// daemon is unreachable or rejects the request.
func Register(mainRoot, branch, target string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	response, err := daemon.Call(sockpath, daemon.Request{
		Op: daemon.OpRegisterWorktree,
		Args: map[string]any{
			"repo_root":        mainRoot,
			"repo_name":        filepath.Base(mainRoot),
			"branch_name":      branch,
			"safe_branch_name": worktree.SafeBranch(branch),
			"worktree_path":    target,
		},
	})
	if err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}

// TryRegister is the best-effort variant: if the daemon isn't running, it
// silently returns nil. Used by bootstrap commands like `gitt clone` that
// must work before the user has ever invoked `gitt on`.
func TryRegister(mainRoot, branch, target string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	if err := daemon.Ping(sockpath); err != nil {
		if errors.Is(err, daemon.ErrNotRunning) {
			return nil
		}
		return err
	}
	return Register(mainRoot, branch, target)
}
