# Kafka Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 构建一个只依赖 Kafka、面向研发日常操作、带多集群监控首页的 Go + React 单二进制管理平台。

**Architecture:** 使用 Go 模块化单体提供 REST/SSE API、配置热加载、Kafka 客户端池和本地审计；React + TypeScript 构建产物通过 `embed` 打入 Go 二进制。YAML 是配置唯一事实源，运行指标仅来自 Kafka API 并在内存中短期保存。

**最终部署调整：** 为满足内部单用户场景的最简部署要求，Web 登录凭据和 Kafka 凭据最终直接保存在权限为 `0600` 且被 Git 忽略的 `config.yaml`，运行时不再要求密码哈希或加密主密钥环境变量。下方早期任务中的哈希和密文格式记录保留为实现过程说明。

**Tech Stack:** Go 1.24+、franz-go、chi、YAML v3、Argon2id、React 18、TypeScript、Vite、TanStack Query、Vitest、Testing Library、pnpm。

---

## 实施规则

- 每项业务功能先写失败测试，再写最小实现。
- 每完成一个任务运行对应测试并提交。
- 不引入数据库、Prometheus、Kafka Connect、Schema Registry 或 ksqlDB。
- 不在日志、配置示例、测试夹具中保存真实地址和凭证。
- 每个 API 都必须包含超时、输入上限和统一错误响应。

### Task 1: 初始化 Go、前端与仓库工程

**Files:**
- Create: `.gitignore`
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `Makefile`

**Steps:**

1. 写 `internal/app/app_test.go`，断言健康接口返回 `200` 和 `{"status":"ok"}`。
2. 运行 `go test ./internal/app -run TestHealth -v`，确认因包或实现缺失而失败。
3. 初始化 `go.mod`，实现最小 HTTP App、`/api/v1/health` 和命令行入口。
4. 运行 `go test ./internal/app -v`，确认通过。
5. 初始化 Vite React TypeScript，页面只显示 `Kafka Manager`。
6. 添加 `make dev-backend`、`make dev-frontend`、`make test`、`make build`。
7. 运行 `go test ./...` 与 `pnpm --dir web build`。
8. 提交：`chore: initialize go and react application`。

### Task 2: 配置模型、校验与示例配置

**Files:**
- Create: `internal/config/model.go`
- Create: `internal/config/load.go`
- Create: `internal/config/validate.go`
- Create: `internal/config/config_test.go`
- Create: `config.example.yaml`

**Steps:**

1. 用表驱动测试覆盖：合法的无认证集群、SCRAM 集群、重复 ID、空 Broker、不支持的认证机制和非法审计参数。
2. 运行 `go test ./internal/config -v`，确认失败。
3. 实现 Server、User、Cluster、Security、Audit 和 Dashboard 配置结构、默认值、YAML 严格解析及校验。
4. 支持 `${ENV_NAME}` 环境变量引用，但禁止未解析变量通过校验。
5. 运行 `go test ./internal/config -v`，确认全部通过。
6. 添加根目录 `config.example.yaml`。
7. 提交：`feat: add validated yaml configuration`。

### Task 3: 密码、会话与登录 API

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/session.go`
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/auth_test.go`
- Create: `internal/api/auth.go`
- Create: `internal/api/auth_test.go`
- Create: `web/src/features/auth/LoginPage.tsx`
- Create: `web/src/features/auth/LoginPage.test.tsx`

**Steps:**

1. 写失败测试覆盖 Argon2id 哈希/验证、过期会话、无 Cookie 拒绝和登录失败限速。
2. 运行 `go test ./internal/auth ./internal/api -run 'TestPassword|TestSession|TestLogin' -v`。
3. 实现密码和带签名的 HttpOnly/SameSite 会话 Cookie。
4. 实现 `/api/v1/auth/login`、`logout`、`me` 以及鉴权中间件。
5. 运行后端测试，确认通过。
6. 写登录页失败测试，再实现登录、错误提示和已登录跳转。
7. 运行 `pnpm --dir web test --run`。
8. 提交：`feat: add single-user authentication`。

