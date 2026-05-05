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

// migrator transforms data from a database opened at the source schema
// version to a database opened at the current (target) schema version. It
// reads from oldDB, derives whatever shape the current schema expects, and
// writes into newDB. Each release ships a migrator for every prior version
// the daemon may encounter in the wild — when schema.sql gains or drops a
// column, we update the migrator bodies in lock-step rather than chaining
// step-by-step migrators across versions, so newDB always speaks current.
type migrator func(oldDB, newDB *sql.DB) error

// migrations: registry of migrators from old schema versions to current.
//
// Version semantics:
// 1. v0:
//    - Fresh database (no schema applied yet)
//    - OR pre-versioning v1 database (schema exists but user_version not set)
//    - Handled in Open() with stamp-only (no migration needed)
//
// 2. v1:
//    - worktrees has denormalised columns:
//      (repo_root, repo_name)
//
// 3. v2:
//    - Same worktrees shape as v1
//    - Still uses (repo_root, repo_name)
//
// 4. v3:
//    - worktrees replaces (repo_root, repo_name) with repo_id (FK)
//    - repos table introduced
//
// 5. v4 (current):
//    - worktrees gains is_protected (cache of TOML
//      [branches].protected); column defaults to 0 and is reconciled
//      from config on daemon startup
//
// Migration strategy:
// - v1 and v2 both use migrateLegacyWorktreesToCurrent()
//   because their data shape is identical
// - Both need backfill into repos, then rewrite worktrees with repo_id
// - v3 uses migrateV3ToCurrent: shape-equivalent copy plus the new
//   is_protected column defaulting to 0 (reconciled later by the daemon)
//
// Note:
// - We do NOT chain step-by-step migrations
// - Each migrator converts directly from old version → current schema
var migrations = map[int]migrator{
	1: migrateLegacyWorktreesToCurrent,
	2: migrateLegacyWorktreesToCurrent,
	3: migrateV3ToCurrent,
}

