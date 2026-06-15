package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.PingCount != 4 {
		t.Errorf("expected PingCount 4, got %d", cfg.PingCount)
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg.PingCount = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ping_count")
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	tmpFile := "test_config.yaml"
	defer os.Remove(tmpFile)

	if err := WriteDefaultConfig(tmpFile); err != nil {
		t.Fatalf("WriteDefaultConfig failed: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty config")
	}
}

func TestLoad(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Logf("Load returned error (expected if no config): %v", err)
	}
	_ = cfg
}
