# 管理后台初始设置指南

本指南将帮助您完成管理后台的初始配置，包括设置管理员密码和生成JWT密钥。

## 目录

1. [生成管理员密码哈希](#生成管理员密码哈希)
2. [生成JWT密钥](#生成jwt密钥)
3. [配置管理员账号](#配置管理员账号)
4. [验证配置](#验证配置)
5. [安全最佳实践](#安全最佳实践)

## 生成管理员密码哈希

管理后台使用bcrypt算法对密码进行哈希处理。您需要将密码转换为bcrypt哈希值后再配置到系统中。

### 方法 1: 使用 Go 代码

创建一个临时的Go程序来生成密码哈希：

```go
package main

import (
    "fmt"
    "os"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run generate_hash.go <password>")
        os.Exit(1)
    }
    
    password := os.Args[1]
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        fmt.Printf("Error generating hash: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Password: %s\n", password)
    fmt.Printf("Hash: %s\n", string(hash))
}
```

保存为 `generate_hash.go` 并运行：

```bash
go run generate_hash.go "your-secure-password"
```

### 方法 2: 使用在线工具

1. 访问 https://bcrypt-generator.com/
2. 在输入框中输入您的密码
3. 选择 **Rounds** 为 `10`（这是bcrypt的默认cost）
4. 点击生成按钮
5. 复制生成的哈希值

**注意**: 使用在线工具时，请确保使用HTTPS连接，并且不要在生产环境中使用此方法生成真实密码。

### 方法 3: 使用 htpasswd 命令

如果您的系统安装了Apache工具包，可以使用htpasswd命令：

```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

这将输出bcrypt哈希值。

### 方法 4: 使用 Python

如果您安装了Python和bcrypt库：

```bash
pip install bcrypt
```

然后运行：

```python
import bcrypt
import sys

if len(sys.argv) < 2:
    print("Usage: python generate_hash.py <password>")
    sys.exit(1)

password = sys.argv[1].encode('utf-8')
hash = bcrypt.hashpw(password, bcrypt.gensalt(rounds=10))
print(f"Password: {sys.argv[1]}")
print(f"Hash: {hash.decode('utf-8')}")
```

保存为 `generate_hash.py` 并运行：

```bash
python generate_hash.py "your-secure-password"
```

## 生成JWT密钥

JWT密钥用于签名和验证认证令牌。必须使用强随机密钥。

### 方法 1: 使用 OpenSSL（推荐）

```bash
openssl rand -base64 32
```

这将生成一个32字节（256位）的随机密钥，以base64编码输出。

示例输出：
```
Kx7vJ9mP2nQ8wR5tY6uI3oL1aS4dF7gH9jK0lZ2xC3v=
```

### 方法 2: 使用 /dev/urandom（Linux/macOS）

```bash
head -c 32 /dev/urandom | base64
```

### 方法 3: 使用 Python

```python
import secrets
import base64

key = secrets.token_bytes(32)
print(base64.b64encode(key).decode('utf-8'))
```

### 方法 4: 使用 Node.js

```javascript
const crypto = require('crypto');
const key = crypto.randomBytes(32).toString('base64');
console.log(key);
```

## 配置管理员账号

有两种方式配置管理员账号：配置文件或环境变量。

### 方式 1: 使用配置文件

编辑 `etc/config.yaml`：

```yaml
admin:
  username: "admin"
  password_hash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
  token_secret: "Kx7vJ9mP2nQ8wR5tY6uI3oL1aS4dF7gH9jK0lZ2xC3v="
  token_duration: 24h
```

替换：
- `username`: 您的管理员用户名
- `password_hash`: 使用上面方法生成的bcrypt哈希值
- `token_secret`: 使用上面方法生成的JWT密钥
- `token_duration`: JWT令牌有效期（如 "24h", "12h", "1h30m"）

### 方式 2: 使用环境变量（推荐用于生产环境）

设置以下环境变量：

```bash
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
export ADMIN_TOKEN_SECRET="Kx7vJ9mP2nQ8wR5tY6uI3oL1aS4dF7gH9jK0lZ2xC3v="
export ADMIN_TOKEN_DURATION="24h"
```

**注意**: 密码哈希包含 `$` 符号，在bash中需要使用单引号包裹。

### 方式 3: 混合配置

您可以在配置文件中设置基础配置，然后使用环境变量覆盖敏感信息：

`etc/config.yaml`:
```yaml
admin:
  username: "admin"
  token_duration: 24h
  # 密码和密钥通过环境变量提供
```

环境变量：
```bash
export ADMIN_PASSWORD_HASH='$2a$10$...'
export ADMIN_TOKEN_SECRET="..."
```

## 验证配置

### 1. 启动服务器

```bash
go run cmd/server/main.go
```

检查日志输出，确保没有配置错误。

### 2. 测试登录

使用curl测试登录API：

```bash
curl -X POST http://localhost:10999/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'
```

成功响应示例：
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-01-17T10:30:00Z"
  }
}
```

### 3. 测试受保护的API

使用获取的令牌访问受保护的API：

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET http://localhost:10999/api/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

### 4. 访问Web界面

打开浏览器访问：`http://localhost:10999/admin/`

使用配置的用户名和密码登录。

## 安全最佳实践

### 密码安全

1. **使用强密码**
   - 至少12个字符
   - 包含大写字母、小写字母、数字和特殊字符
   - 不要使用常见单词或个人信息
   - 不要重复使用其他系统的密码

2. **定期更换密码**
   - 建议每3-6个月更换一次管理员密码
   - 更换后重新生成bcrypt哈希并更新配置

3. **安全存储**
   - 不要将密码明文写入文件或代码
   - 使用密码管理器存储密码
   - 配置文件应该设置适当的文件权限（如 `chmod 600`）

### JWT密钥安全

1. **使用强随机密钥**
   - 至少32字节（256位）
   - 使用加密安全的随机数生成器
   - 不要使用可预测的字符串

2. **密钥轮换**
   - 定期更换JWT密钥（建议每6-12个月）
   - 更换密钥后，所有现有令牌将失效，用户需要重新登录

3. **密钥保护**
   - 不要将密钥提交到版本控制系统
   - 使用环境变量或密钥管理服务
   - 限制密钥的访问权限

### 部署安全

1. **使用HTTPS**
   - 生产环境必须使用HTTPS
   - 使用有效的SSL/TLS证书
   - 禁用不安全的协议版本

2. **访问控制**
   - 使用防火墙限制管理后台的访问来源
   - 考虑使用VPN或IP白名单
   - 启用速率限制防止暴力破解

3. **日志监控**
   - 启用认证日志记录
   - 监控失败的登录尝试
   - 设置异常登录告警

4. **令牌管理**
   - 设置合理的令牌有效期（不要太长）
   - 考虑实现令牌刷新机制
   - 提供令牌撤销功能（如果需要）

### 配置文件安全

1. **文件权限**
   ```bash
   chmod 600 etc/config.yaml
   chown app-user:app-group etc/config.yaml
   ```

2. **环境隔离**
   - 开发、测试、生产环境使用不同的密钥
   - 不要在开发环境使用生产密钥

3. **版本控制**
   - 将 `etc/config.yaml` 添加到 `.gitignore`
   - 只提交 `etc/config.example.yaml` 作为模板
   - 使用配置管理工具管理生产配置

## 故障排查

### 问题: 登录失败，提示"Invalid username or password"

**可能原因**:
1. 密码哈希生成不正确
2. bcrypt cost不是10
3. 配置文件中的哈希值被截断或包含额外字符

**解决方法**:
1. 重新生成密码哈希，确保使用cost=10
2. 检查配置文件中的哈希值是否完整
3. 确认密码输入正确（注意大小写）

### 问题: 令牌验证失败

**可能原因**:
1. JWT密钥配置错误
2. 令牌已过期
3. 令牌格式不正确

**解决方法**:
1. 检查 `token_secret` 配置是否正确
2. 检查令牌是否在有效期内
3. 确认Authorization头格式为 `Bearer <token>`

### 问题: 环境变量不生效

**可能原因**:
1. 环境变量名称错误
2. 环境变量未正确导出
3. 应用启动前未设置环境变量

**解决方法**:
1. 检查环境变量名称拼写
2. 使用 `export` 命令导出变量
3. 在启动脚本中设置环境变量

### 问题: 配置文件解析错误

**可能原因**:
1. YAML格式错误
2. 缩进不正确
3. 特殊字符未正确转义

**解决方法**:
1. 使用YAML验证工具检查格式
2. 确保使用空格而不是制表符
3. 对包含特殊字符的值使用引号

## 快速参考

### 生成密码哈希（一行命令）

```bash
# 使用Go
echo 'package main; import ("fmt"; "os"; "golang.org/x/crypto/bcrypt"); func main() { h, _ := bcrypt.GenerateFromPassword([]byte(os.Args[1]), 10); fmt.Println(string(h)) }' > /tmp/hash.go && go run /tmp/hash.go "your-password"

# 使用Python
python3 -c "import bcrypt, sys; print(bcrypt.hashpw(sys.argv[1].encode(), bcrypt.gensalt(10)).decode())" "your-password"

# 使用htpasswd
htpasswd -bnBC 10 "" "your-password" | tr -d ':\n'
```

### 生成JWT密钥（一行命令）

```bash
# 使用OpenSSL
openssl rand -base64 32

# 使用urandom
head -c 32 /dev/urandom | base64

# 使用Python
python3 -c "import secrets, base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
```

### 完整配置示例

```bash
# 1. 生成密码哈希
PASSWORD_HASH=$(htpasswd -bnBC 10 "" "MySecurePassword123!" | tr -d ':\n')

# 2. 生成JWT密钥
JWT_SECRET=$(openssl rand -base64 32)

# 3. 设置环境变量
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD_HASH="$PASSWORD_HASH"
export ADMIN_TOKEN_SECRET="$JWT_SECRET"
export ADMIN_TOKEN_DURATION="24h"

# 4. 启动服务器
go run cmd/server/main.go
```

## 相关文档

- [README.md](../README.md) - 项目主文档
- [etc/config.example.yaml](../etc/config.example.yaml) - 配置文件示例
- [Requirements Document](.kiro/specs/admin-dashboard/requirements.md) - 需求文档
- [Design Document](.kiro/specs/admin-dashboard/design.md) - 设计文档
