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

// Entry represents a worktree entry in the selector.
type Entry struct {
	Branch string
	Path   string
	Rel    string
	IsMain bool
}

// filteredEntry holds an Entry along with its fuzzy match result for rendering.
type filteredEntry struct {
	Entry
	match fuzzy.Match
}

// Select displays an interactive fuzzy selector and returns the selected worktree path.
// Returns empty string if the user cancels.
func Select(entries []Entry) (string, error) {
	m := newModel(entries)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
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
	filtered  []filteredEntry
	textInput textinput.Model
	selected  int
	cancelled bool
}

var (
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	mainStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	promptStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
)

func newModel(entries []Entry) model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = promptStyle
	ti.Prompt = "  "

	// Build initial filtered list with no scoring
	filtered := make([]filteredEntry, len(entries))
	for i, e := range entries {
		filtered[i] = filteredEntry{Entry: e}
	}

	return model{
		entries:   entries,
		filtered:  filtered,
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

	// Filter and score entries
	query := m.textInput.Value()
	if query == "" {
		m.filtered = make([]filteredEntry, len(m.entries))
		for i, e := range m.entries {
			m.filtered[i] = filteredEntry{Entry: e}
		}
	} else {
		m.filtered = nil
		for _, e := range m.entries {
			match := fuzzy.Score(e.Branch, query)
			if match.Matched {
				m.filtered = append(m.filtered, filteredEntry{Entry: e, match: match})
			}
		}
		// Sort by descending score
		sort.Slice(m.filtered, func(i, j int) bool {
			return m.filtered[i].match.Score > m.filtered[j].match.Score
		})
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

	hasQuery := m.textInput.Value() != ""

	for i, fe := range m.filtered {
		cursor := "  "
		var branchText string
		pathText := dimStyle.Render(fe.Rel)

		// Use distinct style for main worktree entries
		baseStyle := lipgloss.NewStyle()
		if fe.IsMain {
			baseStyle = mainStyle
			pathText = mainStyle.Render(fe.Rel)
		}

		if i == m.selected {
			cursor = selectedStyle.Render("> ")
			if hasQuery && fe.match.Positions != nil {
				branchText = highlightBranch(fe.Branch, fe.match.Positions, selectedStyle, highlightStyle)
			} else {
				branchText = selectedStyle.Render(fe.Branch)
			}
			b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, branchText, pathText))
		} else {
			if hasQuery && fe.match.Positions != nil {
				branchText = highlightBranch(fe.Branch, fe.match.Positions, baseStyle, highlightStyle)
			} else {
				if fe.IsMain {
					branchText = mainStyle.Render(fe.Branch)
				} else {
					branchText = fe.Branch
				}
			}
			b.WriteString(fmt.Sprintf("  %s  %s\n", branchText, pathText))
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

// highlightBranch renders a branch name with matched positions highlighted.
func highlightBranch(branch string, positions []int, baseStyle, hlStyle lipgloss.Style) string {
	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}

	runes := []rune(branch)
	var b strings.Builder
	for i, r := range runes {
		if posSet[i] {
			b.WriteString(hlStyle.Render(string(r)))
		} else {
			b.WriteString(baseStyle.Render(string(r)))
		}
	}
	return b.String()
}
