package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/repo"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <branch>",
	Short: "Create a new worktree",
	Long:  "Create a new git worktree for the specified branch in the worktrees directory.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	branch := args[0]

	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	// Check if worktree already exists for this branch
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}
	for _, wt := range worktrees {
		if wt.Branch == branch {
			return fmt.Errorf("worktree for branch %q already exists at %s", branch, wt.Path)
		}
	}

	// Ensure worktrees directory exists
	if err := info.EnsureWorktreesDir(); err != nil {
		return fmt.Errorf("creating worktrees directory: %w", err)
	}

	wtPath := filepath.Join(info.WorktreesDir, branch)

	// Check if branch exists
	exists, err := git.BranchExists(branch)
	if err != nil {
		return err
	}

	if err := git.AddWorktree(wtPath, branch, !exists); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Created worktree for branch %q at %s\n", branch, wtPath)

	// Output cd sentinel to stdout for shell wrapper
	fmt.Printf("__wt_cd:%s", wtPath)
	return nil
}
