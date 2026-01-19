# Implementation Plan: Admin Dashboard

## Overview

本实现计划将管理后台功能分解为增量式的开发任务。每个任务都建立在前面任务的基础上，确保代码逐步集成。实现将遵循以下顺序：配置扩展 → 认证服务 → 认证中间件 → API处理器 → 前端界面。

## Tasks

- [x] 1. 扩展配置结构支持管理员设置
  - 在`internal/config/config.go`中添加`AdminConfig`结构
  - 添加管理员用户名、密码哈希、JWT密钥和令牌有效期字段
  - 实现从环境变量覆盖管理员配置的逻辑
  - 添加管理员配置的验证规则
  - 更新配置示例文件`etc/config.example.yaml`
  - _Requirements: 8.1, 8.2, 8.3_

- [ ]* 1.1 编写配置加载的属性测试
  - **Property 17: 配置加载正确性**
  - **Property 18: 环境变量优先级**
  - **Validates: Requirements 8.1, 8.3**

- [ ]* 1.2 编写配置验证的单元测试
  - 测试默认配置
  - 测试无效配置被拒绝
  - _Requirements: 8.2_

- [-] 2. 实现认证服务
  - [x] 2.1 创建认证服务接口和实现
    - 在`internal/auth/`目录下创建`service.go`
    - 实现`AuthService`接口（ValidateCredentials, GenerateToken, ValidateToken）
    - 使用`golang.org/x/crypto/bcrypt`进行密码验证
    - 使用`github.com/golang-jwt/jwt/v5`生成和验证JWT令牌
    - 定义`Claims`结构包含用户名和标准声明
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [ ]* 2.2 编写凭证验证的属性测试
    - **Property 1: 有效凭证认证成功**
    - **Property 2: 无效凭证认证失败**
    - **Validates: Requirements 1.1, 1.2**

  - [ ]* 2.3 编写令牌生成和验证的属性测试
    - **Property 3: 过期令牌被拒绝**
    - **Property 4: 认证令牌往返一致性**
    - **Validates: Requirements 1.3, 1.4, 6.1**

  - [ ]* 2.4 编写认证服务的单元测试
    - 测试空密码边缘情况
    - 测试特殊字符处理
    - 测试令牌过期时间设置
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 3. 实现认证中间件
  - 在`internal/api/middleware.go`中添加`AuthMiddleware`函数
  - 从Authorization头提取Bearer令牌
  - 验证令牌格式和有效性
  - 将管理员用户名存入Gin上下文
  - 处理各种错误情况（缺少令牌、无效令牌、过期令牌）
  - _Requirements: 1.5, 6.1, 6.2, 6.3, 6.4_

- [ ]* 3.1 编写认证中间件的属性测试
  - **Property 5: 未认证请求被拒绝**
  - **Property 16: 格式错误的令牌返回400**
  - **Validates: Requirements 1.5, 6.2, 6.4**

- [ ]* 3.2 编写认证中间件的单元测试
  - 测试Authorization头提取
  - 测试不同的错误场景
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [x] 4. 实现管理后台API处理器
  - [x] 4.1 创建AdminHandlers结构
    - 在`internal/api/`目录下创建`admin_handlers.go`
    - 定义`AdminHandlers`结构包含authService、repository和logger
    - 创建`NewAdminHandlers`构造函数
    - _Requirements: 所有API相关需求_

  - [x] 4.2 实现登录处理器
    - 实现`Login`方法处理POST /api/admin/login
    - 定义`LoginRequest`和`LoginResponse`结构
    - 验证请求参数
    - 调用authService验证凭证和生成令牌
    - 返回令牌和过期时间
    - _Requirements: 1.1, 1.2, 1.4_

  - [ ]* 4.3 编写登录处理器的单元测试
    - 测试有效登录
    - 测试无效凭证
    - 测试请求验证
    - _Requirements: 1.1, 1.2_

  - [x] 4.3 实现用户列表处理器
    - 实现`ListUsers`方法处理GET /api/admin/users
    - 调用repository.ListUsers获取所有用户
    - 返回用户列表（包含FID、昵称、头像、创建时间）
    - 处理空列表情况
    - _Requirements: 2.1, 2.2, 2.3_

  - [ ]* 4.4 编写用户列表的属性测试
    - **Property 6: 用户列表完整性**
    - **Property 7: API响应包含必需字段**（用户部分）
    - **Validates: Requirements 2.1, 2.2**

  - [x] 4.5 实现添加用户处理器
    - 实现`AddUser`方法处理POST /api/admin/users
    - 定义`AddUserRequest`结构
    - 验证请求参数
    - 调用repository.SaveUser保存用户
    - _Requirements: 2.4, 2.5_

  - [ ]* 4.6 编写添加用户的属性测试
    - **Property 8: 用户添加幂等性**
    - **Validates: Requirements 2.5**

  - [x] 4.7 实现用户兑换记录处理器
    - 实现`GetUserGiftCodes`方法处理GET /api/admin/users/:fid/codes
    - 从URL参数提取FID
    - 调用repository.ListGiftCodesByFID获取兑换记录
    - 返回兑换记录列表（包含激活码、状态、时间、结果）
    - 处理用户不存在和无记录情况
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

  - [ ]* 4.8 编写兑换记录的属性测试
    - **Property 9: 用户兑换记录查询正确性**
    - **Property 10: 兑换记录时间排序**
    - **Property 7: API响应包含必需字段**（兑换记录部分）
    - **Validates: Requirements 3.1, 3.2, 3.5**

  - [x] 4.9 实现任务列表处理器
    - 实现`ListTasks`方法处理GET /api/admin/tasks
    - 调用repository.ListPendingTasks获取任务
    - 返回任务列表（包含代码、状态、重试次数、错误、时间戳）
    - 处理空列表情况
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ]* 4.10 编写任务列表的属性测试
    - **Property 11: 任务列表过滤正确性**
    - **Property 12: 已完成任务包含完成时间**
    - **Property 13: 失败任务包含错误信息**
    - **Property 7: API响应包含必需字段**（任务部分）
    - **Validates: Requirements 4.1, 4.2, 4.4, 4.5**

  - [x] 4.11 实现添加兑换码处理器
    - 实现`AddGiftCode`方法处理POST /api/admin/tasks
    - 定义`AddGiftCodeRequest`结构
    - 验证兑换码格式（非空、非纯空白）
    - 调用repository.CreateTask创建任务
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [ ]* 4.12 编写添加兑换码的属性测试
    - **Property 14: 有效兑换码创建任务成功**
    - **Property 15: 无效兑换码被拒绝**
    - **Validates: Requirements 5.1, 5.2, 5.4**

