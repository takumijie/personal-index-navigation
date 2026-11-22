package config

import (
	"path/filepath"
	"testing"
)

func TestLoadProductionEnvironment(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("HOMEPAGE_CONFIG", filepath.Join(directory, "missing.yaml"))
	t.Setenv("HOMEPAGE_SERVER_MODE", "release")
	t.Setenv("HOMEPAGE_DATABASE_PATH", "/data/homepage.db")
	t.Setenv("HOMEPAGE_STORAGE_PATH", "/storage")
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "container-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Mode != "release" || cfg.Database.Path != "/data/homepage.db" || cfg.Storage.Path != "/storage" || cfg.Admin.Password != "container-secret" {
		t.Fatalf("生产环境变量未生效: %+v", cfg)
	}
}

func TestLoadRejectsDefaultPasswordInReleaseMode(t *testing.T) {
	t.Setenv("HOMEPAGE_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("HOMEPAGE_SERVER_MODE", "release")
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "")

	if _, err := Load(); err == nil {
		t.Fatal("release 模式不应接受默认管理员密码")
	}
}
