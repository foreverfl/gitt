// Package repo holds the per-table CRUD that the daemon runs against the
// SQLite database opened by internal/store. Repo is a thin handle around
// *sql.DB; create one with New(store.DB()) at daemon startup and pass it to
// the request handlers.
package repo

import "database/sql"

// Repo bundles all per-table CRUD methods against a single SQLite handle.
type Repo struct {
	db *sql.DB
}

// New wraps an open SQLite handle. The handle's lifetime (open/close,
// migrations, pragmas) is owned by internal/store; Repo only reads and
// writes through it.
func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}
