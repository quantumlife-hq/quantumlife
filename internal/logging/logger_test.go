package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevel_Color(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "\033[36m"},  // Cyan
		{INFO, "\033[32m"},   // Green
		{WARN, "\033[33m"},   // Yellow
		{ERROR, "\033[31m"},  // Red
		{Level(99), "\033[0m"}, // Reset for unknown
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.Color(); got != tt.want {
				t.Errorf("Level.Color() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	// Save original
	origLevel := defaultLogger.level
	defer func() { defaultLogger.level = origLevel }()

	SetLevel(DEBUG)
	if defaultLogger.level != DEBUG {
		t.Error("SetLevel did not change level")
	}

	SetLevel(ERROR)
	if defaultLogger.level != ERROR {
		t.Error("SetLevel did not change level")
	}
}

func TestSetOutput(t *testing.T) {
	// Save original
	origOutput := defaultLogger.output
	defer func() { defaultLogger.output = origOutput }()

	var buf bytes.Buffer
	SetOutput(&buf)

	if defaultLogger.output != &buf {
		t.Error("SetOutput did not change output")
	}
}

func TestWithField(t *testing.T) {
	logger := WithField("key", "value")

	if logger == nil {
		t.Fatal("WithField returned nil")
	}
	if logger.fields["key"] != "value" {
		t.Error("field not set correctly")
	}
	// Should be a new logger
	if len(defaultLogger.fields) > 0 {
		t.Error("should not modify default logger")
	}
}

func TestWithFields(t *testing.T) {
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	logger := WithFields(fields)

	if logger == nil {
		t.Fatal("WithFields returned nil")
	}
	if logger.fields["key1"] != "value1" {
		t.Error("field key1 not set correctly")
	}
	if logger.fields["key2"] != 42 {
		t.Error("field key2 not set correctly")
	}
}

func TestLogger_WithField(t *testing.T) {
	base := &Logger{
		level:  INFO,
		output: os.Stdout,
		fields: map[string]interface{}{"existing": "value"},
	}

	logger := base.WithField("new", "field")

	// New logger should have both fields
	if logger.fields["existing"] != "value" {
		t.Error("existing field not preserved")
	}
	if logger.fields["new"] != "field" {
		t.Error("new field not added")
	}

	// Original should be unchanged
	if _, ok := base.fields["new"]; ok {
		t.Error("original logger was modified")
	}
}

func TestLogger_WithFields(t *testing.T) {
	base := &Logger{
		level:  INFO,
		output: os.Stdout,
		fields: map[string]interface{}{"existing": "value"},
	}

	newFields := map[string]interface{}{
		"new1": "value1",
		"new2": "value2",
	}
	logger := base.WithFields(newFields)

	if len(logger.fields) != 3 {
		t.Errorf("got %d fields, want 3", len(logger.fields))
	}
	if logger.fields["existing"] != "value" {
		t.Error("existing field not preserved")
	}
	if logger.fields["new1"] != "value1" {
		t.Error("new field not added")
	}
}

func TestLogger_log_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  WARN,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	// DEBUG and INFO should be filtered
	logger.log(DEBUG, "debug message")
	if buf.Len() > 0 {
		t.Error("DEBUG should be filtered when level is WARN")
	}

	logger.log(INFO, "info message")
	if buf.Len() > 0 {
		t.Error("INFO should be filtered when level is WARN")
	}

	// WARN and ERROR should pass
	logger.log(WARN, "warn message")
	if buf.Len() == 0 {
		t.Error("WARN should not be filtered")
	}

	buf.Reset()
	logger.log(ERROR, "error message")
	if buf.Len() == 0 {
		t.Error("ERROR should not be filtered")
	}
}

func TestLogger_log_Format(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	logger.log(INFO, "test message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("output should contain level")
	}
	if !strings.Contains(output, "test message") {
		t.Error("output should contain message")
	}
}

func TestLogger_log_FormatWithArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	logger.log(INFO, "value: %d", 42)

	output := buf.String()
	if !strings.Contains(output, "value: 42") {
		t.Errorf("output should contain formatted value: %s", output)
	}
}

