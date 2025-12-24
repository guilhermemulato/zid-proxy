package agentui

import (
	"path/filepath"
	"testing"
)

func TestConfigManager_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cm := NewConfigManager(path, DefaultConfig())
	cfg := DefaultConfig()
	cfg.Port = 19000
	cfg.DNSFallback = "example.local"
	cfg.IntervalSeconds = 15

	if err := cm.Set(cfg); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	if err := cm.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk() error: %v", err)
	}

	cm2 := NewConfigManager(path, DefaultConfig())
	if err := cm2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk() error: %v", err)
	}

	got := cm2.Get()
	if got.Port != cfg.Port || got.DNSFallback != cfg.DNSFallback || got.IntervalSeconds != cfg.IntervalSeconds {
		t.Fatalf("round-trip mismatch: got=%+v want=%+v", got, cfg)
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port=0")
	}

	cfg = DefaultConfig()
	cfg.DNSFallback = " "
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty dns_fallback")
	}

	cfg = DefaultConfig()
	cfg.IntervalSeconds = 2
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for interval_seconds=2")
	}
}
