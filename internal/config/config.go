package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/afomera/spin/internal/detector"
)

// Config represents the structure of spin.config.json
type Config struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Type         string                          `json:"type"` // "rails", "react", etc.
	Repository   Repository                      `json:"repository"`
	Dependencies Dependencies                    `json:"dependencies"`
	Scripts      Scripts                         `json:"scripts"`
	Env          map[string]EnvMap               `json:"env"`
	Rails        *RailsConfig                    `json:"rails,omitempty"`
	Services     map[string]*DockerServiceConfig `json:"services,omitempty"`
}

// RailsConfig holds Rails-specific configuration
type RailsConfig struct {
	Ruby     RubyConfig              `json:"ruby"`
	Database DatabaseConfig          `json:"database"`
	Rails    RailsInfo               `json:"rails,omitempty"`
	Services detector.ServicesConfig `json:"services"`
}

// RailsInfo holds Rails version information
type RailsInfo struct {
	Version string `json:"version"`
}

// RubyConfig holds Ruby version information
type RubyConfig struct {
	Version string `json:"version"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string            `json:"type"`
	Settings map[string]string `json:"settings"`
}

// Repository defines the GitHub organization and repository name
type Repository struct {
	Organization string `json:"organization"`
	Name         string `json:"name"`
}

// GetFullName returns the full repository name (organization/name)
func (r *Repository) GetFullName() string {
	return fmt.Sprintf("%s/%s", r.Organization, r.Name)
}

// GetCloneURL returns the HTTPS clone URL for the repository
func (r *Repository) GetCloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", r.Organization, r.Name)
}

// ParseRepositoryString parses a repository string in the format "org/name"
func ParseRepositoryString(repoStr string) (*Repository, error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format. Expected 'organization/name', got '%s'", repoStr)
	}
	return &Repository{
		Organization: parts[0],
		Name:         parts[1],
	}, nil
}

// Dependencies defines required services and tools
type Dependencies struct {
	Services []string `json:"services"`
	Tools    []string `json:"tools"`
}

// Scripts defines custom commands
type Scripts struct {
	Setup string `json:"setup"`
	Start string `json:"start"`
	Test  string `json:"test"`
}

// EnvMap holds environment variables
type EnvMap map[string]string

// LoadConfig reads and parses the spin.config.json file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Initialize services map if nil
	if config.Services == nil {
		config.Services = make(map[string]*DockerServiceConfig)
	}

	// Add default Docker configurations for required services that aren't explicitly configured
	for _, serviceName := range config.Dependencies.Services {
		// Handle postgresql -> postgres mapping
		lookupName := serviceName
		if serviceName == "postgresql" {
			lookupName = "postgres"
		}

		if _, exists := config.Services[lookupName]; !exists {
			if dockerCfg := GetDefaultDockerConfig(serviceName); dockerCfg != nil {
				config.Services[lookupName] = dockerCfg
			}
		}
	}

	// If it's a Rails app, also check database type
	if config.Rails != nil && config.Rails.Database.Type != "" {
		dbType := config.Rails.Database.Type
		if _, exists := config.Services[dbType]; !exists {
			if dockerCfg := GetDefaultDockerConfig(dbType); dockerCfg != nil {
				config.Services[dbType] = dockerCfg
			}
		}
	}

	return &config, nil
}

// DetectProjectType attempts to detect the type of project in the given path
func DetectProjectType(path string) (*Config, error) {
	// Try to detect Rails first
	if railsConfig, err := detector.DetectRails(path); err == nil {
		cfg := &Config{
			Name:    "", // Will be set by caller
			Version: "1.0.0",
			Type:    "rails",
			Rails: &RailsConfig{
				Ruby: RubyConfig{Version: railsConfig.Ruby.Version},
				Database: DatabaseConfig{
					Type:     railsConfig.Database.Type,
					Settings: railsConfig.Database.Settings,
				},
				Rails: RailsInfo{
					Version: railsConfig.RailsConfig.Version,
				},
				Services: railsConfig.Services,
			},
			Dependencies: Dependencies{
				Services: []string{},
				Tools:    []string{"ruby", "bundler"},
			},
			Scripts: Scripts{
				Setup: "bundle install && rails db:setup",
				Start: "rails server",
				Test:  "rails test",
			},
			Env: map[string]EnvMap{
				"development": {},
			},
			Services: make(map[string]*DockerServiceConfig),
		}

		// Configure Docker services based on detected requirements
		if cfg.Rails.Database.Type != "" {
			if dockerCfg := GetDefaultDockerConfig(cfg.Rails.Database.Type); dockerCfg != nil {
				cfg.Services[cfg.Rails.Database.Type] = dockerCfg
				cfg.Dependencies.Services = append(cfg.Dependencies.Services, cfg.Rails.Database.Type)
			}
		}

		// Add Redis if needed
		if cfg.Rails.Services.Redis || cfg.Rails.Services.Sidekiq {
			if redisCfg := GetDefaultDockerConfig("redis"); redisCfg != nil {
				cfg.Services["redis"] = redisCfg
				cfg.Dependencies.Services = append(cfg.Dependencies.Services, "redis")
			}
		}

		// Add database to services
		if cfg.Rails.Database.Type != "" {
			cfg.Dependencies.Services = append(cfg.Dependencies.Services, cfg.Rails.Database.Type)
		}

		// Add Redis to services if needed
		if cfg.Rails.Services.Redis || cfg.Rails.Services.Sidekiq {
			cfg.Dependencies.Services = append(cfg.Dependencies.Services, "redis")
		}

		return cfg, nil
	}

	// Add more project type detection here
	// For example: React, Node.js, etc.

	return nil, fmt.Errorf("unable to detect project type")
}

// Save writes the config back to disk
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// GetEnvVars returns environment variables for the specified environment
func (c *Config) GetEnvVars(env string) map[string]string {
	if envVars, ok := c.Env[env]; ok {
		return envVars
	}
	return make(map[string]string)
}

// Exists checks if a config file exists at the given path
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
