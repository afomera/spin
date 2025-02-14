package cmd

import (
	"fmt"
	"os"

	"github.com/afomera/spin/internal/logger"
	"github.com/spf13/cobra"
)

const spinBanner = `
    .-------------------.
    |  .---------------.|
    |  |    ______    ||
    |  |   /     /\   ||
    |  |  /     /  \  ||
    |  | /     /    \ ||
    |  |/     /      \||
    |  |     S P I N  ||
    |  |              ||
    |  |              ||
    |  '---------------'|
    |___________________|
`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spin",
	Short: "A modern development environment manager",
	Long: spinBanner + `
Spin is a CLI tool that helps developers manage their development environments.
It provides commands for setting up, running, and managing applications across different
technology stacks.

Example usage:
  spin setup myapp
  spin up myapp
  spin fetch myapp`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print help
		cmd.Help()
	},
}

func init() {
	var verbose bool
	// Add persistent flags that will be available to all commands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose debug output")

	// Update logger's verbose setting when the flag changes
	cobra.OnInitialize(func() {
		logger.SetVerbose(verbose)
	})
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return nil
}
