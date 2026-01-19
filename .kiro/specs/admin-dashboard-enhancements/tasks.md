# Implementation Plan: Admin Dashboard Enhancements

## Overview

本实现计划将管理后台增强功能分解为一系列增量式的编码任务。实现顺序遵循从底层到上层的原则：首先进行数据库迁移，然后实现数据访问层，接着实现服务层和 API 层，最后更新前端界面。每个任务都包含具体的实现目标和需求引用。

## Tasks

- [x] 1. 数据库迁移：添加任务追踪字段
  - 创建迁移文件 `000002_add_task_tracking_fields.up.sql`
  - 添加 `created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP` 字段
  - 添加 `completed_at TIMESTAMP NULL` 字段
  - 添加 `retry_count INTEGER DEFAULT 0` 字段
  - 添加 `last_error TEXT DEFAULT ''` 字段
  - 创建索引 `idx_task_completed` 用于查询已完成任务
  - 创建对应的 down 迁移文件
  - _Requirements: 2.1, 7.3_

- [x] 2. 数据库迁移：创建通知表
  - 创建迁移文件 `000003_create_notifications_table.up.sql`
  - 创建 `notifications` 表，包含字段：id, channel, title, content, result, status, created_at
  - 添加 status 字段的 CHECK 约束（只允许 'success' 或 'failed'）
  - 创建索引 `idx_notification_created` 用于按时间排序查询
  - 创建对应的 down 迁移文件
  - _Requirements: 5.1, 5.2, 7.4_

- [x] 3. 更新 Repository 接口和模型
  - [x] 3.1 在 `internal/storage/repository.go` 中添加 Notification 模型定义
    - 定义 Notification 结构体，包含所有必需字段
    - 定义 NotificationStatus 常量（Success, Failed）
    - _Requirements: 5.1, 5.2_
  
  - [x] 3.2 在 Repository 接口中添加任务追踪方法
    - 添加 `UpdateTaskRetry(ctx, code, retryCount, lastError) error` 方法
    - 添加 `UpdateTaskComplete(ctx, code, completedAt) error` 方法
    - 添加 `ListCompletedTasks(ctx, limit) ([]*Task, error)` 方法
    - _Requirements: 2.2, 2.3, 2.4, 3.2_
  
  - [x] 3.3 在 Repository 接口中添加通知方法
    - 添加 `SaveNotification(ctx, notification) error` 方法
    - 添加 `ListNotifications(ctx, limit) ([]*Notification, error)` 方法
    - _Requirements: 5.4, 6.2_

- [x] 4. 实现 SQLite Repository 的新方法
  - [x] 4.1 实现 UpdateTaskRetry 方法
    - 编写 SQL UPDATE 语句更新 retry_count 和 last_error
    - 添加错误处理和日志记录
    - _Requirements: 2.4_
  
  - [ ]* 4.2 编写 UpdateTaskRetry 的属性测试
    - **Property 5: Task Retry Increments Counter**
    - **Validates: Requirements 2.4**
  
  - [x] 4.3 实现 UpdateTaskComplete 方法
    - 编写 SQL UPDATE 语句更新 all_done 和 completed_at
    - 添加错误处理和日志记录
    - _Requirements: 2.3_
  
  - [ ]* 4.4 编写 UpdateTaskComplete 的属性测试
    - **Property 4: Task Completion Sets Timestamp**
    - **Validates: Requirements 2.3**
  
  - [x] 4.5 实现 ListCompletedTasks 方法
    - 编写 SQL SELECT 语句查询 all_done = 1 的任务
    - 添加 LIMIT 参数支持
    - 按 completed_at DESC 排序
    - 添加错误处理和日志记录
    - _Requirements: 3.2_
  
  - [ ]* 4.6 编写 ListCompletedTasks 的属性测试
    - **Property 6: Completed Tasks Query Filter**
    - **Validates: Requirements 3.2**
  
  - [x] 4.7 实现 SaveNotification 方法
    - 编写 SQL INSERT 语句保存通知记录
    - 验证 status 字段值（success 或 failed）
    - 添加错误处理和日志记录
    - _Requirements: 5.4, 5.5_
  
  - [ ]* 4.8 编写 SaveNotification 的属性测试
    - **Property 9: Notification Persistence**
    - **Property 10: Notification Record Completeness**
    - **Validates: Requirements 5.4, 5.5, 5.6, 5.7**
  
  - [x] 4.9 实现 ListNotifications 方法
    - 编写 SQL SELECT 语句查询所有通知记录
    - 添加 LIMIT 参数支持
    - 按 created_at DESC 排序
    - 添加错误处理和日志记录
    - _Requirements: 6.2, 6.7_
  
  - [ ]* 4.10 编写 ListNotifications 的属性测试
    - **Property 11: Notification Query Returns All Records**
    - **Property 12: Notification Query Ordering**
    - **Validates: Requirements 6.2, 6.7**

