# Copilot / AI Agent 指南

目标：帮助 AI 代理快速上手本仓库，理解关键组件、运行流程与常见约定，便于安全且一致地修改代码。

- **项目简介**：这是一个用 Go 编写的礼品码/验证码处理服务，主要职责包括任务入库、验证码 OCR 调用和结果记录（见 [README.md](README.md)）。

- **主要模块/边界**:
  - `cmd/`：服务与客户端入口（server、client、verify 等）。
  - `internal/api/`：HTTP 处理器，使用 `gin` 框架，示例：`internal/api/handlers.go`（请求验证、日志、返回统一格式）。
  - `internal/service/`：业务层（例如 `GiftService`），封装核心流程并调用存储与外部服务。
  - `internal/storage/`：数据库仓库（SQLite 实现在 `internal/storage/sqlite_repository.go`），负责迁移、事务和重试。
  - `internal/config/`：配置加载与校验（`internal/config/config.go`），支持 YAML 与环境变量覆盖（`GOOGLE_CREDENTIALS_JSON`, `ACCESS_KEY` 等）。
  - `internal/svc/`：轻量的依赖注入容器（`ServiceContext`），将存储客户端注入到服务中。

- **数据流 / 运行时概览**:
  1. HTTP 请求由 `cmd/server` 启动的 Gin 路由接收，处理器在 `internal/api`。
  2. 处理器校验参数并调用 `service` 层（`internal/service`）执行业务逻辑。
  3. `service` 层使用 `internal/storage` 持久化任务/用户/礼品码记录。
  4. OCR 调用通过 `internal/captcha` 子包按配置的 providers（`ali`/`tencent`/`google`）轮询或负载均衡。

- **关键运行/开发命令**:
  - 构建（基于 `go.mod`）：
    - `go build ./cmd/server` 或使用仓库 `make`（若 Makefile 有定义）。
  - 运行：
    - `go run ./cmd/server`（建议在开发环境用 `etc/config.example.yaml` 或环境变量覆盖）。
  - 测试：
    - `go test ./...`（包内已有若干 `_test.go`，请在改动后运行以防回归）。

- **配置与环境约定**:
  - 优先级：环境变量 > `etc/config.yaml`（见 `internal/config/config.go` 的 `LoadConfig`/`overrideFromEnv` 实现）。
  - 常用 env：`SERVER_PORT`, `DATABASE_PATH`, `GOOGLE_CREDENTIALS_JSON`, `ACCESS_KEY`, `ACCESS_SECRET`, `LOG_LEVEL`, `LOG_FORMAT`。

- **项目特有约定与样式要点**:
  - 错误处理：项目使用内部错误类型（`internal/errors`）封装 DB/NotFound/Internal 错误，请保持该习惯以便一致的错误码和日志。
  - 日志：使用 `logrus`，`Handlers` 会将 `request_id` 注入日志字段以关联调用链（参见 `internal/api/handlers.go`）。
  - 数据库迁移：`internal/storage/sqlite_repository.go` 在初始化时执行迁移（`migrations/`），修改 schema 时应同时更新 migration 文件。
  - 事务：仓库提供 `WithTransaction`，在跨多表操作时必须使用以保证一致性。

- **修改/提交时的注意点（给 AI 的行为准则）**:
  - 变更数据库模式时，请同时：1) 添加 `internal/storage/migrations/` 的 up/down SQL；2) 运行并验证迁移逻辑（本地测试或 CI）。
  - 外部凭据不要硬编码到源文件；使用 `etc/config.yaml` 示例或环境变量注入。若需要示例凭据，仅在测试/示例文件中使用并标注为假数据。
  - 新增 HTTP 接口：遵循 `internal/api/handlers.go` 的风格，返回 `SuccessResponse` / `ErrorResponse`（查看 `internal/api/response.go`）。
  - 保持 `internal/` 包为不可导出对外 API（仅本仓库内部使用）；不应新增对外可复用的公开模块，除非明确需要并已讨论。

-- **示例片段**（参数校验风格）:
  - 参考 `internal/api/handlers.go::AddGiftCode`：先 `TrimSpace`，检测空字符串，日志记录 `request_id`，并返回统一的 JSON 错误结构。

-- **业务层（示例：`GiftService`）调用流程要点**:
  - 入口：`internal/service/gift_service.go::RedeemGiftCode`。校验是否已兑换（`repo.IsGiftCodeReceived`），若未兑换则调用 `getOrCreatePlayer` 获取 `PlayerGiftCode`。
  - `PlayerGiftCode`：由 `giftcode.NewPlayerGiftCode(fid, s.captchaPool.Get, s.keyStorage)` 创建，初始化需调用 `InitWithContext`。
  - OCR / 验证码：通过 `internal/captcha.CaptchaPool` 提供的 `Get` 函数获取验证码解码器（参见 `internal/captcha/*`）。配置由 `etc/config.yaml` 或环境变量控制。
  - 并发与批量：`BatchRedeemGiftCode` 使用限制并发的 semaphore（channel）和 `sync.WaitGroup`，默认 `workerPoolSize=5`。修改批量行为优先改 `JobConfig` 或直接调整调用处的参数。
  - 缓存：`GiftService` 使用 `playerCache`（`sync.Map`）缓存 `PlayerGiftCode` 实例、以及 `userCache`（`internal/cache/lru_cache.go`）缓存用户信息（TTL 10 分钟）。请在修改缓存策略时同时考虑并发安全与内存使用。
  - 错误处理策略：业务层对内部 DB 错误会记录日志并返回封装错误（见 `internal/errors`）；对兑换结果的非致命失败（如已被领取）会记录并继续处理，不总是返回错误给上层 API。

如果你希望我把 `internal/captcha` 的 provider 配置、`internal/storage/migrations/` 的迁移工作流，或把 README 的部署/计费段落合并进说明，我可以继续扩展。谢谢！
