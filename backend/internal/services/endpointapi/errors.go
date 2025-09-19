package endpointapi

import "fmt"

// EndpointApiError 对应 trigger.dev 的 EndpointApiError
type EndpointApiError struct {
	Message string
	stack   string
}

func (e *EndpointApiError) Error() string {
	return fmt.Sprintf("EndpointApiError: %s", e.Message)
}

// Stack 返回错误堆栈
func (e *EndpointApiError) Stack() string {
	return e.stack
}

// NewEndpointApiError 创建带堆栈的错误
func NewEndpointApiError(message, stack string) *EndpointApiError {
	return &EndpointApiError{
		Message: message,
		stack:   stack,
	}
}

// 预定义错误类型
var (
	ErrConnectionFailed = &EndpointApiError{Message: "Could not connect to endpoint"}
	ErrUnauthorized     = &EndpointApiError{Message: "Trigger API key is invalid"}
	ErrEndpointError    = &EndpointApiError{Message: "Endpoint returned error"}
)
