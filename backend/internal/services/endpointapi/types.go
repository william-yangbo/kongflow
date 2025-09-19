package endpointapi

import (
	"net/http"
	"time"
)

// HTTPClient 接口允许测试时注入 mock
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Logger 接口集成现有日志系统
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// Client 对应 trigger.dev 的 EndpointApi 类
type Client struct {
	apiKey     string
	url        string
	endpointID string
	httpClient HTTPClient
	logger     Logger
}

// 请求/响应类型 (严格对齐 trigger.dev)

// PongResponse ping 方法的响应类型
type PongResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// JobMetadata 作业元数据
type JobMetadata struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Trigger       map[string]interface{} `json:"trigger"`
	StartRun      map[string]interface{} `json:"startRun,omitempty"`
	PreprocessRun map[string]interface{} `json:"preprocessRun,omitempty"`
	Examples      []interface{}          `json:"examples,omitempty"`
}

// SourceMetadata 源元数据
type SourceMetadata struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Version  string                 `json:"version"`
	Source   map[string]interface{} `json:"source"`
	Register map[string]interface{} `json:"register,omitempty"`
}

// IndexEndpointResponse indexEndpoint 方法的响应类型
type IndexEndpointResponse struct {
	Jobs            []JobMetadata    `json:"jobs"`
	Sources         []SourceMetadata `json:"sources"`
	DynamicTriggers []interface{}    `json:"dynamicTriggers,omitempty"`
}

// ApiEventLog 事件日志类型
type ApiEventLog struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Payload   map[string]interface{} `json:"payload"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source,omitempty"`
	IsTest    bool                   `json:"isTest,omitempty"`
}

// DeliverEventResponse deliverEvent 方法的响应类型
type DeliverEventResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// RunJobBody executeJobRequest 方法的请求体
type RunJobBody struct {
	ID      string                 `json:"id"`
	Payload map[string]interface{} `json:"payload"`
	Context map[string]interface{} `json:"context"`
	JobRun  map[string]interface{} `json:"jobRun"`
}

// RunJobResponse executeJobRequest 方法的响应类型
type RunJobResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// PreprocessRunBody preprocessRunRequest 方法的请求体
type PreprocessRunBody struct {
	ID      string                 `json:"id"`
	Payload map[string]interface{} `json:"payload"`
	Context map[string]interface{} `json:"context"`
}

// PreprocessRunResponse preprocessRunRequest 方法的响应类型
type PreprocessRunResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// RegisterTriggerBody initializeTrigger 方法的响应类型
type RegisterTriggerBody struct {
	ID     string                 `json:"id"`
	Params map[string]interface{} `json:"params"`
	Source map[string]interface{} `json:"source,omitempty"`
	Job    map[string]interface{} `json:"job,omitempty"`
}

// HttpSourceRequest HTTP 源请求类型
type HttpSourceRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	RawBody []byte            `json:"rawBody"`
}

// DeliverHttpSourceRequestOptions deliverHttpSourceRequest 方法的选项
type DeliverHttpSourceRequestOptions struct {
	Key       string                 `json:"key"`
	DynamicID string                 `json:"dynamicId,omitempty"`
	Secret    string                 `json:"secret"`
	Params    map[string]interface{} `json:"params"`
	Data      map[string]interface{} `json:"data"`
	Request   HttpSourceRequest      `json:"request"`
}

// HttpSourceResponse deliverHttpSourceRequest 方法的响应类型
type HttpSourceResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorWithStack 带堆栈的错误响应
type ErrorWithStack struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// JobExecutionResult executeJobRequest 方法的结果封装
type JobExecutionResult struct {
	Response *http.Response
	Parse    func(interface{}) error
}

// PreprocessRunResult preprocessRunRequest 方法的结果封装
type PreprocessRunResult struct {
	Response *http.Response
	Parse    func(interface{}) error
}