- [x] 5. 实现 Notifier 接口和 WxPusher 实现
  - [x] 5.1 创建 `internal/notification` 包
    - 定义 Notifier 接口
    - 定义 NotificationRequest 结构体
    - 定义 NotificationResult 结构体
    - _Requirements: 4.1, 4.3_
  
  - [x] 5.2 实现 WxPusherNotifier
    - 实现 NewWxPusherNotifier 构造函数
    - 实现 Send 方法，调用 WxPusher API
    - 实现 GetChannel 方法，返回 "wxpusher"
    - 添加 HTTP 客户端配置（超时、重试）
    - 添加错误处理和日志记录
    - _Requirements: 4.2, 4.4, 4.5_
  
  - [ ]* 5.3 编写 WxPusherNotifier 的单元测试
    - 测试成功发送场景
    - 测试失败发送场景（网络错误、API 错误）
    - 使用 mock HTTP 客户端
    - _Requirements: 4.4, 4.5_

- [x] 6. 实现 NotificationService
  - [x] 6.1 创建 `internal/service/notification_service.go`
    - 定义 NotificationService 结构体
    - 实现 NewNotificationService 构造函数
    - 实现 SendAndSave 方法
    - SendAndSave 应该：调用 notifier.Send、根据结果创建 Notification 记录、保存到数据库
    - 添加错误处理和日志记录
    - _Requirements: 5.3, 5.4, 5.5, 5.6, 5.7_
  
  - [ ]* 6.2 编写 NotificationService 的单元测试
    - 测试成功通知的保存
    - 测试失败通知的保存
    - 使用 mock Notifier 和 Repository
    - _Requirements: 5.4, 5.5, 5.6, 5.7_

- [x] 7. 更新 ServiceContext
  - 在 `internal/svc/serviceContext.go` 中添加 NotificationService 字段
  - 在初始化函数中创建 WxPusherNotifier 实例
  - 在初始化函数中创建 NotificationService 实例
  - 从配置文件读取 WxPusher 配置（appToken, uid）
  - _Requirements: 4.6_

- [x] 8. 更新 Job Processor
  - [x] 8.1 修改 `internal/job/getCodeJob.go` 中的 processCodeSafely 方法
    - 在 panic recover 中调用 UpdateTaskRetry
    - 在任务失败时调用 UpdateTaskRetry
    - 在任务成功时调用 UpdateTaskComplete
    - 在任务成功后调用 NotificationService.SendAndSave
    - 移除直接调用 utls.PushWithWxPusher
    - _Requirements: 2.2, 2.3, 2.4, 5.3_
  
  - [ ]* 8.2 编写 Job Processor 的集成测试
    - 测试任务成功完成流程
    - 测试任务失败重试流程
    - 测试通知发送和保存
    - _Requirements: 2.2, 2.3, 2.4, 5.3_

- [x] 9. 添加 API Handlers
  - [x] 9.1 在 `internal/api/admin_handlers.go` 中添加 ListCompletedTasks handler
    - 处理 GET /api/admin/tasks/completed
    - 从 query 参数读取 limit（默认 100）
    - 调用 repository.ListCompletedTasks
    - 返回 JSON 响应
    - 添加错误处理和日志记录
    - _Requirements: 3.2_
  
  - [x] 9.2 在 `internal/api/admin_handlers.go` 中添加 ListNotifications handler
    - 处理 GET /api/admin/notifications
    - 从 query 参数读取 limit（默认 100）
    - 调用 repository.ListNotifications
    - 返回 JSON 响应
    - 添加错误处理和日志记录
    - _Requirements: 6.2_
  
  - [x] 9.3 在路由配置中注册新的 API 端点
    - 在 `cmd/server/main.go` 中添加路由
    - GET /api/admin/tasks/completed → ListCompletedTasks
    - GET /api/admin/notifications → ListNotifications
    - 确保这些端点需要认证
    - _Requirements: 3.2, 6.2_

