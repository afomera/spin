package dashboard

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View renders the dashboard UI
func (m *Model) View() string {
	if m.Quitting {
		return "Goodbye!\n"
	}

	if !m.Ready {
		return "Initializing..."
	}

	// Header with project name
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		TitleStyle.Render("Spin Dashboard"),
		ProjectNameStyle.Render(m.ProjectName),
	)

	// Calculate widths
	processWidth := 29                         // Fixed width for process panel
	detailsWidth := m.Width - processWidth - 4 // Account for margins and borders

	// Left panel with processes
	leftPanel := lipgloss.JoinVertical(
		lipgloss.Left,
		HeaderStyle.Render("Processes"),
		ProcessBoxStyle.Render(m.ProcessView.View()),
	)

	// Right panel with logs/details
	rightPanel := lipgloss.JoinVertical(
		lipgloss.Left,
		HeaderStyle.Render(func() string {
			if m.ViewMode == DetailsMode {
				return "Details"
			}
			return "Logs"
		}()),
		LogBoxStyle.
			Copy().
			Width(detailsWidth).
			Render(m.DetailsView.View()),
	)

	// Main content area (left and right panels)
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		leftPanel,
		rightPanel,
	)

	// Command output panel (bottom)
	var commandPanel string
	if len(m.CommandOutput) > 0 {
		commandPanel = lipgloss.JoinVertical(
			lipgloss.Left,
			OutputStyle.
				Copy().
				Width(m.Width-4). // Full width minus margins
				Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						HeaderStyle.Render("Command Output"),
						m.CommandOutput,
					),
				),
		)
	}

	// Footer with status and help
	status := StatusBarStyle.Render(fmt.Sprintf("Last updated: %s", m.LastUpdate.Format("15:04:05")))
	if m.ErrorMsg != "" {
		status = ErrorStyle.Render(m.ErrorMsg)
	}
	help := HelpStyle.Render(m.Help.View(DefaultKeyMap()))
	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		help,
	)

	// Input panel
	var inputPanel string
	if m.InputActive {
		inputPanel = InputStyle.
			Copy().
			Width(m.Width - 4). // Full width minus margins
			Render(m.Input.View())
	}

	// Join all sections vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		mainContent,
		commandPanel,
		footer,
		inputPanel,
	)
}
