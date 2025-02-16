package detector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// RailsConfig holds Rails-specific configuration
type RailsConfig struct {
	Database    DatabaseConfig `json:"database"`
	Ruby        RubyConfig     `json:"ruby"`
	RailsConfig RailsInfo      `json:"railsConfig,omitempty"`
	Services    ServicesConfig `json:"services,omitempty"`
	Assets      AssetConfig    `json:"assets,omitempty"`
	Testing     TestingConfig  `json:"testing,omitempty"`
}

// ServicesConfig holds information about detected services
type ServicesConfig struct {
	Redis         bool `json:"redis,omitempty"`
	Sidekiq       bool `json:"sidekiq,omitempty"`
	DelayedJob    bool `json:"delayed_job,omitempty"`
	GoodJob       bool `json:"good_job,omitempty"`
	Elasticsearch bool `json:"elasticsearch,omitempty"`
	Memcached     bool `json:"memcached,omitempty"`
	ActionCable   bool `json:"action_cable,omitempty"`
}

// AssetConfig holds information about asset pipeline and JavaScript bundler
type AssetConfig struct {
	Pipeline string `json:"pipeline"` // sprockets, webpacker, propshaft
	Bundler  string `json:"bundler"`  // esbuild, rollup, webpack
}

// TestingConfig holds information about testing frameworks
type TestingConfig struct {
	Framework string `json:"framework"` // rspec, minitest
}

