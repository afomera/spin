package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/logger"
	"github.com/afomera/spin/internal/service/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// loadConfig loads the spin.config.json file from the current directory
func loadConfig() (*config.Config, error) {
	configPath := "spin.config.json"
	if !config.Exists(configPath) {
		return nil, fmt.Errorf("no spin.config.json found in current directory")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	// Initialize Services map if it doesn't exist
	if cfg.Services == nil {
		cfg.Services = make(map[string]*config.DockerServiceConfig)
	}

	return cfg, nil
}

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage services for your application",
	Long:  `Manage Docker-based services like databases, caches, and other dependencies.`,
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%sNAME\tTYPE\tSTATUS\tHEALTH\tPORT%s\n",
			logger.Cyan,
			logger.Reset,
		)

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		for name, service := range cfg.Services {
			status := "stopped"
			health := "-"

			if containerID, err := manager.FindContainer(name); err == nil {
				if container, err := manager.Client().ContainerInspect(context.Background(), containerID); err == nil {
					if container.State.Running {
						status = "running"
						if container.State.Health != nil {
							health = container.State.Health.Status
						} else {
							health = "healthy" // Assume healthy if no health check configured
						}
					}
				}
			}

			// Colorize status
			coloredStatus := status
			if status == "running" {
				coloredStatus = fmt.Sprintf("%s%s%s", logger.Green, status, logger.Reset)
			} else {
				coloredStatus = fmt.Sprintf("%s%s%s", logger.Red, status, logger.Reset)
			}

			// Colorize health
			coloredHealth := health
			switch health {
			case "healthy":
				coloredHealth = fmt.Sprintf("%s%s%s", logger.Green, health, logger.Reset)
			case "unhealthy":
				coloredHealth = fmt.Sprintf("%s%s%s", logger.Red, health, logger.Reset)
			case "-":
				coloredHealth = fmt.Sprintf("%s%s%s", logger.Yellow, health, logger.Reset)
			default:
				coloredHealth = fmt.Sprintf("%s%s%s", logger.Yellow, health, logger.Reset)
			}

			// Colorize name
			coloredName := fmt.Sprintf("%s%s%s", logger.Cyan, name, logger.Reset)

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
				coloredName,
				service.Type,
				coloredStatus,
				coloredHealth,
				service.Port,
			)
		}
		w.Flush()
	},
}

