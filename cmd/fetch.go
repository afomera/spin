package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/afomera/spin/internal/config"
	lg "github.com/afomera/spin/internal/logger"
	"github.com/afomera/spin/internal/script"
	"github.com/afomera/spin/internal/userconfig"
	"github.com/spf13/cobra"
)

var (
	fetchRepoFlag string // Flag to specify repository in org/name format
	skipSetup     bool   // Flag to skip running setup scripts
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [app-name]",
	Short: "Clone and setup an existing application",
	Long: `Fetch clones an existing application repository and sets it up for development.
It expects the application to have a spin.config.json file.

If run inside a repository with a spin.config.json file, it will fetch the latest changes.
Otherwise, it will clone the repository and set it up.

Example:
  spin fetch myapp
  spin fetch myapp --repo=myorg/myapp
  spin fetch (in a repository with spin.config.json)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if git is installed
		if _, err := exec.LookPath("git"); err != nil {
			fmt.Printf("%sError: git is not installed%s\n", lg.Red, lg.Reset)
			os.Exit(1)
		}

		// Load user configuration
		userCfg, err := userconfig.Load()
		if err != nil {
			fmt.Printf("%sError loading user configuration: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// Check if we're in a git repository with spin.config.json
		if _, err := os.Stat(".git"); err == nil {
			if _, err := os.Stat("spin.config.json"); err == nil {
				// We're in a repository with spin.config.json, fetch latest changes
				cfg, err := config.LoadConfig("spin.config.json")
				if err != nil {
					fmt.Printf("%sError loading spin.config.json: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}

				// Get current branch
				branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
				branchOutput, err := branchCmd.Output()
				if err != nil {
					fmt.Printf("%sError getting current branch: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}
				currentBranch := strings.TrimSpace(string(branchOutput))
				fmt.Printf("%sFetching latest changes for %s%s%s...\n", lg.Blue, lg.Cyan, cfg.Repository.GetFullName(), lg.Reset)
				fetchCmd := exec.Command("git", "fetch", "origin", currentBranch)
				fetchCmd.Stdout = os.Stdout
				fetchCmd.Stderr = os.Stderr
				if err := fetchCmd.Run(); err != nil {
					fmt.Printf("%sError fetching changes: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}

				// Merge changes
				mergeCmd := exec.Command("git", "merge", fmt.Sprintf("origin/%s", currentBranch))
				mergeCmd.Stdout = os.Stdout
				mergeCmd.Stderr = os.Stderr
				if err := mergeCmd.Run(); err != nil {
					fmt.Printf("%sError merging changes: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}

				fmt.Printf("%s✨ Successfully updated %s%s%s\n", lg.Green, lg.Cyan, cfg.Repository.GetFullName(), lg.Reset)
				return
			}
		}

		// Not in a repository, proceed with clone
		if len(args) == 0 {
			fmt.Printf("%sError: app name is required when not in a repository%s\n", lg.Red, lg.Reset)
			os.Exit(1)
		}
		appName := args[0]

		// Parse repository information if provided via flag
		var repo *config.Repository
		if fetchRepoFlag != "" {
			repo, err = config.ParseRepositoryString(fetchRepoFlag)
			if err != nil {
				fmt.Printf("%sError parsing repository: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}
		} else {
			// Use default organization from user config, or prompt if not set
			defaultOrg := userCfg.DefaultOrganization
			if defaultOrg == "" {
				fmt.Printf("%sNo default organization set. Please either:%s\n", lg.Yellow, lg.Reset)
				fmt.Printf("1. Set a default organization: %sspin config set-org <organization>%s\n", lg.Cyan, lg.Reset)
				fmt.Printf("2. Specify repository with --repo flag: %sspin fetch %s --repo=org/name%s\n", lg.Cyan, appName, lg.Reset)
				os.Exit(1)
			}
			repo = &config.Repository{
				Organization: defaultOrg,
				Name:         appName,
			}
		}

		// Clone the repository
		fmt.Printf("%sCloning repository %s%s%s...\n", lg.Blue, lg.Cyan, repo.GetFullName(), lg.Reset)
		gitCmd := exec.Command("git", "clone", repo.GetCloneURL(userCfg.PreferSSH), appName)
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr

		if err := gitCmd.Run(); err != nil {
			fmt.Printf("%sError cloning repository: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// Check for spin.config.json
		configPath := filepath.Join(appName, "spin.config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Printf("%sNo spin.config.json found, running project detection...%s\n", lg.Blue, lg.Reset)

			// Change to the app directory to run init
			if err := os.Chdir(appName); err != nil {
				fmt.Printf("%sError changing to app directory: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}

			// Run the init command in the current directory
			initArgs := []string{"."}
			initCmd.Run(initCmd, initArgs)

			// Change back to the original directory
			if err := os.Chdir(".."); err != nil {
				fmt.Printf("%sError changing back to original directory: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}
		} else {
			// Load existing config and update repository information if needed
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				fmt.Printf("%sError loading configuration: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}

			// Update repository information if it was provided via flag
			if fetchRepoFlag != "" {
				cfg.Repository = *repo
				if err := cfg.Save(configPath); err != nil {
					fmt.Printf("%sError updating config file: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}
			}
		}

		// Load the final config to check for setup scripts
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("%sError loading configuration: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// Run setup scripts if they exist and not skipped
		if !skipSetup {
			if setupScript, ok := cfg.Scripts["setup"]; ok {
				fmt.Printf("\n%sRunning setup script...%s\n", lg.Blue, lg.Reset)

				// Create a new script instance
				s := &script.Script{
					Name:        "setup",
					Command:     setupScript.Command,
					Description: setupScript.Description,
					Env:         setupScript.Env,
				}

				// Execute the script with the proper working directory
				opts := &script.RunOptions{
					WorkDir: appName,
					Env:     cfg.GetEnvVars("development"),
				}

				if err := s.Execute(opts); err != nil {
					fmt.Printf("%sError running setup script: %v%s\n", lg.Red, err, lg.Reset)
					os.Exit(1)
				}
				fmt.Printf("%sSetup completed successfully%s\n", lg.Green, lg.Reset)
			}
		}

		fmt.Printf("\n%s✨ Successfully fetched %s%s%s\n", lg.Green, lg.Cyan, appName, lg.Reset)
		fmt.Printf("%sRepository:%s %s\n", lg.Blue, lg.Reset, repo.GetFullName())

		fmt.Printf("\n%sNext steps:%s\n", lg.Purple, lg.Reset)
		if skipSetup {
			fmt.Printf("  %s1.%s cd %s%s%s\n", lg.Yellow, lg.Reset, lg.Cyan, appName, lg.Reset)
			fmt.Printf("  %s2.%s Review %sspin.config.json%s\n", lg.Yellow, lg.Reset, lg.Cyan, lg.Reset)
			fmt.Printf("  %s3.%s Run %sspin setup%s to install dependencies\n", lg.Yellow, lg.Reset, lg.Cyan, lg.Reset)
			fmt.Printf("  %s4.%s Run %sspin up%s to start development\n", lg.Yellow, lg.Reset, lg.Cyan, lg.Reset)
		} else {
			fmt.Printf("  %s1.%s cd %s%s%s\n", lg.Yellow, lg.Reset, lg.Cyan, appName, lg.Reset)
			fmt.Printf("  %s2.%s Run %sspin up%s to start development\n", lg.Yellow, lg.Reset, lg.Cyan, lg.Reset)
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVar(&fetchRepoFlag, "repo", "", "Repository in format organization/name")
	fetchCmd.Flags().BoolVar(&skipSetup, "skip-setup", false, "Skip running setup scripts")
}
