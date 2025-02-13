package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/afomera/spin/internal/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// ServiceManager manages Docker-based services
type ServiceManager struct {
	client  *client.Client
	ctx     context.Context
	dataDir string // Base directory for service data (volumes)
}

// Client returns the Docker client instance
func (m *ServiceManager) Client() *client.Client {
	return m.client
}

// NewServiceManager creates a new Docker service manager
func NewServiceManager(dataDir string) (*ServiceManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &ServiceManager{
		client:  cli,
		ctx:     context.Background(),
		dataDir: dataDir,
	}, nil
}

// StartService starts a Docker service
func (m *ServiceManager) StartService(name string, cfg *config.DockerServiceConfig) error {
	// Check for existing container
	existingID, _ := m.FindContainer(name)
	if existingID != "" {
		// Container exists, check its state
		container, err := m.client.ContainerInspect(m.ctx, existingID)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		if container.State.Running {
			// Container is running, stop it
			timeout := 10 * time.Second
			if err := m.client.ContainerStop(m.ctx, existingID, &timeout); err != nil {
				return fmt.Errorf("failed to stop container: %w", err)
			}
		}
	} else {
		// No existing container, check if port is available
		if !m.isPortAvailable(cfg.Port) {
			return fmt.Errorf("port %d is already in use by another process", cfg.Port)
		}
	}

	// Pull image if needed
	if err := m.pullImage(cfg.Image); err != nil {
		return err
	}

	// Create container if it doesn't exist
	containerID, err := m.createContainer(name, cfg)
	if err != nil {
		return err
	}

	// Start container
	if err := m.client.ContainerStart(m.ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", name, err)
	}

	// Wait for health check if configured
	if cfg.HealthCheck != nil {
		if err := m.waitForHealthy(containerID, cfg.HealthCheck); err != nil {
			return fmt.Errorf("service %s failed health check: %w", name, err)
		}
	}

	return nil
}

// isPortAvailable checks if a port is available
func (m *ServiceManager) isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// StopService stops a Docker service
func (m *ServiceManager) StopService(name string) error {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return err
	}

	timeout := 10 * time.Second
	if err := m.client.ContainerStop(m.ctx, containerID, &timeout); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", name, err)
	}

	return nil
}

// RemoveService removes a Docker service and optionally its volumes
func (m *ServiceManager) RemoveService(name string, removeVolumes bool) error {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return err
	}

	opts := types.ContainerRemoveOptions{
		RemoveVolumes: removeVolumes,
		Force:         true,
	}

	if err := m.client.ContainerRemove(m.ctx, containerID, opts); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", name, err)
	}

	return nil
}

// GetServiceLogs returns logs for a service
func (m *ServiceManager) GetServiceLogs(name string, tail int) (string, error) {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return "", err
	}

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := m.client.ContainerLogs(m.ctx, containerID, opts)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", name, err)
	}
	defer logs.Close()

	// TODO: Properly handle multiplexed stream
	// For now, just read all bytes
	buf := new(strings.Builder)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs for %s: %w", name, err)
	}

	return buf.String(), nil
}

// StreamServiceLogs streams logs for a service to stdout
func (m *ServiceManager) StreamServiceLogs(name string, tail int) error {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return err
	}

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := m.client.ContainerLogs(m.ctx, containerID, opts)
	if err != nil {
		return fmt.Errorf("failed to get logs for %s: %w", name, err)
	}
	defer logs.Close()

	// TODO: Properly handle multiplexed stream
	// For now, just copy to stdout
	_, err = io.Copy(os.Stdout, logs)
	if err != nil {
		return fmt.Errorf("failed to stream logs for %s: %w", name, err)
	}

	return nil
}

// IsRunning checks if a service is running
func (m *ServiceManager) IsRunning(name string) bool {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return false
	}

	container, err := m.client.ContainerInspect(m.ctx, containerID)
	if err != nil {
		return false
	}

	return container.State.Running
}

// GetServiceStats returns resource usage statistics for a service
func (m *ServiceManager) GetServiceStats(name string) (*types.Stats, error) {
	containerID, err := m.FindContainer(name)
	if err != nil {
		return nil, err
	}

	stats, err := m.client.ContainerStats(m.ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for %s: %w", name, err)
	}
	defer stats.Body.Close()

	var containerStats types.Stats
	// TODO: Decode stats from response body
	return &containerStats, nil
}

// CleanupVolumes removes unused Docker volumes created by Spin
func (m *ServiceManager) CleanupVolumes() error {
	// List all containers to check volume references
	containers, err := m.client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Get all volumes
	volumes, err := m.client.VolumeList(m.ctx, filters.NewArgs())
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	// Map of volume names that are in use
	inUse := make(map[string]bool)
	for _, container := range containers {
		for _, mount := range container.Mounts {
			if mount.Type == "volume" {
				inUse[mount.Name] = true
			}
		}
	}

	var removed int
	for _, volume := range volumes.Volumes {
		// Only remove volumes created by Spin (prefixed with "spin_")
		if strings.HasPrefix(volume.Name, "spin_") && !inUse[volume.Name] {
			fmt.Printf("Removing unused volume %s...\n", volume.Name)
			if err := m.client.VolumeRemove(m.ctx, volume.Name, false); err != nil {
				fmt.Printf("Warning: failed to remove volume %s: %v\n", volume.Name, err)
				continue
			}
			removed++
		}
	}

	fmt.Printf("Removed %d unused volumes\n", removed)
	return nil
}

