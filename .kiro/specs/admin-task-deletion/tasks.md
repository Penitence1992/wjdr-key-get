# Implementation Plan: Admin Task Deletion

## Overview

本实现计划将管理后台历史任务删除功能分解为一系列增量式的编码任务。实现顺序遵循从后端到前端的原则，确保每一步都能构建在前一步的基础上，并通过测试验证核心功能。

实现策略：
1. 首先实现数据层（Repository）的删除逻辑和事务管理
2. 然后实现 API 层的请求处理和错误处理
3. 最后实现前端 UI 和交互逻辑
4. 在关键步骤后添加测试任务，尽早发现错误
5. 使用检查点确保增量验证

## Tasks

- [x] 1. 实现 Repository 层删除功能
  - [x] 1.1 在 repository.go 中添加 DeleteTask 接口方法
    - 在 Repository 接口中添加 `DeleteTask(ctx context.Context, taskID int64) error` 方法签名
    - 添加必要的文档注释说明方法功能和事务保证
    - _Requirements: 3.1, 4.1, 5.1_

  - [x] 1.2 在 sqlite_repository.go 中实现 DeleteTask 方法
    - 实现事务管理逻辑（BeginTx, Commit, Rollback）
    - 先删除 gift_codes 表中的关联记录（DELETE FROM gift_codes WHERE task_id = ?）
    - 再删除 gift_code_task 表中的任务记录（DELETE FROM gift_code_task WHERE id = ?）
    - 检查 RowsAffected，如果为 0 则返回 ErrTaskNotFound
    - 添加详细的错误包装和日志记录
    - _Requirements: 3.1, 3.2, 4.1, 5.1, 5.2_

  - [x] 1.3 定义错误类型
    - 在 repository.go 或 errors.go 中定义 `var ErrTaskNotFound = errors.New("task not found")`
    - _Requirements: 3.2_

  - [ ]* 1.4 编写 Repository 层单元测试
    - 测试删除存在的任务（成功场景）
    - 测试删除不存在的任务（应返回 ErrTaskNotFound）
    - 测试删除没有关联兑换码的任务（边界情况）
    - 使用测试数据库或 mock
    - _Requirements: 3.1, 3.2, 4.4_

  - [ ]* 1.5 编写 Repository 层属性测试
    - **Property 6: 删除操作移除任务记录**
    - **Validates: Requirements 3.1, 5.3**
    - 使用 gopter 生成随机任务 ID，验证删除后任务不存在
    - 配置至少 100 次迭代

  - [ ]* 1.6 编写级联删除属性测试
    - **Property 7: 删除操作级联删除兑换码**
    - **Validates: Requirements 4.1**
    - 生成随机任务和关联兑换码，验证删除任务后兑换码也被删除
    - 配置至少 100 次迭代

  - [ ]* 1.7 编写事务原子性属性测试
    - **Property 8: 删除操作的原子性**
    - **Validates: Requirements 3.4, 4.3, 5.2, 5.4**
    - 模拟删除过程中的失败，验证数据回滚到原始状态
    - 配置至少 100 次迭代

- [x] 2. 检查点 - 验证 Repository 层功能
  - 确保所有 Repository 测试通过，如有问题请询问用户

- [ ] 3. 实现 API Handler 层
  - [x] 3.1 在 admin_handlers.go 中添加 DeleteTask 处理函数
    - 实现 `func (h *AdminHandler) DeleteTask(c *gin.Context)` 方法
    - 从 URL 参数解析任务 ID（使用 c.Param("id") 和 strconv.ParseInt）
    - 验证任务 ID 格式，无效时返回 400 错误
    - 调用 repository.DeleteTask 执行删除
    - 根据不同错误类型返回相应的 HTTP 状态码和 JSON 响应
    - 添加错误日志记录
    - _Requirements: 6.1, 6.2, 6.4, 6.5, 8.4_

  - [x] 3.2 处理不同的错误场景
    - 使用 errors.Is 检查 ErrTaskNotFound，返回 404
    - 其他数据库错误返回 500
    - 所有错误响应包含 success: false 和 error 字段
    - 成功响应包含 success: true 和 message 字段
    - _Requirements: 3.2, 6.5, 8.1, 8.2_

  - [x] 3.3 注册 API 路由
    - 在路由配置中添加 `DELETE /api/admin/tasks/:id` 路由
    - 确保路由使用管理员认证中间件
    - _Requirements: 6.1, 6.2_

  - [ ]* 3.4 编写 API Handler 单元测试
    - 测试有效的删除请求（返回 200）
    - 测试无效的任务 ID 格式（返回 400）
    - 测试任务不存在（返回 404）
    - 测试数据库错误（返回 500）
    - 使用 httptest 和 mock repository
    - _Requirements: 6.3, 6.4, 6.5_

  - [ ]* 3.5 编写未授权请求属性测试
    - **Property 9: 未授权请求被拒绝**
    - **Validates: Requirements 6.2**
    - 生成随机任务 ID，使用无效或缺失的认证信息，验证返回 401/403
    - 配置至少 100 次迭代

  - [ ]* 3.6 编写成功响应属性测试
    - **Property 10: 成功删除返回正确状态码**
    - **Validates: Requirements 6.4**
    - 对于任何有效删除，验证返回 200 状态码和成功消息
    - 配置至少 100 次迭代

  - [ ]* 3.7 编写错误响应属性测试
    - **Property 11: 失败删除返回错误信息**
    - **Validates: Requirements 6.5**
    - 对于不同失败场景，验证返回适当的错误状态码和消息
    - 配置至少 100 次迭代

