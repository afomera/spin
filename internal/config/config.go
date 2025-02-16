package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afomera/spin/internal/detector"
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
		Redis         bool `json:"redis"`
		Sidekiq       bool `json:"sidekiq,omitempty"`
		DelayedJob    bool `json:"delayed_job,omitempty"`
		GoodJob       bool `json:"good_job,omitempty"`
		Elasticsearch bool `json:"elasticsearch,omitempty"`
		Memcached     bool `json:"memcached,omitempty"`
		ActionCable   bool `json:"action_cable,omitempty"`
	} `json:"services"`
	Assets struct {
		Pipeline string `json:"pipeline,omitempty"` // sprockets, webpacker, propshaft
		Bundler  string `json:"bundler,omitempty"`  // esbuild, rollup, webpack
	} `json:"assets,omitempty"`
	Testing struct {
		Framework string `json:"framework,omitempty"` // rspec, minitest
	} `json:"testing,omitempty"`
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

// GetHTTPSCloneURL returns the HTTPS clone URL for the repository
func (r *Repository) GetHTTPSCloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", r.Organization, r.Name)
}

// GetSSHCloneURL returns the SSH clone URL for the repository
func (r *Repository) GetSSHCloneURL() string {
	return fmt.Sprintf("git@github.com:%s/%s.git", r.Organization, r.Name)
}

// GetCloneURL returns the preferred clone URL based on the user's configuration
func (r *Repository) GetCloneURL(preferSSH bool) string {
	if preferSSH {
		return r.GetSSHCloneURL()
	}
	return r.GetHTTPSCloneURL()
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
	// Use the Rails detector first
	railsConfig, err := detector.DetectRails(path)
	if err == nil {
		// Convert detector.RailsConfig to our Config structure
		cfg := &Config{
			Type:    "rails",
			Version: "1.0.0",
			Dependencies: Dependencies{
				Services: []string{},
				Tools:    []string{"ruby", "bundler"},
			},
			Processes: &ProcessConfig{
				Procfile: "Procfile.dev",
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
			Rails: &RailsConfig{
				Ruby: struct {
					Version string `json:"version"`
				}{
					Version: railsConfig.Ruby.Version,
				},
				Rails: struct {
					Version string `json:"version"`
				}{
					Version: railsConfig.RailsConfig.Version,
				},
				Database: struct {
					Type     string            `json:"type"`
					Settings map[string]string `json:"settings"`
				}{
					Type:     railsConfig.Database.Type,
					Settings: railsConfig.Database.Settings,
				},
				Services: struct {
					Redis         bool `json:"redis"`
					Sidekiq       bool `json:"sidekiq,omitempty"`
					DelayedJob    bool `json:"delayed_job,omitempty"`
					GoodJob       bool `json:"good_job,omitempty"`
					Elasticsearch bool `json:"elasticsearch,omitempty"`
					Memcached     bool `json:"memcached,omitempty"`
					ActionCable   bool `json:"action_cable,omitempty"`
				}{
					Redis:         railsConfig.Services.Redis,
					Sidekiq:       railsConfig.Services.Sidekiq,
					DelayedJob:    railsConfig.Services.DelayedJob,
					GoodJob:       railsConfig.Services.GoodJob,
					Elasticsearch: railsConfig.Services.Elasticsearch,
					Memcached:     railsConfig.Services.Memcached,
					ActionCable:   railsConfig.Services.ActionCable,
				},
				Assets: struct {
					Pipeline string `json:"pipeline,omitempty"`
					Bundler  string `json:"bundler,omitempty"`
				}{
					Pipeline: railsConfig.Assets.Pipeline,
					Bundler:  railsConfig.Assets.Bundler,
				},
				Testing: struct {
					Framework string `json:"framework,omitempty"`
				}{
					Framework: railsConfig.Testing.Framework,
				},
			},
		}

		// Build services configuration based on detected database and other dependencies
		services := make(map[string]*DockerServiceConfig)

		// Add database service if detected
		switch railsConfig.Database.Type {
		case "postgresql":
			services["postgresql"] = GetDefaultDockerConfig("postgresql")
		case "mysql":
			services["mysql"] = GetDefaultDockerConfig("mysql")
		}

		// Add detected services
		if railsConfig.Services.Redis {
			services["redis"] = GetDefaultDockerConfig("redis")
		}
		if railsConfig.Services.Elasticsearch {
			services["elasticsearch"] = GetDefaultDockerConfig("elasticsearch")
		}
		if railsConfig.Services.Memcached {
			services["memcached"] = GetDefaultDockerConfig("memcached")
		}

		// Add background job services
		if railsConfig.Services.Sidekiq {
			if _, exists := services["redis"]; !exists {
				services["redis"] = GetDefaultDockerConfig("redis")
			}
		}

		cfg.Services = services

		// Update dependencies based on detected services
		for serviceName := range services {
			cfg.Dependencies.Services = append(cfg.Dependencies.Services, serviceName)
		}

		// Update test command based on detected testing framework
		if railsConfig.Testing.Framework == "rspec" {
			cfg.Scripts["test"] = Script{
				Command:     "bundle exec rspec",
				Description: "Run RSpec tests",
				Hooks: Hooks{
					Pre: &Hook{
						Command:     "bundle exec rails db:test:prepare",
						Description: "Prepare test database",
					},
				},
			}
		} else if railsConfig.Testing.Framework == "minitest" {
			cfg.Scripts["test"] = Script{
				Command:     "bundle exec rails test",
				Description: "Run Minitest tests",
				Hooks: Hooks{
					Pre: &Hook{
						Command:     "bundle exec rails db:test:prepare",
						Description: "Prepare test database",
					},
				},
			}
		}

		return cfg, nil
	}

	// Try Node.js detection
	if nodeConfig, err := detector.DetectNode(path); err == nil {
		// Convert detector.NodeConfig to our Config structure
		cfg := &Config{
			Type:    "node",
			Version: nodeConfig.Version,
			Dependencies: Dependencies{
				Services: []string{},
				Tools:    append([]string{"node"}, nodeConfig.DevTools...),
			},
			Scripts: make(map[string]Script),
		}

		// Add services based on detected Node.js services
		services := make(map[string]*DockerServiceConfig)

		// Add database service if detected
		switch nodeConfig.Services.Database {
		case "postgresql":
			services["postgresql"] = GetDefaultDockerConfig("postgresql")
		case "mysql":
			services["mysql"] = GetDefaultDockerConfig("mysql")
		case "mongodb":
			services["mongodb"] = GetDefaultDockerConfig("mongodb")
		}

		// Add cache service if detected
		switch nodeConfig.Services.Cache {
		case "redis":
			services["redis"] = GetDefaultDockerConfig("redis")
		case "memcached":
			services["memcached"] = GetDefaultDockerConfig("memcached")
		}

		// Add search service if detected
		if nodeConfig.Services.Search == "elasticsearch" {
			services["elasticsearch"] = GetDefaultDockerConfig("elasticsearch")
		}

		cfg.Services = services

		// Update dependencies based on detected services
		for serviceName := range services {
			cfg.Dependencies.Services = append(cfg.Dependencies.Services, serviceName)
		}

		// Convert package.json scripts to our Script format
		for _, scriptName := range nodeConfig.Scripts {
			cfg.Scripts[scriptName] = Script{
				Command:     fmt.Sprintf("npm run %s", scriptName),
				Description: fmt.Sprintf("Run npm script: %s", scriptName),
			}
		}

		return cfg, nil
	}

	return nil, fmt.Errorf("unable to detect project type")
}
