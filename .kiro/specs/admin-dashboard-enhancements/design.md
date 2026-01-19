# Design Document: Admin Dashboard Enhancements

## Overview

本设计文档描述了管理后台增强功能的技术实现方案。该功能包括四个主要方面：

1. **简化用户管理界面** - 移除不必要的输入字段，只保留 FID 输入
2. **任务监控增强** - 添加时间戳和重试计数字段以提供更详细的任务追踪
3. **历史任务查看** - 实现已完成任务的查询和展示功能
4. **通知系统重构** - 抽象通知接口，实现通知历史记录和展示

该设计遵循现有的代码架构模式，使用 Go 语言实现后端，原生 JavaScript 实现前端，SQLite 作为数据存储。

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Admin Dashboard UI                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ User Mgmt    │  │ Task Monitor │  │ Notification │      │
│  │ (Simplified) │  │ (Enhanced)   │  │ History      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ HTTP/JSON API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     API Handlers Layer                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Admin        │  │ Task         │  │ Notification │      │
│  │ Handlers     │  │ Handlers     │  │ Handlers     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Task         │  │ Notification │  │ Job          │      │
│  │ Service      │  │ Service      │  │ Processor    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              SQLite Repository                       │   │
│  │  - User CRUD                                         │   │
│  │  - Task CRUD (with new fields)                       │   │
│  │  - Notification CRUD                                 │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      SQLite Database                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ fid_list     │  │ gift_code_   │  │ notifications│      │
│  │              │  │ task         │  │              │      │
│  │              │  │ (enhanced)   │  │ (new)        │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

**User Management Flow:**
```
User Input (FID only) → Validation → Repository.SaveUser → Database
```

**Task Monitoring Flow:**
```
Job Processor → Update Task (timestamps, retry_count) → Repository → Database
Admin UI → API → Repository.ListTasks/ListCompletedTasks → Display
```

**Notification Flow:**
```
Task Complete → Notifier.Send → Save to DB → Display in History
```

## Components and Interfaces

### 1. Database Schema Changes

#### Migration: Add Task Tracking Fields

```sql
-- Migration: 000002_add_task_tracking_fields.up.sql
ALTER TABLE gift_code_task ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE gift_code_task ADD COLUMN completed_at TIMESTAMP NULL;
ALTER TABLE gift_code_task ADD COLUMN retry_count INTEGER DEFAULT 0;
ALTER TABLE gift_code_task ADD COLUMN last_error TEXT DEFAULT '';

-- Create index for completed tasks query
CREATE INDEX IF NOT EXISTS idx_task_completed ON gift_code_task(all_done, completed_at DESC);
```

```sql
-- Migration: 000002_add_task_tracking_fields.down.sql
DROP INDEX IF EXISTS idx_task_completed;
ALTER TABLE gift_code_task DROP COLUMN last_error;
ALTER TABLE gift_code_task DROP COLUMN retry_count;
ALTER TABLE gift_code_task DROP COLUMN completed_at;
ALTER TABLE gift_code_task DROP COLUMN created_at;
```

#### Migration: Create Notifications Table

```sql
-- Migration: 000003_create_notifications_table.up.sql
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    result TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('success', 'failed')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for notification history query
CREATE INDEX IF NOT EXISTS idx_notification_created ON notifications(created_at DESC);
```

```sql
-- Migration: 000003_create_notifications_table.down.sql
DROP INDEX IF EXISTS idx_notification_created;
DROP TABLE IF EXISTS notifications;
```

### 2. Repository Interface Extensions

```go
// Repository interface additions
type Repository interface {
    // ... existing methods ...
    
    // Task operations with enhanced fields
    UpdateTaskRetry(ctx context.Context, code string, retryCount int, lastError string) error
    UpdateTaskComplete(ctx context.Context, code string, completedAt time.Time) error
    ListCompletedTasks(ctx context.Context, limit int) ([]*Task, error)
    
    // Notification operations
    SaveNotification(ctx context.Context, notification *Notification) error
    ListNotifications(ctx context.Context, limit int) ([]*Notification, error)
}

// Notification model
type Notification struct {
    ID        int64     `json:"id"`
    Channel   string    `json:"channel"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Result    string    `json:"result"`
    Status    string    `json:"status"` // "success" or "failed"
    CreatedAt time.Time `json:"created_at"`
}

