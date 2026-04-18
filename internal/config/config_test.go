package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsErrorForMalformedConfig(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, "config")
	configDir := filepath.Join(configHome, "penta")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "penta.yaml")
	if err := os.WriteFile(configPath, []byte("db_path: [unterminated"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	if _, err := Load(); err == nil {
		t.Fatal("expected malformed config to return an error")
	}
}

func TestLoadUsesEnvOverride(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(home, "custom.db")

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("PENTA_DB_PATH", dbPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DBPath != dbPath {
		t.Fatalf("unexpected db path: got %q want %q", cfg.DBPath, dbPath)
	}
}
