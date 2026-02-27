package cmd

import (
	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/repo"
)

// completeWorktreeBranches returns all existing worktree branch names for tab completion,
// including the main worktree branch. Used by wt switch.
func completeWorktreeBranches() []string {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return nil
	}
	var names []string
	for _, wt := range worktrees {
		names = append(names, wt.Branch)
	}
	return names
}

// completeLinkedWorktreeBranches returns linked (non-main) worktree branch names for tab completion.
// Used by wt remove — the main worktree should not be removable.
func completeLinkedWorktreeBranches() []string {
	info, err := repo.Resolve()
	if err != nil {
		return nil
	}
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return nil
	}
	var names []string
	for _, wt := range worktrees {
		if wt.Path != info.MainWorktree {
			names = append(names, wt.Branch)
		}
	}
	return names
}
