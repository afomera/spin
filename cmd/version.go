package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Spin",
	Long:  `Display the current version of the Spin CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("spin version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
