package endpointapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockLogger 简单的 mock logger
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *MockLogger) Error(msg string, fields map[string]interface{}) {}

func TestNewClient(t *testing.T) {
	logger := &MockLogger{}
	client := NewClient("test-key", "http://test.com", "test-endpoint", logger)

	assert.Equal(t, "test-key", client.apiKey)
	assert.Equal(t, "http://test.com", client.url)
	assert.Equal(t, "test-endpoint", client.endpointID)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, logger, client.logger)
}

// TestEndpointApiError 测试 EndpointApiError 结构体
func TestEndpointApiError(t *testing.T) {
	message := "Test error message"
	stack := "Error: Test error\n    at testFunction (/app/test.js:123:45)"

	err := NewEndpointApiError(message, stack)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "EndpointApiError: Test error message")
	assert.Equal(t, stack, err.Stack())
}

// TestEndpointApiError_EmptyStack 测试 EndpointApiError 空堆栈
func TestEndpointApiError_EmptyStack(t *testing.T) {
	message := "Test error message"

	err := NewEndpointApiError(message, "")

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "EndpointApiError: Test error message")
	assert.Empty(t, err.Stack())
}
