# EndpointApi Service

EndpointApi Service 是一个专门用于与远程 endpoints 进行 HTTP 通信的客户端服务。它严格对齐 trigger.dev 的 `endpointApi.ts` 实现，提供标准化的通信接口。

## 🎯 功能概述

### 核心功能

- **连接检测**: `Ping()` - 检测端点连接状态
- **端点索引**: `IndexEndpoint()` - 获取端点的作业和源信息
- **事件投递**: `DeliverEvent()` - 投递事件到端点
- **作业执行**: `ExecuteJobRequest()` - 执行作业请求
- **运行预处理**: `PreprocessRunRequest()` - 预处理运行请求
- **触发器初始化**: `InitializeTrigger()` - 初始化触发器
- **HTTP 源请求投递**: `DeliverHttpSourceRequest()` - 投递 HTTP 源请求

### 技术特征

- **严格对齐**: 与 trigger.dev EndpointApi 行为完全一致
- **类型安全**: 完整的请求/响应类型定义
- **错误处理**: 专门的 EndpointApiError 错误类型
- **可测试**: 支持依赖注入的 HTTP 客户端和日志记录器
- **日志集成**: 调试友好的请求/响应日志

## 🚀 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"

    "kongflow/backend/internal/services/endpointapi"
)

// 假设你有一个实现了 Logger 接口的日志记录器
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, fields map[string]interface{}) {
    log.Printf("DEBUG: %s %v", msg, fields)
}

func (l *MyLogger) Error(msg string, fields map[string]interface{}) {
    log.Printf("ERROR: %s %v", msg, fields)
}

func main() {
    logger := &MyLogger{}

    // 创建客户端
    client := endpointapi.NewClient(
        "your-api-key",
        "https://your-endpoint.com",
        "your-endpoint-id",
        logger,
    )

    // 检测连接
    pong, err := client.Ping(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    if !pong.OK {
        log.Fatalf("Ping failed: %s", pong.Error)
    }

    log.Println("Endpoint is reachable!")
}
```

## 📚 API 文档

### 客户端创建

#### NewClient

```go
func NewClient(apiKey, url, endpointID string, logger Logger) *Client
```

创建新的 EndpointApi 客户端，使用默认的 HTTP 客户端。

**参数**:

- `apiKey`: Trigger API 密钥
- `url`: 端点 URL
- `endpointID`: 端点 ID
- `logger`: 日志记录器接口

#### NewClientWithHTTPClient

```go
func NewClientWithHTTPClient(apiKey, url, endpointID string, httpClient HTTPClient, logger Logger) *Client
```

创建带自定义 HTTP 客户端的 EndpointApi 客户端（主要用于测试）。

### 核心方法

#### Ping

```go
func (c *Client) Ping(ctx context.Context) (*PongResponse, error)
```

检测端点连接状态。

**返回**:

```go
type PongResponse struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
}
```

**示例**:

```go
pong, err := client.Ping(context.Background())
if err != nil {
    // 处理错误
}

if pong.OK {
    fmt.Println("端点可达")
} else {
    fmt.Printf("连接失败: %s", pong.Error)
}
```

#### IndexEndpoint

```go
func (c *Client) IndexEndpoint(ctx context.Context) (*IndexEndpointResponse, error)
```

获取端点的作业和源信息。

**返回**:

```go
type IndexEndpointResponse struct {
    Jobs            []JobMetadata    `json:"jobs"`
    Sources         []SourceMetadata `json:"sources"`
    DynamicTriggers []interface{}    `json:"dynamicTriggers,omitempty"`
}
```

#### DeliverEvent

```go
func (c *Client) DeliverEvent(ctx context.Context, event *ApiEventLog) (*DeliverEventResponse, error)
```

投递事件到端点。

**参数**:

```go
type ApiEventLog struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Payload   map[string]interface{} `json:"payload"`
    Context   map[string]interface{} `json:"context"`
    Timestamp time.Time              `json:"timestamp"`
    Source    string                 `json:"source,omitempty"`
    IsTest    bool                   `json:"isTest,omitempty"`
}
```

#### ExecuteJobRequest

```go
func (c *Client) ExecuteJobRequest(ctx context.Context, options *RunJobBody) (*JobExecutionResult, error)
```

执行作业请求。返回结果包含原始响应和解析函数。

**使用示例**:

```go
result, err := client.ExecuteJobRequest(ctx, jobBody)
if err != nil {
    // 处理错误
}

