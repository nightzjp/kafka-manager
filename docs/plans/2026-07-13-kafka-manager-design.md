# Kafka Manager 重构设计

## 1. 目标

构建一个面向研发人员的轻量 Kafka 日常管理平台，同时为运维人员提供多集群状态大屏。项目使用 Go 后端与 React + TypeScript 前端，最终发布为嵌入前端资源的单个 Go 二进制。

项目只依赖 Kafka，不接入 Kafka Connect、Schema Registry、ksqlDB、Prometheus、JMX Exporter或数据库。

## 2. 产品边界

### 核心功能

- 多 Kafka 集群管理，默认开发、测试两个集群，可继续新增。
- 集群、Broker、Topic、Partition 和 Consumer Group 状态查看。
- Topic 创建、删除、扩分区和配置修改。
- 消息查询、实时跟随、格式化展示和测试消息生产。
- Consumer Group、成员、分区分配和 Lag 查看。
- Consumer Offset 按最早、最新、指定 Offset 或时间重置。
- 首页监控大屏、异常汇总、Lag 排行和 Offset 增长趋势。
- Web 配置管理、连接测试、配置热加载、备份和回滚。
- 单用户登录、危险操作确认和可轮转审计日志。

### 首版不包含

- Kafka Connect、Schema Registry、ksqlDB。
- Kafka ACL、Quota、事务和副本迁移管理。
- CPU、内存、磁盘、JVM、网络字节吞吐等外部监控指标。
- 多用户、角色和细粒度 RBAC。
- 长期指标数据库。

## 3. 总体架构

采用模块化单体架构：

```text
浏览器
   ↓
React + TypeScript
   ↓ REST / SSE
Go 单体服务
   ├── 登录与会话
   ├── 集群配置管理
   ├── Kafka 客户端池
   ├── Topic 管理
   ├── 消息查询与生产
   ├── Consumer Group / Lag
   ├── Broker / Partition 状态
   ├── 首页采样与短期趋势
   └── 审计日志与轮转
        ↓
开发 Kafka / 测试 Kafka / 后续新增集群
```

开发时 Go 与 Vite 分别运行；发布时 React 构建产物嵌入 Go 二进制。部署只需要可执行文件、配置文件和可写数据目录。

## 4. 目录结构

```text
kafka-manager/
├── cmd/kafka-manager/       # 程序入口
├── internal/
│   ├── api/                 # HTTP 路由、校验和响应
│   ├── auth/                # 登录、会话和密码校验
│   ├── config/              # YAML、原子写入、备份和热加载
│   ├── cluster/             # 多集群客户端生命周期
│   ├── kafka/
│   │   ├── broker/
│   │   ├── topic/
│   │   ├── message/
│   │   └── consumer/
│   ├── dashboard/           # 周期采样和短期趋势
│   ├── audit/               # JSONL 审计、轮转和查询
│   └── platform/            # 文件、时间和加密基础能力
├── web/                     # React + TypeScript
├── config.example.yaml
├── docs/
└── data/                    # 运行数据，不提交
```

## 5. 配置与认证

YAML 是唯一配置源。启动时严格校验；Web 保存时先校验字段并测试 Kafka 连接，再通过临时文件、`fsync` 和原子替换写入。替换前生成备份，新客户端可用后才关闭旧客户端。

手工修改配置文件也会触发热加载。无效配置不会替换当前有效运行配置；首次启动配置无效则拒绝启动。

支持每个集群独立选择：

- 无认证连接。
- SASL/PLAIN。
- SCRAM-SHA-256、SCRAM-SHA-512。
- 可选 TLS。

平台使用单个登录用户，密码采用 Argon2id 哈希。会话 Cookie 设置 `HttpOnly` 和 `SameSite`。为保持单二进制加单配置文件的部署方式，Kafka 密码直接写入权限为 `0600` 的 YAML；配置文件不提交到 Git，API 和日志不回显密码。

## 6. 首页与前端体验

首页采用深色监控大屏，管理页采用清爽控制台布局，两者共享统一设计系统。

首页展示：

- 所有集群在线状态和探测延迟。
- Broker、Topic、Partition、Consumer Group 数量。
- 无 Leader、离线 Partition、ISR 不完整等异常。
- Consumer Lag 汇总和排行。
- 根据 Offset 增量估算的消息条数速率和短期趋势。

管理页提供全局集群选择器、统一搜索、筛选、排序、分页和详情抽屉。开发与测试环境具有明显标识。危险操作要求输入资源名称确认。

消息查询页支持指定 Topic、Partition、起始位置、Offset、时间和数量，展示 Key、Value、Headers、Timestamp 等元数据。JSON 自动格式化，非文本内容支持 Base64 和十六进制。实时跟随需要主动开启，并限制消息数量、单条大小和页面缓存。

## 7. API 与数据流

- API 前缀为 `/api/v1`。
- 普通查询使用 REST，实时消息和任务进度使用 SSE。
- 每个 Kafka 请求显式指定集群 ID。
- 后台定时采样首页数据，所有浏览器共享结果。
- 列表查询使用后端分页、超时和数量限制。
- 配置热更新以新客户端成功建立为切换条件。

## 8. 审计与日志

审计日志按日期目录和文件大小轮转：

```text
data/audit/2026-07-13/audit-001.jsonl
data/audit/2026-07-13/audit-002.jsonl
```

目录、保留天数和单文件大小可配置。服务启动时和每日定时清理过期目录。审计记录时间、用户、客户端 IP、集群、动作、资源、结果、耗时和脱敏后的参数，不记录消息正文或凭证。

配置备份写入 `data/config-backups/YYYY-MM-DD/`，支持 Web 查看与回滚。

## 9. 错误处理与安全

- 统一错误响应包含错误码、可读说明和请求 ID。
- 区分 Kafka 超时、认证失败、权限不足和集群不可达。
- 危险操作设置服务端超时、重复提交保护和审计。
- 所有权限与输入校验由后端执行。
- 日志、审计和 API 响应不泄露密码或密钥。
- 服务关闭时停止采样、刷新日志并关闭 Kafka 客户端。

## 10. 测试与验收

- Go 单元测试覆盖配置、加密、日志轮转和业务校验。
- Kafka 集成测试覆盖 Topic、消息和 Consumer Group 操作。
- 前端测试覆盖核心表单、危险确认、消息展示和错误状态。
- API 测试覆盖认证、输入校验、超时和错误映射。
- 验证 Linux/macOS 构建、前端嵌入和干净环境启动。
- 人工验证开发/测试集群以及无认证/SASL 两种连接。
- 验证大量 Topic、Partition、Consumer Group 和大消息边界。

## 11. 开发与部署

后端开发：

```bash
go run ./cmd/kafka-manager --config ./config.dev.yaml
```

前端开发：

```bash
cd web
pnpm dev
```

正式运行：

```bash
./kafka-manager --config ./config.yaml
```

最终提供 README、开发、配置、部署和安全说明等 Markdown 文档。项目作为 `kafka-manager` 独立 Git 仓库维护，便于后续迁移和开源。
