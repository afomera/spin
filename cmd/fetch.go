package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	fetchRepoFlag string // Flag to specify repository in org/name format
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [app-name]",
	Short: "Clone and setup an existing application",
	Long: `Fetch clones an existing application repository and sets it up for development.
It expects the application to have a spin.config.json file.

Example:
  spin fetch myapp
  spin fetch myapp --repo=myorg/myapp`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]

		// Check if git is installed
		if _, err := exec.LookPath("git"); err != nil {
			fmt.Println("Error: git is not installed")
			os.Exit(1)
		}

		// Load user configuration for default organization
		userCfg, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Error loading user configuration: %v\n", err)
			os.Exit(1)
		}

		// Parse repository information if provided via flag
		var repo *config.Repository
		if fetchRepoFlag != "" {
			repo, err = config.ParseRepositoryString(fetchRepoFlag)
			if err != nil {
				fmt.Printf("Error parsing repository: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Use default organization from user config, or prompt if not set
			defaultOrg := userCfg.DefaultOrganization
			if defaultOrg == "" {
				fmt.Println("No default organization set. Please either:")
				fmt.Println("1. Set a default organization: spin config set-org <organization>")
				fmt.Println("2. Specify repository with --repo flag: spin fetch", appName, "--repo=org/name")
				os.Exit(1)
			}
			repo = &config.Repository{
				Organization: defaultOrg,
				Name:         appName,
			}
		}

		// Clone the repository
		fmt.Printf("Cloning repository %s...\n", repo.GetFullName())
		gitCmd := exec.Command("git", "clone", repo.GetCloneURL(), appName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr

		if err := gitCmd.Run(); err != nil {
			fmt.Printf("Error cloning repository: %v\n", err)
			os.Exit(1)
		}

		// Check for spin.config.json
		configPath := filepath.Join(appName, "spin.config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Println("Warning: No spin.config.json found in repository")
			fmt.Println("Creating default configuration...")

			// Create default config with repository information
			cfg := &config.Config{
				Name:    appName,
				Version: "1.0.0",
				Type:    "default",
				Repository: config.Repository{
					Organization: repo.Organization,
					Name:         repo.Name,
				},
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

			if err := cfg.Save(configPath); err != nil {
				fmt.Printf("Error creating config file: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Load existing config and update repository information if needed
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				fmt.Printf("Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			// Update repository information if it was provided via flag
			if fetchRepoFlag != "" {
				cfg.Repository = *repo
				if err := cfg.Save(configPath); err != nil {
					fmt.Printf("Error updating config file: %v\n", err)
					os.Exit(1)
				}
			}
		}

		fmt.Printf("Successfully fetched %s\n", appName)
		fmt.Printf("Repository: %s\n", repo.GetFullName())
		fmt.Println("\nNext steps:")
		fmt.Println("  1. cd", appName)
		fmt.Println("  2. Review spin.config.json")
		fmt.Println("  3. Run 'spin up' to start development")
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVar(&fetchRepoFlag, "repo", "", "Repository in format organization/name")
}
