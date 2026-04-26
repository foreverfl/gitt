package cmd

import (
	"fmt"
	"os"

	"github.com/foreverfl/doctree/internal/banner"
	"github.com/foreverfl/doctree/internal/daemon"
	"github.com/foreverfl/doctree/internal/paths"
	"github.com/foreverfl/doctree/internal/process"
	"github.com/spf13/cobra"
)

var onCmd = &cobra.Command{
	Use:   "on",
	Short: "Start the doctree daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Already running? Skip.
		if pid, ok := process.ReadPid(pidpath); ok && process.Alive(pid) {
			if err := daemon.Ping(sockpath); err == nil {
				fmt.Printf("doctree daemon already running (pid=%d)\n", pid)
				return nil
			}
		}

		// Clean stale state from a previous crash.
		_ = os.Remove(pidpath)
		_ = os.Remove(sockpath)

		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate self: %w", err)
		}

		pid, err := daemon.Spawn(self, sockpath, pidpath, logpath)
		if err != nil {
			return err
		}
		banner.Print(os.Stdout, installedVersion())
		fmt.Printf("doctree daemon started (pid=%d)\n", pid)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(onCmd)
}