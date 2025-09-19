package endpointapi

import "context"

// EndpointAPIClient 定义端点 API 客户端接口
// 这个接口提供了对 endpointapi.Client 的抽象，便于单元测试和依赖注入
type EndpointAPIClient interface {
	// Ping 检测端点连接状态
	Ping(ctx context.Context) (*PongResponse, error)

	// IndexEndpoint 获取端点索引信息
	IndexEndpoint(ctx context.Context) (*IndexEndpointResponse, error)

	// DeliverEvent 投递事件到端点
	DeliverEvent(ctx context.Context, event *ApiEventLog) (*DeliverEventResponse, error)
}

// 确保 Client 实现了 EndpointAPIClient 接口
var _ EndpointAPIClient = (*Client)(nil)
