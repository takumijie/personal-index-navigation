package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
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
		Server: Server{Listen: "0.0.0.0", Port: 8080, Mode: "debug", TrustedProxies: []string{}},
		Database: Database{Path: "data/homepage.db"},
		Storage: Storage{Path: "storage"},
		Admin: Admin{Username: "admin", Password: "admin123"},
		Upload: Upload{MaxImageBytes: 10 << 20, MaxMusicBytes: 50 << 20},
	}
}

func Load() (Config, error) {
	cfg := Default()
	path := os.Getenv("HOMEPAGE_CONFIG")
	if path == "" {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, fmt.Errorf("读取配置: %w", err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("解析配置: %w", err)
		}
	}
	if mode := os.Getenv("HOMEPAGE_SERVER_MODE"); mode != "" {
		cfg.Server.Mode = mode
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
		return cfg, errors.New("管理员用户名不能为空，密码至少 8 位")
	}
	if cfg.Admin.Password == "admin123" {
		if cfg.Server.Mode == "release" {
			return cfg, errors.New("release 模式不能使用默认管理员密码，请设置 HOMEPAGE_ADMIN_PASSWORD")
		}
		slog.Warn("正在使用仅限本地开发的默认管理员密码 admin123")
	}
	if cfg.Upload.MaxImageBytes <= 0 || cfg.Upload.MaxMusicBytes <= 0 {
		return cfg, errors.New("上传大小限制必须大于 0")
	}
	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Listen, c.Server.Port)
}
