# API Authentication Service 迁移计划

## 🎯 迁移目标

将 trigger.dev 的 `apiAuth.server.ts` 迁移到 KongFlow Go 后端，实现与原版完全对齐的 API 认证功能。

## 📋 trigger.dev apiAuth Service 深度分析

### 核心功能概览

trigger.dev 的 `apiAuth.server.ts` 是一个复杂的认证服务，支持多种认证方式：

```typescript
// 主要认证方法
1. authenticateApiRequest() - 通用 API 请求认证（已废弃，推荐使用新版本）
2. authenticateApiRequestWithFailure() - 新版 API 请求认证，返回详细错误信息
3. authenticateRequest() - 支持多种认证方式的统一入口
4. authenticateAuthorizationHeader() - 直接验证 Authorization 头

// 支持的认证类型
- Personal Access Token (PAT) - 个人访问令牌
- Organization Access Token (OAT) - 组织访问令牌
- API Key (Public/Private/JWT) - API 密钥（三种类型）
```

### 详细功能分析

#### 1. **API Key 认证**

```typescript
// API Key 类型识别
- PUBLIC Key: "pk_" 前缀 - 公开密钥，权限受限
- PRIVATE Key: "tr_" 前缀 - 私有密钥，完整权限
- PUBLIC_JWT: JWT 格式的公开密钥，支持自定义声明

// 核心方法
authenticateApiKey(apiKey, options):
  - allowPublicKey: 是否允许公开密钥
  - allowJWT: 是否允许 JWT 密钥
  - branchName: 分支名称（多环境支持）

// 返回结果
ApiAuthenticationResult: {
  ok: boolean,
  environment: RuntimeEnvironment, // 关联的运行时环境
  scopes?: string[],              // JWT 权限范围
  oneTimeUse?: boolean,           // 一次性使用标记
  realtime?: boolean              // 实时功能标记
}
```

#### 2. **Personal Access Token 认证**

```typescript
// 个人访问令牌认证
authenticateApiRequestWithPersonalAccessToken(request):
  - 验证 "Bearer {token}" 格式
  - 返回 { userId: string }
  - 用于用户级别的 API 访问
```

#### 3. **Organization Access Token 认证**

```typescript
// 组织访问令牌认证
authenticateApiRequestWithOrganizationAccessToken(request):
  - 验证 "Bearer {token}" 格式
  - 返回 { organizationId: string }
  - 用于组织级别的 API 访问
```

#### 4. **统一认证入口**

```typescript
// 支持灵活的认证方式配置
authenticateRequest<T>(request, allowedMethods?):
  - 自动检测令牌类型
  - 根据配置启用/禁用认证方式
  - 类型安全的返回结果

// 配置示例
const result = await authenticateRequest(request, {
  personalAccessToken: true,
  organizationAccessToken: false,
  apiKey: true
}); // 只允许 PAT 和 API Key
```

### 数据库集成分析

#### RuntimeEnvironment 查询

```typescript
// 通过 API Key 查找环境
findEnvironmentByApiKey(apiKey) -> RuntimeEnvironment + Project + Organization

// 通过公开 API Key 查找环境
findEnvironmentByPublicApiKey(apiKey, branch?) -> RuntimeEnvironment

// 返回完整的环境信息，包括:
- environment.id, apiKey, slug, type
- project.id, name, slug
- organization.id, title, slug
```

### JWT 功能分析

```typescript
// JWT 生成和验证
generateJWTTokenForEnvironment(environment, payload, options):
  - 基于环境生成 JWT 令牌
  - 支持自定义过期时间
  - 包含环境特定的声明

// JWT 验证
validatePublicJwtKey(jwtToken):
  - 验证 JWT 签名
  - 解析声明（scopes, oneTimeUse, realtime）
  - 返回关联的环境信息
```

## 🏗️ Go 实现架构设计

### 1. 包结构

