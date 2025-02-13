package service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/service/docker"
)

// Service represents a system service like Redis or PostgreSQL
type Service interface {
	Start() error
	Stop() error
	IsRunning() bool
	RequiredBy() []string
	Name() string
}

// BaseService provides common functionality for services
type BaseService struct {
	name         string
	dependencies []string
	checkCmd     string
	startCmd     string
	stopCmd      string
}

func (s *BaseService) Name() string {
	return s.name
}

func (s *BaseService) RequiredBy() []string {
	return s.dependencies
}

func (s *BaseService) IsRunning() bool {
	cmd := exec.Command("sh", "-c", s.checkCmd)
	return cmd.Run() == nil
}

func (s *BaseService) Start() error {
	if s.IsRunning() {
		return nil
	}

	cmd := exec.Command("sh", "-c", s.startCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start %s: %v (%s)", s.name, err, string(output))
	}
	return nil
}

func (s *BaseService) Stop() error {
	if !s.IsRunning() {
		return nil
	}

	cmd := exec.Command("sh", "-c", s.stopCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop %s: %v (%s)", s.name, err, string(output))
	}
	return nil
}

// ServiceManager manages multiple services
type ServiceManager struct {
	services map[string]Service
}

// NewServiceManager creates a new service manager
func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]Service),
	}
}

// RegisterService adds a service to the manager
func (m *ServiceManager) RegisterService(service Service) {
	m.services[service.Name()] = service
}

// StartService starts a specific service and its dependencies
func (m *ServiceManager) StartService(name string) error {
	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	// Start dependencies first
	for _, dep := range service.RequiredBy() {
		if err := m.StartService(dep); err != nil {
			return fmt.Errorf("failed to start dependency %s: %v", dep, err)
		}
	}

	return service.Start()
}

// StopService stops a specific service
func (m *ServiceManager) StopService(name string) error {
	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	return service.Stop()
}

// StopAll stops all services
func (m *ServiceManager) StopAll() {
	for name := range m.services {
		_ = m.StopService(name)
	}
}

// Redis service implementation
type RedisService struct {
	BaseService
}

func NewRedisService() *RedisService {
	return &RedisService{
		BaseService{
			name:         "redis",
			dependencies: []string{},
			checkCmd:     "redis-cli ping",
			startCmd:     "redis-server --daemonize yes",
			stopCmd:      "redis-cli shutdown",
		},
	}
}

// PostgreSQL service implementation
type PostgresService struct {
	BaseService
}

func NewPostgresService() *PostgresService {
	return &PostgresService{
		BaseService{
			name:         "postgresql",
			dependencies: []string{},
			checkCmd:     "pg_isready",
			startCmd:     "pg_ctl start -D /usr/local/var/postgres",
			stopCmd:      "pg_ctl stop -D /usr/local/var/postgres",
		},
	}
}

// SQLite3 service implementation
type SQLite3Service struct {
	BaseService
}

func NewSQLite3Service() *SQLite3Service {
	return &SQLite3Service{
		BaseService{
			name:         "sqlite3",
			dependencies: []string{},
			checkCmd:     "sqlite3 --version",
			startCmd:     "", // SQLite3 doesn't need to be started as a service
			stopCmd:      "", // SQLite3 doesn't need to be stopped
		},
	}
}

// GetAvailableServices returns a list of supported service types
func GetAvailableServices() []string {
	return []string{"redis", "postgres", "postgresql", "sqlite3"}
}

// DockerService represents a Docker-based service
type DockerService struct {
	BaseService
	config *config.DockerServiceConfig
}

func (s *DockerService) Start() error {
	// Use Docker manager to start the service
	manager, err := docker.NewServiceManager("")
	if err != nil {
		return fmt.Errorf("failed to create Docker manager: %w", err)
	}

	return manager.StartService(s.name, s.config)
}

func (s *DockerService) Stop() error {
	manager, err := docker.NewServiceManager("")
	if err != nil {
		return fmt.Errorf("failed to create Docker manager: %w", err)
	}

	return manager.StopService(s.name)
}

func (s *DockerService) IsRunning() bool {
	manager, err := docker.NewServiceManager("")
	if err != nil {
		return false
	}

	return manager.IsRunning(s.name)
}

// CreateService creates a new service instance by name
func CreateService(name string, cfg *config.Config) (Service, error) {
	// Check if there's a Docker configuration for this service
	if dockerCfg, ok := cfg.Services[name]; ok {
		return &DockerService{
			BaseService: BaseService{
				name:         name,
				dependencies: []string{},
			},
			config: dockerCfg,
		}, nil
	}

	// Fall back to local system services
	switch strings.ToLower(name) {
	case "redis":
		return NewRedisService(), nil
	case "postgresql", "postgres":
		return NewPostgresService(), nil
	case "sqlite3":
		return NewSQLite3Service(), nil
	default:
		return nil, fmt.Errorf("unsupported service type: %s", name)
	}
}
