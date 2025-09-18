# Analytics Service Migration Plan

## 📋 概述

将 trigger.dev 的 `analytics.server.ts` 服务迁移到 kongflow Go backend，提供用户行为分析和事件追踪功能。

## 🔍 Analysis of trigger.dev Implementation

### Source Code Analysis

**文件位置**: `trigger.dev/apps/webapp/app/services/analytics.server.ts`  
**核心类**: `BehaviouralAnalytics`  
**导出实例**: `analytics`

### Key Characteristics

1. **PostHog 集成**: 使用 PostHog 作为分析后端
2. **分组追踪**: 支持 user, organization, project, environment 分组
3. **事件捕获**: 结构化的事件数据收集
4. **条件初始化**: API Key 缺失时优雅降级
5. **单例模式**: 全局共享实例

### Core Functionality

#### 1. User Analytics

```typescript
user = {
  identify: ({ user, isNewUser }: { user: User; isNewUser: boolean }) => {
    // 用户身份识别和属性设置
    // 新用户注册事件追踪
  },
};
```

#### 2. Organization Analytics

```typescript
organization = {
  identify: ({ organization }: { organization: Organization }) => {
    // 组织分组识别
  },
  new: ({ userId, organization, organizationCount }) => {
    // 新组织创建事件
  },
};
```

#### 3. Project Analytics

```typescript
project = {
  identify: ({ project }: { project: Project }) => {
    // 项目分组识别
  },
  new: ({ userId, organizationId, project }) => {
    // 新项目创建事件
  },
};
```

#### 4. Environment Analytics

```typescript
environment = {
  identify: ({ environment }: { environment: RuntimeEnvironment }) => {
    // 环境分组识别
  },
};
```

#### 5. General Telemetry

```typescript
telemetry = {
  capture: ({ userId, event, properties, organizationId?, environmentId? }) => {
    // 通用事件捕获
  }
}
```

### Dependencies

- **PostHog SDK**: `posthog-node` 包
- **Environment Config**: `env.POSTHOG_PROJECT_KEY`
- **Type Definitions**:
  - `User` from `~/models/user.server`
  - `Organization` from `~/models/organization.server`
  - `Project` from `~/models/project.server`
  - `RuntimeEnvironment` from `~/models/runtimeEnvironment.server`

### Usage Patterns

1. **Singleton Access**: `analytics.user.identify(...)`
2. **Event Driven**: 响应用户/组织/项目生命周期事件
3. **Contextual Grouping**: 根据层级关系设置分组
4. **Safe Degradation**: PostHog 不可用时静默失败

## 🎯 Go Implementation Strategy

### 1. Architecture Design

```
internal/services/analytics/
├── analytics.go          # 核心服务实现
├── models.go            # 事件和数据结构
├── client.go            # PostHog 客户端封装
├── analytics_test.go    # 单元测试
└── example_test.go      # 使用示例
```

### 2. Dependency Mapping

| trigger.dev               | kongflow Go    | Purpose      |
| ------------------------- | -------------- | ------------ |
| `posthog-node`            | PostHog Go SDK | 分析客户端   |
| `env.POSTHOG_PROJECT_KEY` | 环境变量配置   | API Key 管理 |
| TypeScript 接口           | Go 结构体      | 类型定义     |

### 3. API Design

#### Core Service Interface

```go
type AnalyticsService interface {
    // User analytics
    UserIdentify(ctx context.Context, user *UserData, isNewUser bool) error

    // Organization analytics
    OrganizationIdentify(ctx context.Context, org *OrganizationData) error
    OrganizationNew(ctx context.Context, userID string, org *OrganizationData, count int) error

    // Project analytics
    ProjectIdentify(ctx context.Context, project *ProjectData) error
    ProjectNew(ctx context.Context, userID, orgID string, project *ProjectData) error

    // Environment analytics
    EnvironmentIdentify(ctx context.Context, env *EnvironmentData) error

    // General telemetry
    Capture(ctx context.Context, event *TelemetryEvent) error
}
```

#### Data Structures