```
internal/
├── shared/                           # 🆕 共享数据库层
│   ├── queries/                      # 共享查询
│   │   ├── users.sql                # User 相关查询
│   │   ├── organizations.sql        # Organization 相关查询
│   │   ├── projects.sql             # Project 相关查询
│   │   └── runtime_environments.sql # RuntimeEnvironment 相关查询
│   ├── db.go                         # SQLC生成
│   ├── models.go                     # 共享模型 (User, Organization, Project, RuntimeEnvironment)
│   └── *.sql.go                      # 生成的查询方法
└── apiauth/                          # ApiAuth 服务层
    ├── queries/                      # 服务特定查询
    │   ├── personal_tokens.sql       # PersonalAccessToken 查询
    │   ├── org_tokens.sql            # OrganizationAccessToken 查询
    │   └── api_keys.sql              # ApiKey 查询（如果需要）
    ├── service.go                    # 主服务实现
    ├── types.go                      # 数据类型定义
    ├── validators.go                 # 令牌验证逻辑
    ├── jwt.go                        # JWT 处理
    ├── repository.go                 # 数据库访问层（组合 shared + apiauth 查询）
    ├── middleware.go                 # HTTP 中间件
    ├── db.go                         # SQLC生成
    ├── models.go                     # 服务特定模型 (PersonalAccessToken, OrganizationAccessToken)
    ├── *.sql.go                      # 生成的查询方法
    └── service_test.go               # 单元测试
```

### 2. 核心接口设计

```go
package apiauth

import (
    "context"
    "net/http"
    "time"
)

// APIAuthService API 认证服务接口
type APIAuthService interface {
    // 通用 API 请求认证
    AuthenticateAPIRequest(ctx context.Context, req *http.Request, opts *AuthOptions) (*AuthenticationResult, error)

    // 直接认证 Authorization 头
    AuthenticateAuthorizationHeader(ctx context.Context, authorization string, opts *AuthOptions) (*AuthenticationResult, error)

    // 统一认证入口（支持多种认证方式）
    AuthenticateRequest(ctx context.Context, req *http.Request, config *AuthConfig) (*UnifiedAuthResult, error)

    // JWT 令牌生成
    GenerateJWTToken(ctx context.Context, env *RuntimeEnvironment, payload map[string]interface{}, opts *JWTOptions) (string, error)

    // 获取认证环境信息
    GetAuthenticatedEnvironment(ctx context.Context, authResult *AuthenticationResult, projectRef, envSlug string) (*AuthenticatedEnvironment, error)
}

// 认证选项
type AuthOptions struct {
    AllowPublicKey bool   `json:"allowPublicKey"`
    AllowJWT       bool   `json:"allowJWT"`
    BranchName     string `json:"branchName,omitempty"`
}

// 认证配置（多种认证方式）
type AuthConfig struct {
    PersonalAccessToken    bool `json:"personalAccessToken"`
    OrganizationAccessToken bool `json:"organizationAccessToken"`
    APIKey                 bool `json:"apiKey"`
}

// API Key 认证结果
type AuthenticationResult struct {
    Success     bool                 `json:"success"`
    Error       string               `json:"error,omitempty"`
    APIKey      string               `json:"apiKey"`
    Type        APIKeyType           `json:"type"`
    Environment *RuntimeEnvironment  `json:"environment"`
    Scopes      []string             `json:"scopes,omitempty"`
    OneTimeUse  bool                 `json:"oneTimeUse,omitempty"`
    Realtime    bool                 `json:"realtime,omitempty"`
}

// 统一认证结果
type UnifiedAuthResult struct {
    Type   AuthenticationType `json:"type"`
    UserID string             `json:"userId,omitempty"`
    OrgID  string             `json:"organizationId,omitempty"`
    APIResult *AuthenticationResult `json:"apiResult,omitempty"`
}

// API Key 类型
type APIKeyType string
const (
    APIKeyTypePublic     APIKeyType = "PUBLIC"     // pk_ 前缀
    APIKeyTypePrivate    APIKeyType = "PRIVATE"    // tr_ 前缀
    APIKeyTypePublicJWT  APIKeyType = "PUBLIC_JWT" // JWT 格式
)

// 认证方式类型
type AuthenticationType string
const (
    AuthTypePersonalAccessToken    AuthenticationType = "personalAccessToken"
    AuthTypeOrganizationAccessToken AuthenticationType = "organizationAccessToken"
    AuthTypeAPIKey                 AuthenticationType = "apiKey"
)

// 运行时环境（对应 trigger.dev 的 RuntimeEnvironment）
type RuntimeEnvironment struct {
    ID             string              `json:"id"`
    Slug           string              `json:"slug"`
    APIKey         string              `json:"apiKey"`
    Type           EnvironmentType     `json:"type"`
    OrganizationID string              `json:"organizationId"`
    ProjectID      string              `json:"projectId"`
    OrgMemberID    *string             `json:"orgMemberId,omitempty"`
    CreatedAt      time.Time           `json:"createdAt"`
    UpdatedAt      time.Time           `json:"updatedAt"`

    // 关联数据
    Project      *Project      `json:"project,omitempty"`
    Organization *Organization `json:"organization,omitempty"`
}

type EnvironmentType string
const (
    EnvironmentTypeProduction  EnvironmentType = "PRODUCTION"
    EnvironmentTypeStaging     EnvironmentType = "STAGING"
    EnvironmentTypeDevelopment EnvironmentType = "DEVELOPMENT"
    EnvironmentTypePreview     EnvironmentType = "PREVIEW"
)

// 认证环境（包含完整信息）
type AuthenticatedEnvironment struct {
    *RuntimeEnvironment
    Project      Project      `json:"project"`
    Organization Organization `json:"organization"`
}

// JWT 选项
type JWTOptions struct {
    ExpirationTime interface{} `json:"expirationTime,omitempty"` // number 或 string
    CustomClaims   map[string]interface{} `json:"customClaims,omitempty"`
}
```

