package daemon

import (
	"encoding/json"
	"fmt"

	"github.com/foreverfl/gitt/internal/store"
)

// Per-op payload types. Each Op that carries args or response data has its
// shape pinned down here; the wire envelope stays generic (Request.Args /
// Response.Data are json.RawMessage) and these structs are the typed views
// both sides marshal in and out of.

// --- OpRegisterWorktree ---

type RegisterWorktreeArgs struct {
	RepoRoot       string `json:"repo_root"`
	RepoName       string `json:"repo_name"`
	BranchName     string `json:"branch_name"`
	SafeBranchName string `json:"safe_branch_name"`
	WorktreePath   string `json:"worktree_path"`
}

// WorktreeData carries a single worktree row back to the client. Used as the
// data half of OpRegisterWorktree and OpRenameWorktree.
type WorktreeData struct {
	Worktree store.Worktree `json:"worktree"`
}

// --- OpRenameWorktree ---

type RenameWorktreeArgs struct {
	RepoRoot  string `json:"repo_root"`
	OldBranch string `json:"old_branch"`
	NewBranch string `json:"new_branch"`
}

// --- OpRelease ---

type ReleaseArgs struct {
	RepoRoot   string `json:"repo_root"`
	BranchName string `json:"branch_name"`
}

// --- OpListWorktrees ---

type ListWorktreesData struct {
	Worktrees []store.Worktree `json:"worktrees"`
}

// --- OpSqliteTest ---

type SqliteTestData struct {
	Message string `json:"message"`
}

// --- helpers ---

// EncodeArgs marshals v as JSON for use as Request.Args. Fail rate is
// effectively zero for the structs in this file (all fields are JSON-safe);
// the error is propagated for completeness.
func EncodeArgs(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

// DecodeArgs unmarshals req.Args into v. Returns a friendly error when the
// args envelope is missing entirely so handlers don't have to special-case it.
func DecodeArgs(req Request, v any) error {
	if len(req.Args) == 0 {
		return fmt.Errorf("op %s: missing args", req.Op)
	}
	return json.Unmarshal(req.Args, v)
}

// EncodeData marshals v as JSON for use as Response.Data.
func EncodeData(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

// DecodeData unmarshals resp.Data into v. Callers should only call this on
// ops that are documented to set Data; the error here points at protocol
// drift, not user mistakes.
func DecodeData(resp Response, v any) error {
	if len(resp.Data) == 0 {
		return fmt.Errorf("missing response data")
	}
	return json.Unmarshal(resp.Data, v)
}
