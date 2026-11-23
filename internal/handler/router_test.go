package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"homepage/internal/config"
	"homepage/internal/database"
	"homepage/internal/model"
	"homepage/internal/repository"
	"homepage/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type testApp struct {
	cfg    config.Config
	db     *gorm.DB
	router *gin.Engine
}

type routeDeadlineRecorder struct {
	*httptest.ResponseRecorder
	readDeadline  time.Time
	writeDeadline time.Time
}

func (r *routeDeadlineRecorder) SetReadDeadline(deadline time.Time) error {
	r.readDeadline = deadline
	return nil
}

func (r *routeDeadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.writeDeadline = deadline
	return nil
}

func newTestApp(t *testing.T) testApp {
	t.Helper()
	cfg := config.Default()
	cfg.Server.Mode = gin.TestMode
	cfg.Database.Path = filepath.Join(t.TempDir(), "homepage.db")
	cfg.Storage.Path = filepath.Join(t.TempDir(), "storage")
	cfg.Admin.Username = "admin"
	cfg.Admin.Password = "test-password"
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		t.Fatal(err)
	}
	closeDatabase(t, db)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(db, cfg); err != nil {
		t.Fatal(err)
	}
	return openTestRouter(t, cfg, db)
}

const (
	testIndexHTML    = `<!doctype html><html lang="zh-CN"><head><title>Test</title></head><body><div id="app"></div></body></html>`
	testAssetPath    = "/assets/index-a1b2c3d4.js"
	testAssetContent = `console.log("test asset");`
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                           {Data: []byte(testIndexHTML)},
		strings.TrimPrefix(testAssetPath, "/"): {Data: []byte(testAssetContent)},
	}
}

func openTestRouter(t *testing.T, cfg config.Config, db *gorm.DB) testApp {
	t.Helper()
	repo := repository.New(db)
	uploads := service.NewUploadService(cfg.Storage.Path, cfg.Upload.MaxImageBytes, cfg.Upload.MaxMusicBytes)
	api := NewAPI(
		service.NewContentService(repo, uploads),
		service.NewAuthService(repo),
		uploads,
		service.NewMusicService(repo, cfg.Storage.Path),
	)
	api.SetMusicProviders(service.NewNeteaseService(repo, nil), service.NewS3Service(repo, nil))
	router, err := NewRouterWithAssets(cfg, api, testAssets())
	if err != nil {
		t.Fatal(err)
	}
	return testApp{cfg: cfg, db: db, router: router}
}

func closeDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
}

func login(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	response := performJSON(t, router, http.MethodPost, "/api/v1/admin/login", gin.H{
		"username": "admin", "password": "test-password",
	}, nil, http.StatusOK)
	var session *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookie {
			session = cookie
		}
	}
	if session == nil || session.Value == "" || !session.HttpOnly {
		t.Fatal("登录响应没有设置 HttpOnly 会话 Cookie")
	}
	return session
}

func performJSON(t *testing.T, handler http.Handler, method, path string, body any, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s: 状态 %d，预期 %d，响应 %s", method, path, response.Code, status, response.Body.String())
	}
	return response
}

func responseData[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应失败: %v; %s", err, response.Body.String())
	}
	return envelope.Data
}

func TestSessionCookieSecureConfiguration(t *testing.T) {
	app := newTestApp(t)
	app.cfg.Server.Mode = gin.ReleaseMode
	secureRouter := openTestRouter(t, app.cfg, app.db).router
	cookie := login(t, secureRouter)
	if !cookie.Secure {
		t.Fatal("release 模式默认应设置 Secure Cookie")
	}
	response := performJSON(t, secureRouter, http.MethodPost, "/api/v1/admin/logout", nil, cookie, http.StatusOK)
	var cleared *http.Cookie
	for _, item := range response.Result().Cookies() {
		if item.Name == sessionCookie {
			cleared = item
		}
	}
	if cleared == nil || !cleared.Secure || cleared.Value != "" {
		t.Fatalf("退出 Cookie 属性与登录不一致: %+v", cleared)
	}

	secure := false
	app.cfg.Server.CookieSecure = &secure
	insecureRouter := openTestRouter(t, app.cfg, app.db).router
	if cookie := login(t, insecureRouter); cookie.Secure {
		t.Fatal("显式 HTTP 调试配置不应设置 Secure Cookie")
	}
}

