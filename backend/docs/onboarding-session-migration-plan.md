# OnboardingSession Service 迁移计划

## 📋 项目概述

### 目标

将 trigger.dev 的 `onboardingSession.server.ts` 服务迁移到 KongFlow Go 后端，确保功能完全对齐，为用户引导流程提供会话管理能力。

### 背景

OnboardingSession 服务负责管理用户引导过程中的状态数据，通过 Cookie 会话存储用户的引导进度信息，是提升新用户体验的关键服务。

## 🔍 trigger.dev 源码分析

### 核心配置

```typescript
// 来源：onboardingSession.server.ts
export const onboardingSessionStorage = createCookieSessionStorage({
  cookie: {
    name: '__onboarding', // Cookie 名称
    sameSite: 'lax', // CSRF 防护
    path: '/', // 全站可用
    httpOnly: true, // XSS 防护
    secrets: [env.SESSION_SECRET], // 签名密钥
    secure: env.NODE_ENV === 'production', // 生产环境 HTTPS
    maxAge: 60 * 60 * 24, // 24小时过期
  },
});
```

### API 接口清单

| 函数名                    | 参数                           | 返回值                       | 功能描述                 |
| ------------------------- | ------------------------------ | ---------------------------- | ------------------------ |
| `getOnboardingSession`    | `request: Request`             | `Session`                    | 获取引导会话对象         |
| `commitOnboardingSession` | `session: Session`             | `string`                     | 提交会话并返回 Cookie 头 |
| `getWorkflowDate`         | `request: Request`             | `Promise<Date \| undefined>` | 获取工作流日期           |
| `setWorkflowDate`         | `date: Date, request: Request` | `Promise<Session>`           | 设置工作流日期           |
| `clearWorkflowDate`       | `request: Request`             | `Promise<Session>`           | 清除工作流日期           |

### 核心业务逻辑

1. **会话管理**: 基于 Cookie 的会话存储，24 小时有效期
2. **数据存储**: 存储 `workflowDate` 字段，格式为 ISO 时间字符串
3. **状态操作**: 支持获取、设置、清除工作流日期
4. **安全机制**: HttpOnly + Secure + SameSite 防护

## 🎯 Go 实现设计

### API 接口对齐设计

```go
// OnboardingSessionService 接口定义
type OnboardingSessionService interface {
    // 获取工作流日期
    GetWorkflowDate(r *http.Request) (*time.Time, error)

    // 设置工作流日期（自动提交Cookie）
    SetWorkflowDate(w http.ResponseWriter, r *http.Request, date time.Time) error

    // 清除工作流日期（自动提交Cookie）
    ClearWorkflowDate(w http.ResponseWriter, r *http.Request) error

    // 底层会话操作（高级API）
    GetSession(r *http.Request) (*Session, error)
    CommitSession(w http.ResponseWriter, session *Session) error
}

// Config 配置结构
type Config struct {
    CookieName   string        // "__onboarding"
    SecretKey    []byte        // 签名密钥
    MaxAge       time.Duration // 24 * time.Hour
    Secure       bool          // 基于环境动态设置
    HTTPOnly     bool          // true
    SameSite     http.SameSite // http.SameSiteLaxMode
    Path         string        // "/"
}
```

### 对齐策略

#### 1. Cookie 配置完全对齐

- **Cookie 名称**: `__onboarding` (与 trigger.dev 一致)
- **过期时间**: 24 小时 (与 trigger.dev 一致)
- **安全属性**: HttpOnly=true, SameSite=Lax, Secure=production
- **路径**: "/" (全站可用)

#### 2. API 行为对齐

- **类型兼容**: Go `time.Time` ↔ TypeScript `Date`
- **空值处理**: Go `*time.Time` (nil) ↔ TypeScript `undefined`
- **错误处理**: Go 惯用 `(value, error)` 模式
- **自动提交**: 简化 API，自动处理 Cookie 提交

#### 3. Go 语言适配

- **接口设计**: 符合 Go 接口惯例
- **错误处理**: 使用 Go 标准错误模式
- **类型安全**: 利用 Go 强类型系统
- **性能优化**: 复用 sessionstorage 基础设施

## 📁 项目结构 (简化版)

```
internal/services/onboardingsession/
├── README.md                      # 使用文档
├── onboardingsession.go           # 主服务实现 (参考 sessionstorage.go)
├── onboardingsession_test.go      # 单元测试
└── example_test.go               # 示例代码
```

**简化理由**:

- trigger.dev 源码仅 47 行，功能简单，无需过度分层
- 参考现有 sessionstorage 服务结构(108 行代码完成所有功能)
- 避免过度工程，保持简洁

## 🚀 实施计划 (简化版)

### 阶段 1: 核心实现 (1 天)

#### 任务清单

- [ ] 创建 `onboardingsession.go` - 参考 sessionstorage.go 结构
- [ ] 实现 3 个核心 API: Get/Set/ClearWorkflowDate
- [ ] 复用现有 sessionstorage 基础设施
- [ ] 编写基础单元测试

#### 验收标准

- API 行为与 trigger.dev 一致
- 测试覆盖核心功能即可 (80%+)

### 阶段 2: 完善和文档 (0.5 天)

#### 任务清单

- [ ] 编写 README 和 example_test.go
- [ ] 与 trigger.dev 行为对比验证
- [ ] 代码审查和优化

#### 验收标准

- 功能完整，文档清晰
- 与 trigger.dev 行为基本一致

## 🧪 测试策略 (简化版)

### 单元测试 (覆盖率目标: 80%+)

参考现有服务的测试模式，重点测试：

```go
func TestGetWorkflowDate(t *testing.T) {
    // 基础场景：
    // 1. 无 Cookie 返回 nil
    // 2. 有效 Cookie 正确解析
    // 3. 无效 Cookie 错误处理
}

func TestSetWorkflowDate(t *testing.T) {
    // 基础场景：
    // 1. 设置日期成功
    // 2. Cookie 属性正确
}

func TestClearWorkflowDate(t *testing.T) {
    // 基础场景：
    // 1. 清除已存在日期
    // 2. 幂等性验证
}
```

### 示例测试

```go
func ExampleService_SetWorkflowDate() {
    // 提供使用示例，自动验证文档
}
```

## 🎯 验收标准 (简化版)

### 功能对齐度 (目标: 85%+)

- [ ] Cookie 配置与 trigger.dev 一致
- [ ] 核心 API 行为正确
- [ ] 基础错误处理完善

### 代码质量

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 代码简洁易读
- [ ] 有使用示例和文档

---

**项目时间线**: 1.5 天  
**风险等级**: 极低  
**复杂度**: 简单 (参考 trigger.dev 仅 47 行代码)  
**对齐度目标**: 85%+ (务实目标)

**核心原则**: 避免过度工程，保持简单有效，参考现有服务实现模式。