### 3. 数据库访问层实现

```go
// Repository 接口 - 组合使用 shared + apiauth 查询
type APIAuthRepository interface {
    // 使用 shared.Queries 的方法
    FindEnvironmentByAPIKey(ctx context.Context, apiKey string) (*shared.RuntimeEnvironment, error)
    FindEnvironmentByPublicAPIKey(ctx context.Context, apiKey string, branch *string) (*shared.RuntimeEnvironment, error)
    GetEnvironmentWithProjectAndOrg(ctx context.Context, envID string) (*AuthenticatedEnvironment, error)

    // 使用 apiauth.Queries 的方法
    AuthenticatePersonalAccessToken(ctx context.Context, token string) (*PersonalAccessTokenResult, error)
    AuthenticateOrganizationAccessToken(ctx context.Context, token string) (*OrganizationAccessTokenResult, error)
}

// Repository 实现 - 组合两个查询接口
type apiAuthRepository struct {
    sharedQueries  *shared.Queries     // 共享查询 (User, Organization, Project, RuntimeEnvironment)
    apiAuthQueries *Queries            // 服务特定查询 (PersonalAccessToken, OrganizationAccessToken)
    db             DBTX
}

func NewAPIAuthRepository(db DBTX) APIAuthRepository {
    return &apiAuthRepository{
        sharedQueries:  shared.New(db),
        apiAuthQueries: New(db),
        db:             db,
    }
}

// 共享查询示例 - SQLC 查询定义 (internal/shared/queries/)
-- name: FindRuntimeEnvironmentByAPIKey :one
SELECT * FROM runtime_environments WHERE api_key = $1 LIMIT 1;

-- name: GetEnvironmentWithProjectAndOrg :one
SELECT
    re.*,
    p.id as project_id, p.slug as project_slug, p.name as project_name,
    o.id as org_id, o.slug as org_slug, o.title as org_title
FROM runtime_environments re
INNER JOIN projects p ON re.project_id = p.id
INNER JOIN organizations o ON re.organization_id = o.id
WHERE re.id = $1 LIMIT 1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetOrganization :one
SELECT * FROM organizations WHERE id = $1 LIMIT 1;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1 LIMIT 1;

// 服务特定查询示例 - SQLC 查询定义 (internal/apiauth/queries/)
-- name: FindPersonalAccessToken :one
SELECT * FROM personal_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

-- name: FindOrganizationAccessToken :one
SELECT * FROM organization_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

-- name: CreatePersonalAccessToken :one
INSERT INTO personal_access_tokens (user_id, token, name, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;
```

