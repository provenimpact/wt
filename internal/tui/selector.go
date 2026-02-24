package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Entry represents a worktree entry in the selector.
type Entry struct {
	Branch string
	Path   string
	Rel    string
}

// Select displays an interactive fuzzy selector and returns the selected worktree path.
// Returns empty string if the user cancels.
func Select(entries []Entry) (string, error) {
	m := newModel(entries)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running selector: %w", err)
	}

	result := finalModel.(model)
	if result.cancelled {
		return "", nil
	}
	if result.selected >= 0 && result.selected < len(result.filtered) {
		return result.filtered[result.selected].Path, nil
	}
	return "", nil
}

type model struct {
	entries   []Entry
	filtered  []Entry
	textInput textinput.Model
	selected  int
	cancelled bool
}

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

func newModel(entries []Entry) model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = promptStyle
	ti.Prompt = "  "

	return model{
		entries:   entries,
		filtered:  entries,
		textInput: ti,
		selected:  0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				return m, tea.Quit
			}
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
			}
		case tea.KeyDown:
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Filter entries
	query := strings.ToLower(m.textInput.Value())
	if query == "" {
		m.filtered = m.entries
	} else {
		m.filtered = nil
		for _, e := range m.entries {
			if fuzzyMatch(strings.ToLower(e.Branch), query) {
				m.filtered = append(m.filtered, e)
			}
		}
	}

	// Clamp selection
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(promptStyle.Render("  Worktrees"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	for i, entry := range m.filtered {
		cursor := "  "
		branchText := entry.Branch
		pathText := entry.Rel

		if i == m.selected {
			cursor = selectedStyle.Render("> ")
			branchText = selectedStyle.Render(entry.Branch)
			pathText = dimStyle.Render(entry.Rel)
		} else {
			branchText = fmt.Sprintf("  %s", entry.Branch)
			pathText = dimStyle.Render(entry.Rel)
		}

		if i == m.selected {
			b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, branchText, pathText))
		} else {
			b.WriteString(fmt.Sprintf("%s  %s\n", branchText, pathText))
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

// fuzzyMatch checks if all characters in pattern appear in str in order.
func fuzzyMatch(str, pattern string) bool {
	pi := 0
	for si := 0; si < len(str) && pi < len(pattern); si++ {
		if str[si] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}
