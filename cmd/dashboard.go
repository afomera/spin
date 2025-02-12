package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/dashboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Interactive dashboard for managing processes",
	Long: `A terminal user interface for managing and monitoring your development processes.
Provides real-time process status, resource usage, and quick actions for process control.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load project config from current directory
		configPath := filepath.Join(".", "spin.config.json")
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		// Create and initialize the dashboard
		model, err := dashboard.New(cfg)
		if err != nil {
			fmt.Printf("Error initializing dashboard: %v\n", err)
			return
		}

		// Run the dashboard
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running dashboard: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
