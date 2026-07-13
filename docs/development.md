# 开发说明

## 后端

```bash
CONFIG=./config.dev.yaml make dev-backend
```

等价命令：

```bash
go run ./cmd/kafka-manager --config ./config.dev.yaml
```

## 前端

```bash
pnpm --dir web install
pnpm --dir web dev
```

前端开发服务器默认代理 `/api` 至 `http://localhost:8080`。

## 测试

```bash
go test ./...
go vet ./...
pnpm --dir web test --run
pnpm --dir web build
```

业务改动遵循测试先行：先增加失败测试，确认失败原因，再实现最小代码并运行全部回归测试。

## 目录

- `cmd/kafka-manager`：启动、信号和生命周期。
- `internal/api`：REST API 与认证边界。
- `internal/cluster`：franz-go 客户端池。
- `internal/kafka`：Topic、消息和 Consumer Group 服务。
- `internal/config`：YAML、原子保存、备份和热加载。
- `internal/audit`：JSONL 轮转、清理和查询。
- `internal/webassets`：嵌入前端资源。
- `web`：React + TypeScript 控制台。
