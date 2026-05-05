INSERT INTO worktrees (
  repo_id, branch_name, safe_branch_name, worktree_path
) VALUES (?, ?, ?, ?)
RETURNING id, status, is_protected, created_at, updated_at;
