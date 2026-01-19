package main

import (
	"cdk-get/internal/api"
	"cdk-get/internal/auth"
	"cdk-get/internal/config"
	"cdk-get/internal/job"
	"cdk-get/internal/logging"
	"cdk-get/internal/notification"
	"cdk-get/internal/service"
	"cdk-get/internal/storage"
	"cdk-get/internal/svc"
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//go:embed static
var staticFS embed.FS

func main() {
	printBuildInfo()

	// 加载配置
	cfg, err := config.LoadConfig("./etc/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err := logging.SetupLogger(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		fmt.Printf("Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	// 添加敏感数据脱敏钩子
	logger.AddHook(&logging.SensitiveHook{})

	// 初始化数据库 - 使用新的Repository接口
	repoConfig := storage.SqliteConfig{
		Path:            cfg.Database.Path,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
	}

	repository, err := storage.NewSqliteRepository(repoConfig, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize repository: %v", err)
	}

	// 初始化认证服务
	authService := auth.NewAuthService(
		cfg.Admin.Username,
		cfg.Admin.PasswordHash,
		cfg.Admin.TokenSecret,
		cfg.Admin.TokenDuration,
	)

	// 初始化通知服务
	var notificationService *service.NotificationService
	if cfg.Notification.WxPusher.AppToken != "" && cfg.Notification.WxPusher.UID != "" {
		// 创建WxPusher通知器
		wxpusherNotifier := notification.NewWxPusherNotifier(
			cfg.Notification.WxPusher.AppToken,
			cfg.Notification.WxPusher.UID,
			logger,
		)

		// 创建通知服务
		notificationService = service.NewNotificationService(
			wxpusherNotifier,
			repository,
			logger,
		)

		logger.Info("Notification service initialized with WxPusher")
	} else {
		logger.Warn("Notification service not initialized: WxPusher configuration missing")
	}

	// 初始化API处理器 (暂时不使用GiftService，因为Repository接口还未完全实现)
	handlers := api.NewHandlers(nil, repository, logger)

	// 初始化管理后台处理器
	adminHandlers := api.NewAdminHandlers(authService, repository, logger)

	// 初始化任务调度器（保持向后兼容）
	svcCtx := svc.NewServiceContext(repository, repository, notificationService)
	_ = job.InitTask(svcCtx)

	// 创建服务器
	server := setupServer(cfg, handlers, adminHandlers, authService, logger)

	// 启动服务器
	go func() {
		logger.Infof("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅关闭
	gracefulShutdown(server, repository, logger)
}

// setupServer 设置服务器和路由
func setupServer(cfg *config.Config, handlers *api.Handlers, adminHandlers *api.AdminHandlers, authService auth.AuthService, logger *logrus.Logger) *http.Server {
	// 设置Gin模式
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 添加中间件
	engine.Use(api.RequestIDMiddleware())
	engine.Use(api.RecoveryMiddleware(logger))
	engine.Use(api.LoggerMiddleware(logger))
	engine.Use(api.ValidationMiddleware())

	// 添加CORS中间件（如果启用）
	if cfg.Server.CORS.Enabled {
		corsConfig := cors.Config{
			AllowOrigins:     cfg.Server.CORS.AllowOrigins,
			AllowMethods:     cfg.Server.CORS.AllowMethods,
			AllowHeaders:     cfg.Server.CORS.AllowHeaders,
			ExposeHeaders:    cfg.Server.CORS.ExposeHeaders,
			AllowCredentials: cfg.Server.CORS.AllowCredentials,
			MaxAge:           time.Duration(cfg.Server.CORS.MaxAge) * time.Second,
		}
		engine.Use(cors.New(corsConfig))
	}

	// 添加限流中间件（如果启用）
	if cfg.Security.RateLimit.Enabled {
		engine.Use(api.RateLimitMiddleware(cfg.Security.RateLimit.Rate, cfg.Security.RateLimit.Burst))
	}

	// 注册管理后台API路由（必须在catch-all路由之前）
	adminAPI := engine.Group("/api/admin")
	{
		// 公开路由 - 登录
		adminAPI.POST("/login", adminHandlers.Login)

		// 创建认证服务适配器用于中间件
		authAdapter := api.NewAuthServiceAdapter(authService)

		// 受保护的路由 - 需要认证
		protected := adminAPI.Group("")
		protected.Use(api.AuthMiddleware(authAdapter, logger))
		{
			// 用户管理
			protected.GET("/users", adminHandlers.ListUsers)
			protected.POST("/users", adminHandlers.AddUser)
			protected.GET("/users/:fid/codes", adminHandlers.GetUserGiftCodes)

			// 任务管理
			protected.GET("/tasks", adminHandlers.ListTasks)
			protected.POST("/tasks", adminHandlers.AddGiftCode)
			protected.GET("/tasks/completed", adminHandlers.ListCompletedTasks)
			protected.DELETE("/tasks/:code", adminHandlers.DeleteTask)

			// 通知管理
			protected.GET("/notifications", adminHandlers.ListNotifications)
		}
	}

	// 注册现有路由
	engine.POST("/giftcode", handlers.AddGiftCode)
	engine.POST("/add_user", handlers.AddUser)
	engine.GET("/ip", handlers.GetIP)

	// 管理后台静态文件路由
	// 处理 /admin 和 /admin/ 重定向
	engine.GET("/admin", func(c *gin.Context) {
		// 重定向到登录页（前端会检查token并决定是否跳转到dashboard）
		c.Redirect(http.StatusFound, "/admin/login.html")
	})

	// 提供管理后台静态文件 - 使用具体的文件路由
	engine.GET("/admin/login.html", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(300))
		c.FileFromFS("static/admin/login.html", http.FS(staticFS))
	})
	engine.GET("/admin/dashboard.html", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(300))
		c.FileFromFS("static/admin/dashboard.html", http.FS(staticFS))
	})
	engine.GET("/admin/dashboard.js", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(300))
		c.Writer.Header().Set("Content-Type", "application/javascript")
		c.FileFromFS("static/admin/dashboard.js", http.FS(staticFS))
	})
	engine.GET("/admin/styles.css", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(300))
		c.Writer.Header().Set("Content-Type", "text/css")
		c.FileFromFS("static/admin/styles.css", http.FS(staticFS))
	})

	// 静态文件路由（必须最后注册，因为是catch-all）
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "" || path == "/" {
			c.Redirect(http.StatusFound, "/admin/login.html")
		} else {
			c.Writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(300))
			c.FileFromFS(filepath.Join("./static", path), http.FS(staticFS))
		}
	})

	// 创建HTTP服务器
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// gracefulShutdown 优雅关闭服务器
func gracefulShutdown(server *http.Server, repository storage.Repository, logger *logrus.Logger) {
	// 监听中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	// 关闭数据库连接
	if err := repository.Close(); err != nil {
		logger.Errorf("Failed to close repository: %v", err)
	}

	logger.Info("Server exited")
}

func printBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("Failed to read build info")
		return
	}

	fmt.Println("\n=== Build Info ===")
	fmt.Printf("Go Version: %s\n", info.GoVersion)
	fmt.Printf("Main Module: %s\n", info.Main.Path)
	fmt.Printf("Main Version: %s\n", info.Main.Version)

	// 查找 VCS 信息
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			fmt.Printf("Git Commit: %s\n", setting.Value)
		case "vcs.time":
			fmt.Printf("Build Time: %s\n", setting.Value)
		case "vcs.modified":
			if setting.Value == "true" {
				fmt.Println("Git Status: dirty (modified)")
			} else {
				fmt.Println("Git Status: clean")
			}
		}
	}
}
