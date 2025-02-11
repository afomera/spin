package cmd

import (
	"fmt"
	"os"

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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return nil
}
