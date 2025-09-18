# CurrentOrganization Service 迁移计划

## 📋 项目概述

将 trigger.dev 的 `currentOrganization.server.ts` 迁移到 KongFlow Go 后端，实现多租户组织会话管理功能。

**核心目标**: 严格对齐 trigger.dev 实现，避免过度工程，提供简洁高效的组织切换服务。

---

## 🔍 trigger.dev 源码分析

### 源码结构分析 (47 行代码)

```typescript
// 1. Cookie Session Storage 配置 (与 onboardingSession 完全一致的模式)
export const currentOrgSessionStorage = createCookieSessionStorage({
  cookie: {
    name: '__organization', // Cookie名称
    sameSite: 'lax', // CSRF保护
    path: '/', // 全站生效
    httpOnly: true, // XSS防护
    secrets: [env.SESSION_SECRET], // 加密密钥
    secure: env.NODE_ENV === 'production', // 生产环境HTTPS
    maxAge: 60 * 60 * 24, // 1天过期
  },
});

// 2. 底层会话操作
function getCurrentOrgSession(request: Request);
function commitCurrentOrgSession(session: Session);

// 3. 高级API - 组织管理
async function getCurrentOrg(request: Request): Promise<string | undefined>;
async function setCurrentOrg(slug: string, request: Request);
async function clearCurrentOrg(request: Request);
```

### 核心功能特征

1. **数据结构**: 仅存储组织 `slug` 字符串
2. **生命周期**: 24 小时自动过期
3. **安全特性**: 与 `onboardingSession` 完全相同的安全配置
4. **API 模式**: 获取/设置/清除的标准 CRUD 模式
5. **返回模式**: `setCurrentOrg` 和 `clearCurrentOrg` 返回 session 对象供手动提交

### 使用场景分析

- **组织切换**: 用户在多个组织间切换当前工作组织
- **会话持久化**: 保持用户选择的组织在浏览器会话中
- **多租户支持**: 为多租户应用提供组织上下文

---

## 🎯 Go 实现设计

### 设计原则

1. **严格对齐**: 与 trigger.dev 行为 100%一致
2. **模式复用**: 直接复用 `onboardingSession` 的成功模式
3. **简洁实现**: 避免过度设计，保持实现简单
4. **Go 最佳实践**: 遵循 Go 语言惯例和错误处理

### 核心 API 映射

| trigger.dev                        | KongFlow Go                      | 说明              |
| ---------------------------------- | -------------------------------- | ----------------- |
| `getCurrentOrg(request)`           | `GetCurrentOrg(r *http.Request)` | 获取当前组织 slug |
| `setCurrentOrg(slug, request)`     | `SetCurrentOrg(w, r, slug)`      | 设置当前组织      |
| `clearCurrentOrg(request)`         | `ClearCurrentOrg(w, r)`          | 清除组织选择      |
| `getCurrentOrgSession(request)`    | `GetSession(r)`                  | 获取原始会话      |
| `commitCurrentOrgSession(session)` | `CommitSession(w, r, session)`   | 提交会话更改      |

### 技术栈选择

- **会话管理**: `github.com/gorilla/sessions` (与现有服务一致)
- **加密存储**: Cookie-based，AES 加密
- **配置管理**: 环境变量 `SESSION_SECRET`
- **测试框架**: Go 标准测试 + `httptest`

---

## 📁 项目结构 (极简版)

```
kongflow/backend/internal/services/currentorganization/
├── currentorganization.go      # 核心实现 (~90行)
├── currentorganization_test.go # 单元测试 (~200行)
└── example_test.go            # 使用示例 (~150行)
```

**总代码量预估**: ~440 行 (对比 trigger.dev 47 行，合理扩展)

---

## 🚀 实施计划 (避免过度工程)

### 阶段 1: 核心实现 (90 分钟)

#### 1.1 创建服务结构 (15 分钟)

```bash
mkdir -p internal/services/currentorganization
```

#### 1.2 实现核心服务 (60 分钟)

```go
// currentorganization.go - 核心功能

const (
    cookieName = "__organization"  // 严格对齐 trigger.dev
    orgSlugKey = "currentOrg"      // Session key
)

// 复用 onboardingSession 的初始化模式
func init() {
    // 相同的 Cookie 配置
}

// 核心API实现
func GetCurrentOrg(r *http.Request) (*string, error)
func SetCurrentOrg(w http.ResponseWriter, r *http.Request, slug string) error
func ClearCurrentOrg(w http.ResponseWriter, r *http.Request) error
```

#### 1.3 配置对齐验证 (15 分钟)

- Cookie 名称: `__organization` ✅
- 过期时间: 24 小时 ✅
- 安全配置: 与 onboardingSession 一致 ✅

### 阶段 2: 测试覆盖 (60 分钟)

#### 2.1 单元测试 (40 分钟)

