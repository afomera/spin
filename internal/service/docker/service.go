package docker

import (
	"fmt"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/serviceapi"
)

// DockerService represents a Docker-based service
type DockerService struct {
	base   serviceapi.BaseService
	config *config.DockerServiceConfig
}

// NewDockerService creates a new Docker-based service
func NewDockerService(name string, cfg *config.DockerServiceConfig) *DockerService {
	return &DockerService{
		base:   serviceapi.NewBaseService(name, []string{}),
		config: cfg,
	}
}

func (s *DockerService) Name() string {
	return s.base.Name()
}

func (s *DockerService) RequiredBy() []string {
	return s.base.RequiredBy()
}

func (s *DockerService) Start() error {
	manager, err := NewServiceManager("")
	if err != nil {
		return fmt.Errorf("failed to create Docker manager: %w", err)
	}

	return manager.StartService(s.Name(), s.config)
}

func (s *DockerService) Stop() error {
	manager, err := NewServiceManager("")
	if err != nil {
		return fmt.Errorf("failed to create Docker manager: %w", err)
	}

	return manager.StopService(s.Name())
}

func (s *DockerService) IsRunning() bool {
	manager, err := NewServiceManager("")
	if err != nil {
		return false
	}

	return manager.IsRunning(s.Name())
}
