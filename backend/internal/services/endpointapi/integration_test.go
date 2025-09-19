package endpointapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestClient_Integration_Ping 集成测试 Ping 方法
func TestClient_Integration_Ping(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求格式
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-key", r.Header.Get("x-trigger-api-key"))
		assert.Equal(t, "test-endpoint", r.Header.Get("x-trigger-endpoint-id"))
		assert.Equal(t, "PING", r.Header.Get("x-trigger-action"))

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	result, err := client.Ping(context.Background())

	assert.NoError(t, err)
	assert.True(t, result.OK)
	assert.Empty(t, result.Error)

	// 验证日志记录
	// 注意：简化的 MockLogger 不保存调用历史
}

// TestClient_Integration_IndexEndpoint 集成测试 IndexEndpoint 方法
func TestClient_Integration_IndexEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "INDEX_ENDPOINT", r.Header.Get("x-trigger-action"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"jobs": [
				{
					"id": "job-1",
					"name": "Test Job",
					"version": "1.0.0",
					"trigger": {"type": "http"}
				}
			],
			"sources": [
				{
					"id": "source-1",
					"name": "Test Source",
					"version": "1.0.0",
					"source": {"type": "webhook"}
				}
			]
		}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	result, err := client.IndexEndpoint(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result.Jobs, 1)
	assert.Equal(t, "job-1", result.Jobs[0].ID)
	assert.Equal(t, "Test Job", result.Jobs[0].Name)
	assert.Len(t, result.Sources, 1)
	assert.Equal(t, "source-1", result.Sources[0].ID)
}

// TestClient_Integration_DeliverEvent 集成测试 DeliverEvent 方法
func TestClient_Integration_DeliverEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "DELIVER_EVENT", r.Header.Get("x-trigger-action"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"success":true,"message":"Event delivered"}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	event := &ApiEventLog{
		ID:        "event-123",
		Name:      "test.event",
		Payload:   map[string]interface{}{"key": "value"},
		Context:   map[string]interface{}{"userId": "user-123"},
		Timestamp: time.Now(),
	}

	result, err := client.DeliverEvent(context.Background(), event)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "Event delivered", result.Message)
}

// TestClient_Integration_ErrorHandling 集成测试错误处理
func TestClient_Integration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectPongResp bool
	}{
		{
			name:           "Unauthorized",
			statusCode:     401,
			responseBody:   "",
			expectedError:  "Trigger API key is invalid",
			expectPongResp: true,
		},
		{
			name:           "Server Error",
			statusCode:     500,
			responseBody:   "",
			expectedError:  "Status code: 500",
			expectPongResp: true,
		},
		{
			name:           "Network Timeout",
			statusCode:     0, // 表示网络错误
			responseBody:   "",
			expectedError:  "Could not connect to endpoint",
			expectPongResp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server

			if tt.statusCode > 0 {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					w.Write([]byte(tt.responseBody))
				}))
				defer server.Close()
			}

			logger := &MockLogger{}
			var client *Client

			if tt.statusCode == 0 {
				// 模拟网络错误 - 使用无效的 URL
				client = NewClient("test-key", "http://invalid-url-that-should-fail", "test-endpoint", logger)
			} else {
				client = NewClient("test-key", server.URL, "test-endpoint", logger)
			}

			result, err := client.Ping(context.Background())

			if tt.expectPongResp {
				assert.NoError(t, err)
				assert.False(t, result.OK)
				assert.Contains(t, result.Error, tt.expectedError)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestClient_Integration_Timeout 集成测试超时处理
func TestClient_Integration_Timeout(t *testing.T) {
	// 创建一个慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 延迟 2 秒
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	// 创建一个超时时间很短的客户端
	httpClient := &http.Client{Timeout: 100 * time.Millisecond}
	client := NewClientWithHTTPClient("test-key", server.URL, "test-endpoint", httpClient, logger)

	result, err := client.Ping(context.Background())

	// 应该返回连接失败的 PongResponse
	assert.NoError(t, err)
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "Could not connect to endpoint")

	// 验证错误日志记录
	// 注意：简化的 MockLogger 不保存调用历史
}

