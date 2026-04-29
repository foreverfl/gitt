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

// withFakeMigration installs a v1→v2 migrator for the duration of the test
// and bumps the in-test "current" view of currentSchemaVersion behavior by
// driving MigrateOnDisk directly. We never mutate the real
// currentSchemaVersion constant — tests for that path call MigrateOnDisk with
// explicit version arguments.
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
	// Simulate a database that was created by an old gitt before user_version
	// existed: same schema as v1, but PRAGMA user_version = 0.
	dbpath := filepath.Join(t.TempDir(), "pre.db")
	raw, err := openWithPragmas(dbpath)
	if err != nil {
		t.Fatalf("openWithPragmas: %v", err)
	}
	if _, err := raw.Exec(schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := raw.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("force user_version=0: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO worktrees (repo_root, repo_name, branch_name, safe_branch_name, worktree_path) VALUES (?, ?, ?, ?, ?)`,
		"/repo", "repo", "main", "main", "/repo/.worktrees/main",
	); err != nil {
		t.Fatalf("seed: %v", err)
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

func TestMigrateOnDisk_HappyPathPreservesData(t *testing.T) {
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")

	// Build a v1 database with a row, then drive a fake v1→v2 migration that
	// copies the row. After MigrateOnDisk, the file should be at v2 and the
	// row should still be readable.
	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := repo.New(store.DB()).InsertWorktree(
		"/repo", "repo", "feat/foo", "feat-foo", "/repo/.worktrees/feat-foo",
	); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	withFakeMigration(t, 1, func(oldDB, newDB *sql.DB) error {
		rows, err := oldDB.Query(`SELECT repo_root, repo_name, branch_name, safe_branch_name, worktree_path FROM worktrees`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var repoRoot, repoName, branchName, safeBranchName, worktreePath string
			if err := rows.Scan(&repoRoot, &repoName, &branchName, &safeBranchName, &worktreePath); err != nil {
				return err
			}
			if _, err := newDB.Exec(
				`INSERT INTO worktrees (repo_root, repo_name, branch_name, safe_branch_name, worktree_path) VALUES (?,?,?,?,?)`,
				repoRoot, repoName, branchName, safeBranchName, worktreePath,
			); err != nil {
				return err
			}
		}
		return rows.Err()
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

	var branch string
	if err := migrated.QueryRow(`SELECT branch_name FROM worktrees WHERE repo_root = ?`, "/repo").Scan(&branch); err != nil {
		t.Fatalf("read row from migrated db: %v", err)
	}
	if branch != "feat/foo" {
		t.Errorf("branch = %q, want feat/foo", branch)
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
		"/repo", "repo", "main", "main", "/repo/.worktrees/main",
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
	dir := t.TempDir()
	dbpath := filepath.Join(dir, "data.db")
	store, err := Open(dbpath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := MigrateOnDisk(dbpath, 1, 2); err == nil {
		t.Fatal("expected error when no migrator is registered, got nil")
	}
	if _, err := os.Stat(dbpath); err != nil {
		t.Fatalf("dbpath should remain accessible: %v", err)
	}
}
