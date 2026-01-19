// ============================================
// Utility Functions
// ============================================

/**
 * Show loading overlay
 */
function showLoading() {
    document.getElementById('loadingOverlay').classList.add('show');
}

/**
 * Hide loading overlay
 */
function hideLoading() {
    document.getElementById('loadingOverlay').classList.remove('show');
}

/**
 * Show confirmation dialog
 * @param {string} title - Dialog title
 * @param {string} message - Dialog message
 * @param {function} onConfirm - Callback when confirmed
 */
function showConfirmDialog(title, message, onConfirm) {
    const modal = document.getElementById('confirmModal');
    const titleEl = document.getElementById('confirmTitle');
    const messageEl = document.getElementById('confirmMessage');
    const confirmBtn = document.getElementById('confirmButton');
    
    titleEl.textContent = title;
    messageEl.textContent = message;
    modal.classList.add('show');
    
    // Remove previous event listeners
    const newConfirmBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);
    
    // Add new event listener
    newConfirmBtn.addEventListener('click', () => {
        closeConfirmModal();
        if (onConfirm) onConfirm();
    });
}

/**
 * Close confirmation dialog
 */
function closeConfirmModal() {
    document.getElementById('confirmModal').classList.remove('show');
}

/**
 * Validate form field
 * @param {HTMLInputElement} input - Input element
 * @returns {boolean} - Is valid
 */
function validateField(input) {
    const value = input.value.trim();
    const isRequired = input.hasAttribute('required');
    
    if (isRequired && !value) {
        input.classList.add('error');
        return false;
    }
    
    // Email validation
    if (input.type === 'email' && value) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(value)) {
            input.classList.add('error');
            return false;
        }
    }
    
    // Number validation
    if (input.type === 'number' && value) {
        if (isNaN(value)) {
            input.classList.add('error');
            return false;
        }
    }
    
    input.classList.remove('error');
    return true;
}

/**
 * Validate entire form
 * @param {HTMLFormElement} form - Form element
 * @returns {boolean} - Is valid
 */
function validateForm(form) {
    const inputs = form.querySelectorAll('input[required], input[type="email"], input[type="number"]');
    let isValid = true;
    
    inputs.forEach(input => {
        if (!validateField(input)) {
            isValid = false;
        }
    });
    
    return isValid;
}

// ============================================
// Authentication Interceptor
// ============================================

/**
 * Authentication interceptor that automatically:
 * - Adds Authorization header to all API requests
 * - Handles 401 errors (token expiration)
 * - Redirects to login page when authentication fails
 * - Clears expired tokens from LocalStorage
 */
const AuthInterceptor = {
    /**
     * Get the JWT token from LocalStorage
     */
    getToken() {
        return localStorage.getItem('admin_token');
    },

    /**
     * Check if token exists
     */
    hasToken() {
        return !!this.getToken();
    },

    /**
     * Clear authentication data from LocalStorage
     */
    clearAuth() {
        localStorage.removeItem('admin_token');
        localStorage.removeItem('admin_token_expires_at');
    },

    /**
     * Redirect to login page
     */
    redirectToLogin() {
        window.location.href = '/admin/login.html';
    },

    /**
     * Handle authentication failure
     * Clears tokens and redirects to login
     */
    handleAuthFailure() {
        console.warn('Authentication failed - redirecting to login');
        this.clearAuth();
        this.redirectToLogin();
    },

    /**
     * Intercept fetch requests to add authentication
     * @param {string} url - The URL to fetch
     * @param {object} options - Fetch options
     * @returns {Promise<Response>}
     */
    async fetch(url, options = {}) {
        // Get token
        const token = this.getToken();

        // Prepare headers with Authorization
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        // Add Authorization header if token exists
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        // Make the request
        try {
            const response = await fetch(url, {
                ...options,
                headers
            });

            // Handle 401 Unauthorized - token expired or invalid
            if (response.status === 401) {
                this.handleAuthFailure();
                // Return a rejected promise to prevent further processing
                return Promise.reject(new Error('UNAUTHORIZED'));
            }

            // Handle 400 Bad Request - malformed token
            if (response.status === 400) {
                const data = await response.json();
                // Check if it's an auth-related error
                if (data.error?.code === 'VALIDATION_ERROR' && 
                    data.error?.message?.includes('authorization')) {
                    this.handleAuthFailure();
                    return Promise.reject(new Error('INVALID_TOKEN_FORMAT'));
                }
            }

            return response;
        } catch (error) {
            // Network errors or other fetch failures
            if (error.message === 'UNAUTHORIZED' || error.message === 'INVALID_TOKEN_FORMAT') {
                throw error;
            }
            console.error('Network error:', error);
            throw error;
        }
    }
};

