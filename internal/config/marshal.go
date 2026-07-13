package config

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// Marshal writes a valid configuration with concise field guidance. Keeping
// these comments here means Web saves do not turn config.yaml into an
// undocumented file.
func Marshal(cfg Config) ([]byte, error) {
	var body bytes.Buffer
	encoder := yaml.NewEncoder(&body)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	header := `# Kafka Manager 配置文件
# 修改后会自动热加载；也可以在 Web 的“集群配置”页面保存。
# config.yaml 包含登录及 Kafka 凭据，请设置 0600 权限且不要提交到 Git。

# Web 服务与登录配置
# listenAddress 为监听地址；username/password 为 Web 登录账号密码。
`
	text := body.String()
	text = replaceSection(text, "clusters:\n", "# Kafka 集群列表\n# protocol: PLAINTEXT / SSL / SASL_PLAINTEXT / SASL_SSL\n# SASL mechanism: PLAIN / SCRAM-SHA-256 / SCRAM-SHA-512\nclusters:\n")
	text = replaceSection(text, "audit:\n", "# 本地数据保留：审计日志按日期和大小轮转，配置备份按天清理\naudit:\n")
	text = replaceSection(text, "dashboard:\n", "# 首页监控采样：采样间隔（秒）与内存历史点数\ndashboard:\n")
	return append([]byte(header), []byte(text)...), nil
}

func replaceSection(source, old, replacement string) string {
	return strings.Replace(source, old, replacement, 1)
}
