package endpointapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NewClient 创建新的 EndpointApi 客户端
func NewClient(apiKey, url, endpointID string, logger Logger) *Client {
	return &Client{
		apiKey:     apiKey,
		url:        url,
		endpointID: endpointID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// NewClientWithHTTPClient 创建带自定义 HTTP 客户端的 EndpointApi 客户端
func NewClientWithHTTPClient(apiKey, url, endpointID string, httpClient HTTPClient, logger Logger) *Client {
	return &Client{
		apiKey:     apiKey,
		url:        url,
		endpointID: endpointID,
		httpClient: httpClient,
		logger:     logger,
	}
}

// buildRequest 构建 HTTP 请求
func (c *Client) buildRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.url
	if path != "" {
		url = fmt.Sprintf("%s/%s", strings.TrimRight(c.url, "/"), strings.TrimLeft(path, "/"))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置标准头部
	req.Header.Set("x-trigger-api-key", c.apiKey)

	return req, nil
}

// safeFetch 安全执行 HTTP 请求，对齐 trigger.dev 的 safeFetch 行为
func (c *Client) safeFetch(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("Error while trying to connect to endpoint", map[string]interface{}{
			"url":   req.URL.String(),
			"error": err.Error(),
		})
		return nil, err
	}
	return resp, nil
}

// Ping 检测端点连接 (对齐 trigger.dev ping 方法)
func (c *Client) Ping(ctx context.Context) (*PongResponse, error) {
	req, err := c.buildRequest(ctx, "POST", "", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-trigger-endpoint-id", c.endpointID)
	req.Header.Set("x-trigger-action", "PING")

	resp, err := c.safeFetch(req)
	if err != nil {
		return &PongResponse{
			OK:    false,
			Error: fmt.Sprintf("Could not connect to endpoint %s", c.url),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return &PongResponse{
			OK:    false,
			Error: "Trigger API key is invalid",
		}, nil
	}

	if resp.StatusCode != 200 {
		return &PongResponse{
			OK:    false,
			Error: fmt.Sprintf("Could not connect to endpoint %s. Status code: %d", c.url, resp.StatusCode),
		}, nil
	}

	var result PongResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("ping() response from endpoint", map[string]interface{}{
		"body": result,
	})

	return &result, nil
}

// IndexEndpoint 获取端点索引信息 (对齐 trigger.dev indexEndpoint 方法)
func (c *Client) IndexEndpoint(ctx context.Context) (*IndexEndpointResponse, error) {
	req, err := c.buildRequest(ctx, "POST", "", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-trigger-action", "INDEX_ENDPOINT")

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to endpoint %s", c.url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not connect to endpoint %s. Status code: %d", c.url, resp.StatusCode)
	}

	var result IndexEndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("indexEndpoint() response from endpoint", map[string]interface{}{
		"body": result,
	})

	return &result, nil
}

// DeliverEvent 投递事件 (对齐 trigger.dev deliverEvent 方法)
func (c *Client) DeliverEvent(ctx context.Context, event *ApiEventLog) (*DeliverEventResponse, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := c.buildRequest(ctx, "POST", "", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-trigger-action", "DELIVER_EVENT")

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to endpoint %s", c.url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not connect to endpoint %s. Status code: %d", c.url, resp.StatusCode)
	}

	var result DeliverEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("deliverEvent() response from endpoint", map[string]interface{}{
		"body": result,
	})

	return &result, nil
}

// ExecuteJobRequest 执行作业请求 (对齐 trigger.dev executeJobRequest 方法)
func (c *Client) ExecuteJobRequest(ctx context.Context, options *RunJobBody) (*JobExecutionResult, error) {
	body, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.buildRequest(ctx, "POST", "", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-trigger-action", "EXECUTE_JOB")

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, err
	}

	return &JobExecutionResult{
		Response: resp,
		Parse: func(target interface{}) error {
			defer resp.Body.Close()
			return json.NewDecoder(resp.Body).Decode(target)
		},
	}, nil
}

// PreprocessRunRequest 预处理运行请求 (对齐 trigger.dev preprocessRunRequest 方法)
func (c *Client) PreprocessRunRequest(ctx context.Context, options *PreprocessRunBody) (*PreprocessRunResult, error) {
	body, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.buildRequest(ctx, "POST", "", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-trigger-action", "PREPROCESS_RUN")

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, err
	}

	return &PreprocessRunResult{
		Response: resp,
		Parse: func(target interface{}) error {
			defer resp.Body.Close()
			return json.NewDecoder(resp.Body).Decode(target)
		},
	}, nil
}

// InitializeTrigger 初始化触发器 (对齐 trigger.dev initializeTrigger 方法)
func (c *Client) InitializeTrigger(ctx context.Context, id string, params map[string]interface{}) (*RegisterTriggerBody, error) {
	requestBody := map[string]interface{}{
		"id":     id,
		"params": params,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.buildRequest(ctx, "POST", "", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-trigger-action", "INITIALIZE_TRIGGER")

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to endpoint %s", c.url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// 尝试解析错误信息
		var errorResp ErrorWithStack
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return nil, NewEndpointApiError(errorResp.Message, errorResp.Stack)
		}

		return nil, fmt.Errorf("could not connect to endpoint %s. Status code: %d", c.url, resp.StatusCode)
	}

	var result RegisterTriggerBody
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("initializeTrigger() response from endpoint", map[string]interface{}{
		"body": result,
	})

	return &result, nil
}

// DeliverHttpSourceRequest 投递 HTTP 源请求 (对齐 trigger.dev deliverHttpSourceRequest 方法)
func (c *Client) DeliverHttpSourceRequest(ctx context.Context, options *DeliverHttpSourceRequestOptions) (*HttpSourceResponse, error) {
	req, err := c.buildRequest(ctx, "POST", "", bytes.NewReader(options.Request.RawBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-trigger-action", "DELIVER_HTTP_SOURCE_REQUEST")
	req.Header.Set("x-ts-key", options.Key)
	req.Header.Set("x-ts-secret", options.Secret)

	// 序列化参数和数据为 JSON 字符串
	paramsJSON, _ := json.Marshal(options.Params)
	dataJSON, _ := json.Marshal(options.Data)
	headersJSON, _ := json.Marshal(options.Request.Headers)

	req.Header.Set("x-ts-params", string(paramsJSON))
	req.Header.Set("x-ts-data", string(dataJSON))
	req.Header.Set("x-ts-http-url", options.Request.URL)
	req.Header.Set("x-ts-http-method", options.Request.Method)
	req.Header.Set("x-ts-http-headers", string(headersJSON))

	if options.DynamicID != "" {
		req.Header.Set("x-ts-dynamic-id", options.DynamicID)
	}

	resp, err := c.safeFetch(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to endpoint %s", c.url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not connect to endpoint %s. Status code: %d", c.url, resp.StatusCode)
	}

	var result HttpSourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("deliverHttpSourceRequest() response from endpoint", map[string]interface{}{
		"body": result,
	})

	return &result, nil
}
