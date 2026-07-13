# 安全说明

- 仅部署在可信内网，并通过 HTTPS 反向代理访问。
- 为平台账号设置独立强密码，不与 Kafka 账号复用。
- `KAFKA_MANAGER_SECRET_KEY` 至少 32 字节，不写入 Git 或 YAML。
- 配置文件权限建议为 `0600`，数据目录权限建议为 `0700`。
- Kafka 账号遵循最小权限原则；生产集群可单独限制删除 Topic 等能力。
- 删除 Topic、生产消息和重置 Offset 会写入审计，但不会记录消息正文或密码。
- Web 返回配置时会清空登录哈希和 Kafka 密码。
- 登录失败按客户端地址限速，会话 Cookie 使用 HttpOnly 和 SameSite=Strict。

报告安全问题时请不要创建公开 Issue，使用仓库 Security Advisory。
