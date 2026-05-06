package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	// migrate_test exercises Open + MigrateOnDisk end-to-end; it leans on
	// repo only as a fixture writer (Insert/List). The dependency goes
	// store_test → repo → store at compile time, so there's no cycle: this
	// is the test build, the store package itself never imports repo.
	"github.com/foreverfl/gitt/internal/store/repo"
)

// withFakeMigration installs a migrator at the given source version for
// the duration of the test, restoring the previous mapping on cleanup.
// We never mutate the real currentSchemaVersion constant — tests for that
// path call MigrateOnDisk with explicit version arguments.
func withFakeMigration(t *testing.T, fromVersion int, fn migrator) {
	t.Helper()
	prev, hadPrev := migrations[fromVersion]
	migrations[fromVersion] = fn
	t.Cleanup(func() {
		if hadPrev {
			migrations[fromVersion] = prev
		} else {
			delete(migrations, fromVersion)
		}
	})
}

func TestOpen_FreshStampsCurrentVersion(t *testing.T) {
	dbpath := filepath.Join(t.TempDir(), "fresh.db")
	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Errorf("user_version = %d, want %d", version, currentSchemaVersion)
	}
}

func TestOpen_PreVersioningV0Upgrades(t *testing.T) {
	// Simulate a database that lost its user_version stamp: current schema
	// applied, real data inserted, then user_version forced back to 0. Open
	// should take the stamp-only path, leave the data alone, and bump the
	// stamp to currentSchemaVersion.
	dbpath := filepath.Join(t.TempDir(), "pre.db")
	raw, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("openWithPragmas: %v", err)
	}
	if _, err := raw.Exec(schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO repos (root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at)
		 VALUES (?, ?, ?, '', '', '', 0, datetime('now'), datetime('now'))`,
		"/repo", "/repo/.bare", "/repo/.worktrees",
	); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO worktrees (repo_id, branch_name, safe_branch_name, worktree_path)
		 VALUES ((SELECT id FROM repos WHERE root_path = ?), ?, ?, ?)`,
		"/repo", "main", "main", "/repo/.worktrees/main",
	); err != nil {
		t.Fatalf("seed worktree: %v", err)
	}
	if _, err := raw.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("force user_version=0: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	worktrees, err := repo.New(store.DB()).ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0].BranchName != "main" {
		t.Fatalf("seed row lost: %+v", worktrees)
	}

	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Errorf("user_version = %d, want %d", version, currentSchemaVersion)
	}
}

func TestOpen_NewerSchemaRefused(t *testing.T) {
	dbpath := filepath.Join(t.TempDir(), "newer.db")
	raw, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("openWithPragmas: %v", err)
	}
	if _, err := raw.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatalf("force user_version=99: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	if _, err := Open(dbpath); err == nil {
		t.Fatal("expected open to fail on newer-schema db, got nil")
	}
}

// TestMigrateOnDisk_MovesSidecarsAlongsideMainFile guards the v1→v2 bug
// where the daemon's leftover gitt.db-wal/-shm at the original location
// got applied to the freshly migrated dbpath, making it read as the
// pre-migration version. We seed unique placeholder bytes at <dbpath>-wal
// and <dbpath>-shm before the migration and verify those bytes do not
// remain at <dbpath>-wal/-shm afterwards — they must have moved to
// <oldpath>-wal/-shm during the backup rename so they cannot shadow the
// new database.
func TestMigrateOnDisk_MovesSidecarsAlongsideMainFile(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")
	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := repo.New(store.DB()).InsertWorktree(
		"/repo", "main", "main", "/repo/.worktrees/main", false,
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Use suffix-distinct sentinels so a post-migration read can prove the
	// pre-migration content didn't survive at the original sidecar path.
	sentinel := map[string][]byte{
		"-wal": []byte("pre-migration-wal-marker"),
		"-shm": []byte("pre-migration-shm-marker"),
	}
	for suffix, payload := range sentinel {
		if err := os.WriteFile(dbpath+suffix, payload, 0o644); err != nil {
			t.Fatalf("seed sidecar %s: %v", dbpath+suffix, err)
		}
	}

	withFakeMigration(t, 1, func(oldDB, newDB *sql.DB) error { return nil })

	if err := MigrateOnDisk(dbpath, 1, 2); err != nil {
		t.Fatalf("MigrateOnDisk: %v", err)
	}

	for suffix, payload := range sentinel {
		path := dbpath + suffix
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue // sidecar legitimately gone after migration cleanup
		}
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) == string(payload) {
			t.Errorf("%s still holds pre-migration content; sidecar was not moved alongside the backup rename", path)
		}
	}
}

