package dashboard

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/afomera/spin/internal/config"
	"github.com/afomera/spin/internal/process"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandMsg represents the result of a command execution
type CommandMsg struct {
	Command string
	Output  string
	Error   error
}

// executeCommand runs a shell command and returns its output
func executeCommand(command string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", command)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		output := stdout.String()
		if err != nil {
			if stderr.Len() > 0 {
				output = stderr.String()
			} else {
				output = err.Error()
			}
			return CommandMsg{Command: command, Output: output, Error: err}
		}

		return CommandMsg{Command: command, Output: output}
	}
}

// New creates a new dashboard model with the given configuration
func New(cfg *config.Config) (*Model, error) {
	manager := process.GetManager(cfg)
	manager.SetQuiet(true) // Suppress stdout/stderr output

	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.CharLimit = 100
	ti.Width = 50

	// Load project name from config
	configData, err := os.ReadFile("spin.config.json")
	if err != nil {
		return nil, fmt.Errorf("error reading config: %v", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configData, &configMap); err != nil {
		return nil, fmt.Errorf("error parsing config: %v", err)
	}

	projectName := "Unnamed Project"
	if name, ok := configMap["name"].(string); ok {
		projectName = name
	}

	return &Model{
		Help:        help.New(),
		Manager:     manager,
		ViewMode:    DetailsMode,
		LogBuffer:   make([]string, 0, DefaultConfig().MaxLogBuffer),
		Input:       ti,
		InputActive: false,
		ProjectName: projectName,
	}, nil
}

// Init initializes the dashboard model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.tickCmd(),
		m.readLogsCmd(),
	)
}

// tickCmd returns a command that ticks every second
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(DefaultConfig().RefreshInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// readLogsCmd returns a command that reads from the log channel
func (m *Model) readLogsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.LogChan == nil {
			return nil
		}
		msg := <-m.LogChan
		return LogMsg(msg)
	}
}

// startLogReader starts reading logs for the specified process
func (m *Model) startLogReader(processName string) error {
	// Close existing log file if any
	if m.LogFile != nil {
		m.LogFile.Close()
		m.LogFile = nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	logPath := filepath.Join(home, ".spin", "output", fmt.Sprintf("%s.log", processName))
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}
	m.LogFile = file

	if m.LogChan == nil {
		m.LogChan = make(chan string)
	}

	m.LogBuffer = nil

	go func() {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			m.LogChan <- scanner.Text()
		}
		// Keep watching for new content
		for {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				m.LogChan <- scanner.Text()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	return nil
}

// handleKeyMsg handles keyboard input messages
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (*Model, tea.Cmd) {
	// Handle input mode
	if m.InputActive {
		return m.handleInputMode(msg)
	}

	// Handle search mode
	if m.ViewMode == LogsMode && m.Search.Active {
		return m.handleSearchMode(msg)
	}

	// Handle regular key bindings
	return m.handleRegularKeys(msg)
}

// handleInputMode handles keyboard input when in input mode
func (m *Model) handleInputMode(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.InputActive = false
		m.Input.Reset()
		return m, nil
	case tea.KeyEnter:
		cmd := m.Input.Value()
		if cmd != "" {
			m.Input.Reset()
			m.InputActive = false
			// Execute the command and update the output
			return m, executeCommand(cmd)
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.Input, cmd = m.Input.Update(msg)
		return m, cmd
	}
}

// handleSearchMode handles keyboard input when in search mode
func (m *Model) handleSearchMode(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.Search.Active = false
		m.Search.Term = ""
		m.filterLogs()
		return m, nil
	case tea.KeyBackspace:
		if len(m.Search.Term) > 0 {
			m.Search.Term = m.Search.Term[:len(m.Search.Term)-1]
			m.filterLogs()
		}
		return m, nil
	case tea.KeyRunes:
		m.Search.Term += string(msg.Runes)
		m.filterLogs()
		return m, nil
	}
	return m, nil
}

