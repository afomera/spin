package dashboard

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	// Base Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	ProjectNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("4")).
				PaddingLeft(4).
				Underline(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	LogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4"))

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	// Box Styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(1, 1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	ProcessBoxStyle = BoxStyle.Copy().
			BorderForeground(special).
			Bold(true).
			Width(29).
			MarginRight(1)

	LogBoxStyle = BoxStyle.Copy().
			BorderForeground(lipgloss.Color("63"))

	OutputStyle = BoxStyle.Copy().
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 1).
			MarginTop(1).
			Width(100).
			Align(lipgloss.Left)

	// List Item Styles
	ProcessItemStyle = lipgloss.NewStyle().
				PaddingLeft(1)

	SelectedProcessStyle = ProcessItemStyle.Copy().
				Bold(true).
				Foreground(highlight)

	LogItemStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	// Status Styles
	RunningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")) // Green

	StoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")) // Red

	StartingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")) // Yellow

	// Header Styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)
)
