package config

import (
	"strings"
	"testing"
)

func TestLoadValidConfiguration(t *testing.T) {
	input := `
server:
  listenAddress: ":8080"
  username: admin
  password: login-secret
clusters:
  - id: dev
    name: 开发环境
    brokers: ["localhost:9092"]
    security:
      protocol: PLAINTEXT
  - id: test
    name: 测试环境
    brokers: ["localhost:9093"]
    security:
      protocol: SASL_PLAINTEXT
      mechanism: SCRAM-SHA-256
      username: kafka-user
      password: secret
audit:
  directory: ./data/audit
  retentionDays: 30
  maxFileSizeMB: 50
`

	cfg, err := Load(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Clusters) != 2 {
		t.Fatalf("clusters = %d, want 2", len(cfg.Clusters))
	}
	if cfg.Server.Password != "login-secret" {
		t.Fatalf("password = %q", cfg.Server.Password)
	}
	if cfg.Audit.RetentionDays != 30 {
		t.Fatalf("retention = %d, want 30", cfg.Audit.RetentionDays)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	input := `
server:
  username: admin
  passwordHash: hash
clusters:
  - id: dev
    name: 开发环境
    brokers: ["localhost:9092"]
`

	cfg, err := Load(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.ListenAddress != ":8080" {
		t.Fatalf("listen address = %q, want :8080", cfg.Server.ListenAddress)
	}
	if cfg.Audit.RetentionDays != 30 || cfg.Audit.MaxFileSizeMB != 50 {
		t.Fatalf("unexpected audit defaults: %+v", cfg.Audit)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"duplicate cluster id", validPrefix() + secondCluster("dev", "PLAINTEXT", "") + validAudit(), "duplicate cluster id"},
		{"empty broker", validPrefixWithBroker("") + validAudit(), "broker"},
		{"unsupported protocol", validPrefixWithProtocol("MAGIC") + validAudit(), "security protocol"},
		{"sasl missing mechanism", validPrefixWithProtocol("SASL_PLAINTEXT") + validAudit(), "mechanism"},
		{"invalid retention", validPrefix() + "audit:\n  retentionDays: 0\n  maxFileSizeMB: 50\n", "retentionDays"},
		{"unknown yaml field", validPrefix() + validAudit() + "unknown: true\n", "field unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestLoadRejectsMissingEnvironmentVariable(t *testing.T) {
	input := strings.Replace(validPrefix(), "localhost:9092", "${KAFKA_MANAGER_MISSING_TEST}", 1) + validAudit()

	_, err := Load(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "KAFKA_MANAGER_MISSING_TEST") {
		t.Fatalf("Load() error = %v, want missing variable name", err)
	}
}

func TestLoadPreservesDollarSignsOutsideBracedVariables(t *testing.T) {
	input := strings.Replace(validPrefix(), "passwordHash: hash", "passwordHash: $argon2id$v=19$abc", 1) + validAudit()
	cfg, err := Load(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Load() error=%v", err)
	}
	if cfg.Server.PasswordHash != "$argon2id$v=19$abc" {
		t.Fatalf("hash=%q", cfg.Server.PasswordHash)
	}
}

func TestMarshalIncludesConfigurationCommentsAndReloads(t *testing.T) {
	cfg, err := Load(strings.NewReader(validPrefix() + validAudit()))
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, comment := range []string{"Web 服务与登录配置", "Kafka 集群列表", "审计日志", "首页监控采样"} {
		if !strings.Contains(text, "# "+comment) {
			t.Fatalf("missing comment %q in:\n%s", comment, text)
		}
	}
	if _, err := Load(strings.NewReader(text)); err != nil {
		t.Fatalf("commented output cannot reload: %v", err)
	}
}

func validPrefix() string { return validPrefixWithBroker("localhost:9092") }

func validPrefixWithBroker(broker string) string {
	return "server:\n  username: admin\n  passwordHash: hash\nclusters:\n  - id: dev\n    name: Development\n    brokers: [\"" + broker + "\"]\n"
}

func validPrefixWithProtocol(protocol string) string {
	return validPrefix() + "    security:\n      protocol: " + protocol + "\n"
}

func secondCluster(id, protocol, mechanism string) string {
	result := "  - id: " + id + "\n    name: Second\n    brokers: [\"localhost:9093\"]\n    security:\n      protocol: " + protocol + "\n"
	if mechanism != "" {
		result += "      mechanism: " + mechanism + "\n"
	}
	return result
}

func validAudit() string {
	return "audit:\n  directory: ./data/audit\n  retentionDays: 30\n  maxFileSizeMB: 50\n"
}
