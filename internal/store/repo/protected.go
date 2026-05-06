package repo

import (
	_ "embed"
	"fmt"
)

//go:embed sql/worktree/reset_protected.sql
var worktreeResetProtectedSQL string

//go:embed sql/worktree/mark_protected.sql
var worktreeMarkProtectedSQL string

// ReconcileProtected re-stamps every worktree row's is_protected flag to
// match the supplied protected branch list. Used by the daemon at
// startup: the user can edit [branches].protected while the daemon is
// down, so the cached flag must catch up to the TOML before any
// rename/remove handler reads it.
//
// The TOML is the source of truth; the column is a cache. Implementation
// is reset-then-mark inside one transaction so a concurrent reader on
// the very next request can never see a half-applied state where rows
// were briefly all 0. The mark step is one UPDATE per protected name
// rather than a dynamic IN clause to keep the SQL embedded as a static
// file; the protected list is small in practice.
func (repo *Repo) ReconcileProtected(protectedBranches []string) error {
	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("begin reconcile protected: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(worktreeResetProtectedSQL); err != nil {
		return fmt.Errorf("reset is_protected: %w", err)
	}

	for _, branch := range protectedBranches {
		if _, err := tx.Exec(worktreeMarkProtectedSQL, branch); err != nil {
			return fmt.Errorf("mark protected (%s): %w", branch, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reconcile protected: %w", err)
	}
	return nil
}
