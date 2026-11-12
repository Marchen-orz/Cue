package config

import (
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *ConfigManager {
	t.Helper()
	return &ConfigManager{
		config:     NewDefaultConfig(),
		oldConfig:  NewDefaultConfig(),
		configPath: filepath.Join(t.TempDir(), "config"),
	}
}

// 设置页/主题切换的全量保存不携带窗口尺寸，不能把已保存的尺寸抹掉
func TestUpdateFromJSONKeepsWindowSize(t *testing.T) {
	cm := newTestManager(t)
	if err := cm.Patch(func(cfg *Config) {
		cfg.WindowWidth = 640
		cfg.WindowHeight = 520
	}); err != nil {
		t.Fatalf("Patch: %v", err)
	}

	if err := cm.UpdateFromJSON(`{"theme":"system","opacity":0.9}`); err != nil {
		t.Fatalf("UpdateFromJSON: %v", err)
	}

	got := cm.Get()
	if got.WindowWidth != 640 || got.WindowHeight != 520 {
		t.Errorf("窗口尺寸被覆盖: got %dx%d, want 640x520", got.WindowWidth, got.WindowHeight)
	}
	if got.Theme != "system" {
		t.Errorf("主题未更新: got %q, want %q", got.Theme, "system")
	}
}

// 保存后重新加载，窗口尺寸与主题档位都应还原
func TestWindowSizeAndThemeSurviveReload(t *testing.T) {
	cm := newTestManager(t)
	if err := cm.Patch(func(cfg *Config) {
		cfg.WindowWidth = 700
		cfg.WindowHeight = 500
		cfg.Theme = "system"
	}); err != nil {
		t.Fatalf("Patch: %v", err)
	}

	reloaded := &ConfigManager{configPath: cm.configPath}
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := reloaded.Get()
	if got.WindowWidth != 700 || got.WindowHeight != 500 || got.Theme != "system" {
		t.Errorf("重载后配置不一致: %dx%d theme=%q", got.WindowWidth, got.WindowHeight, got.Theme)
	}
}
