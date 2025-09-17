// Package logger provides a structured logging solution that strictly aligns with trigger.dev's Logger implementation
// Replicates trigger.dev's Logger class behavior for perfect compatibility with Go best practices.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the logging level, aligned with trigger.dev's LogLevel type.
type LogLevel string

const (
	LogLevelLog   LogLevel = "log"
	LogLevelError LogLevel = "error"
	LogLevelWarn  LogLevel = "warn"
	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
)

// logLevels matches trigger.dev's logLevels array exactly
var logLevels = []LogLevel{"log", "error", "warn", "info", "debug"}

// Logger replicates trigger.dev's Logger class behavior exactly
type Logger struct {
	name   string
	level  int       // Array index matching trigger.dev's this.#level
	output io.Writer // For testing, defaults to os.Stdout
}

// New creates a new Logger instance with the specified name.
// Uses "info" as default level, matching trigger.dev's internal Logger default.
// For webapp-style loggers that need "debug" default, use NewWebapp().
func New(name string) *Logger {
	return NewWithLevel(name, "info", os.Stdout)
}

// NewWebapp creates a webapp-style logger matching trigger.dev's webapp logger.
// Uses "debug" as default level like trigger.dev's webapp service.
func NewWebapp(name string) *Logger {
	// Match trigger.dev webapp: (process.env.APP_LOG_LEVEL ?? "debug")
	envLevel := os.Getenv("KONGFLOW_LOG_LEVEL")
	defaultLevel := "debug"
	if envLevel != "" {
		defaultLevel = envLevel
	}
	return NewWithLevel(name, defaultLevel, os.Stdout)
}

// NewWithLevel creates a new Logger instance with specific level and output.
// Used for testing and custom configurations.
func NewWithLevel(name string, levelStr string, output io.Writer) *Logger {
	// Use KONGFLOW_LOG_LEVEL environment variable for KongFlow specific configuration
	envLevel := os.Getenv("KONGFLOW_LOG_LEVEL")
	if envLevel != "" {
		levelStr = envLevel
	}

	// Find level index in array, matching trigger.dev's logLevels.indexOf
	levelIndex := -1
	for i, l := range logLevels {
		if string(l) == levelStr {
			levelIndex = i
			break
		}
	}

	// Default to "info" if not found (index 3), matching trigger.dev behavior
	if levelIndex == -1 {
		levelIndex = 3 // "info"
	}

	return &Logger{
		name:   name,
		level:  levelIndex,
		output: output,
	}
}

// formattedDateTime replicates trigger.dev's formattedDateTime function exactly
func formattedDateTime() string {
	now := time.Now()

	hours := now.Hour()
	minutes := now.Minute()
	seconds := now.Second()
	milliseconds := now.Nanosecond() / 1000000

	// Make sure the time is always 2 digits
	formattedHours := fmt.Sprintf("%02d", hours)
	formattedMinutes := fmt.Sprintf("%02d", minutes)
	formattedSeconds := fmt.Sprintf("%02d", seconds)

	// Format milliseconds to 3 digits
	var formattedMilliseconds string
	if milliseconds < 10 {
		formattedMilliseconds = fmt.Sprintf("00%d", milliseconds)
	} else if milliseconds < 100 {
		formattedMilliseconds = fmt.Sprintf("0%d", milliseconds)
	} else {
		formattedMilliseconds = fmt.Sprintf("%d", milliseconds)
	}

	return fmt.Sprintf("%s:%s:%s.%s", formattedHours, formattedMinutes, formattedSeconds, formattedMilliseconds)
}

// Log implements trigger.dev's log method exactly
func (l *Logger) Log(args ...interface{}) {
	// Replicate: if (this.#level < 0) return;
	if l.level < 0 {
		return
	}

	// Replicate: console.log(`[${formattedDateTime()}] [${this.#name}] `, ...args);
	// Handle multiple args properly like JavaScript's console.log
	var message string
	if len(args) == 0 {
		message = ""
	} else if len(args) == 1 {
		message = fmt.Sprintf("%v", args[0])
	} else {
		message = fmt.Sprint(args...)
	}
	fmt.Fprintf(l.output, "[%s] [%s] %s\n", formattedDateTime(), l.name, message)
}

