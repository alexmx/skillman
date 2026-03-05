package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pickerMode int

const (
	modePick pickerMode = iota
	modeReview
)

type pickerModel struct {
	title    string
	subtitle string
	warning  string
	items    []pickerItem
	cursor   int
	selected map[int]bool
	done     bool
	aborted  bool

	mode     pickerMode
	viewport viewport.Model
	ready    bool
	width    int
	height   int
}

type pickerItem struct {
	name    string
	desc    string
	skillDir string // path to skill directory for review
}

func PickSkills(title string, names, descriptions []string) ([]int, error) {
	return PickSkillsWithOptions(title, "", "", names, descriptions, nil, nil)
}

func PickSkillsWithPreselection(title string, names, descriptions []string, preselected map[int]bool) ([]int, error) {
	return PickSkillsWithOptions(title, "", "", names, descriptions, preselected, nil)
}

func PickSkillsWithOptions(title, subtitle, warning string, names, descriptions []string, preselected map[int]bool, skillDirs []string) ([]int, error) {
	items := make([]pickerItem, len(names))
	for i := range names {
		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i]
		}
		dir := ""
		if i < len(skillDirs) {
			dir = skillDirs[i]
		}
		items[i] = pickerItem{name: names[i], desc: desc, skillDir: dir}
	}

	selected := make(map[int]bool)
	for k, v := range preselected {
		selected[k] = v
	}

	m := pickerModel{
		title:    title,
		subtitle: subtitle,
		warning:  warning,
		items:    items,
		selected: selected,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := result.(pickerModel)
	if final.aborted {
		return nil, nil
	}

	var indices []int
	for i := range final.items {
		if final.selected[i] {
			indices = append(indices, i)
		}
	}
	return indices, nil
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mode == modeReview {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6 // header + 2 dividers + footer + padding
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeReview {
			return m.updateReview(msg)
		}
		return m.updatePick(msg)
	}

	if m.mode == modeReview {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m pickerModel) updatePick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.aborted = true
		m.done = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ":
		if m.selected[m.cursor] {
			delete(m.selected, m.cursor)
		} else {
			m.selected[m.cursor] = true
		}
	case "a":
		allSelected := len(m.selected) == len(m.items)
		if allSelected {
			m.selected = make(map[int]bool)
		} else {
			for i := range m.items {
				m.selected[i] = true
			}
		}
	case "r":
		item := m.items[m.cursor]
		if item.skillDir != "" {
			content := loadSkillContent(item.skillDir)
			height := m.height - 6
			if height < 10 {
				height = 20
			}
			vp := viewport.New(m.width, height)
			vp.SetContent(content)
			m.viewport = vp
			m.mode = modeReview
		}
	case "enter":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m pickerModel) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r", "esc", "q":
		m.mode = modePick
		return m, nil
	case "ctrl+c":
		m.aborted = true
		m.done = true
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	activeName    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	bracketStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	checkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	hintKeyStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	counterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	reviewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("141")).
				Padding(0, 1)
	reviewWarnStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("11")).
				Padding(0, 1)
	reviewDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

func (m pickerModel) View() string {
	if m.done {
		return ""
	}

	if m.mode == modeReview {
		return m.viewReview()
	}
	return m.viewPick()
}

func (m pickerModel) viewPick() string {
	var b strings.Builder

	if m.warning != "" {
		b.WriteString(m.warning)
		b.WriteString("\n\n")
	}

	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")
	if m.subtitle != "" {
		b.WriteString(dimStyle.Render(m.subtitle))
		b.WriteString("\n")
	}

	// Selection counter
	count := len(m.selected)
	total := len(m.items)
	b.WriteString(counterStyle.Render(fmt.Sprintf("  %d of %d selected", count, total)))
	b.WriteString("\n\n")

	// Compute max description width based on terminal width
	// Layout: padding(1) + cursor(2) + checkbox(3) + space(1) + name + " - " + desc
	maxDescWidth := 60
	if m.width > 0 {
		longestName := 0
		for _, item := range m.items {
			if len(item.name) > longestName {
				longestName = len(item.name)
			}
		}
		// 1 padding + 2 cursor + 3 checkbox + 1 space + name + 3 " - " + desc
		available := m.width - 1 - 2 - 3 - 1 - longestName - 3
		if available > 20 {
			maxDescWidth = available
		}
	}

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		check := bracketStyle.Render("[") + " " + bracketStyle.Render("]")
		nameRendered := item.name
		if i == m.cursor && !m.selected[i] {
			nameRendered = activeName.Render(item.name)
		}
		if m.selected[i] {
			check = bracketStyle.Render("[") + checkStyle.Render("x") + bracketStyle.Render("]")
			nameRendered = selectedStyle.Render(item.name)
		}

		desc := ""
		if item.desc != "" {
			d := item.desc
			if len(d) > maxDescWidth {
				d = d[:maxDescWidth-3] + "..."
			}
			desc = dimStyle.Render(fmt.Sprintf(" - %s", d))
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, check, nameRendered, desc))
	}

	b.WriteString("\n")

	// Hints bar
	hasReviewable := false
	for _, item := range m.items {
		if item.skillDir != "" {
			hasReviewable = true
			break
		}
	}

	hints := [][]string{
		{"space", "toggle"},
		{"a", "all"},
	}
	if hasReviewable {
		hints = append(hints, []string{"r", "review"})
	}
	hints = append(hints, []string{"enter", "confirm"}, []string{"q", "cancel"})
	b.WriteString("  ")
	for i, h := range hints {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(hintKeyStyle.Render(h[0]))
		b.WriteString(hintStyle.Render(": " + h[1]))
	}
	b.WriteString("\n")

	return lipgloss.NewStyle().PaddingLeft(1).Render(b.String())
}

func (m pickerModel) viewReview() string {
	item := m.items[m.cursor]
	header := reviewTitleStyle.Render(fmt.Sprintf(" %s - SKILL.md ", item.name)) + " " +
		reviewWarnStyle.Render(" ! Review carefully before installing ")

	// Divider line
	divWidth := m.width - 2
	if divWidth < 40 {
		divWidth = 40
	}
	divider := reviewDivider.Render(strings.Repeat("─", divWidth))

	// Scroll percentage
	pct := m.viewport.ScrollPercent() * 100
	scrollInfo := counterStyle.Render(fmt.Sprintf("%.0f%%", pct))

	// Consistent hint bar
	var hints strings.Builder
	for i, h := range [][]string{{"r/esc", "back"}, {"j/k", "scroll"}, {"ctrl+c", "quit"}} {
		if i > 0 {
			hints.WriteString("  ")
		}
		hints.WriteString(hintKeyStyle.Render(h[0]))
		hints.WriteString(hintStyle.Render(": " + h[1]))
	}

	footer := "  " + hints.String() + "  " + scrollInfo

	view := header + "\n" + divider + "\n" + m.viewport.View() + "\n" + divider + "\n" + footer
	return lipgloss.NewStyle().PaddingLeft(1).Render(view)
}

func loadSkillContent(dir string) string {
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading SKILL.md: %v", err)
	}
	return string(data)
}
