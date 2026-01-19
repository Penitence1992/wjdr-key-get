package api

import (
	"cdk-get/internal/auth"
	"cdk-get/internal/storage"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AdminHandlers 管理后台API处理器
type AdminHandlers struct {
	authService auth.AuthService
	repository  storage.Repository
	logger      *logrus.Logger
}

// NewAdminHandlers 创建管理后台处理器实例
func NewAdminHandlers(authService auth.AuthService, repository storage.Repository, logger *logrus.Logger) *AdminHandlers {
	return &AdminHandlers{
		authService: authService,
		repository:  repository,
		logger:      logger,
	}
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Login 管理员登录处理器
// 处理 POST /api/admin/login
func (h *AdminHandlers) Login(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("login request validation failed")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// 验证凭证
	if err := h.authService.ValidateCredentials(req.Username, req.Password); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"username":   req.Username,
		}).Warn("invalid credentials")

		c.JSON(401, ErrorResponse("INVALID_CREDENTIALS", "Invalid username or password"))
		return
	}

	// 生成token
	token, expiresAt, err := h.authService.GenerateToken(req.Username)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"username":   req.Username,
			"error":      err.Error(),
		}).Error("failed to generate token")

		c.JSON(500, ErrorResponse("TOKEN_GENERATION_FAILED", "Failed to generate token"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   req.Username,
	}).Info("login successful")

	c.JSON(200, SuccessResponse(LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}))
}

// ListUsers 获取用户列表处理器
// 处理 GET /api/admin/users
func (h *AdminHandlers) ListUsers(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")
	ctx := c.Request.Context()

	users, err := h.repository.ListUsers(ctx)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("failed to fetch users")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch users"))
		return
	}

	// 处理空列表情况 - 返回空数组而不是nil
	if users == nil {
		users = []*storage.User{}
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"count":      len(users),
	}).Info("users fetched successfully")

	c.JSON(200, SuccessResponse(gin.H{"users": users}))
}

// AddUserRequest 添加用户请求结构
type AddUserRequest struct {
	FID         string `json:"fid" binding:"required"`
	Nickname    string `json:"nickname"`
	KID         int    `json:"kid"`
	AvatarImage string `json:"avatar_image"`
}

// AddUser 添加用户处理器
// 处理 POST /api/admin/users
func (h *AdminHandlers) AddUser(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("add user request validation failed")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	ctx := c.Request.Context()

	user := &storage.User{
		FID:         req.FID,
		Nickname:    req.Nickname,
		KID:         req.KID,
		AvatarImage: req.AvatarImage,
	}

	if err := h.repository.SaveUser(ctx, user); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"fid":        req.FID,
			"error":      err.Error(),
		}).Error("failed to save user")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to save user"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"fid":        req.FID,
	}).Info("user added successfully")

	c.JSON(200, SuccessResponse(gin.H{
		"message": "User added successfully",
		"fid":     req.FID,
	}))
}

// GetUserGiftCodes 获取用户兑换记录处理器
// 处理 GET /api/admin/users/:fid/codes
func (h *AdminHandlers) GetUserGiftCodes(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	// 从URL参数提取FID
	fid := c.Param("fid")
	ctx := c.Request.Context()

	records, err := h.repository.ListGiftCodesByFID(ctx, fid)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"fid":        fid,
			"error":      err.Error(),
		}).Error("failed to fetch gift codes")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch gift codes"))
		return
	}

	// 处理空列表情况 - 返回空数组而不是nil
	if records == nil {
		records = []*storage.GiftCodeRecord{}
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"fid":        fid,
		"count":      len(records),
	}).Info("gift codes fetched successfully")

	c.JSON(200, SuccessResponse(gin.H{"records": records}))
}

// ListTasks 获取任务列表处理器
// 处理 GET /api/admin/tasks
func (h *AdminHandlers) ListTasks(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")
	ctx := c.Request.Context()

	tasks, err := h.repository.ListPendingTasks(ctx)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("failed to fetch tasks")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to fetch tasks"))
		return
	}

	// 处理空列表情况 - 返回空数组而不是nil
	if tasks == nil {
		tasks = []*storage.Task{}
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"count":      len(tasks),
	}).Info("tasks fetched successfully")

	c.JSON(200, SuccessResponse(gin.H{"tasks": tasks}))
}