- [x] 5. Checkpoint - 确保所有后端测试通过
  - 运行所有单元测试和属性测试
  - 确认测试覆盖率达到要求
  - 如有问题请询问用户

- [x] 6. 集成API路由到服务器
  - 在`cmd/server/main.go`中初始化AuthService
  - 创建AdminHandlers实例
  - 注册管理后台API路由（/api/admin/*）
  - 对受保护的路由应用AuthMiddleware
  - 保持现有路由不变
  - _Requirements: 所有API需求_

- [ ]* 6.1 编写API集成测试
  - 使用httptest测试完整的HTTP流程
  - 测试登录获取令牌
  - 测试使用令牌访问受保护资源
  - 测试CORS和中间件集成
  - _Requirements: 1.1, 1.5, 6.1_

- [x] 7. 实现前端登录页面
  - 创建`cmd/server/static/admin/login.html`
  - 实现登录表单（用户名、密码输入）
  - 使用Fetch API调用POST /api/admin/login
  - 将JWT令牌存储到LocalStorage
  - 登录成功后重定向到控制面板
  - 显示错误信息（如果登录失败）
  - 添加基本CSS样式
  - _Requirements: 7.1, 7.4, 7.5_

- [x] 8. 实现前端控制面板
  - [x] 8.1 创建控制面板主页面
    - 创建`cmd/server/static/admin/dashboard.html`
    - 实现导航菜单（用户管理、任务监控）
    - 检查LocalStorage中的令牌，如果没有则重定向到登录页
    - 添加退出登录功能
    - _Requirements: 7.2, 7.3_

  - [x] 8.2 实现用户列表视图
    - 调用GET /api/admin/users获取用户列表
    - 在表格中显示用户信息（FID、昵称、头像、创建时间）
    - 实现添加用户表单
    - 调用POST /api/admin/users添加新用户
    - 显示操作反馈（成功/失败消息）
    - _Requirements: 2.1, 2.2, 2.4, 7.4, 7.5_

  - [x] 8.3 实现用户详情视图
    - 点击用户时显示该用户的兑换记录
    - 调用GET /api/admin/users/:fid/codes
    - 在表格中显示兑换记录（激活码、状态、时间、结果）
    - 处理无记录情况
    - _Requirements: 3.1, 3.2, 3.4_

  - [x] 8.4 实现任务监控视图
    - 调用GET /api/admin/tasks获取任务列表
    - 在表格中显示任务信息（代码、状态、重试次数、错误、时间）
    - 实现自动刷新（每30秒）
    - 处理空列表情况
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 8.5 实现添加兑换码表单
    - 在控制面板添加兑换码输入表单
    - 调用POST /api/admin/tasks添加兑换码
    - 验证输入（非空）
    - 显示操作反馈
    - 添加成功后刷新任务列表
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 7.4, 7.5_

- [x] 9. 实现前端认证拦截器
  - 在所有API请求中自动添加Authorization头
  - 从LocalStorage读取JWT令牌
  - 处理401错误（令牌过期），自动重定向到登录页
  - 清除LocalStorage中的过期令牌
  - _Requirements: 1.3, 1.5, 6.1, 6.2_

- [x] 10. 添加前端样式和用户体验优化
  - 创建统一的CSS样式文件
  - 实现响应式布局（支持移动设备）
  - 添加加载指示器（API请求期间）
  - 优化表单验证和错误提示
  - 添加确认对话框（删除操作等）
  - _Requirements: 7.4, 7.5_

- [x] 11. 更新静态文件路由
  - 在`cmd/server/main.go`中添加/admin/*路由
  - 处理/admin/重定向（已登录→控制面板，未登录→登录页）
  - 确保静态文件正确提供
  - _Requirements: 7.1, 7.2_

- [x] 12. 更新文档和配置示例
  - 更新README.md添加管理后台使用说明
  - 更新etc/config.example.yaml添加管理员配置示例
  - 创建初始管理员密码设置指南
  - 添加JWT密钥生成说明
  - _Requirements: 8.1, 8.2_

- [ ] 13. Final Checkpoint - 端到端测试
  - 手动测试完整的用户流程
  - 测试登录→查看用户→查看兑换记录→添加兑换码→查看任务
  - 测试错误场景（无效凭证、过期令牌）
  - 测试不同浏览器的兼容性
  - 确保所有测试通过
  - 如有问题请询问用户

## Notes

- 标记为`*`的任务是可选的测试任务，可以跳过以加快MVP开发
- 每个任务都引用了具体的需求条款以便追溯
- Checkpoint任务确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证特定示例和边缘情况
- 前端实现采用渐进式方法，先实现核心功能再优化用户体验