- [x] 10. 简化前端用户管理界面
  - [x] 10.1 修改 `cmd/server/static/admin/dashboard.js` 中的 loadUsersView 函数
    - 移除 nickname、kid、avatar_image 输入框
    - 只保留 FID 输入框
    - 更新表单 HTML
    - _Requirements: 1.1_
  
  - [x] 10.2 修改 addUser 函数
    - 只从表单读取 FID
    - 设置 nickname 为空字符串
    - 设置 kid 为 0
    - 设置 avatar_image 为空字符串
    - _Requirements: 1.2, 1.3_
  
  - [ ]* 10.3 编写 FID 验证的属性测试
    - **Property 1: FID Validation Rejects Invalid Input**
    - **Property 2: User Creation Sets Default Values**
    - **Validates: Requirements 1.2, 1.3**

- [x] 11. 增强前端任务监控界面
  - [x] 11.1 修改 `cmd/server/static/admin/dashboard.js` 中的 loadTasksView 函数
    - 在表格中添加"创建时间"、"完成时间"、"重试次数"列
    - 格式化时间戳显示
    - 对未完成任务的 completed_at 显示 "-"
    - 添加"历史任务"按钮
    - _Requirements: 2.5, 2.6, 3.1_
  
  - [x] 11.2 实现 loadCompletedTasksView 函数
    - 调用 GET /api/admin/tasks/completed API
    - 显示已完成任务列表
    - 显示所有任务字段（包括新增的时间戳和重试次数）
    - 添加"返回当前任务"按钮
    - 处理空列表情况
    - _Requirements: 3.2, 3.3, 3.4, 3.5_

- [x] 12. 添加前端通知历史界面
  - [x] 12.1 在 `cmd/server/static/admin/dashboard.html` 中添加通知历史视图
    - 在侧边栏导航中添加"通知历史"选项
    - 添加 notifications-view div
    - 添加 notifications-message div
    - 添加 notifications-content div
    - _Requirements: 6.1_
  
  - [x] 12.2 在 `cmd/server/static/admin/dashboard.js` 中实现 loadNotificationsView 函数
    - 调用 GET /api/admin/notifications API
    - 显示通知列表表格
    - 显示字段：渠道、标题、内容、时间、状态、结果
    - 对长内容进行截断（超过 50 字符）
    - 使用 title 属性显示完整内容
    - 根据 status 显示不同的状态标识
    - 处理空列表情况
    - _Requirements: 6.2, 6.3, 6.4, 6.5, 6.6_
  
  - [x] 12.3 更新 showView 函数
    - 添加 'notifications' 视图的处理
    - 调用 loadNotificationsView
    - _Requirements: 6.1_

- [x] 13. 更新配置文件
  - 在 `etc/config.yaml` 和 `etc/config.example.yaml` 中添加通知配置
  - 添加 notification 部分，包含 wxpusher 配置（appToken, uid）
  - 添加配置注释说明
  - _Requirements: 4.6_

- [x] 14. 运行数据库迁移
  - 执行迁移脚本添加任务追踪字段
  - 执行迁移脚本创建通知表
  - 验证数据库 schema 正确性
  - 为现有任务记录设置默认值（created_at = CURRENT_TIMESTAMP, retry_count = 0）
  - _Requirements: 7.3, 7.4, 7.5_

- [ ] 15. 集成测试和验证
  - [ ] 15.1 测试简化的用户管理界面
    - 验证只显示 FID 输入框
    - 测试添加用户功能
    - 验证默认值设置
    - _Requirements: 1.1, 1.2, 1.3_
  
  - [ ] 15.2 测试增强的任务监控
    - 验证任务列表显示新字段
    - 测试历史任务查看功能
    - 验证时间戳和重试次数显示
    - _Requirements: 2.5, 2.6, 3.1, 3.2, 3.3_
  
  - [ ] 15.3 测试通知系统
    - 创建测试任务并完成
    - 验证通知发送
    - 验证通知保存到数据库
    - 验证通知历史界面显示
    - _Requirements: 5.3, 5.4, 6.2, 6.3_
  
  - [ ] 15.4 端到端测试
    - 完整流程：添加用户 → 创建任务 → 任务执行 → 通知发送 → 查看历史
    - 验证所有功能正常工作
    - _Requirements: All_

- [ ] 16. 最终检查点
  - 确保所有测试通过
  - 验证代码符合项目规范
  - 检查日志输出是否合理
  - 验证错误处理是否完善
  - 询问用户是否有问题或需要调整

## Notes

- 任务按照从底层到上层的顺序组织：数据库 → Repository → Service → API → Frontend
- 标记为 `*` 的任务是可选的测试任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，便于追溯
- 属性测试使用 gopter 库，每个测试至少运行 100 次迭代
- 集成测试确保各组件正确协作
- 最终检查点确保所有功能正常工作
