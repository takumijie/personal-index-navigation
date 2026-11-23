package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	t.Setenv("HOMEPAGE_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8, ::1")
	t.Setenv("HOMEPAGE_SERVER_COOKIE_SECURE", "false")
	t.Setenv("HOMEPAGE_DATABASE_PATH", "/data/homepage.db")
	t.Setenv("HOMEPAGE_STORAGE_PATH", "/storage")
	t.Setenv("HOMEPAGE_ADMIN_USERNAME", "operator")
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "container-secret")
	t.Setenv("HOMEPAGE_UPLOAD_MAX_IMAGE_BYTES", "1048577")
	t.Setenv("HOMEPAGE_UPLOAD_MAX_MUSIC_BYTES", "52428801")
	t.Setenv("HOMEPAGE_MEDIA_ALLOW_INSECURE_HTTP", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Mode != "release" || cfg.Server.Listen != "0.0.0.0" || cfg.Server.Port != 9090 || cfg.Database.Path != "/data/homepage.db" || cfg.Storage.Path != "/storage" || cfg.Admin.Username != "operator" || cfg.Admin.Password != "container-secret" {
		t.Fatalf("生产环境变量未生效: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.Server.TrustedProxies, []string{"127.0.0.1", "10.0.0.0/8", "::1"}) {
		t.Fatalf("可信代理环境变量未生效: %+v", cfg.Server.TrustedProxies)
	}
	if cfg.Upload.MaxImageBytes != 1048577 || cfg.Upload.MaxMusicBytes != 52428801 {
		t.Fatalf("上传上限环境变量未生效: %+v", cfg.Upload)
	}
	if !cfg.Media.AllowInsecureHTTP {
		t.Fatal("媒体兼容模式环境变量未生效")
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

func TestLoadAllowsMissingAdministratorPassword(t *testing.T) {
	t.Setenv("HOMEPAGE_CONFIG", emptyConfigFile(t))
	t.Setenv("HOMEPAGE_ADMIN_PASSWORD", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置本身不应要求管理员密码: %v", err)
	}
	if cfg.Admin.Password != "" {
		t.Fatalf("管理员密码应为空，实际为 %q", cfg.Admin.Password)
	}
}

func TestLoadRejectsUnknownYAMLFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  unknown_field: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMEPAGE_CONFIG", path)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "unknown_field") {
		t.Fatalf("未知 YAML 字段应解析失败，实际错误: %v", err)
	}
}

func TestLoadValidatesTrustedProxies(t *testing.T) {
	for _, proxy := range []string{"https://proxy.example.com", "localhost", "10.0.0.1/33", "127.0.0.1,"} {
		t.Run(proxy, func(t *testing.T) {
			t.Setenv("HOMEPAGE_CONFIG", emptyConfigFile(t))
			t.Setenv("HOMEPAGE_TRUSTED_PROXIES", proxy)
			if _, err := Load(); err == nil {
				t.Fatalf("非法可信代理 %q 应被拒绝", proxy)
			}
		})
	}
}

func TestValidateAdminPasswordUsesRunesAndUTF8Bytes(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{password: "12345678", valid: true},
		{password: "密码密码密码密码", valid: true},
		{password: strings.Repeat("a", 72), valid: true},
		{password: strings.Repeat("密", 24), valid: true},
		{password: "密码密码密码密", valid: false},
		{password: strings.Repeat("a", 73), valid: false},
		{password: strings.Repeat("密", 25), valid: false},
		{password: "admin123", valid: false},
	}
	for _, test := range tests {
		if err := ValidateAdminPassword(test.password); (err == nil) != test.valid {
			t.Errorf("ValidateAdminPassword(%q) error = %v，预期合法=%v", test.password, err, test.valid)
		}
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