// handleRegularKeys handles keyboard input in regular mode
func (m *Model) handleRegularKeys(msg tea.KeyMsg) (*Model, tea.Cmd) {
	keys := DefaultKeyMap()

	switch {
	case key.Matches(msg, keys.Quit):
		m.Quitting = true
		return m, tea.Quit

	case key.Matches(msg, keys.Search):
		if m.ViewMode == LogsMode {
			m.Search.Active = true
			m.Search.Term = ""
			m.ErrorMsg = "Search mode: Type to filter logs, ESC to exit"
		}

	case key.Matches(msg, keys.Tab):
		if m.ActivePanel == ProcessList {
			m.ActivePanel = ProcessDetails
		} else {
			m.ActivePanel = ProcessList
		}

	case key.Matches(msg, keys.Up):
		if m.ActivePanel == ProcessList {
			if m.Cursor > 0 {
				m.Cursor--
				m.updateDetailsView()
			}
		} else {
			m.DetailsView.LineUp(1)
		}

	case key.Matches(msg, keys.Down):
		if m.ActivePanel == ProcessList {
			if m.Cursor < len(m.Processes)-1 {
				m.Cursor++
				m.updateDetailsView()
			}
		} else {
			m.DetailsView.LineDown(1)
		}

	case key.Matches(msg, keys.PageUp):
		if m.ActivePanel == ProcessDetails {
			m.DetailsView.HalfViewUp()
		}

	case key.Matches(msg, keys.PageDown):
		if m.ActivePanel == ProcessDetails {
			m.DetailsView.HalfViewDown()
		}

	case key.Matches(msg, keys.Stop):
		if len(m.Processes) > 0 && m.Cursor < len(m.Processes) {
			proc := m.Processes[m.Cursor]
			if err := m.Manager.StopProcess(proc.Name); err != nil {
				m.ErrorMsg = fmt.Sprintf("Error stopping process: %v", err)
			}
		}

	case key.Matches(msg, keys.Debug):
		if len(m.Processes) > 0 && m.Cursor < len(m.Processes) {
			proc := m.Processes[m.Cursor]
			if err := m.Manager.DebugProcess(proc.Name); err != nil {
				m.ErrorMsg = fmt.Sprintf("Error debugging process: %v", err)
			}
		}

	case key.Matches(msg, keys.Logs):
		if len(m.Processes) > 0 && m.Cursor < len(m.Processes) {
			if m.ViewMode == DetailsMode {
				m.ViewMode = LogsMode
				proc := m.Processes[m.Cursor]
				if err := m.startLogReader(proc.Name); err != nil {
					m.ErrorMsg = fmt.Sprintf("Error reading logs: %v", err)
					m.ViewMode = DetailsMode
				}
			} else {
				m.ViewMode = DetailsMode
			}
			m.updateDetailsView()
		}

	case key.Matches(msg, keys.ToggleInput):
		m.InputActive = !m.InputActive
		if m.InputActive {
			m.Input.Focus()
			return m, textinput.Blink
		}
		m.Input.Reset()
		return m, nil

	case key.Matches(msg, keys.Escape):
		if m.CommandOutput != "" {
			m.CommandOutput = ""
			m.OutputBuffer = nil
			return m, nil
		}
	}

	return m, nil
}

// filterLogs applies the current search term to the log buffer
func (m *Model) filterLogs() {
	if !m.Search.Active || m.Search.Term == "" {
		m.DetailsView.SetContent(strings.Join(m.LogBuffer, "\n"))
		return
	}

	var filtered []string
	searchTerm := m.Search.Term
	if !m.Search.MatchCase {
		searchTerm = strings.ToLower(searchTerm)
	}

	for _, line := range m.LogBuffer {
		compareLine := line
		if !m.Search.MatchCase {
			compareLine = strings.ToLower(line)
		}
		if strings.Contains(compareLine, searchTerm) {
			filtered = append(filtered, line)
		}
	}

	if len(filtered) > 0 {
		m.DetailsView.SetContent(strings.Join(filtered, "\n"))
	} else {
		m.DetailsView.SetContent("No matches found for: " + m.Search.Term)
	}
}

// Update handles updating the model based on messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case CommandMsg:
		// Format command output
		output := fmt.Sprintf("> %s\n%s", msg.Command, msg.Output)
		if msg.Error != nil {
			output = ErrorStyle.Render(output)
		}
		m.CommandOutput = output
		m.OutputBuffer = append(m.OutputBuffer, output)
		return m, nil

	case tea.KeyMsg:
		model, cmd := m.handleKeyMsg(msg)
		return model, cmd

	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)

	case TickMsg:
		m.LastUpdate = time.Time(msg)
		processes := m.Manager.ListProcesses()

		// Sort processes by name
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].Name < processes[j].Name
		})

		m.Processes = processes
		m.updateProcessView()
		if m.ViewMode == DetailsMode {
			m.updateDetailsView()
		}
		// Force rerender every second
		return m, tea.Batch(
			m.tickCmd(),
			m.readLogsCmd(),
			func() tea.Msg { return tea.WindowSizeMsg{Width: m.Width, Height: m.Height} },
		)

	case LogMsg:
		return m.handleLogMsg(msg)
	}

	// Handle viewport updates
	m.ProcessView, cmd = m.ProcessView.Update(msg)
	cmds = append(cmds, cmd)
	m.DetailsView, cmd = m.DetailsView.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleWindowResize handles window resize events
func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
	m.Width = msg.Width
	m.Height = msg.Height

	headerHeight := 3
	footerHeight := 4
	commandOutputHeight := 6
	verticalMargins := headerHeight + footerHeight + commandOutputHeight

	processWidth := 29                         // 25 chars + 4 for borders/padding
	detailsWidth := m.Width - processWidth - 2 // -2 for margin between boxes

	if !m.Ready {
		m.ProcessView = viewport.New(processWidth, m.Height-verticalMargins)
		m.DetailsView = viewport.New(detailsWidth, m.Height-verticalMargins-commandOutputHeight)
		m.Ready = true
	} else {
		m.ProcessView.Width = processWidth
		m.ProcessView.Height = m.Height - verticalMargins
		m.DetailsView.Width = detailsWidth
		m.DetailsView.Height = m.Height - verticalMargins - commandOutputHeight
	}

	return m, nil
}

