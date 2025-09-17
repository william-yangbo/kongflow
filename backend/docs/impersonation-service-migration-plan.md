# Impersonation Service 迁移计划

## 1. 服务概述

### 1.1 原服务分析 (trigger.dev)

- **文件位置**: `apps/webapp/app/services/impersonation.server.ts`
- **代码行数**: 46 行
- **核心功能**: 管理管理员用户身份模拟会话
- **技术栈**: Remix createCookieSessionStorage
- **使用场景**: 管理员模拟普通用户身份进行操作和调试

### 1.2 功能特性

1. **Cookie 会话存储**: 使用加密 Cookie 存储被模拟的用户 ID
2. **安全配置**: httpOnly、sameSite=lax、secure (生产环境)
3. **过期时间**: 1 天过期时间 (86400 秒)
4. **Cookie 名称**: `__impersonate`
5. **操作接口**: 设置、获取、清除模拟用户 ID

### 1.3 API 接口分析

```typescript
// trigger.dev 原始接口
export function getImpersonationSession(request: Request): Promise<Session>;
export function commitImpersonationSession(session: Session): Promise<string>;
export async function getImpersonationId(
  request: Request
): Promise<string | undefined>;
export async function setImpersonationId(
  userId: string,
  request: Request
): Promise<Session>;
export async function clearImpersonationId(request: Request): Promise<Session>;
```

### 1.4 使用模式分析

**核心使用场景**:

1. **身份检查** (`session.server.ts`): 优先检查模拟身份，如果存在则使用模拟用户 ID
2. **设置模拟** (`admin._index.tsx`): 管理员选择用户进行模拟
3. **清除模拟** (`resources.impersonation.ts`): 结束模拟会话

**关键业务逻辑**:

```typescript
// 用户身份获取的优先级顺序
export async function getUserId(request: Request): Promise<string | undefined> {
  const impersonatedUserId = await getImpersonationId(request);
  if (impersonatedUserId) return impersonatedUserId; // 优先使用模拟身份

  let authUser = await authenticator.isAuthenticated(request);
  return authUser?.userId; // 后备真实身份
}
```

### 1.5 依赖关系

- **内部依赖**: `~/env.server` (SESSION_SECRET)
- **外部依赖**: `@remix-run/node` (createCookieSessionStorage)
- **使用者**:
  - `session.server.ts` (身份验证核心)
  - `admin._index.tsx` (管理界面)
  - `resources.impersonation.ts` (清除接口)
  - `_app/route.tsx` (全局检查)

## 2. Go 迁移设计

### 2.1 技术选型

```go
// 核心依赖
- net/http (HTTP 请求和 Cookie 管理)
- crypto/hmac + crypto/sha256 (HMAC 签名，与 Remix 兼容)
- encoding/base64 (编码解码)
- encoding/json (JSON 序列化)
- time (过期时间管理)
- errors (错误处理)
```

### 2.2 架构设计

```
├── internal/impersonation/
│   ├── types.go              # 接口定义和配置结构
│   ├── service.go            # 核心服务实现 (Go风格)
│   ├── cookie.go             # Cookie 操作辅助函数
│   ├── crypto.go             # HMAC 签名和验证
│   ├── README.md             # 服务文档和使用指南
│   ├── impersonation_test.go # 核心功能测试
│   └── example_test.go       # 使用示例测试
```

### 2.3 接口设计

#### Go 风格接口 (专注实用性)

```go
type ImpersonationService interface {
    // 设置模拟用户ID
    SetImpersonation(w http.ResponseWriter, r *http.Request, userID string) error

    // 获取模拟用户ID
    GetImpersonation(r *http.Request) (string, error)

    // 清除模拟用户ID
    ClearImpersonation(w http.ResponseWriter, r *http.Request) error

    // 检查是否处于模拟状态
    IsImpersonating(r *http.Request) bool
}

type Config struct {
    SecretKey   []byte        // HMAC 密钥
    CookieName  string        // Cookie 名称，默认 "__impersonate"
    Domain      string        // Cookie 域名
    Path        string        // Cookie 路径，默认 "/"
    MaxAge      time.Duration // 过期时间，默认 24 小时
    Secure      bool          // HTTPS 环境设置
    HttpOnly    bool          // HttpOnly 标志，默认 true
    SameSite    http.SameSite // SameSite 策略，默认 Lax
}
```

### 2.4 实现策略

#### Go 原生实现方案

基于项目的技术栈（Go 后端 + React 前端），采用单一的 Go 风格实现：

1. **直接 Cookie 操作**:

   - 自动管理 HTTP Cookie 设置和清除
   - Go 式错误处理，简洁明确
   - 高性能，低内存分配

2. **与 React 前端集成**:
   - HTTP API 调用模式
   - 标准的 JSON 响应格式
   - RESTful 接口设计

#### Cookie 格式设计

```go
// Cookie 值结构 (JSON + Base64 + HMAC)
type CookieValue struct {
    ImpersonatedUserID string    `json:"impersonatedUserId"`
    CreatedAt          time.Time `json:"createdAt"`
}

// Cookie 签名格式: base64(json_data).hmac_signature
```

#### 安全配置

```go
func DefaultConfig() *Config {
    return &Config{
        CookieName: "__impersonate",
        Path:       "/",
        MaxAge:     24 * time.Hour,
        HttpOnly:   true,
        SameSite:   http.SameSiteLaxMode,
        Secure:     false, // 通过环境变量控制
    }
}
```

## 3. 对齐验证

### 3.1 功能对齐检查表

