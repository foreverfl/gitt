package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/foreverfl/doctree/internal/daemon"
	"github.com/foreverfl/doctree/internal/paths"
	"github.com/foreverfl/doctree/internal/process"
	"github.com/foreverfl/doctree/internal/prompt"
	"github.com/foreverfl/doctree/internal/release"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update doctree to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		current := installedVersion()

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
			if err := daemon.Ping(sockpath); err == nil {
				daemonRunning = true
			}
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if daemonRunning && !yes {
			ok, err := prompt.Confirm("doctree daemon is running. stop and restart it after update?", true)
			if err != nil {
				if errors.Is(err, prompt.ErrNoTTY) {
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
			if err := daemon.Shutdown(sockpath, pidpath, os.Stdout, os.Stderr); err != nil {
				_ = os.Remove(newPath)
				return fmt.Errorf("stop daemon: %w", err)
			}
		}

		if err := os.Rename(newPath, selfPath); err != nil {
			_ = os.Remove(newPath)
			return fmt.Errorf("replace binary: %w", err)
		}

		if vpath, verr := paths.VersionPath(); verr == nil {
			_ = os.WriteFile(vpath, []byte(latest+"\n"), 0o644)
		}

		fmt.Printf("updated to %s\n", latest)

		if daemonRunning {
			fmt.Println("restarting daemon...")
			pid, err := daemon.Spawn(selfPath, sockpath, pidpath, logpath)
			if err != nil {
				return fmt.Errorf("restart daemon: %w", err)
			}
			fmt.Printf("doctree daemon started (pid=%d)\n", pid)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
