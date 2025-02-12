package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ProcessInfo stores serializable process information
type ProcessInfo struct {
	Name          string        `json:"name"`
	Pid           int           `json:"pid"`
	Status        ProcessStatus `json:"status"`
	WorkDir       string        `json:"workdir"`
	CPUPercent    float64       `json:"cpu_percent"`
	MemoryUsage   uint64        `json:"memory_usage"` // in bytes
	MemoryPercent float64       `json:"memory_percent"`
	LastUpdated   time.Time     `json:"last_updated"`
}

// Store manages persistent process information
type Store struct {
	path    string
	mu      sync.RWMutex
	manager *Manager // Reference to the process manager for debug logging
}

// NewStore creates a new process store
func NewStore(manager *Manager) *Store {
	// Store process info in user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		manager.debugf("Debug: Error getting home directory: %v\n", err)
		home = "."
	}
	spinDir := filepath.Join(home, ".spin")
	if err := os.MkdirAll(spinDir, 0755); err != nil {
		manager.debugf("Debug: Error creating spin directory: %v\n", err)
	}

	storePath := filepath.Join(spinDir, "processes.json")
	manager.debugf("Debug: Process store path: %s\n", storePath)

	// Ensure the file exists with proper permissions
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		manager.debugf("Debug: Creating new process store file\n")
		if err := os.WriteFile(storePath, []byte("{}"), 0644); err != nil {
			manager.debugf("Debug: Error creating process store file: %v\n", err)
		}
	}

	return &Store{
		path:    storePath,
		manager: manager,
	}
}

// SaveProcess saves process information to the store
func (s *Store) SaveProcess(info ProcessInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.manager.debugf("Debug: Saving process %s (PID: %d) to store\n", info.Name, info.Pid)

	processes, err := s.loadProcesses()
	if err != nil {
		s.manager.debugf("Debug: Error loading processes: %v, creating new map\n", err)
		processes = make(map[string]ProcessInfo)
	}

	processes[info.Name] = info

	return s.saveProcesses(processes)
}

// RemoveProcess removes a process from the store
func (s *Store) RemoveProcess(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.manager.debugf("Debug: Removing process %s from store\n", name)

	processes, err := s.loadProcesses()
	if err != nil {
		return err
	}

	delete(processes, name)
	return s.saveProcesses(processes)
}

// GetProcess retrieves process information from the store
func (s *Store) GetProcess(name string) (ProcessInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.manager.debugf("Debug: Getting process %s from store\n", name)

	processes, err := s.loadProcesses()
	if err != nil {
		s.manager.debugf("Debug: Error loading processes: %v\n", err)
		return ProcessInfo{}, err
	}

	info, exists := processes[name]
	if !exists {
		s.manager.debugf("Debug: Process %s not found in store\n", name)
		return ProcessInfo{}, fmt.Errorf("process %s not found", name)
	}

	s.manager.debugf("Debug: Found process %s (PID: %d) in store\n", name, info.Pid)
	return info, nil
}

// ListProcesses returns all processes in the store
func (s *Store) ListProcesses() ([]ProcessInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.manager.debugf("Debug: Listing all processes from store\n")

	processes, err := s.loadProcesses()
	if err != nil {
		s.manager.debugf("Debug: Error loading processes: %v\n", err)
		return nil, err
	}

	result := make([]ProcessInfo, 0, len(processes))
	for _, info := range processes {
		// Check if process is still running
		if info.Pid > 0 {
			if proc, err := os.FindProcess(info.Pid); err == nil {
				// On Unix systems, this always succeeds, so we need to send signal 0
				// to test if the process exists
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					s.manager.debugf("Debug: Process %s (PID: %d) is still running\n", info.Name, info.Pid)
					result = append(result, info)
					continue
				}
				s.manager.debugf("Debug: Process %s (PID: %d) is not responding to signals\n", info.Name, info.Pid)
			}
			s.manager.debugf("Debug: Process %s (PID: %d) not found, removing from store\n", info.Name, info.Pid)
			// Process is not running, remove it from store
			delete(processes, info.Name)
		}
	}

	// Save cleaned up processes
	if err := s.saveProcesses(processes); err != nil {
		s.manager.debugf("Debug: Error saving cleaned up processes: %v\n", err)
	}

	s.manager.debugf("Debug: Found %d running processes\n", len(result))
	return result, nil
}

// loadProcesses reads the processes from disk
func (s *Store) loadProcesses() (map[string]ProcessInfo, error) {
	s.manager.debugf("Debug: Loading processes from %s\n", s.path)

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.manager.debugf("Debug: Store file does not exist, creating new map\n")
			return make(map[string]ProcessInfo), nil
		}
		s.manager.debugf("Debug: Error reading store file: %v\n", err)
		return nil, err
	}

	var processes map[string]ProcessInfo
	if err := json.Unmarshal(data, &processes); err != nil {
		s.manager.debugf("Debug: Error unmarshaling store data: %v\n", err)
		return nil, err
	}

	s.manager.debugf("Debug: Loaded %d processes from store\n", len(processes))
	return processes, nil
}

// saveProcesses writes the processes to disk
func (s *Store) saveProcesses(processes map[string]ProcessInfo) error {
	s.manager.debugf("Debug: Saving %d processes to store\n", len(processes))

	data, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		s.manager.debugf("Debug: Error marshaling processes: %v\n", err)
		return err
	}

	// Write to a temporary file first
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		s.manager.debugf("Debug: Error writing temporary file: %v\n", err)
		return err
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tmpPath, s.path); err != nil {
		s.manager.debugf("Debug: Error renaming temporary file: %v\n", err)
		return err
	}

	s.manager.debugf("Debug: Successfully saved processes to store\n")
	return nil
}

// Cleanup removes dead processes from the store
func (s *Store) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.manager.debugf("Debug: Cleaning up dead processes\n")

	processes, err := s.loadProcesses()
	if err != nil {
		return err
	}

	cleaned := make(map[string]ProcessInfo)
	for name, info := range processes {
		if info.Pid > 0 {
			if proc, err := os.FindProcess(info.Pid); err == nil {
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					s.manager.debugf("Debug: Process %s (PID: %d) is still running\n", name, info.Pid)
					cleaned[name] = info
				} else {
					s.manager.debugf("Debug: Process %s (PID: %d) is dead\n", name, info.Pid)
				}
			}
		}
	}

	s.manager.debugf("Debug: Cleaned up store, %d processes remaining\n", len(cleaned))
	return s.saveProcesses(cleaned)
}
