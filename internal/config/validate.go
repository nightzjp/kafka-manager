package config

import (
	"fmt"
	"strings"
)

func (c Config) Validate() error {
	if strings.TrimSpace(c.Server.Username) == "" {
		return fmt.Errorf("server username is required")
	}
	if strings.TrimSpace(c.Server.PasswordHash) == "" {
		return fmt.Errorf("server passwordHash is required")
	}
	if len(c.Clusters) == 0 {
		return fmt.Errorf("at least one cluster is required")
	}
	seen := make(map[string]struct{}, len(c.Clusters))
	for i, cluster := range c.Clusters {
		if err := validateCluster(cluster); err != nil {
			return fmt.Errorf("cluster %d: %w", i, err)
		}
		if _, exists := seen[cluster.ID]; exists {
			return fmt.Errorf("duplicate cluster id %q", cluster.ID)
		}
		seen[cluster.ID] = struct{}{}
	}
	if c.Audit.RetentionDays < 1 {
		return fmt.Errorf("audit retentionDays must be greater than zero")
	}
	if c.Audit.MaxFileSizeMB < 1 {
		return fmt.Errorf("audit maxFileSizeMB must be greater than zero")
	}
	return nil
}

func validateCluster(cluster ClusterConfig) error {
	if strings.TrimSpace(cluster.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(cluster.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(cluster.Brokers) == 0 {
		return fmt.Errorf("at least one broker is required")
	}
	for _, broker := range cluster.Brokers {
		if strings.TrimSpace(broker) == "" {
			return fmt.Errorf("broker must not be empty")
		}
	}

	protocol := cluster.Security.Protocol
	if protocol == "" {
		protocol = "PLAINTEXT"
	}
	switch protocol {
	case "PLAINTEXT", "SSL":
		return nil
	case "SASL_PLAINTEXT", "SASL_SSL":
		if cluster.Security.Mechanism == "" {
			return fmt.Errorf("SASL mechanism is required")
		}
		switch cluster.Security.Mechanism {
		case "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
		default:
			return fmt.Errorf("unsupported SASL mechanism %q", cluster.Security.Mechanism)
		}
		if cluster.Security.Username == "" {
			return fmt.Errorf("SASL username is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported security protocol %q", protocol)
	}
}
