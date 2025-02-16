package cmd

import (
	"fmt"
	"os"

	"github.com/afomera/spin/internal/userconfig"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Spin configuration",
	Long: `Configure Spin settings. This includes user-level configuration
such as default organization name and git URL preferences.

Example:
	 spin config set-org myorg     # Set default organization
	 spin config set-ssh true      # Prefer SSH URLs for git operations
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
		fmt.Printf("Prefer SSH: %v\n", config.PreferSSH)
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

// configSetSSHCmd represents the config set-ssh command
var configSetSSHCmd = &cobra.Command{
	Use:   "set-ssh [true|false]",
	Short: "Set whether to prefer SSH URLs for git operations",
	Long: `Set whether to prefer SSH URLs when performing git operations.
When enabled, SSH URLs will be used instead of HTTPS URLs.

Example:
  spin config set-ssh true    # Enable SSH preference
  spin config set-ssh false   # Disable SSH preference`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		preferSSH := args[0] == "true"

		config, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		config.PreferSSH = preferSSH
		if err := config.Save(); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Prefer SSH set to: %v\n", preferSSH)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetOrgCmd)
	configCmd.AddCommand(configSetSSHCmd)
}
