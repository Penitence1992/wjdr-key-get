# Implementation Plan: Code Quality Improvements

## Overview

本实施计划将现有礼品码兑换系统的代码质量改进分解为可执行的任务。改进采用渐进式重构策略，分 6 个阶段进行，每个阶段都确保系统保持可运行状态。任务按照依赖关系组织，优先建立基础设施，然后逐步重构各层。

## Tasks

- [ ] 1. Phase 1: Foundation - 配置和错误处理基础设施
- [x] 1.1 创建配置管理包
  - 在 `internal/config` 创建配置结构体和加载逻辑
  - 实现从 YAML 文件和环境变量加载配置
  - 实现配置验证逻辑（必填字段、类型检查、范围验证）
  - 实现环境变量覆盖文件配置的优先级逻辑
  - 为可选参数提供默认值
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ]* 1.2 为配置管理编写属性测试
  - **Property 18: Configuration Precedence Correctness**
  - **Property 19: Configuration Validation Completeness**
  - **Property 20: Default Configuration Values**
  - _Requirements: 7.2, 7.3, 7.4, 7.5_

- [x] 1.3 创建错误处理包
  - 在 `internal/errors` 创建 AppError 结构体
  - 实现错误代码常量（DATABASE_ERROR, VALIDATION_ERROR 等）
  - 实现错误构造函数（NewDatabaseError, NewValidationError 等）
  - 实现错误包装函数，保留错误链
  - _Requirements: 2.1, 2.2_

- [ ]* 1.4 为错误处理编写属性测试
  - **Property 6: Structured Error Format**
  - **Property 7: Error Context Preservation**
  - _Requirements: 2.1, 2.2_

- [x] 1.5 重构 HTTP 客户端工厂
  - 在 `internal/httpclient` 创建 ClientFactory
  - 实现可配置的超时、连接池参数
  - 确保连接复用和连接池限制
  - _Requirements: 5.3, 11.3_

- [ ]* 1.6 为 HTTP 客户端编写属性测试
  - **Property 12: HTTP Response Body Closure**
  - _Requirements: 3.2_

- [x] 1.7 改进日志配置
  - 配置 logrus 使用结构化 JSON 格式
  - 添加全局字段（service_name, version）
  - 实现敏感数据脱敏函数
  - 配置日志级别（从配置文件读取）
  - _Requirements: 8.1, 8.2, 8.3, 1.3, 8.5_

- [ ]* 1.8 为日志脱敏编写属性测试
  - **Property 3: Sensitive Data Redaction in Logs**
  - **Property 21: Structured Logging Consistency**
  - _Requirements: 1.3, 8.1, 8.2, 8.5_

- [x] 1.9 Checkpoint - 验证基础设施
  - 确保所有测试通过
  - 验证配置可以从文件和环境变量加载
  - 验证错误包装保留错误链
  - 如有问题，询问用户

- [-] 2. Phase 2: Storage Layer - 数据访问层重构
- [x] 2.1 定义 Repository 接口
  - 在 `internal/storage` 创建 Repository 接口
  - 定义所有数据访问方法（SaveGiftCode, GetUser, CreateTask 等）
  - 定义 User 和 Task 模型结构体
  - 添加 WithTransaction 方法支持事务
  - 添加 Ping 和 Close 方法
  - _Requirements: 5.4, 10.2_

- [x] 2.2 重构 SQLite 实现
  - 重构 `internal/storage/sqlite.go` 实现 Repository 接口
  - 使用 context.Context 支持取消和超时
  - 所有查询使用 prepared statements
  - 实现连接池配置（MaxOpenConns, MaxIdleConns, ConnMaxLifetime）
  - 添加连接重试逻辑（指数退避）
  - _Requirements: 3.1, 10.1, 10.5_

- [ ]* 2.3 为数据库操作编写属性测试
  - **Property 11: Connection Pool Limit Enforcement**
  - **Property 26: SQL Injection Prevention**
  - **Property 28: Connection Retry with Exponential Backoff**
  - _Requirements: 3.1, 10.1, 10.5_

- [x] 2.4 实现事务支持
  - 在 SqliteStorage 实现 WithTransaction 方法
  - 使用 sql.Tx 包装事务操作
  - 确保事务回滚和提交的正确性
  - _Requirements: 10.2_

- [ ]* 2.5 为事务编写属性测试
  - **Property 27: Transaction Atomicity**
  - _Requirements: 10.2_

- [x] 2.6 改进数据库 schema
  - 创建 migrations 目录和迁移脚本
  - 添加 created_at, updated_at 时间戳字段
  - 添加索引优化查询性能
  - 添加 status 和 message 字段到 gift_code_records
  - _Requirements: 10.3, 10.4_

- [x] 2.7 实现数据库迁移工具
  - 集成 golang-migrate 或编写简单的迁移工具
  - 在系统启动时自动运行迁移
  - _Requirements: 10.4_

- [ ]* 2.8 为数据库操作编写单元测试
  - 使用 in-memory SQLite 测试所有 Repository 方法
  - 测试错误场景（连接失败、约束违反等）
  - _Requirements: 6.5_

