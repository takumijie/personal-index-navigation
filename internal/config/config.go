package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Storage  Storage  `yaml:"storage"`
	Admin    Admin    `yaml:"admin"`
	Upload   Upload   `yaml:"upload"`
}

type Server struct {
	Listen         string   `yaml:"listen"`
	Port           int      `yaml:"port"`
	Mode           string   `yaml:"mode"`
	TrustedProxies []string `yaml:"trusted_proxies"`
	CookieSecure   *bool    `yaml:"cookie_secure"`
}

type Database struct {
	Path string `yaml:"path"`
}

type Storage struct {
	Path string `yaml:"path"`
}

type Admin struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Upload struct {
	MaxImageBytes int64 `yaml:"max_image_bytes"`
	MaxMusicBytes int64 `yaml:"max_music_bytes"`
}

func Default() Config {
	return Config{
		Server:   Server{Listen: "127.0.0.1", Port: 8080, Mode: "release", TrustedProxies: []string{}},
		Database: Database{Path: "data/homepage.db"},
		Storage:  Storage{Path: "storage"},
		Admin:    Admin{Username: "admin"},
		Upload:   Upload{MaxImageBytes: 10 << 20, MaxMusicBytes: 50 << 20},
	}
}

func Load() (Config, error) {
	cfg := Default()
	path, explicitPath := os.LookupEnv("HOMEPAGE_CONFIG")
	if explicitPath && strings.TrimSpace(path) == "" {
		return cfg, errors.New("HOMEPAGE_CONFIG 已设置但路径为空")
	}
	if !explicitPath {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if explicitPath || !errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("读取配置: %w", err)
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置: %w", err)
	}
	if mode := os.Getenv("HOMEPAGE_SERVER_MODE"); mode != "" {
		cfg.Server.Mode = mode
	}
	if listen := os.Getenv("HOMEPAGE_SERVER_LISTEN"); listen != "" {
		cfg.Server.Listen = listen
	}
	if value := os.Getenv("HOMEPAGE_SERVER_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return cfg, errors.New("HOMEPAGE_SERVER_PORT 必须是整数")
		}
		cfg.Server.Port = port
	}
	if value := os.Getenv("HOMEPAGE_SERVER_COOKIE_SECURE"); value != "" {
		secure, err := strconv.ParseBool(value)
		if err != nil {
			return cfg, errors.New("HOMEPAGE_SERVER_COOKIE_SECURE 仅支持 true 或 false")
		}
		cfg.Server.CookieSecure = &secure
	}
	if path := os.Getenv("HOMEPAGE_DATABASE_PATH"); path != "" {
		cfg.Database.Path = path
	}
	if path := os.Getenv("HOMEPAGE_STORAGE_PATH"); path != "" {
		cfg.Storage.Path = path
	}
	if password := os.Getenv("HOMEPAGE_ADMIN_PASSWORD"); password != "" {
		cfg.Admin.Password = password
	}
	if cfg.Server.Listen == "" || cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return cfg, errors.New("server.listen 或 server.port 配置无效")
	}
	if cfg.Server.Mode != "debug" && cfg.Server.Mode != "release" && cfg.Server.Mode != "test" {
		return cfg, errors.New("server.mode 仅支持 debug、release、test")
	}
	if cfg.Database.Path == "" || cfg.Storage.Path == "" {
		return cfg, errors.New("数据库与 storage 路径不能为空")
	}
	if strings.TrimSpace(cfg.Admin.Username) == "" || len(cfg.Admin.Password) < 8 {
		return cfg, errors.New("管理员用户名不能为空，且必须通过配置或 HOMEPAGE_ADMIN_PASSWORD 提供至少 8 位密码")
	}
	if cfg.Admin.Password == "admin123" {
		return cfg, errors.New("不能使用默认管理员密码 admin123，请设置安全密码")
	}
	if cfg.Upload.MaxImageBytes <= 0 || cfg.Upload.MaxMusicBytes <= 0 {
		return cfg, errors.New("上传大小限制必须大于 0")
	}
	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Listen, c.Server.Port)
}

func (c Config) SecureCookies() bool {
	if c.Server.CookieSecure != nil {
		return *c.Server.CookieSecure
	}
	return c.Server.Mode == "release"
}
