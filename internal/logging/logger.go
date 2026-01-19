package logging

import (
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// sensitivePatterns 敏感信息的正则表达式模式
var sensitivePatterns = []*regexp.Regexp{
	// API Keys (通常是20+位的字母数字字符串)
	regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?key|secret[_-]?key)["\s:=]+([a-zA-Z0-9/+]{20,})`),
	// Tokens (Bearer tokens, JWT等)
	regexp.MustCompile(`(?i)(token|bearer)["\s:=]+([a-zA-Z0-9_\-\.]{20,})`),
	// 密码
	regexp.MustCompile(`(?i)(password|passwd|pwd)["\s:=]+([^\s"']{6,})`),
	// 通用密钥模式 (secret后跟下划线或冒号)
	regexp.MustCompile(`(?i)(secret)["\s:_=]+([a-zA-Z0-9_]{12,})`),
}

// SetupLogger 配置日志系统
func SetupLogger(level, format string) (*logrus.Logger, error) {
	logger := logrus.New()

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(logLevel)

	// 设置日志格式
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
	}

	// 设置输出
	logger.SetOutput(os.Stdout)

	// 启用调用者信息（文件名和行号）
	logger.SetReportCaller(true)

	return logger, nil
}

// RedactSensitiveData 脱敏敏感数据
func RedactSensitiveData(data string) string {
	result := data

	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 保留键名和部分值
			parts := pattern.FindStringSubmatch(match)
			if len(parts) >= 3 {
				key := parts[1]
				value := parts[2]
				// 只显示前4个和后4个字符
				if len(value) > 8 {
					masked := value[:4] + "****" + value[len(value)-4:]
					return key + "=" + masked
				}
				// 如果值太短，完全脱敏
				return key + "=****"
			}
			return "****"
		})
	}

	return result
}

// SensitiveHook 日志钩子，自动脱敏敏感信息
type SensitiveHook struct{}

// Levels 返回此钩子应用的日志级别
func (hook *SensitiveHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 在日志写入前执行
func (hook *SensitiveHook) Fire(entry *logrus.Entry) error {
	// 脱敏日志消息
	entry.Message = RedactSensitiveData(entry.Message)

	// 脱敏字段中的敏感数据
	for key, value := range entry.Data {
		if str, ok := value.(string); ok {
			// 对于可能是敏感值的字段，直接脱敏
			if isSensitiveField(key) && len(str) >= 20 {
				// 只显示前4个和后4个字符
				if len(str) > 8 {
					entry.Data[key] = str[:4] + "****" + str[len(str)-4:]
				} else {
					entry.Data[key] = "****"
				}
			} else {
				// 对于其他字段，使用正则脱敏
				entry.Data[key] = RedactSensitiveData(str)
			}
		}
	}

	return nil
}

// isSensitiveField 检查字段名是否为敏感字段
func isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"api_key", "apikey", "access_key", "accesskey",
		"secret_key", "secretkey", "secret", "password",
		"passwd", "pwd", "token", "bearer", "authorization",
	}

	lowerField := strings.ToLower(fieldName)
	for _, sf := range sensitiveFields {
		if lowerField == sf || strings.Contains(lowerField, sf) {
			return true
		}
	}
	return false
}

// AddGlobalFields 添加全局字段到logger
func AddGlobalFields(logger *logrus.Logger, serviceName, version string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
	})
}

// WithRequestID 添加请求ID到日志
func WithRequestID(logger *logrus.Entry, requestID string) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

// WithCorrelationID 添加关联ID到日志
func WithCorrelationID(logger *logrus.Entry, correlationID string) *logrus.Entry {
	return logger.WithField("correlation_id", correlationID)
}

// WithError 添加错误信息到日志
func WithError(logger *logrus.Entry, err error) *logrus.Entry {
	return logger.WithError(err)
}

// WithOperation 添加操作名称到日志
func WithOperation(logger *logrus.Entry, operation string) *logrus.Entry {
	return logger.WithField("operation", operation)
}

// SanitizeForLog 清理字符串以便安全记录
func SanitizeForLog(s string) string {
	// 移除换行符和控制字符
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// 脱敏敏感信息
	s = RedactSensitiveData(s)

	return s
}
