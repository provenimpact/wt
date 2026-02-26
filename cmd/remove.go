package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/repo"
	"github.com/provenimpact/wt/internal/tui"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a worktree",
	Long:  "Remove a git worktree. If no name is given, an interactive selector is shown.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRemove,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeLinkedWorktreeBranches(), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal even with uncommitted changes")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	// Filter to linked worktrees only
	var linked []git.Worktree
	for _, wt := range worktrees {
		if wt.Path != info.MainWorktree {
			linked = append(linked, wt)
		}
	}

	if len(linked) == 0 {
		fmt.Fprintln(os.Stderr, "No worktrees to remove.")
		return nil
	}

	var targetPath string
	var targetBranch string

	if len(args) == 1 {
		// Find by name
		name := args[0]
		for _, wt := range linked {
			if wt.Branch == name || filepath.Base(wt.Path) == name {
				targetPath = wt.Path
				targetBranch = wt.Branch
				break
			}
		}
		if targetPath == "" {
			return fmt.Errorf("worktree %q not found", name)
		}
	} else {
		// Interactive selector
		var entries []tui.Entry
		for _, wt := range linked {
			rel, _ := filepath.Rel(filepath.Dir(info.MainWorktree), wt.Path)
			entries = append(entries, tui.Entry{
				Branch: wt.Branch,
				Path:   wt.Path,
				Rel:    rel,
			})
		}

		selected, err := tui.Select(entries)
		if err != nil {
			return err
		}
		if selected == "" {
			return nil // User cancelled
		}
		targetPath = selected
		// Find branch for the selected path
		for _, wt := range linked {
			if wt.Path == selected {
				targetBranch = wt.Branch
				break
			}
		}
	}

	// Check dirty state
	if !removeForce {
		dirty, err := git.IsDirty(targetPath)
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("worktree %q has uncommitted changes; use --force to remove anyway", targetBranch)
		}
	}

	if err := git.RemoveWorktree(targetPath, removeForce); err != nil {
		return err
	}

	// Clean up empty parent directories between the removed path and worktrees dir
	cleanEmptyParents(targetPath, info.WorktreesDir)

	fmt.Fprintf(os.Stderr, "Removed worktree %q\n", targetBranch)
	return nil
}

// cleanEmptyParents walks upward from path toward stopAt, removing empty directories.
func cleanEmptyParents(path, stopAt string) {
	dir := filepath.Dir(path)
	for dir != stopAt && len(dir) > len(stopAt) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
