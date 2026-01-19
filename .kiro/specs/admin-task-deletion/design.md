# Design Document: Admin Task Deletion

## Overview

本设计文档描述了管理后台历史任务删除功能的技术实现方案。该功能允许管理员删除已完成的任务及其关联的兑换码数据，确保数据删除的原子性和一致性。

系统采用三层架构：
- **前端层**：使用原生 JavaScript 实现用户界面和交互逻辑
- **API 层**：使用 Go + Gin 框架提供 RESTful API 端点
- **数据层**：使用 SQLite 数据库和仓储模式管理数据持久化

核心设计原则：
1. **原子性**：使用数据库事务确保删除操作的原子性
2. **用户友好**：提供确认对话框防止误删除
3. **错误处理**：完善的错误处理和用户反馈机制
4. **权限控制**：确保只有授权管理员可以执行删除操作

## Architecture

系统采用经典的 MVC 架构模式，分为以下层次：

```
┌─────────────────────────────────────────┐
│         Frontend (JavaScript)           │
│  - dashboard.js (UI Logic)              │
│  - Confirmation Dialog                  │
│  - Event Handlers                       │
└──────────────┬──────────────────────────┘
               │ HTTP DELETE Request
               ▼
┌─────────────────────────────────────────┐
│      API Layer (Go + Gin)               │
│  - admin_handlers.go                    │
│  - Authentication Middleware            │
│  - Request Validation                   │
└──────────────┬──────────────────────────┘
               │ Repository Interface
               ▼
┌─────────────────────────────────────────┐
│    Repository Layer (Go)                │
│  - sqlite_repository.go                 │
│  - Transaction Management               │
│  - Data Access Logic                    │
└──────────────┬──────────────────────────┘
               │ SQL Queries
               ▼
┌─────────────────────────────────────────┐
│      Database (SQLite)                  │
│  - gift_code_task table                 │
│  - gift_codes table                     │
└─────────────────────────────────────────┘
```

### 交互流程

1. **用户触发删除**：管理员点击历史任务列表中的删除按钮
2. **显示确认对话框**：前端显示确认对话框，包含任务信息
3. **发送删除请求**：用户确认后，前端发送 DELETE 请求到 API
4. **权限验证**：API 层验证管理员权限
5. **执行删除操作**：仓储层在事务中删除任务和关联兑换码
6. **返回结果**：API 返回操作结果
7. **更新 UI**：前端根据结果刷新列表或显示错误

## Components and Interfaces

### 1. Frontend Component (dashboard.js)

**职责**：处理用户交互、显示确认对话框、发送 API 请求、更新 UI

**关键函数**：

```javascript
// 在历史任务列表中添加删除按钮
function renderTaskDeleteButton(task) {
  // 创建删除按钮元素
  // 绑定点击事件处理器
  // 返回按钮 DOM 元素
}

// 显示删除确认对话框
function showDeleteConfirmation(taskId, taskName) {
  // 创建模态对话框
  // 显示任务信息
  // 提供确认和取消按钮
  // 返回 Promise<boolean>
}

// 执行删除操作
async function deleteTask(taskId) {
  // 发送 DELETE 请求到 /api/admin/tasks/:id
  // 处理响应
  // 返回 Promise<{success: boolean, error?: string}>
}

// 刷新历史任务列表
function refreshCompletedTasks() {
  // 重新加载历史任务数据
  // 更新 DOM
}

// 删除按钮点击处理器
async function handleDeleteClick(taskId, taskName) {
  // 1. 显示确认对话框
  // 2. 如果用户确认，调用 deleteTask
  // 3. 根据结果刷新列表或显示错误
}
```

**接口**：
- 输入：任务 ID、任务名称
- 输出：UI 更新、错误消息显示
- API 调用：DELETE /api/admin/tasks/:id

### 2. API Handler (admin_handlers.go)

**职责**：处理 HTTP 请求、验证权限、调用仓储层、返回响应

**函数签名**：

```go
// DeleteTask 处理删除任务的 HTTP 请求
func (h *AdminHandler) DeleteTask(c *gin.Context) {
  // 1. 从 URL 参数获取任务 ID
  // 2. 验证管理员权限（通过中间件）
  // 3. 调用仓储层删除任务
  // 4. 返回 JSON 响应
}
```

**接口**：
- HTTP 方法：DELETE
- 路径：/api/admin/tasks/:id
- 请求参数：id (URL 参数)
- 响应格式：
  ```json
  {
    "success": true,
    "message": "任务删除成功"
  }
  ```
  或
  ```json
  {
    "success": false,
    "error": "错误描述"
  }
  ```

