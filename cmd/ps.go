package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/afomera/spin/internal/config"
	lg "github.com/afomera/spin/internal/logger"
	"github.com/afomera/spin/internal/process"
	"github.com/spf13/cobra"
)

// colorizeStatus returns a colored string representation of a process status
func colorizeStatus(status process.ProcessStatus) string {
	statusStr := string(status)
	switch status {
	case process.StatusRunning:
		return fmt.Sprintf("%s%s%s", lg.Green, statusStr, lg.Reset)
	case process.StatusStopped:
		return fmt.Sprintf("%s%s%s", lg.Red, statusStr, lg.Reset)
	case process.StatusError:
		return fmt.Sprintf("%s%s%s", lg.Red, statusStr, lg.Reset)
	default:
		return fmt.Sprintf("%s%s%s", lg.Yellow, statusStr, lg.Reset)
	}
}

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

		// Print headers with cyan color
		fmt.Fprintf(w, "%sAPP\tNAME\tSTATUS\tPID\tOUTPUT FILE\tINTERACTIVE\tERROR%s\n",
			lg.Cyan,
			lg.Reset,
		)

		// Load configuration
		cfg, err := config.LoadConfig("spin.config.json")
		if err != nil {
			fmt.Printf("%sError loading configuration: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// Get all processes from the manager
		manager := process.GetManager(cfg)
		processes := manager.ListProcesses()

		if len(processes) == 0 {
			fmt.Fprintf(w, "%sNo running processes%s\n", lg.Yellow, lg.Reset)
		} else {
			for _, p := range processes {
				interactive := "no"
				if p.IsDebug {
					interactive = "yes"
				}

				errStr := ""
				if p.Error != nil {
					errStr = fmt.Sprintf("%s%s%s", lg.Red, p.Error.Error(), lg.Reset)
				}

				pid := 0
				if p.Command != nil && p.Command.Process != nil {
					pid = p.Command.Process.Pid
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
					p.AppName,
					p.Name,
					colorizeStatus(p.Status),
					pid,
					fmt.Sprintf("~/.spin/output/%s/%s.log", process.SanitizeAppName(p.AppName), p.Name),
					interactive,
					errStr,
				)
			}
		}

		w.Flush()

		// Print help text with blue color
		fmt.Printf("\n%sTo view process output:%s\n", lg.Blue, lg.Reset)
		fmt.Printf("  spin logs <process-name>\n")
		fmt.Printf("\n%sTo debug a process:%s\n", lg.Blue, lg.Reset)
		fmt.Printf("  spin debug <process-name>\n")
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