// NotificationStatus constants
const (
    NotificationStatusSuccess = "success"
    NotificationStatusFailed  = "failed"
)
```

### 3. Notifier Interface

```go
// Package: internal/notification

// Notifier defines the interface for sending notifications
type Notifier interface {
    // Send sends a notification and returns the result
    Send(ctx context.Context, req NotificationRequest) (*NotificationResult, error)
    
    // GetChannel returns the channel name for this notifier
    GetChannel() string
}

// NotificationRequest contains notification parameters
type NotificationRequest struct {
    Title   string
    Summary string
    Content string
}

// NotificationResult contains the notification send result
type NotificationResult struct {
    Success bool
    Message string
    Error   error
}

// WxPusherNotifier implements Notifier for WxPusher
type WxPusherNotifier struct {
    appToken string
    uid      string
    client   *http.Client
    logger   *logrus.Logger
}

// NewWxPusherNotifier creates a new WxPusher notifier
func NewWxPusherNotifier(appToken, uid string, logger *logrus.Logger) *WxPusherNotifier {
    return &WxPusherNotifier{
        appToken: appToken,
        uid:      uid,
        client:   &http.Client{Timeout: 30 * time.Second},
        logger:   logger,
    }
}

// Send implements Notifier.Send
func (w *WxPusherNotifier) Send(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
    // Implementation details...
}

// GetChannel implements Notifier.GetChannel
func (w *WxPusherNotifier) GetChannel() string {
    return "wxpusher"
}
```

### 4. Notification Service

```go
// Package: internal/service

// NotificationService handles notification sending and persistence
type NotificationService struct {
    notifier   notification.Notifier
    repository storage.Repository
    logger     *logrus.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
    notifier notification.Notifier,
    repository storage.Repository,
    logger *logrus.Logger,
) *NotificationService {
    return &NotificationService{
        notifier:   notifier,
        repository: repository,
        logger:     logger,
    }
}

// SendAndSave sends a notification and saves the result to database
func (s *NotificationService) SendAndSave(ctx context.Context, title, summary, content string) error {
    // Send notification
    req := notification.NotificationRequest{
        Title:   title,
        Summary: summary,
        Content: content,
    }
    
    result, err := s.notifier.Send(ctx, req)
    
    // Prepare notification record
    notif := &storage.Notification{
        Channel:   s.notifier.GetChannel(),
        Title:     title,
        Content:   content,
        CreatedAt: time.Now(),
    }
    
    if err != nil || !result.Success {
        notif.Status = storage.NotificationStatusFailed
        notif.Result = fmt.Sprintf("Error: %v", err)
        if result != nil && result.Message != "" {
            notif.Result = result.Message
        }
    } else {
        notif.Status = storage.NotificationStatusSuccess
        notif.Result = result.Message
    }
    
    // Save to database
    if saveErr := s.repository.SaveNotification(ctx, notif); saveErr != nil {
        s.logger.WithError(saveErr).Error("failed to save notification record")
        return saveErr
    }
    
    return err
}
```

### 5. API Handler Extensions

```go
// AdminHandlers additions

// ListCompletedTasks handles GET /api/admin/tasks/completed
func (h *AdminHandlers) ListCompletedTasks(c *gin.Context) {
    requestID, _ := c.Get("request_id")
    ctx := c.Request.Context()
    
    // Get limit from query parameter (default 100)
    limit := 100
    if limitStr := c.Query("limit"); limitStr != "" {
        if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
            limit = parsedLimit
        }
    }
    
    tasks, err := h.repository.ListCompletedTasks(ctx, limit)
    if err != nil {
        h.logger.WithFields(logrus.Fields{
            "request_id": requestID,
            "error":      err.Error(),
        }).Error("failed to fetch completed tasks")
        
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch completed tasks"))
        return
    }
    
    if tasks == nil {
        tasks = []*storage.Task{}
    }
    
    h.logger.WithFields(logrus.Fields{
        "request_id": requestID,
        "count":      len(tasks),
    }).Info("completed tasks fetched successfully")
    
    c.JSON(200, SuccessResponse(gin.H{"tasks": tasks}))
}