**错误码**：
- 200: 删除成功
- 400: 无效的任务 ID
- 401/403: 未授权
- 404: 任务不存在
- 500: 服务器内部错误

### 3. Repository Interface

**职责**：定义数据访问接口

在 `internal/storage/repository.go` 中添加：

```go
type Repository interface {
  // 现有方法...
  
  // DeleteTask 删除任务及其关联的兑换码
  // 在事务中执行，确保原子性
  DeleteTask(ctx context.Context, taskID int64) error
}
```

### 4. Repository Implementation (sqlite_repository.go)

**职责**：实现数据删除逻辑，管理数据库事务

**函数实现**：

```go
// DeleteTask 删除任务及其关联的兑换码
func (r *SQLiteRepository) DeleteTask(ctx context.Context, taskID int64) error {
  // 1. 开始事务
  tx, err := r.db.BeginTx(ctx, nil)
  if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
  }
  defer tx.Rollback() // 如果没有提交，自动回滚
  
  // 2. 删除关联的兑换码
  _, err = tx.ExecContext(ctx, 
    "DELETE FROM gift_codes WHERE task_id = ?", 
    taskID)
  if err != nil {
    return fmt.Errorf("failed to delete gift codes: %w", err)
  }
  
  // 3. 删除任务记录
  result, err := tx.ExecContext(ctx, 
    "DELETE FROM gift_code_task WHERE id = ?", 
    taskID)
  if err != nil {
    return fmt.Errorf("failed to delete task: %w", err)
  }
  
  // 4. 检查任务是否存在
  rowsAffected, err := result.RowsAffected()
  if err != nil {
    return fmt.Errorf("failed to get rows affected: %w", err)
  }
  if rowsAffected == 0 {
    return ErrTaskNotFound
  }
  
  // 5. 提交事务
  if err = tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit transaction: %w", err)
  }
  
  return nil
}
```

**错误定义**：

```go
var (
  ErrTaskNotFound = errors.New("task not found")
)
```

## Data Models

### 数据库表结构

**gift_code_task 表**（已存在）：
- id: INTEGER PRIMARY KEY
- 其他字段...（任务相关信息）

**gift_codes 表**（已存在）：
- id: INTEGER PRIMARY KEY
- task_id: INTEGER (外键，关联到 gift_code_task.id)
- 其他字段...（兑换码相关信息）

### 数据关系

```
gift_code_task (1) ──< (N) gift_codes
     │                      │
     └──────────────────────┘
        通过 task_id 关联
```

删除操作顺序：
1. 先删除 gift_codes 表中 task_id 匹配的所有记录
2. 再删除 gift_code_task 表中的任务记录

这个顺序确保不会违反外键约束（如果存在）。

### API 数据模型

**请求**：
- URL 参数：taskID (int64)

**响应**：

```go
type DeleteTaskResponse struct {
  Success bool   `json:"success"`
  Message string `json:"message,omitempty"`
  Error   string `json:"error,omitempty"`
}
```


## Correctness Properties

*属性（Property）是系统在所有有效执行中都应该保持为真的特征或行为——本质上是关于系统应该做什么的形式化陈述。属性是人类可读规范和机器可验证正确性保证之间的桥梁。*

基于需求文档中的验收标准，我们定义以下正确性属性：

### Property 1: 删除按钮渲染完整性

*对于任何*非空的任务列表，渲染函数应该为每个任务生成一个删除按钮元素。

**Validates: Requirements 1.1**

### Property 2: 点击删除按钮触发确认对话框

*对于任何*任务，当点击其删除按钮时，系统应该显示确认对话框。

**Validates: Requirements 2.1**

### Property 3: 确认对话框包含任务信息

*对于任何*任务，确认对话框的内容应该包含该任务的关键信息（任务 ID 或任务名称）。

**Validates: Requirements 2.2**

### Property 4: 取消操作保持数据不变

*对于任何*任务，在确认对话框中点击"取消"后，该任务的数据应该保持不变，且不应该调用删除 API。

**Validates: Requirements 2.4**

### Property 5: 确认操作触发删除

*对于任何*任务，在确认对话框中点击"确认"后，系统应该向 API 发送删除请求。

**Validates: Requirements 2.5**

### Property 6: 删除操作移除任务记录

*对于任何*存在于数据库中的任务，成功执行删除操作后，该任务记录应该不再存在于 gift_code_task 表中。

**Validates: Requirements 3.1, 5.3**

### Property 7: 删除操作级联删除兑换码

*对于任何*有关联兑换码的任务，删除该任务后，gift_codes 表中所有 task_id 匹配的记录都应该被删除。

**Validates: Requirements 4.1**

### Property 8: 删除操作的原子性

