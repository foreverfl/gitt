INSERT INTO worktrees (
  repo_root, repo_name, branch_name, safe_branch_name, worktree_path
) VALUES (?, ?, ?, ?, ?)
RETURNING id, status, created_at, updated_at;