// 解析响应
var jobResponse RunJobResponse
err = result.Parse(&jobResponse)
if err != nil {
    // 处理解析错误
}
```

#### InitializeTrigger

```go
func (c *Client) InitializeTrigger(ctx context.Context, id string, params map[string]interface{}) (*RegisterTriggerBody, error)
```

初始化触发器。

#### DeliverHttpSourceRequest

```go
func (c *Client) DeliverHttpSourceRequest(ctx context.Context, options *DeliverHttpSourceRequestOptions) (*HttpSourceResponse, error)
```

投递 HTTP 源请求。

## 🔧 配置选项

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:    10,
        IdleConnTimeout: 90 * time.Second,
    },
}

client := endpointapi.NewClientWithHTTPClient(
    "api-key",
    "url",
    "endpoint-id",
    httpClient,
    logger,
)
```

## 🚨 错误处理

### EndpointApiError

特殊的错误类型，对应 trigger.dev 的 EndpointApiError：

```go
type EndpointApiError struct {
    Message string
    stack   string
}

func (e *EndpointApiError) Error() string
func (e *EndpointApiError) Stack() string
```

### 错误处理示例

```go
result, err := client.InitializeTrigger(ctx, "trigger-id", params)
if err != nil {
    if endpointErr, ok := err.(*endpointapi.EndpointApiError); ok {
        fmt.Printf("端点错误: %s\n", endpointErr.Message)
        fmt.Printf("堆栈: %s\n", endpointErr.Stack())
    } else {
        fmt.Printf("其他错误: %v\n", err)
    }
    return
}
```

## 🧪 测试

### 单元测试

使用 mock HTTP 客户端进行测试：

```go
type MockHTTPClient struct {
    mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    args := m.Called(req)
    return args.Get(0).(*http.Response), args.Error(1)
}

func TestExample(t *testing.T) {
    mockHTTP := &MockHTTPClient{}
    logger := &MockLogger{}

    client := endpointapi.NewClientWithHTTPClient(
        "test-key",
        "http://test.com",
        "test-endpoint",
        mockHTTP,
        logger,
    )

    // 设置 mock 响应
    resp := &http.Response{
        StatusCode: 200,
        Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
    }
    mockHTTP.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)

    // 执行测试
    result, err := client.Ping(context.Background())

    assert.NoError(t, err)
    assert.True(t, result.OK)
    mockHTTP.AssertExpectations(t)
}
```

### 运行测试

```bash
# 运行所有测试
go test ./internal/services/endpointapi

# 查看测试覆盖率
go test -cover ./internal/services/endpointapi

# 详细测试输出
go test -v ./internal/services/endpointapi
```

## 🔍 日志记录

所有 HTTP 请求和响应都会记录调试日志：

```go
c.logger.Debug("ping() response from endpoint", map[string]interface{}{
    "body": result,
})
```

错误情况也会记录：

```go
c.logger.Debug("Error while trying to connect to endpoint", map[string]interface{}{
    "url":   req.URL.String(),
    "error": err.Error(),
})
```

## 📋 与 trigger.dev 的对齐

### HTTP 头部对齐

- `x-trigger-api-key`: API 密钥认证
- `x-trigger-endpoint-id`: 端点标识
- `x-trigger-action`: 操作类型（PING, INDEX_ENDPOINT, 等）
- `x-ts-*`: HTTP 源请求的特殊头部

### 响应格式对齐

所有响应格式都严格对齐 trigger.dev 的 Schema 定义。

### 错误处理对齐

- 401 状态码 → "Trigger API key is invalid"
- 连接失败 → "Could not connect to endpoint {url}"
- 非 200 状态码 → "Could not connect to endpoint {url}. Status code: {code}"

## 🚀 性能考虑

- 默认 30 秒 HTTP 超时
- 支持自定义 HTTP 客户端配置
- 连接池复用
- 最小内存分配

## 🔄 版本兼容性

当前版本严格对齐 trigger.dev endpointApi.ts 的实现，确保 API 兼容性。

---

**最后更新**: 2025 年 9 月 19 日
**版本**: 1.0.0
**状态**: 生产就绪
