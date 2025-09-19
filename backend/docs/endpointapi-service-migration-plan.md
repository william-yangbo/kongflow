# EndpointApi Service 迁移计划

## 📋 项目概述

### 服务描述

EndpointApi Service 是一个专门用于与远程 endpoints 进行 HTTP 通信的客户端服务。它封装了所有与 trigger endpoints 的交互逻辑，提供标准化的通信接口。

### 对齐目标

严格对齐 trigger.dev 的 `endpointApi.ts` 实现，保持相同的 API 接口、错误处理模式和通信协议，同时适配 Go 语言最佳实践。

## 🎯 核心功能分析

### trigger.dev EndpointApi 核心功能

```typescript
class EndpointApi {
  // 1. 连接检测
  async ping(): Promise<PongResponse>

  // 2. 端点索引
  async indexEndpoint()

  // 3. 事件投递
  async deliverEvent(event: ApiEventLog)

  // 4. 作业执行
  async executeJobRequest(options: RunJobBody)

  // 5. 运行预处理
  async preprocessRunRequest(options: PreprocessRunBody)

  // 6. 触发器初始化
  async initializeTrigger(id: string, params: any)

  // 7. HTTP源请求投递
  async deliverHttpSourceRequest(options: {...})
}
```

### 技术特征

- **HTTP 客户端封装**: 标准化的 fetch 调用
- **认证机制**: API key + endpoint ID 头部认证
- **统一错误处理**: EndpointApiError + 状态码处理
- **Schema 验证**: 响应数据的类型安全验证
- **日志集成**: 调试友好的请求/响应日志

## 🏗️ Go 语言架构设计

### 目录结构

```
internal/services/endpointapi/
├── README.md                 # 服务文档
├── client.go                 # 核心客户端实现
├── types.go                  # 类型定义
├── errors.go                 # 错误类型
├── client_test.go            # 单元测试
├── integration_test.go       # 集成测试
└── examples/
    └── basic_usage.go        # 使用示例
```

### 核心类型定义

```go
// types.go
package endpointapi

import (
    "context"
    "time"
)

// Client 对应 trigger.dev 的 EndpointApi 类
type Client struct {
    apiKey     string
    url        string
    endpointID string
    httpClient HTTPClient
    logger     Logger
}

// HTTPClient 接口允许测试时注入 mock
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Logger 接口集成现有日志系统
type Logger interface {
    Debug(msg string, fields map[string]interface{})
    Error(msg string, fields map[string]interface{})
}

// 请求/响应类型 (严格对齐 trigger.dev)
type PongResponse struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
}

type IndexEndpointResponse struct {
    Jobs     []JobMetadata     `json:"jobs"`
    Sources  []SourceMetadata  `json:"sources"`
    // ... 其他字段对齐 trigger.dev
}

type ApiEventLog struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Payload   map[string]interface{} `json:"payload"`
    Context   map[string]interface{} `json:"context"`
    Timestamp time.Time              `json:"timestamp"`
    // ... 其他字段
}
```

### 错误处理设计

```go
// errors.go
package endpointapi

import "fmt"

// EndpointApiError 对应 trigger.dev 的 EndpointApiError
type EndpointApiError struct {
    Message string
    Stack   string
}

func (e *EndpointApiError) Error() string {
    return fmt.Sprintf("EndpointApiError: %s", e.Message)
}

// 预定义错误类型
var (
    ErrConnectionFailed = &EndpointApiError{Message: "Could not connect to endpoint"}
    ErrUnauthorized     = &EndpointApiError{Message: "Trigger API key is invalid"}
    ErrEndpointError    = &EndpointApiError{Message: "Endpoint returned error"}
)

// NewEndpointApiError 创建带堆栈的错误
func NewEndpointApiError(message, stack string) *EndpointApiError {
    return &EndpointApiError{
        Message: message,
        Stack:   stack,
    }
}
```

### 核心客户端实现

```go
// client.go
package endpointapi

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
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

// Ping 检测端点连接 (对齐 trigger.dev ping 方法)
func (c *Client) Ping(ctx context.Context) (*PongResponse, error) {
    req, err := c.buildRequest(ctx, "POST", "", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("x-trigger-action", "PING")
    req.Header.Set("x-trigger-endpoint-id", c.endpointID)

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

// 其他方法实现...
```

