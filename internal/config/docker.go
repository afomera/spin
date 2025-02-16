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
	case "mongodb":
		return &HealthCheckConfig{
			Command:     []string{"mongosh", "--eval", "db.adminCommand('ping')"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "30s",
		}
	case "elasticsearch":
		return &HealthCheckConfig{
			Command:     []string{"curl", "-f", "http://localhost:9200"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "60s",
		}
	case "memcached":
		return &HealthCheckConfig{
			Command:     []string{"memcached-tool", "localhost:11211", "stats"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "30s",
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
	case "mongodb":
		return &DockerServiceConfig{
			Type:  "docker",
			Image: "mongodb/mongodb-community-server:7.0",
			Port:  27017,
			Environment: map[string]string{
				"MONGODB_INITDB_ROOT_USERNAME": "mongodb",
				"MONGODB_INITDB_ROOT_PASSWORD": "mongodb",
			},
			Volumes: map[string]string{
				"data": "/data/db",
			},
			HealthCheck: GetDefaultHealthCheck("mongodb"),
		}
	case "elasticsearch":
		return &DockerServiceConfig{
			Type:  "docker",
			Image: "elasticsearch:8.11.3",
			Port:  9200,
			Environment: map[string]string{
				"discovery.type":         "single-node",
				"xpack.security.enabled": "false",
				"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
			},
			Volumes: map[string]string{
				"data": "/usr/share/elasticsearch/data",
			},
			HealthCheck: GetDefaultHealthCheck("elasticsearch"),
		}
	case "memcached":
		return &DockerServiceConfig{
			Type:        "docker",
			Image:       "memcached:1.6",
			Port:        11211,
			HealthCheck: GetDefaultHealthCheck("memcached"),
		}
	default:
		return nil
	}
}