func TestAdminContentLifecycleAndPersistence(t *testing.T) {
	app := newTestApp(t)
	performJSON(t, app.router, http.MethodGet, "/api/v1/admin/session", nil, nil, http.StatusUnauthorized)
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/login", gin.H{
		"username": "admin", "password": "incorrect",
	}, nil, http.StatusUnauthorized)
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/login", gin.H{
		"username": "does-not-exist", "password": "incorrect",
	}, nil, http.StatusUnauthorized)
	cookie := login(t, app.router)

	profile := responseData[model.SiteProfile](t, performJSON(t, app.router, http.MethodGet, "/api/v1/profile", nil, nil, http.StatusOK))
	profile.Name = "Persisted Name"
	profile.Avatar = "/images/avatar.webp"
	profile.Tags = []string{"Go", "Vue"}
	updatedProfile := responseData[model.SiteProfile](t, performJSON(t, app.router, http.MethodPut, "/api/v1/admin/profile", profile, cookie, http.StatusOK))
	if updatedProfile.Name != profile.Name || len(updatedProfile.Tags) != 2 {
		t.Fatalf("资料更新结果不正确: %+v", updatedProfile)
	}

	navigation := responseData[model.NavigationItem](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/navigation", model.NavigationItem{
		Title: "Hidden link", URL: "https://example.com/hidden", Icon: "Link", Sort: 99, Enabled: false, OpenMode: "blank",
	}, cookie, http.StatusOK))
	publicNavigation := responseData[[]model.NavigationItem](t, performJSON(t, app.router, http.MethodGet, "/api/v1/navigation", nil, nil, http.StatusOK))
	for _, item := range publicNavigation {
		if item.ID == navigation.ID {
			t.Fatal("停用导航出现在公开接口")
		}
	}
	navigation.Enabled = true
	navigation.Title = "Visible link"
	navigation = responseData[model.NavigationItem](t, performJSON(t, app.router, http.MethodPut, fmt.Sprintf("/api/v1/admin/navigation/%d", navigation.ID), navigation, cookie, http.StatusOK))
	if !navigation.Enabled || navigation.Title != "Visible link" {
		t.Fatalf("导航更新失败: %+v", navigation)
	}

	device := responseData[model.Device](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/devices", model.Device{
		Name: "Hidden device", Description: "test", Icon: "Laptop", Category: "computer", Sort: 99, Enabled: false,
	}, cookie, http.StatusOK))
	device.Enabled = true
	device.Name = "Visible device"
	device = responseData[model.Device](t, performJSON(t, app.router, http.MethodPut, fmt.Sprintf("/api/v1/admin/devices/%d", device.ID), device, cookie, http.StatusOK))
	if !device.Enabled || device.Name != "Visible device" {
		t.Fatalf("设备更新失败: %+v", device)
	}

	settings := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	settings.SiteTitle = "Persisted site"
	settings.GuestbookMaxLength = 123
	settings.Terminal.FastfetchLoadingMS = 1500
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", settings, cookie, http.StatusOK)

	sqlDB, err := app.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(app.cfg.Database.Path)
	if err != nil {
		t.Fatal(err)
	}
	closeDatabase(t, reopened)
	app = openTestRouter(t, app.cfg, reopened)
	cookie = login(t, app.router)

	persistedProfile := responseData[model.SiteProfile](t, performJSON(t, app.router, http.MethodGet, "/api/v1/profile", nil, nil, http.StatusOK))
	persistedSettings := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	if persistedProfile.Name != "Persisted Name" || persistedSettings.SiteTitle != "Persisted site" ||
		persistedSettings.GuestbookMaxLength != 123 || persistedSettings.Terminal.FastfetchLoadingMS != 1500 {
		t.Fatal("SQLite 重开后资料或设置未保留")
	}

	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/navigation/%d", navigation.ID), nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/navigation/%d", navigation.ID), nil, cookie, http.StatusNotFound)
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/devices/%d", device.ID), nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/logout", nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodGet, "/api/v1/admin/session", nil, cookie, http.StatusUnauthorized)
}