// handleLogMsg handles new log messages
func (m *Model) handleLogMsg(msg LogMsg) (*Model, tea.Cmd) {
	if m.ViewMode == LogsMode {
		logLine := LogStyle.Render(string(msg))
		m.LogBuffer = append(m.LogBuffer, logLine)

		if m.Search.Active {
			m.filterLogs()
		} else {
			var content strings.Builder
			content.WriteString(m.DetailsView.View())
			content.WriteString("\n")
			content.WriteString(logLine)
			m.DetailsView.SetContent(content.String())
		}
		m.DetailsView.GotoBottom()
	}
	return m, m.readLogsCmd()
}

// updateProcessView updates the process list view
func (m *Model) updateProcessView() {
	var b strings.Builder

	for i, p := range m.Processes {
		cursor := " "
		if m.Cursor == i {
			cursor = ">"
		}

		// Status emoji
		statusEmoji := "🔴" // stopped
		if p.Status == process.StatusRunning {
			statusEmoji = "🟢"
		} else if p.Status == process.StatusStarting {
			statusEmoji = "🟡"
		}

		// Status style based on process state
		var statusStyle lipgloss.Style
		switch p.Status {
		case process.StatusRunning:
			statusStyle = RunningStyle
		case process.StatusStarting:
			statusStyle = StartingStyle
		default:
			statusStyle = StoppedStyle
		}

		// Format process line with resource usage
		// First line with name and status
		processLine := fmt.Sprintf("%s %s %s %s",
			cursor,
			p.Name,
			statusEmoji,
			statusStyle.Render(string(p.Status)),
		)
		processLine = fmt.Sprintf("%-25s\n", processLine) // Pad to 25 chars

		// Second line with resource usage
		resourceLine := fmt.Sprintf("  CPU: %.1f%% MEM: %.1f%%",
			p.CPUPercent,
			p.MemoryPercent,
		)
		processLine += fmt.Sprintf("%-25s", resourceLine) // Pad to 25 chars

		if i == m.Cursor {
			processLine = SelectedProcessStyle.Render(processLine)
		} else {
			processLine = ProcessItemStyle.Render(processLine)
		}

		b.WriteString(processLine)
	}

	if len(m.Processes) == 0 {
		b.WriteString("No processes running\n")
	}

	m.ProcessView.SetContent(b.String())
}

// updateDetailsView updates the details/logs view
func (m *Model) updateDetailsView() {
	var b strings.Builder

	if len(m.Processes) > 0 && m.Cursor < len(m.Processes) {
		proc := m.Processes[m.Cursor]

		if m.ViewMode == DetailsMode {
			b.WriteString(HeaderStyle.Render("Process Details") + "\n")
			b.WriteString(fmt.Sprintf("Name: %s\n", SelectedProcessStyle.Render(proc.Name)))
			b.WriteString(fmt.Sprintf("Status: %s\n", RunningStyle.Render(string(proc.Status))))
			b.WriteString(fmt.Sprintf("Debug Mode: %s\n", StoppedStyle.Render("Disabled")))

			b.WriteString("\n" + HeaderStyle.Render("Resource Usage") + "\n")
			b.WriteString(fmt.Sprintf("CPU: %.1f%%\n", proc.CPUPercent))
			b.WriteString(fmt.Sprintf("Memory: %.1f%% (%.2f MB)\n",
				proc.MemoryPercent,
				float64(proc.MemoryUsage)/(1024*1024),
			))
			b.WriteString(fmt.Sprintf("Last Updated: %s\n", proc.LastUpdated.Format("15:04:05")))

			if proc.OutputFile != "" {
				b.WriteString("\n" + HeaderStyle.Render("Log Information") + "\n")
				b.WriteString(fmt.Sprintf("Log File: %s\n", proc.OutputFile))
			}

			b.WriteString("\n" + InfoStyle.Render("Press 'l' to view logs"))
		} else {
			// Logs view
			b.WriteString(HeaderStyle.Render(fmt.Sprintf("Logs: %s", proc.Name)) + "\n")
			b.WriteString(InfoStyle.Render("Press 'l' to return to details"))
			b.WriteString(InfoStyle.Render(" • "))
			b.WriteString(InfoStyle.Render("Press '/' to search logs"))
			b.WriteString(InfoStyle.Render(" • "))
			b.WriteString(InfoStyle.Render("Use ↑/↓, PgUp/PgDn to scroll\n"))
			if m.Search.Active {
				b.WriteString(fmt.Sprintf("\nSearch: %s\n", m.Search.Term))
			}
		}
	} else {
		b.WriteString("Select a process to view details")
	}

	m.DetailsView.SetContent(b.String())
}
