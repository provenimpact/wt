// Feature: worktree-management
// Spec version: 1.0.0
// Generated from: spec.adoc
//
// Integration tests exercising CLI commands against real git repos.
//
// Spec coverage:
//   WT-004: No worktrees message with suggestion
//   WT-006: Create worktree command
//   WT-007: Worktree placement convention
//   WT-008: Create new branch if not exists
//   WT-009: Checkout existing branch
//   WT-010: Prevent duplicate worktree
//   WT-011: Output new worktree path
//   WT-012: Remove worktree and directory
//   WT-014: Refuse remove with uncommitted changes
//   WT-015: Force remove with uncommitted changes
//   WT-016: Error on nonexistent worktree remove
//   WT-017: Remove confirmation message
//   WT-018: List worktrees with details
//   WT-019: No additional worktrees message
//   WT-020: Switch outputs directory-change instruction
//   WT-021: Switch error with available worktrees
//   WT-022: Status summary with branch/dirty/remote
//   WT-023: Dirty worktree indicator
//   WT-025: Auto-create worktrees directory

package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runWt builds and runs the wt binary with the given args in the given dir.
// Returns stdout, stderr, and error.
func runWt(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()

	// Build the binary once per test run
	binary := wtBinary(t)

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

var cachedBinary string

func wtBinary(t *testing.T) string {
	t.Helper()
	if cachedBinary != "" {
		if _, err := os.Stat(cachedBinary); err == nil {
			return cachedBinary
		}
	}

	// Build to a stable location outside t.TempDir() (which is per-test)
	tmpDir := os.TempDir()
	binary := filepath.Join(tmpDir, "wt-test-binary")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = projectRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build wt binary: %v\n%s", err, out)
	}
	cachedBinary = binary
	return binary
}

func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from this file to find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

func setupTestRepo(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	parent, _ = filepath.EvalSymlinks(parent)
	dir := filepath.Join(parent, "testrepo")
	os.MkdirAll(dir, 0o755)

	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "initial")

	return dir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// --- Create tests ---

// WT-006, WT-007, WT-008, WT-011, WT-025: Create worktree with new branch,
// correct placement, path output, auto-create worktrees dir.
func TestCreate_NewBranch(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, stderr, err := runWt(t, dir, "create", "feature-new")
	if err != nil {
		t.Fatalf("wt create failed: %v\nstderr: %s", err, stderr)
	}

	// WT-011: output includes path
	expectedDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "feature-new")
	if !strings.Contains(stdout, "__wt_cd:"+expectedDir) {
		t.Errorf("stdout = %q, want __wt_cd:%s", stdout, expectedDir)
	}

	// WT-007: worktree placed in sibling dir
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("worktree directory was not created at expected location")
	}

	// WT-006: appears in git worktree list
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "feature-new") {
		t.Error("feature-new not in git worktree list")
	}

	// WT-025: worktrees parent dir was auto-created
	wtParent := filepath.Join(filepath.Dir(dir), "testrepo-worktrees")
	if _, err := os.Stat(wtParent); os.IsNotExist(err) {
		t.Error("worktrees directory was not auto-created")
	}
}

// WT-009: Create worktree with existing branch.
func TestCreate_ExistingBranch(t *testing.T) {
	dir := setupTestRepo(t)
	gitRun(t, dir, "branch", "existing-b")

	stdout, stderr, err := runWt(t, dir, "create", "existing-b")
	if err != nil {
		t.Fatalf("wt create failed: %v\nstderr: %s", err, stderr)
	}

	expectedDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "existing-b")
	if !strings.Contains(stdout, "__wt_cd:"+expectedDir) {
		t.Errorf("stdout = %q, want __wt_cd:%s", stdout, expectedDir)
	}
}

// WT-010: Prevent duplicate worktree.
func TestCreate_Duplicate(t *testing.T) {
	dir := setupTestRepo(t)

	_, _, err := runWt(t, dir, "create", "dup-branch")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, stderr, err := runWt(t, dir, "create", "dup-branch")
	if err == nil {
		t.Fatal("second create should fail")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("stderr should mention 'already exists', got: %s", stderr)
	}
}

// --- List tests ---

// WT-018: List worktrees with details.
func TestList_WithWorktrees(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "list-test")

	_, stderr, err := runWt(t, dir, "list")
	if err != nil {
		t.Fatalf("wt list failed: %v", err)
	}

	if !strings.Contains(stderr, "list-test") {
		t.Error("list output should contain 'list-test' branch")
	}
	if !strings.Contains(stderr, "BRANCH") {
		t.Error("list output should contain header 'BRANCH'")
	}
	if !strings.Contains(stderr, "MAIN") {
		t.Error("list output should contain header 'MAIN'")
	}
	if !strings.Contains(stderr, "*") {
		t.Error("list output should mark main worktree with *")
	}
}

// WT-019: No additional worktrees message.
func TestList_NoWorktrees(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir, "list")
	if err != nil {
		t.Fatalf("wt list failed: %v", err)
	}

	if !strings.Contains(stderr, "No additional worktrees") {
		t.Errorf("stderr should say 'No additional worktrees', got: %s", stderr)
	}
}

// --- Switch tests ---

// WT-020: Switch outputs directory-change instruction.
func TestSwitch_ExistingWorktree(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "switch-target")

	stdout, _, err := runWt(t, dir, "switch", "switch-target")
	if err != nil {
		t.Fatalf("wt switch failed: %v", err)
	}

	if !strings.HasPrefix(stdout, "__wt_cd:") {
		t.Errorf("stdout should start with __wt_cd:, got: %q", stdout)
	}
	if !strings.Contains(stdout, "switch-target") {
		t.Errorf("stdout should contain worktree path, got: %q", stdout)
	}
}

