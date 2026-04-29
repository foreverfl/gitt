package cmd

import (
	"fmt"

	"github.com/foreverfl/gitt/internal/release"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the installed gitt version",
	RunE: func(cmd *cobra.Command, args []string) error {
		installed := release.Installed()
		if installed == "" {
			fmt.Println("unknown (not installed via install.sh)")
			return nil
		}
		fmt.Println(installed)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
