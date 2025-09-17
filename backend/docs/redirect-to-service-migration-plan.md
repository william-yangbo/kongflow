# RedirectTo Service 迁移计划

## 1. 服务概述

### 1.1 原服务分析 (trigger.dev)

- **文件位置**: `apps/webapp/app/services/redirectTo.server.ts`
- **代码行数**: 52 行
- **核心功能**: 管理用户登录后的重定向目标 URL
- **技术栈**: Remix createCookieSessionStorage + Zod 验证
- **使用场景**: 登录流程中保存和恢复用户原始访问意图

### 1.2 功能特性

1. **Cookie 会话存储**: 使用加密 Cookie 存储重定向 URL
2. **安全配置**: httpOnly、sameSite、secure 等安全选项
3. **过期时间**: 1 天过期时间 (86400 秒)
4. **数据验证**: 使用 Zod 进行字符串验证
5. **操作接口**: 设置、获取、清除重定向 URL

### 1.3 依赖关系

- **内部依赖**: `~/env.server` (环境变量)
- **外部依赖**: `@remix-run/node`, `zod`
- **使用者**: 登录页面、魔法链接页面、应用主路由

## 2. Go 迁移设计

### 2.1 技术选型

```go
// 核心依赖
- net/http (Cookie 管理)
- crypto/aes + crypto/cipher (加密)
- encoding/base64 (编码)
- time (过期时间)
- encoding/json (序列化)
```

### 2.2 架构设计

```
├── internal/services/redirectto/
│   ├── service.go          # 服务主逻辑
│   ├── cookie.go           # Cookie 操作
│   ├── crypto.go           # 加密解密
│   └── types.go            # 类型定义
└── internal/services/redirectto/
    └── redirectto_test.go  # 测试文件
```

### 2.3 核心接口设计

```go
type RedirectToService interface {
    SetRedirectTo(w http.ResponseWriter, r *http.Request, redirectTo string) error
    GetRedirectTo(r *http.Request) (string, error)
    ClearRedirectTo(w http.ResponseWriter, r *http.Request) error
}

type Config struct {
    CookieName   string
    SecretKey    []byte
    MaxAge       time.Duration
    Secure       bool
    HTTPOnly     bool
    SameSite     http.SameSite
    Path         string
}
```

## 3. 实施步骤

### 3.1 第一阶段：核心服务实现

**目标**: 实现基础的 RedirectTo 服务功能

**任务列表**:

1. 创建服务目录结构
2. 实现 `types.go` - 定义接口和配置结构
3. 实现 `crypto.go` - AES 加密解密功能
4. 实现 `cookie.go` - Cookie 操作辅助函数
5. 实现 `service.go` - 主服务逻辑

**验收标准**:

- [ ] 服务可以正确设置加密的重定向 Cookie
- [ ] 服务可以正确读取和解密 Cookie 内容
- [ ] 服务可以正确清除 Cookie
- [ ] Cookie 配置与 trigger.dev 完全对齐

### 3.2 第二阶段：配置和环境

**目标**: 集成配置管理和环境变量

**任务列表**:

1. 在 `internal/config/config.go` 中添加 RedirectTo 配置
2. 在环境变量中添加必要的密钥配置
3. 创建服务实例化和依赖注入
4. 在服务注册中心注册 RedirectTo 服务

**验收标准**:

- [ ] 配置可以从环境变量正确加载
- [ ] 服务可以在不同环境下正确工作 (dev/prod)
- [ ] 密钥管理安全且可配置

### 3.3 第三阶段：HTTP 中间件集成

**目标**: 创建便于使用的 HTTP 中间件

**任务列表**:

1. 创建 RedirectTo 中间件
2. 集成到认证流程中
3. 添加便利的 helper 函数
4. 文档和使用示例

**验收标准**:

- [ ] 中间件可以轻松集成到路由中
- [ ] 认证流程可以无缝使用重定向功能
- [ ] API 设计直观易用

### 3.4 第四阶段：测试覆盖

**目标**: 全面测试覆盖，确保功能正确性

**任务列表**:

1. 单元测试 - 加密解密功能
2. 单元测试 - Cookie 操作
3. 集成测试 - 完整的重定向流程
4. 边界测试 - 异常情况处理
5. 性能测试 - 加密性能评估

**验收标准**:

- [ ] 单元测试覆盖率 > 90%
- [ ] 所有边界情况都有对应测试
- [ ] 集成测试验证端到端流程
- [ ] 性能满足预期要求