## 📋 实施计划

### 阶段 1: 基础架构搭建 (1 天)

**任务清单**:

- [ ] 创建服务目录结构
- [ ] 实现 `types.go` - 核心类型定义
- [ ] 实现 `errors.go` - 错误处理类型
- [ ] 实现基础的 `Client` 结构

**验收标准**:

- [ ] 类型定义与 trigger.dev 严格对齐
- [ ] 错误类型完整覆盖
- [ ] 基础结构编译通过

### 阶段 2: 核心方法实现 (2 天)

**任务清单**:

- [ ] 实现 `Ping()` 方法
- [ ] 实现 `IndexEndpoint()` 方法
- [ ] 实现 `DeliverEvent()` 方法
- [ ] 实现 `ExecuteJobRequest()` 方法
- [ ] 实现 HTTP 请求辅助方法

**验收标准**:

- [ ] 所有方法与 trigger.dev 行为一致
- [ ] HTTP 头部设置正确
- [ ] 错误处理完善

### 阶段 3: 高级功能实现 (1 天)

**任务清单**:

- [ ] 实现 `PreprocessRunRequest()` 方法
- [ ] 实现 `InitializeTrigger()` 方法
- [ ] 实现 `DeliverHttpSourceRequest()` 方法
- [ ] 集成日志系统

**验收标准**:

- [ ] 复杂参数处理正确
- [ ] 特殊头部设置完整
- [ ] 日志输出格式对齐

### 阶段 4: 测试覆盖 (1 天)

**任务清单**:

- [ ] 编写单元测试 (使用 mock HTTP 客户端)
- [ ] 编写集成测试 (使用测试服务器)
- [ ] 错误场景测试
- [ ] 性能基准测试

**验收标准**:

- [ ] 测试覆盖率 > 85%
- [ ] 所有 HTTP 状态码场景覆盖
- [ ] Mock 测试验证正确性

## 🔧 技术实现细节

### HTTP 客户端配置

```go
type ClientOptions struct {
    Timeout         time.Duration
    RetryAttempts   int
    RetryDelay      time.Duration
    MaxIdleConns    int
    IdleConnTimeout time.Duration
}

func NewClientWithOptions(apiKey, url, endpointID string, opts ClientOptions, logger Logger) *Client {
    client := &http.Client{
        Timeout: opts.Timeout,
        Transport: &http.Transport{
            MaxIdleConns:    opts.MaxIdleConns,
            IdleConnTimeout: opts.IdleConnTimeout,
        },
    }

    return &Client{
        apiKey:     apiKey,
        url:        url,
        endpointID: endpointID,
        httpClient: client,
        logger:     logger,
        options:    opts,
    }
}
```

### 请求构建辅助方法

```go
func (c *Client) buildRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
    var bodyReader io.Reader

    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
        bodyReader = bytes.NewReader(jsonBody)
    }

    url := c.url
    if path != "" {
        url = fmt.Sprintf("%s/%s", strings.TrimRight(c.url, "/"), strings.TrimLeft(path, "/"))
    }

    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // 设置标准头部
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-trigger-api-key", c.apiKey)

    return req, nil
}

func (c *Client) safeFetch(req *http.Request) (*http.Response, error) {
    resp, err := c.httpClient.Do(req)
    if err != nil {
        c.logger.Debug("Error while trying to connect to endpoint", map[string]interface{}{
            "url": req.URL.String(),
            "error": err.Error(),
        })
        return nil, err
    }
    return resp, nil
}
```

## 🧪 测试策略

### 单元测试结构

```go
// client_test.go
package endpointapi

import (
    "context"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
    mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    args := m.Called(req)
    return args.Get(0).(*http.Response), args.Error(1)
}

type MockLogger struct{}

func (l *MockLogger) Debug(msg string, fields map[string]interface{}) {}
func (l *MockLogger) Error(msg string, fields map[string]interface{}) {}

func TestClient_Ping_Success(t *testing.T) {
    mockHTTP := &MockHTTPClient{}
    logger := &MockLogger{}

    client := &Client{
        apiKey:     "test-key",
        url:        "http://test.com",
        endpointID: "test-endpoint",
        httpClient: mockHTTP,
        logger:     logger,
    }

    // Mock 成功响应
    resp := &http.Response{
        StatusCode: 200,
        Body:       ioutil.NopCloser(strings.NewReader(`{"ok":true}`)),
    }
    mockHTTP.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)

    result, err := client.Ping(context.Background())

    assert.NoError(t, err)
    assert.True(t, result.OK)
    assert.Empty(t, result.Error)
    mockHTTP.AssertExpectations(t)
}
```

