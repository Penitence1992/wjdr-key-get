package api

import (
	"cdk-get/internal/service"
	"cdk-get/internal/storage"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handlers API处理器集合
type Handlers struct {
	giftService *service.GiftService
	storage     storage.KeyStorage
	logger      *logrus.Logger
}

// NewHandlers 创建API处理器
func NewHandlers(giftService *service.GiftService, storage storage.KeyStorage, logger *logrus.Logger) *Handlers {
	return &Handlers{
		giftService: giftService,
		storage:     storage,
		logger:      logger,
	}
}

// AddGiftCode 添加礼品码任务
func (h *Handlers) AddGiftCode(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	// 获取并验证code参数
	code := c.Query("code")
	code = strings.TrimSpace(code)

	if code == "" {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      "code parameter is missing",
		}).Warn("validation failed")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", "code is required"))
		return
	}

	// 添加任务到数据库
	err := h.storage.AddTask(code)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"code":       code,
			"error":      err,
		}).Error("failed to create task")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to add gift code task"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"code":       code,
	}).Info("gift code task created successfully")

	c.JSON(200, SuccessResponse(gin.H{
		"message": "Gift code task added successfully",
		"code":    code,
	}))
}

// AddUser 添加用户
func (h *Handlers) AddUser(c *gin.Context) {
	// 获取请求ID用于日志关联
	requestID, _ := c.Get("request_id")

	// 获取并验证fid参数
	fid := c.Query("fid")
	fid = strings.TrimSpace(fid)

	if fid == "" {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      "fid parameter is missing",
		}).Warn("validation failed")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", "fid is required"))
		return
	}

	// 验证fid是否为有效的整数
	ifid, err := strconv.ParseInt(fid, 10, 64)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"fid":        fid,
			"error":      err,
		}).Warn("invalid fid format")

		c.JSON(400, ErrorResponse("VALIDATION_ERROR", "fid must be a valid integer"))
		return
	}

	// 保存用户到数据库
	err = h.storage.SaveFidInfo(int(ifid), "", -1, "")
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"fid":        fid,
			"error":      err,
		}).Error("failed to save user")

		c.JSON(500, ErrorResponse("DATABASE_ERROR", "Failed to add user"))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"fid":        fid,
	}).Info("user added successfully")

	c.JSON(200, SuccessResponse(gin.H{
		"message": "User added successfully",
		"fid":     fid,
	}))
}

// GetIP 获取服务器IP地址
func (h *Handlers) GetIP(c *gin.Context) {
	// 返回配置的IP地址
	c.String(200, "47.120.61.46")
}
