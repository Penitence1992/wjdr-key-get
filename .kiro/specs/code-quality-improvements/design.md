# Design Document: Code Quality Improvements

## Overview

本设计文档描述了对现有礼品码兑换系统的全面改进方案。改进重点包括安全性增强、错误处理优化、资源管理改进、并发安全保障、代码结构重组、可测试性提升、配置管理标准化、日志系统完善、API 设计规范化、数据库操作优化、性能提升和可观测性增强。

改进将采用渐进式重构策略，确保系统在改进过程中保持稳定运行。所有改进都将遵循 Go 语言最佳实践和工业标准。

## Architecture

### 整体架构改进

当前系统采用单体架构，包含以下主要组件：
- HTTP API Server (Gin框架)
- 定时任务调度器
- SQLite 数据存储
- 验证码识别客户端（阿里云/腾讯云）
- HTTP 客户端工具

改进后的架构将保持单体结构，但增强模块化和解耦：

```
┌─────────────────────────────────────────────────────────┐
│                     Application Layer                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ HTTP Handlers│  │  Job Scheduler│  │  CLI Commands│  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
└─────────┼──────────────────┼──────────────────┼─────────┘
          │                  │                  │
┌─────────┼──────────────────┼──────────────────┼─────────┐
│         │      Business Logic Layer           │         │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐  │
│  │ Gift Service │  │  Task Service │  │  User Service│  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
└─────────┼──────────────────┼──────────────────┼─────────┘
          │                  │                  │
┌─────────┼──────────────────┼──────────────────┼─────────┐
│         │      Infrastructure Layer           │         │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐  │
│  │   Storage    │  │    Captcha   │  │  HTTP Client │  │
│  │  Repository  │  │    Client    │  │    Factory   │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
          │                  │                  │
┌─────────┼──────────────────┼──────────────────┼─────────┐
│         │      External Dependencies          │         │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐  │
│  │   SQLite DB  │  │  Ali/TC OCR  │  │  Remote APIs │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 关键设计原则

1. **依赖注入**: 所有组件通过构造函数注入依赖
2. **接口隔离**: 定义清晰的接口边界
3. **单一职责**: 每个模块专注于单一功能
4. **配置外部化**: 所有配置通过配置文件或环境变量管理
5. **错误透明**: 错误信息包含完整上下文

## Components and Interfaces

### 1. Configuration Management (config package)

**职责**: 统一管理所有系统配置

```go
// Config 主配置结构
type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Captcha   CaptchaConfig
    Job       JobConfig
    Logging   LoggingConfig
    Security  SecurityConfig
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
    Host         string
    Port         int
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
    Path            string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

// CaptchaConfig 验证码服务配置
type CaptchaConfig struct {
    Providers []CaptchaProvider
}

type CaptchaProvider struct {
    Type      string // "ali" or "tencent"
    AccessKey string
    SecretKey string
}

// LoadConfig 从文件和环境变量加载配置
func LoadConfig(configPath string) (*Config, error)

// Validate 验证配置有效性
func (c *Config) Validate() error
```

### 2. Error Handling (errors package)

**职责**: 提供结构化错误类型和错误包装

```go
// AppError 应用错误类型
type AppError struct {
    Code    string
    Message string
    Err     error
    Context map[string]interface{}
}

func (e *AppError) Error() string
func (e *AppError) Unwrap() error

// 预定义错误代码
const (
    ErrCodeDatabase      = "DATABASE_ERROR"
    ErrCodeCaptcha       = "CAPTCHA_ERROR"
    ErrCodeValidation    = "VALIDATION_ERROR"
    ErrCodeNotFound      = "NOT_FOUND"
    ErrCodeAlreadyExists = "ALREADY_EXISTS"
    ErrCodeExternal      = "EXTERNAL_API_ERROR"
)

