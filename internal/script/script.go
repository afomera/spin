package script

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Script represents a runnable script with optional hooks and environment variables
type Script struct {
	Name        string            // Name of the script
	Command     string            // Command to execute
	Description string            // Description of what the script does
	Env         map[string]string // Environment variables for the script
	Hooks       map[string]*Hook  // Pre and post execution hooks
}

// Hooks represents pre and post execution hooks
type Hooks struct {
	Pre  *Hook `json:"pre,omitempty"`
	Post *Hook `json:"post,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle both string and object formats
func (s *Script) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as a string (old format)
	var command string
	if err := json.Unmarshal(data, &command); err == nil {
		s.Command = command
		s.Description = ""
		s.Env = make(map[string]string)
		s.Hooks = make(map[string]*Hook)
		return nil
	}

	// If that fails, try to unmarshal as an object (new format)
	type ScriptAlias Script // Use alias to avoid recursive UnmarshalJSON calls
	var alias struct {
		Name        string            `json:"name,omitempty"`
		Command     string            `json:"command"`
		Description string            `json:"description,omitempty"`
		Env         map[string]string `json:"env,omitempty"`
		Hooks       Hooks             `json:"hooks,omitempty"`
	}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	s.Name = alias.Name
	s.Command = alias.Command
	s.Description = alias.Description
	s.Env = alias.Env
	if s.Env == nil {
		s.Env = make(map[string]string)
	}

	// Convert Hooks struct to map
	s.Hooks = make(map[string]*Hook)
	if alias.Hooks.Pre != nil {
		s.Hooks["pre"] = alias.Hooks.Pre
	}
	if alias.Hooks.Post != nil {
		s.Hooks["post"] = alias.Hooks.Post
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to always use the new object format
func (s *Script) MarshalJSON() ([]byte, error) {
	// Convert map hooks back to struct format
	hooks := Hooks{}
	if pre, ok := s.Hooks["pre"]; ok {
		hooks.Pre = pre
	}
	if post, ok := s.Hooks["post"]; ok {
		hooks.Post = post
	}

	// Use struct for marshaling
	obj := struct {
		Name        string            `json:"name,omitempty"`
		Command     string            `json:"command"`
		Description string            `json:"description,omitempty"`
		Env         map[string]string `json:"env,omitempty"`
		Hooks       *Hooks            `json:"hooks,omitempty"`
	}{
		Name:        s.Name,
		Command:     s.Command,
		Description: s.Description,
		Env:         s.Env,
	}

	// Only include hooks if they exist
	if hooks.Pre != nil || hooks.Post != nil {
		obj.Hooks = &hooks
	}

	return json.Marshal(obj)
}

// Hook represents a script hook that runs before or after the main script
type Hook struct {
	Command     string            `json:"command"`
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

// RunOptions contains options for script execution
type RunOptions struct {
	Env              map[string]string // Additional environment variables
	WorkDir          string            // Working directory for script execution
	SkipHooksOnError bool              // Whether to continue if a hook fails
}

// NewScript creates a new Script instance
func NewScript(name, command, description string) *Script {
	return &Script{
		Name:        name,
		Command:     command,
		Description: description,
		Env:         make(map[string]string),
		Hooks:       make(map[string]*Hook),
	}
}

// AddHook adds a hook to the script
func (s *Script) AddHook(name string, hook *Hook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	if _, exists := s.Hooks[name]; exists {
		return fmt.Errorf("hook %s already exists", name)
	}

	s.Hooks[name] = hook
	return nil
}

// SetEnv sets an environment variable for the script
func (s *Script) SetEnv(key, value string) {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	s.Env[key] = value
}

// mergeEnv merges the script's environment variables with the system environment
// and any additional environment variables provided in RunOptions
func (s *Script) mergeEnv(opts *RunOptions) []string {
	env := os.Environ()
	merged := make(map[string]string)

	// Start with current environment
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			merged[parts[0]] = parts[1]
		}
	}

	// Add script-specific environment variables
	for k, v := range s.Env {
		merged[k] = v
	}

	// Add run options environment variables
	if opts != nil && opts.Env != nil {
		for k, v := range opts.Env {
			merged[k] = v
		}
	}

	// Convert back to string slice
	result := make([]string, 0, len(merged))
	for k, v := range merged {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// Execute runs the script with the given options
func (s *Script) Execute(opts *RunOptions) error {
	if s.Command == "" {
		return fmt.Errorf("script command cannot be empty")
	}

	// Split the command into parts
	parts := strings.Fields(s.Command)
	if len(parts) == 0 {
		return fmt.Errorf("invalid command format")
	}

	// Create command with the merged environment
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = s.mergeEnv(opts)

	// Set working directory if specified
	if opts != nil && opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Connect to standard streams
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Validate checks if the script is properly configured
func (s *Script) Validate() error {
	if s.Command == "" {
		return fmt.Errorf("script command cannot be empty")
	}

	// Validate hooks
	for name, hook := range s.Hooks {
		if hook == nil {
			return fmt.Errorf("hook %s is nil", name)
		}
		if hook.Command == "" {
			return fmt.Errorf("hook %s command cannot be empty", name)
		}
	}

	return nil
}
