package dashboard

import (
	"os"
	"time"

	"github.com/afomera/spin/internal/process"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

// Panel represents different UI panels in the dashboard
type Panel int

const (
	ProcessList Panel = iota
	ProcessDetails
)

// ViewMode represents different view modes for the details panel
type ViewMode int

const (
	DetailsMode ViewMode = iota
	LogsMode
	SearchMode
)

// SearchState holds the current search configuration
type SearchState struct {
	Active    bool
	Term      string
	MatchCase bool
}

// Model represents the application state
type Model struct {
	// Process-related fields
	Processes []*process.Process
	Cursor    int
	Manager   *process.Manager

	// UI components
	Help        help.Model
	ProcessView viewport.Model
	DetailsView viewport.Model
	Input       textinput.Model

	// Window dimensions
	Width  int
	Height int
	Ready  bool

	// View state
	ActivePanel   Panel
	ViewMode      ViewMode
	Quitting      bool
	InputActive   bool
	LastUpdate    time.Time
	ErrorMsg      string
	CommandOutput string
	ProjectName   string

	// Logging
	LogChan      chan string
	LogFile      *os.File
	LogBuffer    []string
	OutputBuffer []string
	Search       SearchState
}

// TickMsg is sent when we should update process information
type TickMsg time.Time

// LogMsg is sent when new log content is available
type LogMsg string

// Config holds the dashboard configuration
type Config struct {
	// Add any dashboard-specific configuration options here
	RefreshInterval time.Duration
	MaxLogBuffer    int
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		RefreshInterval: time.Second,
		MaxLogBuffer:    1000,
	}
}
