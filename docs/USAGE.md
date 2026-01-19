# 使用指南

## 目录

- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [管理后台](#管理后台)
- [API 接口](#api-接口)
- [任务系统](#任务系统)
- [OCR 服务](#ocr-服务)
- [通知服务](#通知服务)
- [故障排查](#故障排查)

## 快速开始

### 环境要求

- Go 1.20+
- SQLite3

### 安装运行

```bash
# 克隆项目
git clone <repository-url>
cd cdk-get

# 运行服务
go run cmd/server/main.go
```

服务默认在 `http://localhost:10999` 启动。

### 目录结构

```
cdk-get/
├── cmd/                    # 命令行入口
│   ├── server/            # 主服务
│   ├── verify/            # 验证工具
│   └── zqwn/              # 其他工具
├── internal/              # 内部包
│   ├── api/               # HTTP API 处理
│   ├── auth/              # 认证服务
│   ├── captcha/           # OCR 验证码识别
│   ├── config/            # 配置管理
│   ├── errors/            # 错误定义
│   ├── giftcode/          # 兑换码客户端
│   ├── httpclient/        # HTTP 客户端
│   ├── job/               # 任务调度
│   ├── logging/           # 日志
│   ├── notification/      # 通知服务
│   ├── service/           # 业务逻辑
│   ├── storage/           # 数据存储
│   ├── svc/               # 服务上下文
│   └── utls/              # 工具函数
├── etc/                   # 配置文件
├── docs/                  # 文档
└── static/                # 静态资源
```

## 配置说明

### 配置文件

主配置文件位于 `etc/config.yaml`：

```yaml
# 服务配置
server:
  host: "0.0.0.0"
  port: 10999

# 管理员配置
admin:
  username: "admin"
  password_hash: "$2a$10$..."
  token_secret: "your-secret-key"
  token_duration: 24h

# 验证码配置
captcha:
  providers:
    - type: "ali"
      access_key: ""
      secret_key: ""
    - type: "tencent"
      access_key: ""
      secret_key: ""
    - type: "google"
      credentials_json: ""

# 通知配置
notification:
  enabled: false
  wxpusher:
    app_token: ""
    uids: []
```

### 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `ADMIN_USERNAME` | 管理员用户名 | - |
| `ADMIN_PASSWORD_HASH` | bcrypt 密码哈希 | - |
| `ADMIN_TOKEN_SECRET` | JWT 密钥 | - |
| `ADMIN_TOKEN_DURATION` | Token 有效期 | 24h |
| `ACCESS_KEY` | 阿里云 AccessKey | - |
| `ACCESS_SECRET` | 阿里云 SecretKey | - |
| `GOOGLE_CREDENTIALS_JSON` | Google 凭证 JSON | - |
| `SERVER_PORT` | 服务端口 | 10999 |

### 生成密码哈希

```bash
# 方法1: 使用 Go
go run -e 'package main; import ("fmt"; "golang.org/x/crypto/bcrypt"); func main() { h, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost); fmt.Println(string(h)) }'

# 方法2: 使用 htpasswd
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

## 管理后台

访问 `http://localhost:10999/admin/` 进入管理后台。

### 功能特性

- **用户管理**: 添加、查看系统用户
- **任务监控**: 实时查看任务执行状态，支持自动刷新
- **兑换记录**: 查看用户兑换历史
- **通知历史**: 查看系统通知发送记录

### 登录配置

首次使用需要配置管理员账号，详见 [ADMIN_SETUP.md](ADMIN_SETUP.md)。

## API 接口

### 认证接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/admin/login` | 管理员登录 |

**登录请求**:
```json
{
  "username": "admin",
  "password": "your-password"
}
```

**响应**:
```json
{
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-01-20T10:00:00Z"
  }
}
```

### 用户接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/admin/users` | 获取用户列表 | 是 |
| POST | `/api/admin/users` | 添加用户 | 是 |
| GET | `/api/admin/users/:fid/codes` | 获取用户兑换记录 | 是 |

### 任务接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/admin/tasks` | 获取待处理任务 | 是 |
| POST | `/api/admin/tasks` | 添加兑换码任务 | 是 |
| GET | `/api/admin/tasks/completed` | 获取已完成任务 | 是 |
| DELETE | `/api/admin/tasks/:code` | 删除任务 | 是 |

### 通知接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/admin/notifications` | 获取通知历史 | 是 |

### 认证方式

所有需要认证的接口需要在 Header 中携带 Token：

```
Authorization: Bearer <your-jwt-token>
```

## 任务系统

### 工作流程

1. 通过管理后台或 API 添加兑换码任务
2. 调度器定期检查待处理任务
3. GetCodeJob 执行兑换操作
4. 记录兑换结果，更新任务状态
5. 兑换成功后发送通知

### 任务状态

- **待处理**: 任务刚创建，等待执行
- **处理中**: 正在执行兑换
- **已完成**: 兑换成功或确认 CDK 不存在
- **失败**: 兑换失败，达到最大重试次数

### 重试机制

- 默认重试间隔: 2秒
- 任务执行周期: 30秒
- 失败时会记录错误信息和重试次数

## OCR 服务

系统支持多个 OCR 提供商进行验证码识别。

### 支持的提供商

| 提供商 | 类型 | 说明 |
|--------|------|------|
| 阿里云 | ali | 适合国内服务 |
| 腾讯云 | tencent | 适合国内服务 |
| Google | google | 适合海外服务 |

### 配置示例

```yaml
captcha:
  providers:
    - type: "ali"
      access_key: "your-access-key"
      secret_key: "your-secret-key"
    - type: "google"
      credentials_json: '{"type":"service_account",...}'
```

### 负载均衡

系统会在多个 OCR 提供商之间自动进行负载均衡，提高可用性。

## 通知服务

### 支持的通知渠道

| 渠道 | 配置项 | 说明 |
|------|--------|------|
| 微信推送 | wxpusher | 通过 wxpusher 发送微信通知 |

### 配置示例

```yaml
notification:
  enabled: true
  wxpusher:
    app_token: "your-app-token"
    uids:
      - "user-id-1"
      - "user-id-2"
```

### 通知触发

- 兑换码兑换成功时自动发送通知

## 故障排查

### 无法启动

```
Error: bind: address already in use
```

解决方法：修改配置文件中的端口号，或停止占用端口的进程。

### 无法登录管理后台

- 检查用户名和密码是否正确
- 确认密码哈希格式正确（bcrypt, cost=10）
- 查看服务器日志排查问题

### OCR 识别失败

- 检查 OCR 服务配置是否正确
- 确认 API Key/ Secret 有效
- 查看日志中的具体错误信息

### 任务执行失败

- 检查网络连接
- 确认用户 FID 有效
- 查看任务错误信息
- 检查 OCR 服务是否正常

### 数据库问题

```
Error: database is locked
```

解决方法：
- 确保没有其他进程访问数据库
- 检查数据库文件权限

## Docker 部署

```dockerfile
# 构建镜像
docker build -t cdk-get .

# 运行容器
docker run -d \
  -p 10999:10999 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/etc/config.yaml:/app/etc/config.yaml \
  cdk-get
```

## 监控与日志

### 日志位置

- 默认输出到标准输出
- 可配置日志文件路径

### 日志级别

- `debug`: 详细调试信息
- `info`: 一般信息
- `warn`: 警告
- `error`: 错误

### 健康检查

```
GET /health
```

返回服务健康状态。

## 安全建议

1. **强密码**: 使用复杂的管理员密码
2. **密钥管理**: 使用随机生成的 JWT 密钥
3. **HTTPS**: 生产环境必须启用 HTTPS
4. **访问控制**: 使用防火墙限制管理后台访问
5. **定期更新**: 定期更换密钥和密码
