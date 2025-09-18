// 优化建议：重构测试以减少冗余并明确职责

## 测试架构重构建议

### 1. 保留两个测试，但明确分工

#### email_consumer_integration_test.go (workerqueue 包)

- **职责**：专门测试 River 队列的内部机制
- **保留**：
  - Worker 配置和启动
  - 延迟任务调度 (RunAt)
  - 队列优先级处理
  - 错误重试机制
- **移除**：
  - 重复的多邮件测试
  - 基础入队出队测试

#### workerqueue_real_test.go (email 包)

- **职责**：端到端集成测试
- **保留**：
  - Testcontainers 环境
  - 邮件模板处理
  - 性能和并发测试
  - 完整的 email 服务流程
- **优化**：
  - 简化基础测试场景
  - 专注高级功能验证

### 2. 创建共享测试工具

```go
// internal/services/workerqueue/testutil/shared.go
package testutil

// 共享的邮件任务创建函数
func CreateTestEmailJobs(count int) []workerqueue.ScheduleEmailArgs

// 共享的等待和验证逻辑
func WaitForJobCompletion(t *testing.T, expectedCount int, timeout time.Duration)
```

### 3. 测试分层建议

```
Level 1: Unit Tests (Mock)
├── workerqueue_test.go (existing mock tests)

Level 2: Integration Tests (Real components)
├── email_consumer_integration_test.go (专注 River 队列)
└── workerqueue_real_test.go (端到端 email 集成)

Level 3: E2E Tests (Full system)
└── e2e_email_workflow_test.go (可选，完整业务流程)
```

### 4. 具体重构步骤

1. **简化 email_consumer_integration_test.go**

   - 移除多邮件批量测试 (与 workerqueue_real_test 重复)
   - 专注测试延迟调度、优先级、重试等 River 特性

2. **增强 workerqueue_real_test.go**

   - 添加模板处理错误测试
   - 添加邮件发送失败重试测试
   - 保持 Testcontainers 的隔离性优势

3. **提取共享工具**
   - 创建 testutil 包共享测试数据生成
   - 统一等待和验证逻辑

### 结论

两个测试各有价值，建议**保留但重构分工**，而不是删除其中一个。