### Task 4: 配置加密、原子保存、备份与热加载

**Files:**
- Create: `internal/config/crypto.go`
- Create: `internal/config/store.go`
- Create: `internal/config/watcher.go`
- Create: `internal/config/store_test.go`
- Create: `internal/config/watcher_test.go`
- Create: `internal/api/config.go`
- Create: `internal/api/config_test.go`

**Steps:**

1. 写失败测试覆盖 AES-GCM 往返、错误主密钥、原子写入、日期备份、无效文件不替换当前配置以及自写配置不重复加载。
2. 运行 `go test ./internal/config -run 'TestEncrypt|TestStore|TestWatcher' -v`。
3. 实现 `enc:v1:` 密文格式、临时文件 + fsync + rename、备份与文件监听。
4. 实现配置读取、保存、备份列表与回滚 API；响应必须掩码密码。
5. 运行 `go test ./internal/config ./internal/api -v`。
6. 提交：`feat: add secure hot-reloadable configuration`。

### Task 5: Kafka 客户端池与集群探测

**Files:**
- Create: `internal/cluster/client.go`
- Create: `internal/cluster/factory.go`
- Create: `internal/cluster/manager.go`
- Create: `internal/cluster/manager_test.go`
- Create: `internal/kafka/admin.go`
- Create: `internal/api/clusters.go`
- Create: `internal/api/clusters_test.go`

**Steps:**

1. 定义可替换的 Admin 接口，并写失败测试覆盖新增、替换、删除客户端和“新连接失败时保留旧客户端”。
2. 运行 `go test ./internal/cluster -v`，确认失败。
3. 使用 franz-go 实现无认证、SASL/PLAIN、SCRAM 和 TLS 客户端工厂。
4. 实现并发安全的客户端池与优雅关闭。
5. 实现集群列表、详情、连接测试和 Broker 探测 API。
6. 运行 `go test ./internal/cluster ./internal/api -v`。
7. 提交：`feat: add multi-cluster kafka client manager`。

### Task 6: 首页采样与监控大屏

**Files:**
- Create: `internal/dashboard/model.go`
- Create: `internal/dashboard/sampler.go`
- Create: `internal/dashboard/sampler_test.go`
- Create: `internal/api/dashboard.go`
- Create: `web/src/features/dashboard/DashboardPage.tsx`
- Create: `web/src/features/dashboard/DashboardPage.test.tsx`
- Create: `web/src/features/dashboard/dashboard.css`

**Steps:**

1. 写失败测试覆盖集群状态、Broker/Topic/Partition/Group 数量、无 Leader/ISR 异常、Lag 排行、Offset 增量速率和固定长度内存序列。
2. 运行 `go test ./internal/dashboard -v`。
3. 实现共享后台采样器，确保单次慢集群不会阻塞其他集群。
4. 实现 `/api/v1/dashboard/summary` 和趋势 API。
5. 写前端失败测试覆盖在线/离线、异常卡片、Lag 排行、加载和错误状态。
6. 实现深色首页、自动刷新、全屏与集群跳转。
7. 运行 `go test ./internal/dashboard ./internal/api -v` 与 `pnpm --dir web test --run`。
8. 提交：`feat: add multi-cluster monitoring dashboard`。

### Task 7: Topic 与 Partition 管理

**Files:**
- Create: `internal/kafka/topic/service.go`
- Create: `internal/kafka/topic/service_test.go`
- Create: `internal/api/topics.go`
- Create: `internal/api/topics_test.go`
- Create: `web/src/features/topics/TopicsPage.tsx`
- Create: `web/src/features/topics/TopicDetails.tsx`
- Create: `web/src/features/topics/TopicForm.tsx`
- Create: `web/src/features/topics/TopicsPage.test.tsx`

**Steps:**

1. 写失败测试覆盖 Topic 分页/搜索、Partition/Leader/Replica/ISR、配置读取、创建、删除、扩分区和配置修改。
2. 对删除、扩分区和配置修改增加参数及超时测试。
3. 实现 Topic 服务与 REST API。
4. 运行 `go test ./internal/kafka/topic ./internal/api -run Topic -v`。
5. 写前端失败测试覆盖表格、筛选、详情抽屉、创建表单和输入 Topic 名称的删除确认。
6. 实现 Topic 页面并运行前端测试。
7. 提交：`feat: add topic and partition management`。

