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

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Git worktree manager",
	Long:  "A CLI tool for creating, managing, and switching between git worktrees.",
	// When invoked with no subcommand, run the interactive selector.
	RunE: runSelector,
	// Silence default usage/error output so we control what goes to stderr.
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return err
	}
	return nil
}

func runSelector(cmd *cobra.Command, args []string) error {
	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	// Build entry list including main worktree
	var entries []tui.Entry
	hasLinked := false
	for _, wt := range worktrees {
		isMain := wt.Path == info.MainWorktree
		if !isMain {
			hasLinked = true
		}
		rel, _ := filepath.Rel(filepath.Dir(info.MainWorktree), wt.Path)
		entries = append(entries, tui.Entry{
			Branch: wt.Branch,
			Path:   wt.Path,
			Rel:    rel,
			IsMain: isMain,
		})
	}

	if !hasLinked {
		fmt.Fprintln(os.Stderr, "No worktrees found. Create one with: wt create <branch>")
		return nil
	}

	selected, err := tui.Select(entries)
	if err != nil {
		return err
	}

	if selected != "" {
		// Output cd sentinel to stdout for shell wrapper
		fmt.Printf("__wt_cd:%s", selected)
	}
	return nil
}
