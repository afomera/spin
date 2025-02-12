package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/afomera/spin/internal/process"
	"github.com/spf13/cobra"
)

// psCmd represents the ps command
var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running processes",
	Long: `List all running processes in the current development environment.
Shows process names, statuses, and additional information.

Example:
  spin ps     # List all processes`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create a new tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tPID\tOUTPUT FILE\tINTERACTIVE\tERROR")

		// Get all processes from the manager
		manager := process.GetManager(nil)
		processes := manager.ListProcesses()

		if len(processes) == 0 {
			fmt.Fprintln(w, "No running processes")
		} else {
			for _, p := range processes {
				interactive := "no"
				if p.IsDebug {
					interactive = "yes"
				}

				errStr := ""
				if p.Error != nil {
					errStr = p.Error.Error()
				}

				pid := 0
				if p.Command != nil && p.Command.Process != nil {
					pid = p.Command.Process.Pid
				}

				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
					p.Name,
					p.Status,
					pid,
					p.OutputFile,
					interactive,
					errStr,
				)
			}
		}

		w.Flush()

		// Print help text
		fmt.Println("\nTo view process output:")
		fmt.Println("  spin logs <process-name>")
		fmt.Println("\nTo debug a process:")
		fmt.Println("  spin debug <process-name>")
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
