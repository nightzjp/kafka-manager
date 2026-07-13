# 部署与升级

## 最小部署

运行时只需要三个内容：

```text
kafka-manager
config.yaml
data/                 # 必须可写
```

启动：

```bash
./kafka-manager --config ./config.yaml
```

建议使用专用系统用户运行，并让配置文件和数据目录仅该用户可读写。

## systemd 示例

```ini
[Unit]
Description=Kafka Manager
After=network-online.target

[Service]
User=kafka-manager
WorkingDirectory=/opt/kafka-manager
ExecStart=/opt/kafka-manager/kafka-manager --config /etc/kafka-manager/config.yaml
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

## 升级

1. 备份二进制、`config.yaml` 和 `data/`。
2. 检查新版本配置说明和变更记录。
3. 停止旧进程并替换二进制。
4. 使用原配置启动新版本。
5. 检查首页集群状态和日志。

配置 Web 保存前会在 `data/config-backups/YYYY-MM-DD/` 生成备份。

## Docker Compose 部署

服务器需要 Docker Engine 和 Docker Compose 插件，不需要安装 Go、Node.js 或 pnpm。

```bash
cp config.example.yaml config.yaml
chmod 600 config.yaml
mkdir -p data
docker compose up -d --build
```

Compose 使用以下映射：

```text
./config.yaml -> /app/config.yaml
./data        -> /app/data
8080          -> 8080
```

配置文件挂载没有使用 `:ro`，因为 Web 的“集群配置”页面需要原子写回该文件。容器默认使用 UID/GID `1000:1000` 的非 root 用户；Linux 服务器如果当前用户不是 UID 1000，需要让该用户能够写入 `config.yaml` 和 `data/`。

常用命令：

```bash
# 启动或重新构建
docker compose up -d --build

# 查看日志和健康状态
docker compose logs -f kafka-manager
docker compose ps

# 停止；不会删除 config.yaml 和 data/
docker compose down
```

修改宿主机 `config.yaml` 后，运行中的应用会自动热加载。通过 Web 保存配置也会同步更新宿主机文件。

## 反向代理

生产环境推荐在 Nginx、Caddy 或内部网关后启用 HTTPS。会话 Cookie 在 HTTPS 请求下自动设置 `Secure`。如果直接暴露 HTTP，仅应在可信内网使用。

## 健康检查

```text
GET /api/v1/health
```

健康接口表示进程可服务；Kafka 集群是否在线以首页和 `/api/v1/clusters` 为准。