### 4. 服务实现

```go
// 主服务实现 - 使用混合架构
type apiAuthService struct {
    repo      APIAuthRepository       // 组合了 shared + apiauth 查询的仓库
    jwtSecret []byte
    logger    logger.Logger
}

func NewAPIAuthService(repo APIAuthRepository, jwtSecret string, logger logger.Logger) APIAuthService {
    return &apiAuthService{
        repo:      repo,
        jwtSecret: []byte(jwtSecret),
        logger:    logger,
    }
}

// 实际使用示例 - 在业务逻辑中组合使用共享和服务特定查询
func (s *apiAuthService) CreatePersonalToken(ctx context.Context, userID, orgID, name string) (*PersonalAccessToken, error) {
    // 1. 使用 shared.Queries 验证 User 和 Organization 存在
    user, err := s.repo.sharedQueries.GetUser(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }

    org, err := s.repo.sharedQueries.GetOrganization(ctx, orgID)
    if err != nil {
        return nil, fmt.Errorf("organization not found: %w", err)
    }

    // 2. 生成令牌
    token := generateSecureToken()

    // 3. 使用 apiauth.Queries 创建 PersonalAccessToken
    pat, err := s.repo.apiAuthQueries.CreatePersonalAccessToken(ctx, CreatePersonalAccessTokenParams{
        UserID:    user.ID,
        Token:     token,
        Name:      name,
        ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 90 天过期
    })

    return pat, err
}

// 实现核心认证逻辑
func (s *apiAuthService) AuthenticateAPIRequest(ctx context.Context, req *http.Request, opts *AuthOptions) (*AuthenticationResult, error) {
    // 1. 提取 Authorization 头
    authHeader := req.Header.Get("Authorization")
    if authHeader == "" {
        return nil, errors.New("missing authorization header")
    }

    // 2. 验证 Bearer 格式
    if !strings.HasPrefix(authHeader, "Bearer ") {
        return nil, errors.New("invalid authorization format")
    }

    // 3. 提取 API Key
    apiKey := strings.TrimPrefix(authHeader, "Bearer ")

    // 4. 委托给具体认证方法
    return s.authenticateAPIKey(ctx, apiKey, opts)
}

func (s *apiAuthService) authenticateAPIKey(ctx context.Context, apiKey string, opts *AuthOptions) (*AuthenticationResult, error) {
    // 1. 判断 API Key 类型
    keyType := s.getAPIKeyType(apiKey)

    // 2. 根据选项检查是否允许
    if !opts.AllowPublicKey && keyType == APIKeyTypePublic {
        return &AuthenticationResult{
            Success: false,
            Error:   "public API keys are not allowed for this request",
        }, nil
    }

    if !opts.AllowJWT && keyType == APIKeyTypePublicJWT {
        return &AuthenticationResult{
            Success: false,
            Error:   "public JWT API keys are not allowed for this request",
        }, nil
    }

    // 3. 根据类型进行相应的验证
    switch keyType {
    case APIKeyTypePublic:
        return s.authenticatePublicKey(ctx, apiKey, opts.BranchName)
    case APIKeyTypePrivate:
        return s.authenticatePrivateKey(ctx, apiKey, opts.BranchName)
    case APIKeyTypePublicJWT:
        return s.authenticateJWTKey(ctx, apiKey)
    default:
        return &AuthenticationResult{
            Success: false,
            Error:   "invalid API key format",
        }, nil
    }
}
```

### 5. JWT 处理实现

