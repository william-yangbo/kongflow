package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoggerCreation(t *testing.T) {
	logger := New("test")
	if logger.GetName() != "test" {
		t.Errorf("Expected logger name 'test', got '%s'", logger.GetName())
	}
}

func TestWebappLogger(t *testing.T) {
	// Clear environment variable first
	os.Unsetenv("KONGFLOW_LOG_LEVEL")

	logger := NewWebapp("webapp")
	if logger.GetName() != "webapp" {
		t.Errorf("Expected webapp logger name 'webapp', got '%s'", logger.GetName())
	}

	// Should default to debug level (index 4) like trigger.dev webapp
	if logger.level != 4 {
		t.Errorf("Expected webapp logger default level 4 (debug), got %d", logger.level)
	}
}

func TestEnvironmentVariablePrecedence(t *testing.T) {
	// Test KONGFLOW_LOG_LEVEL environment variable
	t.Setenv("KONGFLOW_LOG_LEVEL", "error")

	var buf bytes.Buffer
	logger := NewWithLevel("test", "debug", &buf)

	// Should use env var "error" (index 1), not parameter "debug" (index 4)
	if logger.level != 1 {
		t.Errorf("Expected level 1 (error), got %d", logger.level)
	}
}

func TestLogLevelHierarchy(t *testing.T) {
	testCases := []struct {
		setLevel   string
		shouldShow []string
		shouldHide []string
	}{
		{
			setLevel:   "log", // index 0
			shouldShow: []string{"log"},
			shouldHide: []string{"error", "warn", "info", "debug"},
		},
		{
			setLevel:   "error", // index 1
			shouldShow: []string{"log", "error"},
			shouldHide: []string{"warn", "info", "debug"},
		},
		{
			setLevel:   "warn", // index 2
			shouldShow: []string{"log", "error", "warn"},
			shouldHide: []string{"info", "debug"},
		},
		{
			setLevel:   "info", // index 3
			shouldShow: []string{"log", "error", "warn", "info"},
			shouldHide: []string{"debug"},
		},
		{
			setLevel:   "debug", // index 4
			shouldShow: []string{"log", "error", "warn", "info", "debug"},
			shouldHide: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run("level_"+tc.setLevel, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewWithLevel("test", tc.setLevel, &buf)

			// Test levels that should show
			for _, level := range tc.shouldShow {
				buf.Reset()
				switch level {
				case "log":
					logger.Log("test message")
				case "error":
					logger.Error("test message")
				case "warn":
					logger.Warn("test message")
				case "info":
					logger.Info("test message")
				case "debug":
					logger.Debug("test message")
				}

				if buf.Len() == 0 {
					t.Errorf("Level %s should be visible when set to %s, but no output", level, tc.setLevel)
				}
			}

			// Test levels that should be hidden
			for _, level := range tc.shouldHide {
				buf.Reset()
				switch level {
				case "log":
					logger.Log("test message")
				case "error":
					logger.Error("test message")
				case "warn":
					logger.Warn("test message")
				case "info":
					logger.Info("test message")
				case "debug":
					logger.Debug("test message")
				}

				if buf.Len() > 0 {
					t.Errorf("Level %s should be hidden when set to %s, but got output: %s", level, tc.setLevel, buf.String())
				}
			}
		})
	}
}

func TestOutputFormats(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithLevel("test", "debug", &buf)

	t.Run("simple_levels_use_console_format", func(t *testing.T) {
		// Test that log/error/warn/info use simple console format
		levels := []string{"log", "error", "warn", "info"}

		for _, level := range levels {
			buf.Reset()
			switch level {
			case "log":
				logger.Log("test message")
			case "error":
				logger.Error("test message")
			case "warn":
				logger.Warn("test message")
			case "info":
				logger.Info("test message")
			}

			output := buf.String()
			// Should contain timestamp, name, and message in console format
			if !strings.Contains(output, "[test]") {
				t.Errorf("Level %s output should contain logger name [test], got: %s", level, output)
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("Level %s output should contain message, got: %s", level, output)
			}
			// Should NOT be JSON
			if strings.HasPrefix(output, "{") {
				t.Errorf("Level %s should use console format, not JSON: %s", level, output)
			}
		}
	})

	t.Run("debug_uses_json_format", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message", map[string]interface{}{"key": "value"})

		output := buf.String()

		// Should be valid JSON
		var jsonLog map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &jsonLog); err != nil {
			t.Errorf("Debug output should be valid JSON, got: %s, error: %v", output, err)
		}

		// Should contain required fields
		if jsonLog["timestamp"] == nil {
			t.Error("Debug JSON should contain timestamp field")
		}
		if jsonLog["name"] != "test" {
			t.Errorf("Debug JSON should contain name=test, got: %v", jsonLog["name"])
		}
		if jsonLog["message"] != "debug message" {
			t.Errorf("Debug JSON should contain message='debug message', got: %v", jsonLog["message"])
		}
		if jsonLog["args"] == nil {
			t.Error("Debug JSON should contain args field")
		}
	})
}

func TestDefaultBehavior(t *testing.T) {
	// Clear any existing environment variable
	os.Unsetenv("KONGFLOW_LOG_LEVEL")

	var buf bytes.Buffer
	logger := NewWithLevel("test", "info", &buf)

	// Default should be "info" (index 3) when not specified
	if logger.level != 3 {
		t.Errorf("Expected default level 3 (info), got %d", logger.level)
	}

	// Test that info and above are shown
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Error("Info message should be shown with default level")
	}

	buf.Reset()
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should be hidden with default level")
	}
}

func TestGoStyleMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithLevel("test", "debug", &buf)

	// Test printf-style methods
	logger.Logf("Log message %d", 1)
	if !strings.Contains(buf.String(), "Log message 1") {
		t.Errorf("Logf should format message, got: %s", buf.String())
	}

	buf.Reset()
	logger.Errorf("Error %s", "occurred")
	if !strings.Contains(buf.String(), "Error occurred") {
		t.Errorf("Errorf should format message, got: %s", buf.String())
	}

	buf.Reset()
	logger.Infof("Info: %v", true)
	if !strings.Contains(buf.String(), "Info: true") {
		t.Errorf("Infof should format message, got: %s", buf.String())
	}
}
