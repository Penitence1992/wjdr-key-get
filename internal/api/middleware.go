package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RequestIDMiddleware 为每个请求生成唯一ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取，如果没有则生成新的
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// LoggerMiddleware 结构化请求日志
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 获取请求ID
		requestID, _ := c.Get("request_id")

		// 记录请求信息
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("request started")

		// 处理请求
		c.Next()

		// 计算请求耗时
		duration := time.Since(startTime)

		// 记录响应信息
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration.Milliseconds(),
			"size":       c.Writer.Size(),
		}).Info("request completed")
	}
}

// RecoveryMiddleware panic恢复中间件
func RecoveryMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取请求ID
				requestID, _ := c.Get("request_id")

				// 记录panic信息
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"panic":      err,
				}).Error("panic recovered")

				// 返回500错误
				c.JSON(500, ErrorResponse("INTERNAL_ERROR", "Internal server error"))
				c.Abort()
			}
		}()

		c.Next()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(rateLimit int, burst int) gin.HandlerFunc {
	// 创建限流器
	limiter := rate.NewLimiter(rate.Limit(rateLimit), burst)

	return func(c *gin.Context) {
		// 检查是否允许请求
		if !limiter.Allow() {
			c.JSON(429, ErrorResponse("RATE_LIMIT_EXCEEDED", "Too many requests"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidationMiddleware 请求验证中间件
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 对于POST/PUT/PATCH请求，验证Content-Type
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.ContentType()

			// 允许的Content-Type
			validContentTypes := map[string]bool{
				"application/json":                  true,
				"application/x-www-form-urlencoded": true,
				"multipart/form-data":               true,
			}

			// 检查Content-Type是否有效
			if contentType != "" && !validContentTypes[contentType] {
				c.JSON(415, ErrorResponse("UNSUPPORTED_MEDIA_TYPE",
					fmt.Sprintf("Content-Type '%s' is not supported", contentType)))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AuthService 认证服务接口（用于中间件）
type AuthService interface {
	ValidateToken(token string) (*AuthClaims, error)
}

// AuthClaims JWT声明结构（用于中间件）
type AuthClaims struct {
	Username string
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService AuthService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求ID用于日志
		requestID, _ := c.Get("request_id")

		// 从Authorization头提取token
		authHeader := c.GetHeader("Authorization")

		// 检查Authorization头是否存在
		if authHeader == "" {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       c.Request.URL.Path,
			}).Warn("missing authorization header")

			c.JSON(401, ErrorResponse("UNAUTHORIZED", "Missing authorization header"))
			c.Abort()
			return
		}

		// 验证Bearer格式
		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       c.Request.URL.Path,
			}).Warn("invalid authorization header format")

			c.JSON(400, ErrorResponse("INVALID_TOKEN_FORMAT", "Authorization header must be in format: Bearer <token>"))
			c.Abort()
			return
		}

		// 提取token
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       c.Request.URL.Path,
			}).Warn("empty token in authorization header")

			c.JSON(400, ErrorResponse("INVALID_TOKEN_FORMAT", "Token cannot be empty"))
			c.Abort()
			return
		}

		// 验证token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       c.Request.URL.Path,
				"error":      err.Error(),
			}).Warn("token validation failed")

			// 根据错误类型返回不同的响应
			errorMsg := "Invalid or expired token"
			c.JSON(401, ErrorResponse("UNAUTHORIZED", errorMsg))
			c.Abort()
			return
		}

		// 将管理员用户名存入上下文
		c.Set("admin_username", claims.Username)

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"username":   claims.Username,
			"path":       c.Request.URL.Path,
		}).Debug("authentication successful")

		c.Next()
	}
}
