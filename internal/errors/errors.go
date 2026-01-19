package errors

import (
	"fmt"
)

// 错误代码常量
const (
	ErrCodeDatabase      = "DATABASE_ERROR"
	ErrCodeCaptcha       = "CAPTCHA_ERROR"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodeExternal      = "EXTERNAL_API_ERROR"
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeTimeout       = "TIMEOUT_ERROR"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeRateLimit     = "RATE_LIMIT_EXCEEDED"
)

// AppError 应用错误类型
type AppError struct {
	Code    string                 // 错误代码
	Message string                 // 错误消息
	Err     error                  // 原始错误
	Context map[string]interface{} // 错误上下文
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口，支持错误链
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New 创建新的应用错误
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// Wrap 包装现有错误
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// NewDatabaseError 创建数据库错误
func NewDatabaseError(operation string, err error) *AppError {
	return Wrap(err, ErrCodeDatabase, fmt.Sprintf("database operation failed: %s", operation)).
		WithContext("operation", operation)
}

// NewValidationError 创建验证错误
func NewValidationError(field, reason string) *AppError {
	return New(ErrCodeValidation, fmt.Sprintf("validation failed for field '%s': %s", field, reason)).
		WithContext("field", field).
		WithContext("reason", reason)
}

// NewNotFoundError 创建资源不存在错误
func NewNotFoundError(resource, id string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found: %s", resource, id)).
		WithContext("resource", resource).
		WithContext("id", id)
}

// NewAlreadyExistsError 创建资源已存在错误
func NewAlreadyExistsError(resource, id string) *AppError {
	return New(ErrCodeAlreadyExists, fmt.Sprintf("%s already exists: %s", resource, id)).
		WithContext("resource", resource).
		WithContext("id", id)
}

// NewCaptchaError 创建验证码错误
func NewCaptchaError(message string, err error) *AppError {
	if err != nil {
		return Wrap(err, ErrCodeCaptcha, message)
	}
	return New(ErrCodeCaptcha, message)
}

// NewExternalAPIError 创建外部API错误
func NewExternalAPIError(api string, err error) *AppError {
	return Wrap(err, ErrCodeExternal, fmt.Sprintf("external API call failed: %s", api)).
		WithContext("api", api)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(operation string) *AppError {
	return New(ErrCodeTimeout, fmt.Sprintf("operation timed out: %s", operation)).
		WithContext("operation", operation)
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(reason string) *AppError {
	return New(ErrCodeUnauthorized, fmt.Sprintf("unauthorized: %s", reason)).
		WithContext("reason", reason)
}

// NewRateLimitError 创建限流错误
func NewRateLimitError() *AppError {
	return New(ErrCodeRateLimit, "rate limit exceeded")
}

// NewInternalError 创建内部错误
func NewInternalError(message string, err error) *AppError {
	if err != nil {
		return Wrap(err, ErrCodeInternal, message)
	}
	return New(ErrCodeInternal, message)
}
