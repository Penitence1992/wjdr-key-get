package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		format    string
		wantError bool
	}{
		{
			name:      "valid json logger",
			level:     "info",
			format:    "json",
			wantError: false,
		},
		{
			name:      "valid text logger",
			level:     "debug",
			format:    "text",
			wantError: false,
		},
		{
			name:      "invalid level",
			level:     "invalid",
			format:    "json",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := SetupLogger(tt.level, tt.format)
			if (err != nil) != tt.wantError {
				t.Errorf("SetupLogger() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && logger == nil {
				t.Error("expected non-nil logger")
			}
		})
	}
}

func TestRedactSensitiveData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string // 应该包含的内容
		notContains []string // 不应该包含的内容
	}{
		{
			name:        "redact API key",
			input:       "api_key=AKIAIOSFODNN7EXAMPLE",
			contains:    []string{"api_key", "AKIA", "MPLE"},
			notContains: []string{"AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:        "redact access key",
			input:       "ACCESS_KEY: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			contains:    []string{"ACCESS_KEY"},
			notContains: []string{"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
		},
		{
			name:        "redact bearer token",
			input:       "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains:    []string{"Bearer"},
			notContains: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
		{
			name:        "redact password",
			input:       "password=MySecretPassword123",
			contains:    []string{"password"},
			notContains: []string{"MySecretPassword123"},
		},
		{
			name:        "redact secret",
			input:       "secret: my_super_secret_value_12345",
			contains:    []string{"secret"},
			notContains: []string{"my_super_secret_value_12345"},
		},
		{
			name:     "no sensitive data",
			input:    "This is a normal log message",
			contains: []string{"This is a normal log message"},
		},
		{
			name:        "multiple sensitive values",
			input:       "api_key=KEY1234567890123456789 password=Pass123456",
			contains:    []string{"api_key", "password"},
			notContains: []string{"KEY1234567890123456789", "Pass123456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}

			for _, s := range tt.notContains {
				if strings.Contains(result, s) {
					t.Errorf("expected result NOT to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestSensitiveHook(t *testing.T) {
	// 创建logger
	logger := logrus.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// 添加敏感信息钩子
	logger.AddHook(&SensitiveHook{})

	// 记录包含敏感信息的日志（使用足够长的密钥）
	logger.WithField("api_key", "AKIAIOSFODNN7EXAMPLEKEY123").Info("test message with api_key=AKIAIOSFODNN7EXAMPLEKEY123")

	output := buf.String()

	// 验证敏感信息被脱敏
	if strings.Contains(output, "AKIAIOSFODNN7EXAMPLEKEY123") {
		t.Errorf("sensitive data should be redacted in log output, got: %s", output)
	}

	// 验证日志仍然包含部分信息
	if !strings.Contains(output, "api_key") {
		t.Error("log should still contain field name")
	}
}

func TestAddGlobalFields(t *testing.T) {
	logger := logrus.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	entry := AddGlobalFields(logger, "test-service", "1.0.0")
	entry.Info("test message")

	output := buf.String()

	// 解析JSON输出
	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logData); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logData["service"] != "test-service" {
		t.Errorf("expected service=test-service, got %v", logData["service"])
	}

	if logData["version"] != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %v", logData["version"])
	}
}

func TestWithRequestID(t *testing.T) {
	logger := logrus.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	entry := logger.WithFields(logrus.Fields{})
	entry = WithRequestID(entry, "req-123")
	entry.Info("test message")

	output := buf.String()

	if !strings.Contains(output, "req-123") {
		t.Error("log should contain request_id")
	}
}

func TestWithCorrelationID(t *testing.T) {
	logger := logrus.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	entry := logger.WithFields(logrus.Fields{})
	entry = WithCorrelationID(entry, "corr-456")
	entry.Info("test message")

	output := buf.String()

	if !strings.Contains(output, "corr-456") {
		t.Error("log should contain correlation_id")
	}
}

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		notContains []string
	}{
		{
			name:        "remove newlines",
			input:       "line1\nline2\rline3",
			notContains: []string{"\n", "\r"},
		},
		{
			name:        "remove tabs",
			input:       "col1\tcol2\tcol3",
			notContains: []string{"\t"},
		},
		{
			name:        "redact sensitive data",
			input:       "api_key=SECRET1234567890123456",
			notContains: []string{"SECRET1234567890123456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLog(tt.input)

			for _, s := range tt.notContains {
				if strings.Contains(result, s) {
					t.Errorf("expected result NOT to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	logger, err := SetupLogger("info", "json")
	if err != nil {
		t.Fatalf("failed to setup logger: %v", err)
	}

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.Info("test message")

	output := buf.String()

	// 验证输出是有效的JSON
	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logData); err != nil {
		t.Errorf("log output is not valid JSON: %v", err)
	}

	// 验证必需字段存在
	requiredFields := []string{"timestamp", "level", "message"}
	for _, field := range requiredFields {
		if _, ok := logData[field]; !ok {
			t.Errorf("log output missing required field: %s", field)
		}
	}
}