// Check authentication on page load
if (!AuthInterceptor.hasToken()) {
    AuthInterceptor.redirectToLogin();
}

// API base URL
const API_BASE = '/api/admin';

// Current view state
let currentView = 'users';
let autoRefreshInterval = null;
let refreshInterval = 30000; // Default 30 seconds

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadRefreshInterval();
    setupNavigation();
    setupFormValidation();
    setupModalCloseOnOutsideClick();
    loadUsersView();
});

// Setup navigation
function setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const view = item.dataset.view;
            showView(view);
        });
    });
}

// Setup form validation
function setupFormValidation() {
    // Add real-time validation to all forms
    document.addEventListener('input', (e) => {
        if (e.target.tagName === 'INPUT') {
            validateField(e.target);
        }
    });
}

// Setup modal close on outside click
function setupModalCloseOnOutsideClick() {
    document.getElementById('confirmModal').addEventListener('click', (e) => {
        if (e.target.id === 'confirmModal') {
            closeConfirmModal();
        }
    });
}

// Load refresh interval from localStorage
function loadRefreshInterval() {
    const savedInterval = localStorage.getItem('refreshInterval');
    if (savedInterval) {
        refreshInterval = parseInt(savedInterval, 10);
    }
}

// Change refresh interval
function changeRefreshInterval(interval) {
    refreshInterval = parseInt(interval, 10);
    localStorage.setItem('refreshInterval', refreshInterval);

    // Restart auto-refresh with new interval
    if (currentView === 'tasks' && autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = setInterval(() => {
            if (currentView === 'tasks') {
                loadTasksView();
            }
        }, refreshInterval);
    }
}

// Show view
function showView(viewName) {
    // Clear auto-refresh
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }

    // Update navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.view === viewName) {
            item.classList.add('active');
        }
    });

    // Update views
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
    });

    currentView = viewName;

    // Load view content
    if (viewName === 'users') {
        document.getElementById('users-view').classList.add('active');
        loadUsersView();
    } else if (viewName === 'tasks') {
        document.getElementById('tasks-view').classList.add('active');
        loadTasksView();
    } else if (viewName === 'notifications') {
        document.getElementById('notifications-view').classList.add('active');
        loadNotificationsView();
    }
}

// Logout
function logout() {
    showConfirmDialog(
        '确认退出',
        '您确定要退出登录吗？',
        () => {
            AuthInterceptor.clearAuth();
            AuthInterceptor.redirectToLogin();
        }
    );
}

// API request helper using AuthInterceptor
async function apiRequest(endpoint, options = {}) {
    try {
        const response = await AuthInterceptor.fetch(`${API_BASE}${endpoint}`, options);

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error?.message || 'Request failed');
        }

        return data;
    } catch (error) {
        // If it's an auth error, it's already handled by the interceptor
        if (error.message === 'UNAUTHORIZED' || error.message === 'INVALID_TOKEN_FORMAT') {
            return null;
        }
        console.error('API request failed:', error);
        throw error;
    }
}

// Show message
function showMessage(viewId, message, type = 'success') {
    const messageEl = document.getElementById(`${viewId}-message`);
    messageEl.textContent = message;
    messageEl.className = `message ${type} show`;
    
    setTimeout(() => {
        messageEl.classList.remove('show');
    }, 5000);
}

