UPDATE worktrees
SET branch_name      = ?,
    safe_branch_name = ?,
    worktree_path    = ?,
    updated_at       = CURRENT_TIMESTAMP
WHERE repo_id = (SELECT id FROM repos WHERE root_path = ?)
  AND branch_name = ?
RETURNING
  id,
  repo_id,
  (SELECT root_path FROM repos WHERE repos.id = repo_id) AS repo_root,
  branch_name,
  safe_branch_name,
  worktree_path,
  status,
  is_protected,
  created_at,
  updated_at;
