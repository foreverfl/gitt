UPDATE worktrees
SET is_protected = 1,
    updated_at   = CURRENT_TIMESTAMP
WHERE branch_name = ?
  AND is_protected = 0;
