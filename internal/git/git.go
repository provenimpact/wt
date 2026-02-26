package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Worktree represents a single git worktree.
type Worktree struct {
	Path   string
	Branch string
	HEAD   string
	Bare   bool
}

// ListWorktrees returns all worktrees for the repository.
// It must be called from within a git repository (main or linked worktree).
func ListWorktrees() ([]Worktree, error) {
	out, err := gitOutput("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			// branch is in refs/heads/... format
			branch := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "detached":
			if current.Branch == "" {
				current.Branch = "(detached)"
			}
		case line == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
		}
	}
	// Handle last entry if no trailing newline
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree at the given path for the given branch.
// If createBranch is true, a new branch is created. When createBranch is true
// and base is non-empty, the new branch starts from the specified base reference
// instead of HEAD.
func AddWorktree(path, branch string, createBranch bool, base string) error {
	args := []string{"worktree", "add"}
	if createBranch {
		args = append(args, "-b", branch, path)
		if base != "" {
			args = append(args, base)
		}
	} else {
		args = append(args, path, branch)
	}

	if err := gitRun(args...); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	return nil
}

// RemoveWorktree removes the worktree at the given path.
func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	if err := gitRun(args...); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}
	return nil
}

// IsDirty returns true if the worktree at the given path has uncommitted changes.
func IsDirty(path string) (bool, error) {
	out, err := gitOutput("-C", path, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("checking dirty state: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// AheadBehind returns the number of commits ahead and behind the upstream.
// Returns (0, 0, nil) if there is no upstream configured.
func AheadBehind(path string) (ahead int, behind int, err error) {
	out, err := gitOutput("-C", path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		// No upstream configured is not an error
		if strings.Contains(err.Error(), "no upstream") || strings.Contains(err.Error(), "unknown revision") {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("checking ahead/behind: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, nil
	}

	ahead, _ = strconv.Atoi(parts[0])
	behind, _ = strconv.Atoi(parts[1])
	return ahead, behind, nil
}

// BranchExists checks if a branch exists locally or remotely.
func BranchExists(name string) (bool, error) {
	// Check local
	err := gitRun("show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}

	// Check remote (any remote)
	out, err := gitOutput("branch", "-r", "--list", "*/"+name)
	if err != nil {
		return false, fmt.Errorf("checking remote branches: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// ListLocalBranches returns sorted local branch names.
func ListLocalBranches() ([]string, error) {
	out, err := gitOutput("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}
	return parseLines(out), nil
}

// ListRemoteBranches returns sorted remote branch names with the remote prefix stripped.
// Deduplicates across remotes and excludes HEAD pointer entries.
func ListRemoteBranches() ([]string, error) {
	out, err := gitOutput("branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("listing remote branches: %w", err)
	}

	seen := make(map[string]bool)
	var branches []string
	for _, line := range parseLines(out) {
		// Skip HEAD pointer entries like "origin/HEAD"
		if strings.HasSuffix(line, "/HEAD") {
			continue
		}
		// Strip remote prefix: "origin/feature-x" -> "feature-x"
		parts := strings.SplitN(line, "/", 2)
		name := line
		if len(parts) == 2 {
			name = parts[1]
		}
		if !seen[name] {
			seen[name] = true
			branches = append(branches, name)
		}
	}

	sort.Strings(branches)
	return branches, nil
}

func parseLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return lines
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
