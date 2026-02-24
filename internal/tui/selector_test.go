// Feature: worktree-management
// Spec version: 1.0.0
// Generated from: spec.adoc
//
// Spec coverage:
//   WT-002: Selector entry shows branch and path (via model View)
//   WT-005: Cancel exits silently (via model Update)
//   Fuzzy match algorithm unit tests

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFuzzyMatch(t *testing.T) {
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
		{"", "", true},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.str+"/"+tt.pattern, func(t *testing.T) {
			got := fuzzyMatch(tt.str, tt.pattern)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.want)
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