// TestClient_Integration_ExecuteJobRequest 集成测试 ExecuteJobRequest 方法
func TestClient_Integration_ExecuteJobRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("content-type"))
		assert.Equal(t, "EXECUTE_JOB", r.Header.Get("x-trigger-action"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"run-123","status":"COMPLETED","output":{"result":"success"}}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	options := &RunJobBody{
		ID:      "job-123",
		Payload: map[string]interface{}{"input": "test data"},
		Context: map[string]interface{}{"userId": "user-123"},
	}

	result, err := client.ExecuteJobRequest(context.Background(), options)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Response)
	assert.NotNil(t, result.Parse)

	// 测试解析功能
	var response map[string]interface{}
	err = result.Parse(&response)
	assert.NoError(t, err)
	assert.Equal(t, "run-123", response["id"])
	assert.Equal(t, "COMPLETED", response["status"])
}

// TestClient_Integration_ExecuteJobRequest_MarshalError 测试 JSON 序列化错误
func TestClient_Integration_ExecuteJobRequest_MarshalError(t *testing.T) {
	logger := &MockLogger{}
	client := NewClient("test-key", "http://test.com", "test-endpoint", logger)

	// 创建一个无法序列化的 RunJobBody（包含循环引用）
	invalidData := make(map[string]interface{})
	invalidData["self"] = invalidData

	options := &RunJobBody{
		ID:      "job-123",
		Payload: invalidData,
	}

	result, err := client.ExecuteJobRequest(context.Background(), options)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to marshal request")
}

// TestClient_Integration_InitializeTrigger 集成测试 InitializeTrigger 方法
func TestClient_Integration_InitializeTrigger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "INITIALIZE_TRIGGER", r.Header.Get("x-trigger-action"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"id": "trigger-123",
			"params": {"webhookUrl": "https://test.com/hook"},
			"source": {"type": "webhook"},
			"job": {"id": "job-456", "name": "Test Job"}
		}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	params := map[string]interface{}{
		"webhookUrl": "https://test.com/hook",
		"events":     []string{"push", "pull_request"},
	}

	result, err := client.InitializeTrigger(context.Background(), "trigger-123", params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "trigger-123", result.ID)
	assert.NotNil(t, result.Params)
	assert.Equal(t, "https://test.com/hook", result.Params["webhookUrl"])
	assert.NotNil(t, result.Source)
	assert.NotNil(t, result.Job)
}

