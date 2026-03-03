package tui

import "github.com/charmbracelet/lipgloss"

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	WarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
