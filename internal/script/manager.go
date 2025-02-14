package script

import (
	"fmt"
	"sync"
)

// Manager handles script registration and execution
type Manager struct {
	scripts map[string]*Script
	mu      sync.RWMutex
}

// NewManager creates a new script manager instance
func NewManager() *Manager {
	return &Manager{
		scripts: make(map[string]*Script),
	}
}

// Register adds a script to the manager
func (m *Manager) Register(script *Script) error {
	if script == nil {
		return NewValidationError("script cannot be nil")
	}

	if err := script.Validate(); err != nil {
		return NewValidationError("invalid script", err.Error())
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.scripts[script.Name]; exists {
		return NewValidationError(fmt.Sprintf("script %s already registered", script.Name))
	}

	m.scripts[script.Name] = script
	return nil
}

// Get retrieves a script by name
func (m *Manager) Get(name string) (*Script, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	script, exists := m.scripts[name]
	if !exists {
		return nil, NewScriptError(fmt.Sprintf("script %s not found", name)).
			WithFix(fmt.Sprintf("Available scripts:\n%s", m.formatAvailableScripts()))
	}

	return script, nil
}

// Run executes a script by name with the given options
func (m *Manager) Run(name string, opts *RunOptions) error {
	script, err := m.Get(name)
	if err != nil {
		return err
	}

	// Run pre hooks
	if err := m.runHooks(script, "pre", opts); err != nil {
		if opts != nil && opts.SkipHooksOnError {
			// Log warning about skipping failed hook
			fmt.Printf("Warning: Pre-hook failed but continuing due to SkipHooksOnError: %v\n", err)
		} else {
			return err
		}
	}

	// Run the main script
	if err := script.Execute(opts); err != nil {
		return NewExecutionError(fmt.Sprintf("failed to execute script %s", name), err.Error())
	}

	// Run post hooks
	if err := m.runHooks(script, "post", opts); err != nil {
		if opts != nil && opts.SkipHooksOnError {
			fmt.Printf("Warning: Post-hook failed but continuing due to SkipHooksOnError: %v\n", err)
		} else {
			return err
		}
	}

	return nil
}

// runHooks executes all hooks of a given type for a script
func (m *Manager) runHooks(script *Script, hookType string, opts *RunOptions) error {
	hook, exists := script.Hooks[hookType]
	if !exists || hook == nil {
		return nil
	}

	// Create a new script for the hook
	hookScript := NewScript(
		fmt.Sprintf("%s:%s", script.Name, hookType),
		hook.Command,
		hook.Description,
	)
	hookScript.Env = hook.Env

	// Execute the hook
	if err := hookScript.Execute(opts); err != nil {
		return NewHookError(
			fmt.Sprintf("failed to execute %s hook for script %s", hookType, script.Name),
			err.Error(),
		)
	}

	return nil
}

// List returns all registered scripts
func (m *Manager) List() []*Script {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scripts := make([]*Script, 0, len(m.scripts))
	for _, script := range m.scripts {
		scripts = append(scripts, script)
	}
	return scripts
}

// formatAvailableScripts returns a formatted string of available scripts
func (m *Manager) formatAvailableScripts() string {
	scripts := m.List()
	if len(scripts) == 0 {
		return "  No scripts available"
	}

	var result string
	for _, script := range scripts {
		desc := script.Description
		if desc == "" {
			desc = "No description available"
		}
		result += fmt.Sprintf("  - %s: %s\n", script.Name, desc)
	}
	return result
}

// Unregister removes a script from the manager
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.scripts[name]; !exists {
		return NewScriptError(fmt.Sprintf("script %s not found", name))
	}

	delete(m.scripts, name)
	return nil
}

// Clear removes all scripts from the manager
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scripts = make(map[string]*Script)
}