// Load users view
async function loadUsersView() {
    const contentEl = document.getElementById('users-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';

    try {
        const response = await apiRequest('/users');
        const users = response.data.users || [];

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

        if (users.length === 0) {
            html += '<div class="empty-state">暂无用户</div>';
        } else {
            html += `
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>头像</th>
                                <th>用户ID (FID)</th>
                                <th>昵称</th>
                                <th>KID</th>
                                <th>创建时间</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody>
            `;

            users.forEach(user => {
                const createdAt = user.created_at ? new Date(user.created_at).toLocaleString('zh-CN') : '-';
                const avatar = user.avatar_image || 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="40" height="40"%3E%3Crect fill="%23ddd" width="40" height="40"/%3E%3C/svg%3E';
                
                html += `
                    <tr>
                        <td><img src="${avatar}" alt="avatar" class="avatar" onerror="this.src='data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%2240%22 height=%2240%22%3E%3Crect fill=%22%23ddd%22 width=%2240%22 height=%2240%22/%3E%3C/svg%3E'"></td>
                        <td>${user.fid || '-'}</td>
                        <td>${user.nickname || '-'}</td>
                        <td>${user.kid || '-'}</td>
                        <td>${createdAt}</td>
                        <td><span class="clickable" onclick="showUserDetails('${user.fid}')">查看兑换记录</span></td>
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
        showMessage('users', error.message, 'error');
    }
}

// Add user
async function addUser(event) {
    event.preventDefault();
    
    const form = event.target;
    
    // Validate form
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

    // Show loading
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

// Load tasks view
async function loadTasksView() {
    const contentEl = document.getElementById('tasks-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';

    try {
        const response = await apiRequest('/tasks');
        const tasks = response.data.tasks || [];

        let html = `
            <div style="margin-bottom: 2rem;">
                <h3 style="margin-bottom: 1rem;">添加兑换码</h3>
                <form id="add-giftcode-form" onsubmit="addGiftCode(event)">
                    <div class="form-group">
                        <label>兑换码 *</label>
                        <input type="text" name="code" required placeholder="请输入兑换码">
                    </div>
                    <button type="submit" class="btn">添加兑换码</button>
                </form>
            </div>

            <div style="margin-bottom: 1rem; display: flex; gap: 1rem; align-items: center; flex-wrap: wrap;">
                <h3 style="margin: 0;">任务列表 (${tasks.length})</h3>
                <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <label for="refresh-interval" style="font-size: 0.9rem; color: #6c757d;">自动刷新:</label>
                    <select id="refresh-interval" onchange="changeRefreshInterval(this.value)" style="padding: 0.25rem 0.5rem; border: 1px solid #ddd; border-radius: 4px; font-size: 0.9rem;">
                        <option value="1000" ${refreshInterval === 1000 ? 'selected' : ''}>1秒</option>
                        <option value="2000" ${refreshInterval === 2000 ? 'selected' : ''}>2秒</option>
                        <option value="5000" ${refreshInterval === 5000 ? 'selected' : ''}>5秒</option>
                        <option value="10000" ${refreshInterval === 10000 ? 'selected' : ''}>10秒</option>
                        <option value="30000" ${refreshInterval === 30000 ? 'selected' : ''}>30秒</option>
                    </select>
                </div>
                <button class="btn btn-secondary" onclick="loadCompletedTasksView()">历史任务</button>
            </div>
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
                const createdAt = task.created_at ? new Date(task.created_at).toLocaleString('zh-CN') : '-';
                const completedAt = task.completed_at ? new Date(task.completed_at).toLocaleString('zh-CN') : '-';
                
                let status = 'pending';
                let statusText = '待处理';
                
                if (task.all_done) {
                    status = 'completed';
                    statusText = '已完成';
                } else if (task.retry_count > 0) {
                    status = 'failed';
                    statusText = '失败';
                } else if (task.retry_count === 0 && !task.all_done) {
                    status = 'processing';
                    statusText = '处理中';
                }
                
                const error = task.last_error || '-';
                
                html += `
                    <tr>
                        <td>${task.code || '-'}</td>
                        <td><span class="status-badge status-${status}">${statusText}</span></td>
                        <td>${task.retry_count || 0}</td>
                        <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${error}">${error}</td>
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

        // Setup auto-refresh with selected interval
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
        }
        autoRefreshInterval = setInterval(() => {
            if (currentView === 'tasks') {
                loadTasksView();
            }
        }, refreshInterval);

    } catch (error) {
        contentEl.innerHTML = '<div class="empty-state">加载失败，请重试</div>';
        showMessage('tasks', error.message, 'error');
    }
}

/**
 * Render delete button for a task
 * @param {object} task - Task object
 * @returns {string} - HTML string for delete button
 */
function renderTaskDeleteButton(task) {
    // Use task.code as the identifier since the API endpoint uses :code parameter
    const taskCode = task.code || task.id || '';
    const taskName = task.code || '未命名任务';
    
    return `<button class="btn btn-danger btn-sm" 
                    data-task-id="${taskCode}" 
                    data-task-name="${taskName}"
                    style="padding: 0.25rem 0.5rem; font-size: 0.85rem;">
                🗑️ 删除
            </button>`;
}

/**
 * Show delete confirmation dialog
 * @param {number} taskId - Task ID
 * @param {string} taskName - Task name
 * @returns {Promise<boolean>} - Resolves to true if confirmed, false if cancelled
 */
async function showDeleteConfirmation(taskId, taskName) {
    return new Promise((resolve) => {
        // Create modal overlay
        const overlay = document.createElement('div');
        overlay.className = 'modal-overlay delete-confirm-modal show';
        overlay.id = 'deleteConfirmModal';
        
        // Create modal content
        const modal = document.createElement('div');
        modal.className = 'modal';
        
        // Modal header
        const header = document.createElement('div');
        header.className = 'modal-header';
        const title = document.createElement('h3');
        title.textContent = '确认删除任务';
        header.appendChild(title);
        
        // Modal body
        const body = document.createElement('div');
        body.className = 'modal-body';
        const message = document.createElement('p');
        message.innerHTML = `您确定要删除以下任务吗？<br><br><strong>任务 ID:</strong> ${taskId}<br><strong>兑换码:</strong> ${taskName}<br><br>此操作将删除任务及其所有关联的兑换码记录，且无法撤销。`;
        body.appendChild(message);
        
        // Modal footer
        const footer = document.createElement('div');
        footer.className = 'modal-footer';
        
        // Cancel button
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'btn btn-secondary';
        cancelBtn.textContent = '取消';
        cancelBtn.onclick = () => {
            document.body.removeChild(overlay);
            resolve(false);
        };
        
        // Confirm button
        const confirmBtn = document.createElement('button');
        confirmBtn.className = 'btn btn-danger';
        confirmBtn.textContent = '确认删除';
        confirmBtn.onclick = () => {
            document.body.removeChild(overlay);
            resolve(true);
        };
        
        footer.appendChild(cancelBtn);
        footer.appendChild(confirmBtn);
        
        // Assemble modal
        modal.appendChild(header);
        modal.appendChild(body);
        modal.appendChild(footer);
        overlay.appendChild(modal);
        
        // Close on overlay click
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                document.body.removeChild(overlay);
                resolve(false);
            }
        });
        
        // Add to document
        document.body.appendChild(overlay);
    });
}

/**
 * Delete a task via API
 * @param {number} taskId - Task ID
 * @returns {Promise<{success: boolean, error?: string, statusCode?: number}>}
 */
async function deleteTask(taskId) {
    try {
        const response = await AuthInterceptor.fetch(`${API_BASE}/tasks/${taskId}`, {
            method: 'DELETE'
        });

        const data = await response.json();

        if (!response.ok) {
            // Log detailed error information to console
            console.error('Delete task failed:', {
                taskId: taskId,
                statusCode: response.status,
                statusText: response.statusText,
                error: data.error,
                timestamp: new Date().toISOString()
            });

            // Return error with status code for specific handling
            let errorMessage = data.error || '删除失败';
            
            // Provide user-friendly error messages based on status code
            switch (response.status) {
                case 400:
                    errorMessage = '无效的任务ID格式';
                    break;
                case 401:
                case 403:
                    errorMessage = '权限不足，请重新登录';
                    break;
                case 404:
                    errorMessage = '任务不存在或已被删除';
                    break;
                case 500:
                    errorMessage = '服务器错误，请稍后重试';
                    break;
                default:
                    errorMessage = data.error || `删除失败 (错误代码: ${response.status})`;
            }

            return {
                success: false,
                error: errorMessage,
                statusCode: response.status
            };
        }

        // Log successful deletion
        console.log('Delete task successful:', {
            taskId: taskId,
            timestamp: new Date().toISOString()
        });

        return {
            success: true
        };
    } catch (error) {
        // Handle authentication errors
        if (error.message === 'UNAUTHORIZED' || error.message === 'INVALID_TOKEN_FORMAT') {
            console.error('Authentication error during delete:', {
                taskId: taskId,
                error: error.message,
                timestamp: new Date().toISOString()
            });
            
            return {
                success: false,
                error: '认证失败，请重新登录',
                statusCode: 401
            };
        }
        
        // Handle network errors
        console.error('Network error during delete task:', {
            taskId: taskId,
            error: error.message,
            stack: error.stack,
            timestamp: new Date().toISOString()
        });
        
        return {
            success: false,
            error: '网络连接失败，请检查网络后重试',
            statusCode: 0 // 0 indicates network error
        };
    }
}

/**
 * Show error message
 * @param {string} message - Error message to display
 */
function showErrorMessage(message) {
    // Use the existing showMessage function with error type
    showMessage('tasks', message, 'error');
}

/**
 * Refresh completed tasks list
 */
function refreshCompletedTasks() {
    loadCompletedTasksView();
}

/**
 * Handle delete button click
 * @param {number} taskId - Task ID
 * @param {string} taskName - Task name
 */
async function handleDeleteClick(taskId, taskName) {
    // Show confirmation dialog
    const confirmed = await showDeleteConfirmation(taskId, taskName);
    
    // If user cancelled, return without doing anything
    if (!confirmed) {
        console.log('Delete operation cancelled by user:', { taskId, taskName });
        return;
    }
    
    // Find the delete button and disable it during deletion
    const deleteButton = document.querySelector(`[data-task-id="${taskId}"]`);
    if (deleteButton) {
        deleteButton.disabled = true;
        deleteButton.textContent = '删除中...';
    }
    
    // Show loading overlay
    showLoading();
    
    try {
        // Call deleteTask API
        const result = await deleteTask(taskId);
        
        if (result.success) {
            // Show success message
            showMessage('tasks', '任务删除成功', 'success');
            
            // Refresh the completed tasks list
            refreshCompletedTasks();
        } else {
            // Log error details to console
            console.error('Delete task failed:', {
                taskId: taskId,
                taskName: taskName,
                error: result.error,
                statusCode: result.statusCode,
                timestamp: new Date().toISOString()
            });
            
            // Show error message with specific details
            showErrorMessage(result.error || '删除任务失败');
            
            // Re-enable the button
            if (deleteButton) {
                deleteButton.disabled = false;
                deleteButton.textContent = '🗑️ 删除';
            }
        }
    } catch (error) {
        // Log unexpected errors
        console.error('Unexpected error during delete operation:', {
            taskId: taskId,
            taskName: taskName,
            error: error.message,
            stack: error.stack,
            timestamp: new Date().toISOString()
        });
        
        // Show error message
        showErrorMessage('删除任务时发生错误');
        
        // Re-enable the button
        if (deleteButton) {
            deleteButton.disabled = false;
            deleteButton.textContent = '🗑️ 删除';
        }
    } finally {
        hideLoading();
    }
}

// Load completed tasks view
async function loadCompletedTasksView() {
    const contentEl = document.getElementById('tasks-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';

    // Clear auto-refresh when viewing completed tasks
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }

    try {
        const response = await apiRequest('/tasks/completed?limit=100');
        const tasks = response.data.tasks || [];

        let html = `
            <div style="margin-bottom: 1rem; display: flex; gap: 1rem; align-items: center;">
                <h3 style="margin: 0;">历史任务列表 (${tasks.length})</h3>
                <button class="btn btn-secondary" onclick="loadTasksView()">返回当前任务</button>
            </div>
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
                                <th>操作</th>
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
                        <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${error}">${error}</td>
                        <td>${createdAt}</td>
                        <td>${completedAt}</td>
                        <td>${renderTaskDeleteButton(task)}</td>
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
        
        // Bind delete button click events using event delegation
        contentEl.querySelectorAll('.btn-danger').forEach(button => {
            button.addEventListener('click', (e) => {
                const taskId = e.target.dataset.taskId;
                const taskName = e.target.dataset.taskName;
                handleDeleteClick(taskId, taskName);
            });
        });
    } catch (error) {
        contentEl.innerHTML = '<div class="empty-state">加载失败，请重试</div>';
        showMessage('tasks', error.message, 'error');
    }
}

// Add gift code
async function addGiftCode(event) {
    event.preventDefault();
    
    const form = event.target;
    
    // Validate form
    if (!validateForm(form)) {
        showMessage('tasks', '请填写所有必填字段', 'error');
        return;
    }
    
    const formData = new FormData(form);
    const code = formData.get('code').trim();

    // Validate input
    if (!code) {
        showMessage('tasks', '兑换码不能为空', 'error');
        return;
    }

    // Show loading
    showLoading();

    try {
        await apiRequest('/tasks', {
            method: 'POST',
            body: JSON.stringify({ code })
        });

        showMessage('tasks', '兑换码添加成功', 'success');
        form.reset();
        
        // Refresh task list
        loadTasksView();
    } catch (error) {
        showMessage('tasks', `添加失败: ${error.message}`, 'error');
    } finally {
        hideLoading();
    }
}

// Show user details
async function showUserDetails(fid) {
    // Hide all views
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
    });
    
    // Show user details view
    document.getElementById('user-details-view').classList.add('active');
    
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });

    const contentEl = document.getElementById('user-details-content');
    contentEl.innerHTML = '<div class="loading">加载中...</div>';

    try {
        const response = await apiRequest(`/users/${fid}/codes`);
        const records = response.data.records || [];

        let html = `<h3 style="margin-bottom: 1rem;">用户 ${fid} 的兑换记录 (${records.length})</h3>`;

        if (records.length === 0) {
            html += '<div class="empty-state">该用户暂无兑换记录</div>';
        } else {
            html += `
                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>激活码</th>
                                <th>状态</th>
                                <th>兑换时间</th>
                                <th>结果</th>
                            </tr>
                        </thead>
                        <tbody>
            `;

            records.forEach(record => {
                const createdAt = record.created_at ? new Date(record.created_at).toLocaleString('zh-CN') : '-';
                const status = record.all_done ? 'completed' : (record.retry_count > 0 ? 'failed' : 'pending');
                const statusText = record.status == "success" ? '已完成' : (record.status == 'failed' ? '失败' : '重复领取');
                const result = record.result || (record.last_error || '-');
                
                html += `
                    <tr>
                        <td>${record.code || '-'}</td>
                        <td><span class="status-badge status-${status}">${statusText}</span></td>
                        <td>${createdAt}</td>
                        <td>${result}</td>
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
        showMessage('user-details', error.message, 'error');
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
