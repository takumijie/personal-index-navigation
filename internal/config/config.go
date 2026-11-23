package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Storage  Storage  `yaml:"storage"`
	Admin    Admin    `yaml:"admin"`
	Upload   Upload   `yaml:"upload"`
	Media    Media    `yaml:"media"`
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

type Media struct {
	AllowInsecureHTTP bool `yaml:"allow_insecure_http"`
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
	} else {
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
			return cfg, fmt.Errorf("解析配置: %w", err)
		}
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
	if value := os.Getenv("HOMEPAGE_TRUSTED_PROXIES"); value != "" {
		proxies := strings.Split(value, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		cfg.Server.TrustedProxies = proxies
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
	if username := os.Getenv("HOMEPAGE_ADMIN_USERNAME"); username != "" {
		cfg.Admin.Username = username
	}
	if password := os.Getenv("HOMEPAGE_ADMIN_PASSWORD"); password != "" {
		cfg.Admin.Password = password
	}
	if value := os.Getenv("HOMEPAGE_UPLOAD_MAX_IMAGE_BYTES"); value != "" {
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return cfg, errors.New("HOMEPAGE_UPLOAD_MAX_IMAGE_BYTES 必须是整数")
		}
		cfg.Upload.MaxImageBytes = limit
	}
	if value := os.Getenv("HOMEPAGE_UPLOAD_MAX_MUSIC_BYTES"); value != "" {
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return cfg, errors.New("HOMEPAGE_UPLOAD_MAX_MUSIC_BYTES 必须是整数")
		}
		cfg.Upload.MaxMusicBytes = limit
	}
	if value := os.Getenv("HOMEPAGE_MEDIA_ALLOW_INSECURE_HTTP"); value != "" {
		allow, err := strconv.ParseBool(value)
		if err != nil {
			return cfg, errors.New("HOMEPAGE_MEDIA_ALLOW_INSECURE_HTTP 仅支持 true 或 false")
		}
		cfg.Media.AllowInsecureHTTP = allow
	}
	if cfg.Server.Listen == "" || cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return cfg, errors.New("server.listen 或 server.port 配置无效")
	}
	if cfg.Server.Mode != "debug" && cfg.Server.Mode != "release" && cfg.Server.Mode != "test" {
		return cfg, errors.New("server.mode 仅支持 debug、release、test")
	}
	for _, proxy := range cfg.Server.TrustedProxies {
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return cfg, fmt.Errorf("server.trusted_proxies 包含无效 IP/CIDR: %q", proxy)
			}
		}
	}
	if cfg.Database.Path == "" || cfg.Storage.Path == "" {
		return cfg, errors.New("数据库与 storage 路径不能为空")
	}
	if strings.TrimSpace(cfg.Admin.Username) == "" {
		return cfg, errors.New("管理员用户名不能为空")
	}
	if cfg.Admin.Password != "" {
		if err := ValidateAdminPassword(cfg.Admin.Password); err != nil {
			return cfg, err
		}
	}
	if cfg.Upload.MaxImageBytes <= 0 || cfg.Upload.MaxMusicBytes <= 0 {
		return cfg, errors.New("上传大小限制必须大于 0")
	}
	return cfg, nil
}

func ValidateAdminPassword(password string) error {
	if !utf8.ValidString(password) {
		return errors.New("管理员密码必须是有效的 UTF-8 字符串")
	}
	if utf8.RuneCountInString(password) < 8 {
		return errors.New("管理员密码至少需要 8 个 Unicode 字符")
	}
	if len(password) > 72 {
		return errors.New("管理员密码的 UTF-8 编码不能超过 72 字节")
	}
	if password == "admin123" {
		return errors.New("不能使用默认管理员密码 admin123，请设置安全密码")
	}
	return nil
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