```go
// JWT 处理
func (s *apiAuthService) GenerateJWTToken(ctx context.Context, env *RuntimeEnvironment, payload map[string]interface{}, opts *JWTOptions) (string, error) {
    // 1. 构建声明
    claims := jwt.MapClaims{
        "sub": env.ID,
        "pub": true,
        "iat": time.Now().Unix(),
    }

    // 2. 添加自定义声明
    for k, v := range payload {
        claims[k] = v
    }

    // 3. 设置过期时间
    if opts != nil && opts.ExpirationTime != nil {
        switch exp := opts.ExpirationTime.(type) {
        case string:
            duration, err := time.ParseDuration(exp)
            if err != nil {
                return "", err
            }
            claims["exp"] = time.Now().Add(duration).Unix()
        case int64:
            claims["exp"] = exp
        }
    } else {
        // 默认 1 小时过期
        claims["exp"] = time.Now().Add(time.Hour).Unix()
    }

    // 4. 生成令牌
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

func (s *apiAuthService) authenticateJWTKey(ctx context.Context, jwtToken string) (*AuthenticationResult, error) {
    // 1. 解析 JWT
    token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.jwtSecret, nil
    })

    if err != nil {
        return &AuthenticationResult{
            Success: false,
            Error:   "invalid JWT token",
        }, nil
    }

    // 2. 验证声明
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        // 3. 获取环境信息
        envID, ok := claims["sub"].(string)
        if !ok {
            return &AuthenticationResult{
                Success: false,
                Error:   "invalid JWT claims",
            }, nil
        }

        // 4. 查找环境
        env, err := s.repo.FindEnvironmentByAPIKey(ctx, envID)
        if err != nil {
            return &AuthenticationResult{
                Success: false,
                Error:   "environment not found",
            }, nil
        }

        // 5. 构建结果
        result := &AuthenticationResult{
            Success:     true,
            APIKey:      jwtToken,
            Type:        APIKeyTypePublicJWT,
            Environment: &env.RuntimeEnvironment,
        }

        // 6. 解析额外声明
        if scopes, ok := claims["scopes"].([]interface{}); ok {
            for _, scope := range scopes {
                if s, ok := scope.(string); ok {
                    result.Scopes = append(result.Scopes, s)
                }
            }
        }

        if otu, ok := claims["otu"].(bool); ok {
            result.OneTimeUse = otu
        }

        if realtime, ok := claims["realtime"].(bool); ok {
            result.Realtime = realtime
        }

        return result, nil
    }

    return &AuthenticationResult{
        Success: false,
        Error:   "invalid JWT token",
    }, nil
}
```

## 🔧 实现步骤

### Phase 1: 数据库层 (1 天)

#### 1.1 创建共享数据库层 (0.5 天)

1. **创建共享实体表结构**

   ```sql
   -- db/migrations/002_shared_entities.sql

   -- Users 表
   CREATE TABLE users (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       email VARCHAR(255) UNIQUE NOT NULL,
       name VARCHAR(255),
       avatar_url TEXT,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
   );

   -- Organizations 表
   CREATE TABLE organizations (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       title VARCHAR(255) NOT NULL,
       slug VARCHAR(100) UNIQUE NOT NULL,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
   );

   -- Projects 表
   CREATE TABLE projects (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       name VARCHAR(255) NOT NULL,
       slug VARCHAR(100) NOT NULL,
       organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       UNIQUE(organization_id, slug)
   );

   -- RuntimeEnvironments 表
   CREATE TABLE runtime_environments (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       slug VARCHAR(100) NOT NULL,
       api_key VARCHAR(255) UNIQUE NOT NULL,
       type VARCHAR(50) NOT NULL CHECK (type IN ('PRODUCTION', 'STAGING', 'DEVELOPMENT', 'PREVIEW')),
       organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
       project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
       org_member_id UUID,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       UNIQUE(project_id, slug)
   );

   -- 索引
   CREATE INDEX idx_runtime_environments_api_key ON runtime_environments(api_key);
   CREATE INDEX idx_runtime_environments_org_id ON runtime_environments(organization_id);
   CREATE INDEX idx_runtime_environments_project_id ON runtime_environments(project_id);
   ```

