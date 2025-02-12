package cmd

import (
	"fmt"

	"github.com/afomera/spin/internal/process"
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
