package store

import (
	_ "embed"
	"fmt"
)

//go:embed sql/test/drop.sql
var testDropSQL string

//go:embed sql/test/create.sql
var testCreateSQL string

//go:embed sql/test/insert.sql
var testInsertSQL string

//go:embed sql/test/select.sql
var testSelectSQL string

// Test creates a scratch table, inserts and reads back a row, then drops the
// table. Returns a one-line summary on success. Used by `doctree sqlite` to
// confirm the daemon's database is reachable and writable end-to-end.
func (store *Store) Test() (string, error) {
	if _, err := store.db.Exec(testDropSQL); err != nil {
		return "", fmt.Errorf("predrop: %w", err)
	}
	if _, err := store.db.Exec(testCreateSQL); err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	if _, err := store.db.Exec(testInsertSQL, "hello"); err != nil {
		_, _ = store.db.Exec(testDropSQL)
		return "", fmt.Errorf("insert: %w", err)
	}
	var note string
	if err := store.db.QueryRow(testSelectSQL).Scan(&note); err != nil {
		_, _ = store.db.Exec(testDropSQL)
		return "", fmt.Errorf("select: %w", err)
	}
	if _, err := store.db.Exec(testDropSQL); err != nil {
		return "", fmt.Errorf("drop: %w", err)
	}
	if note != "hello" {
		return "", fmt.Errorf("unexpected value: %q", note)
	}
	return fmt.Sprintf("sqlite OK: created test, inserted+read %q, dropped", note), nil
}