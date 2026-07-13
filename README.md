# Kafka Manager

一个面向研发人员的轻量 Kafka 日常管理平台。后端使用 Go，前端使用 React + TypeScript；发布产物是包含前端资源的单个二进制，只依赖 Kafka。

## 功能

- 多集群状态首页：Broker、Topic、Partition、Consumer Group、ISR 异常、总 Lag 和短期 Lag 趋势。
- Topic 工作台：搜索、概览、创建、删除、扩分区和运行配置修改；进入 Topic 后可直接查看消息，不需要再次填写 Topic 名称。
- 消息：按 Partition、起始位置、Offset 或时间读取，SSE 实时跟随，查看 Key、Value、Headers，并生产测试消息；JSON 自动识别、格式化、折叠和复制，仍支持原始文本。
- Consumer Group：成员、分区 Lag、删除和 Offset 重置。
- 配置：Web 编辑、连接验证、YAML 原子写回、备份、Web 回滚和热加载。
- 界面：日间、夜间、跟随系统三种主题，主题选择自动保存在浏览器；桌面与移动端均可使用。
- 安全：单用户登录、签名 Cookie、登录限速和危险操作审计。
- 审计：按日期目录保存，按文件大小轮转并自动清理。

不包含 Kafka Connect、Schema Registry、ksqlDB、Prometheus、数据库或 JVM 指标。

## 五分钟启动

要求：Go 1.25+、Node.js 22+、pnpm 10+（Node/pnpm 仅构建前端时需要）。

```bash
pnpm --dir web install
make build

cp config.example.yaml config.yaml
./build/kafka-manager --config ./config.yaml
```

打开 `http://localhost:8080`。Kafka 地址、用户名和密码都直接配置在 `config.yaml` 中，不要求额外环境变量。

## Docker Compose

部署服务器只需要 Docker 和 Compose：

```bash
cp config.example.yaml config.yaml
chmod 600 config.yaml
mkdir -p data
docker compose up -d --build
```

打开 `http://服务器地址:8080`。Compose 会把宿主机 `config.yaml` 映射到容器，Web 页面修改配置后会直接写回该文件；审计日志和配置备份保存在宿主机 `data/`。

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
