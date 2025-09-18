package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"kongflow/backend/internal/services/workerqueue"
)

// 示例 payload 结构
type PerformRunExecutionArgs struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
}

type UserTaskArgs struct {
	UserID   string `json:"userId"`
	UserPlan string `json:"userPlan"`
	Region   string `json:"region"`
	TaskData string `json:"taskData"`
}

// demonstrateDynamicQueueConfiguration 展示如何使用动态队列配置
func demonstrateDynamicQueueConfiguration() {
	logger := slog.Default()

	// 1. 静态队列配置（向后兼容）
	staticTaskCatalog := workerqueue.TaskCatalog{
		"legacyTask": workerqueue.TaskDefinition{
			QueueName:   workerqueue.StaticQueueName("legacy-queue"), // ✅ 静态队列名称
			Priority:    1,
			MaxAttempts: 3,
			Handler:     handleLegacyTask,
		},
	}

	// 2. 动态队列配置（新功能）
	dynamicTaskCatalog := workerqueue.TaskCatalog{
		// 按运行 ID 分配队列 - 实现运行级别隔离
		"performRunExecution": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				if runArgs, ok := payload.(PerformRunExecutionArgs); ok {
					return fmt.Sprintf("runs:%s", runArgs.ID) // 动态生成：runs:run-123
				}
				return "default-runs" // 回退队列
			}),
			MaxAttempts: 1,
			Handler:     handlePerformRunExecution,
		},

		// 按项目 ID 分配队列 - 实现项目级别隔离
		"startQueuedRuns": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				if runArgs, ok := payload.(PerformRunExecutionArgs); ok {
					return fmt.Sprintf("project:%s:runs", runArgs.ProjectID) // 动态生成：project:proj-123:runs
				}
				return "default-project-runs"
			}),
			MaxAttempts: 3,
			Handler:     handleStartQueuedRuns,
		},

		// 按用户等级和地理位置分配队列 - 实现多维度路由
		"processUserTask": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				if userTask, ok := payload.(UserTaskArgs); ok {
					// 复杂的路由逻辑
					switch userTask.UserPlan {
					case "enterprise":
						return fmt.Sprintf("enterprise:%s:high-priority", userTask.Region)
					case "pro":
						return fmt.Sprintf("pro:%s:medium-priority", userTask.Region)
					default:
						return fmt.Sprintf("standard:%s:normal-priority", userTask.Region)
					}
				}
				return "default-user-tasks"
			}),
			MaxAttempts: 5,
			Handler:     handleUserTask,
		},
	}

	// 3. 混合配置（静态 + 动态）
	mixedTaskCatalog := workerqueue.TaskCatalog{
		// 静态队列：适用于简单场景
		"indexEndpoint": workerqueue.TaskDefinition{
			QueueName:   workerqueue.StaticQueueName("internal-queue"),
			MaxAttempts: 7,
			Handler:     handleIndexEndpoint,
		},

		// 动态队列：适用于复杂业务逻辑
		"smartTaskRouting": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				// 可以根据任何条件动态决定队列
				payloadBytes, _ := json.Marshal(payload)
				payloadStr := string(payloadBytes)

				// 示例：根据 payload 大小选择队列
				if len(payloadStr) > 1000 {
					return "large-payload-queue"
				} else if len(payloadStr) > 100 {
					return "medium-payload-queue"
				}
				return "small-payload-queue"
			}),
			MaxAttempts: 3,
			Handler:     handleSmartTask,
		},
	}

	// 输出示例配置
	logger.Info("Static Task Catalog", "tasks", len(staticTaskCatalog))
	logger.Info("Dynamic Task Catalog", "tasks", len(dynamicTaskCatalog))
	logger.Info("Mixed Task Catalog", "tasks", len(mixedTaskCatalog))

	// 4. 演示队列名称解析
	demonstrateQueueResolution(dynamicTaskCatalog, logger)
}