// RailsInfo holds Rails version and configuration
type RailsInfo struct {
	Version string `json:"version"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string            `json:"type"`     // postgresql, mysql, etc.
	Settings map[string]string `json:"settings"` // Additional database settings
}

// RubyConfig holds Ruby-specific configuration
type RubyConfig struct {
	Version string `json:"version"`
}

// DetectRails checks if the given path contains a Rails application
// and returns its configuration
func DetectRails(path string) (*RailsConfig, error) {
	config := &RailsConfig{
		Database: DatabaseConfig{
			Settings: make(map[string]string),
		},
		Services: ServicesConfig{},
	}

	// Initialize with empty values
	config.Ruby.Version = ""
	config.Database.Type = ""

	// Count how many Rails indicators we find
	railsIndicators := 0

	// Check for Ruby version using various methods
	rubyVersion, err := detectRubyVersion(path)
	if err == nil {
		config.Ruby.Version = rubyVersion
		railsIndicators++
	} else {
		// Try Gemfile.lock if .ruby-version fails
		rubyVersion, err = detectRubyVersionFromGemfileLock(path)
		if err == nil {
			config.Ruby.Version = rubyVersion
			railsIndicators++
		} else {
			// Try system Ruby as a last resort
			rubyVersion, err = detectSystemRubyVersion()
			if err == nil {
				config.Ruby.Version = rubyVersion
				railsIndicators++
			}
		}
	}

	// Check for database configuration
	if dbConfig, err := detectDatabaseConfig(path); err == nil {
		config.Database = dbConfig
		railsIndicators++
	}

	// Check for Gemfile with Rails
	if hasRailsGem(path) {
		railsIndicators++
	}

	// Check for config/routes.rb
	if _, err := os.Stat(filepath.Join(path, "config", "routes.rb")); err == nil {
		railsIndicators++
	}

	// Detect Rails version from Gemfile.lock
	railsVersion, err := detectRailsVersion(path)
	if err == nil {
		config.RailsConfig.Version = railsVersion
		railsIndicators++
	}

	// Detect services from Gemfile
	if services, err := detectServices(path); err == nil {
		config.Services = services
	}

	// Detect asset pipeline and bundler configuration
	if assetConfig, err := detectAssetConfig(path); err == nil {
		config.Assets = assetConfig
	}

	// Detect testing framework
	if testingConfig, err := detectTestingConfig(path); err == nil {
		config.Testing = testingConfig
	}

	// If we found at least 2 indicators, consider it a Rails app
	if railsIndicators >= 2 {
		return config, nil
	}

	return nil, fmt.Errorf("not enough Rails indicators found")
}

// detectServices checks for Redis, Sidekiq, and other services in Gemfile
func detectServices(path string) (ServicesConfig, error) {
	services := ServicesConfig{}

	gemfilePath := filepath.Join(path, "Gemfile")
	data, err := os.ReadFile(gemfilePath)
	if err != nil {
		return services, err
	}

	content := string(data)

	// Helper function to check for gem presence
	hasGem := func(name string) bool {
		return strings.Contains(content, fmt.Sprintf("gem '%s'", name)) ||
			strings.Contains(content, fmt.Sprintf("gem \"%s\"", name))
	}

	// Check for Redis
	if hasGem("redis") {
		services.Redis = true
	}

	// Check for Sidekiq
	if hasGem("sidekiq") {
		services.Sidekiq = true
		services.Redis = true // Sidekiq requires Redis
	}

	// Check for DelayedJob
	if hasGem("delayed_job") || hasGem("delayed_job_active_record") {
		services.DelayedJob = true
	}

	// Check for GoodJob
	if hasGem("good_job") {
		services.GoodJob = true
	}

	// Check for Elasticsearch
	if hasGem("elasticsearch") || hasGem("searchkick") || hasGem("elastic-enterprise-search") {
		services.Elasticsearch = true
	}

	// Check for Memcached
	if hasGem("dalli") || hasGem("memcached") {
		services.Memcached = true
	}

	// Check for ActionCable
	cablePath := filepath.Join(path, "config", "cable.yml")
	if _, err := os.Stat(cablePath); err == nil {
		services.ActionCable = true
	}

	return services, nil
}

// detectAssetConfig determines the asset pipeline and JavaScript bundler configuration
func detectAssetConfig(path string) (AssetConfig, error) {
	config := AssetConfig{}

	// Check for asset pipeline type
	if _, err := os.Stat(filepath.Join(path, "config", "webpacker.yml")); err == nil {
		config.Pipeline = "webpacker"
	} else if _, err := os.Stat(filepath.Join(path, "config", "propshaft.rb")); err == nil {
		config.Pipeline = "propshaft"
	} else if _, err := os.Stat(filepath.Join(path, "config", "initializers", "assets.rb")); err == nil {
		config.Pipeline = "sprockets"
	}

	// Check for JavaScript bundler
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		data, err := os.ReadFile(filepath.Join(path, "package.json"))
		if err == nil {
			content := string(data)
			switch {
			case strings.Contains(content, "\"@rails/webpacker\""):
				config.Bundler = "webpack"
			case strings.Contains(content, "\"esbuild\""):
				config.Bundler = "esbuild"
			case strings.Contains(content, "\"rollup\""):
				config.Bundler = "rollup"
			}
		}
	}

	return config, nil
}

// detectTestingConfig determines the testing framework configuration
func detectTestingConfig(path string) (TestingConfig, error) {
	config := TestingConfig{}

	// Check for RSpec
	hasRspec := false
	if _, err := os.Stat(filepath.Join(path, ".rspec")); err == nil {
		hasRspec = true
	}
	if _, err := os.Stat(filepath.Join(path, "spec")); err == nil {
		hasRspec = true
	}

	if hasRspec {
		config.Framework = "rspec"
	} else {
		// Default to minitest if no RSpec found (Rails default)
		config.Framework = "minitest"
	}

	return config, nil
}

// detectSystemRubyVersion attempts to get the Ruby version from the system Ruby
func detectSystemRubyVersion() (string, error) {
	// First, find the Ruby executable
	rubyPath, err := exec.Command("which", "ruby").Output()
	if err != nil {
		return "", fmt.Errorf("ruby not found in PATH: %w", err)
	}

	// Run ruby --version and capture output
	cmd := exec.Command(strings.TrimSpace(string(rubyPath)), "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting ruby version: %w", err)
	}

	// Parse version from output (format: "ruby 3.2.2p53 (2023-03-30 revision 957bb7cb81) [x86_64-darwin22]")
	versionPattern := regexp.MustCompile(`ruby (\d+\.\d+\.\d+)`)
	matches := versionPattern.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("could not parse ruby version from output: %s", output)
}

// detectRailsVersion attempts to find Rails version in Gemfile.lock
func detectRailsVersion(path string) (string, error) {
	gemfileLockPath := filepath.Join(path, "Gemfile.lock")
	file, err := os.Open(gemfileLockPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Look for exact "rails" gem with word boundaries
	railsPattern := regexp.MustCompile(`(?m)^\s*rails\s+\((\d+\.\d+\.\d+(?:\.\d+)?)\)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := railsPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Rails version not found in Gemfile.lock")
}