// ListNotifications handles GET /api/admin/notifications
func (h *AdminHandlers) ListNotifications(c *gin.Context) {
    requestID, _ := c.Get("request_id")
    ctx := c.Request.Context()
    
    // Get limit from query parameter (default 100)
    limit := 100
    if limitStr := c.Query("limit"); limitStr != "" {
        if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
            limit = parsedLimit
        }
    }
    
    notifications, err := h.repository.ListNotifications(ctx, limit)
    if err != nil {
        h.logger.WithFields(logrus.Fields{
            "request_id": requestID,
            "error":      err.Error(),
        }).Error("failed to fetch notifications")
        
        c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch notifications"))
        return
    }
    
    if notifications == nil {
        notifications = []*storage.Notification{}
    }
    
    h.logger.WithFields(logrus.Fields{
        "request_id": requestID,
        "count":      len(notifications),
    }).Info("notifications fetched successfully")
    
    c.JSON(200, SuccessResponse(gin.H{"notifications": notifications}))
}
```

### 6. Job Processor Updates

```go
// GetCodeJob modifications

// processCodeSafely updates to track retry count and errors
func (g *GetCodeJob) processCodeSafely(code string, fids []string) {
    defer func() {
        if err := recover(); err != nil {
            logrus.Errorf("处理code %s 时发生panic: %v", code, err)
            logrus.Errorf("堆栈信息: %s", debug.Stack())
            
            // Update retry count and error
            ctx := context.Background()
            task, _ := g.svcCtx.Repository.GetTaskByCode(ctx, code)
            if task != nil {
                retryCount := task.RetryCount + 1
                errorMsg := fmt.Sprintf("Panic: %v", err)
                _ = g.svcCtx.Repository.UpdateTaskRetry(ctx, code, retryCount, errorMsg)
            }
        }
    }()
    
    logrus.Infof("开始执行code: %s任务, 处理人: %v", code, fids)
    startTime := time.Now()
    
    alldone, msg, err := g.once(code, fids)
    
    ctx := context.Background()
    
    if err != nil {
        logrus.Errorf("GetCodeJob GetTask err: %v", err)
        
        // Update retry count and error
        task, _ := g.svcCtx.Repository.GetTaskByCode(ctx, code)
        if task != nil {
            retryCount := task.RetryCount + 1
            _ = g.svcCtx.Repository.UpdateTaskRetry(ctx, code, retryCount, err.Error())
        }
    } else if alldone {
        // Mark task as complete with timestamp
        completedAt := time.Now()
        if err := g.svcCtx.Repository.UpdateTaskComplete(ctx, code, completedAt); err != nil {
            logrus.Errorf("GetCodeJob UpdateTaskComplete err: %v", err)
        } else {
            // Send notification using NotificationService
            if g.svcCtx.NotificationService != nil {
                title := "兑换码兑换成功"
                summary := fmt.Sprintf("兑换码[%s]兑换成功", code)
                _ = g.svcCtx.NotificationService.SendAndSave(ctx, title, summary, msg)
            }
        }
    }
    
    logrus.Infof("任务执行信息: %s", msg)
    endTime := time.Now()
    logrus.Infof("完成code: %s任务, 耗时: %s", code, endTime.Sub(startTime).String())
}
```

### 7. Frontend Updates

#### User Management Interface (Simplified)

```javascript
// dashboard.js modifications