### Task 8: 消息查询、实时跟随与生产

**Files:**
- Create: `internal/kafka/message/service.go`
- Create: `internal/kafka/message/format.go`
- Create: `internal/kafka/message/service_test.go`
- Create: `internal/api/messages.go`
- Create: `internal/api/messages_test.go`
- Create: `web/src/features/messages/MessagesPage.tsx`
- Create: `web/src/features/messages/MessageDetails.tsx`
- Create: `web/src/features/messages/ProduceMessageForm.tsx`
- Create: `web/src/features/messages/MessagesPage.test.tsx`

**Steps:**

1. 写失败测试覆盖 earliest/latest/offset/time 查询、数量上限、消息大小上限、取消消费、Key/Header 和生产分区。
2. 实现有界批量消费与生产服务。
3. 实现 REST 查询/生产 API 和 SSE 实时跟随；客户端断开必须立即取消 Kafka 消费。
4. 运行消息服务和 API 测试。
5. 写前端失败测试覆盖 JSON、文本、Base64、十六进制、查询条件、实时停止和生产表单。
6. 实现三栏消息页和有界前端缓存。
7. 运行全部相关测试。
8. 提交：`feat: add bounded message browser and producer`。

### Task 9: Consumer Group、Lag 与 Offset 重置

**Files:**
- Create: `internal/kafka/consumer/service.go`
- Create: `internal/kafka/consumer/reset.go`
- Create: `internal/kafka/consumer/service_test.go`
- Create: `internal/api/consumer_groups.go`
- Create: `internal/api/consumer_groups_test.go`
- Create: `web/src/features/consumers/ConsumerGroupsPage.tsx`
- Create: `web/src/features/consumers/ConsumerGroupDetails.tsx`
- Create: `web/src/features/consumers/ResetOffsetsDialog.tsx`
- Create: `web/src/features/consumers/ConsumerGroupsPage.test.tsx`

**Steps:**

1. 写失败测试覆盖 Group 列表/状态/成员/分配、Current/Latest Offset、Lag 和分页。
2. 写重置测试覆盖 earliest/latest/absolute/timestamp、活动 Group 拒绝、越界 Offset 和部分失败。
3. 实现 Consumer 服务和 API。
4. 运行后端相关测试。
5. 写前端失败测试覆盖 Lag 排行、分区详情、重置模式和危险确认。
6. 实现 Consumer 页面并运行前端测试。
7. 提交：`feat: add consumer group and offset management`。

### Task 10: Web 集群配置与回滚

**Files:**
- Create: `web/src/features/settings/ClustersSettingsPage.tsx`
- Create: `web/src/features/settings/ClusterForm.tsx`
- Create: `web/src/features/settings/BackupsPage.tsx`
- Create: `web/src/features/settings/ClustersSettingsPage.test.tsx`

**Steps:**

1. 写失败测试覆盖新增/编辑/禁用/删除集群、认证字段联动、密码不回显、留空保持、连接测试、保存和回滚。
2. 实现配置表单和 API 集成。
3. 对不可写配置、无主密钥保存密码、连接失败和并发更新显示明确错误。
4. 运行 `pnpm --dir web test --run` 和配置 API 后端测试。
5. 提交：`feat: add web-based cluster configuration`。

### Task 11: 审计写入、轮转、清理与查询

**Files:**
- Create: `internal/audit/model.go`
- Create: `internal/audit/writer.go`
- Create: `internal/audit/cleanup.go`
- Create: `internal/audit/query.go`
- Create: `internal/audit/audit_test.go`
- Create: `internal/api/audit.go`
- Create: `web/src/features/audit/AuditPage.tsx`
- Create: `web/src/features/audit/AuditPage.test.tsx`

**Steps:**

