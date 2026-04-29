package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

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
	response, err := Call(sockpath, Request{
		Op: OpRegisterWorktree,
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
// typed slice. The daemon serialises []store.Worktree into Response.Data,
// which arrives over the wire as a generic map; this helper json-round-trips
// it back into the typed shape so callers don't have to reach into
// map[string]any themselves.
func ListWorktrees() ([]store.Worktree, error) {
	sockpath, err := paths.SockPath()
	if err != nil {
		return nil, err
	}
	response, err := Call(sockpath, Request{Op: OpListWorktrees})
	if err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, fmt.Errorf("%s", response.Error)
	}
	raw, err := json.Marshal(response.Data["worktrees"])
	if err != nil {
		return nil, fmt.Errorf("encode worktrees: %w", err)
	}
	var worktrees []store.Worktree
	if err := json.Unmarshal(raw, &worktrees); err != nil {
		return nil, fmt.Errorf("decode worktrees: %w", err)
	}
	return worktrees, nil
}

// ReleaseWorktree tells the daemon to drop the worktree row identified by
// (mainRoot, branch). cmd/remove calls this after `git worktree remove`
// succeeds so the daemon's view stays in sync with the filesystem.
func ReleaseWorktree(mainRoot, branch string) error {
	sockpath, err := paths.SockPath()
	if err != nil {
		return err
	}
	response, err := Call(sockpath, Request{
		Op: OpRelease,
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