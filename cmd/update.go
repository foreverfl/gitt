package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/daemon/client"
	"github.com/foreverfl/gitt/internal/paths"
	"github.com/foreverfl/gitt/internal/process"
	"github.com/foreverfl/gitt/internal/release"
	"github.com/foreverfl/gitt/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gitt to the latest release (preserves runtime data and worktrees)",
	Long: `Fetch and install the latest gitt release.

Update preserves your data:
  1. Stops the running daemon (if any).
  2. Replaces the binary.
  3. Restarts the daemon.

The daemon's SQLite database (~/.gitt/gitt.db) is kept across updates. When
the new binary's schema differs from the on-disk schema, the daemon migrates
the file in place using a backup/swap flow (gitt.db → gitt.db.old → new file
→ rename) so a failed migration leaves the original data recoverable. Your
registered worktree folders on disk are also left untouched.

Use -y/--yes to skip the confirmation ui.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		current := release.Installed()

		fmt.Println("checking latest release...")
		latest, err := release.LatestTag()
		if err != nil {
			return err
		}

		if current != "" && current == latest {
			fmt.Printf("already at latest (%s)\n", current)
			return nil
		}
		if current == "" {
			fmt.Printf("updating to %s\n", latest)
		} else {
			fmt.Printf("updating %s -> %s\n", current, latest)
		}

		sockpath, err := paths.SockPath()
		if err != nil {
			return err
		}
		pidpath, err := paths.PidPath()
		if err != nil {
			return err
		}
		logpath, err := paths.LogPath()
		if err != nil {
			return err
		}

		daemonRunning := false
		if pid, ok := process.ReadPid(pidpath); ok && process.Alive(pid) {
			if err := client.Ping(sockpath); err == nil {
				daemonRunning = true
			}
		}

		if daemonRunning {
			fmt.Println("the running daemon will be stopped and restarted; the database and registered worktrees are kept")
		} else {
			fmt.Println("the database and registered worktrees are kept")
		}
		fmt.Println()

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			ok, err := ui.Confirm("proceed?", false)
			if err != nil {
				if errors.Is(err, ui.ErrNoTTY) {
					return fmt.Errorf("non-interactive shell — pass --yes to confirm")
				}
				return err
			}
			if !ok {
				fmt.Println("aborted.")
				return nil
			}
		}

		selfPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate self: %w", err)
		}
		newPath := selfPath + ".new"

		fmt.Println("downloading...")
		if err := release.Download(latest, newPath); err != nil {
			_ = os.Remove(newPath)
			return err
		}

		if daemonRunning {
			if err := client.Shutdown(sockpath, pidpath, os.Stdout, os.Stderr); err != nil {
				_ = os.Remove(newPath)
				return fmt.Errorf("stop daemon: %w", err)
			}
		}

		if err := os.Rename(newPath, selfPath); err != nil {
			_ = os.Remove(newPath)
			return fmt.Errorf("replace binary: %w", err)
		}

		_ = release.MarkInstalled(latest)

		fmt.Printf("updated to %s\n", latest)

		if daemonRunning {
			fmt.Println("restarting daemon...")
			pid, err := client.Spawn(selfPath, sockpath, pidpath, logpath)
			if err != nil {
				return fmt.Errorf("restart daemon: %w", err)
			}
			fmt.Printf("gitt daemon started (pid=%d)\n", pid)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
