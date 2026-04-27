package daemon

import (
	"errors"
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/worktree"
)

// handleRegisterWorktree persists a worktree row from the request args.
// Required args: repo_root, repo_name, branch_name, safe_branch_name,
// worktree_path. The unique constraint on (repo_root, branch_name) and
// worktree_path is enforced by the store; conflicts surface as the error.
func (server *server) handleRegisterWorktree(req Request) Response {
	repoRoot, err := stringArg(req, "repo_root")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	repoName, err := stringArg(req, "repo_name")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	branchName, err := stringArg(req, "branch_name")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	safeBranchName, err := stringArg(req, "safe_branch_name")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	worktreePath, err := stringArg(req, "worktree_path")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	worktree, err := server.store.InsertWorktree(repoRoot, repoName, branchName, safeBranchName, worktreePath)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true, Data: map[string]any{"worktree": worktree}}
}

// handleRenameWorktree atomically renames a branch and its worktree folder,
// then updates the matching row. Required args: repo_root, old_branch,
// new_branch.
//
// Order: branch -m (cheapest, easy to revert) -> worktree move (filesystem,
// hardest to revert) -> store UPDATE. Each step's failure rolls back the
// previous successful steps in reverse.
func (server *server) handleRenameWorktree(req Request) Response {
	// 1. Get the existing row to check the current worktree path and ensure the repo is registered.
	repoRoot, err := stringArg(req, "repo_root")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	oldBranch, err := stringArg(req, "old_branch")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	newBranch, err := stringArg(req, "new_branch")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if oldBranch == newBranch {
		return Response{OK: false, Error: "old_branch and new_branch are the same"}
	}

	// 2. Check the mutation's preconditions
	existing, err := server.store.GetWorktree(repoRoot, oldBranch)
	if err != nil {
		return Response{OK: false, Error: fmt.Sprintf("not registered with gitt: %s", err)}
	}

	newSafe := worktree.SafeBranch(newBranch)
	newPath := worktree.Path(repoRoot, newBranch)

	if existing.WorktreePath == newPath {
		return Response{OK: false, Error: fmt.Sprintf("new branch %q resolves to the same path", newBranch)}
	}
	if _, err := os.Stat(newPath); err == nil {
		return Response{OK: false, Error: fmt.Sprintf("target path already exists: %s", newPath)}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Response{OK: false, Error: fmt.Sprintf("stat target: %s", err)}
	}

	// 3. Rename the branch in git. This is the cheapest operation and easiest to
	if err := gitx.BranchRename(repoRoot, oldBranch, newBranch); err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	// 4. Move the worktree folder.
	if err := gitx.WorktreeMove(repoRoot, existing.WorktreePath, newPath); err != nil {
		if revertErr := gitx.BranchRename(repoRoot, newBranch, oldBranch); revertErr != nil {
			return Response{OK: false, Error: fmt.Sprintf("%s; revert branch -m also failed: %s", err, revertErr)}
		}
		return Response{OK: false, Error: err.Error()}
	}

	// 5. Update the store with the new values.
	updated, err := server.store.UpdateWorktree(repoRoot, oldBranch, newBranch, newSafe, newPath)
	if err != nil {
		moveErr := gitx.WorktreeMove(repoRoot, newPath, existing.WorktreePath)
		branchErr := gitx.BranchRename(repoRoot, newBranch, oldBranch)
		switch {
		case moveErr != nil && branchErr != nil:
			return Response{OK: false, Error: fmt.Sprintf("%s; revert worktree move failed: %s; revert branch -m failed: %s", err, moveErr, branchErr)}
		case moveErr != nil:
			return Response{OK: false, Error: fmt.Sprintf("%s; revert worktree move failed: %s", err, moveErr)}
		case branchErr != nil:
			return Response{OK: false, Error: fmt.Sprintf("%s; revert branch -m failed: %s", err, branchErr)}
		default:
			return Response{OK: false, Error: err.Error()}
		}
	}

	return Response{OK: true, Data: map[string]any{"worktree": updated}}
}

// handleRelease removes a worktree row from the database. The filesystem
// removal is the caller's responsibility (cmd/remove runs `git worktree
// remove` first); this op only frees the daemon-side bookkeeping. Required
// args: repo_root, branch_name.
func (server *server) handleRelease(req Request) Response {
	repoRoot, err := stringArg(req, "repo_root")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	branchName, err := stringArg(req, "branch_name")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := server.store.DeleteWorktree(repoRoot, branchName); err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true}
}

// handleListWorktrees returns every persisted worktree row.
func (server *server) handleListWorktrees(_ Request) Response {
	worktrees, err := server.store.ListWorktrees()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true, Data: map[string]any{"worktrees": worktrees}}
}

func stringArg(req Request, name string) (string, error) {
	raw, ok := req.Args[name]
	if !ok {
		return "", fmt.Errorf("missing arg: %s", name)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("arg %s must be a string", name)
	}
	if value == "" {
		return "", fmt.Errorf("arg %s must not be empty", name)
	}
	return value, nil
}