package config

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Clusters  []ClusterConfig `yaml:"clusters"`
	Audit     AuditConfig     `yaml:"audit"`
	Dashboard DashboardConfig `yaml:"dashboard"`
}

type ServerConfig struct {
	ListenAddress string `yaml:"listenAddress"`
	Username      string `yaml:"username"`
	PasswordHash  string `yaml:"passwordHash"`
	SessionHours  int    `yaml:"sessionHours"`
}

type ClusterConfig struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Brokers  []string       `yaml:"brokers"`
	Enabled  *bool          `yaml:"enabled,omitempty"`
	Security SecurityConfig `yaml:"security"`
}

type SecurityConfig struct {
	Protocol  string `yaml:"protocol"`
	Mechanism string `yaml:"mechanism,omitempty"`
	Username  string `yaml:"username,omitempty"`
	Password  string `yaml:"password,omitempty"`
	TLS       bool   `yaml:"tls,omitempty"`
}

type AuditConfig struct {
	Enabled       *bool  `yaml:"enabled,omitempty"`
	Directory     string `yaml:"directory"`
	RetentionDays int    `yaml:"retentionDays"`
	MaxFileSizeMB int    `yaml:"maxFileSizeMB"`
}

type DashboardConfig struct {
	SampleIntervalSeconds int `yaml:"sampleIntervalSeconds"`
	HistoryPoints         int `yaml:"historyPoints"`
}
