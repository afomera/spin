package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/afomera/spin/internal/script"
)

var (
	scriptEnv     []string
	workDir       string
	skipHookError bool
)

func init() {
	// Add scripts command
	rootCmd.AddCommand(scriptsCmd)

	// Add subcommands
	scriptsCmd.AddCommand(scriptsListCmd)
	scriptsCmd.AddCommand(scriptsRunCmd)

	// Add flags
	scriptsRunCmd.Flags().StringSliceVarP(&scriptEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
	scriptsRunCmd.Flags().StringVarP(&workDir, "workdir", "w", "", "Working directory")
	scriptsRunCmd.Flags().BoolVarP(&skipHookError, "skip-hook-error", "s", false, "Skip hook errors")
}

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Manage scripts",
	Long:  `List and run scripts defined in your configuration.`,
}

var scriptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := script.NewManager()

		// Load scripts from config
		configPath := script.DefaultConfigPath()
		if err := script.LoadAndRegisterScripts(manager, configPath); err != nil {
			return fmt.Errorf("failed to load scripts: %w", err)
		}

		// Get and display scripts
		scripts := manager.List()
		if len(scripts) == 0 {
			fmt.Println("No scripts available")
			return nil
		}

		fmt.Println("Available scripts:")
		for _, s := range scripts {
			desc := s.Description
			if desc == "" {
				desc = "No description available"
			}
			fmt.Printf("  - %s: %s\n", s.Name, desc)
		}

		return nil
	},
}

var scriptsRunCmd = &cobra.Command{
	Use:   "run [script]",
	Short: "Run a script",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptName := args[0]
		manager := script.NewManager()

		// Load scripts from config
		configPath := script.DefaultConfigPath()
		if err := script.LoadAndRegisterScripts(manager, configPath); err != nil {
			return fmt.Errorf("failed to load scripts: %w", err)
		}

		// Parse environment variables
		env := make(map[string]string)
		for _, e := range scriptEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid environment variable format: %s", e)
			}
			env[parts[0]] = parts[1]
		}

		// Create run options
		opts := &script.RunOptions{
			Env:              env,
			WorkDir:          workDir,
			SkipHooksOnError: skipHookError,
		}

		// Run the script
		if err := manager.Run(scriptName, opts); err != nil {
			return fmt.Errorf("failed to run script: %w", err)
		}

		return nil
	},
}

// Add shorthand commands for common scripts
func addShorthandCommand(name string) {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Run the %s script", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Forward to scripts run command
			return scriptsRunCmd.RunE(cmd, []string{name})
		},
	}

	// Add the same flags as scriptsRunCmd
	cmd.Flags().StringSliceVarP(&scriptEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVarP(&workDir, "workdir", "w", "", "Working directory")
	cmd.Flags().BoolVarP(&skipHookError, "skip-hook-error", "s", false, "Skip hook errors")

	rootCmd.AddCommand(cmd)
}

func init() {
	// Add common shorthand commands
	addShorthandCommand("setup")
	addShorthandCommand("test")
	addShorthandCommand("server")
}