```go
// 核心测试用例 (80/20原则)
func TestGetCurrentOrg_NoCookie()        // 空状态测试
func TestSetCurrentOrg()                 // 设置组织测试
func TestGetCurrentOrg_ValidCookie()     // 获取组织测试
func TestClearCurrentOrg()               // 清除组织测试
func TestOrgSessionRoundTrip()           // 完整流程测试
func TestSetCurrentOrg_EmptySlug()       // 边界条件测试
func TestAdvancedAPI()                   // 高级API测试
```

#### 2.2 示例文档 (20 分钟)

```go
// example_test.go - 可执行示例
func ExampleGetCurrentOrg()
func ExampleSetCurrentOrg()
func ExampleClearCurrentOrg()
func Example_organizationWorkflow()
```

### 阶段 3: 文档和验证 (30 分钟)

#### 3.1 API 文档注释 (15 分钟)

- 每个函数包含 trigger.dev 对比说明
- 使用场景和错误处理说明

#### 3.2 对齐度验证 (15 分钟)

- Cookie 配置 100%对齐验证
- API 行为一致性验证
- 错误场景处理验证

**总实施时间**: 3 小时 (对比 onboardingSession 的成功经验)

---

## 🧪 测试策略 (80/20 原则)

### 核心测试覆盖 (目标: 75%+)

1. **基础功能测试** (60%权重)

   - 获取、设置、清除组织的核心流程
   - Cookie 正确设置和读取
   - 空状态和正常状态处理

2. **边界条件测试** (25%权重)

   - 空字符串组织 slug 处理
   - 无效 cookie 处理
   - 重复操作幂等性

3. **集成测试** (15%权重)
   - 完整的用户组织切换流程
   - 高级 API 使用场景

### 测试数据设计

```go
// 测试用例设计
testCases := []struct {
    name     string
    orgSlug  string
    expected string
}{
    {"normal org", "acme-corp", "acme-corp"},
    {"org with hyphens", "my-test-org", "my-test-org"},
    {"empty slug", "", ""},
}
```

---

## 🎯 验收标准 (简化版)

### 功能对齐标准

✅ **Cookie 配置 100%对齐**

- 名称: `__organization`
- 生命周期: 24 小时
- 安全设置: 与 trigger.dev 完全一致

✅ **API 行为 100%对齐**

- `GetCurrentOrg()` 返回 `nil` 对应 trigger.dev 的 `undefined`
- `SetCurrentOrg()` 自动提交会话
- `ClearCurrentOrg()` 幂等操作

✅ **数据格式兼容**

- 组织 slug 存储格式一致
- 跨平台 cookie 可读性

### 质量标准

- **测试覆盖率**: 75%+ (遵循 80/20 原则)
- **代码行数**: <100 行核心实现
- **实施时间**: 3 小时内完成
- **零回归**: 现有服务不受影响

---

## 📊 实施检查点

### 检查点 1: 核心实现完成 (90 分钟后)

- [ ] Cookie 配置与 trigger.dev 对齐
- [ ] 三个核心 API 实现完成
- [ ] 基础错误处理就位

### 检查点 2: 测试覆盖完成 (150 分钟后)

- [ ] 7 个核心测试用例通过
- [ ] 测试覆盖率达到 75%+
- [ ] 示例代码可运行

### 检查点 3: 交付就绪 (180 分钟后)

- [ ] 所有测试通过
- [ ] 文档注释完整
- [ ] 对齐度验证通过

---

## 🔧 依赖和前置条件

### 环境依赖

- `SESSION_SECRET` 环境变量 (已存在)
- `NODE_ENV` 环境变量 (已存在)

### 代码依赖

- `github.com/gorilla/sessions` (已使用)
- Go 标准库: `net/http`, `time`, `os`

### 基础设施依赖

- 无需额外基础设施
- 复用现有 session 管理基础设施

---

## ✅ 交付物清单

1. **核心实现**
   - `currentorganization.go` - 主要服务实现
2. **测试覆盖**

   - `currentorganization_test.go` - 单元测试
   - `example_test.go` - 使用示例

3. **文档**
   - 代码内 API 文档注释
   - 使用示例和最佳实践

---

## 🚨 风险评估与缓解

### 低风险项目

- **实现风险**: 低 (复用成熟模式)
- **技术风险**: 低 (技术栈已验证)
- **依赖风险**: 低 (零新依赖)

### 缓解措施

- **渐进实施**: 先实现核心功能，再完善边界情况
- **充分测试**: 复用 onboardingSession 的测试模式
- **对齐验证**: 每个功能点都与 trigger.dev 对比验证

---

## 📝 成功标准

**最低可行产品 (MVP)**:

- ✅ 3 个核心 API 功能正确
- ✅ Cookie 配置与 trigger.dev 对齐
- ✅ 基础测试覆盖通过

**完整交付标准**:

- ✅ 75%+测试覆盖率
- ✅ 完整示例文档
- ✅ 95%+对齐度验证

---

**项目预期**: 3 小时内完成高质量、严格对齐的 currentOrganization 服务迁移，为多租户组织管理奠定坚实基础。
