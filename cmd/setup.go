package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/logger"
	"github.com/afomera/spin/internal/userconfig"
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
				fmt.Fprintf(os.Stderr, "%sError getting current directory: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
			// Extract app name from directory
			appName = filepath.Base(appPath)
		} else {
			appPath = appName
			// Create project directory if it doesn't exist
			if err := os.MkdirAll(appPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "%sError creating directory: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
		}

		configPath := filepath.Join(appPath, "spin.config.json")
		if config.Exists(configPath) && !force {
			fmt.Printf("%sWarning: spin.config.json already exists in %s%s\n", logger.Yellow, appPath, logger.Reset)
			fmt.Printf("%sDo you want to overwrite it? (y/N)%s\n", logger.Blue, logger.Reset)

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sError reading input: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Printf("%sSetup cancelled%s\n", logger.Yellow, logger.Reset)
				os.Exit(0)
			}
		}

		// Load user configuration for default organization
		userCfg, err := userconfig.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading user configuration: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Parse repository information
		var repo *config.Repository
		if repoFlag != "" {
			repo, err = config.ParseRepositoryString(repoFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sError parsing repository: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
		} else {
			defaultOrg := userCfg.DefaultOrganization
			if defaultOrg == "" {
				fmt.Printf("%sNo default organization set. Please either:%s\n", logger.Yellow, logger.Reset)
				fmt.Printf("1. Set a default organization: %sspin config set-org <organization>%s\n", logger.Cyan, logger.Reset)
				fmt.Printf("2. Specify repository with --repo flag: %sspin setup %s --repo=org/name%s\n",
					logger.Cyan, appName, logger.Reset)
				os.Exit(1)
			}
			repo = &config.Repository{
				Organization: defaultOrg,
				Name:         appName,
			}
		}

		// Detect project type and configuration
		fmt.Printf("\n%sAnalyzing project structure...%s\n", logger.Blue, logger.Reset)
		detected, err := config.DetectProjectType(appPath)
		if err != nil {
			fmt.Printf("%sWarning: Could not detect project type: %v%s\n", logger.Yellow, err, logger.Reset)
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
			fmt.Printf("\n%sDetected Rails application:%s\n", logger.Blue, logger.Reset)

			// Ruby version
			if detected.Rails.Ruby.Version != "" {
				fmt.Printf("  %s✓%s Ruby Version: %s%s%s\n", logger.Green, logger.Reset, logger.Cyan, detected.Rails.Ruby.Version, logger.Reset)
			} else {
				fmt.Printf("  %s⚠%s Ruby Version: %snot found%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
			}

			// Rails version
			if detected.Rails.Rails.Version != "" {
				fmt.Printf("  %s✓%s Rails Version: %s%s%s\n", logger.Green, logger.Reset, logger.Cyan, detected.Rails.Rails.Version, logger.Reset)
			} else {
				fmt.Printf("  %s⚠%s Rails Version: %snot found%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
			}

			// Database
			if detected.Rails.Database.Type != "" {
				fmt.Printf("  %s✓%s Database: %s%s%s\n", logger.Green, logger.Reset, logger.Cyan, detected.Rails.Database.Type, logger.Reset)
				for key, value := range detected.Rails.Database.Settings {
					fmt.Printf("    %s-%s %s: %s%s%s\n", logger.Blue, logger.Reset, key, logger.Cyan, value, logger.Reset)
				}
			} else {
				fmt.Printf("  %s⚠%s Database: %snot configured%s\n", logger.Yellow, logger.Reset, logger.Red, logger.Reset)
			}

			// Services
			if detected.Rails.Services.Redis {
				fmt.Printf("  %s✓%s Redis: %senabled%s\n", logger.Green, logger.Reset, logger.Cyan, logger.Reset)
			}
			if detected.Rails.Services.Sidekiq {
				fmt.Printf("  %s✓%s Sidekiq: %senabled%s\n", logger.Green, logger.Reset, logger.Cyan, logger.Reset)
			}

			// Scripts
			fmt.Printf("\n%sGenerated Scripts:%s\n", logger.Blue, logger.Reset)
			fmt.Printf("  %ssetup:%s %s\n", logger.Purple, logger.Reset, detected.Scripts.Setup)
			fmt.Printf("  %sstart:%s %s\n", logger.Purple, logger.Reset, detected.Scripts.Start)
			fmt.Printf("  %stest:%s  %s\n", logger.Purple, logger.Reset, detected.Scripts.Test)
		}

		// Save configuration
		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating config file: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("\n%s✨ Successfully initialized %s%s%s\n", logger.Green, logger.Cyan, appName, logger.Reset)
		fmt.Printf("%sRepository:%s %s\n", logger.Blue, logger.Reset, cfg.Repository.GetFullName())
		fmt.Printf("%sConfiguration:%s %s\n", logger.Blue, logger.Reset, configPath)

		fmt.Printf("\n%sNext steps:%s\n", logger.Purple, logger.Reset)
		fmt.Printf("  %s1.%s cd %s%s%s\n", logger.Yellow, logger.Reset, logger.Cyan, appName, logger.Reset)
		fmt.Printf("  %s2.%s Edit %sspin.config.json%s to customize your project\n", logger.Yellow, logger.Reset, logger.Cyan, logger.Reset)
		fmt.Printf("  %s3.%s Run %sspin up%s to start development\n", logger.Yellow, logger.Reset, logger.Cyan, logger.Reset)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringVar(&repoFlag, "repo", "", "Repository in format organization/name")
	setupCmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing configuration")
}
