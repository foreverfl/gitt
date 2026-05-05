SELECT
  worktrees.id,
  worktrees.repo_id,
  repos.root_path AS repo_root,
  worktrees.branch_name,
  worktrees.safe_branch_name,
  worktrees.worktree_path,
  worktrees.status,
  worktrees.is_protected,
  worktrees.created_at,
  worktrees.updated_at
FROM worktrees
JOIN repos ON repos.id = worktrees.repo_id
ORDER BY repos.root_path, worktrees.branch_name;
