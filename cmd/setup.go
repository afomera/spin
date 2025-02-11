package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afomera/dev_spin/internal/config"
	"github.com/afomera/dev_spin/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	repoFlag string // Flag to specify repository in org/name format
	force    bool   // Flag to force overwrite existing configuration
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup [app-name]",
	Short: "Setup a new application",
	Long: `Setup initializes a new application with the specified name.
It creates the project directory, initializes a new spin.config.json file,
and sets up the basic project structure.

Example:
  spin setup myapp
  spin setup myapp --repo=myorg/myapp
  spin setup . --force    # Setup in current directory`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		var appPath string

		// Handle setup in current directory if "." is specified
		if appName == "." {
			var err error
			appPath, err = os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current directory: %v\n", err)
				os.Exit(1)
			}
			// Extract app name from directory
			appName = filepath.Base(appPath)
		} else {
			appPath = appName
			// Create project directory if it doesn't exist
			if err := os.MkdirAll(appPath, 0755); err != nil {
				fmt.Printf("Error creating directory: %v\n", err)
				os.Exit(1)
			}
		}

		configPath := filepath.Join(appPath, "spin.config.json")
		if config.Exists(configPath) && !force {
			fmt.Printf("Warning: spin.config.json already exists in %s\n", appPath)
			fmt.Println("Do you want to overwrite it? (y/N)")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				os.Exit(1)
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Println("Setup cancelled")
				os.Exit(0)
			}
		}

		// Load user configuration for default organization
		userCfg, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Error loading user configuration: %v\n", err)
			os.Exit(1)
		}

		// Parse repository information
		var repo *config.Repository
		if repoFlag != "" {
			repo, err = config.ParseRepositoryString(repoFlag)
			if err != nil {
				fmt.Printf("Error parsing repository: %v\n", err)
				os.Exit(1)
			}
		} else {
			defaultOrg := userCfg.DefaultOrganization
			if defaultOrg == "" {
				fmt.Println("No default organization set. Please either:")
				fmt.Println("1. Set a default organization: spin config set-org <organization>")
				fmt.Println("2. Specify repository with --repo flag: spin setup", appName, "--repo=org/name")
				os.Exit(1)
			}
			repo = &config.Repository{
				Organization: defaultOrg,
				Name:         appName,
			}
		}

		// Detect project type and configuration
		fmt.Println("\nAnalyzing project structure...")
		detected, err := config.DetectProjectType(appPath)
		if err != nil {
			fmt.Printf("Warning: Could not detect project type: %v\n", err)
			detected = &config.Config{
				Type: "unknown",
			}
		}

		// Create initial configuration
		cfg := detected
		if cfg == nil {
			cfg = &config.Config{
				Name:    appName,
				Version: "1.0.0",
				Type:    "unknown",
				Dependencies: config.Dependencies{
					Services: []string{},
					Tools:    []string{},
				},
				Scripts: config.Scripts{
					Setup: "",
					Start: "",
					Test:  "",
				},
				Env: map[string]config.EnvMap{
					"development": {},
				},
			}
		}

		// Set repository information
		cfg.Name = appName
		cfg.Repository = *repo

		// Add detected configurations
		if detected != nil && detected.Rails != nil {
			fmt.Println("\nDetected Rails application:")

			// Ruby version
			if detected.Rails.Ruby.Version != "" {
				fmt.Printf("  ✓ Ruby Version: %s\n", detected.Rails.Ruby.Version)
			} else {
				fmt.Println("  ⚠ Ruby Version: not found")
			}

			// Rails version
			if detected.Rails.Rails.Version != "" {
				fmt.Printf("  ✓ Rails Version: %s\n", detected.Rails.Rails.Version)
			} else {
				fmt.Println("  ⚠ Rails Version: not found")
			}

			// Database
			if detected.Rails.Database.Type != "" {
				fmt.Printf("  ✓ Database: %s\n", detected.Rails.Database.Type)
				for key, value := range detected.Rails.Database.Settings {
					fmt.Printf("    - %s: %s\n", key, value)
				}
			} else {
				fmt.Println("  ⚠ Database: not configured")
			}

			// Services
			if detected.Rails.Services.Redis {
				fmt.Println("  ✓ Redis: enabled")
			}
			if detected.Rails.Services.Sidekiq {
				fmt.Println("  ✓ Sidekiq: enabled")
			}

			// Scripts
			fmt.Println("\nGenerated Scripts:")
			fmt.Printf("  setup: %s\n", detected.Scripts.Setup)
			fmt.Printf("  start: %s\n", detected.Scripts.Start)
			fmt.Printf("  test:  %s\n", detected.Scripts.Test)
		}

		// Save configuration
		if err := cfg.Save(configPath); err != nil {
			fmt.Printf("Error creating config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nSuccessfully initialized %s\n", appName)
		fmt.Printf("Repository: %s\n", cfg.Repository.GetFullName())
		fmt.Printf("Configuration: %s\n", configPath)

		fmt.Println("\nNext steps:")
		fmt.Println("  1. cd", appName)
		fmt.Println("  2. Edit spin.config.json to customize your project")
		fmt.Println("  3. Run 'spin up' to start development")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringVar(&repoFlag, "repo", "", "Repository in format organization/name")
	setupCmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing configuration")
}
