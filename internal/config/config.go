package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Type         string                          `json:"type"`
	Repository   Repository                      `json:"repository"`
	Dependencies Dependencies                    `json:"dependencies"`
	Scripts      map[string]Script               `json:"scripts"`
	Env          map[string]EnvMap               `json:"env"`
	Processes    *ProcessConfig                  `json:"processes,omitempty"`
	Rails        *RailsConfig                    `json:"rails,omitempty"`
	Services     map[string]*DockerServiceConfig `json:"services,omitempty"`
}

type Script struct {
	Command     string            `json:"command"`
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Hooks       Hooks             `json:"hooks,omitempty"`
}

type Hooks struct {
	Pre  *Hook `json:"pre,omitempty"`
	Post *Hook `json:"post,omitempty"`
}

type Hook struct {
	Command     string            `json:"command"`
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

type Repository struct {
	Organization string `json:"organization"`
	Name         string `json:"name"`
}

type Dependencies struct {
	Services []string `json:"services"`
	Tools    []string `json:"tools"`
}

type EnvMap map[string]string

type ProcessConfig struct {
	Procfile string `json:"procfile"`
}

// RailsConfig represents Rails-specific configuration
type RailsConfig struct {
	Ruby struct {
		Version string `json:"version"`
	} `json:"ruby"`
	Rails struct {
		Version string `json:"version"`
	} `json:"rails"`
	Database struct {
		Type     string            `json:"type"`
		Settings map[string]string `json:"settings"`
	} `json:"database"`
	Services struct {
		Redis   bool `json:"redis"`
		Sidekiq bool `json:"sidekiq"`
	} `json:"services"`
}

// DatabaseYMLConfig represents Rails database.yml configuration
type DatabaseYMLConfig map[string]struct {
	Adapter  string `yaml:"adapter"`
	Database string `yaml:"database"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// GetEnvVars returns environment variables for the specified environment
func (c *Config) GetEnvVars(env string) map[string]string {
	if envVars, ok := c.Env[env]; ok {
		return envVars
	}
	return make(map[string]string)
}

// GetProcfilePath returns the path to the Procfile
func (c *Config) GetProcfilePath() string {
	if c.Processes != nil && c.Processes.Procfile != "" {
		return c.Processes.Procfile
	}
	return "Procfile.dev"
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load reads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfig is an alias for Load to maintain compatibility
func LoadConfig(path string) (*Config, error) {
	return Load(path)
}

// Exists checks if a configuration file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFullName returns the full repository name (organization/name)
func (r *Repository) GetFullName() string {
	return r.Organization + "/" + r.Name
}

// GetCloneURL returns the HTTPS clone URL for the repository
func (r *Repository) GetCloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", r.Organization, r.Name)
}

// ParseRepositoryString parses a repository string in the format "org/name"
func ParseRepositoryString(s string) (*Repository, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s (expected org/name)", s)
	}

	return &Repository{
		Organization: parts[0],
		Name:         parts[1],
	}, nil
}

// DetectProjectType analyzes a directory and returns a configuration based on detected project type
func DetectProjectType(path string) (*Config, error) {
	// Check for Rails project
	if isRailsProject(path) {
		return detectRailsConfig(path)
	}

	// Add detection for other project types here

	return nil, fmt.Errorf("unable to detect project type")
}

// isRailsProject checks if the directory contains a Rails project
func isRailsProject(path string) bool {
	gemfilePath := filepath.Join(path, "Gemfile")
	if _, err := os.Stat(gemfilePath); err != nil {
		return false
	}

	data, err := os.ReadFile(gemfilePath)
	if err != nil {
		return false
	}

	return strings.Contains(string(data), "rails")
}

// detectRailsConfig returns a configuration for a Rails project
func detectRailsConfig(path string) (*Config, error) {
	// Detect database configuration first to determine services
	dbType, dbSettings, _ := detectDatabaseConfig(path)

	// Build services configuration based on detected database and other dependencies
	services := make(map[string]*DockerServiceConfig)

	// Add database service if detected
	switch dbType {
	case "postgresql":
		services["postgresql"] = GetDefaultDockerConfig("postgresql")
	case "mysql":
		services["mysql"] = GetDefaultDockerConfig("mysql")
	}

	// Check for Redis dependency
	gemfilePath := filepath.Join(path, "Gemfile")
	hasRedis := false
	hasSidekiq := false
	if data, err := os.ReadFile(gemfilePath); err == nil {
		content := string(data)
		hasRedis = strings.Contains(content, "redis")
		hasSidekiq = strings.Contains(content, "sidekiq")
		if hasRedis {
			services["redis"] = GetDefaultDockerConfig("redis")
		}
	}

	cfg := &Config{
		Type: "rails",
		Dependencies: Dependencies{
			Services: []string{},
			Tools:    []string{"ruby", "bundler"},
		},
		Scripts: map[string]Script{
			"setup": {
				Command:     "bundle install",
				Description: "Install dependencies",
				Hooks: Hooks{
					Post: &Hook{
						Command:     "bundle exec rails db:setup",
						Description: "Set up database",
					},
				},
			},
			"server": {
				Command:     "bundle exec rails server",
				Description: "Start Rails server",
				Hooks: Hooks{
					Pre: &Hook{
						Command:     "bundle exec rails db:prepare",
						Description: "Prepare database",
					},
				},
			},
			"test": {
				Command:     "bundle exec rspec",
				Description: "Run tests",
				Hooks: Hooks{
					Pre: &Hook{
						Command:     "bundle exec rails db:test:prepare",
						Description: "Prepare test database",
					},
				},
			},
		},
		Env: map[string]EnvMap{
			"development": {},
		},
		Rails:    &RailsConfig{},
		Services: services,
	}

	// Update dependencies based on detected services
	for serviceName := range services {
		cfg.Dependencies.Services = append(cfg.Dependencies.Services, serviceName)
	}

	// Detect Ruby version
	if rubyVersion, err := detectRubyVersion(path); err == nil {
		cfg.Rails.Ruby.Version = rubyVersion
	}

	// Detect Rails version
	if railsVersion, err := detectRailsVersion(path); err == nil {
		cfg.Rails.Rails.Version = railsVersion
	}

	// Set database configuration
	if dbType != "" {
		cfg.Rails.Database.Type = dbType
		cfg.Rails.Database.Settings = dbSettings
	}

	// Set Redis and Sidekiq flags
	cfg.Rails.Services.Redis = hasRedis
	cfg.Rails.Services.Sidekiq = hasSidekiq

	return cfg, nil
}

// detectRubyVersion attempts to detect the Ruby version from .ruby-version or Gemfile
func detectRubyVersion(path string) (string, error) {
	// Try .ruby-version first
	rubyVersionPath := filepath.Join(path, ".ruby-version")
	if data, err := os.ReadFile(rubyVersionPath); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Try Gemfile
	gemfilePath := filepath.Join(path, "Gemfile")
	if data, err := os.ReadFile(gemfilePath); err == nil {
		content := string(data)
		if idx := strings.Index(content, "ruby '"); idx != -1 {
			version := content[idx+6:]
			if endIdx := strings.Index(version, "'"); endIdx != -1 {
				return version[:endIdx], nil
			}
		}
	}

	return "", fmt.Errorf("could not detect Ruby version")
}

// detectRailsVersion attempts to detect the Rails version from Gemfile.lock
func detectRailsVersion(path string) (string, error) {
	gemfileLockPath := filepath.Join(path, "Gemfile.lock")
	data, err := os.ReadFile(gemfileLockPath)
	if err != nil {
		return "", err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// First look in the Specs section for the exact version
	inSpecsSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "specs:" {
			inSpecsSection = true
			continue
		}
		if inSpecsSection && line == "" {
			inSpecsSection = false
			continue
		}
		if inSpecsSection && strings.HasPrefix(line, "rails (") {
			// Extract version from within parentheses
			start := strings.Index(line, "(")
			end := strings.Index(line, ")")
			if start != -1 && end != -1 && end > start {
				version := strings.TrimSpace(line[start+1 : end])
				// Remove any version constraints
				if idx := strings.Index(version, ","); idx != -1 {
					version = strings.TrimSpace(version[:idx])
				}
				if !strings.Contains(version, ">=") && !strings.Contains(version, "~>") {
					return version, nil
				}
			}
		}
	}

	// If not found in specs, try looking in the GEM section
	inGemSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "GEM" {
			inGemSection = true
			continue
		}
		if inGemSection && line == "" {
			inGemSection = false
			continue
		}
		if inGemSection && strings.HasPrefix(line, "rails (") {
			// Extract version from within parentheses
			start := strings.Index(line, "(")
			end := strings.Index(line, ")")
			if start != -1 && end != -1 && end > start {
				version := strings.TrimSpace(line[start+1 : end])
				// Remove any version constraints
				if idx := strings.Index(version, ","); idx != -1 {
					version = strings.TrimSpace(version[:idx])
				}
				if !strings.Contains(version, ">=") && !strings.Contains(version, "~>") {
					return version, nil
				}
			}
		}
	}

	// If still not found, try looking in the Gemfile
	gemfilePath := filepath.Join(path, "Gemfile")
	if data, err := os.ReadFile(gemfilePath); err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gem 'rails'") || strings.HasPrefix(line, "gem \"rails\"") {
				// Look for version in quotes after rails
				if start := strings.Index(line, "'"); start != -1 {
					version := line[start+1:]
					if end := strings.Index(version, "'"); end != -1 {
						version = version[:end]
						if !strings.Contains(version, ">=") && !strings.Contains(version, "~>") {
							return version, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not detect Rails version")
}

// detectDatabaseConfig attempts to detect database configuration from Gemfile and database.yml
func detectDatabaseConfig(path string) (string, map[string]string, error) {
	// First check Gemfile for database gems
	gemfilePath := filepath.Join(path, "Gemfile")
	if data, err := os.ReadFile(gemfilePath); err == nil {
		content := string(data)
		switch {
		case strings.Contains(content, "gem 'pg'"):
			return "postgresql", nil, nil
		case strings.Contains(content, "gem 'mysql2'"):
			return "mysql", nil, nil
		case strings.Contains(content, "gem 'sqlite3'"):
			return "sqlite3", nil, nil
		}
	}

	// Then check database.yml
	dbConfigPath := filepath.Join(path, "config", "database.yml")
	if data, err := os.ReadFile(dbConfigPath); err == nil {
		var dbConfig DatabaseYMLConfig
		if err := yaml.Unmarshal(data, &dbConfig); err == nil {
			if dev, ok := dbConfig["development"]; ok {
				settings := make(map[string]string)
				if dev.Database != "" {
					settings["database"] = dev.Database
				}
				if dev.Host != "" {
					settings["host"] = dev.Host
				}
				if dev.Port != 0 {
					settings["port"] = fmt.Sprintf("%d", dev.Port)
				}
				if dev.Username != "" {
					settings["username"] = dev.Username
				}
				return dev.Adapter, settings, nil
			}
		}
	}

	return "", nil, fmt.Errorf("could not detect database configuration")
}
