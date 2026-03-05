package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmModel struct {
	message  string
	yes      bool
	done     bool
}

func Confirm(message string) (bool, error) {
	m := confirmModel{message: message}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}
	return result.(confirmModel).yes, nil
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "y":
			m.yes = true
			m.done = true
			return m, tea.Quit
		case "n", "q", "ctrl+c", "esc":
			m.yes = false
			m.done = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	confirmMsgStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
	confirmKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	confirmDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	msg := confirmMsgStyle.Render(m.message)
	hint := confirmDimStyle.Render(" [") + confirmKeyStyle.Render("y") + confirmDimStyle.Render("/") + confirmKeyStyle.Render("N") + confirmDimStyle.Render("] ")
	return " " + msg + hint
}
