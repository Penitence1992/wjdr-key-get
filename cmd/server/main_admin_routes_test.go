package main

import (
	"cdk-get/internal/api"
	"cdk-get/internal/auth"
	"cdk-get/internal/config"
	"cdk-get/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestAdminStaticRoutes tests the admin static file routes
func TestAdminStaticRoutes(t *testing.T) {
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
		path           string
		expectedStatus int
		expectedType   string
		checkRedirect  bool
		redirectTo     string
	}{
		{
			name:           "Admin root redirects to login",
			path:           "/admin",
			expectedStatus: http.StatusFound,
			checkRedirect:  true,
			redirectTo:     "/admin/login.html",
		},
		{
			name:           "Login page is accessible",
			path:           "/admin/login.html",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
		{
			name:           "Dashboard page is accessible",
			path:           "/admin/dashboard.html",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
		{
			name:           "Dashboard JS is accessible",
			path:           "/admin/dashboard.js",
			expectedStatus: http.StatusOK,
			expectedType:   "application/javascript",
		},
		{
			name:           "Styles CSS is accessible",
			path:           "/admin/styles.css",
			expectedStatus: http.StatusOK,
			expectedType:   "text/css",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.Handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			if tt.checkRedirect {
				location := w.Header().Get("Location")
				assert.Equal(t, tt.redirectTo, location, "Redirect location mismatch")
			}

			if tt.expectedType != "" && w.Code == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				assert.Contains(t, contentType, tt.expectedType, "Content-Type mismatch")
			}

			// Verify Cache-Control header is set
			if w.Code == http.StatusOK {
				cacheControl := w.Header().Get("Cache-Control")
				assert.NotEmpty(t, cacheControl, "Cache-Control header should be set")
			}
		})
	}
}
