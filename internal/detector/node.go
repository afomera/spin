package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// NodeConfig holds Node.js-specific configuration
type NodeConfig struct {
	Version     string             `json:"version"`     // Node.js version from .nvmrc, .node-version, or package.json engines
	PackageJSON PackageJSONInfo    `json:"packageJson"` // Information from package.json
	Framework   FrameworkConfig    `json:"framework"`   // Detected framework (React, Next.js, etc.)
	TypeScript  TypeScriptConfig   `json:"typescript"`  // TypeScript configuration if present
	Testing     TestConfig         `json:"testing"`     // Testing framework configuration
	Services    NodeServicesConfig `json:"services"`    // Detected services
	Scripts     []string           `json:"scripts"`     // Available npm scripts
	DevTools    []string           `json:"devTools"`    // Development tools (eslint, prettier, etc.)
}

// PackageJSONInfo represents the relevant parts of package.json
type PackageJSONInfo struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Engines         struct {
		Node string `json:"node"`
		NPM  string `json:"npm"`
		Yarn string `json:"yarn"`
	} `json:"engines"`
}

// FrameworkConfig holds information about the detected framework
type FrameworkConfig struct {
	Name    string `json:"name"`    // react, next, vue, etc.
	Version string `json:"version"` // Framework version
}

// TypeScriptConfig holds TypeScript-specific configuration
type TypeScriptConfig struct {
	Enabled bool   `json:"enabled"` // Whether TypeScript is used
	Version string `json:"version"` // TypeScript version
	Strict  bool   `json:"strict"`  // Whether strict mode is enabled
}

// TestConfig holds testing framework configuration
type TestConfig struct {
	Framework string   `json:"framework"` // jest, mocha, vitest, etc.
	Runner    string   `json:"runner"`    // test runner if different from framework
	Features  []string `json:"features"`  // Additional testing tools (cypress, playwright, etc.)
}

// NodeServicesConfig holds information about detected Node.js services
type NodeServicesConfig struct {
	Database    string `json:"database,omitempty"`    // mongodb, postgresql, etc.
	Cache       string `json:"cache,omitempty"`       // redis, memcached
	Queue       string `json:"queue,omitempty"`       // rabbitmq, kafka
	Search      string `json:"search,omitempty"`      // elasticsearch, meilisearch
	FileStorage string `json:"fileStorage,omitempty"` // s3, minio
}

// DetectNode checks if the given path contains a Node.js application
// and returns its configuration
func DetectNode(path string) (*NodeConfig, error) {
	// Check for package.json first
	packageJSONPath := filepath.Join(path, "package.json")
	if _, err := os.Stat(packageJSONPath); err != nil {
		return nil, fmt.Errorf("no package.json found: %w", err)
	}

	config := &NodeConfig{
		PackageJSON: PackageJSONInfo{
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		},
	}

	// Parse package.json
	if err := parsePackageJSON(packageJSONPath, &config.PackageJSON); err != nil {
		return nil, fmt.Errorf("error parsing package.json: %w", err)
	}

	// Detect Node.js version
	if version, err := detectNodeVersion(path, config.PackageJSON); err == nil {
		config.Version = version
	}

	// Detect framework
	config.Framework = detectFramework(config.PackageJSON)

	// Detect TypeScript
	config.TypeScript = detectTypeScript(path, config.PackageJSON)

	// Detect testing setup
	config.Testing = detectTesting(path, config.PackageJSON)

	// Detect services
	config.Services = detectNodeServices(config.PackageJSON)

	// Get available scripts
	config.Scripts = getScripts(config.PackageJSON)

	// Detect development tools
	config.DevTools = detectDevTools(config.PackageJSON)

	return config, nil
}

func parsePackageJSON(path string, info *PackageJSONInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, info)
}

func detectNodeVersion(path string, pkgInfo PackageJSONInfo) (string, error) {
	// Check .nvmrc first
	if data, err := os.ReadFile(filepath.Join(path, ".nvmrc")); err == nil {
		return string(data), nil
	}

	// Check .node-version
	if data, err := os.ReadFile(filepath.Join(path, ".node-version")); err == nil {
		return string(data), nil
	}

	// Check engines in package.json
	if pkgInfo.Engines.Node != "" {
		return pkgInfo.Engines.Node, nil
	}

	return "", fmt.Errorf("could not detect Node.js version")
}

func detectFramework(pkgInfo PackageJSONInfo) FrameworkConfig {
	framework := FrameworkConfig{}

	// Check for Next.js
	if version, ok := pkgInfo.Dependencies["next"]; ok {
		framework.Name = "next"
		framework.Version = version
		return framework
	}

	// Check for React
	if version, ok := pkgInfo.Dependencies["react"]; ok {
		framework.Name = "react"
		framework.Version = version
		return framework
	}

	// Check for Vue
	if version, ok := pkgInfo.Dependencies["vue"]; ok {
		framework.Name = "vue"
		framework.Version = version
		return framework
	}

	// Check for Angular
	if version, ok := pkgInfo.Dependencies["@angular/core"]; ok {
		framework.Name = "angular"
		framework.Version = version
		return framework
	}

	return framework
}