// Error implements trigger.dev's error method exactly
func (l *Logger) Error(args ...interface{}) {
	// Replicate: if (this.#level < 1) return;
	if l.level < 1 {
		return
	}

	// Replicate: console.error(`[${formattedDateTime()}] [${this.#name}] `, ...args);
	var message string
	if len(args) == 0 {
		message = ""
	} else if len(args) == 1 {
		message = fmt.Sprintf("%v", args[0])
	} else {
		message = fmt.Sprint(args...)
	}
	fmt.Fprintf(l.output, "[%s] [%s] %s\n", formattedDateTime(), l.name, message)
}

// Warn implements trigger.dev's warn method exactly
func (l *Logger) Warn(args ...interface{}) {
	// Replicate: if (this.#level < 2) return;
	if l.level < 2 {
		return
	}

	// Replicate: console.warn(`[${formattedDateTime()}] [${this.#name}] `, ...args);
	var message string
	if len(args) == 0 {
		message = ""
	} else if len(args) == 1 {
		message = fmt.Sprintf("%v", args[0])
	} else {
		message = fmt.Sprint(args...)
	}
	fmt.Fprintf(l.output, "[%s] [%s] %s\n", formattedDateTime(), l.name, message)
}

// Info implements trigger.dev's info method exactly
func (l *Logger) Info(args ...interface{}) {
	// Replicate: if (this.#level < 3) return;
	if l.level < 3 {
		return
	}

	// Replicate: console.info(`[${formattedDateTime()}] [${this.#name}] `, ...args);
	var message string
	if len(args) == 0 {
		message = ""
	} else if len(args) == 1 {
		message = fmt.Sprintf("%v", args[0])
	} else {
		message = fmt.Sprint(args...)
	}
	fmt.Fprintf(l.output, "[%s] [%s] %s\n", formattedDateTime(), l.name, message)
}

// Debug implements trigger.dev's debug method exactly with structured JSON output
func (l *Logger) Debug(message string, args ...interface{}) {
	// Replicate: if (this.#level < 4) return;
	if l.level < 4 {
		return
	}

	// Replicate trigger.dev's structured debug output exactly
	structuredLog := map[string]interface{}{
		"timestamp": time.Now(),
		"name":      l.name,
		"message":   message,
	}

	// Add args if provided (matching trigger.dev's structureArgs logic)
	if len(args) > 0 {
		if len(args) == 1 {
			// Single object case
			structuredLog["args"] = args[0]
		} else {
			// Multiple args case
			structuredLog["args"] = args
		}
	}

	// Replicate: console.debug(JSON.stringify(structuredLog, createReplacer(...)))
	jsonBytes, err := json.Marshal(structuredLog)
	if err != nil {
		// Fallback to simple format if JSON marshaling fails (Go best practice)
		fmt.Fprintf(l.output, "[%s] [%s] DEBUG: %s (JSON marshal error: %v)\n",
			formattedDateTime(), l.name, message, err)
		return
	}
	fmt.Fprintln(l.output, string(jsonBytes))
}

// GetName returns the logger's name, providing access to the name field
func (l *Logger) GetName() string {
	return l.name
}

// Logf provides printf-style logging for the log level (Go convenience)
func (l *Logger) Logf(format string, args ...interface{}) {
	l.Log(fmt.Sprintf(format, args...))
}

// Errorf provides printf-style logging for the error level (Go convenience)
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Warnf provides printf-style logging for the warn level (Go convenience)
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Infof provides printf-style logging for the info level (Go convenience)
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Debugf provides printf-style logging for the debug level (Go convenience)
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}
