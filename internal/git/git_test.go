// Feature: worktree-management
// Spec version: 1.0.0
// Generated from: spec.adoc
//
// Spec coverage:
//   WT-006, WT-007, WT-008, WT-009, WT-010: worktree creation
//   WT-012, WT-014, WT-015: worktree removal
//   WT-022, WT-023: dirty/clean detection and ahead/behind

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repo and returns its path and a cleanup func.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	dir, _ = filepath.EvalSymlinks(dir)

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

	// Change to the test repo so git commands work
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	return dir
}

func TestListWorktrees_MainOnly(t *testing.T) {
	dir := setupTestRepo(t)

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].Path != dir {
		t.Errorf("worktree path = %q, want %q", wts[0].Path, dir)
	}
	if wts[0].Branch != "main" {
		t.Errorf("worktree branch = %q, want %q", wts[0].Branch, "main")
	}
}

// WT-006: When the user invokes `wt create <branch>`, the system shall create
// a new git worktree in the repository's worktrees directory.
// WT-008: Create new branch if not exists.
func TestAddWorktree_NewBranch(t *testing.T) {
	setupTestRepo(t)

	wtPath := filepath.Join(t.TempDir(), "feature-x")
	err := AddWorktree(wtPath, "feature-x", true, "")
	if err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify it appears in list
	wts, _ := ListWorktrees()
	found := false
	for _, wt := range wts {
		if wt.Branch == "feature-x" {
			found = true
			break
		}
	}
	if !found {
		t.Error("feature-x not found in worktree list after creation")
	}
}

// WT-009: When the user invokes `wt create <branch>` and a branch exists,
// the system shall check out that existing branch.
func TestAddWorktree_ExistingBranch(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a branch first
	cmd := exec.Command("git", "branch", "existing-branch")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch failed: %v\n%s", err, out)
	}

	wtPath := filepath.Join(t.TempDir(), "existing-branch")
	err := AddWorktree(wtPath, "existing-branch", false, "")
	if err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}

	wts, _ := ListWorktrees()
	found := false
	for _, wt := range wts {
		if wt.Branch == "existing-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("existing-branch not found in worktree list")
	}
}

// WT-012: Remove worktree and directory.
func TestRemoveWorktree(t *testing.T) {
	setupTestRepo(t)

	wtPath := filepath.Join(t.TempDir(), "to-remove")
	if err := AddWorktree(wtPath, "to-remove", true, ""); err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}

	err := RemoveWorktree(wtPath, false)
	if err != nil {
		t.Fatalf("RemoveWorktree() error: %v", err)
	}

	// Verify removed from list
	wts, _ := ListWorktrees()
	for _, wt := range wts {
		if wt.Branch == "to-remove" {
			t.Error("to-remove still in worktree list after removal")
		}
	}
}

// WT-023: If a worktree has uncommitted changes, the system shall indicate
// it as dirty.
func TestIsDirty_CleanRepo(t *testing.T) {
	setupTestRepo(t)

	dirty, err := IsDirty(".")
	if err != nil {
		t.Fatalf("IsDirty() error: %v", err)
	}
	if dirty {
		t.Error("fresh repo should not be dirty")
	}
}

func TestIsDirty_WithChanges(t *testing.T) {
	dir := setupTestRepo(t)

	// Create an untracked file
	os.WriteFile(filepath.Join(dir, "new-file.txt"), []byte("hello"), 0o644)

	dirty, err := IsDirty(dir)
	if err != nil {
		t.Fatalf("IsDirty() error: %v", err)
	}
	if !dirty {
		t.Error("repo with untracked file should be dirty")
	}
}

// WT-022: ahead/behind with no upstream returns (0, 0, nil)
func TestAheadBehind_NoUpstream(t *testing.T) {
	dir := setupTestRepo(t)

	ahead, behind, err := AheadBehind(dir)
	if err != nil {
		t.Fatalf("AheadBehind() error: %v", err)
	}
	if ahead != 0 || behind != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", ahead, behind)
	}
}

func TestBranchExists_LocalBranch(t *testing.T) {
	dir := setupTestRepo(t)

	// 'main' should exist
	exists, err := BranchExists("main")
	if err != nil {
		t.Fatalf("BranchExists() error: %v", err)
	}
	if !exists {
		t.Error("'main' branch should exist")
	}

	// Create another branch
	cmd := exec.Command("git", "branch", "test-branch")
	cmd.Dir = dir
	cmd.CombinedOutput()

	exists, err = BranchExists("test-branch")
	if err != nil {
		t.Fatalf("BranchExists() error: %v", err)
	}
	if !exists {
		t.Error("'test-branch' should exist after creation")
	}
}

func TestBranchExists_NonexistentBranch(t *testing.T) {
	setupTestRepo(t)

	exists, err := BranchExists("nonexistent-branch-xyz")
	if err != nil {
		t.Fatalf("BranchExists() error: %v", err)
	}
	if exists {
		t.Error("nonexistent branch should not exist")
	}
}

// WT-015: Force remove worktree with uncommitted changes.
func TestRemoveWorktree_ForceWithDirtyState(t *testing.T) {
	setupTestRepo(t)

	wtPath := filepath.Join(t.TempDir(), "dirty-wt")
	if err := AddWorktree(wtPath, "dirty-wt", true, ""); err != nil {
		t.Fatalf("AddWorktree() error: %v", err)
	}

	// Make it dirty
	os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0o644)

	dirty, _ := IsDirty(wtPath)
	if !dirty {
		t.Fatal("worktree should be dirty after writing file")
	}

	// Force remove should succeed
	err := RemoveWorktree(wtPath, true)
	if err != nil {
		t.Fatalf("RemoveWorktree(force=true) error: %v", err)
	}
}