| 功能项目    | trigger.dev          | Go Service         | 对齐度 |
| ----------- | -------------------- | ------------------ | ------ |
| Cookie 名称 | `__impersonate`      | ✅                 | 100%   |
| 过期时间    | 24 小时              | ✅                 | 100%   |
| HttpOnly    | true                 | ✅                 | 100%   |
| SameSite    | lax                  | ✅                 | 100%   |
| 安全配置    | 生产环境 secure      | ✅                 | 100%   |
| 设置操作    | setImpersonationId   | SetImpersonation   | 90%    |
| 获取操作    | getImpersonationId   | GetImpersonation   | 90%    |
| 清除操作    | clearImpersonationId | ClearImpersonation | 90%    |
| 核心功能    | 用户身份模拟         | ✅                 | 100%   |
| 安全性      | HMAC 签名保护        | ✅                 | 100%   |

### 3.2 API 对齐分析

**Go Service (90% 功能对齐)**:

- 核心功能完全对齐：用户身份模拟管理
- Cookie 配置完全对齐：名称、安全属性、过期时间
- API 风格适配：遵循 Go 语言惯例和 HTTP 标准
- 集成方式优化：与 Go 后端和 React 前端完美配合

## 4. 实施计划

### 4.1 开发阶段

#### 阶段 1: 核心结构搭建 (20 分钟)

- [ ] 创建目录结构
- [ ] 定义接口和类型 (`types.go`)
- [ ] 实现基础配置 (`Config` 结构)

#### 阶段 2: 加密和 Cookie 基础 (30 分钟)

- [ ] 实现 HMAC 签名功能 (`crypto.go`)
- [ ] 实现 Cookie 操作辅助函数 (`cookie.go`)
- [ ] 基础序列化和反序列化

#### 阶段 3: 核心服务实现 (40 分钟)

- [ ] 实现 Go 风格 Service (`service.go`)
- [ ] 错误处理和边界情况
- [ ] 与 HTTP 中间件集成

#### 阶段 4: 测试开发 (50 分钟)

- [ ] 核心功能测试 (`impersonation_test.go`)
- [ ] 使用示例测试 (`example_test.go`)
- [ ] 安全性和边界测试
- [ ] HTTP 集成测试

#### 阶段 5: 文档和完善 (20 分钟)

- [ ] README 文档
- [ ] API 使用指南
- [ ] 与 React 前端集成示例

### 4.2 质量目标

- **测试覆盖率**: 80%+ (遵循 80/20 原则)
- **性能目标**: Cookie 操作 < 1ms
- **内存效率**: 零不必要的分配
- **功能对齐度**: 90%+ 核心功能对齐

### 4.3 测试策略

#### 4.3.1 核心功能测试

```go
// 基础操作测试
func TestSetGetClearImpersonation(t *testing.T)
func TestImpersonationExpiry(t *testing.T)
func TestInvalidCookieHandling(t *testing.T)

// 并发安全测试
func TestConcurrentAccess(t *testing.T)

// 边界条件测试
func TestEmptyUserID(t *testing.T)
func TestLongUserID(t *testing.T)
```

#### 4.3.2 HTTP 集成测试

```go
// Go HTTP 集成测试
func TestHTTPMiddlewareIntegration(t *testing.T)
func TestReactFrontendIntegration(t *testing.T)
func TestAPIEndpointBehavior(t *testing.T)
```

#### 4.3.3 安全性测试

```go
// 签名验证测试
func TestSignatureValidation(t *testing.T)
func TestTamperedCookieRejection(t *testing.T)
func TestSecretKeyRotation(t *testing.T)
```

### 4.4 验收标准

1. **功能完整性**: 实现所有 trigger.dev 原有功能
2. **功能对齐度**: 90%+ 核心功能对齐
3. **测试覆盖**: 80% 以上测试覆盖率
4. **性能标准**: Cookie 操作延迟 < 1ms
5. **安全性**: 通过签名验证和篡改检测测试
6. **文档完备**: 完整的 README 和使用示例

## 5. 风险评估与缓解

### 5.1 技术风险

**风险**: HMAC 签名与 Remix 不兼容

- **概率**: 低
- **影响**: 中
- **缓解**: 参考 sessionStorage 成功实现，使用相同签名算法

**风险**: Cookie 格式差异

- **概率**: 低
- **影响**: 中
- **缓解**: 详细测试 Base64 编码和 JSON 序列化

### 5.2 业务风险

**风险**: 模拟会话安全漏洞

- **概率**: 低
- **影响**: 高
- **缓解**: 强制签名验证，设置合理过期时间

**风险**: 与现有认证系统冲突

- **概率**: 低
- **影响**: 中
- **缓解**: 设计独立的 Cookie 命名空间

## 6. 成功指标

### 6.1 技术指标

- ✅ 90%+ trigger.dev 核心功能对齐度
- ✅ Go 语言最佳实践遵循
- ✅ 80%+ 测试覆盖率
- ✅ < 1ms Cookie 操作延迟

### 6.2 业务指标

- ✅ 支持管理员用户模拟功能
- ✅ 安全的会话隔离
- ✅ 与认证系统无缝集成
- ✅ 便于调试和故障排查

## 7. 后续计划

### 7.1 集成规划

1. 与现有认证中间件集成
2. React 前端 API 调用接口
3. 添加审计日志功能

### 7.2 扩展可能

1. 支持权限范围限制
2. 模拟会话时间追踪
3. 管理界面增强 (React 组件)

---

**预计总工时**: 2.5-3 小时 (精简后)
**风险等级**: 低
**优先级**: 中等
**依赖服务**: 无 (完全独立)