func TestSettingsOptimisticConflictAndCrossPanelUpdate(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	first := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	stale := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	if first.Version != 1 || stale.Version != first.Version {
		t.Fatalf("设置初始版本不正确: first=%d stale=%d", first.Version, stale.Version)
	}

	first.SiteTitle = "settings panel"
	first.ClockLabel = ""
	first.Music.LyricsEnabled = true
	updated := responseData[model.Settings](t, performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", first, cookie, http.StatusOK))
	if updated.Version != 2 || updated.ClockLabel != "" {
		t.Fatalf("设置更新响应缺少新版本或显式空字符串: %+v", updated)
	}
	stale.SiteTitle = "stale overwrite"
	stale.Music.LyricsEnabled = false
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", stale, cookie, http.StatusConflict)

	latest := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	if latest.Version != 2 || latest.SiteTitle != "settings panel" || !latest.Music.LyricsEnabled {
		t.Fatalf("冲突请求覆盖了成功方设置: %+v", latest)
	}
	latest.Music.LyricsEnabled = false
	crossPanel := responseData[model.Settings](t, performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", latest, cookie, http.StatusOK))
	if crossPanel.Version != 3 || crossPanel.SiteTitle != "settings panel" || crossPanel.Music.LyricsEnabled {
		t.Fatalf("刷新版本后的播放面板更新不正确: %+v", crossPanel)
	}
}

func TestVisitorGETReturnsRequestContext(t *testing.T) {
	app := newTestApp(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/visitor", nil)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET visitor 返回 %d: %s", response.Code, response.Body.String())
	}
	visitor := responseData[model.VisitorInfo](t, response)
	if visitor.IPAddress == "" || visitor.VisitTime.IsZero() {
		t.Fatalf("GET visitor 缺少请求上下文: %+v", visitor)
	}
}

func TestGuestbookModerationOrderingAndRateLimit(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	entries := make([]model.GuestbookEntry, 0, 2)
	for _, message := range []string{"first message", "second message"} {
		entry := responseData[model.GuestbookEntry](t, performJSON(t, app.router, http.MethodPost, "/api/v1/guestbook", service.GuestbookInput{
			Message: message, SystemName: "linux", Distribution: "linux", BrowserName: "Firefox", BrowserVersion: "1",
		}, nil, http.StatusOK))
		if entry.Status != "pending" {
			t.Fatalf("新留言状态为 %q，预期 pending", entry.Status)
		}
		entries = append(entries, entry)
	}
	if got := responseData[[]model.GuestbookEntry](t, performJSON(t, app.router, http.MethodGet, "/api/v1/guestbook", nil, nil, http.StatusOK)); len(got) != 0 {
		t.Fatal("待审核留言不应公开")
	}
	for _, entry := range entries {
		performJSON(t, app.router, http.MethodPatch, fmt.Sprintf("/api/v1/admin/guestbook/%d", entry.ID), gin.H{"status": "published"}, cookie, http.StatusOK)
	}
	published := responseData[[]model.GuestbookEntry](t, performJSON(t, app.router, http.MethodGet, "/api/v1/guestbook", nil, nil, http.StatusOK))
	if len(published) != 2 || published[0].Message != "first message" || published[1].Message != "second message" {
		t.Fatalf("公开留言顺序不是时间正序: %+v", published)
	}
	performJSON(t, app.router, http.MethodPatch, fmt.Sprintf("/api/v1/admin/guestbook/%d", entries[0].ID), gin.H{"status": "rejected"}, cookie, http.StatusOK)
	published = responseData[[]model.GuestbookEntry](t, performJSON(t, app.router, http.MethodGet, "/api/v1/guestbook", nil, nil, http.StatusOK))
	if len(published) != 1 || published[0].ID != entries[1].ID {
		t.Fatal("拒绝留言后公开列表不正确")
	}
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/guestbook/%d", entries[1].ID), nil, cookie, http.StatusOK)

	for range 3 {
		performJSON(t, app.router, http.MethodPost, "/api/v1/guestbook", service.GuestbookInput{Message: "rate limit"}, nil, http.StatusOK)
	}
	performJSON(t, app.router, http.MethodPost, "/api/v1/guestbook", service.GuestbookInput{Message: "limited"}, nil, http.StatusTooManyRequests)
}

func TestUploadsMusicPlaylistAndRange(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	imageURL := responseData[struct {
		URL string `json:"url"`
	}](t, performUpload(t, app.router, "/api/v1/admin/uploads/image", "pixel.png", png, cookie, http.StatusOK)).URL
	if imageURL == "" {
		t.Fatal("图片上传没有返回 URL")
	}
	performUpload(t, app.router, "/api/v1/admin/uploads/music", "fake.mp3", []byte("not audio"), cookie, http.StatusBadRequest)
	wav := testWAV(200)
	musicURL := responseData[struct {
		URL string `json:"url"`
	}](t, performUpload(t, app.router, "/api/v1/admin/uploads/music", "silence.wav", wav, cookie, http.StatusOK)).URL

	track := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/create", model.MusicTrack{
		Title: "Silence", Artist: "Test", Cover: imageURL, URL: musicURL, Source: "local", Sort: 1, Enabled: true,
	}, cookie, http.StatusOK))
	hidden := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/create", model.MusicTrack{
		Title: "Hidden", URL: musicURL, Source: "local", Sort: 2, Enabled: false,
	}, cookie, http.StatusOK))
	playlist := responseData[[]model.MusicTrack](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusOK))
	if len(playlist) != 1 || playlist[0].ID != track.ID {
		t.Fatalf("播放列表包含错误曲目: %+v", playlist)
	}
	track.Title = "Updated silence"
	track = responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/update", track, cookie, http.StatusOK))
	playlist = responseData[[]model.MusicTrack](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusOK))
	if len(playlist) != 1 || playlist[0].Title != "Updated silence" {
		t.Fatal("曲目修改未同步到播放列表")
	}
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/delete", gin.H{"id": hidden.ID}, cookie, http.StatusOK)

	request := httptest.NewRequest(http.MethodGet, musicURL, nil)
	request.Header.Set("Range", "bytes=0-99")
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.Len() != 100 || response.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("音频 Range 响应错误: status=%d bytes=%d headers=%v", response.Code, response.Body.Len(), response.Header())
	}
	if !bytes.Equal(response.Body.Bytes(), wav[:100]) {
		t.Fatal("音频 Range 返回了错误内容")
	}
}

