package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config 主配置结构
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Captcha      CaptchaConfig      `yaml:"captcha"`
	Job          JobConfig          `yaml:"job"`
	Logging      LoggingConfig      `yaml:"logging"`
	Security     SecurityConfig     `yaml:"security"`
	Admin        AdminConfig        `yaml:"admin"`
	Notification NotificationConfig `yaml:"notification"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
	CORS         CORSConfig    `yaml:"cors"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"` // 预检请求缓存时间（秒）
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path            string        `yaml:"path"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// CaptchaConfig 验证码服务配置
type CaptchaConfig struct {
	Providers []CaptchaProvider `yaml:"providers"`
}

// CaptchaProvider 验证码提供商配置
type CaptchaProvider struct {
	Type            string `yaml:"type"`             // "ali", "tencent", or "google"
	AccessKey       string `yaml:"access_key"`       // 可以从环境变量覆盖 (ali/tencent)
	SecretKey       string `yaml:"secret_key"`       // 可以从环境变量覆盖 (ali/tencent)
	CredentialsJSON string `yaml:"credentials_json"` // Google Cloud credentials JSON (google)
}

// JobConfig 任务调度配置
type JobConfig struct {
	DelayTime      time.Duration `yaml:"delay_time"`
	PeriodTime     time.Duration `yaml:"period_time"`
	WorkerPoolSize int           `yaml:"worker_pool_size"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled bool `yaml:"enabled"`
	Rate    int  `yaml:"rate"`  // 每秒请求数
	Burst   int  `yaml:"burst"` // 突发请求数
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Username      string        `yaml:"username"`       // 管理员用户名
	PasswordHash  string        `yaml:"password_hash"`  // 管理员密码的bcrypt哈希
	TokenSecret   string        `yaml:"token_secret"`   // JWT签名密钥
	TokenDuration time.Duration `yaml:"token_duration"` // JWT令牌有效期
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	WxPusher WxPusherConfig `yaml:"wxpusher"`
}

// WxPusherConfig WxPusher通知配置
type WxPusherConfig struct {
	AppToken string `yaml:"app_token"` // WxPusher应用Token
	UID      string `yaml:"uid"`       // WxPusher用户UID
}

// LoadConfig 从文件和环境变量加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 设置默认配置
	config := defaultConfig()

	// 如果提供了配置文件路径，从文件加载
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// 从环境变量覆盖配置
	overrideFromEnv(config)

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         10999,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			CORS: CORSConfig{
				Enabled:          true,
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
				ExposeHeaders:    []string{"X-Request-ID"},
				AllowCredentials: false,
				MaxAge:           3600,
			},
		},
		Database: DatabaseConfig{
			Path:            "./giftcode.db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Captcha: CaptchaConfig{
			Providers: []CaptchaProvider{},
		},
		Job: JobConfig{
			DelayTime:      2 * time.Second,
			PeriodTime:     30 * time.Second,
			WorkerPoolSize: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Security: SecurityConfig{
			RateLimit: RateLimitConfig{
				Enabled: true,
				Rate:    10,
				Burst:   20,
			},
		},
		Admin: AdminConfig{
			Username:      "admin",
			PasswordHash:  "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // bcrypt hash of "admin123"
			TokenSecret:   "default-secret-key-change-in-production-min-32-characters",
			TokenDuration: 24 * time.Hour,
		},
		Notification: NotificationConfig{
			WxPusher: WxPusherConfig{
				AppToken: "",
				UID:      "",
			},
		},
	}
}