- [ ] 4. 检查点 - 验证 API 层功能
  - 确保所有 API 测试通过，可以使用 curl 或 Postman 手动测试 API 端点

- [x] 5. 实现前端删除按钮 UI
  - [x] 5.1 在 dashboard.js 的 loadCompletedTasksView 函数中添加删除按钮渲染逻辑
    - 创建 renderTaskDeleteButton(task) 函数，返回删除按钮 DOM 元素
    - 为删除按钮添加适当的 CSS 类和图标（如 trash icon 或"删除"文本）
    - 在任务行渲染时调用此函数，将删除按钮添加到每个任务行
    - _Requirements: 1.1, 1.3, 1.4_

  - [x] 5.2 绑定删除按钮点击事件
    - 为每个删除按钮添加 click 事件监听器
    - 事件处理器调用 handleDeleteClick(taskId, taskName) 函数
    - _Requirements: 2.1_

  - [ ]* 5.3 编写删除按钮渲染单元测试
    - 测试空列表不显示删除按钮（边界情况）
    - _Requirements: 1.2_

  - [ ]* 5.4 编写删除按钮渲染属性测试
    - **Property 1: 删除按钮渲染完整性**
    - **Validates: Requirements 1.1**
    - 使用 fast-check 生成随机任务列表，验证每个任务都有删除按钮
    - 配置至少 100 次迭代

- [x] 6. 实现确认对话框
  - [x] 6.1 创建 showDeleteConfirmation 函数
    - 实现 `async function showDeleteConfirmation(taskId, taskName)` 函数
    - 创建模态对话框 DOM 元素
    - 在对话框中显示任务信息（任务 ID 和名称）
    - 添加"确认"和"取消"两个按钮
    - 返回 Promise，根据用户选择 resolve(true) 或 resolve(false)
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 6.2 实现对话框样式
    - 添加必要的 CSS 样式使对话框居中显示
    - 添加遮罩层（overlay）
    - 确保对话框在其他内容之上（z-index）
    - _Requirements: 2.1_

  - [ ]* 6.3 编写确认对话框单元测试
    - 测试对话框包含"确认"和"取消"按钮（示例）
    - _Requirements: 2.3_

  - [ ]* 6.4 编写对话框触发属性测试
    - **Property 2: 点击删除按钮触发确认对话框**
    - **Validates: Requirements 2.1**
    - 对于任何任务，模拟点击删除按钮，验证对话框被显示
    - 配置至少 100 次迭代

  - [ ]* 6.5 编写对话框内容属性测试
    - **Property 3: 确认对话框包含任务信息**
    - **Validates: Requirements 2.2**
    - 对于任何任务，验证对话框内容包含任务 ID 或名称
    - 配置至少 100 次迭代

