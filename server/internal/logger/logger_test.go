package logger

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARNING", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // Default to INFO
		{"", slog.LevelInfo},         // Default to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Load config from non-existent file
	config, err := LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Fatalf("LoadConfig returned error for missing file: %v", err)
	}

	// Verify defaults
	if config.Level != "INFO" {
		t.Errorf("Default level = %q, want %q", config.Level, "INFO")
	}
	if !config.ConsoleEnabled {
		t.Error("Default ConsoleEnabled = false, want true")
	}
	if config.ConsoleFormat != "text" {
		t.Errorf("Default ConsoleFormat = %q, want %q", config.ConsoleFormat, "text")
	}
	if config.FileEnabled {
		t.Error("Default FileEnabled = true, want false")
	}
	if config.FilePath != "logs/server.log" {
		t.Errorf("Default FilePath = %q, want %q", config.FilePath, "logs/server.log")
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Create a temporary YAML file
	tmpFile, err := os.CreateTemp("", "logging-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	yamlContent := `logging:
  level: DEBUG
  console_enabled: true
  console_format: json
  file_enabled: true
  file_path: test.log
  file_max_size_mb: 20
`
	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Load config
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Verify loaded values
	if config.Level != "DEBUG" {
		t.Errorf("Level = %q, want %q", config.Level, "DEBUG")
	}
	if config.ConsoleFormat != "json" {
		t.Errorf("ConsoleFormat = %q, want %q", config.ConsoleFormat, "json")
	}
	if !config.FileEnabled {
		t.Error("FileEnabled = false, want true")
	}
	if config.FilePath != "test.log" {
		t.Errorf("FilePath = %q, want %q", config.FilePath, "test.log")
	}
	if config.FileMaxSizeMB != 20 {
		t.Errorf("FileMaxSizeMB = %d, want %d", config.FileMaxSizeMB, 20)
	}
}

func TestEnvVarOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("LOG_LEVEL", "ERROR")
	os.Setenv("LOG_CONSOLE_FORMAT", "json")
	os.Setenv("LOG_FILE_ENABLED", "true")
	os.Setenv("LOG_FILE_PATH", "/custom/path.log")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_CONSOLE_FORMAT")
		os.Unsetenv("LOG_FILE_ENABLED")
		os.Unsetenv("LOG_FILE_PATH")
	}()

	// Load config (no file)
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Verify env var overrides
	if config.Level != "ERROR" {
		t.Errorf("Level = %q, want %q (from env var)", config.Level, "ERROR")
	}
	if config.ConsoleFormat != "json" {
		t.Errorf("ConsoleFormat = %q, want %q (from env var)", config.ConsoleFormat, "json")
	}
	if !config.FileEnabled {
		t.Error("FileEnabled = false, want true (from env var)")
	}
	if config.FilePath != "/custom/path.log" {
		t.Errorf("FilePath = %q, want %q (from env var)", config.FilePath, "/custom/path.log")
	}
}

func TestInitializeWithTextFormat(t *testing.T) {
	var buf bytes.Buffer

	// Create a custom logger for testing
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger = slog.New(handler)

	// Log some messages
	Info("Test message", "key", "value")
	Debug("This should not appear") // Below INFO level

	output := buf.String()

	if !strings.Contains(output, "Test message") {
		t.Errorf("Output missing INFO message: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Output missing structured field: %s", output)
	}
	if strings.Contains(output, "This should not appear") {
		t.Errorf("Output contains DEBUG message when level is INFO: %s", output)
	}
}

func TestInitializeWithJSONFormat(t *testing.T) {
	var buf bytes.Buffer

	// Create a custom logger for testing
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger = slog.New(handler)

	// Log a message
	Info("JSON test", "field1", "value1", "field2", 42)

	output := buf.String()

	// Verify JSON format (should contain JSON structure)
	if !strings.Contains(output, `"msg":"JSON test"`) {
		t.Errorf("Output missing JSON message field: %s", output)
	}
	if !strings.Contains(output, `"field1":"value1"`) {
		t.Errorf("Output missing JSON field: %s", output)
	}
	if !strings.Contains(output, `"field2":42`) {
		t.Errorf("Output missing numeric JSON field: %s", output)
	}
}

func TestAlwaysBypassesLogLevel(t *testing.T) {
	var buf bytes.Buffer

	// Create logger with ERROR level (highest standard level)
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				if level == LevelAlways {
					a.Value = slog.StringValue("ALWAYS")
				}
			}
			return a
		},
	})
	logger = slog.New(handler)

	// Log at different levels
	Debug("Debug message")   // Should not appear
	Info("Info message")     // Should not appear
	Warning("Warning")       // Should not appear
	Error("Error message")   // Should appear
	Always("Always message") // Should appear (bypasses level filter)

	output := buf.String()

	// Only ERROR and ALWAYS should appear
	if strings.Contains(output, "Debug message") {
		t.Error("DEBUG appeared when level is ERROR")
	}
	if strings.Contains(output, "Info message") {
		t.Error("INFO appeared when level is ERROR")
	}
	if strings.Contains(output, "Warning") {
		t.Error("WARNING appeared when level is ERROR")
	}
	if !strings.Contains(output, "Error message") {
		t.Error("ERROR message missing from output")
	}
	if !strings.Contains(output, "Always message") {
		t.Error("ALWAYS message missing from output (should bypass level filter)")
	}
	if !strings.Contains(output, "level=ALWAYS") {
		t.Error("ALWAYS level not formatted correctly")
	}
}

func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger = slog.New(handler)

	// Test formatted variants
	Debugf("Debug: %d + %d = %d", 1, 2, 3)
	Infof("Info: %s", "test")
	Warningf("Warning: %.2f%%", 99.95)
	Errorf("Error: %v", "failed")
	Alwaysf("Always: %s %d", "count", 5)

	output := buf.String()

	if !strings.Contains(output, "Debug: 1 + 2 = 3") {
		t.Error("Debugf output incorrect")
	}
	if !strings.Contains(output, "Info: test") {
		t.Error("Infof output incorrect")
	}
	if !strings.Contains(output, "Warning: 99.95%") {
		t.Error("Warningf output incorrect")
	}
	if !strings.Contains(output, "Error: failed") {
		t.Error("Errorf output incorrect")
	}
	if !strings.Contains(output, "Always: count 5") {
		t.Error("Alwaysf output incorrect")
	}
}

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	// Create two handlers
	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Create multi-handler
	multiH := newMultiHandler(handler1, handler2)
	logger = slog.New(multiH)

	// Log a message
	Info("Multi-handler test", "field", "value")

	output1 := buf1.String()
	output2 := buf2.String()

	// Both outputs should contain the message
	if !strings.Contains(output1, "Multi-handler test") {
		t.Error("First handler did not receive message")
	}
	if !strings.Contains(output2, "Multi-handler test") {
		t.Error("Second handler did not receive message")
	}
	if !strings.Contains(output1, "field=value") {
		t.Error("First handler missing structured field")
	}
	if !strings.Contains(output2, "field=value") {
		t.Error("Second handler missing structured field")
	}
}

func TestNilLogger(t *testing.T) {
	// Set logger to nil to test defensive nil checks
	logger = nil

	// These should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logging with nil logger caused panic: %v", r)
		}
	}()

	Debug("debug")
	Info("info")
	Warning("warning")
	Error("error")
	Always("always")
}