## 4. 技术实现细节

### 4.1 Cookie 加密方案

```go
// 使用 AES-GCM 模式进行加密，确保数据完整性和机密性
func (s *service) encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(s.config.SecretKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.URLEncoding.EncodeToString(ciphertext), nil
}
```

### 4.2 配置对齐

```go
// 确保与 trigger.dev 配置完全对齐
var DefaultConfig = Config{
    CookieName: "__redirectTo",
    MaxAge:     24 * time.Hour,  // ONE_DAY = 60 * 60 * 24
    HTTPOnly:   true,
    SameSite:   http.SameSiteLaxMode,  // "lax"
    Path:       "/",
    Secure:     false,  // 将根据环境动态设置
}
```

### 4.3 错误处理

```go
var (
    ErrInvalidCookie     = errors.New("invalid cookie format")
    ErrDecryptionFailed  = errors.New("failed to decrypt cookie")
    ErrCookieNotFound    = errors.New("redirect cookie not found")
    ErrInvalidRedirectURL = errors.New("invalid redirect URL")
)
```

## 5. 测试策略

### 5.1 单元测试用例

1. **加密解密测试**

   - 正常加密解密流程
   - 不同长度字符串处理
   - 错误输入处理

2. **Cookie 操作测试**

   - 设置 Cookie 属性验证
   - Cookie 解析正确性
   - 过期时间处理

3. **服务接口测试**
   - SetRedirectTo 功能验证
   - GetRedirectTo 功能验证
   - ClearRedirectTo 功能验证

### 5.2 集成测试场景

1. **登录重定向流程**

   ```
   用户访问受保护页面 → 重定向到登录 → 登录成功 → 重定向到原页面
   ```

2. **Cookie 生命周期**

   ```
   设置 Cookie → 读取 Cookie → 清除 Cookie → 验证清除成功
   ```

3. **安全性测试**
   ```
   篡改 Cookie → 验证拒绝访问
   过期 Cookie → 验证自动清理
   ```

## 6. 风险评估与缓解

### 6.1 技术风险

| 风险          | 影响 | 缓解措施                    |
| ------------- | ---- | --------------------------- |
| 加密密钥泄露  | 高   | 环境变量管理 + 密钥轮换机制 |
| Cookie 兼容性 | 中   | 详细的浏览器兼容性测试      |
| 性能影响      | 低   | 加密算法性能测试 + 缓存优化 |

### 6.2 业务风险

| 风险         | 影响 | 缓解措施              |
| ------------ | ---- | --------------------- |
| 用户体验断裂 | 高   | 渐进式迁移 + 回退机制 |
| 重定向失效   | 中   | 充分的回归测试        |

## 7. 交付物清单

### 7.1 代码交付

- [ ] `internal/services/redirectto/` 完整实现
- [ ] 配置文件更新
- [ ] 中间件集成代码
- [ ] 完整测试套件

### 7.2 文档交付

- [ ] API 文档
- [ ] 使用示例
- [ ] 部署指南
- [ ] 故障排除手册

### 7.3 质量保证

- [ ] 代码审查通过
- [ ] 测试覆盖率达标
- [ ] 性能基准测试
- [ ] 安全审计通过

## 8. 时间估算

| 阶段                 | 工作量   | 依赖     |
| -------------------- | -------- | -------- |
| 第一阶段：核心实现   | 0.5-1 天 | 无       |
| 第二阶段：配置集成   | 0.5 天   | 第一阶段 |
| 第三阶段：中间件集成 | 0.5 天   | 第二阶段 |
| 第四阶段：测试覆盖   | 1 天     | 前三阶段 |

**总估算**: 2.5-3 天

## 9. 验收标准

### 9.1 功能验收

- [ ] 与 trigger.dev 行为完全一致
- [ ] 所有 API 接口正常工作
- [ ] Cookie 配置完全对齐
- [ ] 安全性要求满足

### 9.2 质量验收

- [ ] 单元测试覆盖率 ≥ 90%
- [ ] 集成测试通过
- [ ] 代码审查通过
- [ ] 性能满足要求

### 9.3 文档验收

- [ ] API 文档完整
- [ ] 使用示例清晰
- [ ] 故障排除手册完备

---

**注意**: 本迁移计划遵循"保持对齐，避免过度工程"的原则，专注于复现 trigger.dev 的核心功能，避免不必要的复杂性。
