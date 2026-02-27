// Feature: worktree-management
// Spec version: 1.2.0
// Generated from: spec.adoc
//
// Integration tests exercising CLI commands against real git repos.
//
// Spec coverage:
//   WT-001: Interactive fuzzy selector including main worktree (interactive, covered by TUI unit tests)
//   WT-004: No linked worktrees message with suggestion
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
//   WT-018: List all worktrees including main with branch, path, and main indicator
//   WT-019: No additional worktrees message with create suggestion
//   WT-020: Switch outputs directory-change instruction
//   WT-021: Switch error lists all worktrees including main
//   WT-022: Status summary with branch, dirty/clean, and ahead/behind counts
//   WT-023: Dirty worktree indicator
//   WT-025: Auto-create worktrees directory
//   WT-032: Create flat directory for branches with slashes
//   WT-033: Preserve original branch name for git operations
//   WT-034: Remove empty parent directories on worktree removal
//   WT-039: Direct creation bypasses interactive selector
//   WT-040: Create branch from specified base reference
//   WT-043: Tab completion for create suggests branches
//   WT-044: Tab completion for switch suggests all worktrees including main
//   WT-045: Tab completion for remove suggests linked worktrees
//   WT-046: Completion command outputs shell scripts
//   WT-047: Error on unsupported shell for completion
//   WT-054: Switch to main worktree by branch name
//   WT-055: Exclude main worktree from remove choices
//
// Interactive-only (require TUI, not testable via binary):
//   WT-013: Interactive remove selector
//   WT-035: Interactive branch selector on no-arg create
//   WT-037: Filter to local branches with --local flag
//   WT-038: Filter to remote branches with --remote flag
//   WT-041: Base branch selector for new branches in interactive mode

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

// WT-018: List all worktrees including main with branch name, path, and main indicator.
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
	// WT-018: path column must be present
	if !strings.Contains(stderr, "PATH") {
		t.Error("list output should contain header 'PATH'")
	}
	if !strings.Contains(stderr, "MAIN") {
		t.Error("list output should contain header 'MAIN'")
	}
	if !strings.Contains(stderr, "*") {
		t.Error("list output should mark main worktree with *")
	}
	// WT-018: main worktree branch should be listed
	if !strings.Contains(stderr, "main") {
		t.Error("list output should contain main worktree branch")
	}
	// WT-018: verify paths are shown (relative path for the linked worktree)
	if !strings.Contains(stderr, "testrepo-worktrees") {
		t.Error("list output should contain worktree path")
	}
}

// WT-019: No additional worktrees message with suggestion to create.
func TestList_NoWorktrees(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir, "list")
	if err != nil {
		t.Fatalf("wt list failed: %v", err)
	}

	if !strings.Contains(stderr, "No additional worktrees") {
		t.Errorf("stderr should say 'No additional worktrees', got: %s", stderr)
	}
	// WT-019: should suggest the wt create command
	if !strings.Contains(stderr, "wt create") {
		t.Errorf("stderr should suggest 'wt create' command, got: %s", stderr)
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

// WT-021: Switch error lists all worktrees including main.
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
	// WT-021: main worktree should also be listed in available worktrees
	if !strings.Contains(stderr, "main") {
		t.Errorf("stderr should list main worktree in available worktrees, got: %s", stderr)
	}
}

// WT-054: Switch to main worktree by branch name.
func TestSwitch_ToMainWorktree(t *testing.T) {
	dir := setupTestRepo(t)
	runWt(t, dir, "create", "some-wt")

	stdout, _, err := runWt(t, dir, "switch", "main")
	if err != nil {
		t.Fatalf("wt switch main failed: %v", err)
	}

	if !strings.HasPrefix(stdout, "__wt_cd:") {
		t.Errorf("stdout should start with __wt_cd:, got: %q", stdout)
	}
	// The cd target should be the main repo directory
	if !strings.Contains(stdout, "testrepo") {
		t.Errorf("stdout should contain main repo path, got: %q", stdout)
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

// WT-022: Status summary with branch, clean/dirty state, and ahead/behind remote counts.
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
	// WT-022: ahead/behind remote counts must be shown
	if !strings.Contains(stderr, "AHEAD") {
		t.Error("status should have AHEAD header")
	}
	if !strings.Contains(stderr, "BEHIND") {
		t.Error("status should have BEHIND header")
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

// --- Branch name sanitization tests ---

// WT-032: When the user invokes `wt create fix/bug-123`, the system shall
// create the worktree directory as `fix-bug-123` within the worktrees directory.
func TestCreate_SlashBranch_FlatDirectory(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, stderr, err := runWt(t, dir, "create", "fix/bug-123")
	if err != nil {
		t.Fatalf("wt create fix/bug-123 failed: %v\nstderr: %s", err, stderr)
	}

	// Directory should be flat: fix-bug-123, not fix/bug-123
	expectedDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "fix-bug-123")
	if !strings.Contains(stdout, "__wt_cd:"+expectedDir) {
		t.Errorf("stdout = %q, want __wt_cd:%s", stdout, expectedDir)
	}

	// Flat directory should exist
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("sanitized worktree directory was not created")
	}

	// Nested directory should NOT exist
	nestedDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "fix", "bug-123")
	if _, err := os.Stat(nestedDir); err == nil {
		t.Error("nested directory should not exist; directory should be flat")
	}
}

