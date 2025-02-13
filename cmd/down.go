package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/afomera/spin/internal/config"
	lg "github.com/afomera/spin/internal/logger"
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
				fmt.Printf("%sStopping services...%s\n", lg.Blue, lg.Reset)
				for _, serviceName := range cfg.Dependencies.Services {
					svc, err := service.CreateService(serviceName, cfg)
					if err != nil {
						fmt.Printf("%sWarning: Failed to create service %s: %v%s\n", lg.Yellow, serviceName, err, lg.Reset)
						continue
					}
					svcManager.RegisterService(svc)

					if svc.IsRunning() {
						fmt.Printf("Stopping %s%s%s...\n", lg.Cyan, serviceName, lg.Reset)
						if err := svcManager.StopService(serviceName); err != nil {
							fmt.Printf("%sWarning: Failed to stop service %s: %v%s\n", lg.Yellow, serviceName, err, lg.Reset)
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
			fmt.Printf("%sNo running processes%s\n", lg.Yellow, lg.Reset)
			return
		}

		fmt.Printf("%sStopping all processes...%s\n", lg.Blue, lg.Reset)
		for _, p := range processes {
			fmt.Printf("Stopping %s%s%s...\n", lg.Cyan, p.Name, lg.Reset)
			if err := manager.StopProcess(p.Name); err != nil {
				fmt.Printf("%sWarning: Failed to stop %s: %v%s\n", lg.Yellow, p.Name, err, lg.Reset)
			}
		}

		fmt.Printf("%sAll processes stopped%s\n", lg.Green, lg.Reset)
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
