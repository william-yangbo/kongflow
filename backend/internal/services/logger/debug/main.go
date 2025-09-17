package main

import (
	"fmt"
	"kongflow/backend/internal/services/logger"
)

func main() {
	fmt.Printf("LogLevelLog: %v\n", logger.LogLevelLog)
	fmt.Printf("LogLevelError: %v\n", logger.LogLevelError)
	fmt.Printf("LogLevelWarn: %v\n", logger.LogLevelWarn)
	fmt.Printf("LogLevelInfo: %v\n", logger.LogLevelInfo)
	fmt.Printf("LogLevelDebug: %v\n", logger.LogLevelDebug)
}
