package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

func openTestRouter(t *testing.T, cfg config.Config, db *gorm.DB) testApp {
	t.Helper()
	repo := repository.New(db)
	api := NewAPI(
		service.NewContentService(repo),
		service.NewAuthService(repo),
		service.NewUploadService(cfg.Storage.Path, cfg.Upload.MaxImageBytes, cfg.Upload.MaxMusicBytes),
	)
	router, err := NewRouter(cfg, api)
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
	if persistedProfile.Name != "Persisted Name" || persistedSettings.SiteTitle != "Persisted site" || persistedSettings.GuestbookMaxLength != 123 {
		t.Fatal("SQLite 重开后资料或设置未保留")
	}

	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/navigation/%d", navigation.ID), nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/navigation/%d", navigation.ID), nil, cookie, http.StatusNotFound)
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/devices/%d", device.ID), nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/logout", nil, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodGet, "/api/v1/admin/session", nil, cookie, http.StatusUnauthorized)
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

	track := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks", model.MusicTrack{
		Title: "Silence", Artist: "Test", Cover: imageURL, URL: musicURL, Source: "local", Sort: 1, Enabled: true,
	}, cookie, http.StatusOK))
	hidden := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks", model.MusicTrack{
		Title: "Hidden", URL: "https://example.com/hidden.mp3", Source: "remote", Sort: 2, Enabled: false,
	}, cookie, http.StatusOK))
	playlist := responseData[[]model.MusicTrack](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusOK))
	if len(playlist) != 1 || playlist[0].ID != track.ID {
		t.Fatalf("播放列表包含错误曲目: %+v", playlist)
	}
	track.Title = "Updated silence"
	track = responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPut, fmt.Sprintf("/api/v1/admin/music/tracks/%d", track.ID), track, cookie, http.StatusOK))
	playlist = responseData[[]model.MusicTrack](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusOK))
	if len(playlist) != 1 || playlist[0].Title != "Updated silence" {
		t.Fatal("曲目修改未同步到播放列表")
	}
	performJSON(t, app.router, http.MethodDelete, fmt.Sprintf("/api/v1/admin/music/tracks/%d", hidden.ID), nil, cookie, http.StatusOK)

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