- [x] 2.9 Checkpoint - 验证存储层
  - 确保所有测试通过
  - 验证事务原子性
  - 验证连接池限制
  - 如有问题，询问用户

- [-] 3. Phase 3: Business Logic - 业务逻辑层重构
- [x] 3.1 创建验证码客户端池
  - 在 `internal/captcha` 创建 CaptchaPool
  - 使用 atomic.Uint32 实现无锁轮询
  - 从配置加载多个验证码提供商
  - _Requirements: 11.5_

- [x] 3.2 创建 GiftService
  - 在 `internal/service` 创建 GiftService
  - 实现 RedeemGiftCode 方法（单个兑换）
  - 实现 BatchRedeemGiftCode 方法（批量兑换）
  - 使用 Repository 接口而非直接访问数据库
  - 使用 CaptchaPool 获取验证码客户端
  - 添加完整的错误处理和日志
  - _Requirements: 5.5, 2.3, 8.4_

- [ ]* 3.3 为 GiftService 编写单元测试
  - 使用 mock Repository 和 CaptchaClient
  - 测试成功兑换场景
  - 测试各种错误场景（验证码失败、已兑换、不存在等）
  - _Requirements: 6.1_

- [x] 3.4 实现 Worker Pool 并发控制
  - 在 GiftService 的 BatchRedeemGiftCode 中实现 worker pool
  - 使用 semaphore 或 channel 限制并发数
  - 从配置读取 worker pool 大小
  - _Requirements: 11.1_

- [ ]* 3.5 为并发控制编写属性测试
  - **Property 29: Worker Pool Concurrency Limit**
  - _Requirements: 11.1_

- [x] 3.6 重构 PlayerGiftCode
  - 重构 `internal/giftcode/client.go` 使用新的错误处理
  - 添加 context 支持
  - 改进日志（添加 correlation ID）
  - 使用 HTTP 客户端工厂
  - _Requirements: 3.4, 8.4_

- [ ]* 3.7 为 context 取消编写属性测试
  - **Property 13: Context Cancellation Propagation**
  - _Requirements: 3.4_

- [x] 3.8 重构任务调度器
  - 重构 `internal/job` 使用新的 Scheduler 接口
  - 实现 Job 接口
  - 添加 context 支持优雅关闭
  - 确保 goroutine 可以被正确终止
  - 修复 retry.go 中的死锁问题
  - _Requirements: 3.5, 4.5_

- [ ]* 3.9 为调度器编写单元测试
  - 测试任务调度和执行
  - 测试优雅关闭
  - 测试 goroutine 终止
  - _Requirements: 3.5_

- [x] 3.10 实现用户信息缓存
  - 在 GiftService 中添加 LRU 缓存
  - 缓存用户信息（10 分钟 TTL）
  - 使用 sync.Map 或带锁的 map
  - _Requirements: 11.2_

- [ ] 3.11 Checkpoint - 验证业务逻辑层
  - 确保所有测试通过
  - 验证并发控制正常工作
  - 验证缓存减少数据库查询
  - 如有问题，询问用户

- [x] 4. Phase 4: API Layer - API 层改进
- [x] 4.1 创建标准响应结构
  - 在 `internal/api` 创建 Response 和 ErrorInfo 结构体
  - 实现 SuccessResponse 和 ErrorResponse 辅助函数
  - _Requirements: 9.1_

- [ ]* 4.2 为 API 响应编写属性测试
  - **Property 23: API Response Format Consistency**
  - **Property 24: HTTP Status Code Correctness**
  - _Requirements: 9.1, 9.2_

- [x] 4.3 实现中间件栈
  - 实现 RequestIDMiddleware（生成唯一请求 ID）
  - 实现 LoggerMiddleware（结构化请求日志）
  - 实现 RecoveryMiddleware（panic 恢复）
  - 实现 RateLimitMiddleware（使用 golang.org/x/time/rate）
  - 实现 ValidationMiddleware（请求验证）
  - _Requirements: 1.5, 8.4_

- [ ]* 4.4 为限流中间件编写属性测试
  - **Property 5: Rate Limiting Enforcement**
  - _Requirements: 1.5_

- [x] 4.5 重构 HTTP handlers
  - 重构 `cmd/server/main.go` 中的 handlers
  - 使用 GiftService 而非直接访问存储层
  - 使用标准响应格式
  - 添加输入验证
  - 添加适当的 HTTP 状态码
  - _Requirements: 5.5, 9.1, 9.2, 1.4_

- [ ]* 4.6 为输入验证编写属性测试
  - **Property 4: Input Validation Completeness**
  - **Property 25: Content-Type Validation**
  - _Requirements: 1.4, 9.5_

- [x] 4.7 添加 CORS 支持
  - 使用 gin-contrib/cors 中间件
  - 从配置读取 CORS 设置
  - _Requirements: 9.4_

- [x] 4.8 重构服务器启动逻辑
  - 使用依赖注入初始化所有组件
  - 实现优雅关闭（监听 SIGINT/SIGTERM）
  - 在关闭时清理所有资源
  - _Requirements: 3.3, 5.2_