// WT-033: The system shall preserve the original branch name for all git operations.
func TestCreate_SlashBranch_PreservesBranchName(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir, "create", "fix/bug-456")
	if err != nil {
		t.Fatalf("wt create fix/bug-456 failed: %v\nstderr: %s", err, stderr)
	}

	// Git branch should use original name, not sanitized
	cmd := exec.Command("git", "branch")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "fix/bug-456") {
		t.Errorf("git branch should show 'fix/bug-456', got: %s", out)
	}

	// wt list should show original branch name
	_, listStderr, err := runWt(t, dir, "list")
	if err != nil {
		t.Fatalf("wt list failed: %v", err)
	}
	if !strings.Contains(listStderr, "fix/bug-456") {
		t.Errorf("wt list should show 'fix/bug-456', got: %s", listStderr)
	}
}

// WT-034: When a worktree is removed, if the removal leaves empty parent
// directories within the worktrees directory, then the system shall remove
// those empty parent directories.
func TestRemove_CleansEmptyParentDirs(t *testing.T) {
	dir := setupTestRepo(t)

	// Manually create a nested worktree structure to simulate legacy behavior
	// (before sanitization, fix/bug-123 would create nested dirs)
	wtDir := filepath.Join(filepath.Dir(dir), "testrepo-worktrees")
	nestedPath := filepath.Join(wtDir, "fix", "old-bug")
	os.MkdirAll(nestedPath, 0o755)

	// Create a worktree at the nested path using git directly
	gitRun(t, dir, "worktree", "add", "-b", "fix-old-bug", nestedPath)

	// Remove via wt
	_, stderr, err := runWt(t, dir, "remove", "fix-old-bug")
	if err != nil {
		t.Fatalf("wt remove failed: %v\nstderr: %s", err, stderr)
	}

	// The orphan 'fix' directory should have been cleaned up
	fixDir := filepath.Join(wtDir, "fix")
	if _, err := os.Stat(fixDir); err == nil {
		t.Error("empty 'fix' parent directory should have been removed")
	}
}

// WT-039: When the user invokes `wt create <branch>`, the system shall use
// the direct-branch creation flow without launching the interactive selector.
func TestCreate_DirectBranch_NoInteraction(t *testing.T) {
	dir := setupTestRepo(t)

	// Direct create should work without hanging waiting for TTY input
	stdout, stderr, err := runWt(t, dir, "create", "direct-branch")
	if err != nil {
		t.Fatalf("wt create direct-branch failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "__wt_cd:") {
		t.Errorf("stdout should contain __wt_cd:, got: %q", stdout)
	}
}

// WT-040: When the user invokes `wt create <branch> --base <ref>`, the system
// shall create a new branch starting from the specified base reference.
func TestCreate_WithBaseFlag(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a branch with some commits
	gitRun(t, dir, "checkout", "-b", "develop")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "develop commit")
	gitRun(t, dir, "checkout", "main")

	stdout, stderr, err := runWt(t, dir, "create", "new-feature", "--base", "develop")
	if err != nil {
		t.Fatalf("wt create --base failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "__wt_cd:") {
		t.Errorf("stdout should contain __wt_cd:, got: %q", stdout)
	}

	// Verify the new branch was created
	cmd := exec.Command("git", "branch")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "new-feature") {
		t.Error("new-feature branch should exist")
	}

	// Verify the branch is based on develop (has the develop commit)
	wtPath := filepath.Join(filepath.Dir(dir), "testrepo-worktrees", "new-feature")
	cmd = exec.Command("git", "-C", wtPath, "log", "--oneline", "-1")
	out, _ = cmd.Output()
	if !strings.Contains(string(out), "develop commit") {
		t.Errorf("new-feature should be based on develop, last commit: %s", out)
	}
}

// --- Completion command tests ---

// WT-046: The system shall provide a `wt completion <shell>` command.
func TestCompletion_Bash(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, err := runWt(t, dir, "completion", "bash")
	if err != nil {
		t.Fatalf("wt completion bash failed: %v", err)
	}
	if len(stdout) == 0 {
		t.Error("completion bash should produce output")
	}
	// Bash completion v2 should reference the binary name
	if !strings.Contains(stdout, "wt") {
		t.Error("bash completion should reference 'wt'")
	}
}

