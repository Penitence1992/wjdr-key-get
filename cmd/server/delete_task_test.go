package main

import (
	"cdk-get/internal/api"
	"cdk-get/internal/auth"
	"cdk-get/internal/config"
	"cdk-get/internal/storage"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestDeleteTaskEndpoint tests the DELETE /api/admin/tasks/:code endpoint
func TestDeleteTaskEndpoint(t *testing.T) {
	// Setup test configuration
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
		},
		Admin: config.AdminConfig{
			Username:      "admin",
			PasswordHash:  "$2a$10$test",
			TokenSecret:   "test-secret",
			TokenDuration: 24 * time.Hour,
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create mock repository
	mockRepo := &storage.MockRepository{}

	// Create auth service
	authService := auth.NewAuthService(
		cfg.Admin.Username,
		cfg.Admin.PasswordHash,
		cfg.Admin.TokenSecret,
		cfg.Admin.TokenDuration,
	)

	// Create handlers
	handlers := api.NewHandlers(nil, nil, logger)
	adminHandlers := api.NewAdminHandlers(authService, mockRepo, logger)

	// Setup server
	server := setupServer(cfg, handlers, adminHandlers, authService, logger)

	// Generate a valid token for authentication
	token, _, err := authService.GenerateToken("admin")
	assert.NoError(t, err)

	tests := []struct {
		name           string
		code           string
		setupMock      func(*storage.MockRepository)
		expectedStatus int
		expectedBody   map[string]interface{}
		skipBodyCheck  bool
	}{
		{
			name: "Successfully delete existing task",
			code: "TEST123",
			setupMock: func(m *storage.MockRepository) {
				m.DeleteTaskFunc = func(ctx context.Context, code string) error {
					if code == "TEST123" {
						return nil
					}
					return storage.ErrTaskNotFound
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"message": "任务删除成功",
			},
		},
		{
			name: "Task not found",
			code: "NOTFOUND",
			setupMock: func(m *storage.MockRepository) {
				m.DeleteTaskFunc = func(ctx context.Context, code string) error {
					return storage.ErrTaskNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"success": false,
				"error":   "任务不存在",
			},
		},
		{
			name: "Whitespace only task code",
			code: "   ",
			setupMock: func(m *storage.MockRepository) {
				m.DeleteTaskFunc = func(ctx context.Context, code string) error {
					return nil
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"error":   "无效的任务代码",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.setupMock(mockRepo)

			// Create request - URL encode the code to handle special characters
			urlPath := "/api/admin/tasks/" + url.PathEscape(tt.code)
			req := httptest.NewRequest(http.MethodDelete, urlPath, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			// Execute request
			server.Handler.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Skip body check for certain tests
			if tt.skipBodyCheck {
				return
			}

			// Assert response body
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)

			for key, expectedValue := range tt.expectedBody {
				actualValue, exists := response[key]
				assert.True(t, exists, "Expected key %s not found in response", key)
				assert.Equal(t, expectedValue, actualValue, "Value mismatch for key %s", key)
			}
		})
	}
}

// TestDeleteTaskUnauthorized tests that unauthorized requests are rejected
func TestDeleteTaskUnauthorized(t *testing.T) {
	// Setup test configuration
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "json",
		},
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			CORS: config.CORSConfig{
				Enabled: false,
			},
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
		},
		Admin: config.AdminConfig{
			Username:      "admin",
			PasswordHash:  "$2a$10$test",
			TokenSecret:   "test-secret",
			TokenDuration: 24 * time.Hour,
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create mock repository
	mockRepo := &storage.MockRepository{}

	// Create auth service
	authService := auth.NewAuthService(
		cfg.Admin.Username,
		cfg.Admin.PasswordHash,
		cfg.Admin.TokenSecret,
		cfg.Admin.TokenDuration,
	)

	// Create handlers
	handlers := api.NewHandlers(nil, nil, logger)
	adminHandlers := api.NewAdminHandlers(authService, mockRepo, logger)

	// Setup server
	server := setupServer(cfg, handlers, adminHandlers, authService, logger)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token format",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing Bearer prefix",
			authHeader:     "invalid-token",
			expectedStatus: http.StatusBadRequest, // Auth middleware returns 400 for invalid format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/tasks/TEST123", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			// Execute request
			server.Handler.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")
		})
	}
}
