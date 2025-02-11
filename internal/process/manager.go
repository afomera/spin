package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/afomera/dev_spin/internal/config"
	"github.com/afomera/dev_spin/internal/logger"
)

type ProcessStatus string

const (
	StatusStopped  ProcessStatus = "stopped"
	StatusRunning  ProcessStatus = "running"
	StatusStarting ProcessStatus = "starting"
	StatusError    ProcessStatus = "error"
)

// Process represents a running process
type Process struct {
	Name         string
	Command      *exec.Cmd
	Status       ProcessStatus
	Error        error
	OutputFile   string // Path to the output file
	IsDebug      bool   // Whether this is a debug session
	OutputWriter io.Writer
	TmuxSession  string // Name of the tmux session
}

// Manager handles multiple processes
type Manager struct {
	processes map[string]*Process
	config    *config.Config
	mu        sync.RWMutex
	wg        sync.WaitGroup
	store     *Store
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton instance of the process manager
func GetManager(cfg *config.Config) *Manager {
	once.Do(func() {
		instance = &Manager{
			processes: make(map[string]*Process),
			config:    cfg,
			store:     NewStore(),
		}
	})
	return instance
}

// getSpinDir returns the spin directory path
func getSpinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".spin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

// isDebugCommand checks if a command should run in debug mode
func isDebugCommand(command string, args []string) bool {
	if command == "bundle" && len(args) > 0 && args[0] == "exec" {
		if len(args) > 1 {
			switch args[1] {
			case "rails", "irb", "pry":
				return true
			}
		}
	}
	return false
}

// findProcess tries to find a process by name in both memory and store
func (m *Manager) findProcess(name string) (*Process, error) {
	// First check in-memory processes
	m.mu.RLock()
	process, exists := m.processes[name]
	m.mu.RUnlock()
	if exists {
		fmt.Printf("Debug: Found process %s in memory\n", name)
		return process, nil
	}

	// Then check the store
	info, err := m.store.GetProcess(name)
	if err != nil {
		fmt.Printf("Debug: Process %s not found in store: %v\n", name)
		return nil, err
	}
	fmt.Printf("Debug: Found process %s in store (PID: %d)\n", name, info.Pid)

	// Try to find the process
	proc, err := os.FindProcess(info.Pid)
	if err != nil {
		fmt.Printf("Debug: Failed to find process %s with PID %d: %v\n", name, info.Pid, err)
		return nil, fmt.Errorf("failed to find process: %w", err)
	}

	// Check if process is still running
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		fmt.Printf("Debug: Process %s (PID: %d) is not running: %v\n", name, info.Pid, err)
		// Remove from store since it's not running
		m.store.RemoveProcess(name)
		return nil, fmt.Errorf("process is not running: %w", err)
	}

	fmt.Printf("Debug: Process %s (PID: %d) is running\n", name, info.Pid)

	// Get spin directory for output file
	spinDir, err := getSpinDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get spin directory: %w", err)
	}

	// Get tmux session name
	sessionName := fmt.Sprintf("spin-%s", name)

	// Check if tmux session exists and get pane PID
	listCmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := listCmd.Output()
	if err != nil {
		fmt.Printf("Debug: No tmux session for process %s\n", name)
		return nil, fmt.Errorf("process has no tmux session")
	}

	// Parse PID from output
	panePid := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(panePid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pane PID: %w", err)
	}

	// Get the process
	proc, err = os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process with PID %d: %w", pid, err)
	}

	// Create a new Process instance
	process = &Process{
		Name:        info.Name,
		Command:     &exec.Cmd{Process: proc},
		Status:      info.Status,
		OutputFile:  filepath.Join(spinDir, "output", fmt.Sprintf("%s.log", name)),
		TmuxSession: sessionName,
	}
	fmt.Printf("Debug: Found tmux session for process %s\n", name)

	// Add to manager's processes map
	m.mu.Lock()
	m.processes[name] = process
	m.mu.Unlock()

	return process, nil
}

