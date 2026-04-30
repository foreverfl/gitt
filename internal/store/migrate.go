package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// renameSqliteDB renames a SQLite database file together with its WAL and
// SHM sidecars, treating a missing sidecar as a no-op. SQLite resolves the
// WAL/SHM by appending "-wal"/"-shm" to the open path, so renaming only
// the main file leaves the sidecars at the source location and lets them
// shadow the destination: the old WAL frames get applied to the moved-in
// file and reads silently see the previous database's state. We hit this
// during the v1→v2 rollout when an upgrading daemon's gitt.db-wal stayed
// at the original path and made the post-migration db look like v1 again.
func renameSqliteDB(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		src, dst := from+suffix, to+suffix
		if err := os.Rename(src, dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("rename %s: %w", src, err)
		}
	}
	return nil
}

// migrator copies data from a database opened at schema v(N) to one opened
// at schema v(N+1). The destination already has the v(N+1) schema applied,
// so a migrator only needs to read from oldDB and write the equivalent rows
// (renamed columns, split tables, derived defaults, etc.) into newDB.
type migrator func(oldDB, newDB *sql.DB) error

// migrations is the registry of step migrators keyed by source schema
// version. v0 → v1 is handled in Open as a stamp-only path, so it has no
// entry. When schema.sql changes, bump currentSchemaVersion and register
// migrations[currentSchemaVersion-1] = ... here.
var migrations = map[int]migrator{
	1: migrateV1ToV2,
}

// migrateV1ToV2 carries the v1 worktrees rows forward unchanged. v2 adds
// the new `repos` table and re-shapes `ports`: same per-worktree intent as
// v1 (still keyed by worktree_id) but with extra columns (id, name,
// container_port, protocol, timestamps) that v1 has no source for. Rather
// than fabricate defaults for required columns, v1 ports rows are dropped
// — that table was never populated by any CRUD or daemon path, so the drop
// is a no-op for real installs. New tables start empty and the daemon or
// future commands populate them.
func migrateV1ToV2(oldDB, newDB *sql.DB) error {
	worktreeRows, err := oldDB.Query(`
		SELECT id, repo_root, repo_name, branch_name, safe_branch_name,
		       worktree_path, status, created_at, updated_at
		  FROM worktrees`)
	if err != nil {
		return fmt.Errorf("read worktrees: %w", err)
	}
	defer worktreeRows.Close()
	for worktreeRows.Next() {
		var (
			id                                                           int64
			repoRoot, repoName, branchName, safeBranchName, worktreePath string
			status, createdAt, updatedAt                                 string
		)
		if err := worktreeRows.Scan(&id, &repoRoot, &repoName, &branchName, &safeBranchName, &worktreePath, &status, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan worktree: %w", err)
		}
		if _, err := newDB.Exec(
			`INSERT INTO worktrees (id, repo_root, repo_name, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			id, repoRoot, repoName, branchName, safeBranchName, worktreePath, status, createdAt, updatedAt,
		); err != nil {
			return fmt.Errorf("insert worktree id=%d: %w", id, err)
		}
	}
	return worktreeRows.Err()
}

// MigrateOnDisk migrates the SQLite file at dbpath from fromVersion to
// toVersion using a safe backup/swap flow that never overwrites the original
// file in place:
//
//  1. Rename <dbpath> → <dbpath>.old (with WAL/SHM sidecars; see
//     renameSqliteDB). The original is parked at .old; <dbpath> no longer
//     exists, so any crash from this point on leaves an obvious recovery
//     file rather than a half-written primary.
//  2. Create <dbpath>.new and apply the v(toVersion) schema + user_version
//     stamp.
//  3. Open <dbpath>.old read/write and run each registered migrator from
//     fromVersion up to toVersion-1, copying rows into <dbpath>.new.
//  4. On success, rename <dbpath>.new → <dbpath> (again with sidecars), then
//     remove <dbpath>.old plus its WAL/SHM sidecars.
//  5. On any failure, remove <dbpath>.new and restore <dbpath>.old → <dbpath>
//     so the next daemon start sees the original database untouched.
//
// The caller must guarantee no other process holds the database open;
// MigrateOnDisk is invoked from Open after the live handle is closed.
func MigrateOnDisk(dbpath string, fromVersion, toVersion int) error {
	if fromVersion >= toVersion {
		return fmt.Errorf("nothing to migrate: from v%d to v%d", fromVersion, toVersion)
	}
	oldpath := dbpath + ".old"
	newpath := dbpath + ".new"

	// Drop any stale scratch files from a previous failed attempt so they
	// don't interfere with this run.
	_ = os.Remove(oldpath)
	_ = os.Remove(oldpath + "-wal")
	_ = os.Remove(oldpath + "-shm")
	_ = os.Remove(newpath)
	_ = os.Remove(newpath + "-wal")
	_ = os.Remove(newpath + "-shm")

	if err := renameSqliteDB(dbpath, oldpath); err != nil {
		return fmt.Errorf("backup db (%s → %s): %w", dbpath, oldpath, err)
	}

	// abort wraps cause with restore-on-failure semantics: the caller's error
	// is preserved, and we additionally try to put the original file back so
	// the daemon can keep running with pre-migration data.
	abort := func(cause error) error {
		_ = os.Remove(newpath)
		_ = os.Remove(newpath + "-wal")
		_ = os.Remove(newpath + "-shm")
		if restoreErr := renameSqliteDB(oldpath, dbpath); restoreErr != nil {
			return fmt.Errorf("%w (and restoring backup failed: %v)", cause, restoreErr)
		}
		return cause
	}

	newDB, err := openWithPragmas(newpath)
	if err != nil {
		return abort(fmt.Errorf("open new db: %w", err))
	}
	if err := applySchema(newDB, toVersion); err != nil {
		_ = newDB.Close()
		return abort(err)
	}

	oldDB, err := openWithPragmas(oldpath)
	if err != nil {
		_ = newDB.Close()
		return abort(fmt.Errorf("open backup db: %w", err))
	}

	for v := fromVersion; v < toVersion; v++ {
		step, ok := migrations[v]
		if !ok {
			_ = oldDB.Close()
			_ = newDB.Close()
			return abort(fmt.Errorf("no migrator registered for v%d → v%d", v, v+1))
		}
		if err := step(oldDB, newDB); err != nil {
			_ = oldDB.Close()
			_ = newDB.Close()
			return abort(fmt.Errorf("migrate v%d → v%d: %w", v, v+1, err))
		}
	}

	if err := oldDB.Close(); err != nil {
		_ = newDB.Close()
		return abort(fmt.Errorf("close backup db: %w", err))
	}
	if err := newDB.Close(); err != nil {
		return abort(fmt.Errorf("close new db: %w", err))
	}

	if err := renameSqliteDB(newpath, dbpath); err != nil {
		return abort(fmt.Errorf("install new db (%s → %s): %w", newpath, dbpath, err))
	}

	// At this point the migration succeeded; the new primary is in place.
	// Drop the backup and any leftover SQLite sidecars from either path.
	// Failures here are not fatal — leftover files won't corrupt the new db,
	// they'd just clutter the runtime dir.
	_ = os.Remove(oldpath)
	_ = os.Remove(oldpath + "-wal")
	_ = os.Remove(oldpath + "-shm")
	_ = os.Remove(newpath + "-wal")
	_ = os.Remove(newpath + "-shm")
	return nil
}
