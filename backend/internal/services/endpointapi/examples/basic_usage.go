package main

import (
	"context"
	"log"
	"time"

	"kongflow/backend/internal/services/endpointapi"
)

// SimpleLogger 实现 Logger 接口的简单日志记录器
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, fields map[string]interface{}) {
	log.Printf("DEBUG: %s %+v", msg, fields)
}

func (l *SimpleLogger) Error(msg string, fields map[string]interface{}) {
	log.Printf("ERROR: %s %+v", msg, fields)
}

func main() {
	logger := &SimpleLogger{}

	// 创建 EndpointApi 客户端
	client := endpointapi.NewClient(
		"your-api-key",              // 替换为实际的 API 密钥
		"https://your-endpoint.com", // 替换为实际的端点 URL
		"your-endpoint-id",          // 替换为实际的端点 ID
		logger,
	)

	log.Println("EndpointApi 客户端创建成功")

	// 1. 检测连接
	log.Println("\n=== 测试连接 ===")
	ctx := context.Background()

	pong, err := client.Ping(ctx)
	if err != nil {
		log.Fatalf("Ping 请求失败: %v", err)
	}

	if pong.OK {
		log.Println("✅ 端点连接成功")
	} else {
		log.Printf("❌ 端点连接失败: %s", pong.Error)
		return
	}

	// 2. 获取端点索引
	log.Println("\n=== 获取端点索引 ===")

	indexResp, err := client.IndexEndpoint(ctx)
	if err != nil {
		log.Printf("❌ 获取端点索引失败: %v", err)
	} else {
		log.Printf("✅ 端点索引获取成功")
		log.Printf("   - 作业数量: %d", len(indexResp.Jobs))
		log.Printf("   - 源数量: %d", len(indexResp.Sources))

		for i, job := range indexResp.Jobs {
			log.Printf("   - 作业 %d: ID=%s, Name=%s", i+1, job.ID, job.Name)
		}
	}

	// 3. 投递事件
	log.Println("\n=== 投递事件 ===")

	event := &endpointapi.ApiEventLog{
		ID:        "event-123",
		Name:      "test.event",
		Payload:   map[string]interface{}{"message": "Hello from Go!"},
		Context:   map[string]interface{}{"source": "endpointapi-example"},
		Timestamp: time.Now(),
		IsTest:    true,
	}

	deliverResp, err := client.DeliverEvent(ctx, event)
	if err != nil {
		log.Printf("❌ 事件投递失败: %v", err)
	} else {
		log.Printf("✅ 事件投递成功: Success=%t", deliverResp.Success)
		if deliverResp.Message != "" {
			log.Printf("   响应消息: %s", deliverResp.Message)
		}
	}

	// 4. 执行作业请求（仅作示例）
	log.Println("\n=== 执行作业请求 ===")

	jobBody := &endpointapi.RunJobBody{
		ID:      "job-123",
		Payload: map[string]interface{}{"data": "test-payload"},
		Context: map[string]interface{}{"user": "example-user"},
		JobRun:  map[string]interface{}{"attempt": 1},
	}

	jobResult, err := client.ExecuteJobRequest(ctx, jobBody)
	if err != nil {
		log.Printf("❌ 作业执行请求失败: %v", err)
	} else {
		log.Printf("✅ 作业执行请求发送成功")

		// 解析响应
		var jobResponse endpointapi.RunJobResponse
		err = jobResult.Parse(&jobResponse)
		if err != nil {
			log.Printf("❌ 作业响应解析失败: %v", err)
		} else {
			log.Printf("   作业ID: %s", jobResponse.ID)
			log.Printf("   状态: %s", jobResponse.Status)
		}
	}

	// 5. 初始化触发器
	log.Println("\n=== 初始化触发器 ===")

	triggerParams := map[string]interface{}{
		"type":     "webhook",
		"endpoint": "https://example.com/webhook",
		"events":   []string{"user.created", "user.updated"},
	}

	triggerResp, err := client.InitializeTrigger(ctx, "trigger-123", triggerParams)
	if err != nil {
		// 检查是否是 EndpointApiError
		if endpointErr, ok := err.(*endpointapi.EndpointApiError); ok {
			log.Printf("❌ 触发器初始化失败 (EndpointApiError): %s", endpointErr.Message)
			if endpointErr.Stack() != "" {
				log.Printf("   堆栈信息: %s", endpointErr.Stack())
			}
		} else {
			log.Printf("❌ 触发器初始化失败: %v", err)
		}
	} else {
		log.Printf("✅ 触发器初始化成功")
		log.Printf("   触发器ID: %s", triggerResp.ID)
	}

	log.Println("\n=== 示例完成 ===")
}
