## 无尽冬日兑换码兑换

### 管理后台

系统提供了一个基于Web的管理后台，用于管理用户、查看兑换记录、监控任务和添加新的兑换码。

#### 功能特性

- **用户管理**: 查看所有系统用户，添加新用户
- **兑换记录查询**: 查看特定用户的激活码兑换历史
- **任务监控**: 实时监控后台兑换任务的执行状态
- **兑换码管理**: 添加新的兑换码到系统中

#### 快速开始

> **详细设置指南**: 请参阅 [管理后台初始设置指南](docs/ADMIN_SETUP.md) 获取完整的配置说明和安全最佳实践。

1. **配置管理员账号**

   编辑 `etc/config.yaml` 或使用环境变量配置管理员账号：

   ```yaml
   admin:
     username: "admin"
     password_hash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
     token_secret: "your-secret-key-change-in-production-min-32-chars"
     token_duration: 24h
   ```

2. **生成密码哈希**

   使用以下方法之一生成bcrypt密码哈希：

   **方法 1: 使用 Go 代码**
   ```go
   package main
   
   import (
       "fmt"
       "golang.org/x/crypto/bcrypt"
   )
   
   func main() {
       password := "your-password"
       hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
       fmt.Println(string(hash))
   }
   ```

   **方法 2: 使用在线工具**
   - 访问 https://bcrypt-generator.com/
   - 输入密码并选择 cost 为 10
   - 复制生成的哈希值

   **方法 3: 使用 htpasswd 命令**
   ```bash
   htpasswd -bnBC 10 "" your-password | tr -d ':\n'
   ```

3. **生成 JWT 密钥**

   使用 openssl 生成强随机密钥：

   ```bash
   openssl rand -base64 32
   ```

   将生成的密钥设置到配置文件的 `admin.token_secret` 字段。

   **重要**: 生产环境必须使用强随机密钥，不要使用示例中的默认值！

4. **使用环境变量（推荐用于生产环境）**

   为了安全起见，建议在生产环境中使用环境变量而不是配置文件：

   ```bash
   export ADMIN_USERNAME="admin"
   export ADMIN_PASSWORD_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
   export ADMIN_TOKEN_SECRET="your-generated-secret-key"
   export ADMIN_TOKEN_DURATION="24h"
   ```

   环境变量会覆盖配置文件中的设置。

5. **启动服务器**

   ```bash
   go run cmd/server/main.go
   ```

6. **访问管理后台**

   打开浏览器访问: `http://localhost:10999/admin/`

   使用配置的用户名和密码登录。

#### API 端点

管理后台提供以下 RESTful API 端点：

| 端点 | 方法 | 认证 | 描述 |
|------|------|------|------|
| `/api/admin/login` | POST | 否 | 管理员登录，获取JWT令牌 |
| `/api/admin/users` | GET | 是 | 获取所有用户列表 |
| `/api/admin/users` | POST | 是 | 添加新用户 |
| `/api/admin/users/:fid/codes` | GET | 是 | 获取指定用户的兑换记录 |
| `/api/admin/tasks` | GET | 是 | 获取任务列表 |
| `/api/admin/tasks` | POST | 是 | 添加新的兑换码任务 |

#### 认证机制

管理后台使用 JWT (JSON Web Token) 进行身份认证：

1. 通过 `/api/admin/login` 端点提交用户名和密码
2. 成功后返回 JWT 令牌和过期时间
3. 在后续请求中，将令牌添加到 `Authorization` 头：
   ```
   Authorization: Bearer <your-jwt-token>
   ```
4. 令牌过期后需要重新登录

#### 安全建议

- **强密码**: 使用至少 12 个字符的强密码，包含大小写字母、数字和特殊字符
- **强密钥**: JWT 密钥至少 32 个字符，使用随机生成的字符串
- **HTTPS**: 生产环境必须使用 HTTPS 保护传输安全
- **定期更换**: 定期更换管理员密码和 JWT 密钥
- **访问控制**: 使用防火墙限制管理后台的访问来源
- **日志监控**: 定期检查认证日志，发现异常登录尝试

#### 故障排查

**问题: 无法登录**
- 检查用户名和密码是否正确
- 确认密码哈希是否正确生成（bcrypt cost 为 10）
- 查看服务器日志中的错误信息

**问题: 令牌过期**
- 令牌默认有效期为 24 小时
- 可以通过 `admin.token_duration` 配置调整有效期
- 过期后需要重新登录获取新令牌

**问题: 401 未授权错误**
- 检查 Authorization 头格式是否正确（`Bearer <token>`）
- 确认令牌未过期
- 验证 JWT 密钥配置是否正确

### OCR 服务配置

本系统支持多个 OCR 服务提供商用于验证码识别：

#### 支持的提供商

- **阿里云 OCR** (ali)
- **腾讯云 OCR** (tencent)
- **Google Cloud Vision API** (google)

#### Google Cloud Vision API 设置

1. **创建 Google Cloud 项目**
   - 访问 [Google Cloud Console](https://console.cloud.google.com/)
   - 创建新项目或选择现有项目

2. **启用 Cloud Vision API**
   - 在项目中导航到 "APIs & Services" > "Library"
   - 搜索 "Cloud Vision API"
   - 点击 "Enable"

3. **创建服务账号**
   - 导航到 "IAM & Admin" > "Service Accounts"
   - 点击 "Create Service Account"
   - 输入服务账号名称和描述
   - 授予 "Cloud Vision API User" 角色
   - 点击 "Done"

4. **下载凭证 JSON**
   - 在服务账号列表中，点击刚创建的服务账号
   - 转到 "Keys" 标签
   - 点击 "Add Key" > "Create new key"
   - 选择 "JSON" 格式
   - 下载 JSON 密钥文件

#### 配置方式

**方式 1: YAML 配置文件**

编辑 `etc/config.yaml`:

```yaml
captcha:
  providers:
    - type: "google"
      credentials_json: '{"type":"service_account","project_id":"your-project",...}'
    - type: "ali"
      access_key: "your-ali-access-key"
      secret_key: "your-ali-secret-key"
    - type: "tencent"
      access_key: "your-tencent-access-key"
      secret_key: "your-tencent-secret-key"
```

**方式 2: 环境变量**

```bash
# Google Cloud Vision
export GOOGLE_CREDENTIALS_JSON='{"type":"service_account","project_id":"your-project",...}'

# 或使用默认凭证
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# 阿里云 (向后兼容)
export ACCESS_KEY="your-ali-access-key"
export ACCESS_SECRET="your-ali-secret-key"
```

**方式 3: 混合配置**

环境变量会覆盖 YAML 配置，可以在 YAML 中配置基础设置，用环境变量覆盖敏感信息。

#### 负载均衡

系统会自动在所有配置的 OCR 提供商之间进行轮询负载均衡，提高可用性和吞吐量。

#### 费用说明

Google Cloud Vision API 定价（截至 2025 年）：
- 前 1,000 次请求/月：免费
- 1,001 - 5,000,000 次：$1.50 / 1,000 次
- 文本检测计为 1 个单位/图片

建议设置计费警报以监控使用情况。

