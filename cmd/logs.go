package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

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

		// Get the process manager instance
		manager := process.GetManager(nil)

		// Check if process exists
		if _, err := manager.GetProcessStatus(processName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Get spin directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Construct log file path
		logFile := filepath.Join(home, ".spin", "output", fmt.Sprintf("%s.log", processName))

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