// migrateLegacyWorktreesToCurrent backfills the v3 repos table from the
// denormalised (repo_root, repo_name) pair on legacy worktrees rows, then
// inserts each worktree under its derived repo_id. Both v1 and v2 store
// worktrees with the same columns so the same body works for either —
// migrateLegacyWorktreesToCurrent is registered under migrations[1] and
// migrations[2]. Repo metadata that legacy schemas never tracked
// (default_branch, language, framework, compose_monorepo) defaults to ""
// or 0 here; the daemon or a follow-up `gitt repos set` flow can fill
// real values in once detected. Legacy ports rows have no equivalent in
// the v3 ports shape and are dropped; that table was never populated in
// either v1 or v2.
func migrateLegacyWorktreesToCurrent(oldDB, newDB *sql.DB) error {
	worktreeRows, err := oldDB.Query(`
		SELECT id, repo_root, repo_name, branch_name, safe_branch_name,
		       worktree_path, status, created_at, updated_at
		  FROM worktrees
		ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read worktrees: %w", err)
	}
	defer worktreeRows.Close()

	// One repos row per unique repo_root encountered; reuse the inserted
	// id for every worktree under that root. Insertion order tracks the
	// first worktree we see per repo, so repos.id sequencing is stable
	// and reproducible from the source data.
	repoIDs := map[string]int64{}
	for worktreeRows.Next() {
		var (
			id                                                           int64
			repoRoot, repoName, branchName, safeBranchName, worktreePath string
			status, createdAt, updatedAt                                 string
		)
		if err := worktreeRows.Scan(&id, &repoRoot, &repoName, &branchName, &safeBranchName, &worktreePath, &status, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan worktree: %w", err)
		}

		repoID, ok := repoIDs[repoRoot]
		if !ok {
			res, err := newDB.Exec(
				`INSERT INTO repos (root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at)
				 VALUES (?, ?, ?, '', '', '', 0, ?, ?)`,
				repoRoot,
				repoRoot+"/.bare",
				repoRoot+"/.worktrees",
				createdAt, updatedAt,
			)
			if err != nil {
				return fmt.Errorf("insert repo for %s: %w", repoRoot, err)
			}
			repoID, err = res.LastInsertId()
			if err != nil {
				return fmt.Errorf("repo id for %s: %w", repoRoot, err)
			}
			_ = repoName // legacy column dropped in v3; presence covered by the SELECT only
			repoIDs[repoRoot] = repoID
		}

		if _, err := newDB.Exec(
			`INSERT INTO worktrees (id, repo_id, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at)
			 VALUES (?,?,?,?,?,?,?,?)`,
			id, repoID, branchName, safeBranchName, worktreePath, status, createdAt, updatedAt,
		); err != nil {
			return fmt.Errorf("insert worktree id=%d: %w", id, err)
		}
	}
	return worktreeRows.Err()
}

// migrateV3ToCurrent copies a v3 database into the v4 shape. v3 already has
// the repos table and worktrees-with-repo_id structure; the only delta is
// the new worktrees.is_protected column, which defaults to 0 here. The
// daemon's startup reconciliation pass (against the user's current
// [branches].protected TOML list) is what sets is_protected to 1 on the
// rows that should actually be protected — we deliberately do NOT read
// config from inside the migrator so the migration stays a pure data move.
func migrateV3ToCurrent(oldDB, newDB *sql.DB) error {
	repoRows, err := oldDB.Query(`
		SELECT id, root_path, bare_path, worktrees_path, default_branch,
		       language, framework, compose_monorepo, created_at, updated_at
		  FROM repos
		ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read repos: %w", err)
	}
	defer repoRows.Close()
	for repoRows.Next() {
		var (
			id                                                                  int64
			rootPath, barePath, worktreesPath, defaultBranch, language, framework string
			composeMonorepo                                                     int
			createdAt, updatedAt                                                string
		)
		if err := repoRows.Scan(&id, &rootPath, &barePath, &worktreesPath, &defaultBranch, &language, &framework, &composeMonorepo, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan repo: %w", err)
		}
		if _, err := newDB.Exec(
			`INSERT INTO repos (id, root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			id, rootPath, barePath, worktreesPath, defaultBranch, language, framework, composeMonorepo, createdAt, updatedAt,
		); err != nil {
			return fmt.Errorf("insert repo id=%d: %w", id, err)
		}
	}
	if err := repoRows.Err(); err != nil {
		return fmt.Errorf("iterate repos: %w", err)
	}

	worktreeRows, err := oldDB.Query(`
		SELECT id, repo_id, branch_name, safe_branch_name, worktree_path,
		       status, created_at, updated_at
		  FROM worktrees
		ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read worktrees: %w", err)
	}
	defer worktreeRows.Close()
	for worktreeRows.Next() {
		var (
			id, repoID                                       int64
			branchName, safeBranchName, worktreePath, status string
			createdAt, updatedAt                             string
		)
		if err := worktreeRows.Scan(&id, &repoID, &branchName, &safeBranchName, &worktreePath, &status, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan worktree: %w", err)
		}
		// is_protected omitted on purpose; schema DEFAULT 0 fills it.
		if _, err := newDB.Exec(
			`INSERT INTO worktrees (id, repo_id, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at)
			 VALUES (?,?,?,?,?,?,?,?)`,
			id, repoID, branchName, safeBranchName, worktreePath, status, createdAt, updatedAt,
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
//  3. Open <dbpath>.old read/write and run migrations[fromVersion], which
//     transforms the source data straight into the current (toVersion)
//     shape inside <dbpath>.new — see the migrator type comment for why
//     it is a single jump rather than a v(N)→v(N+1) chain.
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

	step, ok := migrations[fromVersion]
	if !ok {
		_ = oldDB.Close()
		_ = newDB.Close()
		return abort(fmt.Errorf("no migrator registered for v%d → v%d", fromVersion, toVersion))
	}
	if err := step(oldDB, newDB); err != nil {
		_ = oldDB.Close()
		_ = newDB.Close()
		return abort(fmt.Errorf("migrate v%d → v%d: %w", fromVersion, toVersion, err))
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
