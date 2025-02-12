package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/process"
	"github.com/afomera/spin/internal/service"
	"github.com/spf13/cobra"
)

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all running processes",
	Long: `Stop all running processes and clean up tmux sessions.

Example:
  spin down     # Stop all processes`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration from current directory
		configPath := filepath.Join(".", "spin.config.json")
		cfg, err := config.LoadConfig(configPath)
		if err == nil && cfg != nil {
			// Initialize service manager
			svcManager := service.NewServiceManager()
			if len(cfg.Dependencies.Services) > 0 {
				fmt.Println("Stopping services...")
				for _, serviceName := range cfg.Dependencies.Services {
					svc, err := service.CreateService(serviceName)
					if err != nil {
						fmt.Printf("Warning: Failed to create service %s: %v\n", serviceName, err)
						continue
					}
					svcManager.RegisterService(svc)

					if svc.IsRunning() {
						fmt.Printf("Stopping %s...\n", serviceName)
						if err := svcManager.StopService(serviceName); err != nil {
							fmt.Printf("Warning: Failed to stop service %s: %v\n", serviceName, err)
						}
					}
				}
			}
		}

		// Get the process manager instance
		manager := process.GetManager(nil)

		// Get all processes
		processes := manager.ListProcesses()
		if len(processes) == 0 {
			fmt.Println("No running processes")
			return
		}

		fmt.Println("Stopping all processes...")
		for _, p := range processes {
			fmt.Printf("Stopping %s...\n", p.Name)
			if err := manager.StopProcess(p.Name); err != nil {
				fmt.Printf("Warning: Failed to stop %s: %v\n", p.Name, err)
			}
		}

		fmt.Println("All processes stopped")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
