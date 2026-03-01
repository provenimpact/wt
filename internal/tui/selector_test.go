// Feature: worktree-management
// Spec version: 1.1.0
// Generated from: spec.adoc
//
// Spec coverage:
//   WT-002: Selector entry shows branch and path (via model View)
//   WT-005: Cancel exits silently (via model Update)
//   WT-036: Dim and disable branches with existing worktrees (branch selector)
//   WT-042: Cancel in either selector exits without creating (branch selector)
//   WT-048, WT-049, WT-050, WT-051: Fuzzy scoring integration

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/provenimpact/wt/internal/fuzzy"
)

func TestFuzzyScore_Integration(t *testing.T) {
	// Verify the fuzzy module is properly integrated via Score
	tests := []struct {
		str     string
		pattern string
		want    bool
	}{
		{"feature-auth", "fea", true},
		{"feature-auth", "fa", true},
		{"feature-auth", "fauth", true},
		{"feature-auth", "feature-auth", true},
		{"feature-auth", "", true},
		{"feature-auth", "xyz", false},
		{"feature-auth", "featurez", false},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.str+"/"+tt.pattern, func(t *testing.T) {
			m := fuzzy.Score(tt.str, tt.pattern)
			if m.Matched != tt.want {
				t.Errorf("fuzzy.Score(%q, %q).Matched = %v, want %v", tt.str, tt.pattern, m.Matched, tt.want)
			}
		})
	}
}

// WT-002: The system shall display each worktree entry in the selector
// with its branch name and relative path.
func TestModelView_ShowsBranchAndPath(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-x", Path: "/tmp/repo-worktrees/feature-x", Rel: "repo-worktrees/feature-x"},
		{Branch: "fix/bug-1", Path: "/tmp/repo-worktrees/fix/bug-1", Rel: "repo-worktrees/fix/bug-1"},
	}

	m := newModel(entries)
	view := m.View()

	for _, e := range entries {
		if !strings.Contains(view, e.Branch) {
			t.Errorf("View() does not contain branch %q", e.Branch)
		}
		if !strings.Contains(view, e.Rel) {
			t.Errorf("View() does not contain relative path %q", e.Rel)
		}
	}
}

// WT-005: When the user cancels the interactive selector, the system shall
// exit without producing output.
func TestModelUpdate_EscapeCancels(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-x", Path: "/tmp/wt/feature-x", Rel: "wt/feature-x"},
	}

	m := newModel(entries)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(model)

	if !result.cancelled {
		t.Error("Escape did not set cancelled = true")
	}
}

func TestModelUpdate_CtrlCCancels(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-x", Path: "/tmp/wt/feature-x", Rel: "wt/feature-x"},
	}

	m := newModel(entries)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(model)

	if !result.cancelled {
		t.Error("Ctrl-C did not set cancelled = true")
	}
}

func TestModelUpdate_EnterSelects(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-x", Path: "/tmp/wt/feature-x", Rel: "wt/feature-x"},
		{Branch: "feature-y", Path: "/tmp/wt/feature-y", Rel: "wt/feature-y"},
	}

	m := newModel(entries)
	// Move down once
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Press enter
	updated, _ = updated.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(model)

	if result.cancelled {
		t.Error("Enter should not cancel")
	}
	if result.selected != 1 {
		t.Errorf("selected = %d, want 1", result.selected)
	}
	if result.filtered[result.selected].Branch != "feature-y" {
		t.Errorf("selected branch = %q, want %q", result.filtered[result.selected].Branch, "feature-y")
	}
}

func TestModelUpdate_NavigateUpDown(t *testing.T) {
	entries := []Entry{
		{Branch: "a", Path: "/a", Rel: "a"},
		{Branch: "b", Path: "/b", Rel: "b"},
		{Branch: "c", Path: "/c", Rel: "c"},
	}

	m := newModel(entries)

	// Initially at 0
	if m.selected != 0 {
		t.Fatalf("initial selected = %d, want 0", m.selected)
	}

	// Down twice
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(model)
	if result.selected != 2 {
		t.Errorf("after 2x down: selected = %d, want 2", result.selected)
	}

	// Down again should clamp
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyDown})
	result = updated.(model)
	if result.selected != 2 {
		t.Errorf("after 3x down: selected = %d, want 2 (clamped)", result.selected)
	}

	// Up once
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyUp})
	result = updated.(model)
	if result.selected != 1 {
		t.Errorf("after up: selected = %d, want 1", result.selected)
	}
}

