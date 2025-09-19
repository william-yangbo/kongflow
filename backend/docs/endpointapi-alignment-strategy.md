# EndpointApi Service 技术架构对比与对齐策略

## 📊 trigger.dev vs Go 架构对比

### TypeScript 原始实现分析

```typescript
// trigger.dev/apps/webapp/app/services/endpointApi.ts
export class EndpointApi {
  constructor(
    private apiKey: string,
    private url: string,
    private id: string
  ) {}

  // 7个核心方法的详细分析
}
```

### Go 语言等价实现

```go
// kongflow/backend/internal/services/endpointapi/client.go
type Client struct {
    apiKey     string  // 对应 TypeScript private apiKey
    url        string  // 对应 TypeScript private url
    endpointID string  // 对应 TypeScript private id
    // 增强字段
    httpClient HTTPClient // 可测试的HTTP客户端
    logger     Logger     // 集成日志系统
}
```

## 🔍 方法对齐详细分析

### 1. Ping 方法对齐

**trigger.dev 实现**:

```typescript
async ping(): Promise<PongResponse> {
  const response = await safeFetch(this.url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-trigger-api-key": this.apiKey,
      "x-trigger-endpoint-id": this.id,
      "x-trigger-action": "PING",
    },
  });

  if (!response) {
    return { ok: false, error: `Could not connect to endpoint ${this.url}` };
  }

  if (response.status === 401) {
    return { ok: false, error: `Trigger API key is invalid` };
  }

  if (!response.ok) {
    return { ok: false, error: `Could not connect to endpoint ${this.url}. Status code: ${response.status}` };
  }

  const anyBody = await response.json();
  logger.debug("ping() response from endpoint", { body: anyBody });
  return PongResponseSchema.parse(anyBody);
}
```

**Go 等价实现**:

```go
func (c *Client) Ping(ctx context.Context) (*PongResponse, error) {
    req, err := c.buildRequest(ctx, "POST", "", nil)
    if err != nil {
        return nil, err
    }

    // 完全对齐的头部设置
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-trigger-api-key", c.apiKey)
    req.Header.Set("x-trigger-endpoint-id", c.endpointID)
    req.Header.Set("x-trigger-action", "PING")

    resp, err := c.safeFetch(req)
    if err != nil {
        // 对齐 trigger.dev 的错误响应格式
        return &PongResponse{
            OK:    false,
            Error: fmt.Sprintf("Could not connect to endpoint %s", c.url),
        }, nil
    }
    defer resp.Body.Close()

    // 完全对齐的状态码处理
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

    // 对齐的日志输出
    c.logger.Debug("ping() response from endpoint", map[string]interface{}{
        "body": result,
    })

    return &result, nil
}
```

### 2. IndexEndpoint 方法对齐

**trigger.dev 实现**:

```typescript
async indexEndpoint() {
  const response = await safeFetch(this.url, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "x-trigger-api-key": this.apiKey,
      "x-trigger-action": "INDEX_ENDPOINT",
    },
  });

  if (!response) {
    throw new Error(`Could not connect to endpoint ${this.url}`);
  }

  if (!response.ok) {
    throw new Error(`Could not connect to endpoint ${this.url}. Status code: ${response.status}`);
  }

  const anyBody = await response.json();
  logger.debug("indexEndpoint() response from endpoint", { body: anyBody });
  return IndexEndpointResponseSchema.parse(anyBody);
}
```

**Go 等价实现**:

```go
func (c *Client) IndexEndpoint(ctx context.Context) (*IndexEndpointResponse, error) {
    req, err := c.buildRequest(ctx, "POST", "", nil)
    if err != nil {
        return nil, err
    }

    // 对齐的头部设置
    req.Header.Set("Accept", "application/json")
    req.Header.Set("x-trigger-api-key", c.apiKey)
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
```

## 📋 类型系统对齐策略

### Schema 验证对齐

**trigger.dev 使用 Zod Schema**:

```typescript
import {
  PongResponseSchema,
  IndexEndpointResponseSchema,
  DeliverEventResponseSchema,
  RunJobResponseSchema,
  // ...
} from '@trigger.dev/internal';

return PongResponseSchema.parse(anyBody);
```

**Go 使用 struct tags + 验证**:

```go
type PongResponse struct {
    OK    bool   `json:"ok" validate:"required"`
    Error string `json:"error,omitempty"`
}

// 可选：添加验证逻辑
func (p *PongResponse) Validate() error {
    if p.OK && p.Error != "" {
        return fmt.Errorf("response cannot be OK with error message")
    }
    return nil
}
```

### 复杂类型对齐

**trigger.dev ApiEventLog**:

```typescript
interface ApiEventLog {
  id: string;
  name: string;
  payload: any;
  context: any;
  timestamp: Date;
  // ...
}
```

**Go 等价类型**:

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

## 🔧 HTTP 客户端行为对齐

### safeFetch 函数对齐

**trigger.dev 实现**:

```typescript
async function safeFetch(url: string, options: RequestInit) {
  try {
    return await fetch(url, options);
  } catch (error) {
    logger.debug('Error while trying to connect to endpoint', { url });
  }
}
```

**Go 等价实现**:

```go
func (c *Client) safeFetch(req *http.Request) (*http.Response, error) {
    resp, err := c.httpClient.Do(req)
    if err != nil {
        c.logger.Debug("Error while trying to connect to endpoint", map[string]interface{}{
            "url": req.URL.String(),
        })
        return nil, err
    }
    return resp, nil
}
```

