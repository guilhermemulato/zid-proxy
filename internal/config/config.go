package config

import "time"

// Config holds the application configuration
type Config struct {
	// ListenAddr is the address to listen on (e.g., ":443" or "0.0.0.0:8443")
	ListenAddr string

	// RulesFile is the path to the access rules file
	RulesFile string

	// LogFile is the path to the log file
	LogFile string

	// PidFile is the path to the PID file
	PidFile string

	// ReadTimeout is the timeout for reading from connections
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for writing to connections
	WriteTimeout time.Duration

	// ActiveIPsFile is the JSON snapshot file path for active IP stats
	ActiveIPsFile string
	// ActiveIPsInterval controls how often the snapshot is written
	ActiveIPsInterval time.Duration
	// ActiveIPsTimeout removes IPs after this idle time (no traffic and no active conns)
	ActiveIPsTimeout time.Duration
	// ActiveIPsMax caps the number of tracked IPs
	ActiveIPsMax int

	// AgentListenAddr enables the agent HTTP API listener when non-empty (e.g. "192.168.1.1:18443")
	AgentListenAddr string
	// AgentTTL removes agent entries after this idle time (no heartbeat)
	AgentTTL time.Duration
}

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		ListenAddr:        ":443",
		RulesFile:         "/usr/local/etc/zid-proxy/access_rules.txt",
		LogFile:           "/var/log/zid-proxy.log",
		PidFile:           "/var/run/zid-proxy.pid",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ActiveIPsFile:     "/var/run/zid-proxy.active_ips.json",
		ActiveIPsInterval: 2 * time.Second,
		ActiveIPsTimeout:  120 * time.Second,
		ActiveIPsMax:      5000,
		AgentListenAddr:   "",
		AgentTTL:          120 * time.Second,
	}
}
