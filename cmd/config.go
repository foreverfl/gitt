package cmd

import (
	"fmt"

	"github.com/foreverfl/gitt/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open ~/.gitt/config.toml in your editor",
	Long: "Opens ~/.gitt/config.toml in $VISUAL, $EDITOR, or vi (tried in that order).\n\n" +
		"On first run the file is created from the built-in defaults so\n" +
		"the editor always has something to open.\n\n" +
		"Does not require the gitt daemon to be running.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.EnsureFile()
		if err != nil {
			return fmt.Errorf("ensure config file: %w", err)
		}
		return config.OpenInEditor(cmd.Context(), path)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
