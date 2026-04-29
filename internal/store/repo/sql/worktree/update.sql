UPDATE worktrees
SET branch_name      = ?,
    safe_branch_name = ?,
    worktree_path    = ?,
    updated_at       = CURRENT_TIMESTAMP
WHERE repo_root  = ?
  AND branch_name = ?
RETURNING id, repo_root, repo_name, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at;