// detectRubyVersionFromGemfileLock attempts to find Ruby version in Gemfile.lock
func detectRubyVersionFromGemfileLock(path string) (string, error) {
	gemfileLockPath := filepath.Join(path, "Gemfile.lock")
	file, err := os.Open(gemfileLockPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	rubyPattern := regexp.MustCompile(`RUBY VERSION\s*ruby (\d+\.\d+\.\d+(?:p\d+)?)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := rubyPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("Ruby version not found in Gemfile.lock")
}

// hasRailsGem checks if the Gemfile contains Rails
func hasRailsGem(path string) bool {
	gemfilePath := filepath.Join(path, "Gemfile")
	data, err := os.ReadFile(gemfilePath)
	if err != nil {
		return false
	}

	content := string(data)
	// Look for exact "rails" gem with word boundaries
	railsPattern := regexp.MustCompile(`(?m)^\s*gem\s+['"]rails['"]`)
	return railsPattern.MatchString(content)
}

// detectRubyVersion reads the Ruby version from .ruby-version file
func detectRubyVersion(path string) (string, error) {
	rubyVersionPath := filepath.Join(path, ".ruby-version")
	data, err := os.ReadFile(rubyVersionPath)
	if err != nil {
		// Try with ruby- prefix
		rubyVersionPath = filepath.Join(path, "ruby-version")
		data, err = os.ReadFile(rubyVersionPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Try reading from Gemfile if .ruby-version doesn't exist
				return detectRubyVersionFromGemfile(path)
			}
			return "", err
		}
	}

	version := strings.TrimSpace(string(data))
	// Remove "ruby-" prefix if present
	version = strings.TrimPrefix(version, "ruby-")
	return version, nil
}

// detectRubyVersionFromGemfile attempts to find Ruby version in Gemfile
func detectRubyVersionFromGemfile(path string) (string, error) {
	gemfilePath := filepath.Join(path, "Gemfile")
	data, err := os.ReadFile(gemfilePath)
	if err != nil {
		return "", err
	}

	// Look for ruby '2.7.0' or similar in Gemfile
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ruby") {
			// Extract version from quotes
			parts := strings.Split(line, "'")
			if len(parts) >= 2 {
				return parts[1], nil
			}
			parts = strings.Split(line, "\"")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("Ruby version not found in Gemfile")
}

// DatabaseYAML represents the structure of database.yml
type DatabaseYAML struct {
	Development struct {
		Adapter  string `yaml:"adapter"`
		Database string `yaml:"database"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"development"`
}

// detectDatabaseConfig reads and parses config/database.yml
func detectDatabaseConfig(path string) (DatabaseConfig, error) {
	dbConfig := DatabaseConfig{
		Settings: make(map[string]string),
	}

	dbYamlPath := filepath.Join(path, "config", "database.yml")
	data, err := os.ReadFile(dbYamlPath)
	if err != nil {
		return dbConfig, fmt.Errorf("error reading database.yml: %w", err)
	}

	var dbYAML DatabaseYAML
	if err := yaml.Unmarshal(data, &dbYAML); err != nil {
		return dbConfig, fmt.Errorf("error parsing database.yml: %w", err)
	}

	// Set database type based on adapter
	dbConfig.Type = dbYAML.Development.Adapter

	// Copy relevant settings
	if dbYAML.Development.Host != "" {
		dbConfig.Settings["host"] = dbYAML.Development.Host
	}
	if dbYAML.Development.Port != "" {
		dbConfig.Settings["port"] = dbYAML.Development.Port
	}
	if dbYAML.Development.Database != "" {
		dbConfig.Settings["database"] = dbYAML.Development.Database
	}
	if dbYAML.Development.Username != "" {
		dbConfig.Settings["username"] = dbYAML.Development.Username
	}
	// Note: We might want to handle password differently for security

	return dbConfig, nil
}