// TestMigrateOnDisk_HappyPathPreservesData drives MigrateOnDisk with a
// fake migrator that copies repos and worktrees rows in the current
// schema shape, mirroring what the real migrator does for legacy data.
// It exercises the framework end-to-end (backup, swap, cleanup) and
// asserts the seeded row is still readable through the new primary file.
func TestMigrateOnDisk_HappyPathPreservesData(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")

	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := repo.New(store.DB()).InsertWorktree(
		"/repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo", false,
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	withFakeMigration(t, 1, func(oldDB, newDB *sql.DB) error {
		repoRows, err := oldDB.Query(`SELECT id, root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at FROM repos`)
		if err != nil {
			return err
		}
		defer repoRows.Close()
		for repoRows.Next() {
			var (
				id                                                                      int64
				rootPath, barePath, worktreesPath, defaultBranch, language, framework   string
				createdAt, updatedAt                                                    string
				composeMonorepo                                                         int
			)
			if err := repoRows.Scan(&id, &rootPath, &barePath, &worktreesPath, &defaultBranch, &language, &framework, &composeMonorepo, &createdAt, &updatedAt); err != nil {
				return err
			}
			if _, err := newDB.Exec(
				`INSERT INTO repos (id, root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at)
				 VALUES (?,?,?,?,?,?,?,?,?,?)`,
				id, rootPath, barePath, worktreesPath, defaultBranch, language, framework, composeMonorepo, createdAt, updatedAt,
			); err != nil {
				return err
			}
		}
		if err := repoRows.Err(); err != nil {
			return err
		}

		wtRows, err := oldDB.Query(`SELECT id, repo_id, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at FROM worktrees`)
		if err != nil {
			return err
		}
		defer wtRows.Close()
		for wtRows.Next() {
			var (
				id, repoID                                            int64
				branchName, safeBranchName, worktreePath, status      string
				createdAt, updatedAt                                  string
			)
			if err := wtRows.Scan(&id, &repoID, &branchName, &safeBranchName, &worktreePath, &status, &createdAt, &updatedAt); err != nil {
				return err
			}
			if _, err := newDB.Exec(
				`INSERT INTO worktrees (id, repo_id, branch_name, safe_branch_name, worktree_path, status, created_at, updated_at)
				 VALUES (?,?,?,?,?,?,?,?)`,
				id, repoID, branchName, safeBranchName, worktreePath, status, createdAt, updatedAt,
			); err != nil {
				return err
			}
		}
		return wtRows.Err()
	})

	if err := MigrateOnDisk(dbpath, 1, 2); err != nil {
		t.Fatalf("MigrateOnDisk: %v", err)
	}

	if _, err := os.Stat(dbpath + ".old"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".old should be removed on success, stat err = %v", err)
	}
	if _, err := os.Stat(dbpath + ".new"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".new should be removed on success, stat err = %v", err)
	}

	migrated, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("reopen migrated db: %v", err)
	}
	defer migrated.Close()

	var version int
	if err := migrated.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 2 {
		t.Errorf("user_version = %d, want 2", version)
	}

	worktrees, err := repo.New(migrated).ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees on migrated db: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0].BranchName != "feat/foo" {
		t.Fatalf("data lost or unexpected: %+v", worktrees)
	}
}

