package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/provenimpact/wt/internal/fuzzy"
)

// BranchEntry represents a branch in the branch selector.
type BranchEntry struct {
	Name        string
	Source      string // "local" or "remote"
	HasWorktree bool
}

// filteredBranchEntry holds a BranchEntry along with its fuzzy match result.
type filteredBranchEntry struct {
	BranchEntry
	match fuzzy.Match
}

// SelectBranch displays an interactive fuzzy selector for branches.
// Returns the selected branch name, or empty string if cancelled.
func SelectBranch(entries []BranchEntry, header string) (string, error) {
	m := newBranchModel(entries, header)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running branch selector: %w", err)
	}

	result := finalModel.(branchModel)
	if result.cancelled {
		return "", nil
	}
	if result.selected >= 0 && result.selected < len(result.filtered) {
		fe := result.filtered[result.selected]
		if fe.HasWorktree {
			return "", nil // Non-selectable entry
		}
		return fe.Name, nil
	}
	return "", nil
}

type branchModel struct {
	entries   []BranchEntry
	filtered  []filteredBranchEntry
	textInput textinput.Model
	selected  int
	cancelled bool
	header    string
}

var (
	disabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Faint(true)
	worktreeMarker = dimStyle.Render(" [worktree]")
)

func newBranchModel(entries []BranchEntry, header string) branchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = promptStyle
	ti.Prompt = "  "

	filtered := make([]filteredBranchEntry, len(entries))
	for i, e := range entries {
		filtered[i] = filteredBranchEntry{BranchEntry: e}
	}

	// Start selection on first selectable entry
	startIdx := 0
	for i, fe := range filtered {
		if !fe.HasWorktree {
			startIdx = i
			break
		}
	}

	return branchModel{
		entries:   entries,
		filtered:  filtered,
		textInput: ti,
		selected:  startIdx,
		header:    header,
	}
}

func (m branchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m branchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 && !m.filtered[m.selected].HasWorktree {
				return m, tea.Quit
			}
		case tea.KeyUp:
			m.moveSelection(-1)
		case tea.KeyDown:
			m.moveSelection(1)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter and score entries
	query := m.textInput.Value()
	if query == "" {
		m.filtered = make([]filteredBranchEntry, len(m.entries))
		for i, e := range m.entries {
			m.filtered[i] = filteredBranchEntry{BranchEntry: e}
		}
	} else {
		m.filtered = nil
		for _, e := range m.entries {
			match := fuzzy.Score(e.Name, query)
			if match.Matched {
				m.filtered = append(m.filtered, filteredBranchEntry{BranchEntry: e, match: match})
			}
		}
		sort.Slice(m.filtered, func(i, j int) bool {
			return m.filtered[i].match.Score > m.filtered[j].match.Score
		})
	}

	// Clamp selection
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}

	// Skip to nearest selectable entry if current is disabled
	if len(m.filtered) > 0 && m.filtered[m.selected].HasWorktree {
		m.moveSelection(1) // Try down first
	}

	return m, cmd
}

func (m *branchModel) moveSelection(dir int) {
	if len(m.filtered) == 0 {
		return
	}
	start := m.selected
	for {
		next := m.selected + dir
		if next < 0 || next >= len(m.filtered) {
			return // Can't move further
		}
		m.selected = next
		if !m.filtered[m.selected].HasWorktree {
			return // Found selectable entry
		}
		if m.selected == start {
			return // Wrapped around, no selectable entries
		}
	}
}

func (m branchModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(promptStyle.Render("  " + m.header))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	hasQuery := m.textInput.Value() != ""

	for i, fe := range m.filtered {
		if fe.HasWorktree {
			// Disabled entry: dimmed with marker
			b.WriteString(fmt.Sprintf("  %s%s\n", disabledStyle.Render(fe.Name), worktreeMarker))
			continue
		}

		cursor := "  "
		var nameText string

		if i == m.selected {
			cursor = selectedStyle.Render("> ")
			if hasQuery && fe.match.Positions != nil {
				nameText = highlightBranch(fe.Name, fe.match.Positions, selectedStyle, highlightStyle)
			} else {
				nameText = selectedStyle.Render(fe.Name)
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, nameText))
		} else {
			if hasQuery && fe.match.Positions != nil {
				nameText = highlightBranch(fe.Name, fe.match.Positions, lipgloss.NewStyle(), highlightStyle)
			} else {
				nameText = fe.Name
			}
			b.WriteString(fmt.Sprintf("  %s\n", nameText))
		}
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matches"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  ↑/↓ navigate • enter select • esc cancel"))
	b.WriteString("\n")

	return b.String()
}
