# Docker 部署设计

## 目标

为 Kafka Manager 提供无需在部署服务器安装 Go、Node.js 或 pnpm 的容器化部署方式。使用者只需准备 `config.yaml`，执行 `docker compose up -d --build`。

## 镜像

Dockerfile 使用三个阶段：Node 22 Alpine 安装锁定依赖并构建 React；Go 1.25 Alpine 下载模块并构建静态二进制；最终 Alpine 镜像只保留 CA 证书、时区数据和应用二进制。容器使用非 root 用户运行。

## Compose

Compose 构建当前目录镜像，将宿主机 `./config.yaml` 只读映射到 `/app/config.yaml`，将 `./data` 可写映射到 `/app/data`，映射端口 `8080:8080`，设置 `restart: unless-stopped`，并通过公开健康接口执行健康检查。

配置监听器继续监控 `/app/config.yaml`。宿主机直接编辑并原子替换配置文件时，容器会热加载；Web 保存配置需要写权限，因此 Compose 中不能只读挂载配置文件。考虑到 Web 配置管理是现有功能，最终采用可读写单文件挂载，并在文档中要求宿主机将文件权限设为 `0600`。

## 数据与安全

审计日志和配置备份持久化到 `./data`。`config.yaml`、`data/`、镜像构建产物不提交 Git。健康检查不包含 Kafka 在线状态，Kafka 可用性由首页展示。

## 验证

验证 Dockerfile 多阶段依赖复制、Compose 配置解析、Go/前端全量测试、镜像构建（若本机 Docker 可用）以及挂载配置启动后的健康接口。