// demonstrateQueueResolution 演示队列名称解析过程
func demonstrateQueueResolution(catalog workerqueue.TaskCatalog, logger *slog.Logger) {
	logger.Info("=== 动态队列名称解析演示 ===")

	// 测试运行执行任务
	runPayload := PerformRunExecutionArgs{
		ID:        "run-12345",
		ProjectID: "proj-abc",
	}

	if taskDef, exists := catalog["performRunExecution"]; exists {
		queueName := taskDef.QueueName.ResolveQueueName(runPayload)
		logger.Info("Run Execution Task",
			"payload", runPayload,
			"resolvedQueue", queueName, // 输出：runs:run-12345
		)
	}

	// 测试用户任务路由
	userPayloads := []UserTaskArgs{
		{UserID: "user-1", UserPlan: "enterprise", Region: "us-east-1", TaskData: "important"},
		{UserID: "user-2", UserPlan: "pro", Region: "eu-west-1", TaskData: "medium"},
		{UserID: "user-3", UserPlan: "free", Region: "ap-southeast-1", TaskData: "basic"},
	}

	if taskDef, exists := catalog["processUserTask"]; exists {
		for _, userPayload := range userPayloads {
			queueName := taskDef.QueueName.ResolveQueueName(userPayload)
			logger.Info("User Task Routing",
				"user", userPayload.UserID,
				"plan", userPayload.UserPlan,
				"region", userPayload.Region,
				"resolvedQueue", queueName,
			)
		}
	}
}

// 示例处理器函数
func handleLegacyTask(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Println("Processing legacy task in static queue:", job.Queue)
	return nil
}

func handlePerformRunExecution(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing run execution in queue: %s\n", job.Queue)
	return nil
}

func handleStartQueuedRuns(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Starting queued runs in queue: %s\n", job.Queue)
	return nil
}

func handleUserTask(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing user task in queue: %s\n", job.Queue)
	return nil
}

func handleIndexEndpoint(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Indexing endpoint in queue: %s\n", job.Queue)
	return nil
}

func handleSmartTask(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing smart task in queue: %s\n", job.Queue)
	return nil
}

// 业务场景示例
func businessScenarioExamples() {
	logger := slog.Default()
	logger.Info("=== 业务场景示例 ===")

	// 场景 1: 多租户 SaaS 应用
	multiTenantCatalog := workerqueue.TaskCatalog{
		"processCustomerData": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				// 根据客户等级分配资源
				if customerData, ok := payload.(map[string]interface{}); ok {
					if tier, ok := customerData["tier"].(string); ok {
						switch tier {
						case "enterprise":
							return "enterprise-customers" // 专用高性能队列
						case "business":
							return "business-customers" // 中等性能队列
						default:
							return "standard-customers" // 标准队列
						}
					}
				}
				return "default-customers"
			}),
			MaxAttempts: 5,
			Handler:     handleCustomerData,
		},
	}

	// 场景 2: 地理位置数据合规
	geoComplianceCatalog := workerqueue.TaskCatalog{
		"processGDPRData": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				// 确保数据在合规区域处理
				if dataReq, ok := payload.(map[string]interface{}); ok {
					if region, ok := dataReq["region"].(string); ok {
						return fmt.Sprintf("gdpr:%s:data-processing", region)
					}
				}
				return "gdpr:default:data-processing"
			}),
			MaxAttempts: 3,
			Handler:     handleGDPRData,
		},
	}

	// 场景 3: 工作负载优先级
	workloadPriorityCatalog := workerqueue.TaskCatalog{
		"processAnalytics": workerqueue.TaskDefinition{
			QueueName: workerqueue.DynamicQueueName(func(payload interface{}) string {
				// 根据数据量和紧急程度分配队列
				if analytics, ok := payload.(map[string]interface{}); ok {
					urgent := analytics["urgent"].(bool)
					dataSize := analytics["dataSize"].(int)

					if urgent {
						return "analytics:urgent:high-priority"
					} else if dataSize > 10000 {
						return "analytics:large:batch-processing"
					}
					return "analytics:standard:normal"
				}
				return "analytics:default"
			}),
			MaxAttempts: 3,
			Handler:     handleAnalytics,
		},
	}

	logger.Info("Multi-tenant catalog", "tasks", len(multiTenantCatalog))
	logger.Info("Geo-compliance catalog", "tasks", len(geoComplianceCatalog))
	logger.Info("Workload priority catalog", "tasks", len(workloadPriorityCatalog))
}

// 示例处理器
func handleCustomerData(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing customer data in queue: %s\n", job.Queue)
	return nil
}

func handleGDPRData(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing GDPR data in queue: %s\n", job.Queue)
	return nil
}

func handleAnalytics(ctx context.Context, payload json.RawMessage, job workerqueue.JobContext) error {
	fmt.Printf("Processing analytics in queue: %s\n", job.Queue)
	return nil
}

func main() {
	fmt.Println("🚀 KongFlow 动态队列配置示例")
	fmt.Println("=====================================")

	// 基础功能演示
	demonstrateDynamicQueueConfiguration()

	fmt.Println("\n=====================================")

	// 业务场景示例
	businessScenarioExamples()

	fmt.Println("\n✅ 动态队列配置 Phase 1 实施完成!")
}
