# Requirements Document

## Introduction

本文档定义了管理后台增强功能的需求。该功能旨在简化用户管理界面、增强任务监控能力、添加历史任务查看功能，以及实现通知系统的抽象和历史记录管理。

## Glossary

- **System**: 礼品码管理系统
- **Admin_Dashboard**: 管理后台界面
- **User_Management_Interface**: 用户管理界面
- **Task_Monitor**: 任务监控界面
- **Notification_System**: 通知系统
- **Database**: SQLite 数据库
- **FID**: 用户唯一标识符（Farcaster ID）
- **Task**: 礼品码兑换任务
- **Notification**: 系统发送的通知消息
- **Notification_Channel**: 通知渠道（如 WxPusher）
- **History_Task**: 已完成的历史任务
- **Notification_Record**: 通知历史记录

## Requirements

### Requirement 1: 简化用户管理界面

**User Story:** 作为管理员，我希望在添加用户时只需要输入 FID，这样可以简化操作流程并减少输入错误。

#### Acceptance Criteria

1. WHEN 管理员访问用户管理界面 THEN THE Admin_Dashboard SHALL 显示简化的添加用户表单，该表单仅包含 FID 输入框
2. WHEN 管理员在添加用户表单中输入 FID 并提交 THEN THE System SHALL 验证 FID 格式是否非空且非纯空白字符
3. WHEN FID 验证通过 THEN THE System SHALL 创建用户记录，其中 nickname、kid 和 avatar_image 字段使用默认值或空值
4. WHEN 用户添加成功 THEN THE Admin_Dashboard SHALL 显示成功消息并刷新用户列表
5. WHEN FID 验证失败 THEN THE Admin_Dashboard SHALL 显示错误消息并保持表单内容

### Requirement 2: 任务监控字段增强

**User Story:** 作为管理员，我希望在任务监控中看到任务的创建时间、完成时间和重试次数，以便更好地追踪任务执行情况。

#### Acceptance Criteria

1. THE Database SHALL 在 gift_code_task 表中包含 created_at、completed_at 和 retry_count 字段
2. WHEN 创建新任务 THEN THE System SHALL 自动设置 created_at 为当前时间戳
3. WHEN 任务完成 THEN THE System SHALL 设置 completed_at 为完成时的时间戳
4. WHEN 任务执行失败需要重试 THEN THE System SHALL 增加 retry_count 计数
5. WHEN 管理员访问任务监控界面 THEN THE Task_Monitor SHALL 显示每个任务的创建时间、完成时间和重试次数
6. WHEN 任务尚未完成 THEN THE Task_Monitor SHALL 对 completed_at 字段显示占位符（如 "-"）

### Requirement 3: 历史任务查看功能

**User Story:** 作为管理员，我希望能够查看已完成的历史任务，以便审计和追溯过往的兑换记录。

#### Acceptance Criteria

1. WHEN 管理员访问任务监控界面 THEN THE Task_Monitor SHALL 显示"历史任务"入口按钮
2. WHEN 管理员点击"历史任务"按钮 THEN THE System SHALL 查询所有 all_done 为 true 的任务记录
3. WHEN 历史任务查询完成 THEN THE Admin_Dashboard SHALL 显示历史任务列表，包含兑换码、状态、重试次数、错误信息、创建时间和完成时间
4. WHEN 历史任务列表为空 THEN THE Admin_Dashboard SHALL 显示"暂无历史任务"提示
5. WHEN 管理员在历史任务视图中 THEN THE Admin_Dashboard SHALL 提供返回当前任务列表的按钮

### Requirement 4: 通知系统抽象

**User Story:** 作为系统架构师，我希望对通知方法进行抽象，以便支持多种通知渠道并便于扩展。

#### Acceptance Criteria

1. THE System SHALL 定义 Notifier 接口，该接口包含 Send 方法用于发送通知
2. THE System SHALL 实现 WxPusher 通知器，该通知器实现 Notifier 接口
3. WHEN 调用 Notifier.Send 方法 THEN THE Notifier SHALL 接收通知渠道、标题、内容等参数
4. WHEN 通知发送成功 THEN THE Notifier SHALL 返回成功状态和通知结果
5. WHEN 通知发送失败 THEN THE Notifier SHALL 返回失败状态和错误信息
6. THE System SHALL 支持通过配置注入不同的 Notifier 实现

### Requirement 5: 通知结果持久化

**User Story:** 作为管理员，我希望系统能够保存所有通知结果到数据库，以便追踪通知发送历史。

#### Acceptance Criteria

1. THE Database SHALL 包含 notifications 表，该表存储通知历史记录
2. THE notifications 表 SHALL 包含以下字段：id、channel、title、content、result、status、created_at
3. WHEN 任务完成后发送通知 THEN THE System SHALL 调用 Notifier.Send 方法发送通知
4. WHEN 通知发送完成（无论成功或失败）THEN THE System SHALL 将通知记录保存到 notifications 表
5. WHEN 保存通知记录 THEN THE System SHALL 记录通知渠道、标题、内容、发送结果、状态和创建时间
6. WHEN 通知发送成功 THEN THE System SHALL 将 status 字段设置为 "success"
7. WHEN 通知发送失败 THEN THE System SHALL 将 status 字段设置为 "failed" 并记录错误信息到 result 字段

### Requirement 6: 通知历史展示

**User Story:** 作为管理员，我希望在页面中查看所有历史通知内容，以便了解系统的通知发送情况。

#### Acceptance Criteria

1. WHEN 管理员访问管理后台 THEN THE Admin_Dashboard SHALL 在导航菜单中显示"通知历史"选项
2. WHEN 管理员点击"通知历史"选项 THEN THE System SHALL 查询 notifications 表中的所有记录
3. WHEN 通知历史查询完成 THEN THE Admin_Dashboard SHALL 显示通知列表，包含渠道、标题、内容、时间和通知结果
4. WHEN 通知列表为空 THEN THE Admin_Dashboard SHALL 显示"暂无通知记录"提示
5. WHEN 显示通知内容 THEN THE Admin_Dashboard SHALL 对长内容进行截断并提供查看完整内容的方式
6. WHEN 显示通知结果 THEN THE Admin_Dashboard SHALL 根据 status 字段显示不同的状态标识（成功/失败）
7. WHEN 通知历史列表加载 THEN THE System SHALL 按创建时间倒序排列通知记录

### Requirement 7: 数据库迁移支持

**User Story:** 作为开发者，我希望通过数据库迁移脚本来添加新字段和新表，以便保持数据库结构的版本控制。

#### Acceptance Criteria

1. THE System SHALL 创建新的数据库迁移文件以添加任务表的新字段
2. THE System SHALL 创建新的数据库迁移文件以添加 notifications 表
3. WHEN 执行迁移脚本 THEN THE System SHALL 在 gift_code_task 表中添加 created_at、completed_at 和 retry_count 字段
4. WHEN 执行迁移脚本 THEN THE System SHALL 创建 notifications 表及其所有字段
5. WHEN 迁移脚本执行 THEN THE System SHALL 为现有任务记录设置合理的默认值
6. WHEN 迁移脚本执行失败 THEN THE System SHALL 回滚更改并报告错误
