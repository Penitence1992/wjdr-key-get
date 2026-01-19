package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config.Server.Port != 10999 {
		t.Errorf("expected default port 10999, got %d", config.Server.Port)
	}

	if config.Database.Path != "./giftcode.db" {
		t.Errorf("expected default db path './giftcode.db', got %s", config.Database.Path)
	}

	if config.Logging.Level != "info" {
		t.Errorf("expected default log level 'info', got %s", config.Logging.Level)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// 测试不提供配置文件时使用默认值
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if config.Server.Port != 10999 {
		t.Errorf("expected port 10999, got %d", config.Server.Port)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "valid config",
			config:    defaultConfig(),
			wantError: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server:   ServerConfig{Port: 0},
				Database: DatabaseConfig{Path: "./test.db", MaxOpenConns: 10},
				Job:      JobConfig{PeriodTime: 1 * time.Second, WorkerPoolSize: 1},
				Logging:  LoggingConfig{Level: "info", Format: "json"},
				Security: SecurityConfig{RateLimit: RateLimitConfig{Enabled: false}},
			},
			wantError: true,
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server:   ServerConfig{Port: 70000},
				Database: DatabaseConfig{Path: "./test.db", MaxOpenConns: 10},
				Job:      JobConfig{PeriodTime: 1 * time.Second, WorkerPoolSize: 1},
				Logging:  LoggingConfig{Level: "info", Format: "json"},
				Security: SecurityConfig{RateLimit: RateLimitConfig{Enabled: false}},
			},
			wantError: true,
		},
		{
			name: "empty database path",
			config: &Config{
				Server:   ServerConfig{Port: 8080, ReadTimeout: 1 * time.Second, WriteTimeout: 1 * time.Second},
				Database: DatabaseConfig{Path: "", MaxOpenConns: 10},
				Job:      JobConfig{PeriodTime: 1 * time.Second, WorkerPoolSize: 1},
				Logging:  LoggingConfig{Level: "info", Format: "json"},
				Security: SecurityConfig{RateLimit: RateLimitConfig{Enabled: false}},
			},
			wantError: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Server:   ServerConfig{Port: 8080, ReadTimeout: 1 * time.Second, WriteTimeout: 1 * time.Second},
				Database: DatabaseConfig{Path: "./test.db", MaxOpenConns: 10},
				Job:      JobConfig{PeriodTime: 1 * time.Second, WorkerPoolSize: 1},
				Logging:  LoggingConfig{Level: "invalid", Format: "json"},
				Security: SecurityConfig{RateLimit: RateLimitConfig{Enabled: false}},
			},
			wantError: true,
		},
		{
			name: "invalid captcha provider type",
			config: &Config{
				Server:   ServerConfig{Port: 8080, ReadTimeout: 1 * time.Second, WriteTimeout: 1 * time.Second},
				Database: DatabaseConfig{Path: "./test.db", MaxOpenConns: 10},
				Captcha: CaptchaConfig{
					Providers: []CaptchaProvider{
						{Type: "invalid", AccessKey: "key", SecretKey: "secret"},
					},
				},
				Job:      JobConfig{PeriodTime: 1 * time.Second, WorkerPoolSize: 1},
				Logging:  LoggingConfig{Level: "info", Format: "json"},
				Security: SecurityConfig{RateLimit: RateLimitConfig{Enabled: false}},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEnvOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("LOG_LEVEL")
	}()

	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Server.Port != 8080 {
		t.Errorf("expected port 8080 from env, got %d", config.Server.Port)
	}

	if config.Database.Path != "/tmp/test.db" {
		t.Errorf("expected db path '/tmp/test.db' from env, got %s", config.Database.Path)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("expected log level 'debug' from env, got %s", config.Logging.Level)
	}
}

func TestCaptchaProviderFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("ACCESS_KEY", "test-access-key")
	os.Setenv("ACCESS_SECRET", "test-secret-key")
	defer func() {
		os.Unsetenv("ACCESS_KEY")
		os.Unsetenv("ACCESS_SECRET")
	}()

	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 应该从环境变量添加阿里云提供商
	found := false
	for _, provider := range config.Captcha.Providers {
		if provider.Type == "ali" {
			found = true
			if provider.AccessKey != "test-access-key" {
				t.Errorf("expected access key 'test-access-key', got %s", provider.AccessKey)
			}
			if provider.SecretKey != "test-secret-key" {
				t.Errorf("expected secret key 'test-secret-key', got %s", provider.SecretKey)
			}
			break
		}
	}

	if !found {
		t.Error("expected ali provider to be added from environment variables")
	}
}
