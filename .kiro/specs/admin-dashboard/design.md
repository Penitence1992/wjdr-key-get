# Design Document: Admin Dashboard

## Overview

管理后台系统为礼品码兑换系统提供Web管理界面和RESTful API。系统采用前后端分离架构，后端使用Go语言和Gin框架提供API服务，前端使用HTML/JavaScript构建单页应用。

核心功能包括：
- 基于JWT的管理员认证和会话管理
- 用户信息的CRUD操作
- 用户兑换历史查询
- 后台任务监控
- 兑换码管理

系统设计遵循现有代码库的架构模式，复用现有的中间件、日志、存储等基础设施。

## Architecture

### 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                               │
│                   (Admin Dashboard UI)                       │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    Gin HTTP Server                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Middleware Stack                        │   │
│  │  - RequestID                                         │   │
│  │  - Recovery                                          │   │
│  │  - Logger                                            │   │
│  │  - CORS                                              │   │
│  │  - RateLimit                                         │   │
│  │  - Auth (new)                                        │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Admin Handlers                          │   │
│  │  - Login                                             │   │
│  │  - ListUsers                                         │   │
│  │  - AddUser                                           │   │
│  │  - GetUserGiftCodes                                  │   │
│  │  - ListTasks                                         │   │
│  │  - AddGiftCode                                       │   │
│  └──────────────────────────────────────────────────────┘   │
│                         │                                    │
│                         ▼                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Auth Service                            │   │
│  │  - ValidateCredentials                               │   │
│  │  - GenerateToken                                     │   │
│  │  - ValidateToken                                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                         │                                    │
│                         ▼                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Repository                              │   │
│  │  - ListUsers                                         │   │
│  │  - SaveUser                                          │   │
│  │  - ListGiftCodesByFID                                │   │
│  │  - ListPendingTasks                                  │   │
│  │  - CreateTask                                        │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   SQLite DB  │
                  └──────────────┘
```

### 认证流程

```
┌────────┐                ┌────────────┐              ┌──────────┐
│ Client │                │   Server   │              │  Config  │
└───┬────┘                └─────┬──────┘              └────┬─────┘
    │                           │                          │
    │  POST /api/admin/login    │                          │
    │  {username, password}     │                          │
    ├──────────────────────────>│                          │
    │                           │                          │
    │                           │  Load admin credentials  │
    │                           ├─────────────────────────>│
    │                           │                          │
    │                           │<─────────────────────────┤
    │                           │                          │
    │                           │  Validate credentials    │
    │                           │  Generate JWT token      │
    │                           │                          │
    │  200 OK                   │                          │
    │  {token, expires_at}      │                          │
    │<──────────────────────────┤                          │
    │                           │                          │
    │  GET /api/admin/users     │                          │
    │  Authorization: Bearer    │                          │
    │  <token>                  │                          │
    ├──────────────────────────>│                          │
    │                           │                          │
    │                           │  Validate JWT token      │
    │                           │                          │
    │  200 OK                   │                          │
    │  {users: [...]}           │                          │
    │<──────────────────────────┤                          │
    │                           │                          │
```

## Components and Interfaces

### 1. Authentication Service

认证服务负责管理员身份验证和JWT令牌管理。

```go
// AuthService 认证服务接口
type AuthService interface {
    // ValidateCredentials 验证管理员凭证
    ValidateCredentials(username, password string) error
    
    // GenerateToken 生成JWT令牌
    GenerateToken(username string) (token string, expiresAt time.Time, err error)
    
    // ValidateToken 验证JWT令牌
    ValidateToken(token string) (*Claims, error)
}

// Claims JWT声明
type Claims struct {
    Username string `json:"username"`
    jwt.RegisteredClaims
}

// AdminConfig 管理员配置
type AdminConfig struct {
    Username      string        `yaml:"username"`
    PasswordHash  string        `yaml:"password_hash"`
    TokenSecret   string        `yaml:"token_secret"`
    TokenDuration time.Duration `yaml:"token_duration"`
}
```

实现细节：
- 使用bcrypt进行密码哈希验证
- 使用JWT (github.com/golang-jwt/jwt/v5) 生成和验证令牌
- 令牌默认有效期为24小时
- 密钥从配置文件或环境变量加载

### 2. Authentication Middleware

认证中间件拦截受保护的API请求，验证JWT令牌。

```go
// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService AuthService, logger *logrus.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从Authorization头提取token
        authHeader := c.GetHeader("Authorization")
        
        // 验证Bearer格式
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(401, ErrorResponse("UNAUTHORIZED", "Missing or invalid authorization header"))
            c.Abort()
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        
        // 验证token
        claims, err := authService.ValidateToken(token)
        if err != nil {
            c.JSON(401, ErrorResponse("UNAUTHORIZED", "Invalid or expired token"))
            c.Abort()
            return
        }
        
        // 将用户信息存入上下文
        c.Set("admin_username", claims.Username)
        c.Next()
    }
}
```

### 3. Admin Handlers

管理后台API处理器。

```go
// AdminHandlers 管理后台处理器
type AdminHandlers struct {
    authService AuthService
    repository  storage.Repository
    logger      *logrus.Logger
}

