package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/process"
	"github.com/afomera/spin/internal/service"
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [app-name]",
	Short: "Start the development environment",
	Long: `Up starts the development environment for the specified application.
It reads the spin.config.json file, sets up environment variables,
and executes the start script.

Example:
  spin up myapp`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// If no app name is provided, use current directory
		appPath := "."
		if len(args) > 0 {
			appPath = args[0]
		}

		// Load configuration
		configPath := filepath.Join(appPath, "spin.config.json")
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Initialize service manager and required services
		svcManager := service.NewServiceManager()
		if len(cfg.Dependencies.Services) > 0 {
			fmt.Println("Checking required services...")
			for _, serviceName := range cfg.Dependencies.Services {
				svc, err := service.CreateService(serviceName)
				if err != nil {
					fmt.Printf("Error creating service %s: %v\n", serviceName, err)
					os.Exit(1)
				}
				svcManager.RegisterService(svc)

				if !svc.IsRunning() {
					fmt.Printf("Starting %s...\n", serviceName)
					if err := svcManager.StartService(serviceName); err != nil {
						fmt.Printf("Error starting service %s: %v\n", serviceName, err)
						os.Exit(1)
					}
				} else {
					fmt.Printf("Service %s is already running\n", serviceName)
				}
			}
		}

		// Check if start script is defined
		if cfg.Scripts.Start == "" {
			fmt.Println("Error: No start script defined in spin.config.json")
			fmt.Println("Add a start script to your configuration:")
			fmt.Println(`{
  "scripts": {
    "start": "your-start-command"
  }
}`)
			os.Exit(1)
		}

		// Set up environment variables
		envVars := cfg.GetEnvVars("development")
		env := os.Environ() // Get existing environment
		for key, value := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}

		// Create command from start script
		// Split the command string into command and arguments
		parts := strings.Fields(cfg.Scripts.Start)
		if len(parts) == 0 {
			fmt.Println("Error: Invalid start script")
			os.Exit(1)
		}

		command := parts[0]
		var cmdArgs []string
		if len(parts) > 1 {
			cmdArgs = parts[1:]
		}

		// Get process manager
		processManager := process.GetManager(cfg)

		// Run bundle install if Gemfile exists
		if _, err := os.Stat(filepath.Join(appPath, "Gemfile")); err == nil {
			fmt.Println("Running bundle install...")
			bundleCmd := exec.Command("bundle", "install")
			bundleCmd.Dir = appPath
			bundleCmd.Stdout = os.Stdout
			bundleCmd.Stderr = os.Stderr
			if err := bundleCmd.Run(); err != nil {
				fmt.Printf("Error running bundle install: %v\n", err)
				os.Exit(1)
			}

			// Run database migrations
			fmt.Println("Running database migrations...")
			migrateCmd := exec.Command("bundle", "exec", "rails", "db:migrate")
			migrateCmd.Dir = appPath
			migrateCmd.Stdout = os.Stdout
			migrateCmd.Stderr = os.Stderr
			if err := migrateCmd.Run(); err != nil {
				fmt.Printf("Error running migrations: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Starting development environment for %s...\n", cfg.Name)

		// Start the process
		if err := processManager.StartProcess("web", command, cmdArgs, env, appPath); err != nil {
			fmt.Printf("Error starting development server: %v\n", err)
			os.Exit(1)
		}

		// If we have a Procfile.dev, start those processes too
		procfilePath := filepath.Join(appPath, "Procfile.dev")
		if _, err := os.Stat(procfilePath); err == nil {
			// Parse Procfile.dev
			procfile, err := os.Open(procfilePath)
			if err != nil {
				fmt.Printf("Error reading Procfile.dev: %v\n", err)
				os.Exit(1)
			}
			defer procfile.Close()

			scanner := bufio.NewScanner(procfile)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					continue
				}

				procName := strings.TrimSpace(parts[0])
				procCommand := strings.TrimSpace(parts[1])

				// Skip web process as it's already started
				if procName == "web" {
					continue
				}

				// Split the command into command and arguments
				cmdParts := strings.Fields(procCommand)
				if len(cmdParts) == 0 {
					continue
				}

				command := cmdParts[0]
				var args []string
				if len(cmdParts) > 1 {
					args = cmdParts[1:]
				}

				if err := processManager.StartProcess(procName, command, args, env, appPath); err != nil {
					fmt.Printf("Error starting process %s: %v\n", procName, err)
					os.Exit(1)
				}
			}
		}

		fmt.Println("Press Ctrl+C to stop all processes")

		// Handle signals for graceful shutdown
		processManager.HandleSignals()

		// Wait for all processes to complete
		processManager.WaitForAll()

		// Stop services if they were started by us
		if len(cfg.Dependencies.Services) > 0 {
			fmt.Println("Stopping services...")
			svcManager.StopAll()
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
