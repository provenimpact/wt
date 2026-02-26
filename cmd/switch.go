package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/names"
	"github.com/provenimpact/wt/internal/repo"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to a worktree",
	Long:  "Switch to a specific worktree by branch name.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSwitch,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeWorktreeBranches(), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]

	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	sanitized := names.Sanitize(name)
	for _, wt := range worktrees {
		if wt.Branch == name || filepath.Base(wt.Path) == name || filepath.Base(wt.Path) == sanitized {
			fmt.Printf("__wt_cd:%s", wt.Path)
			return nil
		}
	}

	// Not found -- show available worktrees
	fmt.Fprintf(os.Stderr, "Worktree %q not found. Available worktrees:\n", name)
	for _, wt := range worktrees {
		if wt.Path == info.MainWorktree {
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s\n", wt.Branch)
	}
	return fmt.Errorf("worktree %q not found", name)
}