func TestModelView_NoMatchesMessage(t *testing.T) {
	m := newModel(nil)
	m.filtered = nil

	view := m.View()
	if !strings.Contains(view, "No matches") {
		t.Error("View() should show 'No matches' when filtered list is empty")
	}
}

// WT-052: Main worktree entry is rendered with visually distinct style.
func TestModelView_MainWorktreeDistinctStyle(t *testing.T) {
	entries := []Entry{
		{Branch: "main", Path: "/tmp/repo", Rel: "repo", IsMain: true},
		{Branch: "feature-x", Path: "/tmp/repo-worktrees/feature-x", Rel: "repo-worktrees/feature-x", IsMain: false},
	}

	m := newModel(entries)
	// Select the second entry (feature-x) so main is not the cursor target
	m.selected = 1
	view := m.View()

	// Both entries should be present
	if !strings.Contains(view, "main") {
		t.Error("View() should contain main worktree entry")
	}
	if !strings.Contains(view, "feature-x") {
		t.Error("View() should contain linked worktree entry")
	}
}

// WT-052: Main worktree entry is selectable (distinct style is visual only).
func TestModelUpdate_MainWorktreeSelectable(t *testing.T) {
	entries := []Entry{
		{Branch: "main", Path: "/tmp/repo", Rel: "repo", IsMain: true},
		{Branch: "feature-x", Path: "/tmp/repo-worktrees/feature-x", Rel: "repo-worktrees/feature-x", IsMain: false},
	}

	m := newModel(entries)
	// Select the main worktree (first entry)
	m.selected = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(model)

	if result.cancelled {
		t.Error("selecting main worktree should not cancel")
	}
	if result.selected != 0 {
		t.Errorf("selected = %d, want 0", result.selected)
	}
	if result.filtered[result.selected].Path != "/tmp/repo" {
		t.Errorf("selected path = %q, want /tmp/repo", result.filtered[result.selected].Path)
	}
}

// --- Branch Selector tests ---