```go
type UserData struct {
    ID                   string    `json:"id"`
    Email               string    `json:"email"`
    Name                string    `json:"name"`
    AuthenticationMethod string    `json:"authenticationMethod"`
    Admin               bool      `json:"admin"`
    CreatedAt           time.Time `json:"createdAt"`
}

type OrganizationData struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Slug      string    `json:"slug"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type TelemetryEvent struct {
    UserID         string                 `json:"userId"`
    Event          string                 `json:"event"`
    Properties     map[string]interface{} `json:"properties"`
    OrganizationID *string                `json:"organizationId,omitempty"`
    EnvironmentID  *string                `json:"environmentId,omitempty"`
}
```

## 🚀 Implementation Plan

### Phase 1: Foundation (Day 1-2)

- [ ] 创建服务包结构
- [ ] 设置 PostHog Go SDK 依赖
- [ ] 定义核心数据结构
- [ ] 实现基础客户端封装

### Phase 2: Core Features (Day 3-4)

- [ ] 实现用户分析功能
- [ ] 实现组织分析功能
- [ ] 实现项目分析功能
- [ ] 实现环境分析功能

### Phase 3: Integration (Day 5)

- [ ] 实现通用遥测功能
- [ ] 添加错误处理和重试机制
- [ ] 性能优化和批处理

### Phase 4: Testing & Validation (Day 6-7)

- [ ] 单元测试覆盖
- [ ] 集成测试
- [ ] 与 trigger.dev 输出对齐验证
- [ ] 性能基准测试

## 🔧 Technical Requirements

### Dependencies

```go
// go.mod additions
require (
    github.com/posthog/posthog-go v1.2.19  // PostHog Go SDK
    github.com/google/uuid v1.6.0          // UUID 生成 (已有)
)
```

### Environment Variables

```bash
# 分析服务配置
POSTHOG_PROJECT_KEY="your_posthog_key"
POSTHOG_HOST="https://app.posthog.com"  # 可选，默认值
ANALYTICS_ENABLED="true"                # 可选，默认启用
```

### Configuration

```go
type Config struct {
    PostHogProjectKey string
    PostHogHost       string
    Enabled           bool
}
```

## 📊 Alignment Requirements

### Functional Alignment

- ✅ **API 兼容性**: 所有 trigger.dev 的分析功能都有对应的 Go 实现
- ✅ **数据格式**: 事件和属性结构与 trigger.dev 保持一致
- ✅ **分组策略**: user/organization/project/environment 分组逻辑对齐
- ✅ **错误处理**: 优雅降级机制匹配

### Behavioral Alignment

- ✅ **单例模式**: 全局共享分析服务实例
- ✅ **条件初始化**: API Key 缺失时的静默处理
- ✅ **事件命名**: 使用相同的事件名称和属性键
- ✅ **PostHog 集成**: 相同的 PostHog 配置和使用模式

## 🧪 Testing Strategy

### 1. Unit Tests (80/20 原则)

```go
func TestAnalyticsService_UserIdentify(t *testing.T)
func TestAnalyticsService_OrganizationNew(t *testing.T)
func TestAnalyticsService_ProjectIdentify(t *testing.T)
func TestAnalyticsService_Capture(t *testing.T)
func TestAnalyticsService_DisabledClient(t *testing.T)
```

### 2. Integration Tests

- PostHog 客户端集成测试
- 批处理和错误恢复测试
- 性能负载测试

### 3. Alignment Validation

- 事件数据格式对比
- 分组行为验证
- API 调用模式匹配

## 📈 Success Metrics

1. **功能覆盖率**: 100% trigger.dev 功能对应
2. **测试覆盖率**: >90% 代码覆盖率
3. **性能目标**:
   - 事件捕获延迟 <10ms
   - 批处理吞吐量 >1000 events/s
4. **可靠性**: 99.9% 成功率（PostHog 可用时）

## 🔒 Security & Privacy

- ✅ **数据脱敏**: 敏感信息在发送前处理
- ✅ **配置安全**: API Key 通过环境变量管理
- ✅ **优雅降级**: 分析失败不影响业务逻辑
- ✅ **GDPR 合规**: 支持用户数据删除请求

## 📝 Implementation Notes

### Best Practices

1. **错误隔离**: 分析失败不应影响主业务流程
2. **异步处理**: 事件发送使用异步机制
3. **批处理**: 合并多个事件以提高性能
4. **重试机制**: 网络故障时的指数退避重试
5. **监控埋点**: 分析服务本身的健康监控

### Go Idioms

1. **Context 传递**: 所有公共方法接受 context.Context
2. **错误处理**: 明确的错误返回和处理
3. **并发安全**: 使用适当的同步机制
4. **接口定义**: 清晰的接口抽象便于测试和扩展

## 🎯 Migration Checklist

### Preparation

- [ ] 分析 trigger.dev 所有使用场景
- [ ] 确定 PostHog Go SDK 版本兼容性
- [ ] 设计 Go 版本的 API 接口

### Implementation

- [ ] 基础结构和配置
- [ ] 用户分析功能
- [ ] 组织分析功能
- [ ] 项目分析功能
- [ ] 环境分析功能
- [ ] 通用遥测功能

### Validation

- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 性能基准达标
- [ ] 与 trigger.dev 对齐验证

### Documentation

- [ ] API 文档更新
- [ ] 使用示例编写
- [ ] 迁移指南制作

---

**Target Timeline**: 7 工作日  
**Risk Level**: 🟢 Low (独立服务，依赖简单)  
**Business Impact**: 🟡 Medium (用户行为分析对产品运营重要)