// LoginRequest 登录请求
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
    Token     string    `json:"token"`
    ExpiresAt time.Time `json:"expires_at"`
}

// Login 管理员登录
func (h *AdminHandlers) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
        return
    }
    
    // 验证凭证
    if err := h.authService.ValidateCredentials(req.Username, req.Password); err != nil {
        c.JSON(401, ErrorResponse("INVALID_CREDENTIALS", "Invalid username or password"))
        return
    }
    
    // 生成token
    token, expiresAt, err := h.authService.GenerateToken(req.Username)
    if err != nil {
        c.JSON(500, ErrorResponse("TOKEN_GENERATION_FAILED", "Failed to generate token"))
        return
    }
    
    c.JSON(200, SuccessResponse(LoginResponse{
        Token:     token,
        ExpiresAt: expiresAt,
    }))
}

// ListUsers 获取用户列表
func (h *AdminHandlers) ListUsers(c *gin.Context) {
    ctx := c.Request.Context()
    
    users, err := h.repository.ListUsers(ctx)
    if err != nil {
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch users"))
        return
    }
    
    c.JSON(200, SuccessResponse(gin.H{"users": users}))
}

// GetUserGiftCodes 获取用户的兑换记录
func (h *AdminHandlers) GetUserGiftCodes(c *gin.Context) {
    fid := c.Param("fid")
    ctx := c.Request.Context()
    
    records, err := h.repository.ListGiftCodesByFID(ctx, fid)
    if err != nil {
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch gift codes"))
        return
    }
    
    c.JSON(200, SuccessResponse(gin.H{"records": records}))
}

// ListTasks 获取任务列表
func (h *AdminHandlers) ListTasks(c *gin.Context) {
    ctx := c.Request.Context()
    
    tasks, err := h.repository.ListPendingTasks(ctx)
    if err != nil {
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch tasks"))
        return
    }
    
    c.JSON(200, SuccessResponse(gin.H{"tasks": tasks}))
}

// AddGiftCodeRequest 添加兑换码请求
type AddGiftCodeRequest struct {
    Code string `json:"code" binding:"required"`
}

// AddGiftCode 添加兑换码
func (h *AdminHandlers) AddGiftCode(c *gin.Context) {
    var req AddGiftCodeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
        return
    }
    
    ctx := c.Request.Context()
    
    if err := h.repository.CreateTask(ctx, req.Code); err != nil {
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to create task"))
        return
    }
    
    c.JSON(200, SuccessResponse(gin.H{
        "message": "Gift code task created successfully",
        "code":    req.Code,
    }))
}

// AddUserRequest 添加用户请求
type AddUserRequest struct {
    FID         string `json:"fid" binding:"required"`
    Nickname    string `json:"nickname"`
    KID         int    `json:"kid"`
    AvatarImage string `json:"avatar_image"`
}

