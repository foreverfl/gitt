package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/gitx"
)

// handleRegisterWorktree persists a worktree row from the request args.
// The unique constraint on (repo_root, branch_name) and worktree_path is
// enforced by the store; conflicts surface as the error.
func (s *server) handleRegisterWorktree(req daemon.Request) daemon.Response {
	var args daemon.RegisterWorktreeArgs
	if err := daemon.DecodeArgs(req, &args); err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	if args.RepoRoot == "" || args.RepoName == "" || args.BranchName == "" ||
		args.SafeBranchName == "" || args.WorktreePath == "" {
		return daemon.Response{OK: false, Error: "register_worktree: missing required arg"}
	}

	row, err := s.repo.InsertWorktree(args.RepoRoot, args.RepoName, args.BranchName, args.SafeBranchName, args.WorktreePath)
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	data, err := daemon.EncodeData(daemon.WorktreeData{Worktree: row})
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	return daemon.Response{OK: true, Data: data}
}

// handleRenameWorktree atomically renames a branch and its worktree folder,
// then updates the matching row.
//
// Order: branch -m (cheapest, easy to revert) -> worktree move (filesystem,
// hardest to revert) -> store UPDATE. Each step's failure rolls back the
// previous successful steps in reverse.
func (s *server) handleRenameWorktree(req daemon.Request) daemon.Response {
	var args daemon.RenameWorktreeArgs
	if err := daemon.DecodeArgs(req, &args); err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	if args.RepoRoot == "" || args.OldBranch == "" || args.NewBranch == "" {
		return daemon.Response{OK: false, Error: "rename_worktree: missing required arg"}
	}
	if args.OldBranch == args.NewBranch {
		return daemon.Response{OK: false, Error: "old_branch and new_branch are the same"}
	}

	existing, err := s.repo.GetWorktree(args.RepoRoot, args.OldBranch)
	if err != nil {
		return daemon.Response{OK: false, Error: fmt.Sprintf("not registered with gitt: %s", err)}
	}

	newSafe := gitx.SafeBranch(args.NewBranch)
	newPath := gitx.WorktreePath(args.RepoRoot, args.NewBranch)

	if existing.WorktreePath == newPath {
		return daemon.Response{OK: false, Error: fmt.Sprintf("new branch %q resolves to the same path", args.NewBranch)}
	}
	if _, err := os.Stat(newPath); err == nil {
		return daemon.Response{OK: false, Error: fmt.Sprintf("target path already exists: %s", newPath)}
	} else if !errors.Is(err, os.ErrNotExist) {
		return daemon.Response{OK: false, Error: fmt.Sprintf("stat target: %s", err)}
	}

	if err := gitx.BranchRename(args.RepoRoot, args.OldBranch, args.NewBranch); err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}

	if err := gitx.WorktreeMove(args.RepoRoot, existing.WorktreePath, newPath); err != nil {
		if revertErr := gitx.BranchRename(args.RepoRoot, args.NewBranch, args.OldBranch); revertErr != nil {
			return daemon.Response{OK: false, Error: fmt.Sprintf("%s; revert branch -m also failed: %s", err, revertErr)}
		}
		return daemon.Response{OK: false, Error: err.Error()}
	}

	updated, err := s.repo.UpdateWorktree(args.RepoRoot, args.OldBranch, args.NewBranch, newSafe, newPath)
	if err != nil {
		moveErr := gitx.WorktreeMove(args.RepoRoot, newPath, existing.WorktreePath)
		branchErr := gitx.BranchRename(args.RepoRoot, args.NewBranch, args.OldBranch)
		switch {
		case moveErr != nil && branchErr != nil:
			return daemon.Response{OK: false, Error: fmt.Sprintf("%s; revert worktree move failed: %s; revert branch -m failed: %s", err, moveErr, branchErr)}
		case moveErr != nil:
			return daemon.Response{OK: false, Error: fmt.Sprintf("%s; revert worktree move failed: %s", err, moveErr)}
		case branchErr != nil:
			return daemon.Response{OK: false, Error: fmt.Sprintf("%s; revert branch -m failed: %s", err, branchErr)}
		default:
			return daemon.Response{OK: false, Error: err.Error()}
		}
	}

	data, err := daemon.EncodeData(daemon.WorktreeData{Worktree: updated})
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	return daemon.Response{OK: true, Data: data}
}

// handleRelease removes a worktree row from the database. The filesystem
// removal is the caller's responsibility (cmd/remove runs `git worktree
// remove` first); this op only frees the daemon-side bookkeeping.
func (s *server) handleRelease(req daemon.Request) daemon.Response {
	var args daemon.ReleaseArgs
	if err := daemon.DecodeArgs(req, &args); err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	if args.RepoRoot == "" || args.BranchName == "" {
		return daemon.Response{OK: false, Error: "release: missing required arg"}
	}
	if err := s.repo.DeleteWorktree(args.RepoRoot, args.BranchName); err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	return daemon.Response{OK: true}
}

// handleListWorktrees returns every persisted worktree row.
func (s *server) handleListWorktrees(_ daemon.Request) daemon.Response {
	worktrees, err := s.repo.ListWorktrees()
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	data, err := daemon.EncodeData(daemon.ListWorktreesData{Worktrees: worktrees})
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	return daemon.Response{OK: true, Data: data}
}
