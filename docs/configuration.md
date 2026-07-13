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
    brokers: ["121.41.66.5:19094"]
    security:
      protocol: SASL_PLAINTEXT
      mechanism: PLAIN
      username: kafka
      password: "${KAFKA_TEST_PASSWORD}"

  - id: internal
    name: 内网环境
    brokers: ["192.168.20.200:9092"]
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

仅 `${NAME}` 形式会展开环境变量；普通 `$` 会原样保留，因此不会破坏 Argon2id 哈希。

上面的测试环境对应原 kafka-ui 中的 `SASL_PLAINTEXT + PLAIN` 配置；Java 的 `sasl.jaas.config` 在本项目中拆分为 `mechanism`、`username` 和 `password`。内网环境沿用无认证的 `PLAINTEXT`。启动前设置测试集群密码：

```bash
export KAFKA_TEST_PASSWORD='测试集群密码'
```

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

通过 Web 保存的 Kafka 密码使用 AES-GCM 加密，格式为 `enc:v1:...`。加密密钥来自 `KAFKA_MANAGER_SECRET_KEY`，至少 32 字节。丢失密钥后无法恢复已加密密码。

## 审计

目录结构：

```text
data/audit/2026-07-13/audit-001.jsonl
data/audit/2026-07-13/audit-002.jsonl
```

`retentionDays` 和 `maxFileSizeMB` 均可配置。日志不会记录消息正文、登录密码或 Kafka 密码。
