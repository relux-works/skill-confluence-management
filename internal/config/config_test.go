package config

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpConfigPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "config.yaml")
}

func TestConfigManager_GetConfig_NoFile(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))
	cfg, err := m.GetConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveSpace != "" {
		t.Errorf("expected empty active space, got %q", cfg.ActiveSpace)
	}
}

func TestConfigManager_SetAndGetActiveSpace(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))

	if err := m.SetActiveSpace("DEV"); err != nil {
		t.Fatalf("SetActiveSpace error: %v", err)
	}

	cfg, err := m.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}
	if cfg.ActiveSpace != "DEV" {
		t.Errorf("active_space = %q, want DEV", cfg.ActiveSpace)
	}
}

func TestConfigManager_SetInstanceURL(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))

	if err := m.SetInstanceURL("https://test.atlassian.net/wiki"); err != nil {
		t.Fatalf("error: %v", err)
	}

	cfg, _ := m.GetConfig()
	if cfg.InstanceURL != "https://test.atlassian.net/wiki" {
		t.Errorf("instance_url = %q", cfg.InstanceURL)
	}
}

func TestConfigManager_SetInstanceType(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))

	if err := m.SetInstanceType("cloud"); err != nil {
		t.Fatalf("error: %v", err)
	}

	cfg, _ := m.GetConfig()
	if cfg.InstanceType != "cloud" {
		t.Errorf("instance_type = %q", cfg.InstanceType)
	}
}

func TestConfigManager_SetAuthType(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))

	if err := m.SetAuthType("bearer"); err != nil {
		t.Fatalf("error: %v", err)
	}

	cfg, _ := m.GetConfig()
	if cfg.AuthType != "bearer" {
		t.Errorf("auth_type = %q", cfg.AuthType)
	}
}

func TestConfigManager_Exists(t *testing.T) {
	path := tmpConfigPath(t)
	m := NewConfigManagerWithPath(path)

	if m.Exists() {
		t.Error("file should not exist yet")
	}

	_ = m.SetActiveSpace("X")

	if !m.Exists() {
		t.Error("file should exist after save")
	}
}

func TestConfigManager_ConfigPath(t *testing.T) {
	path := "/tmp/test-config.yaml"
	m := NewConfigManagerWithPath(path)
	if m.ConfigPath() != path {
		t.Errorf("path = %q, want %q", m.ConfigPath(), path)
	}
}

func TestConfigManager_PreservesFields(t *testing.T) {
	m := NewConfigManagerWithPath(tmpConfigPath(t))

	_ = m.SetInstanceURL("https://test.atlassian.net/wiki")
	_ = m.SetActiveSpace("DEV")
	_ = m.SetInstanceType("cloud")

	cfg, _ := m.GetConfig()
	if cfg.InstanceURL != "https://test.atlassian.net/wiki" {
		t.Errorf("instance_url lost: %q", cfg.InstanceURL)
	}
	if cfg.ActiveSpace != "DEV" {
		t.Errorf("active_space lost: %q", cfg.ActiveSpace)
	}
	if cfg.InstanceType != "cloud" {
		t.Errorf("instance_type lost: %q", cfg.InstanceType)
	}
}

func TestConfigManager_CorruptYAML(t *testing.T) {
	path := tmpConfigPath(t)
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("{{{{invalid yaml"), 0o644)

	m := NewConfigManagerWithPath(path)
	_, err := m.GetConfig()
	if err == nil {
		t.Fatal("expected error for corrupt YAML")
	}
}