var servicesStartCmd = &cobra.Command{
	Use:   "start [service-name]",
	Short: "Start a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sStarting %s%s%s service...%s\n", logger.Blue, logger.Cyan, serviceName, logger.Blue, logger.Reset)
		if err := manager.StartService(serviceName, service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError starting service: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
		fmt.Printf("%sService %s%s%s started successfully%s\n", logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesStopCmd = &cobra.Command{
	Use:   "stop [service-name]",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		fmt.Printf("%sStopping %s%s%s service...%s\n", logger.Blue, logger.Cyan, serviceName, logger.Blue, logger.Reset)
		if err := manager.StopService(serviceName); err != nil {
			fmt.Fprintf(os.Stderr, "%sError stopping service: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
		fmt.Printf("%sService %s%s%s stopped successfully%s\n", logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesLogsCmd = &cobra.Command{
	Use:   "logs [service-name]",
	Short: "View service logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")

		if follow {
			// Stream logs continuously
			if err := manager.StreamServiceLogs(serviceName, tail); err != nil {
				fmt.Fprintf(os.Stderr, "%sError streaming logs: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
		} else {
			// Get logs once
			logs, err := manager.GetServiceLogs(serviceName, tail)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sError getting logs: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
			fmt.Print(logs)
		}
	},
}

type serviceConfigModel struct {
	serviceType string
	config      *config.DockerServiceConfig
	step        int
	err         error
	choices     []string
	cursor      int
	input       string
	envKey      string
	envValue    string
	volumePath  string
}

func (m *serviceConfigModel) Init() tea.Cmd {
	return nil
}

func (m *serviceConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			switch m.step {
			case 0: // Service type selection
				m.serviceType = m.choices[m.cursor]
				m.config = config.GetDefaultDockerConfig(m.serviceType)
				m.step++
				m.cursor = 0
				m.choices = []string{"latest", "previous", "specific"}
			case 1: // Version selection
				version := ""
				switch m.cursor {
				case 0:
					version = "latest"
				case 1:
					switch m.serviceType {
					case "postgresql":
						version = "14"
					case "redis":
						version = "6"
					case "mysql":
						version = "5.7"
					}
				case 2:
					// TODO: Add specific version input
					version = "latest"
				}
				m.config.Image = fmt.Sprintf("%s:%s", m.serviceType, version)
				m.step++
				m.input = fmt.Sprintf("%d", m.config.Port) // Default port
			case 2: // Port configuration
				if port, err := strconv.Atoi(m.input); err == nil {
					m.config.Port = port
					m.step++
					m.choices = []string{"yes", "no"}
					m.cursor = 0
				}
			case 3: // Configure environment variables?
				if m.choices[m.cursor] == "yes" {
					m.step = 4   // Go to env key input
					m.input = "" // Reset input for env key
				} else {
					m.step = 6 // Skip to volume config
					m.choices = []string{"yes", "no"}
					m.cursor = 0
				}
			case 4: // Environment variable key
				if m.input != "" {
					m.envKey = m.input
					m.input = "" // Reset input for value
					m.step++
				}
			case 5: // Environment variable value
				if m.input != "" {
					if m.config.Environment == nil {
						m.config.Environment = make(map[string]string)
					}
					m.config.Environment[m.envKey] = m.input
					m.step = 3 // Back to env vars question
					m.input = ""
					m.choices = []string{"yes", "no"}
					m.cursor = 0
				}
			case 6: // Configure volume?
				if m.choices[m.cursor] == "yes" {
					m.step++
					m.input = "" // Reset input for volume path
				} else {
					return m, tea.Quit
				}
			case 7: // Volume path
				if m.input != "" {
					if m.config.Volumes == nil {
						m.config.Volumes = make(map[string]string)
					}
					m.config.Volumes["data"] = m.input
					return m, tea.Quit
				}
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if m.step == 2 || m.step == 4 || m.step == 5 || m.step == 7 {
				m.input += msg.String()
			}
		}
	}
	return m, nil
}

func (m *serviceConfigModel) View() string {
	s := strings.Builder{}

	switch m.step {
	case 0:
		s.WriteString("Select service type:\n\n")
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
		}
	case 1:
		s.WriteString(fmt.Sprintf("Select %s version:\n\n", m.serviceType))
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
		}
	case 2:
		s.WriteString(fmt.Sprintf("Enter port number for %s: %s\n", m.serviceType, m.input))
	case 3:
		s.WriteString("Configure environment variables?\n\n")
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
		}
	case 4:
		s.WriteString("Enter environment variable name: ")
		s.WriteString(m.input)
	case 5:
		s.WriteString(fmt.Sprintf("Enter value for %s: ", m.envKey))
		s.WriteString(m.input)
	case 6:
		s.WriteString("Configure data volume?\n\n")
		for i, choice := range m.choices {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
		}
	case 7:
		s.WriteString("Enter volume path: ")
		s.WriteString(m.input)
	}

	s.WriteString("\n\n(Press q to quit)\n")

	return s.String()
}

var servicesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new service",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		model := &serviceConfigModel{
			step:    0,
			choices: []string{"postgresql", "redis", "mysql"},
		}

		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}

		m := finalModel.(*serviceConfigModel)
		if m.config == nil {
			fmt.Println("Service configuration cancelled")
			return
		}

		// Check if service already exists
		if _, exists := cfg.Services[m.serviceType]; exists {
			fmt.Fprintf(os.Stderr, "Service %s already exists\n", m.serviceType)
			os.Exit(1)
		}

		// Add the service to config
		if cfg.Services == nil {
			cfg.Services = make(map[string]*config.DockerServiceConfig)
		}
		cfg.Services[m.serviceType] = m.config

		// Save the updated config
		if err := cfg.Save("spin.config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Service %s added successfully\n", m.serviceType)
	},
}