*对于任何*删除操作，如果任何步骤（删除兑换码或删除任务）失败，则所有数据应该保持原始状态，不应该有部分删除的情况。

**Validates: Requirements 3.4, 4.3, 5.2, 5.4**

### Property 9: 未授权请求被拒绝

*对于任何*没有管理员权限的删除请求，API 应该返回 401 或 403 状态码，且不应该执行删除操作。

**Validates: Requirements 6.2**

### Property 10: 成功删除返回正确状态码

*对于任何*有效的删除操作，API 应该返回 200 状态码和成功消息。

**Validates: Requirements 6.4**

### Property 11: 失败删除返回错误信息

*对于任何*失败的删除操作（如任务不存在、数据库错误），API 应该返回适当的错误状态码（404 或 500）和描述性错误消息。

**Validates: Requirements 6.5**

### Property 12: 成功删除后刷新列表

*对于任何*成功的删除操作，系统应该自动调用列表刷新函数。

**Validates: Requirements 7.1**

### Property 13: 刷新后列表不包含已删除任务

*对于任何*被删除的任务，刷新后的任务列表不应该包含该任务。

**Validates: Requirements 7.2**

### Property 14: 失败删除不刷新列表

*对于任何*失败的删除操作，系统应该显示错误消息，但不应该刷新任务列表。

**Validates: Requirements 7.3**

### Property 15: 错误消息显示

*对于任何*失败的删除操作，系统应该向用户显示包含失败原因的错误消息。

**Validates: Requirements 8.1, 8.2**

### Property 16: 错误日志记录

*对于任何*删除操作中发生的错误，系统应该记录详细的错误日志。

**Validates: Requirements 8.4**

## Error Handling

### 前端错误处理

1. **网络错误**：
   - 捕获 fetch 请求的网络异常
   - 显示用户友好的错误消息："网络连接失败，请检查网络后重试"
   - 不刷新任务列表

2. **API 错误响应**：
   - 解析 API 返回的错误消息
   - 根据状态码显示相应提示：
     - 404: "任务不存在或已被删除"
     - 401/403: "权限不足，请重新登录"
     - 500: "服务器错误，请稍后重试"
   - 记录错误到浏览器控制台

3. **用户操作错误**：
   - 防止重复点击（添加加载状态）
   - 在删除过程中禁用删除按钮

### 后端错误处理

1. **参数验证错误**：
   ```go
   taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
   if err != nil {
     c.JSON(400, gin.H{
       "success": false,
       "error": "无效的任务 ID",
     })
     return
   }
   ```

2. **任务不存在错误**：
   ```go
   if errors.Is(err, storage.ErrTaskNotFound) {
     c.JSON(404, gin.H{
       "success": false,
       "error": "任务不存在",
     })
     return
   }
   ```

3. **数据库错误**：
   ```go
   if err != nil {
     log.Printf("Failed to delete task %d: %v", taskID, err)
     c.JSON(500, gin.H{
       "success": false,
       "error": "删除任务失败，请稍后重试",
     })
     return
   }
   ```

4. **事务错误**：
   - 所有数据库操作在事务中执行
   - 任何步骤失败自动回滚
   - 记录详细错误日志用于调试

### 错误日志格式

```go
log.Printf("[ERROR] DeleteTask: taskID=%d, error=%v, stack=%s", 
  taskID, err, debug.Stack())
```

## Testing Strategy

本功能采用双重测试策略，结合单元测试和基于属性的测试，确保全面的代码覆盖和正确性验证。

### 测试方法论

1. **单元测试**：用于验证特定示例、边界情况和错误条件
2. **基于属性的测试**：用于验证跨所有输入的通用属性
3. **集成测试**：用于验证组件之间的交互

这两种方法是互补的：单元测试捕获具体的错误，基于属性的测试验证通用正确性。

### 基于属性的测试配置

