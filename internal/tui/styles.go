package tui

import "github.com/charmbracelet/lipgloss"

var (
	alertBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("11")).
			Foreground(lipgloss.Color("11")).
			Padding(0, 1).
			Width(72)

	alertTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("11")).
			Padding(0, 1)
)

func AlertBox(title, message string) string {
	header := alertTitleStyle.Render(title)
	body := alertBoxStyle.Render(message)
	return header + "\n" + body
}

func SecurityWarning() string {
	return AlertBox(" ! Warning ",
		"Skills contain instructions that are injected into your AI agent's\n"+
			"context. Only install skills from sources you trust. Malicious\n"+
			"skills could lead to prompt injection attacks.")
}