func TestCompletion_Zsh(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, err := runWt(t, dir, "completion", "zsh")
	if err != nil {
		t.Fatalf("wt completion zsh failed: %v", err)
	}
	if len(stdout) == 0 {
		t.Error("completion zsh should produce output")
	}
}

func TestCompletion_Fish(t *testing.T) {
	dir := setupTestRepo(t)

	stdout, _, err := runWt(t, dir, "completion", "fish")
	if err != nil {
		t.Fatalf("wt completion fish failed: %v", err)
	}
	if len(stdout) == 0 {
		t.Error("completion fish should produce output")
	}
}

// WT-047: If the user invokes `wt completion` with an unsupported shell,
// the system shall display an error listing the supported shells.
func TestCompletion_UnsupportedShell(t *testing.T) {
	dir := setupTestRepo(t)

	_, stderr, err := runWt(t, dir, "completion", "powershell")
	if err == nil {
		t.Fatal("wt completion powershell should fail")
	}
	if !strings.Contains(stderr, "unsupported") {
		t.Errorf("stderr should mention 'unsupported', got: %s", stderr)
	}
	// Should list supported shells
	if !strings.Contains(stderr, "bash") || !strings.Contains(stderr, "zsh") || !strings.Contains(stderr, "fish") {
		t.Errorf("stderr should list supported shells, got: %s", stderr)
	}
}

// --- Tab completion tests ---

// WT-043: Tab completion for create suggests branches excluding those with worktrees.
func TestCompletion_CreateSuggestsBranches(t *testing.T) {
	dir := setupTestRepo(t)

	// Create some branches
	gitRun(t, dir, "branch", "feature-a")
	gitRun(t, dir, "branch", "feature-b")

	// Create a worktree for feature-a
	runWt(t, dir, "create", "feature-a")

	// Use Cobra's __complete hidden command for testing completions
	stdout, _, err := runWt(t, dir, "__complete", "create", "")
	if err != nil {
		// __complete may return exit code 1 with valid output
		// Check stdout regardless
	}
	_ = err

	// feature-b should be suggested (no worktree)
	if !strings.Contains(stdout, "feature-b") {
		t.Errorf("completion should suggest 'feature-b', got: %s", stdout)
	}
	// feature-a should NOT be suggested (has worktree)
	// Note: main is the worktree for the main repo, should also be excluded
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "feature-a" || strings.HasPrefix(trimmed, "feature-a\t") {
			t.Errorf("completion should NOT suggest 'feature-a' (has worktree), got line: %s", line)
		}
	}
}

// WT-044: Tab completion for switch suggests all worktree branch names including main.
func TestCompletion_SwitchSuggestsWorktrees(t *testing.T) {
	dir := setupTestRepo(t)

	runWt(t, dir, "create", "wt-alpha")
	runWt(t, dir, "create", "wt-beta")

	stdout, _, _ := runWt(t, dir, "__complete", "switch", "")

	if !strings.Contains(stdout, "wt-alpha") {
		t.Errorf("completion should suggest 'wt-alpha', got: %s", stdout)
	}
	if !strings.Contains(stdout, "wt-beta") {
		t.Errorf("completion should suggest 'wt-beta', got: %s", stdout)
	}
	// WT-044: main worktree branch should also be suggested
	if !strings.Contains(stdout, "main") {
		t.Errorf("completion should suggest 'main' (main worktree), got: %s", stdout)
	}
}

// WT-045: Tab completion for remove suggests existing linked worktree branch names.
func TestCompletion_RemoveSuggestsLinkedWorktrees(t *testing.T) {
	dir := setupTestRepo(t)

	runWt(t, dir, "create", "rm-target")

	stdout, _, _ := runWt(t, dir, "__complete", "remove", "")

	if !strings.Contains(stdout, "rm-target") {
		t.Errorf("completion should suggest 'rm-target', got: %s", stdout)
	}
	// main should NOT be suggested
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "main" || strings.HasPrefix(trimmed, "main\t") {
			t.Errorf("completion should NOT suggest 'main' (not a linked worktree), got line: %s", line)
		}
	}
}

// --- Switch with sanitized name ---

// Test that switch works with sanitized directory name for slash branches.
func TestSwitch_SanitizedName(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a worktree for a branch with slash
	runWt(t, dir, "create", "fix/switch-test")

	// Switch using the original branch name
	stdout, _, err := runWt(t, dir, "switch", "fix/switch-test")
	if err != nil {
		t.Fatalf("wt switch fix/switch-test failed: %v", err)
	}
	if !strings.Contains(stdout, "__wt_cd:") {
		t.Errorf("stdout should contain __wt_cd:, got: %q", stdout)
	}
}
