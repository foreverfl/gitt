package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	work := t.TempDir()
	t.Chdir(work)

	if err := initCmd.Flags().Set("initial-branch", "trunk"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	t.Cleanup(func() { _ = initCmd.Flags().Set("initial-branch", "") })

	if err := initCmd.RunE(initCmd, []string{"myproj"}); err != nil {
		t.Fatalf("init: %v", err)
	}

	project := filepath.Join(work, "myproj")

	bareInfo, err := os.Stat(filepath.Join(project, ".bare"))
	if err != nil {
		t.Errorf(".bare stat: %v", err)
	} else if !bareInfo.IsDir() {
		t.Errorf(".bare is not a directory")
	}

	pointer, err := os.ReadFile(filepath.Join(project, ".git"))
	if err != nil {
		t.Fatalf("read .git pointer: %v", err)
	}
	if got := strings.TrimSpace(string(pointer)); got != "gitdir: ./.bare" {
		t.Errorf(".git pointer = %q, want %q", got, "gitdir: ./.bare")
	}

	worktreeInfo, err := os.Stat(filepath.Join(project, ".worktrees", "trunk"))
	if err != nil {
		t.Errorf(".worktrees/trunk stat: %v", err)
	} else if !worktreeInfo.IsDir() {
		t.Errorf(".worktrees/trunk is not a directory")
	}

	if _, err := os.Stat(filepath.Join(project, ".worktrees", "trunk", ".git")); err != nil {
		t.Errorf("expected worktree .git link file, got err=%v", err)
	}
}

func TestInitCmd_RefusesExistingGit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	work := t.TempDir()
	t.Chdir(work)

	target := filepath.Join(work, "occupied")
	if err := os.MkdirAll(filepath.Join(target, ".git"), 0o755); err != nil {
		t.Fatalf("seed .git: %v", err)
	}

	err := initCmd.RunE(initCmd, []string{"occupied"})
	if err == nil {
		t.Fatal("expected error when .git exists, got nil")
	}
	if !strings.Contains(err.Error(), ".git") {
		t.Errorf("error = %v, want mention of .git", err)
	}
}
