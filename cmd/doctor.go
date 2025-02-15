package cmd

import (
	"fmt"
	"os/exec"

	"github.com/afomera/spin/internal/logger"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system requirements for Spin",
	Long:  `Check if required dependencies (tmux, docker) are installed and available.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("\nChecking system requirements...\n\n")

		// Check tmux
		if _, err := exec.LookPath("tmux"); err == nil {
			fmt.Printf("  %s✓%s tmux: %sinstalled%s\n", logger.Green, logger.Reset, logger.Cyan, logger.Reset)
		} else {
			fmt.Printf("  %s⚠%s tmux: %snot found%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
		}

		// Check docker
		if _, err := exec.LookPath("docker"); err == nil {
			// Check if docker daemon is running
			cmd := exec.Command("docker", "info")
			if err := cmd.Run(); err == nil {
				fmt.Printf("  %s✓%s docker: %srunning%s\n", logger.Green, logger.Reset, logger.Cyan, logger.Reset)
			} else {
				fmt.Printf("  %s⚠%s docker: %sinstalled but not running%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
				fmt.Printf("  %s→%s please start Docker Desktop to use docker features%s\n", logger.Blue, logger.Reset, logger.Reset)
			}
		} else {
			fmt.Printf("  %s⚠%s docker: %snot found%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
