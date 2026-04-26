package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/foreverfl/doctree/internal/daemon"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "doctree",
	Short:        "doctree — git worktree + docker compose orchestrator",
	Long:         "Coordinates per-branch git worktrees and their docker compose stacks via a small SQLite-backed daemon.",
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "skip confirmation prompts")
}

func Execute() error {
	return rootCmd.Execute()
}

// runtimeDir returns ~/.doctree, creating it on demand.
func runtimeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".doctree")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sockPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doctree.sock"), nil
}

func pidPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doctree.pid"), nil
}

func logPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doctree.log"), nil
}

func dbPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doctree.db"), nil
}

// requireDaemon errors out with an init hint when the daemon isn't reachable.
func requireDaemon() error {
	sockpath, err := sockPath()
	if err != nil {
		return err
	}
	if err := daemon.Ping(sockpath); err != nil {
		if errors.Is(err, daemon.ErrNotRunning) {
			return fmt.Errorf("doctree daemon is not running. start it first: doctree on")
		}
		return err
	}
	return nil
}
