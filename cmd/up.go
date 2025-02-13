package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/afomera/spin/internal/config"
	lg "github.com/afomera/spin/internal/logger"
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
			fmt.Printf("%sError loading configuration: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// Initialize service manager and required services
		svcManager := service.NewServiceManager()
		if len(cfg.Dependencies.Services) > 0 {
			fmt.Printf("%sChecking required services...%s\n", lg.Blue, lg.Reset)
			for _, serviceName := range cfg.Dependencies.Services {
				svc, err := service.CreateService(serviceName, cfg)
				if err != nil {
					fmt.Printf("%sError creating service %s: %v%s\n", lg.Red, serviceName, err, lg.Reset)
					os.Exit(1)
				}
				svcManager.RegisterService(svc)

				if !svc.IsRunning() {
					fmt.Printf("Starting %s%s%s...\n", lg.Cyan, serviceName, lg.Reset)
					if err := svcManager.StartService(serviceName); err != nil {
						fmt.Printf("%sError starting service %s: %v%s\n", lg.Red, serviceName, err, lg.Reset)
						os.Exit(1)
					}
				} else {
					fmt.Printf("%sService %s%s%s is already running%s\n", lg.Green, lg.Cyan, serviceName, lg.Green, lg.Reset)
				}
			}
		}

		// Check if start script is defined
		if cfg.Scripts.Start == "" {
			fmt.Printf("%sError: No start script defined in spin.config.json%s\n", lg.Red, lg.Reset)
			fmt.Printf("%sAdd a start script to your configuration:%s\n", lg.Yellow, lg.Reset)
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
			fmt.Printf("%sError: Invalid start script%s\n", lg.Red, lg.Reset)
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
			fmt.Printf("%sRunning bundle install...%s\n", lg.Blue, lg.Reset)
			bundleCmd := exec.Command("bundle", "install")
			bundleCmd.Dir = appPath
			bundleCmd.Stdout = os.Stdout
			bundleCmd.Stderr = os.Stderr
			if err := bundleCmd.Run(); err != nil {
				fmt.Printf("%sError running bundle install: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}

			// Run database migrations
			fmt.Printf("%sRunning database migrations...%s\n", lg.Blue, lg.Reset)
			migrateCmd := exec.Command("bundle", "exec", "rails", "db:migrate")
			migrateCmd.Dir = appPath
			migrateCmd.Stdout = os.Stdout
			migrateCmd.Stderr = os.Stderr
			if err := migrateCmd.Run(); err != nil {
				fmt.Printf("%sError running migrations: %v%s\n", lg.Red, err, lg.Reset)
				os.Exit(1)
			}
		}

		fmt.Printf("%sStarting development environment for %s%s%s...%s\n", lg.Blue, lg.Cyan, cfg.Name, lg.Blue, lg.Reset)

		// Start the web process
		webCmd := command
		if len(cmdArgs) > 0 {
			webCmd += " " + strings.Join(cmdArgs, " ")
		}
		fmt.Printf("%s-> Starting web: %s%s\n", lg.Blue, webCmd, lg.Reset)

		if err := processManager.StartProcess("web", command, cmdArgs, env, appPath); err != nil {
			fmt.Printf("%sError starting development server: %v%s\n", lg.Red, err, lg.Reset)
			os.Exit(1)
		}

		// If we have a Procfile.dev, start those processes too
		procfilePath := filepath.Join(appPath, "Procfile.dev")
		if _, err := os.Stat(procfilePath); err == nil {
			fmt.Printf("\n%sDetected Procfile.dev%s\n", lg.Blue, lg.Reset)

			// Parse Procfile.dev
			procfile, err := os.Open(procfilePath)
			if err != nil {
				fmt.Printf("%sError reading Procfile.dev: %v%s\n", lg.Red, err, lg.Reset)
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

				// Special handling for npm-related commands to preserve colons and other special characters
				var command string
				var args []string

				if strings.HasPrefix(procCommand, "yarn ") ||
					strings.HasPrefix(procCommand, "npm ") ||
					strings.HasPrefix(procCommand, "npx ") {
					// For npm-related commands, keep the command intact
					parts := strings.SplitN(procCommand, " ", 2)
					command = parts[0] // yarn, npm, or npx
					if len(parts) > 1 {
						// Keep the rest as a single argument to preserve colons and other special characters
						args = []string{parts[1]}
					}
				} else {
					// For other commands, split normally
					cmdParts := strings.Fields(procCommand)
					if len(cmdParts) == 0 {
						continue
					}
					command = cmdParts[0]
					if len(cmdParts) > 1 {
						args = cmdParts[1:]
					}
				}

				// Log the process we're about to start
				processCmd := command
				if len(args) > 0 {
					processCmd += " " + strings.Join(args, " ")
				}
				fmt.Printf("%s-> Starting %s: %s%s\n", lg.Blue, procName, processCmd, lg.Reset)

				if err := processManager.StartProcess(procName, command, args, env, appPath); err != nil {
					fmt.Printf("%sError starting process %s: %v%s\n", lg.Red, procName, err, lg.Reset)
					os.Exit(1)
				}
			}
		}

		fmt.Printf("\n%sPress Ctrl+C to stop all processes%s\n", lg.Yellow, lg.Reset)

		// Handle signals for graceful shutdown
		processManager.HandleSignals()

		// Wait for all processes to complete
		processManager.WaitForAll()

		// Stop services if they were started by us
		if len(cfg.Dependencies.Services) > 0 {
			fmt.Printf("%sStopping services...%s\n", lg.Blue, lg.Reset)
			svcManager.StopAll()
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
