package config

type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Clusters  []ClusterConfig `yaml:"clusters" json:"clusters"`
	Audit     AuditConfig     `yaml:"audit" json:"audit"`
	Dashboard DashboardConfig `yaml:"dashboard" json:"dashboard"`
}

type ServerConfig struct {
	ListenAddress string `yaml:"listenAddress" json:"listenAddress"`
	Username      string `yaml:"username" json:"username"`
	Password      string `yaml:"password,omitempty" json:"password,omitempty"`
	PasswordHash  string `yaml:"passwordHash,omitempty" json:"passwordHash,omitempty"`
	SessionHours  int    `yaml:"sessionHours" json:"sessionHours"`
}

type ClusterConfig struct {
	ID       string         `yaml:"id" json:"id"`
	Name     string         `yaml:"name" json:"name"`
	Brokers  []string       `yaml:"brokers" json:"brokers"`
	Enabled  *bool          `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Security SecurityConfig `yaml:"security" json:"security"`
}

type SecurityConfig struct {
	Protocol  string `yaml:"protocol" json:"protocol"`
	Mechanism string `yaml:"mechanism,omitempty" json:"mechanism,omitempty"`
	Username  string `yaml:"username,omitempty" json:"username,omitempty"`
	Password  string `yaml:"password,omitempty" json:"password,omitempty"`
	TLS       bool   `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type AuditConfig struct {
	Enabled       *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Directory     string `yaml:"directory" json:"directory"`
	RetentionDays int    `yaml:"retentionDays" json:"retentionDays"`
	MaxFileSizeMB int    `yaml:"maxFileSizeMB" json:"maxFileSizeMB"`
}

type DashboardConfig struct {
	SampleIntervalSeconds int `yaml:"sampleIntervalSeconds" json:"sampleIntervalSeconds"`
	HistoryPoints         int `yaml:"historyPoints" json:"historyPoints"`
}
