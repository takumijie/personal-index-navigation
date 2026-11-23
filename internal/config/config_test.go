package config

import (
	"os"
	"path/filepath"
	"testing"
)

func emptyConfigFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestLoadProductionEnvironment(t *testing.T) {
	t.Setenv("HOMEPAGE_CONFIG", emptyConfigFile(t))
	t.Setenv("HOMEPAGE_SERVER_MODE", "release")
	t.Setenv("HOMEPAGE_SERVER_LISTEN", "0.0.0.0")
	t.Setenv("HOMEPAGE_SERVER_PORT", "9090")
	t.Setenv("HOMEPAGE_SERVER_COOKIE_SECURE", "false")
	t.Setenv("HOMEPAGE_DATABASE_PATH", "/data/homepage.db")
	t.Setenv("HOMEPAGE_STORAGE_PATH", "/storage")
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "container-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Mode != "release" || cfg.Server.Listen != "0.0.0.0" || cfg.Server.Port != 9090 || cfg.Database.Path != "/data/homepage.db" || cfg.Storage.Path != "/storage" || cfg.Admin.Password != "container-secret" {
		t.Fatalf("生产环境变量未生效: %+v", cfg)
	}
	if cfg.SecureCookies() {
		t.Fatal("显式 HTTP 调试配置应关闭 Secure Cookie")
	}
}

func TestLoadUsesSafeDefaultsWithoutConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, key := range []string{"HOMEPAGE_CONFIG", "HOMEPAGE_SERVER_MODE", "HOMEPAGE_SERVER_LISTEN", "HOMEPAGE_SERVER_PORT", "HOMEPAGE_SERVER_COOKIE_SECURE"} {
		unsetEnv(t, key)
	}
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "safe-local-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Mode != "release" || cfg.Server.Listen != "127.0.0.1" || cfg.Server.Port != 8080 || !cfg.SecureCookies() {
		t.Fatalf("内置安全配置不符合预期: %+v", cfg.Server)
	}
}

func TestLoadRejectsMissingOrMalformedExplicitConfig(t *testing.T) {
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "safe-local-secret")
	t.Run("missing", func(t *testing.T) {
		t.Setenv("HOMEPAGE_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
		if _, err := Load(); err == nil {
			t.Fatal("显式配置文件缺失时不应回退到默认配置")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte("server: ["), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOMEPAGE_CONFIG", path)
		if _, err := Load(); err == nil {
			t.Fatal("显式配置解析失败时不应继续启动")
		}
	})
}

func TestLoadRejectsDefaultPasswordInEveryMode(t *testing.T) {
	for _, mode := range []string{"debug", "release", "test"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("HOMEPAGE_CONFIG", emptyConfigFile(t))
			t.Setenv("HOMEPAGE_SERVER_MODE", mode)
			t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "admin123")
			if _, err := Load(); err == nil {
				t.Fatal("所有模式都不应接受默认管理员密码")
			}
		})
	}
}

func TestLoadRequiresAdministratorPassword(t *testing.T) {
	t.Setenv("HOMEPAGE_CONFIG", emptyConfigFile(t))
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("未配置管理员密码时不应启动")
	}
}

func TestSecureCookieDefaults(t *testing.T) {
	for _, test := range []struct {
		mode   string
		secure bool
	}{{"debug", false}, {"test", false}, {"release", true}} {
		cfg := Default()
		cfg.Server.Mode = test.mode
		if got := cfg.SecureCookies(); got != test.secure {
			t.Errorf("%s 模式 Secure Cookie = %v，预期 %v", test.mode, got, test.secure)
		}
	}
	secure := false
	cfg := Default()
	cfg.Server.CookieSecure = &secure
	if cfg.SecureCookies() {
		t.Fatal("显式 false 不应被 release 默认值覆盖")
	}
}
