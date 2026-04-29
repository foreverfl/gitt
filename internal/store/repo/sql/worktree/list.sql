SELECT
  id,
  repo_root,
  repo_name,
  branch_name,
  safe_branch_name,
  worktree_path,
  status,
  created_at,
  updated_at
FROM worktrees
ORDER BY repo_name, branch_name;