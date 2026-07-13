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
EnvironmentFile=/etc/kafka-manager/env
ExecStart=/opt/kafka-manager/kafka-manager --config /etc/kafka-manager/config.yaml
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

## 升级

1. 备份二进制、`config.yaml` 和 `data/`。
2. 使用新二进制执行 `--print-password-hash`，确认可以启动。
3. 停止旧进程并替换二进制。
4. 使用原配置启动新版本。
5. 检查首页集群状态和日志。

配置 Web 保存前会在 `data/config-backups/YYYY-MM-DD/` 生成备份。

## 反向代理

生产环境推荐在 Nginx、Caddy 或内部网关后启用 HTTPS。会话 Cookie 在 HTTPS 请求下自动设置 `Secure`。如果直接暴露 HTTP，仅应在可信内网使用。

## 健康检查

```text
GET /api/v1/health
```

健康接口表示进程可服务；Kafka 集群是否在线以首页和 `/api/v1/clusters` 为准。