2. **创建共享查询文件**

   ```sql
   -- internal/shared/queries/users.sql
   -- name: GetUser :one
   SELECT * FROM users WHERE id = $1 LIMIT 1;

   -- name: FindUserByEmail :one
   SELECT * FROM users WHERE email = $1 LIMIT 1;

   -- internal/shared/queries/organizations.sql
   -- name: GetOrganization :one
   SELECT * FROM organizations WHERE id = $1 LIMIT 1;

   -- name: FindOrganizationBySlug :one
   SELECT * FROM organizations WHERE slug = $1 LIMIT 1;

   -- internal/shared/queries/projects.sql
   -- name: GetProject :one
   SELECT * FROM projects WHERE id = $1 LIMIT 1;

   -- name: FindProjectBySlug :one
   SELECT * FROM projects WHERE organization_id = $1 AND slug = $2 LIMIT 1;

   -- internal/shared/queries/runtime_environments.sql
   -- name: FindRuntimeEnvironmentByAPIKey :one
   SELECT * FROM runtime_environments WHERE api_key = $1 LIMIT 1;

   -- name: FindRuntimeEnvironmentByPublicAPIKey :one
   SELECT * FROM runtime_environments WHERE api_key = $1 AND type != 'PRODUCTION' LIMIT 1;

   -- name: GetEnvironmentWithProjectAndOrg :one
   SELECT
       re.*,
       p.id as project_id, p.slug as project_slug, p.name as project_name,
       o.id as org_id, o.slug as org_slug, o.title as org_title
   FROM runtime_environments re
   INNER JOIN projects p ON re.project_id = p.id
   INNER JOIN organizations o ON re.organization_id = o.id
   WHERE re.id = $1 LIMIT 1;
   ```

#### 1.2 创建 ApiAuth 特定表结构 (0.5 天)

3. **创建 ApiAuth 服务表结构**

   ```sql
   -- db/migrations/003_apiauth_service.sql

   -- Personal Access Tokens 表
   CREATE TABLE personal_access_tokens (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
       token VARCHAR(255) UNIQUE NOT NULL,
       name VARCHAR(255) NOT NULL,
       expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
       last_used_at TIMESTAMP WITH TIME ZONE,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
   );

   -- Organization Access Tokens 表
   CREATE TABLE organization_access_tokens (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
       token VARCHAR(255) UNIQUE NOT NULL,
       name VARCHAR(255) NOT NULL,
       expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
       last_used_at TIMESTAMP WITH TIME ZONE,
       created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
       updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
   );

   -- 索引
   CREATE INDEX idx_personal_access_tokens_token ON personal_access_tokens(token);
   CREATE INDEX idx_personal_access_tokens_user_id ON personal_access_tokens(user_id);
   CREATE INDEX idx_organization_access_tokens_token ON organization_access_tokens(token);
   CREATE INDEX idx_organization_access_tokens_org_id ON organization_access_tokens(organization_id);
   ```

4. **创建 ApiAuth 查询文件**

   ```sql
   -- internal/apiauth/queries/personal_tokens.sql
   -- name: FindPersonalAccessToken :one
   SELECT * FROM personal_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

   -- name: CreatePersonalAccessToken :one
   INSERT INTO personal_access_tokens (user_id, token, name, expires_at)
   VALUES ($1, $2, $3, $4)
   RETURNING *;

   -- name: UpdatePersonalTokenLastUsed :exec
   UPDATE personal_access_tokens SET last_used_at = NOW() WHERE id = $1;

   -- internal/apiauth/queries/org_tokens.sql
   -- name: FindOrganizationAccessToken :one
   SELECT * FROM organization_access_tokens WHERE token = $1 AND expires_at > NOW() LIMIT 1;

   -- name: CreateOrganizationAccessToken :one
   INSERT INTO organization_access_tokens (organization_id, token, name, expires_at)
   VALUES ($1, $2, $3, $4)
   RETURNING *;

   -- name: UpdateOrgTokenLastUsed :exec
   UPDATE organization_access_tokens SET last_used_at = NOW() WHERE id = $1;
   ```

#### 1.3 更新 SQLC 配置 (0.1 天)

