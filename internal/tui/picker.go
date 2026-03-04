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
	return PickSkillsWithOptions(title, "", names, descriptions, nil, nil)
}

func PickSkillsWithPreselection(title string, names, descriptions []string, preselected map[int]bool) ([]int, error) {
	return PickSkillsWithOptions(title, "", names, descriptions, preselected, nil)
}

func PickSkillsWithOptions(title, warning string, names, descriptions []string, preselected map[int]bool, skillDirs []string) ([]int, error) {
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
			m.viewport.Height = msg.Height - 4 // room for header/footer
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
			height := m.height - 4
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
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	reviewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("99")).
				Padding(0, 1)
	reviewFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
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

	hasReviewable := false
	for _, item := range m.items {
		if item.skillDir != "" {
			hasReviewable = true
			break
		}
	}

	hint := "  space: toggle  a: all  enter: confirm  q: cancel"
	if hasReviewable {
		hint = "  space: toggle  a: all  r: review  enter: confirm  q: cancel"
	}
	b.WriteString(dimStyle.Render(hint))
	b.WriteString("\n\n")

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		check := "[ ]"
		nameRendered := item.name
		if m.selected[i] {
			check = selectedStyle.Render("[x]")
			nameRendered = selectedStyle.Render(item.name)
		}

		desc := ""
		if item.desc != "" {
			d := item.desc
			if len(d) > 60 {
				d = d[:57] + "..."
			}
			desc = dimStyle.Render(fmt.Sprintf(" - %s", d))
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, check, nameRendered, desc))
	}

	return b.String()
}

func (m pickerModel) viewReview() string {
	item := m.items[m.cursor]
	header := reviewTitleStyle.Render(fmt.Sprintf(" %s - SKILL.md ", item.name))
	footer := reviewFooterStyle.Render("  r/esc: back  arrows/j/k: scroll  ctrl+c: quit")

	pct := m.viewport.ScrollPercent() * 100
	scrollInfo := reviewFooterStyle.Render(fmt.Sprintf("  %.0f%%", pct))

	return header + "\n" + m.viewport.View() + "\n" + footer + scrollInfo
}

func loadSkillContent(dir string) string {
	path := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading SKILL.md: %v", err)
	}
	return string(data)
}
