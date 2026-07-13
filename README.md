# Kafka Manager

一个面向研发人员的轻量 Kafka 日常管理平台。后端使用 Go，前端使用 React + TypeScript；发布产物是包含前端资源的单个二进制，只依赖 Kafka。

## 功能

- 多集群状态首页：Broker、Topic、Partition、Consumer Group、ISR 异常、总 Lag 和短期 Lag 趋势。
- Topic：搜索、详情、创建、删除、扩分区和运行配置修改。
- 消息：按 Partition、起始位置、Offset 或时间读取，SSE 实时跟随，查看 Key、Value、Headers，并生产测试消息。
- Consumer Group：成员、分区 Lag、删除和 Offset 重置。
- 配置：Web 编辑、连接验证、密码加密、YAML 原子写回、备份、Web 回滚和热加载。
- 安全：单用户登录、Argon2id、签名 Cookie、登录限速和危险操作审计。
- 审计：按日期目录保存，按文件大小轮转并自动清理。

不包含 Kafka Connect、Schema Registry、ksqlDB、Prometheus、数据库或 JVM 指标。

## 五分钟启动

要求：Go 1.25+、Node.js 22+、pnpm 10+（Node/pnpm 仅构建前端时需要）。

```bash
pnpm --dir web install
make build

export KAFKA_MANAGER_PASSWORD='change-me'
./build/kafka-manager --print-password-hash
```

复制示例配置并把生成的哈希写入 `server.passwordHash`：

```bash
cp configs/config.example.yaml config.yaml
export KAFKA_MANAGER_SECRET_KEY='replace-with-at-least-32-random-bytes'
./build/kafka-manager --config ./config.yaml
```

打开 `http://localhost:8080`。正式环境应把主密钥放在进程环境或操作系统 Secret 中。

## 开发

```bash
# 终端 1
CONFIG=./config.dev.yaml make dev-backend

# 终端 2
make dev-frontend
```

Vite 会把 `/api` 代理至 `localhost:8080`。测试与构建：

```bash
make test
make build
```

## 文档

- [配置说明](docs/configuration.md)
- [部署与升级](docs/deployment.md)
- [开发说明](docs/development.md)
- [安全说明](docs/security.md)
- [架构设计](docs/plans/2026-07-13-kafka-manager-design.md)

## License

Apache License 2.0
