package handler

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"homepage/internal/config"
	"homepage/internal/middleware"
	"homepage/web"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, api *API) (*gin.Engine, error) {
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
	maxBody := cfg.Upload.MaxMusicBytes
	if cfg.Upload.MaxImageBytes > maxBody {
		maxBody = cfg.Upload.MaxImageBytes
	}
	router.Use(middleware.BodyLimit(maxBody + (1 << 20)))

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
	v1.POST("/guestbook", api.CreateGuestbook)
	v1.GET("/music/tracks", api.Tracks)
	v1.GET("/music/playlist", api.Playlist)
	v1.GET("/music/playlists", api.Playlists)
	v1.GET("/music/playlists/:id", api.PublicPlaylistByID)
	v1.GET("/music/tracks/:id/playback", api.Playback)
	v1.GET("/music/tracks/:id/lyrics", api.Lyrics)
	v1.GET("/music/tracks/:id/cover", api.Cover)
	v1.GET("/visitor", api.Visitor)
	v1.POST("/visitor", api.Visitor)

	admin := v1.Group("/admin")
	admin.POST("/login", api.Login)
	admin.Use(middleware.RequireAuth(api.auth, sessionCookie))
	admin.POST("/logout", api.Logout)
	admin.GET("/session", api.Session)
	admin.GET("/profile", api.Profile)
	admin.PUT("/profile", api.UpdateProfile)
	admin.GET("/navigation", api.AdminNavigation)
	admin.POST("/navigation", api.CreateNavigation)
	admin.PUT("/navigation/:id", api.UpdateNavigation)
	admin.DELETE("/navigation/:id", api.DeleteNavigation)
	admin.GET("/devices", api.AdminDevices)
	admin.POST("/devices", api.CreateDevice)
	admin.PUT("/devices/:id", api.UpdateDevice)
	admin.DELETE("/devices/:id", api.DeleteDevice)
	admin.GET("/guestbook", api.AdminGuestbook)
	admin.PATCH("/guestbook/:id", api.UpdateGuestbook)
	admin.DELETE("/guestbook/:id", api.DeleteGuestbook)
	admin.GET("/settings", api.Settings)
	admin.PUT("/settings", api.UpdateSettings)
	musicAdmin := admin.Group("/music", middleware.BodyLimit(1<<20))
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
	musicAdmin.POST("/netease/import-playlist", api.NeteaseImportPlaylist)
	musicAdmin.POST("/s3/config/read", api.S3ReadConfig)
	musicAdmin.POST("/s3/config/save", api.S3SaveConfig)
	musicAdmin.POST("/s3/test", api.S3Test)
	musicAdmin.POST("/s3/objects/list", api.S3Objects)
	musicAdmin.POST("/s3/objects/delete", api.S3DeleteObject)
	musicAdmin.POST("/s3/import", api.S3Import)
	musicAdmin.POST("/s3/append-playlist", api.S3AppendPlaylist)
	admin.POST("/uploads/image", api.UploadImage)
	admin.POST("/uploads/music", api.UploadMusic)
	admin.POST("/uploads/lyrics", api.UploadLyrics)

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
				http.ServeFileFS(c.Writer, c.Request, assets, name)
				return
			}
		}
		if _, err := fs.Stat(assets, "index.html"); err != nil {
			fail(c, http.StatusNotFound, "not_found", "前端资源尚未构建")
			return
		}
		c.Header("Cache-Control", "no-cache")
		http.ServeFileFS(c.Writer, c.Request, assets, "index.html")
	})
	return router, nil
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