func TestRouteBodyLimits(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	largeLogin := []byte(`{"username":"` + strings.Repeat("a", 64<<10) + `","password":"password"}`)
	performRawBody(t, app.router, "/api/v1/admin/login", largeLogin, "application/json", nil, false, http.StatusRequestEntityTooLarge)

	largeJSON := []byte(`{"message":"` + strings.Repeat("a", (2<<20)+1) + `"}`)
	performRawBody(t, app.router, "/api/v1/guestbook", largeJSON, "application/json", nil, true, http.StatusRequestEntityTooLarge)
	performRawBody(t, app.router, "/api/v1/admin/music/tracks/create", largeJSON, "application/json", cookie, true, http.StatusRequestEntityTooLarge)
	performUpload(t, app.router, "/api/v1/admin/uploads/lyrics", "large.lrc", make([]byte, 2<<20), cookie, http.StatusRequestEntityTooLarge)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", bytes.NewReader(make([]byte, 3<<20)))
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET 不应受 JSON 请求体限制: %d %s", response.Code, response.Body.String())
	}
}

func TestNeteaseBulkPlaylistBodyLimit(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	makeTracks := func(firstID int) []model.MusicTrack {
		tracks := make([]model.MusicTrack, 1000)
		for i := range tracks {
			tracks[i] = model.MusicTrack{
				Title: fmt.Sprintf("Track %d", firstID+i), Artist: "Artist", Source: "netease",
				ExternalID: fmt.Sprint(firstID + i), URL: "https://example.com/song.mp3",
				Cover: "https://example.com/" + strings.Repeat("c", 2000), Enabled: true,
			}
		}
		return tracks
	}
	payload := gin.H{"name": "Bulk import", "tracks": makeTracks(1)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) <= 2<<20 || len(encoded) >= 8<<20 {
		t.Fatalf("测试请求体大小 %d 不在 2 至 8 MiB 之间", len(encoded))
	}
	created := responseData[struct {
		Playlist model.Playlist `json:"playlist"`
		Imported int            `json:"imported"`
	}](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/netease/create-playlist", payload, cookie, http.StatusOK))
	if created.Imported != 1000 || len(created.Playlist.Tracks) != 1000 {
		t.Fatalf("批量创建数量错误: imported=%d tracks=%d", created.Imported, len(created.Playlist.Tracks))
	}
	appended := responseData[struct {
		Imported int `json:"imported"`
	}](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/netease/append-playlist", gin.H{
		"playlist_id": created.Playlist.ID, "tracks": makeTracks(1001),
	}, cookie, http.StatusOK))
	if appended.Imported != 1000 {
		t.Fatalf("批量追加数量错误: %d", appended.Imported)
	}
	performRawBody(t, app.router, "/api/v1/admin/music/netease/create-playlist", make([]byte, (8<<20)+1), "application/json", cookie, false, http.StatusRequestEntityTooLarge)
}

