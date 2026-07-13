# Docker Deployment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 提供可通过 `docker compose up -d --build` 部署 Kafka Manager 的多阶段镜像和 Compose 配置。

**Architecture:** Node Alpine 阶段构建前端，Go Alpine 阶段复制前端产物并构建静态二进制，最终 Alpine 阶段以非 root 用户运行。Compose 映射可写 `config.yaml` 和 `data/`，提供端口、重启策略和健康检查。

**Tech Stack:** Docker 多阶段构建、Docker Compose、Node 22 Alpine、pnpm 10.15.1、Go 1.25 Alpine、Alpine Linux。

---

### Task 1: Build context and multi-stage image

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

1. 添加前端依赖缓存友好的 Node 构建阶段，使用 `pnpm install --frozen-lockfile` 和 `pnpm build`。
2. 添加 Go 模块缓存友好的构建阶段，复制前端产物到嵌入目录并执行静态构建。
3. 添加仅包含 CA、时区、非 root 用户和二进制的最终 Alpine 镜像。
4. 添加健康检查和容器入口参数。
5. 使用 `docker build` 验证镜像（Docker 可用时）。

### Task 2: Compose deployment

**Files:**
- Create: `docker-compose.yaml`

1. 配置当前目录构建、`8080:8080` 端口、`unless-stopped`。
2. 将 `./config.yaml` 可读写映射到 `/app/config.yaml`。
3. 将 `./data` 映射到 `/app/data`。
4. 添加健康检查和合理的停止宽限期。
5. 使用 `docker compose config` 验证语法。

### Task 3: Documentation and regression

**Files:**
- Modify: `README.md`
- Modify: `docs/deployment.md`

1. 添加首次部署、启动、查看日志、更新、停止命令。
2. 说明配置挂载必须可写以及宿主机权限要求。
3. 运行 Go 测试、Go vet、前端测试与构建。
4. 检查 Git 状态，确保镜像、构建产物和真实配置未进入提交。
5. 提交 Docker 部署实现。