func detectTypeScript(path string, pkgInfo PackageJSONInfo) TypeScriptConfig {
	config := TypeScriptConfig{}

	// Check for TypeScript dependency
	if version, ok := pkgInfo.DevDependencies["typescript"]; ok {
		config.Enabled = true
		config.Version = version

		// Check tsconfig.json for strict mode
		tsconfigPath := filepath.Join(path, "tsconfig.json")
		if data, err := os.ReadFile(tsconfigPath); err == nil {
			var tsconfig struct {
				CompilerOptions struct {
					Strict bool `json:"strict"`
				} `json:"compilerOptions"`
			}
			if err := json.Unmarshal(data, &tsconfig); err == nil {
				config.Strict = tsconfig.CompilerOptions.Strict
			}
		}
	}

	return config
}

func detectTesting(path string, pkgInfo PackageJSONInfo) TestConfig {
	config := TestConfig{}

	// Check for Jest
	if _, ok := pkgInfo.DevDependencies["jest"]; ok {
		config.Framework = "jest"
	}

	// Check for Mocha
	if _, ok := pkgInfo.DevDependencies["mocha"]; ok {
		config.Framework = "mocha"
	}

	// Check for Vitest
	if _, ok := pkgInfo.DevDependencies["vitest"]; ok {
		config.Framework = "vitest"
	}

	// Check for additional testing tools
	features := []string{}

	if _, ok := pkgInfo.DevDependencies["cypress"]; ok {
		features = append(features, "cypress")
	}
	if _, ok := pkgInfo.DevDependencies["@playwright/test"]; ok {
		features = append(features, "playwright")
	}
	if _, ok := pkgInfo.DevDependencies["@testing-library/react"]; ok {
		features = append(features, "testing-library")
	}

	config.Features = features
	return config
}

func detectNodeServices(pkgInfo PackageJSONInfo) NodeServicesConfig {
	services := NodeServicesConfig{}

	// Check for database dependencies
	switch {
	case hasDependency(pkgInfo, "mongoose"), hasDependency(pkgInfo, "mongodb"):
		services.Database = "mongodb"
	case hasDependency(pkgInfo, "pg"), hasDependency(pkgInfo, "typeorm"):
		services.Database = "postgresql"
	case hasDependency(pkgInfo, "mysql"), hasDependency(pkgInfo, "mysql2"):
		services.Database = "mysql"
	}

	// Check for cache
	switch {
	case hasDependency(pkgInfo, "redis"), hasDependency(pkgInfo, "ioredis"):
		services.Cache = "redis"
	case hasDependency(pkgInfo, "memcached"), hasDependency(pkgInfo, "memjs"):
		services.Cache = "memcached"
	}

	// Check for message queues
	switch {
	case hasDependency(pkgInfo, "amqplib"):
		services.Queue = "rabbitmq"
	case hasDependency(pkgInfo, "kafkajs"):
		services.Queue = "kafka"
	}

	// Check for search
	switch {
	case hasDependency(pkgInfo, "@elastic/elasticsearch"):
		services.Search = "elasticsearch"
	case hasDependency(pkgInfo, "meilisearch"):
		services.Search = "meilisearch"
	}

	// Check for file storage
	switch {
	case hasDependency(pkgInfo, "aws-sdk"), hasDependency(pkgInfo, "@aws-sdk/client-s3"):
		services.FileStorage = "s3"
	case hasDependency(pkgInfo, "minio"):
		services.FileStorage = "minio"
	}

	return services
}

func getScripts(pkgInfo PackageJSONInfo) []string {
	scripts := []string{}
	// This would be populated from package.json scripts
	return scripts
}

func detectDevTools(pkgInfo PackageJSONInfo) []string {
	tools := []string{}

	// Check for common development tools
	devTools := map[string]string{
		"eslint":     "eslint",
		"prettier":   "prettier",
		"babel":      "@babel/core",
		"webpack":    "webpack",
		"rollup":     "rollup",
		"vite":       "vite",
		"nodemon":    "nodemon",
		"ts-node":    "ts-node",
		"husky":      "husky",
		"commitlint": "@commitlint/cli",
	}

	for tool, pkg := range devTools {
		if _, ok := pkgInfo.DevDependencies[pkg]; ok {
			tools = append(tools, tool)
		}
	}

	return tools
}

func hasDependency(pkgInfo PackageJSONInfo, name string) bool {
	_, inDeps := pkgInfo.Dependencies[name]
	_, inDevDeps := pkgInfo.DevDependencies[name]
	return inDeps || inDevDeps
}
