package repo

import (
	_ "embed"
	"fmt"
	"path/filepath"
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

//go:embed sql/repo/upsert.sql
var repoUpsertSQL string

// Worktree is one row of the worktrees table joined to repos. RepoID is
// the FK; RepoRoot comes from the joined repos.root_path; RepoName is
// derived in Go from RepoRoot's basename and not stored anywhere — keeping
// it on the struct lets list-style consumers display "<repo> / <branch>"
// without re-deriving it themselves. IsProtected reflects the cached
// [branches].protected flag — see schema.sql.
type Worktree struct {
	ID             int64  `json:"id"`
	RepoID         int64  `json:"repo_id"`
	RepoRoot       string `json:"repo_root"`
	RepoName       string `json:"repo_name"`
	BranchName     string `json:"branch_name"`
	SafeBranchName string `json:"safe_branch_name"`
	WorktreePath   string `json:"worktree_path"`
	Status         string `json:"status"`
	IsProtected    bool   `json:"is_protected"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// InsertWorktree persists a new worktree row, upserting the parent repos
// row first so callers that pass only a repo_root path don't have to
// register the repository up-front. Returns the populated record (id,
// repo_id, status, timestamps filled in by SQLite). Returns an error
// when (repo_id, branch_name) or worktree_path is already taken.
func (r *Repo) InsertWorktree(repoRoot, branchName, safeBranchName, worktreePath string) (Worktree, error) {
	worktree := Worktree{
		RepoRoot:       repoRoot,
		RepoName:       filepath.Base(repoRoot),
		BranchName:     branchName,
		SafeBranchName: safeBranchName,
		WorktreePath:   worktreePath,
	}

	repoID, err := r.upsertRepo(repoRoot)
	if err != nil {
		return Worktree{}, err
	}
	worktree.RepoID = repoID

	row := r.db.QueryRow(
		worktreeInsertSQL,
		repoID, branchName, safeBranchName, worktreePath,
	)
	if err := row.Scan(&worktree.ID, &worktree.Status, &worktree.IsProtected, &worktree.CreatedAt, &worktree.UpdatedAt); err != nil {
		return Worktree{}, fmt.Errorf("insert worktree: %w", err)
	}
	return worktree, nil
}

// upsertRepo ensures a repos row exists for repoRoot and returns its id,
// using gitt's <root>/.bare and <root>/.worktrees layout convention to
// fill the path columns. The metadata fields (default_branch, language,
// framework, compose_monorepo) start blank and are filled in later by
// repo configuration commands; on a repeat call the row's updated_at is
// bumped to mark "last referenced" without disturbing the configured
// metadata.
func (r *Repo) upsertRepo(repoRoot string) (int64, error) {
	row := r.db.QueryRow(
		repoUpsertSQL,
		repoRoot,
		filepath.Join(repoRoot, ".bare"),
		filepath.Join(repoRoot, ".worktrees"),
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert repo (%s): %w", repoRoot, err)
	}
	return id, nil
}

// GetWorktree fetches a single worktree row by (repoRoot, branchName).
// Returns sql.ErrNoRows wrapped with context when no row matches.
func (r *Repo) GetWorktree(repoRoot, branchName string) (Worktree, error) {
	row := r.db.QueryRow(worktreeGetSQL, repoRoot, branchName)
	var worktree Worktree
	if err := row.Scan(
		&worktree.ID,
		&worktree.RepoID,
		&worktree.RepoRoot,
		&worktree.BranchName,
		&worktree.SafeBranchName,
		&worktree.WorktreePath,
		&worktree.Status,
		&worktree.IsProtected,
		&worktree.CreatedAt,
		&worktree.UpdatedAt,
	); err != nil {
		return Worktree{}, fmt.Errorf("get worktree (%s, %s): %w", repoRoot, branchName, err)
	}
	worktree.RepoName = filepath.Base(worktree.RepoRoot)
	return worktree, nil
}

// UpdateWorktree renames a worktree row identified by (repoRoot, oldBranch),
// rewriting branch_name, safe_branch_name, and worktree_path to the new
// values. Returns the updated record. The unique constraints on
// (repo_id, branch_name) and worktree_path are enforced by the store; a
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
		&worktree.RepoID,
		&worktree.RepoRoot,
		&worktree.BranchName,
		&worktree.SafeBranchName,
		&worktree.WorktreePath,
		&worktree.Status,
		&worktree.IsProtected,
		&worktree.CreatedAt,
		&worktree.UpdatedAt,
	); err != nil {
		return Worktree{}, fmt.Errorf("update worktree (%s, %s): %w", repoRoot, oldBranch, err)
	}
	worktree.RepoName = filepath.Base(worktree.RepoRoot)
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
			&worktree.RepoID,
			&worktree.RepoRoot,
			&worktree.BranchName,
			&worktree.SafeBranchName,
			&worktree.WorktreePath,
			&worktree.Status,
			&worktree.IsProtected,
			&worktree.CreatedAt,
			&worktree.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan worktree: %w", err)
		}
		worktree.RepoName = filepath.Base(worktree.RepoRoot)
		worktrees = append(worktrees, worktree)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate worktrees: %w", err)
	}
	return worktrees, nil
}
