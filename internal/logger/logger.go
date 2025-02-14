package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Colors for different log types
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
)

var (
	verbose bool
	mu      sync.Mutex
)

// SetVerbose enables or disables verbose output
func SetVerbose(v bool) {
	mu.Lock()
	verbose = v
	mu.Unlock()
}

// IsVerbose returns whether verbose output is enabled
func IsVerbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

// Debug writes a debug message if verbose mode is enabled
func Debug(format string, args ...interface{}) {
	if IsVerbose() {
		prefix := fmt.Sprintf("%s[debug]%s ", Yellow, Reset)
		fmt.Printf(prefix+format, args...)
	}
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// PrefixedWriter wraps an io.Writer to prefix each line with a colored tag
type PrefixedWriter struct {
	name   string
	color  string
	writer io.Writer
	mu     sync.Mutex
}

// NewPrefixedWriter creates a new PrefixedWriter
func NewPrefixedWriter(name string, color string, writer io.Writer) *PrefixedWriter {
	return &PrefixedWriter{
		name:   name,
		color:  color,
		writer: writer,
	}
}

// Write implements io.Writer
func (w *PrefixedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	prefix := fmt.Sprintf("%s[%s]%s ", w.color, w.name, Reset)
	if _, err := fmt.Fprintf(w.writer, prefix+"%s", string(p)); err != nil {
		return 0, err
	}

	return len(p), nil
}

// GetColorForService returns a consistent color for a service name
func GetColorForService(name string) string {
	colors := []string{
		Green,  // rails
		Blue,   // web
		Yellow, // worker
		Cyan,   // tailwind
		Purple, // redis
		Red,    // postgres
	}

	// Simple hash function to pick a color
	var sum int
	for _, c := range name {
		sum += int(c)
	}
	return colors[sum%len(colors)]
}

// CreatePrefixedWriter creates a new PrefixedWriter with a consistent color
func CreatePrefixedWriter(name string) io.Writer {
	return NewPrefixedWriter(name, GetColorForService(name), os.Stdout)
}

// MultiWriter creates a writer that duplicates its writes to all provided writers
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}
