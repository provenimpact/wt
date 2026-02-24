// Feature: worktree-management
// Spec version: 1.0.0
// Generated from: spec.adoc
//
// Spec coverage:
//   WT-007: Worktree placement convention (<repo>-worktrees/)
//   WT-024: Commands work from within worktrees
//   WT-025: Auto-create worktrees directory

package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	// Create parent dir to control repo name
	parent := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	parent, _ = filepath.EvalSymlinks(parent)
	dir := filepath.Join(parent, "myrepo")
	os.MkdirAll(dir, 0o755)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("init", "-b", "main")
	run("commit", "--allow-empty", "-m", "initial")

	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	return dir
}

// WT-007: The system shall place new worktrees in a directory named
// `<repository-name>-worktrees/<branch>` located as a sibling to the
// main repository directory.
func TestResolve_WorktreesDirConvention(t *testing.T) {
	dir := setupTestRepo(t)

	info, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if info.MainWorktree != dir {
		t.Errorf("MainWorktree = %q, want %q", info.MainWorktree, dir)
	}

	expectedName := "myrepo"
	if info.RepoName != expectedName {
		t.Errorf("RepoName = %q, want %q", info.RepoName, expectedName)
	}

	parent := filepath.Dir(dir)
	expectedWtDir := filepath.Join(parent, "myrepo-worktrees")
	if info.WorktreesDir != expectedWtDir {
		t.Errorf("WorktreesDir = %q, want %q", info.WorktreesDir, expectedWtDir)
	}
}

// WT-024: When the user invokes any `wt` command from within an existing worktree,
// the system shall resolve the main repository and operate correctly.
func TestResolve_FromLinkedWorktree(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a linked worktree
	wtPath := filepath.Join(filepath.Dir(dir), "myrepo-worktrees", "feature-test")
	os.MkdirAll(filepath.Dir(wtPath), 0o755)

	cmd := exec.Command("git", "worktree", "add", "-b", "feature-test", wtPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}

	// Now cd into the linked worktree and resolve
	os.Chdir(wtPath)

	info, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() from linked worktree error: %v", err)
	}

	if info.MainWorktree != dir {
		t.Errorf("MainWorktree = %q, want %q (from linked worktree)", info.MainWorktree, dir)
	}
	if info.RepoName != "myrepo" {
		t.Errorf("RepoName = %q, want %q", info.RepoName, "myrepo")
	}
}

// WT-025: When the worktrees directory does not exist, the system shall
// create it automatically when creating the first worktree.
func TestEnsureWorktreesDir_CreatesDirectory(t *testing.T) {
	setupTestRepo(t)

	info, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// Worktrees dir should not exist yet
	if _, err := os.Stat(info.WorktreesDir); !os.IsNotExist(err) {
		t.Fatal("worktrees dir should not exist before EnsureWorktreesDir()")
	}

	err = info.EnsureWorktreesDir()
	if err != nil {
		t.Fatalf("EnsureWorktreesDir() error: %v", err)
	}

	stat, err := os.Stat(info.WorktreesDir)
	if err != nil {
		t.Fatalf("worktrees dir does not exist after EnsureWorktreesDir(): %v", err)
	}
	if !stat.IsDir() {
		t.Error("worktrees dir is not a directory")
	}
}

func TestEnsureWorktreesDir_Idempotent(t *testing.T) {
	setupTestRepo(t)

	info, _ := Resolve()

	// Call twice; should not error
	info.EnsureWorktreesDir()
	err := info.EnsureWorktreesDir()
	if err != nil {
		t.Fatalf("second EnsureWorktreesDir() error: %v", err)
	}
}

func TestResolve_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	_, err := Resolve()
	if err == nil {
		t.Error("Resolve() should error in non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error should mention 'not a git repository', got: %v", err)
	}
}