// Simplified add user form
function loadUsersView() {
    // ... existing code ...
    
    let html = `
        <div style="margin-bottom: 2rem;">
            <h3 style="margin-bottom: 1rem;">添加用户</h3>
            <form id="add-user-form" onsubmit="addUser(event)">
                <div class="form-group">
                    <label>用户ID (FID) *</label>
                    <input type="text" name="fid" required placeholder="请输入用户FID">
                </div>
                <button type="submit" class="btn">添加用户</button>
            </form>
        </div>
        
        <h3 style="margin-bottom: 1rem;">用户列表 (${users.length})</h3>
    `;
    
    // ... rest of the code ...
}

// Simplified addUser function
async function addUser(event) {
    event.preventDefault();
    
    const form = event.target;
    
    if (!validateForm(form)) {
        showMessage('users', '请填写所有必填字段', 'error');
        return;
    }
    
    const formData = new FormData(form);
    
    const userData = {
        fid: formData.get('fid').trim(),
        nickname: '',  // Empty default
        kid: 0,        // Zero default
        avatar_image: '' // Empty default
    };
    
    showLoading();
    
    try {
        await apiRequest('/users', {
            method: 'POST',
            body: JSON.stringify(userData)
        });
        
        showMessage('users', '用户添加成功', 'success');
        form.reset();
        loadUsersView();
    } catch (error) {
        showMessage('users', `添加失败: ${error.message}`, 'error');
    } finally {
        hideLoading();
    }
}
```

#### Task Monitor Enhancements

```javascript
// dashboard.js modifications

// Enhanced task list display
function loadTasksView() {
    // ... existing code ...
    
    html += `
        <div style="margin-bottom: 1rem; display: flex; gap: 1rem;">
            <button class="btn" onclick="loadTasksView()">当前任务</button>
            <button class="btn btn-secondary" onclick="loadCompletedTasksView()">历史任务</button>
        </div>
        
        <h3 style="margin-bottom: 1rem;">当前任务列表 (${tasks.length})</h3>
    `;
    
    if (tasks.length === 0) {
        html += '<div class="empty-state">暂无任务</div>';
    } else {
        html += `
            <div class="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>兑换码</th>
                            <th>状态</th>
                            <th>重试次数</th>
                            <th>错误信息</th>
                            <th>创建时间</th>
                            <th>完成时间</th>
                        </tr>
                    </thead>
                    <tbody>
        `;
        
        tasks.forEach(task => {
            const createdAt = task.created_at ? 
                new Date(task.created_at).toLocaleString('zh-CN') : '-';
            const completedAt = task.completed_at ? 
                new Date(task.completed_at).toLocaleString('zh-CN') : '-';
            
            let status = 'pending';
            let statusText = '待处理';
            
            if (task.all_done) {
                status = 'completed';
                statusText = '已完成';
            } else if (task.retry_count > 0) {
                status = 'failed';
                statusText = '失败';
            }
            
            const error = task.last_error || '-';
            
            html += `
                <tr>
                    <td>${task.code || '-'}</td>
                    <td><span class="status-badge status-${status}">${statusText}</span></td>
                    <td>${task.retry_count || 0}</td>
                    <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis;" 
                        title="${error}">${error}</td>
                    <td>${createdAt}</td>
                    <td>${completedAt}</td>
                </tr>
            `;
        });
        
        html += `
                    </tbody>
                </table>
            </div>
        `;
    }
    
    contentEl.innerHTML = html;
}

