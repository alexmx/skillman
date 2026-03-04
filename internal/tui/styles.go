package tui

import "github.com/charmbracelet/lipgloss"

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	WarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

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