- [x] 7. 实现删除操作逻辑
  - [x] 7.1 创建 deleteTask API 调用函数
    - 实现 `async function deleteTask(taskId)` 函数
    - 使用 fetch 发送 DELETE 请求到 `/api/admin/tasks/${taskId}`
    - 包含必要的认证头信息
    - 解析 JSON 响应
    - 处理网络错误和 HTTP 错误
    - 返回 `{success: boolean, error?: string}`
    - _Requirements: 6.1, 8.3_

  - [x] 7.2 实现 handleDeleteClick 函数
    - 实现 `async function handleDeleteClick(taskId, taskName)` 函数
    - 调用 showDeleteConfirmation 显示确认对话框
    - 如果用户取消，直接返回，不执行任何操作
    - 如果用户确认，调用 deleteTask 执行删除
    - 在删除过程中禁用删除按钮，显示加载状态
    - 根据删除结果调用 refreshCompletedTasks 或显示错误消息
    - _Requirements: 2.4, 2.5, 7.1, 7.3, 8.1_

  - [x] 7.3 实现错误消息显示
    - 创建 showErrorMessage(message) 函数显示错误提示
    - 可以使用 alert、toast 通知或自定义错误提示组件
    - _Requirements: 8.1, 8.2_

  - [ ]* 7.4 编写取消操作属性测试
    - **Property 4: 取消操作保持数据不变**
    - **Validates: Requirements 2.4**
    - 对于任何任务，模拟点击取消，验证不调用 API 且数据不变
    - 配置至少 100 次迭代

  - [ ]* 7.5 编写确认操作属性测试
    - **Property 5: 确认操作触发删除**
    - **Validates: Requirements 2.5**
    - 对于任何任务，模拟点击确认，验证调用删除 API
    - 配置至少 100 次迭代

- [x] 8. 实现列表刷新逻辑
  - [x] 8.1 修改或创建 refreshCompletedTasks 函数
    - 如果函数已存在，确保它可以被调用来刷新列表
    - 如果不存在，创建函数重新调用 loadCompletedTasksView 或重新获取数据
    - 确保刷新后的列表不包含已删除的任务
    - _Requirements: 7.1, 7.2_

  - [x] 8.2 在删除成功后调用刷新函数
    - 在 handleDeleteClick 中，当 deleteTask 返回 success: true 时调用 refreshCompletedTasks
    - 在删除失败时不调用刷新函数
    - _Requirements: 7.1, 7.3_

  - [ ]* 8.3 编写列表刷新属性测试
    - **Property 12: 成功删除后刷新列表**
    - **Validates: Requirements 7.1**
    - 对于任何成功删除，验证刷新函数被调用
    - 配置至少 100 次迭代

  - [ ]* 8.4 编写刷新后列表内容属性测试
    - **Property 13: 刷新后列表不包含已删除任务**
    - **Validates: Requirements 7.2**
    - 对于任何已删除任务，验证刷新后列表不包含它
    - 配置至少 100 次迭代

  - [ ]* 8.5 编写失败不刷新属性测试
    - **Property 14: 失败删除不刷新列表**
    - **Validates: Requirements 7.3**
    - 对于任何失败删除，验证不调用刷新函数
    - 配置至少 100 次迭代

- [x] 9. 完善错误处理和日志
  - [x] 9.1 添加前端错误处理
    - 捕获网络错误，显示"网络连接失败"提示
    - 根据不同的 HTTP 状态码显示相应的错误消息
    - 在浏览器控制台记录详细错误信息
    - _Requirements: 8.1, 8.2, 8.3_

  - [x] 9.2 添加后端错误日志
    - 在 DeleteTask handler 中添加详细的错误日志
    - 记录任务 ID、错误类型、错误堆栈等信息
    - 使用统一的日志格式
    - _Requirements: 8.4_

  - [ ]* 9.3 编写错误消息显示属性测试
    - **Property 15: 错误消息显示**
    - **Validates: Requirements 8.1, 8.2**
    - 对于任何失败删除，验证显示包含失败原因的错误消息
    - 配置至少 100 次迭代

  - [ ]* 9.4 编写错误日志属性测试
    - **Property 16: 错误日志记录**
    - **Validates: Requirements 8.4**
    - 对于任何错误，验证记录了详细的错误日志
    - 配置至少 100 次迭代

- [ ] 10. 最终检查点 - 端到端测试
  - 手动测试完整的删除流程：点击删除按钮 → 确认对话框 → 删除成功 → 列表刷新
  - 测试取消删除流程：点击删除按钮 → 取消 → 任务保持不变
  - 测试错误场景：删除不存在的任务、网络错误等
  - 确保所有测试通过
  - 如有问题请询问用户

## Notes

- 任务标记 `*` 的为可选测试任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，确保可追溯性
- 检查点任务确保增量验证，尽早发现问题
- 属性测试验证通用正确性属性，单元测试验证具体示例和边界情况
- 实现顺序从后端到前端，确保每一步都有坚实的基础
