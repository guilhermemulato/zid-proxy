package agentui

import (
	"errors"
	"fmt"
	"strings"
)

type Config struct {
	Port            int    `json:"port"`
	DNSFallback     string `json:"dns_fallback"`
	IntervalSeconds int    `json:"interval_seconds"`
}

func DefaultConfig() Config {
	return Config{
		Port:            18443,
		DNSFallback:     "zid-proxy.lan",
		IntervalSeconds: 30,
	}
}

func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if strings.TrimSpace(c.DNSFallback) == "" {
		return errors.New("dns_fallback is required")
	}
	if c.IntervalSeconds < 5 || c.IntervalSeconds > 3600 {
		return fmt.Errorf("invalid interval_seconds: %d (expected 5..3600)", c.IntervalSeconds)
	}
	return nil
}
