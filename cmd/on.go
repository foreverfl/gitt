package cmd

import (
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/daemon/client"
	"github.com/foreverfl/gitt/internal/paths"
	"github.com/foreverfl/gitt/internal/process"
	"github.com/foreverfl/gitt/internal/release"
	"github.com/foreverfl/gitt/internal/ui"
	"github.com/spf13/cobra"
)

var onCmd = &cobra.Command{
	Use:   "on",
	Short: "Start the gitt daemon",
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
			if err := client.Ping(sockpath); err == nil {
				fmt.Printf("gitt daemon already running (pid=%d)\n", pid)
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

		pid, err := client.Spawn(self, sockpath, pidpath, logpath)
		if err != nil {
			return err
		}
		ui.Banner(os.Stdout, release.Installed())
		fmt.Printf("gitt daemon started (pid=%d)\n", pid)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(onCmd)
}