// WT-021: Switch error with available worktrees.
func TestSwitch_NotFound(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "available-wt")

	stdout, stderr, err := runWt(t, dir, "switch", "nonexistent")
	if err == nil {
		t.Fatal("wt switch nonexistent should fail")
	}

	// Should not output a cd instruction
	if strings.Contains(stdout, "__wt_cd:") {
		t.Error("stdout should not contain __wt_cd: on error")
	}

	// Should list available worktrees
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr should mention 'not found', got: %s", stderr)
	}
	if !strings.Contains(stderr, "available-wt") {
		t.Errorf("stderr should list available worktree 'available-wt', got: %s", stderr)
	}
}

// --- Remove tests ---

// WT-012, WT-017: Remove worktree and confirmation message.
func TestRemove_ByName(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "to-remove")

	_, stderr, err := runWt(t, dir, "remove", "to-remove")
	if err != nil {
		t.Fatalf("wt remove failed: %v\nstderr: %s", err, stderr)
	}

	// WT-017: confirmation message
	if !strings.Contains(stderr, "Removed") {
		t.Errorf("stderr should contain 'Removed', got: %s", stderr)
	}

	// WT-012: verify actually removed
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if strings.Contains(string(out), "to-remove") {
		t.Error("to-remove still in git worktree list after removal")
	}
}

// WT-016: Error on nonexistent worktree remove.
func TestRemove_NotFound(t *testing.T) {
	dir := setupTestRepo(t)

	// Need at least one worktree for remove to not say "no worktrees"
	runWt(t, dir, "create", "some-wt")

	_, stderr, err := runWt(t, dir, "remove", "nonexistent")
	if err == nil {
		t.Fatal("wt remove nonexistent should fail")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr should mention 'not found', got: %s", stderr)
	}
}

// WT-014: Refuse remove with uncommitted changes.
func TestRemove_DirtyRefused(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "dirty-wt")

	// Make it dirty
	wtDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "dirty-wt")
	os.WriteFile(filepath.Join(wtDir, "dirty.txt"), []byte("dirty"), 0o644)

	_, stderr, err := runWt(t, dir, "remove", "dirty-wt")
	if err == nil {
		t.Fatal("wt remove dirty worktree without --force should fail")
	}
	if !strings.Contains(stderr, "uncommitted changes") {
		t.Errorf("stderr should mention 'uncommitted changes', got: %s", stderr)
	}
}

// WT-015: Force remove with uncommitted changes.
func TestRemove_ForceWithDirty(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "force-rm")

	wtDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "force-rm")
	os.WriteFile(filepath.Join(wtDir, "dirty.txt"), []byte("dirty"), 0o644)

	_, stderr, err := runWt(t, dir, "remove", "--force", "force-rm")
	if err != nil {
		t.Fatalf("wt remove --force failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stderr, "Removed") {
		t.Errorf("stderr should contain 'Removed', got: %s", stderr)
	}
}

// --- Status tests ---

// WT-022: Status summary with branch/dirty/remote.
// WT-023: Dirty worktree indicator.
func TestStatus_ShowsDirtyClean(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "clean-wt")
	runWt(t, dir, "create", "dirty-wt")

	// Make dirty-wt dirty
	wtDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "dirty-wt")
	os.WriteFile(filepath.Join(wtDir, "dirty.txt"), []byte("dirty"), 0o644)

	_, stderr, err := runWt(t, dir, "status")
	if err != nil {
		t.Fatalf("wt status failed: %v", err)
	}

	if !strings.Contains(stderr, "BRANCH") {
		t.Error("status should have BRANCH header")
	}
	if !strings.Contains(stderr, "STATUS") {
		t.Error("status should have STATUS header")
	}
	if !strings.Contains(stderr, "dirty") {
		t.Error("status should show 'dirty' for dirty-wt")
	}
	if !strings.Contains(stderr, "clean") {
		t.Error("status should show 'clean' for clean-wt")
	}
	if !strings.Contains(stderr, "*") {
		t.Error("status should mark main worktree with *")
	}
}

// --- Root command (no args) tests ---

// WT-004: No worktrees message with suggestion.
func TestRoot_NoWorktreesMessage(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir)
	if err != nil {
		t.Fatalf("wt (no args) failed: %v", err)
	}

	if !strings.Contains(stderr, "No worktrees found") {
		t.Errorf("stderr should say 'No worktrees found', got: %s", stderr)
	}
	if !strings.Contains(stderr, "create") {
		t.Errorf("stderr should suggest 'create' command, got: %s", stderr)
	}
}

// --- Init tests ---

// WT-027: Shell init command outputs function code.
func TestInit_Bash(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, err := runWt(t, dir, "init", "bash")
	if err != nil {
		t.Fatalf("wt init bash failed: %v", err)
	}
	if !strings.Contains(stdout, "wt()") {
		t.Error("init bash should output wt() function")
	}
	if !strings.Contains(stdout, "__wt_cd:") {
		t.Error("init bash should reference __wt_cd: sentinel")
	}
}

func TestInit_Fish(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, err := runWt(t, dir, "init", "fish")
	if err != nil {
		t.Fatalf("wt init fish failed: %v", err)
	}
	if !strings.Contains(stdout, "function wt") {
		t.Error("init fish should output 'function wt'")
	}
}

// WT-028: Unsupported shell errors.
func TestInit_UnsupportedShell(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir, "init", "powershell")
	if err == nil {
		t.Fatal("wt init powershell should fail")
	}
	if !strings.Contains(stderr, "unsupported") {
		t.Errorf("stderr should mention 'unsupported', got: %s", stderr)
	}
}
