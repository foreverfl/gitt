DELETE FROM worktrees
WHERE repo_root  = ?
  AND branch_name = ?
RETURNING id;