// Load completed tasks view
async function loadCompletedTasksView() {
    const contentEl = document.getElementById('tasks-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';
    
    try {
        const response = await apiRequest('/tasks/completed?limit=100');
        const tasks = response.data.tasks || [];
        
        let html = `
            <div style="margin-bottom: 1rem; display: flex; gap: 1rem;">
                <button class="btn btn-secondary" onclick="loadTasksView()">当前任务</button>
                <button class="btn" onclick="loadCompletedTasksView()">历史任务</button>
            </div>
            
            <h3 style="margin-bottom: 1rem;">历史任务列表 (${tasks.length})</h3>
        `;
        
        if (tasks.length === 0) {
            html += '<div class="empty-state">暂无历史任务</div>';
        } else {
            html += `
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>兑换码</th>
                                <th>状态</th>
                                <th>重试次数</th>
                                <th>错误信息</th>
                                <th>创建时间</th>
                                <th>完成时间</th>
                            </tr>
                        </thead>
                        <tbody>
            `;
            
            tasks.forEach(task => {
                const createdAt = task.created_at ? 
                    new Date(task.created_at).toLocaleString('zh-CN') : '-';
                const completedAt = task.completed_at ? 
                    new Date(task.completed_at).toLocaleString('zh-CN') : '-';
                
                const error = task.last_error || '-';
                
                html += `
                    <tr>
                        <td>${task.code || '-'}</td>
                        <td><span class="status-badge status-completed">已完成</span></td>
                        <td>${task.retry_count || 0}</td>
                        <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis;" 
                            title="${error}">${error}</td>
                        <td>${createdAt}</td>
                        <td>${completedAt}</td>
                    </tr>
                `;
            });
            
            html += `
                        </tbody>
                    </table>
                </div>
            `;
        }
        
        contentEl.innerHTML = html;
    } catch (error) {
        contentEl.innerHTML = '<div class="empty-state">加载失败，请重试</div>';
        showMessage('tasks', error.message, 'error');
    }
}
```

#### Notification History View

```javascript
// dashboard.js additions

// Add navigation item
function setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const view = item.dataset.view;
            showView(view);
        });
    });
}

// Update showView function
function showView(viewName) {
    // ... existing code ...
    
    if (viewName === 'notifications') {
        document.getElementById('notifications-view').classList.add('active');
        loadNotificationsView();
    }
}

