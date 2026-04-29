package repo

import (
	_ "embed"
	"fmt"
)

//go:embed sql/worktree/insert.sql
var worktreeInsertSQL string

//go:embed sql/worktree/list.sql
var worktreeListSQL string

//go:embed sql/worktree/update.sql
var worktreeUpdateSQL string

//go:embed sql/worktree/get.sql
var worktreeGetSQL string

//go:embed sql/worktree/delete.sql
var worktreeDeleteSQL string

// Worktree is a row of the worktrees table. created_at / updated_at are kept
// as the raw SQLite TEXT (ISO-8601 UTC) so callers can format or pass them
// through without imposing a time.Time conversion at the storage layer.
type Worktree struct {
	ID             int64  `json:"id"`
	RepoRoot       string `json:"repo_root"`
	RepoName       string `json:"repo_name"`
	BranchName     string `json:"branch_name"`
	SafeBranchName string `json:"safe_branch_name"`
	WorktreePath   string `json:"worktree_path"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// InsertWorktree persists a new worktree row and returns the populated record
// (id and timestamps filled in by SQLite). Returns an error when (repo_root,
// branch_name) or worktree_path is already taken.
func (r *Repo) InsertWorktree(repoRoot, repoName, branchName, safeBranchName, worktreePath string) (Worktree, error) {
	worktree := Worktree{
		RepoRoot:       repoRoot,
		RepoName:       repoName,
		BranchName:     branchName,
		SafeBranchName: safeBranchName,
		WorktreePath:   worktreePath,
	}
	row := r.db.QueryRow(
		worktreeInsertSQL,
		repoRoot, repoName, branchName, safeBranchName, worktreePath,
	)
	if err := row.Scan(&worktree.ID, &worktree.Status, &worktree.CreatedAt, &worktree.UpdatedAt); err != nil {
		return Worktree{}, fmt.Errorf("insert worktree: %w", err)
	}
	return worktree, nil
}

// GetWorktree fetches a single worktree row by (repoRoot, branchName).
// Returns sql.ErrNoRows wrapped with context when no row matches.
func (r *Repo) GetWorktree(repoRoot, branchName string) (Worktree, error) {
	row := r.db.QueryRow(worktreeGetSQL, repoRoot, branchName)
	var worktree Worktree
	if err := row.Scan(
		&worktree.ID,
		&worktree.RepoRoot,
		&worktree.RepoName,
		&worktree.BranchName,
		&worktree.SafeBranchName,
		&worktree.WorktreePath,
		&worktree.Status,
		&worktree.CreatedAt,
		&worktree.UpdatedAt,
	); err != nil {
		return Worktree{}, fmt.Errorf("get worktree (%s, %s): %w", repoRoot, branchName, err)
	}
	return worktree, nil
}

// UpdateWorktree renames a worktree row identified by (repoRoot, oldBranch),
// rewriting branch_name, safe_branch_name, and worktree_path to the new
// values. Returns the updated record. The unique constraints on
// (repo_root, branch_name) and worktree_path are enforced by the store; a
// conflict with another row surfaces as the error. When no row matches,
// returns sql.ErrNoRows wrapped with context.
func (r *Repo) UpdateWorktree(repoRoot, oldBranch, newBranch, newSafeBranch, newWorktreePath string) (Worktree, error) {
	row := r.db.QueryRow(
		worktreeUpdateSQL,
		newBranch, newSafeBranch, newWorktreePath,
		repoRoot, oldBranch,
	)
	var worktree Worktree
	if err := row.Scan(
		&worktree.ID,
		&worktree.RepoRoot,
		&worktree.RepoName,
		&worktree.BranchName,
		&worktree.SafeBranchName,
		&worktree.WorktreePath,
		&worktree.Status,
		&worktree.CreatedAt,
		&worktree.UpdatedAt,
	); err != nil {
		return Worktree{}, fmt.Errorf("update worktree (%s, %s): %w", repoRoot, oldBranch, err)
	}
	return worktree, nil
}

// DeleteWorktree removes a worktree row identified by (repoRoot, branchName).
// Returns sql.ErrNoRows wrapped with context when no row matches, so callers
// can distinguish "already gone" from real failures.
func (r *Repo) DeleteWorktree(repoRoot, branchName string) error {
	row := r.db.QueryRow(worktreeDeleteSQL, repoRoot, branchName)
	var id int64
	if err := row.Scan(&id); err != nil {
		return fmt.Errorf("delete worktree (%s, %s): %w", repoRoot, branchName, err)
	}
	return nil
}

// ListWorktrees returns every worktree row, sorted by repo then branch.
func (r *Repo) ListWorktrees() ([]Worktree, error) {
	rows, err := r.db.Query(worktreeListSQL)
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []Worktree
	for rows.Next() {
		var worktree Worktree
		if err := rows.Scan(
			&worktree.ID,
			&worktree.RepoRoot,
			&worktree.RepoName,
			&worktree.BranchName,
			&worktree.SafeBranchName,
			&worktree.WorktreePath,
			&worktree.Status,
			&worktree.CreatedAt,
			&worktree.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan worktree: %w", err)
		}
		worktrees = append(worktrees, worktree)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate worktrees: %w", err)
	}
	return worktrees, nil
}