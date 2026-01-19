package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name: "error without wrapped error",
			appError: &AppError{
				Code:    ErrCodeValidation,
				Message: "invalid input",
			},
			want: "[VALIDATION_ERROR] invalid input",
		},
		{
			name: "error with wrapped error",
			appError: &AppError{
				Code:    ErrCodeDatabase,
				Message: "query failed",
				Err:     errors.New("connection refused"),
			},
			want: "[DATABASE_ERROR] query failed: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appError.Error(); got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := Wrap(originalErr, ErrCodeInternal, "wrapped error")

	unwrapped := appErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// 测试 errors.Is
	if !errors.Is(appErr, originalErr) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestAppError_WithContext(t *testing.T) {
	appErr := New(ErrCodeValidation, "test error")
	appErr.WithContext("field", "username").WithContext("value", "test")

	if appErr.Context["field"] != "username" {
		t.Errorf("Context['field'] = %v, want 'username'", appErr.Context["field"])
	}

	if appErr.Context["value"] != "test" {
		t.Errorf("Context['value'] = %v, want 'test'", appErr.Context["value"])
	}
}

func TestNewDatabaseError(t *testing.T) {
	originalErr := errors.New("connection failed")
	appErr := NewDatabaseError("SELECT", originalErr)

	if appErr.Code != ErrCodeDatabase {
		t.Errorf("Code = %v, want %v", appErr.Code, ErrCodeDatabase)
	}

	if appErr.Context["operation"] != "SELECT" {
		t.Errorf("Context['operation'] = %v, want 'SELECT'", appErr.Context["operation"])
	}

	if !errors.Is(appErr, originalErr) {
		t.Error("should preserve error chain")
	}
}

func TestNewValidationError(t *testing.T) {
	appErr := NewValidationError("email", "invalid format")

	if appErr.Code != ErrCodeValidation {
		t.Errorf("Code = %v, want %v", appErr.Code, ErrCodeValidation)
	}

	if appErr.Context["field"] != "email" {
		t.Errorf("Context['field'] = %v, want 'email'", appErr.Context["field"])
	}

	if appErr.Context["reason"] != "invalid format" {
		t.Errorf("Context['reason'] = %v, want 'invalid format'", appErr.Context["reason"])
	}
}

func TestNewNotFoundError(t *testing.T) {
	appErr := NewNotFoundError("user", "123")

	if appErr.Code != ErrCodeNotFound {
		t.Errorf("Code = %v, want %v", appErr.Code, ErrCodeNotFound)
	}

	if appErr.Context["resource"] != "user" {
		t.Errorf("Context['resource'] = %v, want 'user'", appErr.Context["resource"])
	}

	if appErr.Context["id"] != "123" {
		t.Errorf("Context['id'] = %v, want '123'", appErr.Context["id"])
	}
}

func TestErrorChainPreservation(t *testing.T) {
	// 创建错误链
	err1 := errors.New("root cause")
	err2 := Wrap(err1, ErrCodeExternal, "API call failed")
	err3 := Wrap(err2, ErrCodeInternal, "processing failed")

	// 验证错误链完整性
	if !errors.Is(err3, err1) {
		t.Error("error chain should preserve root cause")
	}

	if !errors.Is(err3, err2) {
		t.Error("error chain should preserve intermediate error")
	}

	// 验证可以 unwrap 到原始错误
	unwrapped := errors.Unwrap(err3)
	if unwrapped != err2 {
		t.Error("first unwrap should return err2")
	}

	unwrapped = errors.Unwrap(unwrapped)
	if unwrapped != err1 {
		t.Error("second unwrap should return err1")
	}
}

func TestAllErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		createFn func() *AppError
		wantCode string
	}{
		{
			name:     "NewDatabaseError",
			createFn: func() *AppError { return NewDatabaseError("INSERT", errors.New("test")) },
			wantCode: ErrCodeDatabase,
		},
		{
			name:     "NewValidationError",
			createFn: func() *AppError { return NewValidationError("field", "reason") },
			wantCode: ErrCodeValidation,
		},
		{
			name:     "NewNotFoundError",
			createFn: func() *AppError { return NewNotFoundError("resource", "id") },
			wantCode: ErrCodeNotFound,
		},
		{
			name:     "NewAlreadyExistsError",
			createFn: func() *AppError { return NewAlreadyExistsError("resource", "id") },
			wantCode: ErrCodeAlreadyExists,
		},
		{
			name:     "NewCaptchaError",
			createFn: func() *AppError { return NewCaptchaError("test", nil) },
			wantCode: ErrCodeCaptcha,
		},
		{
			name:     "NewExternalAPIError",
			createFn: func() *AppError { return NewExternalAPIError("api", errors.New("test")) },
			wantCode: ErrCodeExternal,
		},
		{
			name:     "NewTimeoutError",
			createFn: func() *AppError { return NewTimeoutError("operation") },
			wantCode: ErrCodeTimeout,
		},
		{
			name:     "NewUnauthorizedError",
			createFn: func() *AppError { return NewUnauthorizedError("reason") },
			wantCode: ErrCodeUnauthorized,
		},
		{
			name:     "NewRateLimitError",
			createFn: func() *AppError { return NewRateLimitError() },
			wantCode: ErrCodeRateLimit,
		},
		{
			name:     "NewInternalError",
			createFn: func() *AppError { return NewInternalError("message", nil) },
			wantCode: ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createFn()
			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}
			if err.Context == nil {
				t.Error("Context should be initialized")
			}
		})
	}
}
