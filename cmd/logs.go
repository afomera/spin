package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/process"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs [process-name]",
	Short: "View process logs",
	Long: `View the logs for a running process.
Shows the process output in real-time.

Example:
  spin logs web     # View web process logs
  spin logs worker  # View worker process logs`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		processName := args[0]

		// Load configuration
		cfg, err := config.LoadConfig("spin.config.json")
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Get the process manager instance
		manager := process.GetManager(cfg)

		// Check if process exists
		if _, err := manager.GetProcessStatus(cfg.Name, processName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Find the process to get its log file path
		proc, err := manager.FindProcess(processName)
		if err != nil {
			fmt.Printf("Error finding process: %v\n", err)
			os.Exit(1)
		}

		// Get spin directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Use app-specific log directory
		logFile := filepath.Join(home, ".spin", "output", process.SanitizeAppName(proc.AppName), fmt.Sprintf("%s.log", proc.Name))

		// First show recent output
		tail := exec.Command("tail", "-n", "50", logFile)
		tail.Stdout = os.Stdout
		tail.Stderr = os.Stderr
		if err := tail.Run(); err != nil {
			fmt.Printf("Error showing recent logs: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nShowing live logs (Ctrl+C to exit)...")

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		defer signal.Stop(sigChan)

		// Start following output
		follow := exec.Command("tail", "-f", logFile)
		follow.Stdout = os.Stdout
		follow.Stderr = os.Stderr

		// Start the command
		if err := follow.Start(); err != nil {
			fmt.Printf("Error following logs: %v\n", err)
			os.Exit(1)
		}

		// Wait for Ctrl+C
		go func() {
			<-sigChan
			follow.Process.Kill()
		}()

		// Wait for command to finish
		follow.Wait()
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