// AddUser 添加用户
func (h *AdminHandlers) AddUser(c *gin.Context) {
    var req AddUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
        return
    }
    
    ctx := c.Request.Context()
    
    user := &storage.User{
        FID:         req.FID,
        Nickname:    req.Nickname,
        KID:         req.KID,
        AvatarImage: req.AvatarImage,
    }
    
    if err := h.repository.SaveUser(ctx, user); err != nil {
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to save user"))
        return
    }
    
    c.JSON(200, SuccessResponse(gin.H{
        "message": "User added successfully",
        "fid":     req.FID,
    }))
}
```

### 4. Frontend UI

前端使用HTML/JavaScript构建单页应用，提供以下页面：

- **登录页面** (`/admin/login.html`)
- **主控制面板** (`/admin/dashboard.html`)
  - 用户列表
  - 用户详情（兑换记录）
  - 任务列表
  - 添加兑换码表单

前端技术栈：
- 原生JavaScript (ES6+)
- Fetch API用于HTTP请求
- LocalStorage存储JWT令牌
- 简单的CSS样式

## Data Models

### Configuration Extension

扩展现有的配置结构以支持管理员认证：

```go
// Config 扩展
type Config struct {
    // ... 现有字段 ...
    Admin AdminConfig `yaml:"admin"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
    Username      string        `yaml:"username"`
    PasswordHash  string        `yaml:"password_hash"`
    TokenSecret   string        `yaml:"token_secret"`
    TokenDuration time.Duration `yaml:"token_duration"`
}
```

配置文件示例：

```yaml
admin:
  username: admin
  password_hash: $2a$10$... # bcrypt hash
  token_secret: your-secret-key-change-in-production
  token_duration: 24h
```

环境变量覆盖：
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD_HASH`
- `ADMIN_TOKEN_SECRET`
- `ADMIN_TOKEN_DURATION`

### API Routes

```
POST   /api/admin/login              # 管理员登录
GET    /api/admin/users              # 获取用户列表 (需认证)
POST   /api/admin/users              # 添加用户 (需认证)
GET    /api/admin/users/:fid/codes   # 获取用户兑换记录 (需认证)
GET    /api/admin/tasks              # 获取任务列表 (需认证)
POST   /api/admin/tasks              # 添加兑换码任务 (需认证)
```

静态文件路由：
```
GET    /admin/                       # 重定向到登录页或控制面板
GET    /admin/login.html             # 登录页面
GET    /admin/dashboard.html         # 控制面板
GET    /admin/assets/*               # 静态资源
```


## Correctness Properties

属性（Property）是系统在所有有效执行中应该保持为真的特征或行为——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。

### Property 1: 有效凭证认证成功
*对于任何*有效的管理员用户名和密码组合，当提交登录请求时，系统应该验证凭证成功并生成有效的JWT令牌
**Validates: Requirements 1.1, 1.4**

### Property 2: 无效凭证认证失败
*对于任何*无效的凭证（错误的用户名或密码），当提交登录请求时，系统应该拒绝访问并返回认证错误
**Validates: Requirements 1.2**

### Property 3: 过期令牌被拒绝
*对于任何*过期的JWT令牌，当用于访问受保护资源时，系统应该返回401未授权错误
**Validates: Requirements 1.3, 6.3**

### Property 4: 认证令牌往返一致性
*对于任何*成功的登录，生成的JWT令牌应该能够被验证，并且解析出的声明应该包含正确的用户名
**Validates: Requirements 1.4, 6.1**

### Property 5: 未认证请求被拒绝
*对于任何*受保护的API端点，当请求不包含有效的认证令牌时，系统应该返回401未授权错误
**Validates: Requirements 1.5, 6.2**

### Property 6: 用户列表完整性
*对于任何*数据库中的用户集合，当查询用户列表时，返回的用户数量应该等于数据库中的用户总数
**Validates: Requirements 2.1**

### Property 7: API响应包含必需字段
*对于任何*API响应（用户、兑换记录、任务），返回的JSON数据应该包含规范中定义的所有必需字段
**Validates: Requirements 2.2, 3.2, 4.2**

### Property 8: 用户添加幂等性
*对于任何*用户数据，当使用相同的FID添加两次时，数据库中应该只有一条记录，且信息应该被更新为最新的值
**Validates: Requirements 2.5**

### Property 9: 用户兑换记录查询正确性
*对于任何*用户FID和该用户的兑换记录集合，当查询该用户的兑换记录时，返回的所有记录的FID应该匹配查询的FID
**Validates: Requirements 3.1**

### Property 10: 兑换记录时间排序
*对于任何*用户的兑换记录列表，返回的记录应该按创建时间降序排列（最新的在前）
**Validates: Requirements 3.5**

### Property 11: 任务列表过滤正确性
*对于任何*数据库中的任务集合，当查询待处理任务时，返回的所有任务的all_done字段应该为false
**Validates: Requirements 4.1**

### Property 12: 已完成任务包含完成时间
*对于任何*已完成的任务（all_done为true），任务记录应该包含非空的completed_at时间戳
**Validates: Requirements 4.4**

### Property 13: 失败任务包含错误信息
*对于任何*重试次数大于0的任务，任务记录应该包含非空的last_error字段
**Validates: Requirements 4.5**

### Property 14: 有效兑换码创建任务成功
*对于任何*非空的兑换码字符串，当添加兑换码时，系统应该创建一个新的待处理任务
**Validates: Requirements 5.1, 5.4**

### Property 15: 无效兑换码被拒绝
*对于任何*空字符串或仅包含空白字符的兑换码，当尝试添加时，系统应该返回验证错误
**Validates: Requirements 5.2**

### Property 16: 格式错误的令牌返回400
*对于任何*格式错误的Authorization头（不是"Bearer <token>"格式），系统应该返回400错误请求
**Validates: Requirements 6.4**

### Property 17: 配置加载正确性
*对于任何*有效的配置文件，当系统启动时，加载的管理员配置应该与文件中的配置匹配
**Validates: Requirements 8.1**

### Property 18: 环境变量优先级
*对于任何*同时存在于配置文件和环境变量中的配置项，系统应该使用环境变量的值
**Validates: Requirements 8.3**

## Error Handling

### Authentication Errors

| Error Code | HTTP Status | Description | User Action |
|------------|-------------|-------------|-------------|
| INVALID_CREDENTIALS | 401 | 用户名或密码错误 | 检查凭证并重试 |
| UNAUTHORIZED | 401 | 缺少或无效的认证令牌 | 重新登录获取新令牌 |
| TOKEN_EXPIRED | 401 | 令牌已过期 | 重新登录 |
| TOKEN_GENERATION_FAILED | 500 | 令牌生成失败 | 联系管理员 |

### Validation Errors

| Error Code | HTTP Status | Description | User Action |
|------------|-------------|-------------|-------------|
| VALIDATION_ERROR | 400 | 请求参数验证失败 | 检查请求格式和参数 |
| UNSUPPORTED_MEDIA_TYPE | 415 | 不支持的Content-Type | 使用application/json |
| MISSING_REQUIRED_FIELD | 400 | 缺少必需字段 | 提供所有必需字段 |

### Database Errors

| Error Code | HTTP Status | Description | User Action |
|------------|-------------|-------------|-------------|
| DATABASE_ERROR | 500 | 数据库操作失败 | 重试或联系管理员 |
| RECORD_NOT_FOUND | 404 | 记录不存在 | 检查查询参数 |

### Rate Limiting

| Error Code | HTTP Status | Description | User Action |
|------------|-------------|-------------|-------------|
| RATE_LIMIT_EXCEEDED | 429 | 请求频率超限 | 稍后重试 |

### Error Response Format

所有错误响应遵循统一格式：

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message"
  }
}
```

### Error Logging

所有错误应该记录以下信息：
- Request ID（用于追踪）
- Timestamp
- Error type and message
- Stack trace（对于500错误）
- User context（如果已认证）
- Request details（method, path, parameters）

敏感信息（密码、令牌）不应该记录到日志中。

## Testing Strategy

### 测试方法

系统采用双重测试策略，结合单元测试和基于属性的测试：

**单元测试**：
- 验证特定示例和边缘情况
- 测试错误条件和异常处理
- 测试组件之间的集成点
- 使用Go的标准testing包

**基于属性的测试**：
- 验证跨所有输入的通用属性
- 通过随机化提供全面的输入覆盖
- 每个属性测试最少运行100次迭代
- 使用gopter库（Go的属性测试框架）

两种测试方法是互补的：单元测试捕获具体的错误，属性测试验证通用的正确性。

### 测试库选择

- **单元测试**: Go标准库 `testing` + `testify/assert`
- **属性测试**: `github.com/leanovate/gopter`
- **HTTP测试**: `httptest` 包
- **Mock**: `github.com/stretchr/testify/mock`

### 属性测试配置

每个属性测试必须：
1. 运行最少100次迭代（由于随机化）
2. 使用注释引用设计文档中的属性
3. 标签格式：`// Feature: admin-dashboard, Property N: [property text]`
4. 每个正确性属性由单个属性测试实现

示例：

```go
// Feature: admin-dashboard, Property 1: 有效凭证认证成功
func TestProperty_ValidCredentialsAuthenticate(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("valid credentials should authenticate successfully", 
        prop.ForAll(
            func(username, password string) bool {
                // 创建有效凭证
                hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
                authService := NewAuthService(username, string(hash), "secret")
                
                // 验证凭证
                err := authService.ValidateCredentials(username, password)
                if err != nil {
                    return false
                }
                
                // 生成令牌
                token, _, err := authService.GenerateToken(username)
                return err == nil && token != ""
            },
            gen.AlphaString(),
            gen.AlphaString(),
        ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### 测试覆盖范围

**认证服务**：
- 属性测试：凭证验证、令牌生成和验证
- 单元测试：边缘情况（空密码、特殊字符）、错误处理

**认证中间件**：
- 属性测试：令牌验证、未认证请求拒绝
- 单元测试：Authorization头格式、令牌提取

**Admin Handlers**：
- 属性测试：数据完整性、幂等性、排序
- 单元测试：请求验证、错误响应、边缘情况

**配置加载**：
- 属性测试：配置解析、环境变量优先级
- 单元测试：默认值、验证规则

### 集成测试

使用`httptest`包测试完整的HTTP请求流程：
- 登录流程（获取令牌）
- 使用令牌访问受保护资源
- 错误场景（无效令牌、过期令牌）
- CORS和中间件集成

### 测试数据管理

- 使用内存SQLite数据库（`:memory:`）进行测试
- 每个测试前重置数据库状态
- 使用测试夹具（fixtures）提供一致的测试数据
- 属性测试使用生成器创建随机但有效的测试数据

### 性能测试

虽然不是主要关注点，但应该进行基本的性能测试：
- 令牌生成和验证的基准测试
- 数据库查询性能（大量用户/任务）
- 并发请求处理

### 测试执行

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行属性测试（更多迭代）
go test -v -run TestProperty

# 运行基准测试
go test -bench=.
```
