# 架构说明

## 目标与边界

Kafka Manager 是面向研发人员的轻量 Kafka 日常管理平台，同时为运维人员提供多集群状态概览。后端使用 Go，前端使用 React + TypeScript，发布时将前端资源嵌入单个 Go 二进制。

项目只依赖 Kafka Broker，不接入 Kafka Connect、Schema Registry、ksqlDB、Prometheus、JMX Exporter 或数据库。当前不提供多用户 RBAC、Kafka ACL、Quota、事务和副本迁移管理，也不采集 CPU、内存、磁盘或 JVM 等外部监控指标。

## 总体架构

项目采用模块化单体架构：

```text
浏览器
   │
   ├── REST：查询、管理和配置操作
   └── SSE：实时消息
          │
React + TypeScript 前端
          │
Go 单体服务
   ├── 登录与会话
   ├── 集群配置、备份与热加载
   ├── Kafka 客户端池
   ├── Topic、消息和 Consumer Group 管理
   ├── 首页状态采样与短期趋势
   └── 审计日志、轮转与清理
          │
多个 Kafka 集群
```

开发时 Go 与 Vite 分别运行；正式构建时，Vite 产物嵌入 Go 二进制。部署只需要二进制、`config.yaml` 和可写的 `data/` 目录，或者直接使用 Docker Compose。

## 目录职责

```text
kafka-manager/
├── main.go                  # 程序入口
├── internal/
│   ├── api/                 # HTTP 路由、输入校验和响应
│   ├── app/                 # 服务装配与生命周期管理
│   ├── auth/                # 登录、会话和密码校验
│   ├── config/              # YAML、原子写入、备份和热加载
│   ├── cluster/             # 多集群客户端生命周期
│   ├── kafka/               # Broker、Topic、消息和 Consumer Group
│   ├── dashboard/           # 周期采样和短期趋势
│   ├── audit/               # JSONL 审计、轮转和清理
│   └── webassets/           # 嵌入的前端构建产物
├── web/                     # React + TypeScript 前端
├── design-system/           # 前端视觉与交互规范
├── docs/                    # 面向使用和维护的正式文档
├── config.example.yaml      # 无真实凭据的配置模板
└── data/                    # 本地运行数据，不提交 Git
```

Go 的 `internal` 目录用于限制内部包只能由本项目引用，避免尚未稳定的实现被外部项目误当成公共 SDK。

## 配置和集群生命周期

YAML 是唯一配置源。服务启动时校验配置并为每个集群建立 Kafka 客户端。集群可分别使用无认证、SASL/PLAIN、SCRAM-SHA-256、SCRAM-SHA-512 和可选 TLS。

Web 保存配置时会先校验字段和测试连接，然后通过临时文件、同步落盘和原子替换更新 `config.yaml`。旧配置会写入 `data/config-backups/YYYY-MM-DD/`；新客户端建立成功后才替换旧客户端。手工修改配置也会触发热加载，无效配置不会替换当前有效配置。

`config.yaml` 包含 Web 登录信息和 Kafka 凭据，因此不会提交 Git。示例配置只使用虚假占位值。集群设置 `readOnly: true` 后，后端会拒绝该集群的所有写操作，前端同时禁用相应入口。

## 核心数据流

- 所有业务 API 使用 `/api/v1` 前缀，并显式携带集群 ID。
- Topic、Partition、Consumer Group 和状态查询通过 REST 完成。
- 实时消息通过 SSE 推送，普通消息查询支持 Partition、Offset、时间、Key、Value 和 JSON 字段过滤。
- 首页由后端定时采样，各浏览器共享内存中的短期历史结果，不引入指标数据库。
- Kafka 请求具有超时和结果数量限制，错误会区分认证失败、权限不足、集群不可达和请求超时。

## 安全与本地数据

平台使用单个本地登录用户。会话 Cookie 使用签名并设置 `HttpOnly` 和 `SameSite`；登录接口具有限速。Kafka 凭据不会出现在 API 响应、应用日志或审计参数中。

危险写操作由后端再次校验，并记录操作者、客户端地址、集群、资源、结果和耗时。审计日志不记录消息正文或凭据，按日期目录和文件大小轮转：

```text
data/audit/2026-07-13/audit-001.jsonl
data/audit/2026-07-13/audit-002.jsonl
```

服务启动时和每日定时任务会清理超过保留期的审计目录与配置备份。Docker Compose 另外限制容器标准输出日志的文件大小和保留数量。

## 前端结构与体验

前端围绕“选择集群 → 查看资源 → 进入详情 → 执行操作”的路径组织。Topic 详情内直接包含消息浏览，Consumer Group 详情集中展示成员、分区 Lag 和 Offset 操作。

界面支持浅色、深色和跟随系统三种主题；桌面侧边栏状态与主题选择保存在浏览器。JSON 消息自动格式化并保留原始文本视图。视觉和交互约束记录在 `design-system/kafka-manager/MASTER.md`。

## 构建与验证

`make build` 先构建前端，再将产物嵌入 Go 二进制。`make test` 运行 Go 单元测试、静态检查和前端组件测试；`make test-e2e` 使用 Playwright 验证登录、导航、Topic 消息和 Consumer Group 等关键流程。

具体命令和环境要求参见[开发说明](development.md)，配置字段参见[配置说明](configuration.md)，生产部署参见[部署与升级](deployment.md)。
