# 架构文档

## 目录

- [系统概览](#系统概览)
- [技术栈](#技术栈)
- [系统架构](#系统架构)
- [核心模块](#核心模块)
- [数据模型](#数据模型)
- [API 设计](#api-设计)
- [任务调度](#任务调度)
- [安全机制](#安全机制)
- [部署架构](#部署架构)

## 系统概览

无尽冬日兑换码兑换系统是一个用于自动化兑换游戏礼包码的后台管理系统。主要功能包括：

- 用户管理
- 兑换码任务执行与监控
- OCR 验证码识别
- 多渠道通知推送

## 技术栈

### 后端

| 技术 | 用途 | 版本要求 |
|------|------|----------|
| Go | 核心开发语言 | 1.20+ |
| Gin | HTTP 框架 | - |
| SQLite | 嵌入式数据库 | - |
| JWT | 身份认证 | - |
| bcrypt | 密码加密 | - |

### 前端

| 技术 | 用途 |
|------|------|
| HTML5 | 页面结构 |
| CSS3 | 样式 |
| Vanilla JavaScript | 交互逻辑 |

### 外部服务

| 服务 | 用途 |
|------|------|
| 阿里云 OCR | 验证码识别 |
| 腾讯云 OCR | 验证码识别 |
| Google Cloud Vision | 验证码识别 |
| WxPusher | 微信通知推送 |

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端层                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  浏览器      │  │  API 调用   │  │      其他客户端          │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP 服务层                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Gin HTTP Server                      │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────────────┐  │    │
│  │  │ 静态资源    │ │ REST API   │ │  中间件栈          │  │    │
│  │  │ (HTML/JS)  │ │ (JSON)     │ │  - Auth            │  │    │
│  │  │            │ │            │ │  - Logger          │  │    │
│  │  │            │ │            │ │  - Recovery        │  │    │
│  │  └────────────┘ └────────────┘ └────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       业务逻辑层                                 │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │
│  │ 认证服务   │ │ API 处理器 │ │ 任务调度器 │ │ 通知服务   │   │
│  │ (Auth)     │ │ (Handlers) │ │ (Scheduler)│ │ (Notify)   │   │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        数据访问层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Repository 接口 (存储抽象)                  │    │
│  │  ┌───────────────────────────────────────────────────┐  │    │
│  │  │           SqliteRepository 实现                    │  │    │
│  │  └───────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        数据存储层                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    SQLite 数据库                         │    │
│  │  - fid_list (用户)                                      │    │
│  │  - gift_codes (兑换记录)                                │    │
│  │  - gift_code_task (任务)                                │    │
│  │  - notifications (通知记录)                             │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## 核心模块

### 1. API 服务 (cmd/server/main.go)

系统入口，负责初始化所有组件并启动 HTTP 服务器。

**职责**:
- 配置加载
- 日志初始化
- 数据库连接
- 认证服务初始化
- 通知服务初始化
- HTTP 服务器启动
- 优雅关闭处理

### 2. 配置管理 (internal/config)

```go
type Config struct {
    Server      ServerConfig
    Database    DatabaseConfig
    Admin       AdminConfig
    Captcha     CaptchaConfig
    Notification NotificationConfig
    Logging     LoggingConfig
    Security    SecurityConfig
}
```

### 3. 认证服务 (internal/auth)

```go
type AuthService struct {
    username      string
    passwordHash  string
    tokenSecret   string
    tokenDuration time.Duration
}
```

**功能**:
- 密码验证 (bcrypt)
- JWT Token 生成
- Token 验证

### 4. 存储层 (internal/storage)

#### Repository 接口设计

```go
type Repository interface {
    // Gift Code operations
    SaveGiftCode(ctx context.Context, fid, code string) error
    IsGiftCodeReceived(ctx context.Context, fid, code string) (bool, error)
    ListGiftCodesByFID(ctx context.Context, fid string) ([]*GiftCodeRecord, error)

    // User operations
    SaveUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, fid string) (*User, error)
    ListUsers(ctx context.Context) ([]*User, error)

    // Task operations
    CreateTask(ctx context.Context, code string) error
    ListPendingTasks(ctx context.Context) ([]*Task, error)
    MarkTaskComplete(ctx context.Context, code string) error
    GetTaskByCode(ctx context.Context, code string) (*Task, error)
    UpdateTaskRetry(ctx context.Context, code string, retryCount int, lastError string) error
    UpdateTaskComplete(ctx context.Context, code string, completedAt time.Time) error
    ListCompletedTasks(ctx context.Context, limit int) ([]*Task, error)
    DeleteTask(ctx context.Context, code string) error

    // Notification operations
    SaveNotification(ctx context.Context, notification *Notification) error
    ListNotifications(ctx context.Context, limit int) ([]*Notification, error)

    // Transaction support
    WithTransaction(ctx context.Context, fn func(Repository) error) error

    // Health check
    Ping(ctx context.Context) error
    Close() error
}
```

### 5. 任务系统 (internal/job)

#### GetCodeJob 结构

```go
type GetCodeJob struct {
    svcCtx    *svc.ServiceContext
    cliKeep   map[string]*giftcode.PlayerGiftCode
    clients   []captcha.RemoteClient
    shardLock sync.Mutex
    idx       int
}
```

**工作流程**:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ 获取待处理任务 │ ──▶ │ 遍历处理人   │ ──▶ │ 初始化玩家   │
│              │     │              │     │ 客户端       │
└──────────────┘     └──────────────┘     └──────────────┘
                                                │
                                                ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ 发送通知      │ ◀── │ 记录结果     │ ◀── │ 执行兑换     │
│ (成功时)      │     │ 更新状态     │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
```

### 6. OCR 服务 (internal/captcha)

#### 接口定义

```go
type RemoteClient interface {
    Recognize(image []byte) (string, error)
}
```

#### 实现类

| 实现 | 配置项 | 说明 |
|------|--------|------|
| AliCaptchaClient | access_key, secret_key | 阿里云 OCR |
| TcCaptchaClient | access_key, secret_key | 腾讯云 OCR |
| GoogleCaptchaClient | credentials_json | Google Vision |

#### 负载均衡

系统自动在多个 OCR 提供商之间轮询，提高可用性。

### 7. 兑换码客户端 (internal/giftcode)

```go
type PlayerGiftCode struct {
    Fid     string
    getCaptcha captcha.RemoteClient
    repository storage.Repository
    Player   *PlayerInfo
}
```

### 8. 通知服务 (internal/notification)

#### WxPusher 集成

```go
type WxPusherNotifier struct {
    appToken string
    uids     []string
    logger   *logrus.Logger
}
```

## 数据模型

### User (用户)

```go
type User struct {
    FID         string    `json:"fid"`
    Nickname    string    `json:"nickname"`
    KID         int       `json:"kid"`
    AvatarImage string    `json:"avatar_image"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Task (任务)

```go
type Task struct {
    Code        string     `json:"code"`
    AllDone     bool       `json:"all_done"`
    RetryCount  int        `json:"retry_count"`
    LastError   string     `json:"last_error,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}
```

### GiftCodeRecord (兑换记录)

```go
type GiftCodeRecord struct {
    ID        int64     `json:"id"`
    FID       string    `json:"fid"`
    Code      string    `json:"code"`
    Status    string    `json:"status"` // success, failed, duplicate
    Message   string    `json:"message,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Notification (通知记录)

```go
type Notification struct {
    ID        int64     `json:"id"`
    Channel   string    `json:"channel"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Result    string    `json:"result"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 数据库表结构

```sql
-- 用户表
CREATE TABLE fid_list (
    fid TEXT PRIMARY KEY,
    nickname TEXT NOT NULL,
    kid INTEGER NOT NULL,
    avatar_image TEXT NOT NULL
);

-- 兑换记录表
CREATE TABLE gift_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fid TEXT NOT NULL,
    code TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_fid_code ON gift_codes (fid, code);

-- 任务表
CREATE TABLE gift_code_task (
    code TEXT PRIMARY KEY,
    all_done INTEGER NOT NULL,
    created_at TIMESTAMP,
    completed_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT DEFAULT ''
);

-- 通知表
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    result TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP
);
```

## API 设计

### RESTful 风格

```
GET    /api/admin/resources        # 列表
POST   /api/admin/resources        # 创建
GET    /api/admin/resources/:id    # 详情
PUT    /api/admin/resources/:id    # 更新
DELETE /api/admin/resources/:id    # 删除
```

### 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {
    // 业务数据
  }
}
```

### 错误响应

```json
{
  "code": 1001,
  "message": "Invalid parameter",
  "error": {
    "details": "..."
  }
}
```

## 任务调度

### 调度器结构

```go
type Scheduler struct {
    jobs     []Job
    interval time.Duration
    stopCh   chan struct{}
}
```

### 任务配置

| 任务 | 执行间隔 | 周期 | 说明 |
|------|----------|------|------|
| GetCodeJob | 2s | 30s | 兑换码处理任务 |

### 重试机制

- 失败时增加重试计数
- 记录最后错误信息
- 支持panic恢复

## 安全机制

### 认证

- JWT Token 认证
- Token 有效期可配置
- 支持 Token 刷新

### 授权

- 基于角色的访问控制
- 管理后台独立认证

### 数据安全

- 密码 bcrypt 加密存储
- 日志敏感信息脱敏
- SQL 注入防护

### 传输安全

- HTTPS 支持 (生产环境必需)
- CORS 配置

## 部署架构

### 容器化部署

```dockerfile
FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server cmd/server/main.go

EXPOSE 10999

CMD ["./server"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  cdk-get:
    build: .
    ports:
      - "10999:10999"
    volumes:
      - ./data:/app/data
      - ./etc/config.yaml:/app/etc/config.yaml
    restart: unless-stopped
```

### 生产环境建议

1. **反向代理**: 使用 Nginx/Traefik
2. **HTTPS**: 启用 SSL/TLS
3. **监控**: 集成 Prometheus/Grafana
4. **日志**: 集中化日志管理
5. **备份**: 定期备份数据库
6. **高可用**: 多实例部署

## 目录结构

```
cdk-get/
├── cmd/
│   ├── server/               # 主服务入口
│   │   ├── main.go
│   │   └── static/           # 嵌入式静态资源
│   ├── verify/               # 验证工具
│   └── zqwn/                 # 其他工具
├── internal/
│   ├── api/                  # HTTP API
│   │   ├── admin_handlers.go
│   │   ├── handlers.go
│   │   ├── middleware.go
│   │   └── response.go
│   ├── auth/                 # 认证
│   ├── captcha/              # OCR 识别
│   ├── config/               # 配置
│   ├── errors/               # 错误定义
│   ├── giftcode/             # 兑换码客户端
│   ├── httpclient/           # HTTP 客户端
│   ├── job/                  # 任务调度
│   ├── logging/              # 日志
│   ├── notification/         # 通知
│   ├── service/              # 业务逻辑
│   ├── storage/              # 数据存储
│   │   ├── migrations/       # 数据库迁移
│   │   ├── repository.go
│   │   └── sqlite_repository.go
│   ├── svc/                  # 服务上下文
│   └── utls/                 # 工具函数
├── etc/
│   └── config.example.yaml
├── docs/
│   ├── USAGE.md
│   ├── ARCHITECTURE.md
│   └── ADMIN_SETUP.md
├── static/                   # 静态资源
├── Dockerfile
├── go.mod
└── README.md
```

## 扩展指南

### 添加新的 OCR 提供商

1. 实现 `captcha.RemoteClient` 接口
2. 在 `initClients()` 中注册
3. 添加配置项

### 添加新的通知渠道

1. 实现 `notification.Notifier` 接口
2. 在 `main.go` 中初始化
3. 添加配置项

### 添加新的存储后端

1. 实现 `storage.Repository` 接口
2. 在 `main.go` 中替换 `SqliteRepository`
