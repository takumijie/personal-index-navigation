package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"homepage/internal/config"
	"homepage/internal/middleware"
	"homepage/web"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, api *API) (*gin.Engine, error) {
	api.cookieSecure = cfg.SecureCookies()
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	if len(cfg.Server.TrustedProxies) == 0 {
		if err := router.SetTrustedProxies(nil); err != nil {
			return nil, err
		}
	} else if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("配置可信代理: %w", err)
	}
	router.Use(middleware.Logger(), middleware.Recovery(), middleware.Security())
	if cfg.Server.Mode == gin.DebugMode {
		router.Use(middleware.DevelopmentCORS())
	}
	router.GET("/healthz", api.Health)
	router.GET("/storage/images/:filename", staticFile(cfg.Storage.Path, "images"))
	router.GET("/storage/music/:filename", staticFile(cfg.Storage.Path, "music"))

	v1 := router.Group("/api/v1")
	v1.GET("/health", api.Health)
	v1.GET("/profile", api.Profile)
	v1.GET("/navigation", api.Navigation)
	v1.GET("/devices", api.Devices)
	v1.GET("/settings", api.Settings)
	v1.GET("/guestbook", api.Guestbook)
	v1.POST("/guestbook", middleware.BodyLimit(2<<20), api.CreateGuestbook)
	v1.GET("/music/tracks", api.Tracks)
	v1.GET("/music/playlist", api.Playlist)
	v1.GET("/music/playlists", api.Playlists)
	v1.GET("/music/playlists/:id", api.PublicPlaylistByID)
	v1.GET("/music/tracks/:id/playback", api.Playback)
	v1.GET("/music/tracks/:id/lyrics", api.Lyrics)
	v1.GET("/music/tracks/:id/cover", api.Cover)
	v1.GET("/visitor", api.Visitor)
	v1.POST("/visitor", middleware.BodyLimit(2<<20), api.Visitor)

	admin := v1.Group("/admin", middleware.NoStore())
	admin.POST("/login", middleware.BodyLimit(64<<10), api.Login)
	admin.Use(middleware.RequireAuth(api.auth, sessionCookie))
	adminJSON := admin.Group("", middleware.BodyLimit(2<<20))
	adminJSON.POST("/logout", api.Logout)
	adminJSON.GET("/session", api.Session)
	adminJSON.GET("/profile", api.Profile)
	adminJSON.PUT("/profile", api.UpdateProfile)
	adminJSON.GET("/navigation", api.AdminNavigation)
	adminJSON.POST("/navigation", api.CreateNavigation)
	adminJSON.PUT("/navigation/:id", api.UpdateNavigation)
	adminJSON.DELETE("/navigation/:id", api.DeleteNavigation)
	adminJSON.GET("/devices", api.AdminDevices)
	adminJSON.POST("/devices", api.CreateDevice)
	adminJSON.PUT("/devices/:id", api.UpdateDevice)
	adminJSON.DELETE("/devices/:id", api.DeleteDevice)
	adminJSON.GET("/guestbook", api.AdminGuestbook)
	adminJSON.PATCH("/guestbook/:id", api.UpdateGuestbook)
	adminJSON.DELETE("/guestbook/:id", api.DeleteGuestbook)
	adminJSON.GET("/settings", api.Settings)
	adminJSON.PUT("/settings", api.UpdateSettings)
	musicAdmin := adminJSON.Group("/music", middleware.BodyLimit(1<<20))
	musicAdmin.POST("/tracks/list", api.AdminTracks)
	musicAdmin.POST("/tracks/create", api.CreateTrack)
	musicAdmin.POST("/tracks/update", api.UpdateTrack)
	musicAdmin.POST("/tracks/delete", api.DeleteTrack)
	musicAdmin.POST("/playlists/list", api.AdminPlaylists)
	musicAdmin.POST("/playlists/create", api.CreatePlaylist)
	musicAdmin.POST("/playlists/update", api.UpdatePlaylist)
	musicAdmin.POST("/playlists/delete", api.DeletePlaylist)
	musicAdmin.POST("/playlists/tracks", api.SetPlaylistTracks)
	musicAdmin.POST("/netease/config/read", api.NeteaseReadConfig)
	musicAdmin.POST("/netease/config/save", api.NeteaseSaveConfig)
	musicAdmin.POST("/netease/qr/create", api.NeteaseQRCreate)
	musicAdmin.POST("/netease/qr/check", api.NeteaseQRCheck)
	musicAdmin.POST("/netease/status", api.NeteaseStatus)
	musicAdmin.POST("/netease/logout", api.NeteaseLogout)
	musicAdmin.POST("/netease/search", api.NeteaseSearch)
	musicAdmin.POST("/netease/create-playlist", api.NeteaseCreatePlaylist)
	musicAdmin.POST("/netease/append-playlist", api.NeteaseAppendPlaylist)
	musicAdmin.POST("/netease/import-playlist", middleware.RequestDeadline(5*time.Minute), api.NeteaseImportPlaylist)
	musicAdmin.POST("/s3/config/read", api.S3ReadConfig)
	musicAdmin.POST("/s3/config/save", api.S3SaveConfig)
	musicAdmin.POST("/s3/test", api.S3Test)
	musicAdmin.POST("/s3/objects/list", api.S3Objects)
	musicAdmin.POST("/s3/objects/delete", api.S3DeleteObject)
	musicAdmin.POST("/s3/import", api.S3Import)
	musicAdmin.POST("/s3/append-playlist", api.S3AppendPlaylist)
	admin.POST("/uploads/image", middleware.BodyLimit(cfg.Upload.MaxImageBytes+(1<<20)), api.UploadImage)
	admin.POST("/uploads/music", middleware.RequestDeadline(5*time.Minute), middleware.BodyLimit(cfg.Upload.MaxMusicBytes+(1<<20)), api.UploadMusic)
	admin.POST("/uploads/lyrics", middleware.BodyLimit((1<<20)+(64<<10)), api.UploadLyrics)

	assets, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		return nil, err
	}
	router.NoMethod(func(c *gin.Context) {
		fail(c, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持")
	})
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/storage/") {
			fail(c, http.StatusNotFound, "not_found", "接口或资源不存在")
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+c.Request.URL.Path), "/")
		if name != "" && name != "." {
			if info, err := fs.Stat(assets, name); err == nil && !info.IsDir() {
				serveEmbeddedFile(c, assets, name)
				return
			}
			if path.Ext(name) != "" {
				fail(c, http.StatusNotFound, "not_found", "接口或资源不存在")
				return
			}
		}
		if _, err := fs.Stat(assets, "index.html"); err != nil {
			fail(c, http.StatusNotFound, "not_found", "前端资源尚未构建")
			return
		}
		serveEmbeddedFile(c, assets, "index.html")
	})
	return router, nil
}