// 错误构造函数
func NewDatabaseError(op string, err error) *AppError
func NewValidationError(field string, reason string) *AppError
func NewNotFoundError(resource string, id string) *AppError
```

### 3. Storage Layer Refactoring

**职责**: 提供数据访问接口和实现

```go
// Repository 数据仓库接口
type Repository interface {
    // Gift Code operations
    SaveGiftCode(ctx context.Context, fid, code string) error
    IsGiftCodeReceived(ctx context.Context, fid, code string) (bool, error)
    
    // User operations
    SaveUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, fid string) (*User, error)
    ListUsers(ctx context.Context) ([]*User, error)
    
    // Task operations
    CreateTask(ctx context.Context, code string) error
    ListPendingTasks(ctx context.Context) ([]*Task, error)
    MarkTaskComplete(ctx context.Context, code string) error
    
    // Transaction support
    WithTransaction(ctx context.Context, fn func(Repository) error) error
    
    // Health check
    Ping(ctx context.Context) error
    Close() error
}

// User 用户模型
type User struct {
    FID         string
    Nickname    string
    KID         int
    AvatarImage string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Task 任务模型
type Task struct {
    Code      string
    AllDone   bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 4. HTTP Client Factory

**职责**: 创建和管理 HTTP 客户端

```go
// ClientFactory HTTP客户端工厂
type ClientFactory struct {
    config ClientConfig
}

type ClientConfig struct {
    Timeout         time.Duration
    MaxIdleConns    int
    IdleConnTimeout time.Duration
    TLSTimeout      time.Duration
}

func NewClientFactory(config ClientConfig) *ClientFactory
func (f *ClientFactory) NewClient() *http.Client
```

### 5. Captcha Client Pool

**职责**: 管理验证码识别客户端池

```go
// CaptchaPool 验证码客户端池
type CaptchaPool struct {
    clients []captcha.RemoteClient
    idx     atomic.Uint32
}

func NewCaptchaPool(configs []CaptchaConfig) (*CaptchaPool, error)
func (p *CaptchaPool) Get() captcha.RemoteClient
```

### 6. Gift Service

**职责**: 礼品码兑换业务逻辑

```go
// GiftService 礼品码服务
type GiftService struct {
    repo        Repository
    captchaPool *CaptchaPool
    httpClient  *http.Client
    logger      *logrus.Logger
}

func NewGiftService(repo Repository, pool *CaptchaPool, client *http.Client, logger *logrus.Logger) *GiftService

// RedeemGiftCode 兑换礼品码
func (s *GiftService) RedeemGiftCode(ctx context.Context, fid, code string) (*RedeemResult, error)

// BatchRedeemGiftCode 批量兑换礼品码
func (s *GiftService) BatchRedeemGiftCode(ctx context.Context, fids []string, code string) ([]*RedeemResult, error)
```

### 7. Task Scheduler Refactoring

**职责**: 定时任务调度和执行

```go
// Scheduler 任务调度器
type Scheduler struct {
    jobs   []Job
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}

// Job 任务接口
type Job interface {
    Name() string
    Run(ctx context.Context) error
    Schedule() time.Duration // 返回执行间隔
}

func NewScheduler() *Scheduler
func (s *Scheduler) AddJob(job Job)
func (s *Scheduler) Start() error
func (s *Scheduler) Stop() error
```

### 8. API Response Standardization

**职责**: 统一 API 响应格式

```go
// Response 标准响应格式
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// 响应构造函数
func SuccessResponse(data interface{}) Response
func ErrorResponse(code, message string) Response
```

### 9. Middleware Stack

**职责**: HTTP 中间件

```go
// RequestID 中间件 - 为每个请求生成唯一ID
func RequestIDMiddleware() gin.HandlerFunc

// Logger 中间件 - 结构化日志
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc

// Recovery 中间件 - panic恢复
func RecoveryMiddleware(logger *logrus.Logger) gin.HandlerFunc

// RateLimit 中间件 - 限流
func RateLimitMiddleware(rate int, burst int) gin.HandlerFunc

// Validation 中间件 - 请求验证
func ValidationMiddleware() gin.HandlerFunc
```

### 10. Monitoring and Health

**职责**: 健康检查和指标暴露

```go
// HealthChecker 健康检查器
type HealthChecker struct {
    repo   Repository
    checks []HealthCheck
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
}

// HealthStatus 健康状态
type HealthStatus struct {
    Status string                 `json:"status"` // "healthy" or "unhealthy"
    Checks map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
}

func (h *HealthChecker) CheckHealth(ctx context.Context) HealthStatus
```

## Data Models

### Database Schema Improvements

```sql
-- 用户表（改进）
CREATE TABLE IF NOT EXISTS users (
    fid TEXT PRIMARY KEY,
    nickname TEXT NOT NULL DEFAULT '',
    kid INTEGER NOT NULL DEFAULT 0,
    avatar_image TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 礼品码记录表（改进）
CREATE TABLE IF NOT EXISTS gift_code_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fid TEXT NOT NULL,
    code TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'success', -- success, failed, duplicate
    message TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(fid, code)
);

CREATE INDEX IF NOT EXISTS idx_gift_code_records_fid ON gift_code_records(fid);
CREATE INDEX IF NOT EXISTS idx_gift_code_records_code ON gift_code_records(code);
CREATE INDEX IF NOT EXISTS idx_gift_code_records_created_at ON gift_code_records(created_at);

-- 任务表（改进）
CREATE TABLE IF NOT EXISTS tasks (
    code TEXT PRIMARY KEY,
    all_done INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_tasks_all_done ON tasks(all_done);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
```

### Migration Strategy

使用 golang-migrate 或类似工具管理数据库迁移：

```
migrations/
  ├── 000001_initial_schema.up.sql
  ├── 000001_initial_schema.down.sql
  ├── 000002_add_indexes.up.sql
  ├── 000002_add_indexes.down.sql
  └── ...
```

## Correctness Properties

*属性（Property）是关于系统行为的特征或规则，应该在所有有效执行中保持为真。属性是人类可读规范和机器可验证正确性保证之间的桥梁。*

### Property 1: No Hardcoded Credentials
*For any* source code file in the system, there must be no hardcoded API keys, tokens, passwords, or other credentials present as string literals.
**Validates: Requirements 1.1**

### Property 2: Configuration Loading from External Sources
*For any* system startup with valid configuration sources (environment variables or config files), all sensitive configuration must be successfully loaded from these external sources.
**Validates: Requirements 1.2**

### Property 3: Sensitive Data Redaction in Logs
*For any* log output containing sensitive data patterns (API keys, tokens, passwords), the sensitive portions must be redacted or masked before being written to the log.
**Validates: Requirements 1.3, 8.5**

### Property 4: Input Validation Completeness
*For any* API request with user-provided input, the system must validate and sanitize all input fields before processing.
**Validates: Requirements 1.4**

### Property 5: Rate Limiting Enforcement
*For any* sequence of API requests to a rate-limited endpoint exceeding the configured limit, subsequent requests must be rejected with HTTP 429 status.
**Validates: Requirements 1.5**

### Property 6: Structured Error Format
*For any* error returned by the system, the error must be a structured type containing at minimum a code, message, and the original error (if wrapping).
**Validates: Requirements 2.1**

### Property 7: Error Context Preservation
*For any* error that is wrapped at any layer of the system, the wrapping must preserve the original error in the error chain and add contextual information.
**Validates: Requirements 2.2**

### Property 8: Error Logging Context Completeness
*For any* error that is logged, the log entry must include relevant context fields such as function name, operation parameters, and error details.
**Validates: Requirements 2.3**

### Property 9: No Panic for Recoverable Errors
*For any* recoverable error condition (validation failures, not found errors, external API failures), the system must return an error value rather than panicking.
**Validates: Requirements 2.4**

### Property 10: Database Error Specificity
*For any* database operation failure, the returned error message must include the specific operation type (insert, update, delete, select) that failed.
**Validates: Requirements 2.5**

### Property 11: Connection Pool Limit Enforcement
*For any* configured database connection pool with maximum connection limit N, the number of active connections must never exceed N.
**Validates: Requirements 3.1**

### Property 12: HTTP Response Body Closure
*For any* HTTP request made by the system, the response body must be closed exactly once, regardless of whether the request succeeds or fails.
**Validates: Requirements 3.2**

### Property 13: Context Cancellation Propagation
*For any* long-running operation with a cancellable context, when the context is cancelled, the operation must terminate within a reasonable timeout period.
**Validates: Requirements 3.4**

### Property 14: Goroutine Termination Capability
*For any* goroutine created by the system, there must exist a mechanism (context cancellation, done channel, or timeout) to terminate the goroutine.
**Validates: Requirements 3.5**

### Property 15: Concurrent Access Synchronization
*For any* shared state accessed by multiple goroutines, all read and write operations must be protected by appropriate synchronization mechanisms (mutexes, channels, or atomic operations).
**Validates: Requirements 4.1**

### Property 16: Concurrent Map Access Safety
*For any* map accessed concurrently by multiple goroutines, the map must be either a sync.Map or protected by a mutex for all read and write operations.
**Validates: Requirements 4.3**

### Property 17: Deadlock-Free Lock Acquisition
*For any* code path that acquires multiple locks, the locks must always be acquired in a consistent order to prevent circular dependencies and deadlocks.
**Validates: Requirements 4.4**

### Property 18: Configuration Precedence Correctness
*For any* configuration parameter that can be specified in both a config file and an environment variable, the environment variable value must take precedence.
**Validates: Requirements 7.2**

### Property 19: Configuration Validation Completeness
*For any* invalid configuration (missing required fields, invalid values, type mismatches), the system must fail to start and return a descriptive error message indicating the specific validation failure.
**Validates: Requirements 7.3, 7.4**

### Property 20: Default Configuration Values
*For any* optional configuration parameter that is not provided, the system must apply a documented default value.
**Validates: Requirements 7.5**

### Property 21: Structured Logging Consistency
*For any* log entry produced by the system, the entry must use structured format with consistent field names (timestamp, level, message, context fields).
**Validates: Requirements 8.1, 8.2**

### Property 22: Task Correlation ID Propagation
*For any* task processed by the system, all log entries related to that task must include the same correlation ID.
**Validates: Requirements 8.4**

### Property 23: API Response Format Consistency
*For any* API endpoint response (success or error), the response must conform to the standard JSON format with success/data/error fields.
**Validates: Requirements 9.1**

### Property 24: HTTP Status Code Correctness
*For any* API error response, the HTTP status code must be appropriate for the error type (400 for validation, 404 for not found, 500 for server errors, 429 for rate limit).
**Validates: Requirements 9.2**

### Property 25: Content-Type Validation
*For any* POST request to the API, if the Content-Type header is not application/json or application/x-www-form-urlencoded, the request must be rejected with HTTP 415 status.
**Validates: Requirements 9.5**

### Property 26: SQL Injection Prevention
*For any* database query with user-provided input, the input must be passed as parameters to prepared statements, never concatenated into the SQL string.
**Validates: Requirements 10.1**

### Property 27: Transaction Atomicity
*For any* set of related database operations wrapped in a transaction, either all operations must succeed and be committed, or all operations must fail and be rolled back.
**Validates: Requirements 10.2**

### Property 28: Connection Retry with Exponential Backoff
*For any* database connection failure, the system must retry the connection with exponentially increasing delays between attempts (up to a maximum delay).
**Validates: Requirements 10.5**

### Property 29: Worker Pool Concurrency Limit
*For any* batch processing of gift codes with a configured worker pool size N, the number of concurrently executing workers must never exceed N.
**Validates: Requirements 11.1**

### Property 30: Metrics Counter Monotonicity
*For any* metrics counter (request count, error count, task count), the counter value must be monotonically increasing and never decrease during the lifetime of the process.
**Validates: Requirements 12.2, 12.3**

## Error Handling

### Error Handling Strategy

1. **错误分类**
   - Transient errors: 可重试（网络超时、临时服务不可用）
   - Permanent errors: 不可重试（验证失败、资源不存在）
   - Fatal errors: 需要系统停止（配置错误、数据库连接失败）

2. **错误传播**
   ```go
   // 在每一层添加上下文
   if err := repo.SaveUser(ctx, user); err != nil {
       return fmt.Errorf("failed to save user %s: %w", user.FID, err)
   }
   ```

3. **错误日志**
   ```go
   logger.WithFields(logrus.Fields{
       "error": err,
       "fid": fid,
       "code": code,
       "operation": "redeem_gift_code",
   }).Error("failed to redeem gift code")
   ```

4. **错误恢复**
   - HTTP handlers: 返回适当的 HTTP 状态码和错误信息
   - Background jobs: 记录错误并继续处理其他任务
   - Critical operations: 使用 panic recovery 防止进程崩溃

### Retry Strategy

```go
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay time.Duration
    Multiplier float64
}

// 指数退避重试
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error
```

## Testing Strategy

### Unit Testing

**测试范围**:
- 所有业务逻辑函数
- 错误处理路径
- 边界条件
- 配置验证逻辑

**测试工具**:
- `testing` 标准库
- `testify/assert` 用于断言
- `testify/mock` 用于 mock
- `go-sqlmock` 用于数据库测试

**示例**:
```go
func TestGiftService_RedeemGiftCode(t *testing.T) {
    // Setup
    mockRepo := new(MockRepository)
    mockCaptcha := new(MockCaptchaClient)
    service := NewGiftService(mockRepo, mockCaptcha, nil, nil)
    
    // Test cases
    tests := []struct {
        name    string
        fid     string
        code    string
        setup   func()
        wantErr bool
    }{
        {
            name: "successful redemption",
            fid:  "123",
            code: "ABC123",
            setup: func() {
                mockRepo.On("IsGiftCodeReceived", mock.Anything, "123", "ABC123").Return(false, nil)
                mockRepo.On("SaveGiftCode", mock.Anything, "123", "ABC123").Return(nil)
            },
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            _, err := service.RedeemGiftCode(context.Background(), tt.fid, tt.code)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Property-Based Testing

**测试工具**: `gopter` 或 `rapid`

**测试配置**: 每个属性测试至少运行 100 次迭代

**测试标签格式**: `Feature: code-quality-improvements, Property {number}: {property_text}`

**示例**:
```go
// Feature: code-quality-improvements, Property 2: Error Context Preservation
func TestProperty_ErrorContextPreservation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("wrapping errors preserves original error", prop.ForAll(
        func(originalMsg string) bool {
            originalErr := errors.New(originalMsg)
            wrappedErr := fmt.Errorf("context: %w", originalErr)
            return errors.Is(wrappedErr, originalErr)
        },
        gen.AnyString(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Integration Testing

**测试范围**:
- HTTP API 端点
- 数据库操作
- 外部服务集成（使用 mock 服务器）

**测试环境**:
- 使用 in-memory SQLite 数据库
- 使用 httptest 创建测试服务器
- 使用 testcontainers 进行更真实的集成测试（可选）

### Test Coverage Goals

- 核心业务逻辑: 80%+
- 数据访问层: 70%+
- HTTP handlers: 60%+
- 整体代码覆盖率: 60%+

### Continuous Testing

- 在 CI/CD 管道中运行所有测试
- 使用 race detector: `go test -race ./...`
- 生成覆盖率报告: `go test -coverprofile=coverage.out ./...`
- 使用 golangci-lint 进行静态分析

## Implementation Notes

### Phase 1: Foundation (Week 1)
1. 创建 config 包和配置管理
2. 实现错误处理包
3. 重构 HTTP 客户端工厂
4. 添加基础日志中间件

### Phase 2: Storage Layer (Week 2)
1. 定义 Repository 接口
2. 重构 SQLite 实现
3. 添加事务支持
4. 实现数据库迁移

### Phase 3: Business Logic (Week 3)
1. 创建 GiftService
2. 重构 CaptchaPool
3. 改进任务调度器
4. 添加并发控制

### Phase 4: API Layer (Week 4)
1. 标准化 API 响应
2. 添加中间件栈
3. 实现请求验证
4. 添加限流

### Phase 5: Observability (Week 5)
1. 实现健康检查
2. 添加 Prometheus 指标
3. 改进结构化日志
4. 添加分布式追踪（可选）

### Phase 6: Testing & Documentation (Week 6)
1. 编写单元测试
2. 编写属性测试
3. 编写集成测试
4. 更新文档

### Migration Strategy

- 采用 Strangler Fig 模式逐步替换旧代码
- 保持向后兼容性
- 使用 feature flags 控制新功能启用
- 每个阶段都要确保系统可运行

### Performance Considerations

1. **数据库连接池**: 根据负载调整连接数
2. **HTTP 连接复用**: 使用长连接减少握手开销
3. **并发控制**: 使用 worker pool 限制并发数
4. **缓存策略**: 缓存用户信息减少数据库查询
5. **批量操作**: 批量处理礼品码减少网络往返

### Security Considerations

1. **输入验证**: 验证所有用户输入
2. **SQL 注入防护**: 使用参数化查询
3. **敏感信息保护**: 不在日志中输出敏感信息
4. **限流**: 防止 API 滥用
5. **HTTPS**: 生产环境使用 HTTPS
6. **依赖扫描**: 定期扫描依赖漏洞
