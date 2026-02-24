package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds resolved repository paths.
type Info struct {
	// MainWorktree is the absolute path to the main (bare/original) worktree.
	MainWorktree string
	// WorktreesDir is the absolute path to the sibling worktrees directory.
	WorktreesDir string
	// RepoName is the base name of the main repository directory.
	RepoName string
}

// Resolve determines the main repository root and worktrees directory.
// It works correctly whether invoked from the main repo or from inside any worktree.
func Resolve() (*Info, error) {
	// git rev-parse --git-common-dir gives us the shared .git directory
	// For the main worktree, this is just ".git"
	// For linked worktrees, this is something like "/path/to/main/.git"
	out, err := gitCommand("rev-parse", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	commonDir := strings.TrimSpace(out)

	// Make absolute
	if !filepath.IsAbs(commonDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		commonDir = filepath.Join(cwd, commonDir)
	}
	commonDir = filepath.Clean(commonDir)

	// The main worktree is the parent of the .git directory
	mainWorktree := filepath.Dir(commonDir)

	repoName := filepath.Base(mainWorktree)
	parent := filepath.Dir(mainWorktree)
	worktreesDir := filepath.Join(parent, repoName+"-worktrees")

	return &Info{
		MainWorktree: mainWorktree,
		WorktreesDir: worktreesDir,
		RepoName:     repoName,
	}, nil
}

// EnsureWorktreesDir creates the worktrees directory if it does not exist.
func (info *Info) EnsureWorktreesDir() error {
	return os.MkdirAll(info.WorktreesDir, 0o755)
}

func gitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
