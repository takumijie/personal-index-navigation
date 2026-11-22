package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"homepage/internal/database"
	"homepage/internal/model"
	"homepage/internal/repository"
)

func TestNeteaseProviderLifecycle(t *testing.T) {
	var qrCode atomic.Int32
	qrCode.Store(801)
	var lyricCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login/qr/key":
			fmt.Fprint(w, `{"code":200,"data":{"unikey":"upstream-secret-key"}}`)
		case "/login/qr/create":
			fmt.Fprint(w, `{"code":200,"data":{"qrimg":"data:image/png;base64,cXE="}}`)
		case "/login/qr/check":
			if r.URL.Query().Get("key") != "upstream-secret-key" {
				t.Error("二维码上游 Key 未正确传递")
			}
			code := qrCode.Load()
			cookie := ""
			if code == 803 {
				cookie = `,"cookie":"MUSIC_U=private-cookie; Path=/"`
			}
			fmt.Fprintf(w, `{"code":%d,"message":"state"%s}`, code, cookie)
		case "/login/status":
			if r.Method != http.MethodPost || r.Header.Get("Cookie") == "" {
				t.Error("登录状态未携带服务端 Cookie")
			}
			fmt.Fprint(w, `{"code":200,"data":{"profile":{"userId":7,"nickname":"Tester","avatarUrl":"https://example.com/avatar.jpg"}}}`)
		case "/logout":
			fmt.Fprint(w, `{"code":200}`)
		case "/search":
			fmt.Fprint(w, `{"code":200,"result":{"songCount":1,"songs":[{"id":347230,"name":"晴天","ar":[{"name":"周杰伦"}],"al":{"name":"叶惠美","picUrl":"https://example.com/cover.jpg"}}]}}`)
		case "/playlist/detail":
			fmt.Fprint(w, `{"code":200,"playlist":{"name":"Imported","trackIds":[{"id":347230}]}}`)
		case "/playlist/track/all":
			fmt.Fprint(w, `{"code":200,"songs":[{"id":347230,"name":"晴天","artists":[{"name":"周杰伦"}],"album":{"name":"叶惠美","picUrl":"https://example.com/legacy-cover.jpg"}}]}`)
		case "/song/url/v1":
			fmt.Fprint(w, `{"code":200,"data":[{"url":"https://example.com/song.mp3","expi":1200}]}`)
		case "/lyric":
			lyricCalls.Add(1)
			fmt.Fprint(w, `{"code":200,"lrc":{"lyric":"[00:00.00]歌词"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	repo := providerTestRepository(t)
	provider := NewNeteaseService(repo, server.Client())
	if err := provider.SaveConfig(NeteaseConfig{BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	qr, err := provider.QRCreate(context.Background())
	if err != nil || qr.SessionID == "" || qr.QRCodeBase64 == "" {
		t.Fatalf("二维码创建失败: %+v, %v", qr, err)
	}
	state, err := provider.QRCheck(context.Background(), qr.SessionID)
	if err != nil || state.Status != "waiting" {
		t.Fatalf("二维码等待状态错误: %+v, %v", state, err)
	}
	qrCode.Store(803)
	state, err = provider.QRCheck(context.Background(), qr.SessionID)
	if err != nil || state.Status != "authenticated" {
		t.Fatalf("二维码成功状态错误: %+v, %v", state, err)
	}
	status, err := provider.Status(context.Background())
	if err != nil || !status.Authenticated || status.Profile == nil || status.Profile.Nickname != "Tester" {
		t.Fatalf("登录状态错误: %+v, %v", status, err)
	}
	search, err := provider.Search(context.Background(), "晴天", 1, 20)
	if err != nil || len(search.Tracks) != 1 || search.Tracks[0].Artist != "周杰伦" || search.Tracks[0].ExternalID != "347230" || search.Tracks[0].Cover != "https://example.com/cover.jpg" {
		t.Fatalf("搜索转换错误: %+v, %v", search, err)
	}
	playlist, duplicates, err := provider.ImportPlaylist(context.Background(), "https://music.163.com/playlist?id=123", "")
	if err != nil || len(playlist.Tracks) != 1 || duplicates != 0 || playlist.Name != "Imported" || playlist.Tracks[0].Cover != "https://example.com/legacy-cover.jpg" {
		t.Fatalf("歌单导入错误: %+v, duplicates=%d, %v", playlist, duplicates, err)
	}
	playlist.Tracks[0].Cover = ""
	if err := repo.SaveTrack(&playlist.Tracks[0]); err != nil {
		t.Fatal(err)
	}
	second := search.Tracks[0]
	second.ID = 0
	second.ExternalID = "347231"
	second.Title = "七里香"
	updated, imported, skipped, err := provider.AppendToPlaylist(playlist.ID, []model.MusicTrack{search.Tracks[0], second, second})
	if err != nil || imported != 1 || skipped != 2 || len(updated.Tracks) != 2 || updated.Tracks[0].ExternalID != "347230" || updated.Tracks[1].ExternalID != "347231" || updated.Tracks[0].Cover != "https://example.com/cover.jpg" {
		t.Fatalf("追加列表或封面补全错误: imported=%d, skipped=%d, %+v, %v", imported, skipped, updated, err)
	}
	updated, imported, skipped, err = provider.AppendToPlaylist(playlist.ID, []model.MusicTrack{second})
	if err != nil || imported != 0 || skipped != 1 || len(updated.Tracks) != 2 {
		t.Fatalf("重复追加未正确跳过: imported=%d, skipped=%d, %+v, %v", imported, skipped, updated, err)
	}
	if _, _, _, err := provider.AppendToPlaylist(playlist.ID+1000, []model.MusicTrack{second}); err == nil {
		t.Fatal("向不存在的列表追加歌曲应失败")
	}
	if _, _, err := provider.CreateFromTracks(" imported ", 0, search.Tracks); err == nil || err.Error() != "播放列表名称已存在" {
		t.Fatalf("网易云建单未阻止重名: %v", err)
	}
	playback, err := provider.Playback(context.Background(), playlist.Tracks[0])
	if err != nil || playback.URL != "https://example.com/song.mp3" || playback.ExpiresAt == nil {
		t.Fatalf("播放地址错误: %+v, %v", playback, err)
	}
	for range 2 {
		text, err := provider.Lyrics(context.Background(), playlist.Tracks[0])
		if err != nil || text != "[00:00.00]歌词" {
			t.Fatalf("歌词错误: %q, %v", text, err)
		}
	}
	if lyricCalls.Load() != 1 {
		t.Fatalf("歌词缓存未生效，请求 %d 次", lyricCalls.Load())
	}
}

func TestNeteaseCreateFromTracksBackfillsOnlyEmptyCover(t *testing.T) {
	repo := providerTestRepository(t)
	provider := NewNeteaseService(repo, nil)
	track := model.MusicTrack{
		Title: "晴天", Artist: "周杰伦", Source: neteaseProvider, ExternalID: "347230",
		Cover: "https://example.com/first.jpg", Enabled: true,
	}
	first, _, err := provider.CreateFromTracks("First", 0, []model.MusicTrack{track})
	if err != nil {
		t.Fatal(err)
	}
	first.Tracks[0].Cover = ""
	if err := repo.SaveTrack(&first.Tracks[0]); err != nil {
		t.Fatal(err)
	}
	second, _, err := provider.CreateFromTracks("Second", 0, []model.MusicTrack{track})
	if err != nil || len(second.Tracks) != 1 || second.Tracks[0].Cover != track.Cover {
		t.Fatalf("导入复用旧曲目时应补全空封面: %+v, %v", second, err)
	}
	second.Tracks[0].Cover = "https://example.com/custom.jpg"
	if err := repo.SaveTrack(&second.Tracks[0]); err != nil {
		t.Fatal(err)
	}
	track.Cover = "https://example.com/replacement.jpg"
	third, _, err := provider.CreateFromTracks("Third", 0, []model.MusicTrack{track})
	if err != nil || len(third.Tracks) != 1 || third.Tracks[0].Cover != "https://example.com/custom.jpg" {
		t.Fatalf("导入不应覆盖已有封面: %+v, %v", third, err)
	}
}

func TestNeteaseConfigValidationPersistenceAndCredentialReset(t *testing.T) {
	repo := providerTestRepository(t)
	provider := NewNeteaseService(repo, nil)
	initial, err := provider.ReadConfig()
	if err != nil || initial.BaseURL != "" {
		t.Fatalf("初始网易云配置应为空: %+v, %v", initial, err)
	}
	_, err = provider.Search(context.Background(), "test", 1, 20)
	var serviceErr *Error
	if !errors.As(err, &serviceErr) || serviceErr.Code != "provider_not_configured" {
		t.Fatalf("未配置时应返回明确错误: %v", err)
	}
	for _, value := range []string{"", "example.com", "ftp://example.com", "https://user@example.com", "https://example.com?x=1", "https://example.com#part"} {
		if err := provider.SaveConfig(NeteaseConfig{BaseURL: value}); err == nil {
			t.Fatalf("非法 API 地址不应保存: %q", value)
		}
	}
	if err := provider.SaveConfig(NeteaseConfig{BaseURL: " https://example.com/api/ "}); err != nil {
		t.Fatal(err)
	}
	config, err := provider.ReadConfig()
	if err != nil || config.BaseURL != "https://example.com/api" {
		t.Fatalf("API 地址规范化错误: %+v, %v", config, err)
	}
	if err := provider.saveCookie("secret"); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveConfig(NeteaseConfig{BaseURL: "https://example.com/api"}); err != nil {
		t.Fatal(err)
	}
	stored, err := provider.loadProviderConfig()
	if err != nil || stored.Cookie != "secret" {
		t.Fatalf("相同地址不应清除 Cookie: %+v, %v", stored, err)
	}
	reloaded := NewNeteaseService(repo, nil)
	stored, err = reloaded.loadProviderConfig()
	if err != nil || stored.BaseURL != config.BaseURL || stored.Cookie != "secret" {
		t.Fatalf("配置未持久化: %+v, %v", stored, err)
	}
	provider.sessions["stale"] = neteaseQRSession{}
	if err := provider.SaveConfig(NeteaseConfig{BaseURL: "https://other.example.com"}); err != nil {
		t.Fatal(err)
	}
	stored, err = provider.loadProviderConfig()
	if err != nil || stored.Cookie != "" || len(provider.sessions) != 0 {
		t.Fatalf("更换地址应清除凭证和会话: %+v, sessions=%d, %v", stored, len(provider.sessions), err)
	}
}

func providerTestRepository(t *testing.T) *repository.Repository {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return repository.New(db)
}
