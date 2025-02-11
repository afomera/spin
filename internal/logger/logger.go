package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

	scanner := bufio.NewScanner(bufio.NewReader(&prefixReader{data: p}))
	for scanner.Scan() {
		line := scanner.Text()
		// Trim trailing spaces but preserve empty lines
		trimmed := strings.TrimRight(line, " \t")
		prefix := fmt.Sprintf("%s[%s]%s ", w.color, w.name, Reset)
		if _, err := fmt.Fprintf(w.writer, "%s%s\n", prefix, trimmed); err != nil {
			return 0, err
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return len(p), nil
}

// prefixReader helps handle partial lines and preserves all characters
type prefixReader struct {
	data []byte
	pos  int
	buf  []byte
}

func (r *prefixReader) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		// First return any buffered data
		n = copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	// Find next newline
	start := r.pos
	end := r.pos
	for end < len(r.data) && r.data[end] != '\n' {
		end++
	}

	// Include the newline if found
	if end < len(r.data) {
		end++
	}

	// Trim trailing whitespace but keep newline
	trimEnd := end
	if end > start && r.data[end-1] == '\n' {
		trimEnd--
	}
	for trimEnd > start && (r.data[trimEnd-1] == ' ' || r.data[trimEnd-1] == '\t') {
		trimEnd--
	}
	if end > start && r.data[end-1] == '\n' {
		trimEnd++
	}

	// Copy what we can to p
	n = copy(p, r.data[start:trimEnd])

	// If we couldn't copy everything, buffer the rest
	if n < trimEnd-start {
		r.buf = make([]byte, trimEnd-start-n)
		copy(r.buf, r.data[start+n:trimEnd])
	}

	r.pos = end
	return n, nil
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
