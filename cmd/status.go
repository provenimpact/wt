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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all worktrees",
	Long:  "Show the status of all worktrees including branch, clean/dirty state, and ahead/behind counts.",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BRANCH\tPATH\tSTATUS\tAHEAD\tBEHIND\tMAIN")

	for _, wt := range worktrees {
		isMain := ""
		if wt.Path == info.MainWorktree {
			isMain = "*"
		}

		rel, _ := filepath.Rel(filepath.Dir(info.MainWorktree), wt.Path)

		status := "clean"
		dirty, err := git.IsDirty(wt.Path)
		if err != nil {
			status = "error"
		} else if dirty {
			status = "dirty"
		}

		ahead, behind, err := git.AheadBehind(wt.Path)
		aheadStr := fmt.Sprintf("%d", ahead)
		behindStr := fmt.Sprintf("%d", behind)
		if err != nil {
			aheadStr = "-"
			behindStr = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", wt.Branch, rel, status, aheadStr, behindStr, isMain)
	}

	return w.Flush()
}