// TestClient_Integration_InitializeTrigger_EndpointApiError 测试 EndpointApiError 处理
func TestClient_Integration_InitializeTrigger_EndpointApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write([]byte(`{
			"message": "Invalid trigger configuration",
			"stack": "Error: Invalid trigger configuration\n    at validateTrigger (/app/src/trigger.js:123:45)"
		}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	params := map[string]interface{}{
		"invalid": "config",
	}

	result, err := client.InitializeTrigger(context.Background(), "trigger-123", params)

	assert.Error(t, err)
	assert.Nil(t, result)

	// 验证返回的是 EndpointApiError
	endpointErr, ok := err.(*EndpointApiError)
	assert.True(t, ok)
	assert.Contains(t, endpointErr.Error(), "Invalid trigger configuration")
	assert.Contains(t, endpointErr.Stack(), "validateTrigger")
}

// TestClient_Integration_InitializeTrigger_MarshalError 测试 JSON 序列化错误
func TestClient_Integration_InitializeTrigger_MarshalError(t *testing.T) {
	logger := &MockLogger{}
	client := NewClient("test-key", "http://test.com", "test-endpoint", logger)

	// 创建一个无法序列化的参数（包含循环引用）
	invalidParams := make(map[string]interface{})
	invalidParams["self"] = invalidParams

	result, err := client.InitializeTrigger(context.Background(), "trigger-123", invalidParams)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to marshal request")
}

// TestClient_Integration_PreprocessRunRequest 集成测试 PreprocessRunRequest 方法
func TestClient_Integration_PreprocessRunRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "PREPROCESS_RUN", r.Header.Get("x-trigger-action"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"success": true,
			"data": {
				"processedPayload": {"input": "processed data"},
				"metadata": {"timestamp": "2025-09-19T12:00:00Z"}
			}
		}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	options := &PreprocessRunBody{
		ID:      "run-123",
		Payload: map[string]interface{}{"input": "raw data"},
		Context: map[string]interface{}{"userId": "user-123"},
	}

	result, err := client.PreprocessRunRequest(context.Background(), options)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Response)
	assert.NotNil(t, result.Parse)

	// 测试解析功能
	var response PreprocessRunResponse
	err = result.Parse(&response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	assert.Contains(t, response.Data, "processedPayload")
}

// TestClient_Integration_PreprocessRunRequest_MarshalError 测试 JSON 序列化错误
func TestClient_Integration_PreprocessRunRequest_MarshalError(t *testing.T) {
	logger := &MockLogger{}
	client := NewClient("test-key", "http://test.com", "test-endpoint", logger)

	// 创建一个无法序列化的 PreprocessRunBody（包含循环引用）
	invalidData := make(map[string]interface{})
	invalidData["self"] = invalidData

	options := &PreprocessRunBody{
		ID:      "run-123",
		Payload: invalidData,
	}

	result, err := client.PreprocessRunRequest(context.Background(), options)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to marshal request")
}

// TestClient_Integration_DeliverHttpSourceRequest 集成测试 DeliverHttpSourceRequest 方法
func TestClient_Integration_DeliverHttpSourceRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		assert.Equal(t, "DELIVER_HTTP_SOURCE_REQUEST", r.Header.Get("x-trigger-action"))

		// 验证自定义头部
		assert.Equal(t, "webhook-key-123", r.Header.Get("x-ts-key"))
		assert.Equal(t, "webhook-secret-456", r.Header.Get("x-ts-secret"))
		assert.Equal(t, "https://api.example.com/webhook", r.Header.Get("x-ts-http-url"))
		assert.Equal(t, "POST", r.Header.Get("x-ts-http-method"))
		assert.Equal(t, "dynamic-789", r.Header.Get("x-ts-dynamic-id"))

		// 验证 JSON 头部
		assert.Contains(t, r.Header.Get("x-ts-params"), "webhookId")
		assert.Contains(t, r.Header.Get("x-ts-data"), "eventType")
		assert.Contains(t, r.Header.Get("x-ts-http-headers"), "Content-Type")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"success": true,
			"data": {
				"processed": true,
				"responseId": "resp-123"
			}
		}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	options := &DeliverHttpSourceRequestOptions{
		Key:       "webhook-key-123",
		DynamicID: "dynamic-789",
		Secret:    "webhook-secret-456",
		Params:    map[string]interface{}{"webhookId": "hook-123"},
		Data:      map[string]interface{}{"eventType": "push", "payload": "test data"},
		Request: HttpSourceRequest{
			URL:     "https://api.example.com/webhook",
			Method:  "POST",
			Headers: map[string]string{"Content-Type": "application/json"},
			RawBody: []byte(`{"event": "test"}`),
		},
	}

	result, err := client.DeliverHttpSourceRequest(context.Background(), options)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Data)
}

// TestClient_Integration_DeliverHttpSourceRequest_NoDynamicID 测试没有 DynamicID 的情况
func TestClient_Integration_DeliverHttpSourceRequest_NoDynamicID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证没有 x-ts-dynamic-id 头部
		assert.Empty(t, r.Header.Get("x-ts-dynamic-id"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	logger := &MockLogger{}
	client := NewClient("test-key", server.URL, "test-endpoint", logger)

	options := &DeliverHttpSourceRequestOptions{
		Key:    "webhook-key-123",
		Secret: "webhook-secret-456",
		Params: map[string]interface{}{},
		Data:   map[string]interface{}{},
		Request: HttpSourceRequest{
			URL:     "https://api.example.com/webhook",
			Method:  "GET",
			Headers: map[string]string{},
			RawBody: []byte{},
		},
	}

	result, err := client.DeliverHttpSourceRequest(context.Background(), options)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}
