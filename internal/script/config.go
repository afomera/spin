package script

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Config represents the script configuration structure
type Config struct {
	Scripts map[string]ScriptConfig `json:"scripts"`
}

// ScriptConfig represents the configuration for a single script
type ScriptConfig struct {
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Env         map[string]string `json:"env,omitempty"`
	Hooks       HooksConfig       `json:"hooks,omitempty"`
}

// HooksConfig represents the configuration for script hooks
type HooksConfig struct {
	Pre  *HookConfig `json:"pre,omitempty"`
	Post *HookConfig `json:"post,omitempty"`
}

// HookConfig represents the configuration for a single hook
type HookConfig struct {
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Env         map[string]string `json:"env,omitempty"`
}

// LoadConfig loads script configuration from a file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, NewScriptError(
			"failed to open config file",
			err.Error(),
		).WithFix(fmt.Sprintf("Ensure the file exists at %s", path))
	}
	defer file.Close()

	return LoadConfigFromReader(file)
}

// LoadConfigFromReader loads script configuration from an io.Reader
func LoadConfigFromReader(r io.Reader) (*Config, error) {
	var config Config
	if err := json.NewDecoder(r).Decode(&config); err != nil {
		return nil, NewValidationError(
			"failed to parse config file",
			err.Error(),
		).WithFix("Ensure the config file contains valid JSON")
	}

	return &config, nil
}

// ToScripts converts the configuration into Script objects
func (c *Config) ToScripts() ([]*Script, error) {
	scripts := make([]*Script, 0, len(c.Scripts))

	for name, cfg := range c.Scripts {
		script := NewScript(name, cfg.Command, cfg.Description)

		// Add environment variables
		for k, v := range cfg.Env {
			script.SetEnv(k, v)
		}

		// Add pre hook if configured
		if cfg.Hooks.Pre != nil {
			hook := &Hook{
				Command:     cfg.Hooks.Pre.Command,
				Description: cfg.Hooks.Pre.Description,
				Env:         cfg.Hooks.Pre.Env,
			}
			if err := script.AddHook("pre", hook); err != nil {
				return nil, NewValidationError(
					fmt.Sprintf("invalid pre hook for script %s", name),
					err.Error(),
				)
			}
		}

		// Add post hook if configured
		if cfg.Hooks.Post != nil {
			hook := &Hook{
				Command:     cfg.Hooks.Post.Command,
				Description: cfg.Hooks.Post.Description,
				Env:         cfg.Hooks.Post.Env,
			}
			if err := script.AddHook("post", hook); err != nil {
				return nil, NewValidationError(
					fmt.Sprintf("invalid post hook for script %s", name),
					err.Error(),
				)
			}
		}

		scripts = append(scripts, script)
	}

	return scripts, nil
}

// LoadAndRegisterScripts loads scripts from a config file and registers them with a manager
func LoadAndRegisterScripts(manager *Manager, configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	scripts, err := config.ToScripts()
	if err != nil {
		return err
	}

	for _, script := range scripts {
		if err := manager.Register(script); err != nil {
			return NewValidationError(
				fmt.Sprintf("failed to register script %s", script.Name),
				err.Error(),
			)
		}
	}

	return nil
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	// First check for spin.config.json in the current directory
	if _, err := os.Stat("spin.config.json"); err == nil {
		return "spin.config.json"
	}

	// Then check for .spin/config.json in the current directory
	if _, err := os.Stat(filepath.Join(".spin", "config.json")); err == nil {
		return filepath.Join(".spin", "config.json")
	}

	// Finally, check for config in the user's home directory
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".spin", "config.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to spin.config.json in the current directory
	return "spin.config.json"
}

// ValidateConfig validates the configuration structure
func (c *Config) ValidateConfig() error {
	if len(c.Scripts) == 0 {
		return NewValidationError("no scripts defined in configuration")
	}

	for name, script := range c.Scripts {
		if script.Command == "" {
			return NewValidationError(
				fmt.Sprintf("command is required for script %s", name),
			)
		}

		// Validate pre hook if present
		if script.Hooks.Pre != nil {
			if script.Hooks.Pre.Command == "" {
				return NewValidationError(
					fmt.Sprintf("command is required for pre hook in script %s", name),
				)
			}
		}

		// Validate post hook if present
		if script.Hooks.Post != nil {
			if script.Hooks.Post.Command == "" {
				return NewValidationError(
					fmt.Sprintf("command is required for post hook in script %s", name),
				)
			}
		}
	}

	return nil
}