// loadFromFile 从YAML文件加载配置
func loadFromFile(config *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// overrideFromEnv 从环境变量覆盖配置
func overrideFromEnv(config *Config) {
	// Server配置
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &config.Server.Port)
	}

	// Database配置
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}

	// Captcha配置 - 从环境变量添加提供商
	if accessKey := os.Getenv("ACCESS_KEY"); accessKey != "" {
		if secretKey := os.Getenv("ACCESS_SECRET"); secretKey != "" {
			// 检查是否已存在阿里云配置
			found := false
			for i := range config.Captcha.Providers {
				if config.Captcha.Providers[i].Type == "ali" {
					config.Captcha.Providers[i].AccessKey = accessKey
					config.Captcha.Providers[i].SecretKey = secretKey
					found = true
					break
				}
			}
			if !found {
				config.Captcha.Providers = append(config.Captcha.Providers, CaptchaProvider{
					Type:      "ali",
					AccessKey: accessKey,
					SecretKey: secretKey,
				})
			}
		}
	}

	// Google Captcha配置 - 从环境变量添加或覆盖
	if googleCreds := os.Getenv("GOOGLE_CREDENTIALS_JSON"); googleCreds != "" {
		// 检查是否已存在Google配置
		found := false
		for i := range config.Captcha.Providers {
			if config.Captcha.Providers[i].Type == "google" {
				config.Captcha.Providers[i].CredentialsJSON = googleCreds
				found = true
				break
			}
		}
		if !found {
			config.Captcha.Providers = append(config.Captcha.Providers, CaptchaProvider{
				Type:            "google",
				CredentialsJSON: googleCreds,
			})
		}
	}

	// Logging配置
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}

	// Admin配置
	if adminUsername := os.Getenv("ADMIN_USERNAME"); adminUsername != "" {
		config.Admin.Username = adminUsername
	}
	if adminPasswordHash := os.Getenv("ADMIN_PASSWORD_HASH"); adminPasswordHash != "" {
		config.Admin.PasswordHash = adminPasswordHash
	}
	if adminTokenSecret := os.Getenv("ADMIN_TOKEN_SECRET"); adminTokenSecret != "" {
		config.Admin.TokenSecret = adminTokenSecret
	}
	if adminTokenDuration := os.Getenv("ADMIN_TOKEN_DURATION"); adminTokenDuration != "" {
		if duration, err := time.ParseDuration(adminTokenDuration); err == nil {
			config.Admin.TokenDuration = duration
		}
	}

	// Notification配置
	if wxpusherAppToken := os.Getenv("WXPUSHER_APP_TOKEN"); wxpusherAppToken != "" {
		config.Notification.WxPusher.AppToken = wxpusherAppToken
	}
	if wxpusherUID := os.Getenv("WXPUSHER_UID"); wxpusherUID != "" {
		config.Notification.WxPusher.UID = wxpusherUID
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证Server配置
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", c.Server.Port)
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("invalid server read_timeout: %v (must be positive)", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("invalid server write_timeout: %v (must be positive)", c.Server.WriteTimeout)
	}

	// 验证Database配置
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("invalid database max_open_conns: %d (must be positive)", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("invalid database max_idle_conns: %d (must be non-negative)", c.Database.MaxIdleConns)
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database max_idle_conns (%d) cannot exceed max_open_conns (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}

	// 验证Captcha配置
	for i, provider := range c.Captcha.Providers {
		if provider.Type != "ali" && provider.Type != "tencent" && provider.Type != "google" {
			return fmt.Errorf("invalid captcha provider type at index %d: %s (must be 'ali', 'tencent', or 'google')",
				i, provider.Type)
		}

		// 验证不同类型的提供商所需的凭证
		switch provider.Type {
		case "ali", "tencent":
			if provider.AccessKey == "" {
				return fmt.Errorf("captcha provider at index %d is missing access_key", i)
			}
			if provider.SecretKey == "" {
				return fmt.Errorf("captcha provider at index %d is missing secret_key", i)
			}
		case "google":
			if provider.CredentialsJSON == "" {
				return fmt.Errorf("captcha provider at index %d is missing credentials_json", i)
			}
		}
	}

	// 验证Job配置
	if c.Job.DelayTime < 0 {
		return fmt.Errorf("invalid job delay_time: %v (must be non-negative)", c.Job.DelayTime)
	}
	if c.Job.PeriodTime <= 0 {
		return fmt.Errorf("invalid job period_time: %v (must be positive)", c.Job.PeriodTime)
	}
	if c.Job.WorkerPoolSize <= 0 {
		return fmt.Errorf("invalid job worker_pool_size: %d (must be positive)", c.Job.WorkerPoolSize)
	}

	// 验证Logging配置
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s (must be one of: debug, info, warn, error)", c.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s (must be 'json' or 'text')", c.Logging.Format)
	}

	// 验证Security配置
	if c.Security.RateLimit.Enabled {
		if c.Security.RateLimit.Rate <= 0 {
			return fmt.Errorf("invalid rate_limit rate: %d (must be positive when enabled)", c.Security.RateLimit.Rate)
		}
		if c.Security.RateLimit.Burst <= 0 {
			return fmt.Errorf("invalid rate_limit burst: %d (must be positive when enabled)", c.Security.RateLimit.Burst)
		}
	}

	// 验证Admin配置
	if c.Admin.Username == "" {
		return fmt.Errorf("admin username is required")
	}
	if c.Admin.PasswordHash == "" {
		return fmt.Errorf("admin password_hash is required")
	}
	if c.Admin.TokenSecret == "" {
		return fmt.Errorf("admin token_secret is required")
	}
	if len(c.Admin.TokenSecret) < 32 {
		return fmt.Errorf("admin token_secret must be at least 32 characters long")
	}
	if c.Admin.TokenDuration <= 0 {
		return fmt.Errorf("invalid admin token_duration: %v (must be positive)", c.Admin.TokenDuration)
	}

	return nil
}
