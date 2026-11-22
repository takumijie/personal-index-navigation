package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	"homepage/internal/config"
	"homepage/internal/database"
	"homepage/internal/handler"
	"homepage/internal/repository"
	"homepage/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func run() error {
	healthcheck := flag.Bool("healthcheck", false, "检查运行中的服务健康状态")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *healthcheck {
		return checkHealth(cfg)
	}
	if err := os.MkdirAll(cfg.Storage.Path, 0o755); err != nil {
		return fmt.Errorf("创建 storage 目录: %w", err)
	}
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("数据库迁移: %w", err)
	}
	if err := database.Seed(db, cfg); err != nil {
		return fmt.Errorf("初始化数据: %w", err)
	}
	repo := repository.New(db)
	content := service.NewContentService(repo)
	auth := service.NewAuthService(repo)
	uploads := service.NewUploadService(cfg.Storage.Path, cfg.Upload.MaxImageBytes, cfg.Upload.MaxMusicBytes)
	music := service.NewMusicService(repo, cfg.Storage.Path)
	providerClient := &http.Client{Timeout: 20 * time.Second}
	netease, err := service.NewNeteaseService(repo, providerClient, "https://music-api.ilibilib.top")
	if err != nil {
		return fmt.Errorf("初始化网易云音乐: %w", err)
	}
	s3 := service.NewS3Service(repo, providerClient)
	music.RegisterProvider("netease", netease)
	music.RegisterProvider("s3", s3)
	api := handler.NewAPI(content, auth, uploads, music)
	api.SetMusicProviders(netease, s3)
	router, err := handler.NewRouter(cfg, api)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: cfg.Address(), Handler: router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("服务已启动", "address", cfg.Address(), "mode", cfg.Server.Mode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return fmt.Errorf("HTTP 服务: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		slog.Info("正在关闭服务")
		return server.Shutdown(shutdownCtx)
	}
}

func checkHealth(cfg config.Config) error {
	host := cfg.Server.Listen
	if host == "0.0.0.0" || host == "::" || host == "" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Server.Port)) + "/healthz"
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("健康检查返回状态 %d", response.StatusCode)
	}
	return nil
}