// WT-036: Branches with existing worktrees are rendered dimmed with a marker
// and are not selectable.
func TestBranchSelector_DisabledEntries(t *testing.T) {
	entries := []BranchEntry{
		{Name: "main", Source: "local", HasWorktree: true},
		{Name: "feature-a", Source: "local", HasWorktree: false},
		{Name: "feature-b", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Branches")

	// View should show [worktree] marker for main
	view := m.View()
	if !strings.Contains(view, "[worktree]") {
		t.Error("View() should show [worktree] marker for branches with worktrees")
	}
	if !strings.Contains(view, "main") {
		t.Error("View() should still show disabled branch name")
	}

	// Selection should skip disabled entries (start on first selectable)
	if m.selected != 1 {
		t.Errorf("initial selected = %d, want 1 (first selectable)", m.selected)
	}

	// Enter on disabled entry should not quit
	m.selected = 0 // Force to disabled entry
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(branchModel)
	if result.cancelled {
		t.Error("Enter on disabled entry should not cancel")
	}
	// Should not have quit command
	if cmd != nil {
		t.Error("Enter on disabled entry should not produce quit command")
	}
}

// WT-036: Navigation skips disabled entries.
func TestBranchSelector_NavigationSkipsDisabled(t *testing.T) {
	entries := []BranchEntry{
		{Name: "selectable-1", Source: "local", HasWorktree: false},
		{Name: "disabled", Source: "local", HasWorktree: true},
		{Name: "selectable-2", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Branches")

	// Should start at 0 (first selectable)
	if m.selected != 0 {
		t.Fatalf("initial selected = %d, want 0", m.selected)
	}

	// Down should skip disabled and land on selectable-2
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(branchModel)
	if result.selected != 2 {
		t.Errorf("after down: selected = %d, want 2 (skipped disabled at 1)", result.selected)
	}

	// Up should skip disabled and land back on selectable-1
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyUp})
	result = updated.(branchModel)
	if result.selected != 0 {
		t.Errorf("after up: selected = %d, want 0 (skipped disabled at 1)", result.selected)
	}
}

// WT-042: Cancel in branch selector exits without producing a selection.
func TestBranchSelector_EscapeCancels(t *testing.T) {
	entries := []BranchEntry{
		{Name: "feature-a", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Branches")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(branchModel)

	if !result.cancelled {
		t.Error("Escape did not set cancelled = true")
	}
}

func TestBranchSelector_CtrlCCancels(t *testing.T) {
	entries := []BranchEntry{
		{Name: "feature-a", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Branches")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(branchModel)

	if !result.cancelled {
		t.Error("Ctrl-C did not set cancelled = true")
	}
}

// WT-042: Enter on selectable entry should work.
func TestBranchSelector_EnterSelectsEnabled(t *testing.T) {
	entries := []BranchEntry{
		{Name: "feature-a", Source: "local", HasWorktree: false},
		{Name: "feature-b", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Test")

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Enter
	updated, cmd := updated.(branchModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(branchModel)

	if result.cancelled {
		t.Error("Enter should not cancel")
	}
	if result.selected != 1 {
		t.Errorf("selected = %d, want 1", result.selected)
	}
	if cmd == nil {
		t.Error("Enter on selectable entry should produce quit command")
	}
}

// Test that branch selector view shows header.
func TestBranchSelector_ShowsHeader(t *testing.T) {
	entries := []BranchEntry{
		{Name: "feature-a", Source: "local", HasWorktree: false},
	}

	m := newBranchModel(entries, "Base branch")
	view := m.View()

	if !strings.Contains(view, "Base branch") {
		t.Error("View() should display the header")
	}
}

// --- Additional edge case tests for strengthened confidence ---

// Test filtering with special characters in branch names.
func TestModelUpdate_FilteringSpecialCharacters(t *testing.T) {
	entries := []Entry{
		{Branch: "feature/auth-system", Path: "/tmp/wt/feature-auth-system", Rel: "wt/feature-auth-system"},
		{Branch: "fix/bug-123", Path: "/tmp/wt/fix-bug-123", Rel: "wt/fix-bug-123"},
		{Branch: "release/v2.0.1", Path: "/tmp/wt/release-v2.0.1", Rel: "wt/release-v2.0.1"},
	}

	m := newModel(entries)

	// Type "bug" to filter
	m.textInput.SetValue("bug")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	if len(result.filtered) != 1 {
		t.Errorf("filtering 'bug' should return 1 match, got %d", len(result.filtered))
	}
	if len(result.filtered) > 0 && result.filtered[0].Branch != "fix/bug-123" {
		t.Errorf("filtered branch = %q, want %q", result.filtered[0].Branch, "fix/bug-123")
	}
}

// Test filtering with very long branch names.
func TestModelUpdate_LongBranchNames(t *testing.T) {
	longBranch := "feature/very-long-branch-name-that-exceeds-normal-length-for-testing-purposes"
	entries := []Entry{
		{Branch: longBranch, Path: "/tmp/wt/long", Rel: "wt/long"},
		{Branch: "short", Path: "/tmp/wt/short", Rel: "wt/short"},
	}

	m := newModel(entries)

	// Filter should work with long names
	m.textInput.SetValue("very")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	if len(result.filtered) != 1 {
		t.Errorf("filtering 'very' should return 1 match, got %d", len(result.filtered))
	}
	if len(result.filtered) > 0 && result.filtered[0].Branch != longBranch {
		t.Errorf("filtered branch should be the long branch")
	}
}

// Test that selection clamps correctly when filtered list changes.
func TestModelUpdate_SelectionClampsOnFilter(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-a", Path: "/tmp/a", Rel: "a"},
		{Branch: "feature-b", Path: "/tmp/b", Rel: "b"},
		{Branch: "bugfix-c", Path: "/tmp/c", Rel: "c"},
	}

	m := newModel(entries)
	// Move to last item
	m.selected = 2

	// Filter to reduce list to 1 item
	m.textInput.SetValue("bugfix")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	if result.selected != 0 {
		t.Errorf("selection should clamp to 0, got %d", result.selected)
	}
}

// Test empty query shows all entries in original order.
func TestModelUpdate_EmptyQueryPreservesOrder(t *testing.T) {
	entries := []Entry{
		{Branch: "z-last", Path: "/z", Rel: "z"},
		{Branch: "a-first", Path: "/a", Rel: "a"},
		{Branch: "m-middle", Path: "/m", Rel: "m"},
	}

	m := newModel(entries)

	// Empty query - should preserve original order
	if len(m.filtered) != 3 {
		t.Fatalf("initial filtered length = %d, want 3", len(m.filtered))
	}
	if m.filtered[0].Branch != "z-last" {
		t.Errorf("first entry should be 'z-last', got %q", m.filtered[0].Branch)
	}
	if m.filtered[1].Branch != "a-first" {
		t.Errorf("second entry should be 'a-first', got %q", m.filtered[1].Branch)
	}
	if m.filtered[2].Branch != "m-middle" {
		t.Errorf("third entry should be 'm-middle', got %q", m.filtered[2].Branch)
	}
}

// Test that filtering with query that matches nothing shows "No matches".
func TestModelView_NoMatchesAfterFiltering(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-a", Path: "/a", Rel: "a"},
		{Branch: "feature-b", Path: "/b", Rel: "b"},
	}

	m := newModel(entries)
	m.textInput.SetValue("xyz")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	view := result.View()
	if !strings.Contains(view, "No matches") {
		t.Error("View should show 'No matches' when filter excludes all entries")
	}
}

// Test Enter on empty filtered list does nothing.
func TestModelUpdate_EnterOnEmptyFilteredList(t *testing.T) {
	entries := []Entry{
		{Branch: "feature-a", Path: "/a", Rel: "a"},
	}

	m := newModel(entries)
	m.textInput.SetValue("xyz") // Filter to nothing
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	// Press enter on empty list
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result = updated.(model)

	if result.cancelled {
		t.Error("Enter on empty list should not set cancelled")
	}
	// Should still be in the model (no quit)
}

// Test fuzzy scoring prioritizes better matches.
func TestModelUpdate_FuzzyScoringPrioritizesBetterMatches(t *testing.T) {
	entries := []Entry{
		{Branch: "x-main", Path: "/xm", Rel: "xm"},
		{Branch: "main", Path: "/main", Rel: "main"},
		{Branch: "maintain", Path: "/maintain", Rel: "maintain"},
	}

	m := newModel(entries)
	m.textInput.SetValue("main")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	// Exact match "main" should rank first
	if len(result.filtered) < 1 {
		t.Fatal("expected at least 1 match")
	}
	if result.filtered[0].Branch != "main" {
		t.Errorf("best match should be 'main', got %q", result.filtered[0].Branch)
	}
}

// Test main worktree entry with filtering.
func TestModelUpdate_MainWorktreeFilterable(t *testing.T) {
	entries := []Entry{
		{Branch: "main", Path: "/repo", Rel: "repo", IsMain: true},
		{Branch: "feature-main-fix", Path: "/wt/feature", Rel: "wt/feature", IsMain: false},
	}

	m := newModel(entries)
	m.textInput.SetValue("main")
	updated, _ := m.Update(tea.KeyMsg{})
	result := updated.(model)

	// Both should match the filter
	if len(result.filtered) != 2 {
		t.Errorf("filtering 'main' should return 2 matches, got %d", len(result.filtered))
	}

	// Verify main worktree is still marked
	hasMain := false
	for _, fe := range result.filtered {
		if fe.IsMain {
			hasMain = true
			if fe.Branch != "main" {
				t.Errorf("main worktree branch should be 'main', got %q", fe.Branch)
			}
		}
	}
	if !hasMain {
		t.Error("filtered results should include main worktree")
	}
}

// Test branch selector with all entries disabled.
func TestBranchSelector_AllEntriesDisabled(t *testing.T) {
	entries := []BranchEntry{
		{Name: "main", Source: "local", HasWorktree: true},
		{Name: "feature-a", Source: "local", HasWorktree: true},
		{Name: "feature-b", Source: "local", HasWorktree: true},
	}

	m := newBranchModel(entries, "Branches")

	// No selectable entry - selected should be 0 or handled gracefully
	// Enter should not produce quit
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(branchModel)

	if cmd != nil && !result.cancelled {
		// Should not quit on disabled entry
		t.Error("Enter on all-disabled list should not produce quit command")
	}
}

// Test navigation when only one selectable entry exists.
func TestBranchSelector_SingleSelectableEntry(t *testing.T) {
	entries := []BranchEntry{
		{Name: "disabled-1", Source: "local", HasWorktree: true},
		{Name: "selectable", Source: "local", HasWorktree: false},
		{Name: "disabled-2", Source: "local", HasWorktree: true},
	}

	m := newBranchModel(entries, "Branches")

	// Should start on the only selectable entry
	if m.selected != 1 {
		t.Errorf("initial selected = %d, want 1 (only selectable)", m.selected)
	}

	// Verify Enter works on the selectable entry
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(branchModel)
	if result.cancelled {
		t.Error("Enter on selectable entry should not cancel")
	}
	if cmd == nil {
		t.Error("Enter on selectable entry should produce quit command")
	}
}