// AddGiftCodeRequest 添加兑换码请求结构
type AddGiftCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// AddGiftCode 添加兑换码处理器
// 处理 POST /api/admin/tasks
func (h *AdminHandlers) AddGiftCode(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	var req AddGiftCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("add gift code request validation failed")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// 验证兑换码格式（非空、非纯空白）
	trimmedCode := strings.TrimSpace(req.Code)
	if len(trimmedCode) == 0 {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"code":       req.Code,
		}).Warn("invalid gift code format")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", "Gift code cannot be empty or whitespace only"))
		return
	}

	ctx := c.Request.Context()

	if err := h.repository.CreateTask(ctx, req.Code); err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"code":       req.Code,
			"error":      err.Error(),
		}).Error("failed to create task")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to create task"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"code":       req.Code,
	}).Info("gift code task created successfully")

	c.JSON(200, SuccessResponse(gin.H{
		"message": "Gift code task created successfully",
		"code":    req.Code,
	}))
}

// ListCompletedTasks 获取已完成任务列表处理器
// 处理 GET /api/admin/tasks/completed
func (h *AdminHandlers) ListCompletedTasks(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")
	ctx := c.Request.Context()

	// 从query参数读取limit（默认100）
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

	// 处理空列表情况 - 返回空数组而不是nil
	if tasks == nil {
		tasks = []*storage.Task{}
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"count":      len(tasks),
	}).Info("completed tasks fetched successfully")

	c.JSON(200, SuccessResponse(gin.H{"tasks": tasks}))
}

// ListNotifications 获取通知历史列表处理器
// 处理 GET /api/admin/notifications
func (h *AdminHandlers) ListNotifications(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")
	ctx := c.Request.Context()

	// 从query参数读取limit（默认100）
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

	// 处理空列表情况 - 返回空数组而不是nil
	if notifications == nil {
		notifications = []*storage.Notification{}
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"count":      len(notifications),
	}).Info("notifications fetched successfully")

	c.JSON(200, SuccessResponse(gin.H{"notifications": notifications}))
}

// DeleteTask 删除任务处理器
// 处理 DELETE /api/admin/tasks/:code
func (h *AdminHandlers) DeleteTask(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	// 从URL参数提取任务code
	code := c.Param("code")

	// 记录删除请求开始
	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"code":       code,
		"method":     "DeleteTask",
	}).Info("delete task request received")

	// 验证code格式（非空、非纯空白）
	trimmedCode := strings.TrimSpace(code)
	if len(trimmedCode) == 0 {
		h.logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"code":        code,
			"error_type":  "validation_error",
			"method":      "DeleteTask",
			"description": "task code is empty or whitespace only",
		}).Warn("invalid task code format")

		c.JSON(400, gin.H{
			"success": false,
			"error":   "无效的任务代码",
		})
		return
	}

	ctx := c.Request.Context()

	// 调用repository删除任务
	err := h.repository.DeleteTask(ctx, trimmedCode)
	if err != nil {
		// 检查是否是任务不存在错误
		if errors.Is(err, storage.ErrTaskNotFound) {
			h.logger.WithFields(logrus.Fields{
				"request_id":  requestID,
				"code":        trimmedCode,
				"error_type":  "not_found",
				"method":      "DeleteTask",
				"description": "task does not exist in database",
			}).Warn("task not found for deletion")

			c.JSON(404, gin.H{
				"success": false,
				"error":   "任务不存在",
			})
			return
		}

		// 其他数据库错误 - 记录详细错误信息
		h.logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"code":        trimmedCode,
			"error_type":  "database_error",
			"error":       err.Error(),
			"method":      "DeleteTask",
			"description": "failed to delete task from database",
		}).Error("failed to delete task")

		c.JSON(500, gin.H{
			"success": false,
			"error":   "删除任务失败，请稍后重试",
		})
		return
	}

	// 删除成功 - 记录成功日志
	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"code":       trimmedCode,
		"method":     "DeleteTask",
		"result":     "success",
	}).Info("task deleted successfully")

	c.JSON(200, gin.H{
		"success": true,
		"message": "任务删除成功",
	})
}
