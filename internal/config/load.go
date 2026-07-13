package config

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(reader io.Reader) (Config, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var missing string
	variablePattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	expanded := variablePattern.ReplaceAllStringFunc(string(raw), func(match string) string {
		name := variablePattern.FindStringSubmatch(match)[1]
		value, ok := os.LookupEnv(name)
		if !ok && missing == "" {
			missing = name
		}
		return value
	})
	if missing != "" {
		return Config{}, fmt.Errorf("environment variable %s is not set", missing)
	}

	cfg := defaultConfig()
	decoder := yaml.NewDecoder(strings.NewReader(expanded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Server:    ServerConfig{ListenAddress: ":8080", SessionHours: 12},
		Audit:     AuditConfig{Directory: "./data/audit", RetentionDays: 30, MaxFileSizeMB: 50},
		Dashboard: DashboardConfig{SampleIntervalSeconds: 15, HistoryPoints: 240},
	}
}
