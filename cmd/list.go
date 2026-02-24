package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/repo"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	Long:  "List all git worktrees for the current repository.",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	// Check if there are any linked worktrees
	hasLinked := false
	for _, wt := range worktrees {
		if wt.Path != info.MainWorktree {
			hasLinked = true
			break
		}
	}

	if !hasLinked {
		fmt.Fprintln(os.Stderr, "No additional worktrees. Create one with: wt create <branch>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BRANCH\tPATH\tMAIN")

	for _, wt := range worktrees {
		isMain := ""
		if wt.Path == info.MainWorktree {
			isMain = "*"
		}
		rel, _ := filepath.Rel(filepath.Dir(info.MainWorktree), wt.Path)
		fmt.Fprintf(w, "%s\t%s\t%s\n", wt.Branch, rel, isMain)
	}

	return w.Flush()
}