func serveEmbeddedFile(c *gin.Context, assets fs.FS, name string) {
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else if name == "index.html" {
		c.Header("Cache-Control", "no-cache")
	}
	if !compressibleAsset(name) {
		http.ServeFileFS(c.Writer, c.Request, assets, name)
		return
	}
	c.Writer.Header().Add("Vary", "Accept-Encoding")
	if c.GetHeader("Range") != "" || !acceptsGzip(c.GetHeader("Accept-Encoding")) {
		http.ServeFileFS(c.Writer, c.Request, assets, name)
		return
	}
	content, err := fs.ReadFile(assets, name)
	if err != nil {
		fail(c, http.StatusNotFound, "not_found", "资源不存在")
		return
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(content); err != nil {
		_ = writer.Close()
		handleError(c, err)
		return
	}
	if err := writer.Close(); err != nil {
		handleError(c, err)
		return
	}
	c.Header("Content-Encoding", "gzip")
	c.Header("Content-Length", strconv.Itoa(compressed.Len()))
	c.Data(http.StatusOK, assetContentType(name, content), compressed.Bytes())
}

func compressibleAsset(name string) bool {
	mediaType, _, err := mime.ParseMediaType(assetContentType(name, nil))
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/javascript" ||
		mediaType == "application/json" || mediaType == "application/xml" || mediaType == "image/svg+xml"
}

func assetContentType(name string, content []byte) string {
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		return contentType
	}
	return http.DetectContentType(content)
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, item := range strings.Split(header, ",") {
		parts := strings.Split(item, ";")
		coding := strings.ToLower(strings.TrimSpace(parts[0]))
		if coding != "gzip" && coding != "*" {
			continue
		}
		quality := 1.0
		for _, parameter := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				quality = 0
			} else {
				quality = parsed
			}
		}
		if coding == "gzip" {
			return quality > 0
		}
		wildcard = quality > 0
	}
	return wildcard
}

func staticFile(root, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("filename")
		if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || strings.Contains(name, "..") {
			fail(c, http.StatusBadRequest, "invalid_path", "文件路径无效")
			return
		}
		base, err := filepath.Abs(filepath.Join(root, kind))
		if err != nil {
			handleError(c, err)
			return
		}
		filename := filepath.Join(base, name)
		resolvedBase, err := filepath.EvalSymlinks(base)
		if err != nil {
			fail(c, http.StatusNotFound, "not_found", "文件不存在")
			return
		}
		resolvedFile, err := filepath.EvalSymlinks(filename)
		if err != nil {
			fail(c, http.StatusNotFound, "not_found", "文件不存在")
			return
		}
		rel, err := filepath.Rel(resolvedBase, resolvedFile)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			fail(c, http.StatusBadRequest, "invalid_path", "文件路径无效")
			return
		}
		info, err := os.Stat(resolvedFile)
		if err != nil || !info.Mode().IsRegular() {
			fail(c, http.StatusNotFound, "not_found", "文件不存在")
			return
		}
		c.File(resolvedFile)
	}
}
