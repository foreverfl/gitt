package client

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/paths"
	"github.com/foreverfl/gitt/internal/store"
	"github.com/foreverfl/gitt/internal/worktree"
)

// RegisterWorktree tells the running daemon about a worktree. Returns an error
// if the daemon is unreachable or rejects the request.
func RegisterWorktree(mainRoot, branch, target string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	args, err := daemon.EncodeArgs(daemon.RegisterWorktreeArgs{
		RepoRoot:       mainRoot,
		RepoName:       filepath.Base(mainRoot),
		BranchName:     branch,
		SafeBranchName: worktree.SafeBranch(branch),
		WorktreePath:   target,
	})
	if err != nil {
		return err
	}
	response, err := Call(sockpath, daemon.Request{Op: daemon.OpRegisterWorktree, Args: args})
	if err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}

// TryRegisterWorktree is the best-effort variant: if the daemon isn't running,
// it silently returns nil. Used by bootstrap commands like `gitt clone` that
// must work before the user has ever invoked `gitt on`.
func TryRegisterWorktree(mainRoot, branch, target string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	if err := Ping(sockpath); err != nil {
		if errors.Is(err, ErrNotRunning) {
			return nil
		}
		return err
	}
	return RegisterWorktree(mainRoot, branch, target)
}

// ListWorktrees fetches every persisted worktree row from the daemon as a
// typed slice.
func ListWorktrees() ([]store.Worktree, error) {
	sockpath, err := paths.SockPath()
	if err != nil {
		return nil, err
	}
	response, err := Call(sockpath, daemon.Request{Op: daemon.OpListWorktrees})
	if err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, fmt.Errorf("%s", response.Error)
	}
	var data daemon.ListWorktreesData
	if err := daemon.DecodeData(response, &data); err != nil {
		return nil, fmt.Errorf("decode worktrees: %w", err)
	}
	return data.Worktrees, nil
}

// RenameWorktree asks the daemon to rename a branch and move its worktree
// folder atomically. The daemon performs `git branch -m`, `git worktree
// move`, and the row update; failures roll back in reverse order.
func RenameWorktree(mainRoot, oldBranch, newBranch string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	args, err := daemon.EncodeArgs(daemon.RenameWorktreeArgs{
		RepoRoot:  mainRoot,
		OldBranch: oldBranch,
		NewBranch: newBranch,
	})
	if err != nil {
		return err
	}
	response, err := Call(sockpath, daemon.Request{Op: daemon.OpRenameWorktree, Args: args})
	if err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}

// ReleaseWorktree tells the daemon to drop the worktree row identified by
// (mainRoot, branch). cmd/remove calls this after `git worktree remove`
// succeeds so the daemon's view stays in sync with the filesystem.
func ReleaseWorktree(mainRoot, branch string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	args, err := daemon.EncodeArgs(daemon.ReleaseArgs{
		RepoRoot:   mainRoot,
		BranchName: branch,
	})
	if err != nil {
		return err
	}
	response, err := Call(sockpath, daemon.Request{Op: daemon.OpRelease, Args: args})
	if err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("%s", response.Error)
	}
	return nil
}
