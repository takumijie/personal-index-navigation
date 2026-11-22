package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"homepage/internal/database"
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
			fmt.Fprint(w, `{"code":200,"songs":[{"id":347230,"name":"晴天","artists":[{"name":"周杰伦"}],"album":{"name":"叶惠美"}}]}`)
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
	provider, err := NewNeteaseService(repo, server.Client(), server.URL)
	if err != nil {
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
	if err != nil || len(search.Tracks) != 1 || search.Tracks[0].Artist != "周杰伦" || search.Tracks[0].ExternalID != "347230" {
		t.Fatalf("搜索转换错误: %+v, %v", search, err)
	}
	playlist, duplicates, err := provider.ImportPlaylist(context.Background(), "https://music.163.com/playlist?id=123", "")
	if err != nil || len(playlist.Tracks) != 1 || duplicates != 0 || playlist.Name != "Imported" {
		t.Fatalf("歌单导入错误: %+v, duplicates=%d, %v", playlist, duplicates, err)
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