1. 写失败测试覆盖日期目录、文件大小轮转、跨日、并发写入、保留期清理和损坏行跳过。
2. 实现 JSONL Writer、定时清理和按日期/集群/动作/结果查询。
3. 为所有写操作加入审计中间件，测试密码、密钥和消息正文不会落盘。
4. 实现审计 API 与查询页面。
5. 运行后端和前端审计测试。
6. 提交：`feat: add rotating audit log and viewer`。

### Task 12: 应用框架、导航与设计系统

**Files:**
- Create: `web/src/app/router.tsx`
- Create: `web/src/app/queryClient.ts`
- Create: `web/src/components/AppShell.tsx`
- Create: `web/src/components/ClusterSelector.tsx`
- Create: `web/src/components/ErrorState.tsx`
- Create: `web/src/components/ConfirmDangerousAction.tsx`
- Create: `web/src/styles/tokens.css`
- Create: `web/src/styles/global.css`
- Create: `web/src/components/AppShell.test.tsx`

**Steps:**

1. 写失败测试覆盖鉴权路由、全局集群选择、环境标识、主题切换、错误详情和危险确认。
2. 实现统一 AppShell、导航、设计 Token、浅/深主题和响应式布局。
3. 将所有功能页接入统一路由和 Query Client。
4. 运行前端测试、类型检查和构建。
5. 提交：`feat: unify application shell and design system`。

### Task 13: 前端嵌入、优雅关闭与发布构建

**Files:**
- Create: `internal/webassets/embed.go`
- Create: `internal/webassets/handler.go`
- Create: `internal/webassets/handler_test.go`
- Modify: `main.go`
- Modify: `Makefile`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`

**Steps:**

1. 写失败测试覆盖 SPA fallback、静态缓存、API 不回退到 HTML 和缺失资源。
2. 实现构建产物嵌入与静态服务。
3. 实现 SIGINT/SIGTERM 优雅关闭：停止 HTTP、采样器、文件监听、审计并关闭 Kafka 客户端。
4. 添加可复现构建命令，注入版本、提交号和构建时间。
5. 添加 Linux amd64/arm64、macOS amd64/arm64 CI 构建与测试。
6. 运行 `make test`、`make build`，启动二进制并验证 `/api/v1/health` 和前端首页。
7. 提交：`build: add embedded frontend and release pipeline`。

### Task 14: 文档、开源准备与最终验收

**Files:**
- Create: `README.md`
- Create: `docs/development.md`
- Create: `docs/configuration.md`
- Create: `docs/deployment.md`
- Create: `docs/security.md`
- Create: `CONTRIBUTING.md`
- Create: `SECURITY.md`
- Create: `LICENSE`
- Create: `CHANGELOG.md`

**Steps:**

1. 编写五分钟快速启动、`go run` 调试、前端调试、配置字段、密码主密钥、升级/回滚、日志清理和故障排查文档。
2. 添加 Apache-2.0 许可证、贡献指南、安全漏洞报告流程和初始变更日志。
3. 搜索仓库，确认无内部域名、IP、用户名、密码和绝对路径。
4. 运行 `go test ./...`、`go vet ./...`、前端测试、lint、类型检查和生产构建。
5. 使用示例配置启动二进制，验证无 Kafka 时能展示明确离线状态而非崩溃。
6. 使用无认证 Kafka 和 SCRAM Kafka 完成人工验收清单。
7. 检查 `git status --short`，确认只包含预期文件。
8. 提交：`docs: prepare kafka manager for release`。

## 最终验收标准

- `go run . --config ./config.dev.yaml` 可用于后端调试。
- `pnpm --dir web dev` 可用于前端热更新调试。
- `make build` 生成包含前端的单个 Go 二进制。
- `./kafka-manager --config ./config.yaml` 是正式部署唯一必需命令。
- 不需要 Java、Node.js、数据库或 Kafka 以外的运行时服务。
- Web 可管理多个 Kafka 集群并安全更新 YAML。
- Topic、消息、Consumer Group 和基础集群监控功能通过测试。
- 审计按日期目录和大小轮转，保留期可配置。
- README 和部署文档足以让新用户独立安装运行。
