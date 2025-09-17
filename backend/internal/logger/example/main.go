package main

import (
	"kongflow/backend/internal/logger"
)

func main() {
	// Create a logger instance with KongFlow configuration
	// Use KONGFLOW_LOG_LEVEL environment variable to control output
	log := logger.New("webapp")

	// Alternative: Create webapp-style logger (defaults to debug level)
	webappLog := logger.NewWebapp("webapp")
	_ = webappLog // For demonstration

	// Test all logging levels with KongFlow logger
	log.Log("This is a log message")
	log.Error("This is an error message", "error_code", 500)
	log.Warn("This is a warning message", "warning_type", "deprecation")
	log.Info("This is an info message", "request_id", "abc123")
	log.Debug("This is a debug message", map[string]interface{}{
		"user_id": "user_456",
		"action":  "login",
	})

	// Test Go-style printf methods (convenience)
	log.Infof("Service %s started successfully on port %d", "auth", 8080)
	log.Errorf("Connection failed: %v", "timeout")
}