func TestMigrateOnDisk_FailureRestoresOriginal(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")

	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := repo.New(store.DB()).InsertWorktree(
		"/repo", "main", "main", "/repo/.worktrees/main", false,
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	wantErr := errors.New("boom")
	withFakeMigration(t, 1, func(oldDB, newDB *sql.DB) error {
		return wantErr
	})

	err = MigrateOnDisk(dbpath, 1, 2)
	if !errors.Is(err, wantErr) {
		t.Fatalf("MigrateOnDisk error = %v, want wraps %v", err, wantErr)
	}

	if _, err := os.Stat(dbpath); err != nil {
		t.Fatalf("dbpath should be restored, stat err = %v", err)
	}
	if _, err := os.Stat(dbpath + ".new"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".new should be cleaned up on failure, stat err = %v", err)
	}
	if _, err := os.Stat(dbpath + ".old"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".old should be renamed back to dbpath on failure, stat err = %v", err)
	}

	store, err = Open(dbpath)
	if err != nil {
		t.Fatalf("reopen after failed migration: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	worktrees, err := repo.New(store.DB()).ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0].BranchName != "main" {
		t.Fatalf("data lost after failed migration: %+v", worktrees)
	}
}

func TestMigrateOnDisk_MissingMigratorFails(t *testing.T) {
	// Pick a version pair the binary genuinely has no migrator for. As the
	// schema grows we keep moving this past the highest registered step so the
	// test always exercises the "missing migrator" branch instead of accidentally
	// running a real migrator we shipped.
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")
	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := MigrateOnDisk(dbpath, currentSchemaVersion, currentSchemaVersion+1); err == nil {
		t.Fatal("expected error when no migrator is registered, got nil")
	}
	if _, err := os.Stat(dbpath); err != nil {
		t.Fatalf("dbpath should remain accessible: %v", err)
	}
}

// TestMigrateV3ToCurrent_PreservesDataAndDefaultsIsProtected drives the
// v3→current migrator. It seeds a v3 file (repos + worktrees with repo_id
// but no is_protected column) and verifies that after MigrateOnDisk the
// new file carries every row over with is_protected defaulted to 0.
// Reconciliation against the user's [branches].protected list happens at
// daemon startup, not inside the migrator.
func TestMigrateV3ToCurrent_PreservesDataAndDefaultsIsProtected(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "v3.db")

	v3, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("openWithPragmas: %v", err)
	}
	if _, err := v3.Exec(`
		CREATE TABLE repos (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  root_path TEXT NOT NULL UNIQUE,
		  bare_path TEXT NOT NULL,
		  worktrees_path TEXT NOT NULL,
		  default_branch TEXT NOT NULL,
		  language TEXT NOT NULL,
		  framework TEXT NOT NULL DEFAULT '',
		  compose_monorepo INTEGER NOT NULL DEFAULT 0,
		  created_at TEXT NOT NULL,
		  updated_at TEXT NOT NULL
		);
		CREATE TABLE worktrees (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  repo_id INTEGER NOT NULL,
		  branch_name TEXT NOT NULL,
		  safe_branch_name TEXT NOT NULL,
		  worktree_path TEXT NOT NULL,
		  status TEXT NOT NULL DEFAULT 'created',
		  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE,
		  UNIQUE (repo_id, branch_name),
		  UNIQUE (worktree_path)
		);
	`); err != nil {
		t.Fatalf("create v3 schema: %v", err)
	}
	if _, err := v3.Exec(
		`INSERT INTO repos (id, root_path, bare_path, worktrees_path, default_branch, language, framework, compose_monorepo, created_at, updated_at)
		 VALUES (1, ?, ?, ?, 'main', 'go', '', 0, datetime('now'), datetime('now'))`,
		"/repo", "/repo/.bare", "/repo/.worktrees",
	); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	for _, row := range []struct {
		branch, safeBranch, path string
	}{
		{"main", "main", "/repo/.worktrees/main"},
		{"feat/foo", "feat-foo", "/repo/.worktrees/feat-foo"},
	} {
		if _, err := v3.Exec(
			`INSERT INTO worktrees (repo_id, branch_name, safe_branch_name, worktree_path) VALUES (1, ?, ?, ?)`,
			row.branch, row.safeBranch, row.path,
		); err != nil {
			t.Fatalf("seed worktree %s: %v", row.branch, err)
		}
	}
	if _, err := v3.Exec("PRAGMA user_version = 3"); err != nil {
		t.Fatalf("stamp v3: %v", err)
	}
	if err := v3.Close(); err != nil {
		t.Fatalf("close v3: %v", err)
	}

	if err := MigrateOnDisk(dbpath, 3, currentSchemaVersion); err != nil {
		t.Fatalf("MigrateOnDisk: %v", err)
	}

	migrated, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("reopen migrated db: %v", err)
	}
	t.Cleanup(func() { _ = migrated.Close() })

	worktrees, err := repo.New(migrated).ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("worktrees count = %d, want 2", len(worktrees))
	}
	for _, w := range worktrees {
		if w.IsProtected {
			t.Errorf("worktree %s migrated with is_protected=true; expected default 0 (reconciliation runs later)", w.BranchName)
		}
		if w.RepoRoot != "/repo" {
			t.Errorf("worktree %s lost repo_root: got %q", w.BranchName, w.RepoRoot)
		}
	}
}

