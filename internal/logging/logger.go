// Package logging provides structured JSON logging for Magicbox.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Field is a key-value pair attached to a log entry.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger writes structured JSON log entries to a file.
type Logger struct {
	output *os.File
	mu     sync.Mutex
}

// New creates a Logger that writes to a date-stamped log file in logDir.
func New(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	filename := fmt.Sprintf("magicbox-%s.log", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(logDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{output: f}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output.Close()
}

// Info logs a message at INFO level.
func (l *Logger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log("WARN", msg, fields)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields)
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields)
}

func (l *Logger) log(level, msg string, fields []Field) {
	entry := make(map[string]interface{})
	entry["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level
	entry["msg"] = msg

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		// Best-effort fallback if marshaling fails.
		fmt.Fprintf(l.output, `{"ts":"%s","level":"ERROR","msg":"failed to marshal log entry"}%s`,
			time.Now().UTC().Format(time.RFC3339Nano), "\n")
		fmt.Fprintf(os.Stdout, `{"ts":"%s","level":"ERROR","msg":"failed to marshal log entry"}%s`,
			time.Now().UTC().Format(time.RFC3339Nano), "\n")
		return
	}
	l.output.Write(data)
	l.output.Write([]byte("\n"))
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}
