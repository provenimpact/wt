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
