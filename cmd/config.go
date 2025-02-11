package cmd

import (
	"fmt"
	"os"

	"github.com/afomera/dev_spin/internal/userconfig"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dev_spin configuration",
	Long: `Configure dev_spin settings. This includes user-level configuration
such as default organization name.

Example:
  spin config set-org myorg     # Set default organization
  spin config show              # Show current configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show help
		cmd.Help()
	},
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Current Configuration:")
		if config.DefaultOrganization == "" {
			fmt.Println("Default Organization: (not set)")
			fmt.Println("\nTo set a default organization, run:")
			fmt.Println("  spin config set-org <organization>")
		} else {
			fmt.Printf("Default Organization: %s\n", config.DefaultOrganization)
		}
	},
}

// configSetOrgCmd represents the config set-org command
var configSetOrgCmd = &cobra.Command{
	Use:   "set-org [organization]",
	Short: "Set default GitHub organization",
	Long: `Set the default GitHub organization to use when creating or fetching projects.
This will be used when no organization is explicitly specified.

Example:
  spin config set-org myorg`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		orgName := args[0]

		config, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		config.DefaultOrganization = orgName
		if err := config.Save(); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Default organization set to: %s\n", orgName)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetOrgCmd)
}