func TestNeteaseBulkPlaylistRoutesUseExtendedDeadline(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	for _, route := range []string{
		"/api/v1/admin/music/netease/create-playlist",
		"/api/v1/admin/music/netease/append-playlist",
	} {
		request := httptest.NewRequest(http.MethodPost, route, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(cookie)
		recorder := &routeDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
		started := time.Now()
		app.router.ServeHTTP(recorder, request)
		for name, deadline := range map[string]time.Time{"read": recorder.readDeadline, "write": recorder.writeDeadline} {
			remaining := deadline.Sub(started)
			if remaining < 5*time.Minute-time.Second || remaining > 5*time.Minute+time.Second {
				t.Errorf("%s 的 %s deadline 未设置为 5 分钟: %v", route, name, remaining)
			}
		}
	}
}

func TestLocalMusicRoutesUseExtendedDeadline(t *testing.T) {
	app := newTestApp(t)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(method, "/storage/music/missing.mp3", nil)
		recorder := &routeDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
		started := time.Now()
		app.router.ServeHTTP(recorder, request)
		for name, deadline := range map[string]time.Time{"read": recorder.readDeadline, "write": recorder.writeDeadline} {
			remaining := deadline.Sub(started)
			if remaining < 30*time.Minute-time.Second || remaining > 30*time.Minute+time.Second {
				t.Errorf("%s 音频的 %s deadline 未设置为 30 分钟: %v", method, name, remaining)
			}
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/storage/images/missing.png", nil)
	recorder := &routeDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	app.router.ServeHTTP(recorder, request)
	if !recorder.readDeadline.IsZero() || !recorder.writeDeadline.IsZero() {
		t.Fatal("其他静态资源不应继承音频长 deadline")
	}
}

func TestHealthAndStorageHead(t *testing.T) {
	app := newTestApp(t)
	for _, route := range []string{"/healthz", "/api/v1/health"} {
		get := performJSON(t, app.router, http.MethodGet, route, nil, nil, http.StatusOK)
		head := performJSON(t, app.router, http.MethodHead, route, nil, nil, http.StatusOK)
		if head.Body.Len() != 0 || head.Header().Get("Content-Type") != get.Header().Get("Content-Type") {
			t.Errorf("HEAD %s 响应错误: body=%d headers=%v", route, head.Body.Len(), head.Header())
		}
	}

	for _, kind := range []string{"images", "music"} {
		directory := filepath.Join(app.cfg.Storage.Path, kind)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		filename := "sample.dat"
		content := []byte("static-file-content")
		if err := os.WriteFile(filepath.Join(directory, filename), content, 0o600); err != nil {
			t.Fatal(err)
		}
		route := "/storage/" + kind + "/" + filename
		get := performJSON(t, app.router, http.MethodGet, route, nil, nil, http.StatusOK)
		head := performJSON(t, app.router, http.MethodHead, route, nil, nil, http.StatusOK)
		if head.Body.Len() != 0 || head.Header().Get("Content-Type") != get.Header().Get("Content-Type") ||
			head.Header().Get("Content-Length") != get.Header().Get("Content-Length") {
			t.Errorf("HEAD %s 与 GET 表示不一致: GET=%v HEAD=%v body=%d", route, get.Header(), head.Header(), head.Body.Len())
		}

		missing := "/storage/" + kind + "/missing.dat"
		performJSON(t, app.router, http.MethodGet, missing, nil, nil, http.StatusNotFound)
		if response := performJSON(t, app.router, http.MethodHead, missing, nil, nil, http.StatusNotFound); response.Body.Len() != 0 {
			t.Errorf("HEAD %s 不应返回响应体: %d", missing, response.Body.Len())
		}
		invalid := "/storage/" + kind + "/bad%5Cname"
		performJSON(t, app.router, http.MethodGet, invalid, nil, nil, http.StatusBadRequest)
		if response := performJSON(t, app.router, http.MethodHead, invalid, nil, nil, http.StatusBadRequest); response.Body.Len() != 0 {
			t.Errorf("HEAD %s 不应返回响应体: %d", invalid, response.Body.Len())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/storage/music/sample.dat", nil)
	request.Header.Set("Range", "bytes=0-5")
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "static" {
		t.Errorf("存储文件 Range 回归: status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestEmbeddedAssetsCachingCompressionAndSPAFallback(t *testing.T) {
	app := newTestApp(t)
	assetPath := testAssetPath
	original := []byte(testAssetContent)
	get := func(path, acceptEncoding string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if acceptEncoding != "" {
			request.Header.Set("Accept-Encoding", acceptEncoding)
		}
		response := httptest.NewRecorder()
		app.router.ServeHTTP(response, request)
		return response
	}

	response := get(assetPath, "")
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		response.Header().Get("Content-Encoding") != "" || !strings.Contains(response.Header().Get("Vary"), "Accept-Encoding") ||
		!bytes.Equal(response.Body.Bytes(), original) {
		t.Fatalf("原始 hash 资源响应错误: status=%d headers=%v", response.Code, response.Header())
	}

	response = get(assetPath, "gzip")
	if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "gzip" ||
		response.Header().Get("Content-Length") != fmt.Sprint(response.Body.Len()) {
		t.Fatalf("gzip 资源响应头错误: status=%d headers=%v", response.Code, response.Header())
	}
	gzipContentLength := response.Header().Get("Content-Length")
	headRequest := httptest.NewRequest(http.MethodHead, assetPath, nil)
	headRequest.Header.Set("Accept-Encoding", "gzip")
	headResponse := httptest.NewRecorder()
	app.router.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 ||
		headResponse.Header().Get("Content-Encoding") != "gzip" || headResponse.Header().Get("Content-Length") != gzipContentLength ||
		headResponse.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("HEAD gzip 资源响应错误: status=%d body=%d headers=%v", headResponse.Code, headResponse.Body.Len(), headResponse.Header())
	}
	reader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Fatal("gzip 解压后的资源与原始内容不一致")
	}

	response = get(assetPath, "gzip;q=0")
	if response.Header().Get("Content-Encoding") != "" || !bytes.Equal(response.Body.Bytes(), original) {
		t.Fatal("声明 gzip;q=0 的客户端不应收到压缩内容")
	}

	for _, route := range []string{"/", "/admin"} {
		response = get(route, "")
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-cache" ||
			!strings.Contains(response.Body.String(), "<html") {
			t.Fatalf("SPA 路由 %s 响应错误: status=%d headers=%v", route, response.Code, response.Header())
		}
	}
	for _, route := range []string{"/assets/does-not-exist.js", "/favicon.ico"} {
		response = get(route, "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("缺失资源 %s 应返回 404，实际 %d: %s", route, response.Code, response.Body.String())
		}
		headRequest = httptest.NewRequest(http.MethodHead, route, nil)
		headResponse = httptest.NewRecorder()
		app.router.ServeHTTP(headResponse, headRequest)
		if headResponse.Code != http.StatusNotFound || headResponse.Body.Len() != 0 {
			t.Fatalf("HEAD 缺失资源 %s 响应错误: status=%d body=%d", route, headResponse.Code, headResponse.Body.Len())
		}
	}
	headRequest = httptest.NewRequest(http.MethodHead, "/admin", nil)
	headResponse = httptest.NewRecorder()
	app.router.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 || headResponse.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("HEAD SPA 路由响应错误: status=%d body=%d headers=%v", headResponse.Code, headResponse.Body.Len(), headResponse.Header())
	}

	response = performJSON(t, app.router, http.MethodPost, "/api/v1/admin/login", gin.H{
		"username": "admin", "password": "test-password",
	}, nil, http.StatusOK)
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("登录响应应禁止缓存")
	}
	response = performJSON(t, app.router, http.MethodGet, "/api/v1/admin/session", nil, nil, http.StatusUnauthorized)
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("未认证管理响应也应禁止缓存")
	}
}

func TestLoaderBootstrapCachedAndRefreshed(t *testing.T) {
	app := newTestApp(t)
	get := func(acceptEncoding string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if acceptEncoding != "" {
			request.Header.Set("Accept-Encoding", acceptEncoding)
		}
		response := httptest.NewRecorder()
		app.router.ServeHTTP(response, request)
		return response
	}
	assertBootstrap := func(body, theme, background string) {
		t.Helper()
		for _, expected := range []string{
			`data-loader-theme="` + theme + `"`,
			`style="background:` + background + `"`,
		} {
			if !strings.Contains(body, expected) {
				t.Fatalf("首屏缺少 %q: %s", expected, body)
			}
		}
		if strings.Contains(body, "data-loader-sound-enabled") {
			t.Fatal("首屏不应再注入加载页音效设置")
		}
	}

	response := get("")
	if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "" {
		t.Fatalf("首屏原始响应错误: status=%d headers=%v", response.Code, response.Header())
	}
	assertBootstrap(response.Body.String(), "dark", "#0a0a09")

	response = get("gzip")
	if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "gzip" ||
		response.Header().Get("Content-Length") != fmt.Sprint(response.Body.Len()) {
		t.Fatalf("首屏 gzip 响应错误: status=%d headers=%v", response.Code, response.Header())
	}
	reader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	assertBootstrap(string(decompressed), "dark", "#0a0a09")

	cookie := login(t, app.router)
	settings := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	settings.Loader.Theme = "beige"
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", settings, cookie, http.StatusOK)
	response = get("")
	assertBootstrap(response.Body.String(), "beige", "#dad4bb")

	headRequest := httptest.NewRequest(http.MethodHead, "/", nil)
	headRequest.Header.Set("Accept-Encoding", "gzip")
	headResponse := httptest.NewRecorder()
	app.router.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 ||
		headResponse.Header().Get("Content-Encoding") != "gzip" || headResponse.Header().Get("Content-Length") == "" {
		t.Fatalf("首屏 HEAD gzip 响应错误: status=%d body=%d headers=%v", headResponse.Code, headResponse.Body.Len(), headResponse.Header())
	}

	sqlDB, err := app.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	response = get("")
	if response.Code != http.StatusOK {
		t.Fatalf("数据库关闭后首屏仍应从内存响应: status=%d", response.Code)
	}
	assertBootstrap(response.Body.String(), "beige", "#dad4bb")
}

func performRawBody(t *testing.T, handler http.Handler, path string, body []byte, contentType string, cookie *http.Cookie, unknownLength bool, status int) {
	t.Helper()
	var reader io.Reader = bytes.NewReader(body)
	if unknownLength {
		reader = io.NopCloser(bytes.NewReader(body))
	}
	request := httptest.NewRequest(http.MethodPost, path, reader)
	request.Header.Set("Content-Type", contentType)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("POST %s: 状态 %d，预期 %d，响应 %s", path, response.Code, status, response.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if status == http.StatusRequestEntityTooLarge {
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Error.Code != "request_too_large" {
			t.Fatalf("超限响应不统一: %s", response.Body.String())
		}
	}
}

func performUpload(t *testing.T, handler http.Handler, path, filename string, content []byte, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("上传 %s: 状态 %d，预期 %d，响应 %s", filename, response.Code, status, response.Body.String())
	}
	return response
}

func testWAV(dataBytes int) []byte {
	data := make([]byte, 44+dataBytes)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	copy(data[8:12], "WAVE")
	copy(data[12:16], "fmt ")
	binary.LittleEndian.PutUint32(data[16:20], 16)
	binary.LittleEndian.PutUint16(data[20:22], 1)
	binary.LittleEndian.PutUint16(data[22:24], 1)
	binary.LittleEndian.PutUint32(data[24:28], 8000)
	binary.LittleEndian.PutUint32(data[28:32], 16000)
	binary.LittleEndian.PutUint16(data[32:34], 2)
	binary.LittleEndian.PutUint16(data[34:36], 16)
	copy(data[36:40], "data")
	binary.LittleEndian.PutUint32(data[40:44], uint32(dataBytes))
	return data
}
