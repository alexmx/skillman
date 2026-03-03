package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pickerModel struct {
	items    []pickerItem
	cursor   int
	selected map[int]bool
	done     bool
	aborted  bool
}

type pickerItem struct {
	name string
	desc string
}

func PickSkills(names, descriptions []string) ([]int, error) {
	items := make([]pickerItem, len(names))
	for i := range names {
		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i]
		}
		items[i] = pickerItem{name: names[i], desc: desc}
	}

	m := pickerModel{
		items:    items,
		selected: make(map[int]bool),
	}

	p := tea.NewProgram(m)
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
	case tea.KeyMsg:
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
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			allSelected := len(m.selected) == len(m.items)
			if allSelected {
				m.selected = make(map[int]bool)
			} else {
				for i := range m.items {
					m.selected[i] = true
				}
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m pickerModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select skills to install"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  space: toggle  a: all  enter: confirm  q: cancel"))
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
			// Truncate long descriptions
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