## 🎯 特殊方法对齐策略

### ExecuteJobRequest 返回格式对齐

**trigger.dev 实现**:

```typescript
async executeJobRequest(options: RunJobBody) {
  const response = await safeFetch(this.url, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-trigger-api-key": this.apiKey,
      "x-trigger-action": "EXECUTE_JOB",
    },
    body: JSON.stringify(options),
  });

  return {
    response,
    parser: RunJobResponseSchema,
  };
}
```

**Go 对齐实现**:

```go
type JobExecutionResult struct {
    Response *http.Response
    Parse    func(interface{}) error
}

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
    req.Header.Set("x-trigger-api-key", c.apiKey)
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
```

## 🛡️ 错误处理完全对齐

### EndpointApiError 对齐

**trigger.dev 实现**:

```typescript
export class EndpointApiError extends Error {
  constructor(message: string, stack?: string) {
    super(`EndpointApiError: ${message}`);
    this.stack = stack;
    this.name = 'EndpointApiError';
  }
}
```

**Go 等价实现**:

```go
type EndpointApiError struct {
    message string
    stack   string
}

func (e *EndpointApiError) Error() string {
    return fmt.Sprintf("EndpointApiError: %s", e.message)
}

func (e *EndpointApiError) Stack() string {
    return e.stack
}

func NewEndpointApiError(message, stack string) *EndpointApiError {
    return &EndpointApiError{
        message: message,
        stack:   stack,
    }
}

// 实现错误包装接口
func (e *EndpointApiError) Unwrap() error {
    return nil
}
```

## 📊 配置选项对齐

### HTTP 客户端配置

**trigger.dev 隐式配置**:

```typescript
// fetch API 使用默认配置
const response = await fetch(url, options);
```

**Go 显式配置对齐**:

```go
type ClientConfig struct {
    Timeout         time.Duration `default:"30s"`
    MaxRetries      int           `default:"3"`
    RetryDelay      time.Duration `default:"1s"`
    MaxIdleConns    int           `default:"10"`
    IdleConnTimeout time.Duration `default:"90s"`
}

func NewClientWithConfig(apiKey, url, endpointID string, config ClientConfig, logger Logger) *Client {
    httpClient := &http.Client{
        Timeout: config.Timeout,
        Transport: &http.Transport{
            MaxIdleConns:    config.MaxIdleConns,
            IdleConnTimeout: config.IdleConnTimeout,
        },
    }

    return &Client{
        apiKey:     apiKey,
        url:        url,
        endpointID: endpointID,
        httpClient: httpClient,
        logger:     logger,
        config:     config,
    }
}
```

## 🧪 测试对齐策略

### Mock 策略对齐

**trigger.dev 风格测试**:

```typescript
// 使用 jest.mock 或类似工具
jest.mock('./logger', () => ({
  debug: jest.fn(),
}));
```

**Go 等价测试策略**:

```go
type MockHTTPClient struct {
    mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    args := m.Called(req)
    return args.Get(0).(*http.Response), args.Error(1)
}

type MockLogger struct {
    DebugCalls []LogCall
}

type LogCall struct {
    Message string
    Fields  map[string]interface{}
}

func (l *MockLogger) Debug(msg string, fields map[string]interface{}) {
    l.DebugCalls = append(l.DebugCalls, LogCall{
        Message: msg,
        Fields:  fields,
    })
}
```

## 📈 性能对齐目标

### 响应时间对齐

- **trigger.dev**: fetch API 性能基线
- **Go 目标**: 95% 情况下性能不低于 TypeScript 版本
- **测量方法**: 相同网络条件下的基准测试

### 内存使用对齐

- **trigger.dev**: JavaScript 运行时内存使用
- **Go 目标**: 更低的内存占用 (Go 的优势)
- **测量方法**: 大量并发请求的内存分析

## 🔄 部署和集成对齐

### 环境变量对齐

```go
// 与 trigger.dev 相同的环境变量命名
type Config struct {
    APIKey     string `env:"TRIGGER_API_KEY"`
    BaseURL    string `env:"TRIGGER_BASE_URL"`
    EndpointID string `env:"TRIGGER_ENDPOINT_ID"`
    Timeout    string `env:"TRIGGER_TIMEOUT" default:"30s"`
}
```

### 日志格式对齐

```go
// 确保日志输出格式与 trigger.dev 一致
c.logger.Debug("ping() response from endpoint", map[string]interface{}{
    "body":        result,
    "endpoint_id": c.endpointID,
    "url":         c.url,
    "status":      "success",
})
```

## ✅ 对齐验证清单

### 功能对齐验证

- [ ] 所有 7 个方法行为完全一致
- [ ] HTTP 请求头部格式完全对齐
- [ ] 错误响应格式完全对齐
- [ ] 日志输出格式完全对齐

### 类型对齐验证

- [ ] 请求类型与 trigger.dev 一致
- [ ] 响应类型与 trigger.dev 一致
- [ ] 错误类型与 trigger.dev 一致

### 性能对齐验证

- [ ] 响应时间不超过 trigger.dev 110%
- [ ] 内存使用合理
- [ ] 并发性能符合预期

---

**这个对齐策略确保了 Go 版本与 trigger.dev 的 EndpointApi 保持最大程度的一致性，同时充分利用 Go 语言的优势。**