// TestMigrateLegacyWorktreesToCurrent_BackfillsRepos exercises the real
// migrator that handles v1 and v2 sources. It builds a database with the
// legacy worktrees shape (repo_root, repo_name) and no repos rows, then
// runs MigrateOnDisk and verifies that the new file has one repos row
// per unique repo_root, that worktrees point at it via repo_id, and that
// the data carries over with timestamps preserved.
func TestMigrateLegacyWorktreesToCurrent_BackfillsRepos(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "legacy.db")

	// Build a legacy-shape file by hand: minimal v1/v2 worktrees columns
	// (the migrator's SELECT only reads these). We don't bother creating
	// the legacy `ports` table — the migrator drops it anyway.
	legacy, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("openWithPragmas: %v", err)
	}
	if _, err := legacy.Exec(`
		CREATE TABLE worktrees (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  repo_root TEXT NOT NULL,
		  repo_name TEXT NOT NULL,
		  branch_name TEXT NOT NULL,
		  safe_branch_name TEXT NOT NULL,
		  worktree_path TEXT NOT NULL,
		  status TEXT NOT NULL DEFAULT 'created',
		  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	for _, row := range []struct {
		repoRoot, repoName, branch, safeBranch, path string
	}{
		{"/repoA", "repoA", "feat/foo", "feat-foo", "/repoA/.worktrees/feat-foo"},
		{"/repoA", "repoA", "main", "main", "/repoA/.worktrees/main"},
		{"/repoB", "repoB", "main", "main", "/repoB/.worktrees/main"},
	} {
		if _, err := legacy.Exec(
			`INSERT INTO worktrees (repo_root, repo_name, branch_name, safe_branch_name, worktree_path) VALUES (?,?,?,?,?)`,
			row.repoRoot, row.repoName, row.branch, row.safeBranch, row.path,
		); err != nil {
			t.Fatalf("seed legacy worktree %s/%s: %v", row.repoRoot, row.branch, err)
		}
	}
	if _, err := legacy.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatalf("stamp v2: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy: %v", err)
	}

	if err := MigrateOnDisk(dbpath, 2, currentSchemaVersion); err != nil {
		t.Fatalf("MigrateOnDisk: %v", err)
	}

	migrated, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("reopen migrated db: %v", err)
	}
	t.Cleanup(func() { _ = migrated.Close() })

	var repoCount int
	if err := migrated.QueryRow(`SELECT count(*) FROM repos`).Scan(&repoCount); err != nil {
		t.Fatalf("count repos: %v", err)
	}
	if repoCount != 2 {
		t.Errorf("repos count = %d, want 2 (one per unique repo_root)", repoCount)
	}

	worktrees, err := repo.New(migrated).ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 3 {
		t.Fatalf("worktrees count = %d, want 3", len(worktrees))
	}
	for _, w := range worktrees {
		if w.RepoID == 0 {
			t.Errorf("worktree %s missing repo_id: %+v", w.BranchName, w)
		}
		if w.RepoRoot != "/repoA" && w.RepoRoot != "/repoB" {
			t.Errorf("worktree %s has unexpected repo_root %q", w.BranchName, w.RepoRoot)
		}
	}
}
