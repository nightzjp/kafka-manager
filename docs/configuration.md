# 配置说明

Kafka Manager 使用一个 YAML 文件作为配置唯一数据源。配置可以手工修改，也可以在 Web 的“集群配置”页面修改；合法变更会自动热加载。

## 完整示例

```yaml
server:
  listenAddress: ":8080"
  username: admin
  passwordHash: "$argon2id$..."
  sessionHours: 12

clusters:
  - id: test
    name: 测试环境
    brokers: ["kafka-test.example.local:9092"]
    security:
      protocol: SASL_PLAINTEXT
      mechanism: PLAIN
      username: "your-kafka-username"
      password: "your-kafka-password"

  - id: internal
    name: 内网环境
    brokers: ["kafka-internal.example.local:9092"]
    security:
      protocol: PLAINTEXT

audit:
  enabled: true
  directory: ./data/audit
  retentionDays: 30
  maxFileSizeMB: 50

dashboard:
  sampleIntervalSeconds: 15
  historyPoints: 240
```

示例中的地址和凭据均为虚假占位值。`SASL_PLAINTEXT + PLAIN` 对应 Java 配置中的 `security.protocol`、`sasl.mechanism` 和 `sasl.jaas.config`；在本项目中拆分为 `protocol`、`mechanism`、`username` 和 `password`。Kafka 密码直接填写在本地 `config.yaml` 中，不需要额外环境变量。

## Kafka 安全协议

| protocol | 认证 | 加密 |
|---|---|---|
| `PLAINTEXT` | 无 | 无 |
| `SSL` | TLS 证书 | TLS |
| `SASL_PLAINTEXT` | PLAIN/SCRAM | 无 |
| `SASL_SSL` | PLAIN/SCRAM | TLS |

SASL 机制支持 `PLAIN`、`SCRAM-SHA-256`、`SCRAM-SHA-512`。

## 密码

平台登录密码只保存 Argon2id 哈希：

```bash
KAFKA_MANAGER_PASSWORD='new-password' ./kafka-manager --print-password-hash
```

Kafka 密码保存在 YAML 中；Web 接口不会回显已有密码，留空表示保持原密码。请把配置文件权限设置为 `0600`，并确保 `config.yaml` 不提交到 Git。

## 审计

目录结构：

```text
data/audit/2026-07-13/audit-001.jsonl
data/audit/2026-07-13/audit-002.jsonl
```

`retentionDays` 和 `maxFileSizeMB` 均可配置。日志不会记录消息正文、登录密码或 Kafka 密码。
