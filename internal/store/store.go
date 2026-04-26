// Package store wraps the SQLite database the doctree daemon owns.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store wraps the SQLite handle used by the daemon.
// All writes go through BEGIN IMMEDIATE so port allocation is race-free.
type Store struct {
	db *sql.DB
}

// Open opens (and migrates) the SQLite database at path. Sets WAL mode and
// enables foreign keys, then applies schema.sql.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %q: %w", pragma, err)
		}
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying handle. Safe to call on a nil-db Store.
func (store *Store) Close() error {
	if store.db == nil {
		return nil
	}
	return store.db.Close()
}