5. **更新 sqlc.yaml 配置**

   ```yaml
   version: '2'
   sql:
     # 共享数据库层
     - name: shared
       engine: 'postgresql'
       queries: './internal/shared/queries'
       schema: './db/migrations'
       gen:
         go:
           out: './internal/shared'
           package: 'shared'
           sql_package: 'pgx/v5'
           emit_json_tags: true
           emit_interface: true
           emit_prepared_queries: false
           emit_exact_table_names: true
           omit_unused_structs: true

     # 现有服务
     - name: secretstore
       engine: 'postgresql'
       queries: './internal/secretstore/queries'
       schema: './db/migrations'
       gen:
         go:
           out: './internal/secretstore'
           package: 'secretstore'
           sql_package: 'pgx/v5'
           emit_json_tags: true
           emit_interface: true
           emit_prepared_queries: false
           emit_exact_table_names: true
           omit_unused_structs: true

     # ApiAuth 服务
     - name: apiauth
       engine: 'postgresql'
       queries: './internal/apiauth/queries'
       schema: './db/migrations'
       gen:
         go:
           out: './internal/apiauth'
           package: 'apiauth'
           sql_package: 'pgx/v5'
           emit_json_tags: true
           emit_interface: true
           emit_prepared_queries: false
           emit_exact_table_names: true
           omit_unused_structs: true
   ```

6. **生成 SQLC 代码**
   ```bash
   cd kongflow/backend
   sqlc generate
   ```

### Phase 2: 核心认证逻辑 (1.5 天)

1. **API Key 认证实现**

   - 类型识别和验证
   - 公开/私有密钥处理
   - 分支名称支持

2. **Personal/Organization Token 认证**
   - Bearer Token 解析
   - 令牌验证和用户/组织信息获取

### Phase 3: JWT 功能 (1 天)

1. **JWT 生成和验证**

   - 令牌生成逻辑
   - 签名验证
   - 声明解析

2. **高级功能**
   - 一次性使用标记
   - 实时功能支持
   - 自定义声明处理

### Phase 4: 统一认证入口 (0.5 天)

1. **AuthenticateRequest 实现**

   - 多种认证方式支持
   - 类型安全的配置
   - 错误处理标准化

2. **中间件实现**
   - HTTP 中间件封装
   - 请求上下文注入

### Phase 5: 测试和文档 (1 天)

1. **单元测试**

   - 各种认证方式测试
   - 错误场景覆盖
   - Mock 数据库操作

2. **集成测试**
   - 端到端认证流程
   - 多环境配置测试

## 📊 与 trigger.dev 对齐检查

### 功能对齐

- ✅ authenticateApiRequest() → AuthenticateAPIRequest()
- ✅ authenticateApiRequestWithFailure() → 内置错误处理
- ✅ authenticateRequest() → AuthenticateRequest()
- ✅ authenticateAuthorizationHeader() → AuthenticateAuthorizationHeader()
- ✅ generateJWTTokenForEnvironment() → GenerateJWTToken()

### 行为对齐

- ✅ 相同的 API Key 类型识别逻辑
- ✅ 相同的 JWT 生成和验证算法
- ✅ 相同的错误消息和状态码
- ✅ 相同的认证选项和配置

### 数据结构对齐

- ✅ RuntimeEnvironment 字段完全匹配
- ✅ AuthenticationResult 结构对应
- ✅ JWT 声明格式一致

## 🚀 预期产出

1. **代码产出**

   - 完整的 apiauth 服务包
   - 90%+ 的单元测试覆盖率
   - 完整的集成测试套件

2. **集成点**

   - 与数据库层的 SQLC 集成
   - HTTP 中间件支持
   - 为其他服务提供认证基础

3. **配置和部署**
   - 环境变量配置
   - JWT 密钥管理
   - 性能优化建议

## ⚠️ 注意事项

1. **安全考虑**

   - JWT 密钥安全存储
   - API Key 的安全处理
   - 令牌过期和撤销机制

2. **性能考虑**

   - 数据库查询优化
   - 缓存策略（如需要）
   - 并发安全

3. **向后兼容**
   - 与已有认证系统的平滑过渡
   - API 版本管理
   - 渐进式迁移支持

## 📈 成功标准

- [ ] 所有 trigger.dev apiAuth.server.ts 功能 100% 实现
- [ ] 认证性能满足生产环境要求 (< 50ms 响应时间)
- [ ] 单元测试覆盖率 ≥ 90%
- [ ] 集成测试通过率 100%
- [ ] 安全审计通过
- [ ] 文档完整且准确

这个 API Authentication Service 是整个系统的安全基础，迁移完成后将为后续所有需要认证的服务提供统一的认证能力。
