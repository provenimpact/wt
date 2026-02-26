package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/names"
	"github.com/provenimpact/wt/internal/repo"
	"github.com/provenimpact/wt/internal/tui"
	"github.com/spf13/cobra"
)

var (
	createBase   string
	createLocal  bool
	createRemote bool
)

var createCmd = &cobra.Command{
	Use:   "create [branch]",
	Short: "Create a new worktree",
	Long:  "Create a new git worktree for the specified branch in the worktrees directory.\nIf no branch is given, an interactive branch selector is shown.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCreate,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeBranchesForCreate(), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	createCmd.Flags().StringVar(&createBase, "base", "", "Base branch/ref for new branch creation")
	createCmd.Flags().BoolVar(&createLocal, "local", false, "Show only local branches in interactive selector")
	createCmd.Flags().BoolVar(&createRemote, "remote", false, "Show only remote branches in interactive selector")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	info, err := repo.Resolve()
	if err != nil {
		return err
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		return err
	}

	var branch string
	var base string

	if len(args) == 1 {
		// Direct creation mode
		branch = args[0]
		base = createBase
	} else {
		// Interactive branch selection
		branch, base, err = interactiveBranchSelect(worktrees)
		if err != nil {
			return err
		}
		if branch == "" {
			return nil // User cancelled
		}
	}

	// Check if worktree already exists for this branch
	for _, wt := range worktrees {
		if wt.Branch == branch {
			return fmt.Errorf("worktree for branch %q already exists at %s", branch, wt.Path)
		}
	}

	// Ensure worktrees directory exists
	if err := info.EnsureWorktreesDir(); err != nil {
		return fmt.Errorf("creating worktrees directory: %w", err)
	}

	// Sanitize branch name for directory path
	dirName := names.Sanitize(branch)
	wtPath := filepath.Join(info.WorktreesDir, dirName)

	// Check if branch exists
	exists, err := git.BranchExists(branch)
	if err != nil {
		return err
	}

	createBranch := !exists
	if base != "" {
		createBranch = true
	}

	if err := git.AddWorktree(wtPath, branch, createBranch, base); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Created worktree for branch %q at %s\n", branch, wtPath)

	// Output cd sentinel to stdout for shell wrapper
	fmt.Printf("__wt_cd:%s", wtPath)
	return nil
}

// interactiveBranchSelect launches the interactive branch selector.
// Returns the selected branch name and base ref (empty if existing branch).
func interactiveBranchSelect(worktrees []git.Worktree) (branch string, base string, err error) {
	// Build the set of branches that already have worktrees
	wtBranches := make(map[string]bool)
	for _, wt := range worktrees {
		wtBranches[wt.Branch] = true
	}

	// Gather branches based on flags
	var entries []tui.BranchEntry

	if !createRemote {
		local, err := git.ListLocalBranches()
		if err != nil {
			return "", "", err
		}
		for _, b := range local {
			entries = append(entries, tui.BranchEntry{
				Name:        b,
				Source:      "local",
				HasWorktree: wtBranches[b],
			})
		}
	}

	if !createLocal {
		remote, err := git.ListRemoteBranches()
		if err != nil {
			return "", "", err
		}
		// Add remote branches not already in list from local
		seen := make(map[string]bool)
		for _, e := range entries {
			seen[e.Name] = true
		}
		for _, b := range remote {
			if !seen[b] {
				entries = append(entries, tui.BranchEntry{
					Name:        b,
					Source:      "remote",
					HasWorktree: wtBranches[b],
				})
			}
		}
	}

	if len(entries) == 0 {
		return "", "", fmt.Errorf("no branches available")
	}

	// Launch branch selector
	selected, err := tui.SelectBranch(entries, "Branches")
	if err != nil {
		return "", "", err
	}
	if selected == "" {
		return "", "", nil // User cancelled
	}

	// Check if the selected branch exists
	exists, err := git.BranchExists(selected)
	if err != nil {
		return "", "", err
	}

	if !exists {
		// New branch — need a base branch selector
		var baseEntries []tui.BranchEntry
		for _, e := range entries {
			if !e.HasWorktree {
				baseEntries = append(baseEntries, tui.BranchEntry{
					Name:   e.Name,
					Source: e.Source,
				})
			}
		}

		baseSelected, err := tui.SelectBranch(baseEntries, "Base branch")
		if err != nil {
			return "", "", err
		}
		if baseSelected == "" {
			return "", "", nil // User cancelled base selection
		}
		return selected, baseSelected, nil
	}

	return selected, "", nil
}

// completeBranchesForCreate returns branch names for tab completion,
// excluding branches that already have worktrees.
func completeBranchesForCreate() []string {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return nil
	}
	wtBranches := make(map[string]bool)
	for _, wt := range worktrees {
		wtBranches[wt.Branch] = true
	}

	var suggestions []string

	local, err := git.ListLocalBranches()
	if err == nil {
		for _, b := range local {
			if !wtBranches[b] {
				suggestions = append(suggestions, b)
			}
		}
	}

	remote, err := git.ListRemoteBranches()
	if err == nil {
		seen := make(map[string]bool)
		for _, s := range suggestions {
			seen[s] = true
		}
		for _, b := range remote {
			if !wtBranches[b] && !seen[b] {
				suggestions = append(suggestions, b)
			}
		}
	}

	return suggestions
}
