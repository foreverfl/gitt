package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foreverfl/gitt/internal/daemon/client"
	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [dir]",
	Short: "Initialize a fresh repo in gitt's bare layout (<dir>/.bare + .worktrees/<branch>)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolve target: %w", err)
		}
		if err := os.MkdirAll(absTarget, 0o755); err != nil {
			return fmt.Errorf("create target dir: %w", err)
		}
		for _, name := range []string{".bare", ".git"} {
			if _, err := os.Lstat(filepath.Join(absTarget, name)); err == nil {
				return fmt.Errorf("%q already contains %s; refusing to init", absTarget, name)
			}
		}

		branch, _ := cmd.Flags().GetString("initial-branch")
		if branch == "" {
			branch = gitx.DefaultInitBranch()
		}

		bareDir := filepath.Join(absTarget, ".bare")
		if err := gitx.InitBare(bareDir, branch); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(absTarget, ".git"), []byte("gitdir: ./.bare\n"), 0o644); err != nil {
			return fmt.Errorf("write .git pointer: %w", err)
		}

		worktreePath := filepath.Join(absTarget, ".worktrees", branch)
		if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
			return fmt.Errorf("create .worktrees dir: %w", err)
		}
		if err := gitx.WorktreeAddOrphan(absTarget, worktreePath, branch); err != nil {
			return err
		}

		if err := client.TryRegisterWorktree(absTarget, branch, worktreePath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: daemon registration failed: %v\n", err)
		}

		fmt.Printf("\ninitialized\n  project: %s\n  branch:  %s\n  worktree: %s\n", absTarget, branch, worktreePath)
		fmt.Printf("\nNext:\n  cd %s\n", worktreePath)
		return nil
	},
}

func init() {
	initCmd.Flags().StringP("initial-branch", "b", "", "name of initial branch (default: init.defaultBranch or 'main')")
	rootCmd.AddCommand(initCmd)
}
