// Package store wraps the SQLite database the gitt daemon owns.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// currentSchemaVersion is the schema version this binary speaks. Bump when
// schema.sql changes shape and register a v(N-1)→v(N) entry in migrations.
// The value is stamped into PRAGMA user_version so older binaries can detect
// "DB is newer than me" and refuse to downgrade silently.
const currentSchemaVersion = 1

// Store wraps the SQLite handle used by the daemon.
// All writes go through BEGIN IMMEDIATE so port allocation is race-free.
type Store struct {
	db *sql.DB
}

// Open opens the SQLite database at path and brings it up to the binary's
// schema version. Behavior by stamped user_version:
//
//   - 0  — fresh database OR a pre-versioning database that already has the
//     v1 schema in place. Apply schema.sql (idempotent) and stamp v1.
//   - == currentSchemaVersion — schema matches; just (re)apply schema.sql as
//     an idempotent guard.
//   - <  currentSchemaVersion — close the handle, run MigrateOnDisk to swap
//     the file in place, then reopen.
//   - >  currentSchemaVersion — refuse to open. The database was written by a
//     newer gitt; downgrading is not supported.
func Open(path string) (*Store, error) {
	db, err := openWithPragmas(path)
	if err != nil {
		return nil, err
	}
	version, err := readUserVersion(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	switch {
	case version > currentSchemaVersion:
		_ = db.Close()
		return nil, fmt.Errorf("db at %s is schema v%d but this gitt only supports up to v%d (downgrade not supported)", path, version, currentSchemaVersion)
	case version == 0 || version == currentSchemaVersion:
		if err := applySchema(db, currentSchemaVersion); err != nil {
			_ = db.Close()
			return nil, err
		}
	default:
		_ = db.Close()
		if err := MigrateOnDisk(path, version, currentSchemaVersion); err != nil {
			return nil, err
		}
		db, err = openWithPragmas(path)
		if err != nil {
			return nil, err
		}
		stamped, err := readUserVersion(db)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		if stamped != currentSchemaVersion {
			_ = db.Close()
			return nil, fmt.Errorf("post-migration db is v%d, want v%d", stamped, currentSchemaVersion)
		}
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

// DB exposes the underlying *sql.DB so internal/store/repo can run queries
// against it. Lifecycle (open, close, migrations) stays on Store; per-table
// CRUD lives on Repo.
func (store *Store) DB() *sql.DB {
	return store.db
}

// openWithPragmas opens path and sets the connection-level pragmas every
// gitt code path expects (WAL journal, foreign keys on). It does not touch
// schema or user_version, so callers can use it both for first-time setup
// and for opening migration scratch files.
func openWithPragmas(path string) (*sql.DB, error) {
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
	return db, nil
}

func readUserVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read user_version: %w", err)
	}
	return version, nil
}

// applySchema runs schema.sql (CREATE TABLE IF NOT EXISTS, so safe on existing
// databases) and stamps user_version. Used both on fresh opens and as the
// final step of MigrateOnDisk so the new file lands with the right version.
func applySchema(db *sql.DB, version int) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	// PRAGMA user_version doesn't accept parameters, so format the literal.
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}
	return nil
}