var servicesRemoveCmd = &cobra.Command{
	Use:   "remove [service-name]",
	Short: "Remove a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		serviceName := args[0]
		if _, exists := cfg.Services[serviceName]; !exists {
			fmt.Fprintf(os.Stderr, "Service %s not found\n", serviceName)
			os.Exit(1)
		}

		// Remove service from config
		delete(cfg.Services, serviceName)

		// Save the updated config
		if err := cfg.Save("spin.config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		// Remove the service container and volumes if requested
		removeVolumes, _ := cmd.Flags().GetBool("remove-volumes")
		if removeVolumes {
			manager, err := docker.NewServiceManager("./data")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating service manager: %v\n", err)
				os.Exit(1)
			}

			if err := manager.RemoveService(serviceName, true); err != nil {
				fmt.Fprintf(os.Stderr, "Error removing service container: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Service %s removed successfully\n", serviceName)
	},
}

var servicesCleanupCmd = &cobra.Command{
	Use:   "cleanup [resource-type]",
	Short: "Clean up unused resources",
	Long: `Clean up unused resources like volumes.
Example: spin services cleanup volumes`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		if resourceType != "volumes" {
			fmt.Fprintf(os.Stderr, "%sUnsupported resource type: %s\nCurrently only 'volumes' is supported%s\n",
				logger.Red, resourceType, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sCleaning up unused volumes...%s\n", logger.Blue, logger.Reset)
		if err := manager.CleanupVolumes(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError cleaning up volumes: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
		fmt.Printf("%sVolumes cleaned up successfully%s\n", logger.Green, logger.Reset)
	},
}

var servicesRestartCmd = &cobra.Command{
	Use:   "restart [service-name]",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sRestarting %s%s%s service...%s\n", logger.Blue, logger.Cyan, serviceName, logger.Blue, logger.Reset)

		// Stop the service
		if err := manager.StopService(serviceName); err != nil {
			fmt.Fprintf(os.Stderr, "%sError stopping service: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Start the service
		if err := manager.StartService(serviceName, service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError starting service: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sService %s%s%s restarted successfully%s\n", logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesInfoCmd = &cobra.Command{
	Use:   "info [service-name]",
	Short: "Display detailed information about a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		containerID, err := manager.FindContainer(serviceName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError: Service %s%s%s not found or not running\nSuggestion: Run 'spin services start %s' to start the service%s\n",
				logger.Red, logger.Cyan, serviceName, logger.Red, serviceName, logger.Reset)
			os.Exit(1)
		}

		container, err := manager.Client().ContainerInspect(context.Background(), containerID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError inspecting service: %v\nSuggestion: Check if Docker daemon is running%s\n",
				logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		status := "stopped"
		health := "-"
		uptime := "-"

		if container.State.Running {
			status = "running"
			if container.State.Health != nil {
				health = container.State.Health.Status
			} else {
				health = "healthy" // Assume healthy if no health check configured
			}
			startTime, err := time.Parse(time.RFC3339Nano, container.State.StartedAt)
			if err == nil {
				uptime = time.Since(startTime).Round(time.Second).String()
			}
		}

		// Colorize status
		coloredStatus := status
		if status == "running" {
			coloredStatus = fmt.Sprintf("%s%s%s", logger.Green, status, logger.Reset)
		} else {
			coloredStatus = fmt.Sprintf("%s%s%s", logger.Red, status, logger.Reset)
		}

		// Colorize health
		coloredHealth := health
		switch health {
		case "healthy":
			coloredHealth = fmt.Sprintf("%s%s%s", logger.Green, health, logger.Reset)
		case "unhealthy":
			coloredHealth = fmt.Sprintf("%s%s%s", logger.Red, health, logger.Reset)
		case "-":
			coloredHealth = fmt.Sprintf("%s%s%s", logger.Yellow, health, logger.Reset)
		default:
			coloredHealth = fmt.Sprintf("%s%s%s", logger.Yellow, health, logger.Reset)
		}

		// Display service information
		fmt.Printf("%sService:%s %s%s%s\n", logger.Cyan, logger.Reset, logger.Blue, serviceName, logger.Reset)
		fmt.Printf("%sType:%s %s\n", logger.Cyan, logger.Reset, service.Type)
		fmt.Printf("%sImage:%s %s\n", logger.Cyan, logger.Reset, service.Image)
		fmt.Printf("%sStatus:%s %s\n", logger.Cyan, logger.Reset, coloredStatus)
		fmt.Printf("%sHealth:%s %s\n", logger.Cyan, logger.Reset, coloredHealth)
		fmt.Printf("%sUptime:%s %s\n", logger.Cyan, logger.Reset, uptime)
		fmt.Printf("%sPort:%s %d -> %d\n", logger.Cyan, logger.Reset, service.Port, service.Port)

		if len(service.Volumes) > 0 {
			fmt.Printf("\n%sVolumes:%s\n", logger.Cyan, logger.Reset)
			for name, path := range service.Volumes {
				fmt.Printf("  - %s%s%s: %s\n", logger.Blue, name, logger.Reset, path)
			}
		}

		if len(service.Environment) > 0 {
			fmt.Printf("\n%sEnvironment:%s\n", logger.Cyan, logger.Reset)
			for key, value := range service.Environment {
				// Mask sensitive values
				if strings.Contains(strings.ToLower(key), "password") ||
					strings.Contains(strings.ToLower(key), "secret") ||
					strings.Contains(strings.ToLower(key), "token") {
					value = "****"
				}
				fmt.Printf("  - %s%s%s=%s\n", logger.Blue, key, logger.Reset, value)
			}
		}

		if service.HealthCheck != nil {
			fmt.Printf("\n%sHealth Check:%s\n", logger.Cyan, logger.Reset)
			fmt.Printf("  %sCommand:%s %v\n", logger.Blue, logger.Reset, service.HealthCheck.Command)
			fmt.Printf("  %sInterval:%s %s\n", logger.Blue, logger.Reset, service.HealthCheck.Interval)
			fmt.Printf("  %sTimeout:%s %s\n", logger.Blue, logger.Reset, service.HealthCheck.Timeout)
			fmt.Printf("  %sRetries:%s %d\n", logger.Blue, logger.Reset, service.HealthCheck.Retries)
			fmt.Printf("  %sStart Period:%s %s\n", logger.Blue, logger.Reset, service.HealthCheck.StartPeriod)
		}
	},
}

var servicesEditCmd = &cobra.Command{
	Use:   "edit [service-name]",
	Short: "Edit service configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		// Create a temporary file with the service configuration
		tmpfile, err := os.CreateTemp("", "spin-*.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating temp file: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
		defer os.Remove(tmpfile.Name())

		// Write service config to temp file
		encoder := json.NewEncoder(tmpfile)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError writing config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
		tmpfile.Close()

		// Open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim" // Default to vim
		}

		fmt.Printf("%sOpening configuration in %s...%s\n", logger.Blue, editor, logger.Reset)

		cmd2 := exec.Command(editor, tmpfile.Name())
		cmd2.Stdin = os.Stdin
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr

		if err := cmd2.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError running editor: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Read updated config
		data, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError reading updated config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		var updatedService config.DockerServiceConfig
		if err := json.Unmarshal(data, &updatedService); err != nil {
			fmt.Fprintf(os.Stderr, "%sError parsing updated config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Update service config
		cfg.Services[serviceName] = &updatedService

		// Save the updated config
		if err := cfg.Save("spin.config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "%sError saving config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sService %s%s%s configuration updated successfully%s\n",
			logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesExportCmd = &cobra.Command{
	Use:   "export [service-name]",
	Short: "Export service configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sExporting configuration for %s%s%s...%s\n", logger.Blue, logger.Cyan, serviceName, logger.Blue, logger.Reset)
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError exporting config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}
	},
}

var servicesImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import service configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError reading file: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		var service config.DockerServiceConfig
		if err := json.Unmarshal(data, &service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError parsing config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Get service name from command line or file name
		serviceName := cmd.Flag("name").Value.String()
		if serviceName == "" {
			// Use filename without extension as service name
			base := filepath.Base(args[0])
			serviceName = strings.TrimSuffix(base, filepath.Ext(base))
		}

		fmt.Printf("%sImporting service configuration as %s%s%s...%s\n",
			logger.Blue, logger.Cyan, serviceName, logger.Blue, logger.Reset)

		// Add the service to config
		if cfg.Services == nil {
			cfg.Services = make(map[string]*config.DockerServiceConfig)
		}
		cfg.Services[serviceName] = &service

		// Save the updated config
		if err := cfg.Save("spin.config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "%sError saving config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sService %s%s%s imported successfully%s\n",
			logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesUpdateCmd = &cobra.Command{
	Use:   "update [service-name]",
	Short: "Update service image",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		serviceName := args[0]
		service, ok := cfg.Services[serviceName]
		if !ok {
			fmt.Fprintf(os.Stderr, "%sService %s%s%s not found%s\n", logger.Red, logger.Cyan, serviceName, logger.Red, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		// Stop the service if it's running
		if manager.IsRunning(serviceName) {
			if err := manager.StopService(serviceName); err != nil {
				fmt.Fprintf(os.Stderr, "%sError stopping service: %v%s\n", logger.Red, err, logger.Reset)
				os.Exit(1)
			}
		}

		// Check if specific version is requested
		version, _ := cmd.Flags().GetString("version")
		if version != "" {
			// Update image tag to specified version
			imageParts := strings.Split(service.Image, ":")
			service.Image = fmt.Sprintf("%s:%s", imageParts[0], version)
		}

		fmt.Printf("%sUpdating %s%s%s to image %s%s%s...%s\n",
			logger.Blue, logger.Cyan, serviceName, logger.Blue,
			logger.Cyan, service.Image, logger.Blue, logger.Reset)
		if err := manager.StartService(serviceName, service); err != nil {
			fmt.Fprintf(os.Stderr, "%sError updating service: %v\nSuggestion: Check if the specified version exists%s\n",
				logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		fmt.Printf("%sService %s%s%s updated successfully%s\n",
			logger.Green, logger.Cyan, serviceName, logger.Green, logger.Reset)
	},
}

var servicesStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View resource usage for services",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError loading config: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		manager, err := docker.NewServiceManager("./data")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError creating service manager: %v%s\n", logger.Red, err, logger.Reset)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%sNAME\tCPU\tMEMORY\tDISK%s\n",
			logger.Cyan,
			logger.Reset,
		)

		for name := range cfg.Services {
			containerID, err := manager.FindContainer(name)
			if err != nil {
				continue // Skip non-running services
			}

			stats, err := manager.Client().ContainerStats(context.Background(), containerID, false)
			if err != nil {
				fmt.Fprintf(w, "%s%s%s\t%sError%s\t%sError%s\t%sError%s\n",
					logger.Cyan, name, logger.Reset,
					logger.Red, logger.Reset,
					logger.Red, logger.Reset,
					logger.Red, logger.Reset)
				continue
			}
			defer stats.Body.Close()

			var statsData types.Stats
			if err := json.NewDecoder(stats.Body).Decode(&statsData); err != nil {
				fmt.Fprintf(w, "%s%s%s\t%sError%s\t%sError%s\t%sError%s\n",
					logger.Cyan, name, logger.Reset,
					logger.Red, logger.Reset,
					logger.Red, logger.Reset,
					logger.Red, logger.Reset)
				continue
			}

			// Calculate CPU percentage
			cpuDelta := float64(statsData.CPUStats.CPUUsage.TotalUsage - statsData.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := float64(statsData.CPUStats.SystemUsage - statsData.PreCPUStats.SystemUsage)
			cpuPercent := 0.0
			if systemDelta > 0 && cpuDelta > 0 {
				cpuPercent = (cpuDelta / systemDelta) * float64(len(statsData.CPUStats.CPUUsage.PercpuUsage)) * 100.0
			}

			// Calculate memory usage
			memoryUsage := float64(statsData.MemoryStats.Usage) / 1024 / 1024 // Convert to MB

			// Color CPU usage based on percentage
			cpuColor := logger.Green
			if cpuPercent >= 80 {
				cpuColor = logger.Red
			} else if cpuPercent >= 50 {
				cpuColor = logger.Yellow
			}

			// Color memory usage based on amount
			memColor := logger.Green
			if memoryUsage >= 1024 { // >= 1GB
				memColor = logger.Red
			} else if memoryUsage >= 512 { // >= 512MB
				memColor = logger.Yellow
			}

			fmt.Fprintf(w, "%s%s%s\t%s%.1f%%%s\t%s%.0fMB%s\t-\n",
				logger.Cyan, name, logger.Reset,
				cpuColor, cpuPercent, logger.Reset,
				memColor, memoryUsage, logger.Reset)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(servicesCmd)
	servicesCmd.AddCommand(servicesListCmd)
	servicesCmd.AddCommand(servicesStartCmd)
	servicesCmd.AddCommand(servicesStopCmd)
	servicesCmd.AddCommand(servicesRestartCmd)
	servicesCmd.AddCommand(servicesLogsCmd)
	servicesCmd.AddCommand(servicesAddCmd)
	servicesCmd.AddCommand(servicesRemoveCmd)
	servicesCmd.AddCommand(servicesCleanupCmd)
	servicesCmd.AddCommand(servicesInfoCmd)
	servicesCmd.AddCommand(servicesEditCmd)
	servicesCmd.AddCommand(servicesExportCmd)
	servicesCmd.AddCommand(servicesImportCmd)
	servicesCmd.AddCommand(servicesUpdateCmd)
	servicesCmd.AddCommand(servicesStatsCmd)

	// Add flags
	servicesLogsCmd.Flags().IntP("tail", "n", 100, "Number of lines to show from the end of the logs")
	servicesLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	servicesRemoveCmd.Flags().Bool("remove-volumes", false, "Remove associated volumes")
	servicesImportCmd.Flags().String("name", "", "Service name (defaults to filename without extension)")
	servicesUpdateCmd.Flags().String("version", "", "Specific version to update to")
}
