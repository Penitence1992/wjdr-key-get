package main

import (
	"cdk-get/internal/config"
	apperrors "cdk-get/internal/errors"
	"cdk-get/internal/httpclient"
	"cdk-get/internal/logging"
	"errors"
	"fmt"
	"os"
)

func main() {
	fmt.Println("=== Phase 1 基础设施验证 ===")

	// 1. 验证配置加载
	fmt.Println("1. 验证配置管理...")
	cfg, err := config.LoadConfig("./etc/config.yaml")
	if err != nil {
		fmt.Printf("   ❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✅ 配置加载成功\n")
	fmt.Printf("   - Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("   - Database: %s\n", cfg.Database.Path)
	fmt.Printf("   - Log Level: %s\n", cfg.Logging.Level)

	// 2. 验证错误处理
	fmt.Println("\n2. 验证错误处理...")
	testErr := apperrors.New("TEST_ERROR", "test error message")
	wrappedErr := apperrors.Wrap(testErr, "WRAPPED_ERROR", "wrapped test error")
	fmt.Printf("   ✅ 错误创建成功\n")
	fmt.Printf("   - 原始错误: %v\n", testErr)
	fmt.Printf("   - 包装错误: %v\n", wrappedErr)

	// 验证错误链
	if errors.Unwrap(wrappedErr) != testErr {
		fmt.Println("   ❌ 错误链验证失败")
		os.Exit(1)
	}
	fmt.Println("   ✅ 错误链保留正确")

	// 3. 验证 HTTP 客户端工厂
	fmt.Println("\n3. 验证 HTTP 客户端工厂...")
	client := httpclient.NewDefaultClient()
	if client == nil {
		fmt.Println("   ❌ HTTP 客户端创建失败")
		os.Exit(1)
	}
	fmt.Println("   ✅ HTTP 客户端创建成功")
	fmt.Printf("   - Timeout: %v\n", client.Timeout)

	// 4. 验证日志系统
	fmt.Println("\n4. 验证日志系统...")
	logger, err := logging.SetupLogger(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		fmt.Printf("   ❌ 日志系统初始化失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   ✅ 日志系统初始化成功")

	// 添加敏感信息钩子
	logger.AddHook(&logging.SensitiveHook{})

	// 测试日志脱敏
	testData := "api_key=AKIAIOSFODNN7EXAMPLEKEY123"
	redacted := logging.RedactSensitiveData(testData)
	if redacted == testData {
		fmt.Println("   ❌ 敏感数据脱敏失败")
		os.Exit(1)
	}
	fmt.Println("   ✅ 敏感数据脱敏正常")
	fmt.Printf("   - 原始: %s\n", testData)
	fmt.Printf("   - 脱敏: %s\n", redacted)

	// 5. 验证环境变量覆盖
	fmt.Println("\n5. 验证环境变量覆盖...")
	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("LOG_LEVEL", "debug")
	cfg2, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("   ❌ 环境变量配置加载失败: %v\n", err)
		os.Exit(1)
	}
	if cfg2.Server.Port != 9999 {
		fmt.Printf("   ❌ 环境变量覆盖失败: expected port 9999, got %d\n", cfg2.Server.Port)
		os.Exit(1)
	}
	if cfg2.Logging.Level != "debug" {
		fmt.Printf("   ❌ 环境变量覆盖失败: expected log level 'debug', got '%s'\n", cfg2.Logging.Level)
		os.Exit(1)
	}
	fmt.Println("   ✅ 环境变量覆盖正常")
	fmt.Printf("   - Port: %d (from env)\n", cfg2.Server.Port)
	fmt.Printf("   - Log Level: %s (from env)\n", cfg2.Logging.Level)

	fmt.Println("\n=== ✅ Phase 1 基础设施验证通过 ===")
}
