package cmd

import (
	"fmt"
	"os"

	"github.com/afomera/dev_spin/internal/process"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:   "debug [process-name]",
	Short: "Debug a running process",
	Long: `Debug allows you to attach to a running process in interactive mode.
This is particularly useful for processes that require input, like Rails console
or debugging sessions.

Example:
  spin debug web     # Debug the web process (e.g., when hitting binding.irb)
  spin debug console # Attach to a Rails console session`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		processName := args[0]

		fmt.Printf("Attaching to process '%s' in debug mode...\n", processName)
		fmt.Println("Press Ctrl+C to send interrupt to the process")
		fmt.Println("Press Ctrl+D to detach")

		// Get the process manager instance
		manager := process.GetManager(nil)

		// Get current terminal settings
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		// Try to attach to the process in debug mode
		if err := manager.DebugProcess(processName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
}