- [ ]* 4.9 为 API 编写集成测试
  - 使用 httptest 测试所有端点
  - 测试成功和错误场景
  - 测试中间件功能
  - _Requirements: 9.3, 9.4_

- [x] 4.10 Checkpoint - 验证 API 层
  - 确保所有测试通过
  - 手动测试 API 端点
  - 验证限流工作正常
  - 如有问题，询问用户

- [ ] 5. Phase 5: Observability - 可观测性增强
- [ ] 5.1 实现健康检查
  - 在 `internal/health` 创建 HealthChecker
  - 实现数据库健康检查
  - 添加 /health/live 端点（liveness probe）
  - 添加 /health/ready 端点（readiness probe）
  - _Requirements: 12.1_

- [ ] 5.2 实现 Prometheus 指标
  - 集成 prometheus/client_golang
  - 添加请求计数器（按端点、状态码）
  - 添加请求延迟直方图
  - 添加错误计数器
  - 添加任务执行统计
  - 添加 /metrics 端点
  - _Requirements: 12.2, 12.3, 12.5_

- [ ]* 5.3 为指标编写属性测试
  - **Property 30: Metrics Counter Monotonicity**
  - _Requirements: 12.2, 12.3_

- [ ] 5.4 改进结构化日志
  - 为所有关键操作添加结构化日志
  - 使用 correlation ID 关联请求日志
  - 添加任务生命周期日志
  - 确保敏感信息被脱敏
  - _Requirements: 8.1, 8.2, 8.4, 8.5_

- [ ]* 5.5 为日志编写属性测试
  - **Property 22: Task Correlation ID Propagation**
  - _Requirements: 8.4_

- [ ] 5.6 添加分布式追踪（可选）
  - 集成 OpenTelemetry
  - 添加 trace ID 和 span ID
  - 在日志中包含 trace 信息
  - _Requirements: 12.4_

- [ ] 5.7 Checkpoint - 验证可观测性
  - 确保所有测试通过
  - 验证健康检查端点工作
  - 验证指标正确收集
  - 验证日志包含 correlation ID
  - 如有问题，询问用户

- [ ] 6. Phase 6: Security & Final Polish - 安全性和最终完善
- [ ]* 6.1 移除所有硬编码凭证
  - 扫描代码查找硬编码的 API keys 和 tokens
  - 将所有敏感信息移到配置或环境变量
  - 更新 `internal/job/getCodeJob.go` 中的硬编码凭证
  - 更新 `internal/utls/utls.go` 中的硬编码 token
  - _Requirements: 1.1, 1.2_

- [ ]* 6.2 为凭证检查编写属性测试
  - **Property 1: No Hardcoded Credentials**
  - **Property 2: Configuration Loading from External Sources**
  - _Requirements: 1.1, 1.2_

- [ ] 6.3 添加并发安全检查
  - 使用 race detector 运行所有测试
  - 修复所有数据竞争问题
  - 确保所有共享状态有适当的同步
  - _Requirements: 4.1, 4.3_

- [ ]* 6.4 为并发安全编写属性测试
  - **Property 15: Concurrent Access Synchronization**
  - **Property 16: Concurrent Map Access Safety**
  - **Property 17: Deadlock-Free Lock Acquisition**
  - _Requirements: 4.1, 4.3, 4.4_

- [ ] 6.5 完善错误处理
  - 审查所有错误路径
  - 确保所有错误都有适当的上下文
  - 确保数据库错误包含操作类型
  - 确保没有使用 panic 处理可恢复错误
  - _Requirements: 2.3, 2.4, 2.5_

- [ ]* 6.6 为错误处理编写属性测试
  - **Property 8: Error Logging Context Completeness**
  - **Property 9: No Panic for Recoverable Errors**
  - **Property 10: Database Error Specificity**
  - _Requirements: 2.3, 2.4, 2.5_

- [ ] 6.7 完善 goroutine 管理
  - 审查所有 goroutine 创建
  - 确保所有 goroutine 都有终止机制
  - 测试优雅关闭
  - _Requirements: 3.5_

- [ ]* 6.8 为 goroutine 管理编写属性测试
  - **Property 14: Goroutine Termination Capability**
  - _Requirements: 3.5_

- [ ] 6.9 更新文档
  - 更新 README.md 包含新的配置说明
  - 添加 API 文档
  - 添加部署指南
  - 添加监控和告警指南
  - 添加故障排查指南

- [ ] 6.10 运行完整测试套件
  - 运行所有单元测试
  - 运行所有属性测试
  - 运行所有集成测试
  - 使用 race detector 运行测试
  - 生成覆盖率报告，确保达到目标
  - _Requirements: 6.4_

- [ ] 6.11 Final Checkpoint - 最终验证
  - 确保所有测试通过
  - 验证代码覆盖率达到 60%+
  - 验证没有硬编码凭证
  - 验证没有数据竞争
  - 验证所有改进需求都已实现
  - 如有问题，询问用户

## Notes

- 任务标记 `*` 的为可选任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，便于追溯
- Checkpoint 任务确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
- 集成测试验证端到端流程
- 使用 race detector 确保并发安全
- 目标代码覆盖率：核心业务逻辑 80%+，整体 60%+
