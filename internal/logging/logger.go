// Package logging provides structured logging for QuantumLife.
package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents log level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l Level) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m" // Cyan
	case INFO:
		return "\033[32m" // Green
	case WARN:
		return "\033[33m" // Yellow
	case ERROR:
		return "\033[31m" // Red
	default:
		return "\033[0m"
	}
}

// Logger is a structured logger
type Logger struct {
	level  Level
	output io.Writer
	mu     sync.Mutex
	fields map[string]interface{}
}

var defaultLogger = &Logger{
	level:  INFO,
	output: os.Stdout,
	fields: make(map[string]interface{}),
}

// SetLevel sets the global log level
func SetLevel(level Level) {
	defaultLogger.level = level
}

// SetOutput sets the output writer
func SetOutput(w io.Writer) {
	defaultLogger.output = w
}

// WithField returns a logger with a field added
func WithField(key string, value interface{}) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields returns a logger with multiple fields added
func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	reset := "\033[0m"

	formatted := msg
	if len(args) > 0 {
		formatted = fmt.Sprintf(msg, args...)
	}

	// Build fields string
	var fieldsStr string
	if len(l.fields) > 0 {
		fieldsStr = " |"
		for k, v := range l.fields {
			fieldsStr += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	fmt.Fprintf(l.output, "%s %s[%s]%s %s%s\n",
		timestamp, level.Color(), level.String(), reset, formatted, fieldsStr)
}

// Debug logs a debug message
func Debug(msg string, args ...interface{}) {
	defaultLogger.log(DEBUG, msg, args...)
}

// Info logs an info message
func Info(msg string, args ...interface{}) {
	defaultLogger.log(INFO, msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...interface{}) {
	defaultLogger.log(WARN, msg, args...)
}

// Error logs an error message
func Error(msg string, args ...interface{}) {
	defaultLogger.log(ERROR, msg, args...)
}

// Logger methods
func (l *Logger) Debug(msg string, args ...interface{}) { l.log(DEBUG, msg, args...) }
func (l *Logger) Info(msg string, args ...interface{})  { l.log(INFO, msg, args...) }
func (l *Logger) Warn(msg string, args ...interface{})  { l.log(WARN, msg, args...) }
func (l *Logger) Error(msg string, args ...interface{}) { l.log(ERROR, msg, args...) }
