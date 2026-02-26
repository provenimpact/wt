package cmd

import (
	"github.com/provenimpact/wt/internal/git"
	"github.com/provenimpact/wt/internal/repo"
)

// completeWorktreeBranches returns existing worktree branch names for tab completion.
func completeWorktreeBranches() []string {
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

// completeLinkedWorktreeBranches returns linked (non-main) worktree branch names for tab completion.
func completeLinkedWorktreeBranches() []string {
	// Same as completeWorktreeBranches — both exclude the main worktree.
	return completeWorktreeBranches()
}
