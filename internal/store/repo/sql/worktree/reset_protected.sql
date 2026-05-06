UPDATE worktrees
SET is_protected = 0,
    updated_at   = CURRENT_TIMESTAMP
WHERE is_protected = 1;
