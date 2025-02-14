package config

// DockerServiceConfig represents the configuration for a Docker-based service
type DockerServiceConfig struct {
	Type        string             `json:"type"`  // Always "docker"
	Image       string             `json:"image"` // Docker image name and tag
	Port        int                `json:"port"`  // Main service port
	Environment map[string]string  `json:"environment,omitempty"`
	Volumes     map[string]string  `json:"volumes,omitempty"`
	Command     []string           `json:"command,omitempty"`    // Optional override for container command
	Entrypoint  []string           `json:"entrypoint,omitempty"` // Optional override for container entrypoint
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty"`
}

// HealthCheckConfig defines how to check if a service is healthy
type HealthCheckConfig struct {
	Command     []string `json:"command"`      // Command to run to check health
	Interval    string   `json:"interval"`     // Time between checks (e.g., "30s")
	Timeout     string   `json:"timeout"`      // Timeout for each check (e.g., "5s")
	Retries     int      `json:"retries"`      // Number of retries before considering unhealthy
	StartPeriod string   `json:"start_period"` // Initial grace period (e.g., "40s")
}

// GetDefaultHealthCheck returns a default health check configuration for a service
func GetDefaultHealthCheck(serviceType string) *HealthCheckConfig {
	switch serviceType {
	case "postgresql":
		return &HealthCheckConfig{
			Command:     []string{"pg_isready"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "40s",
		}
	case "redis":
		return &HealthCheckConfig{
			Command:     []string{"redis-cli", "ping"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "30s",
		}
	case "mysql":
		return &HealthCheckConfig{
			Command:     []string{"mysqladmin", "ping", "-h", "localhost"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "40s",
		}
	default:
		return nil
	}
}

// GetDefaultDockerConfig returns a default Docker configuration for a service type
func GetDefaultDockerConfig(serviceType string) *DockerServiceConfig {
	switch serviceType {
	case "postgresql":
		return &DockerServiceConfig{
			Type:  "docker",
			Image: "postgres:17",
			Port:  5432,
			Environment: map[string]string{
				"POSTGRES_USER":             "postgres",
				"POSTGRES_PASSWORD":         "postgres",
				"PGDATA":                    "/var/lib/postgresql/data/pgdata",
				"POSTGRES_HOST_AUTH_METHOD": "trust",
			},
			Volumes: map[string]string{
				"data": "/var/lib/postgresql/data",
			},
			HealthCheck: GetDefaultHealthCheck("postgresql"),
		}
	case "redis":
		return &DockerServiceConfig{
			Type:  "docker",
			Image: "redis:7",
			Port:  6379,
			Volumes: map[string]string{
				"data": "/data",
			},
			HealthCheck: GetDefaultHealthCheck("redis"),
		}
	case "mysql":
		return &DockerServiceConfig{
			Type:  "docker",
			Image: "mysql:8",
			Port:  3306,
			Environment: map[string]string{
				"MYSQL_ROOT_PASSWORD": "mysql",
				"MYSQL_DATABASE":      "app_development",
			},
			Volumes: map[string]string{
				"data": "/var/lib/mysql",
			},
			HealthCheck: GetDefaultHealthCheck("mysql"),
		}
	default:
		return nil
	}
}