**测试库选择**：
- Go 后端：使用 [gopter](https://github.com/leanovate/gopter) 库
- JavaScript 前端：使用 [fast-check](https://github.com/dubzzz/fast-check) 库

**配置要求**：
- 每个属性测试至少运行 100 次迭代
- 每个测试必须使用注释标记引用设计文档中的属性
- 标记格式：`// Feature: admin-task-deletion, Property N: [property text]`

### 后端测试计划

#### 1. Repository 层测试 (sqlite_repository_test.go)

**单元测试**：
- 测试删除存在的任务（示例）
- 测试删除不存在的任务（边界情况）
- 测试删除没有关联兑换码的任务（边界情况）
- 测试事务回滚场景（模拟失败）

**基于属性的测试**：
- **Property 6**: 对于任何有效任务 ID，删除后任务不应存在
- **Property 7**: 对于任何有关联兑换码的任务，删除后兑换码也应被删除
- **Property 8**: 对于任何删除操作，失败时数据应保持不变（原子性）

示例代码结构：
```go
func TestDeleteTask_Property6(t *testing.T) {
  // Feature: admin-task-deletion, Property 6: 删除操作移除任务记录
  properties := gopter.NewProperties(nil)
  
  properties.Property("deleted task should not exist", 
    prop.ForAll(
      func(taskID int64) bool {
        // 1. 创建任务
        // 2. 删除任务
        // 3. 验证任务不存在
        return true
      },
      gen.Int64Range(1, 1000000),
    ))
  
  properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

#### 2. API Handler 测试 (admin_handlers_test.go)

**单元测试**：
- 测试有效的删除请求
- 测试无效的任务 ID 格式
- 测试未授权请求（边界情况）
- 测试任务不存在的情况（边界情况）

**基于属性的测试**：
- **Property 9**: 对于任何未授权请求，应返回 401/403
- **Property 10**: 对于任何有效删除，应返回 200
- **Property 11**: 对于任何失败删除，应返回错误状态码和消息

### 前端测试计划

#### 1. UI 渲染测试 (dashboard.test.js)

**单元测试**：
- 测试空列表不显示删除按钮（边界情况）
- 测试确认对话框有"确认"和"取消"按钮（示例）

**基于属性的测试**：
- **Property 1**: 对于任何非空任务列表，每个任务都应有删除按钮
- **Property 2**: 对于任何任务，点击删除按钮应显示确认对话框
- **Property 3**: 对于任何任务，确认对话框应包含任务信息

示例代码结构：
```javascript
// Feature: admin-task-deletion, Property 1: 删除按钮渲染完整性
fc.assert(
  fc.property(
    fc.array(fc.record({
      id: fc.integer(),
      name: fc.string(),
      status: fc.constant('completed')
    }), {minLength: 1}),
    (tasks) => {
      const container = renderTaskList(tasks);
      const deleteButtons = container.querySelectorAll('.delete-btn');
      return deleteButtons.length === tasks.length;
    }
  ),
  { numRuns: 100 }
);
```

#### 2. 交互逻辑测试

**单元测试**：
- 测试取消删除不调用 API（示例）
- 测试网络错误显示错误消息（边界情况）

**基于属性的测试**：
- **Property 4**: 对于任何任务，取消操作不应改变数据
- **Property 5**: 对于任何任务，确认操作应调用删除 API
- **Property 12**: 对于任何成功删除，应刷新列表
- **Property 13**: 对于任何已删除任务，刷新后列表不应包含它
- **Property 14**: 对于任何失败删除，不应刷新列表
- **Property 15**: 对于任何失败删除，应显示错误消息

### 集成测试

**端到端测试场景**：
1. 完整的删除流程：点击按钮 → 确认 → API 调用 → 数据库删除 → UI 更新
2. 并发删除测试：多个删除操作同时进行
3. 事务完整性测试：模拟中途失败，验证回滚

### 测试数据生成

**Go 测试数据生成器**：
```go
// 生成随机任务
func genTask() gopter.Gen {
  return gen.Struct(reflect.TypeOf(&Task{}), map[string]gopter.Gen{
    "ID":     gen.Int64Range(1, 1000000),
    "Name":   gen.AlphaString(),
    "Status": gen.Const("completed"),
  })
}

// 生成随机兑换码列表
func genGiftCodes(taskID int64) gopter.Gen {
  return gen.SliceOf(gen.Struct(reflect.TypeOf(&GiftCode{}), map[string]gopter.Gen{
    "ID":     gen.Int64Range(1, 1000000),
    "TaskID": gen.Const(taskID),
    "Code":   gen.AlphaString(),
  }))
}
```

**JavaScript 测试数据生成器**：
```javascript
// 生成随机任务
const taskArbitrary = fc.record({
  id: fc.integer({min: 1, max: 1000000}),
  name: fc.string({minLength: 1, maxLength: 50}),
  status: fc.constant('completed'),
  createdAt: fc.date()
});

// 生成随机任务列表
const taskListArbitrary = fc.array(taskArbitrary, {minLength: 1, maxLength: 20});
```

### 测试覆盖率目标

- 代码覆盖率：≥ 80%
- 分支覆盖率：≥ 75%
- 所有正确性属性必须有对应的属性测试
- 所有边界情况必须有对应的单元测试

### 持续集成

- 所有测试在 CI/CD 管道中自动运行
- 属性测试失败时，记录失败的输入用于回归测试
- 定期增加属性测试的迭代次数以发现罕见错误