// Load notifications view
async function loadNotificationsView() {
    const contentEl = document.getElementById('notifications-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';
    
    try {
        const response = await apiRequest('/notifications?limit=100');
        const notifications = response.data.notifications || [];
        
        let html = `<h3 style="margin-bottom: 1rem;">通知历史 (${notifications.length})</h3>`;
        
        if (notifications.length === 0) {
            html += '<div class="empty-state">暂无通知记录</div>';
        } else {
            html += `
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>渠道</th>
                                <th>标题</th>
                                <th>内容</th>
                                <th>时间</th>
                                <th>状态</th>
                                <th>结果</th>
                            </tr>
                        </thead>
                        <tbody>
            `;
            
            notifications.forEach(notif => {
                const createdAt = notif.created_at ? 
                    new Date(notif.created_at).toLocaleString('zh-CN') : '-';
                
                const status = notif.status === 'success' ? 'completed' : 'failed';
                const statusText = notif.status === 'success' ? '成功' : '失败';
                
                // Truncate long content
                const content = notif.content.length > 50 ? 
                    notif.content.substring(0, 50) + '...' : notif.content;
                const result = notif.result.length > 50 ? 
                    notif.result.substring(0, 50) + '...' : notif.result;
                
                html += `
                    <tr>
                        <td>${notif.channel || '-'}</td>
                        <td>${notif.title || '-'}</td>
                        <td title="${notif.content}">${content}</td>
                        <td>${createdAt}</td>
                        <td><span class="status-badge status-${status}">${statusText}</span></td>
                        <td title="${notif.result}">${result}</td>
                    </tr>
                `;
            });
            
            html += `
                        </tbody>
                    </table>
                </div>
            `;
        }
        
        contentEl.innerHTML = html;
    } catch (error) {
        contentEl.innerHTML = '<div class="empty-state">加载失败，请重试</div>';
        showMessage('notifications', error.message, 'error');
    }
}
```

```html
<!-- dashboard.html additions -->

<!-- Add to sidebar navigation -->
<aside class="sidebar">
    <ul class="nav-menu">
        <li class="nav-item active" data-view="users">用户管理</li>
        <li class="nav-item" data-view="tasks">任务监控</li>
        <li class="nav-item" data-view="notifications">通知历史</li>
    </ul>
</aside>

<!-- Add notifications view -->
<div id="notifications-view" class="view">
    <div class="view-header">
        <h2>通知历史</h2>
    </div>
    <div id="notifications-message" class="message"></div>
    <div id="notifications-content"></div>
</div>
```

## Data Models

### Enhanced Task Model

```go
type Task struct {
    Code        string     `json:"code"`
    AllDone     bool       `json:"all_done"`
    RetryCount  int        `json:"retry_count"`      // NEW
    LastError   string     `json:"last_error"`       // NEW
    CreatedAt   time.Time  `json:"created_at"`       // NEW
    UpdatedAt   time.Time  `json:"updated_at"`
    CompletedAt *time.Time `json:"completed_at"`     // NEW (nullable)
}
```

### Notification Model

```go
type Notification struct {
    ID        int64     `json:"id"`
    Channel   string    `json:"channel"`   // e.g., "wxpusher"
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Result    string    `json:"result"`    // Success message or error details
    Status    string    `json:"status"`    // "success" or "failed"
    CreatedAt time.Time `json:"created_at"`
}
```

### Simplified User Input

前端只需要发送：
```json
{
  "fid": "366184723"
}
```

后端自动填充默认值：
```json
{
  "fid": "366184723",
  "nickname": "",
  "kid": 0,
  "avatar_image": ""
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*


### Property 1: FID Validation Rejects Invalid Input

*For any* string input to the user creation form, if the string is empty or contains only whitespace characters, the validation should reject it and prevent user creation.

**Validates: Requirements 1.2**

### Property 2: User Creation Sets Default Values

*For any* valid FID input, when creating a new user, the system should set nickname to empty string, kid to 0, and avatar_image to empty string.

**Validates: Requirements 1.3**

### Property 3: Task Creation Sets Timestamp

*For any* new task created, the system should automatically set the created_at field to the current timestamp at the time of creation.

**Validates: Requirements 2.2**

### Property 4: Task Completion Sets Timestamp

*For any* task that is marked as complete, the system should set the completed_at field to the timestamp at the time of completion.

**Validates: Requirements 2.3**

### Property 5: Task Retry Increments Counter

*For any* task that fails and requires retry, the system should increment the retry_count field and record the error message in last_error.

**Validates: Requirements 2.4**

### Property 6: Completed Tasks Query Filter

*For any* query for completed tasks, the system should return only tasks where all_done is true, and should not return any tasks where all_done is false.

**Validates: Requirements 3.2**

### Property 7: Notifier Success Result

*For any* successful notification send operation, the Notifier should return a NotificationResult with Success set to true and a non-empty success message.

**Validates: Requirements 4.4**

### Property 8: Notifier Failure Result

*For any* failed notification send operation, the Notifier should return a NotificationResult with Success set to false and include error information.

**Validates: Requirements 4.5**

### Property 9: Notification Persistence

*For any* notification sent (whether successful or failed), the system should save a corresponding record to the notifications table in the database.

**Validates: Requirements 5.4**

### Property 10: Notification Record Completeness

*For any* notification record saved to the database, the record should contain all required fields: channel, title, content, result, status, and created_at, with status being either "success" or "failed" based on the send result.

**Validates: Requirements 5.5, 5.6, 5.7**

### Property 11: Notification Query Returns All Records

*For any* query for notification history, the system should return all notification records from the notifications table without filtering.

**Validates: Requirements 6.2**

### Property 12: Notification Query Ordering

*For any* query for notification history, the system should return records ordered by created_at in descending order (newest first).

**Validates: Requirements 6.7**

## Error Handling

### Input Validation Errors

**FID Validation:**
- Empty or whitespace-only FID → Return 400 Bad Request with error message "FID cannot be empty or whitespace only"
- Invalid FID format → Return 400 Bad Request with error message "Invalid FID format"

**Task Code Validation:**
- Empty or whitespace-only code → Return 400 Bad Request with error message "Gift code cannot be empty or whitespace only"

### Database Errors

**Connection Failures:**
- Database unavailable → Return 500 Internal Server Error with error message "Database connection failed"
- Use exponential backoff retry strategy (3 attempts with increasing delays)

**Query Failures:**
- Failed to execute query → Return 500 Internal Server Error with error message "Database query failed"
- Log detailed error information for debugging

**Transaction Failures:**
- Transaction rollback on error
- Log rollback errors separately
- Return 500 Internal Server Error with error message "Transaction failed"

### Notification Errors

**Send Failures:**
- Network timeout → Log error, save notification with status "failed" and error details
- Invalid credentials → Log error, save notification with status "failed" and error details
- API error response → Log error, save notification with status "failed" and error details

**Persistence Failures:**
- Failed to save notification record → Log error but do not fail the main operation
- Continue with task completion even if notification save fails

### Migration Errors

**Schema Changes:**
- Migration script failure → Rollback all changes
- Log detailed error information
- Exit with non-zero status code

**Data Migration:**
- Invalid existing data → Set to safe default values
- Log warnings for data that required default values

## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests for comprehensive coverage:

**Unit Tests** focus on:
- Specific examples of valid and invalid inputs
- Edge cases (empty strings, very long strings, special characters)
- Integration points between components
- Error conditions and error handling paths
- Database migration scripts

**Property-Based Tests** focus on:
- Universal properties that hold for all inputs
- Comprehensive input coverage through randomization
- Invariants that must be maintained across operations

### Property-Based Testing Configuration

**Library:** Use `gopter` for Go property-based testing

**Test Configuration:**
- Minimum 100 iterations per property test
- Each property test must reference its design document property
- Tag format: `// Feature: admin-dashboard-enhancements, Property N: [property text]`

**Example Property Test Structure:**

```go
func TestProperty1_FIDValidationRejectsInvalidInput(t *testing.T) {
    // Feature: admin-dashboard-enhancements, Property 1: FID Validation Rejects Invalid Input
    
    properties := gopter.NewProperties(nil)
    
    properties.Property("empty or whitespace FID should be rejected", prop.ForAll(
        func(whitespace string) bool {
            // Generate strings with only whitespace
            fid := strings.Repeat(" ", len(whitespace))
            
            // Validate
            err := validateFID(fid)
            
            // Should return error
            return err != nil
        },
        gen.Identifier(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Unit Test Coverage

**User Management:**
- Test adding user with only FID
- Test FID validation with various invalid inputs
- Test default value assignment
- Test duplicate FID handling

**Task Monitoring:**
- Test task creation with timestamp
- Test task completion with timestamp
- Test retry count increment
- Test error message recording
- Test pending tasks query
- Test completed tasks query
- Test task filtering logic

**Notification System:**
- Test WxPusher notifier send success
- Test WxPusher notifier send failure
- Test notification persistence
- Test notification query
- Test notification ordering
- Test notification record completeness

**Database Migrations:**
- Test migration up scripts
- Test migration down scripts
- Test default value assignment for existing records
- Test schema validation after migration

### Integration Tests

**End-to-End Flows:**
- Create user → Verify in database
- Create task → Process task → Verify completion → Verify notification
- Query completed tasks → Verify results
- Query notification history → Verify results

**API Integration:**
- Test all API endpoints with valid requests
- Test all API endpoints with invalid requests
- Test authentication and authorization
- Test error responses

### Test Data Generation

**For Property Tests:**
- Generate random FIDs (valid and invalid)
- Generate random task codes
- Generate random notification content
- Generate random timestamps
- Generate random error messages

**For Unit Tests:**
- Use fixed test data for reproducibility
- Include edge cases (empty, very long, special characters)
- Include boundary values

### Continuous Testing

**Automated Test Execution:**
- Run all tests on every commit
- Run property tests with increased iterations (1000+) in CI/CD
- Fail build on any test failure

**Test Metrics:**
- Track test coverage (target: >80% for new code)
- Track property test iteration counts
- Track test execution time