### 集成测试

```go
// integration_test.go
package endpointapi

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestClient_Integration_Ping(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "test-key", r.Header.Get("x-trigger-api-key"))
        assert.Equal(t, "test-endpoint", r.Header.Get("x-trigger-endpoint-id"))
        assert.Equal(t, "PING", r.Header.Get("x-trigger-action"))

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(200)
        w.Write([]byte(`{"ok":true}`))
    }))
    defer server.Close()

    client := NewClient("test-key", server.URL, "test-endpoint", &MockLogger{})

    result, err := client.Ping(context.Background())

    assert.NoError(t, err)
    assert.True(t, result.OK)
}
```

## 📊 质量保证

### 代码质量标准

- **对齐度**: 95%+ 与 trigger.dev 行为一致
- **测试覆盖**: 85%+ 代码覆盖率
- **性能**: HTTP 请求延迟 < 100ms (本地网络)
- **错误处理**: 所有 HTTP 状态码场景覆盖

### 验收测试清单

- [ ] **连接测试**: 成功连接到有效端点
- [ ] **认证测试**: API key 验证正确
- [ ] **错误处理**: 各种错误场景正确处理
- [ ] **数据格式**: 请求/响应格式与 trigger.dev 一致
- [ ] **日志输出**: 调试信息完整准确
- [ ] **超时处理**: 网络超时正确处理
- [ ] **并发安全**: 多 goroutine 并发调用安全

## 📚 文档要求

### README.md 内容

1. **服务概述**: 功能描述和使用场景
2. **快速开始**: 基本使用示例
3. **API 文档**: 所有方法的详细说明
4. **配置选项**: 客户端配置参数
5. **错误处理**: 错误类型和处理建议
6. **最佳实践**: 性能优化和使用建议

### 使用示例

```go
// examples/basic_usage.go
package main

import (
    "context"
    "log"

    "kongflow/backend/internal/services/endpointapi"
    "kongflow/backend/internal/services/logger"
)

func main() {
    logger := logger.NewWebapp("endpointapi-example")

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

## ⏰ 时间估算

| 阶段     | 预估时间               | 关键里程碑   |
| -------- | ---------------------- | ------------ |
| 阶段 1   | 8 小时                 | 基础架构完成 |
| 阶段 2   | 16 小时                | 核心方法实现 |
| 阶段 3   | 8 小时                 | 高级功能完成 |
| 阶段 4   | 8 小时                 | 测试覆盖完成 |
| **总计** | **40 小时 (5 工作日)** | **生产就绪** |

## 🔄 风险评估与缓解

### 潜在风险

1. **类型转换复杂性**: TypeScript → Go 类型映射
2. **HTTP 客户端差异**: fetch API → Go http 包
3. **错误处理模式**: JavaScript Promise → Go error

### 缓解策略

1. **严格类型定义**: 使用 struct tags 确保 JSON 序列化一致
2. **HTTP 适配器**: 封装 HTTP 客户端行为对齐 fetch API
3. **错误包装**: 实现 error wrapping 保持错误链

## 📈 成功指标

### 功能指标

- [ ] 7 个核心方法全部实现并通过测试
- [ ] 与 trigger.dev 的 API 兼容性 95%+
- [ ] 错误场景处理覆盖率 100%

### 质量指标

- [ ] 代码测试覆盖率 85%+
- [ ] 集成测试通过率 100%
- [ ] 性能测试满足要求

### 交付物指标

- [ ] 完整的服务实现代码
- [ ] 完善的测试套件
- [ ] 详细的文档和示例
- [ ] 迁移验证报告

---

**文档版本**: 1.0  
**创建时间**: 2025 年 9 月 19 日  
**最后更新**: 2025 年 9 月 19 日  
**状态**: 准备实施

**下一步行动**: 开始阶段 1 - 基础架构搭建