func TestLogger_log_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		output: &buf,
		fields: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	logger.log(INFO, "test")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Error("output should contain field key1")
	}
	if !strings.Contains(output, "key2=42") {
		t.Error("output should contain field key2")
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	origOutput := defaultLogger.output
	origLevel := defaultLogger.level
	defer func() {
		defaultLogger.output = origOutput
		defaultLogger.level = origLevel
	}()

	SetOutput(&buf)
	SetLevel(DEBUG)

	Debug("test debug")

	if !strings.Contains(buf.String(), "[DEBUG]") {
		t.Error("Debug should output DEBUG level")
	}
	if !strings.Contains(buf.String(), "test debug") {
		t.Error("Debug should output message")
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	origOutput := defaultLogger.output
	origLevel := defaultLogger.level
	defer func() {
		defaultLogger.output = origOutput
		defaultLogger.level = origLevel
	}()

	SetOutput(&buf)
	SetLevel(DEBUG)

	Info("test info")

	if !strings.Contains(buf.String(), "[INFO]") {
		t.Error("Info should output INFO level")
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	origOutput := defaultLogger.output
	origLevel := defaultLogger.level
	defer func() {
		defaultLogger.output = origOutput
		defaultLogger.level = origLevel
	}()

	SetOutput(&buf)
	SetLevel(DEBUG)

	Warn("test warn")

	if !strings.Contains(buf.String(), "[WARN]") {
		t.Error("Warn should output WARN level")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	origOutput := defaultLogger.output
	origLevel := defaultLogger.level
	defer func() {
		defaultLogger.output = origOutput
		defaultLogger.level = origLevel
	}()

	SetOutput(&buf)
	SetLevel(DEBUG)

	Error("test error")

	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Error("Error should output ERROR level")
	}
}

func TestLogger_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug msg")
		if !strings.Contains(buf.String(), "[DEBUG]") {
			t.Error("Logger.Debug should output DEBUG level")
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info msg")
		if !strings.Contains(buf.String(), "[INFO]") {
			t.Error("Logger.Info should output INFO level")
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn msg")
		if !strings.Contains(buf.String(), "[WARN]") {
			t.Error("Logger.Warn should output WARN level")
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error("error msg")
		if !strings.Contains(buf.String(), "[ERROR]") {
			t.Error("Logger.Error should output ERROR level")
		}
	})
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DEBUG,
		output: &buf,
		fields: make(map[string]interface{}),
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			logger.Info("message %d", n)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 log lines
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 log lines, got %d", len(lines))
	}
}

func TestLoggerFieldsImmutability(t *testing.T) {
	base := &Logger{
		level:  INFO,
		output: os.Stdout,
		fields: map[string]interface{}{"base": "value"},
	}

	// Create derived logger
	derived := base.WithField("derived", "field")

	// Modify the fields map in derived
	derived.fields["modified"] = "value"

	// Base should be unchanged
	if _, ok := base.fields["derived"]; ok {
		t.Error("base fields should not have derived field")
	}
	if _, ok := base.fields["modified"]; ok {
		t.Error("base fields should not have modified field")
	}
}

func TestLogLevelConstants(t *testing.T) {
	// Test ordering: DEBUG < INFO < WARN < ERROR
	if DEBUG >= INFO {
		t.Error("DEBUG should be less than INFO")
	}
	if INFO >= WARN {
		t.Error("INFO should be less than WARN")
	}
	if WARN >= ERROR {
		t.Error("WARN should be less than ERROR")
	}
}

func TestDefaultLoggerInitialization(t *testing.T) {
	if defaultLogger == nil {
		t.Fatal("defaultLogger should be initialized")
	}
	if defaultLogger.level != INFO {
		t.Error("default level should be INFO")
	}
	if defaultLogger.output != os.Stdout {
		t.Error("default output should be os.Stdout")
	}
	if defaultLogger.fields == nil {
		t.Error("default fields should be initialized")
	}
}