// Helper functions

func (m *ServiceManager) pullImage(image string) error {
	fmt.Printf("Pulling image %s...\n", image)

	reader, err := m.client.ImagePull(m.ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	defer reader.Close()

	// Copy the response to stdout to show pull progress
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("error reading pull response: %w", err)
	}

	fmt.Printf("Successfully pulled image %s\n", image)
	return nil
}

func (m *ServiceManager) createContainer(name string, cfg *config.DockerServiceConfig) (string, error) {
	// Check if container already exists
	if containerID, _ := m.FindContainer(name); containerID != "" {
		// Remove the existing container but keep its volumes
		if err := m.client.ContainerRemove(m.ctx, containerID, types.ContainerRemoveOptions{
			RemoveVolumes: false, // Keep the volumes
			Force:         true,
		}); err != nil {
			return "", fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Prepare port bindings
	portBindings := nat.PortMap{}
	if cfg.Port != 0 {
		containerPort := nat.Port(fmt.Sprintf("%d/tcp", cfg.Port))
		portBindings[containerPort] = []nat.PortBinding{
			{HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%d", cfg.Port)},
		}
	}

	// Prepare volume mounts
	var mounts []mount.Mount
	for name, target := range cfg.Volumes {
		// For PostgreSQL, ensure we're using the correct data directory
		mountTarget := target
		if name == "data" && (cfg.Image == "postgres:15" || strings.HasPrefix(cfg.Image, "postgres:")) {
			// Always use /var/lib/postgresql/data as the container target path
			// This is required by the PostgreSQL image
			mountTarget = "/var/lib/postgresql/data"
		}

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: fmt.Sprintf("spin_%s_data", name),
			Target: mountTarget,
		})
	}

	// Create container
	resp, err := m.client.ContainerCreate(
		m.ctx,
		&container.Config{
			Image:       cfg.Image,
			Env:         m.mapToEnvSlice(cfg.Environment),
			Cmd:         cfg.Command,
			Entrypoint:  cfg.Entrypoint,
			Healthcheck: m.createHealthCheck(cfg.HealthCheck),
		},
		&container.HostConfig{
			PortBindings: portBindings,
			Mounts:       mounts,
		},
		nil,
		nil,
		fmt.Sprintf("spin_%s", strings.ReplaceAll(name, "postgresql", "postgres")),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", name, err)
	}

	return resp.ID, nil
}

// FindContainer returns the container ID for a given service name
func (m *ServiceManager) FindContainer(name string) (string, error) {
	containers, err := m.client.ContainerList(m.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Always use postgres instead of postgresql for container names
	searchName := strings.ReplaceAll(name, "postgresql", "postgres")
	containerName := fmt.Sprintf("/spin_%s", searchName)
	for _, container := range containers {
		for _, n := range container.Names {
			if n == containerName {
				return container.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container %s not found", name)
}

func (m *ServiceManager) waitForHealthy(containerID string, healthCheck *config.HealthCheckConfig) error {
	if healthCheck == nil {
		return nil // No health check configured
	}

	timeout, err := time.ParseDuration(healthCheck.StartPeriod)
	if err != nil {
		timeout = 60 * time.Second // Default timeout
	}

	fmt.Printf("Waiting for service to become healthy (timeout: %s)...\n", timeout)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		container, err := m.client.ContainerInspect(m.ctx, containerID)
		if err != nil {
			return err
		}

		// If container has no health check configured, consider it healthy
		if container.State.Health == nil {
			return nil
		}

		status := container.State.Health.Status
		if status == "healthy" {
			fmt.Println("Service is healthy")
			return nil
		}

		fmt.Printf("Health status: %s, waiting...\n", status)
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("service failed to become healthy within %s", timeout)
}

func (m *ServiceManager) mapToEnvSlice(env map[string]string) []string {
	var result []string
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func (m *ServiceManager) createHealthCheck(cfg *config.HealthCheckConfig) *container.HealthConfig {
	if cfg == nil {
		return nil
	}

	interval, _ := time.ParseDuration(cfg.Interval)
	timeout, _ := time.ParseDuration(cfg.Timeout)
	startPeriod, _ := time.ParseDuration(cfg.StartPeriod)

	return &container.HealthConfig{
		Test:        cfg.Command,
		Interval:    interval,
		Timeout:     timeout,
		Retries:     cfg.Retries,
		StartPeriod: startPeriod,
	}
}