// StartProcess starts a new process with the given name and command
func (m *Manager) StartProcess(name string, command string, args []string, env []string, workDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Printf("Debug: Starting process %s: %s %v\n", name, command, args)

	if _, exists := m.processes[name]; exists {
		return fmt.Errorf("process %s is already running", name)
	}

	// Get spin directory
	spinDir, err := getSpinDir()
	if err != nil {
		return fmt.Errorf("failed to create spin directory: %w", err)
	}

	// Create output directory
	outputDir := filepath.Join(spinDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.log", name))
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Ensure tmux is set up
	if err := setupTmux(); err != nil {
		f.Close()
		return fmt.Errorf("failed to set up tmux: %w", err)
	}

	// Get config path
	home, err := os.UserHomeDir()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(home, ".spin", "tmux.conf")

	// Create a new tmux session for the process
	sessionName := fmt.Sprintf("spin-%s", name)
	createCmd := exec.Command("tmux", "-f", configPath, "new-session", "-d", "-s", sessionName, "-c", workDir)
	createCmd.Env = env
	if err := createCmd.Run(); err != nil {
		f.Close()
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Send the command to the tmux session first
	// Send each part separately to preserve spaces
	sendCmd := exec.Command("tmux", "-f", configPath, "send-keys", "-t", sessionName, command, "Space")
	if err := sendCmd.Run(); err != nil {
		f.Close()
		return fmt.Errorf("failed to send command to tmux session: %w", err)
	}

	// Send each argument separately with spaces
	for _, arg := range args {
		sendCmd = exec.Command("tmux", "-f", configPath, "send-keys", "-t", sessionName, arg, "Space")
		if err := sendCmd.Run(); err != nil {
			f.Close()
			return fmt.Errorf("failed to send argument to tmux session: %w", err)
		}
	}

	// Send Enter key
	sendCmd = exec.Command("tmux", "-f", configPath, "send-keys", "-t", sessionName, "Enter")
	if err := sendCmd.Run(); err != nil {
		f.Close()
		return fmt.Errorf("failed to send enter to tmux session: %w", err)
	}

	// Create prefixed writer for output
	prefixedWriter := logger.CreatePrefixedWriter(name)
	outputWriter := io.MultiWriter(f, prefixedWriter)

	// Set up pipe-pane to capture output in real-time
	pipeCmd := exec.Command("tmux", "pipe-pane", "-t", sessionName, fmt.Sprintf("while IFS= read -r line; do echo \"$line\" >> '%s'; echo \"$line\"; done", outputFile))
	pipeCmd.Stdout = outputWriter
	if err := pipeCmd.Run(); err != nil {
		f.Close()
		return fmt.Errorf("failed to pipe tmux output: %w", err)
	}

	process := &Process{
		Name:         name,
		Command:      createCmd, // Store the tmux command
		Status:       StatusRunning,
		OutputFile:   outputFile,
		OutputWriter: outputWriter,
		IsDebug:      isDebugCommand(command, args),
		TmuxSession:  sessionName,
	}

	m.processes[name] = process

	// Get the PID of the process in the tmux pane
	listCmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := listCmd.Output()
	if err != nil {
		fmt.Printf("Warning: Failed to get pane PID: %v\n", err)
		return fmt.Errorf("failed to get pane PID: %w", err)
	}

	// Parse PID from output
	panePid := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(panePid)
	if err != nil {
		fmt.Printf("Warning: Failed to parse pane PID: %v\n", err)
		return fmt.Errorf("failed to parse pane PID: %w", err)
	}

	// Save process information to store
	info := ProcessInfo{
		Name:    name,
		Pid:     pid,
		Status:  StatusRunning,
		WorkDir: workDir,
	}

	fmt.Printf("Debug: Saving process %s (PID: %d) to store\n", name, info.Pid)
	if err := m.store.SaveProcess(info); err != nil {
		fmt.Printf("Warning: Failed to save process info: %v\n", err)
	}

	return nil
}

// setupTmux ensures tmux is available and configured
func setupTmux() error {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is not installed: %w", err)
	}

	// Create a minimal tmux config that changes the detach key to Ctrl+D
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".spin", "tmux.conf")
	configContent := `
# Use Ctrl+D to detach
unbind-key C-b
set-option -g prefix C-d
bind-key C-d detach-client
`
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write tmux config: %w", err)
	}

	return nil
}

// DebugProcess attaches to a process in debug mode using tmux
func (m *Manager) DebugProcess(name string) error {
	// Ensure tmux is set up
	if err := setupTmux(); err != nil {
		return fmt.Errorf("failed to set up tmux: %w", err)
	}

	// Get config path
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(home, ".spin", "tmux.conf")

	// Get the session name
	sessionName := fmt.Sprintf("spin-%s", name)

	// Check if session exists
	checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("process %s is not running in tmux", name)
	}

	fmt.Printf("Attaching to process '%s' in debug mode...\n", name)
	fmt.Println("Press Ctrl+D to detach")

	// Attach to the tmux session
	attachCmd := exec.Command("tmux", "-f", configPath, "attach-session", "-t", sessionName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr

	return attachCmd.Run()
}

// StopProcess stops a specific process
func (m *Manager) StopProcess(name string) error {
	process, err := m.findProcess(name)
	if err != nil {
		return err
	}

	// Kill the tmux session
	if process.TmuxSession != "" {
		killCmd := exec.Command("tmux", "kill-session", "-t", process.TmuxSession)
		if err := killCmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to kill tmux session: %v\n", err)
		}
	}

	// Close output writer if it's a file
	if f, ok := process.OutputWriter.(*os.File); ok {
		f.Close()
	}

	// Update process status
	process.Status = StatusStopped

	// Remove from store
	if err := m.store.RemoveProcess(name); err != nil {
		fmt.Printf("Warning: Failed to remove process info: %v\n", err)
	}

	// Remove from in-memory map
	m.mu.Lock()
	delete(m.processes, name)
	m.mu.Unlock()

	return nil
}

// StopAll stops all running processes
func (m *Manager) StopAll() {
	m.mu.RLock()
	processes := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		processes = append(processes, p)
	}
	m.mu.RUnlock()

	for _, p := range processes {
		_ = m.StopProcess(p.Name)
	}

	m.wg.Wait()
}

// HandleSignals sets up signal handling for graceful shutdown
func (m *Manager) HandleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal. Stopping all processes...")
		m.StopAll()
	}()
}

// GetProcessStatus returns the status of a specific process
func (m *Manager) GetProcessStatus(name string) (ProcessStatus, error) {
	process, err := m.findProcess(name)
	if err != nil {
		return "", err
	}
	return process.Status, nil
}

// ListProcesses returns a list of all processes
func (m *Manager) ListProcesses() []*Process {
	// Get processes from store
	storeProcesses, err := m.store.ListProcesses()
	if err != nil {
		fmt.Printf("Debug: Error listing processes from store: %v\n", err)
		return nil
	}

	fmt.Printf("Debug: Found %d processes in store\n", len(storeProcesses))

	// Convert store processes to Process objects
	processes := make([]*Process, 0, len(storeProcesses))
	for _, info := range storeProcesses {
		if process, err := m.findProcess(info.Name); err == nil {
			processes = append(processes, process)
		}
	}

	fmt.Printf("Debug: Returning %d active processes\n", len(processes))
	return processes
}

// WaitForAll waits for all processes to complete
func (m *Manager) WaitForAll() {
	m.wg.Wait()
}
