package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"homepage/internal/model"
)

func TestMusicServiceLocalCoverAndCache(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "music"), 0o755); err != nil {
		t.Fatal(err)
	}
	picture := testCoverPNG(t)
	audio := testMP3WithCover(t, picture)
	if err := os.WriteFile(filepath.Join(root, "music", "covered.mp3"), audio, 0o600); err != nil {
		t.Fatal(err)
	}
	repo := providerTestRepository(t)
	track := model.MusicTrack{Title: "Covered", Source: "local", URL: "/storage/music/covered.mp3", Enabled: true}
	if err := repo.SaveTrack(&track); err != nil {
		t.Fatal(err)
	}
	music := NewMusicService(repo, root)
	cover, err := music.Cover(context.Background(), track.ID)
	if err != nil || cover.ContentType != "image/png" || !bytes.Equal(cover.Data, picture) {
		t.Fatalf("本地内嵌封面读取错误: %+v, %v", cover, err)
	}
	if err := os.Remove(filepath.Join(root, "music", "covered.mp3")); err != nil {
		t.Fatal(err)
	}
	cached, err := music.Cover(context.Background(), track.ID)
	if err != nil || !bytes.Equal(cached.Data, picture) {
		t.Fatalf("缓存封面应在音频不可读取时仍可使用: %+v, %v", cached, err)
	}
}

func TestMusicServiceCoverFallbacksAndPathConstraint(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "music"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "music", "plain.mp3"), []byte("ID3\x04\x00\x00\x00\x00\x00\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := providerTestRepository(t)
	music := NewMusicService(repo, root)
	plain := model.MusicTrack{Title: "Plain", Source: "local", URL: "/storage/music/plain.mp3", Enabled: true}
	if err := repo.SaveTrack(&plain); err != nil {
		t.Fatal(err)
	}
	if _, err := music.Cover(context.Background(), plain.ID); err == nil {
		t.Fatal("无内嵌封面时应返回错误")
	}
	invalid := model.MusicTrack{Title: "Invalid", Source: "local", URL: "/storage/music/../secret.mp3", Enabled: true}
	if err := repo.SaveTrack(&invalid); err != nil {
		t.Fatal(err)
	}
	if _, err := music.Cover(context.Background(), invalid.ID); err == nil {
		t.Fatal("本地音乐路径越过约定目录时应被拒绝")
	}
	remote := model.MusicTrack{Title: "Remote", Source: "remote", URL: "https://example.com/song.mp3", Cover: "https://example.com/cover.jpg", Enabled: true}
	if err := repo.SaveTrack(&remote); err != nil {
		t.Fatal(err)
	}
	cover, err := music.Cover(context.Background(), remote.ID)
	if err != nil || cover.URL != remote.Cover {
		t.Fatalf("远程封面应返回重定向地址: %+v, %v", cover, err)
	}
}

func TestMusicServiceNeteaseCoverBackfillAndMissingCache(t *testing.T) {
	var foundCalls atomic.Int32
	var missingCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/song/detail" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("ids") == "347230" {
			foundCalls.Add(1)
			fmt.Fprint(w, `{"code":200,"songs":[{"id":347230,"name":"晴天","al":{"name":"叶惠美","picUrl":"https://example.com/fetched-cover.jpg"}}]}`)
			return
		}
		missingCalls.Add(1)
		fmt.Fprint(w, `{"code":200,"songs":[{"id":347231,"name":"无封面","al":{"name":"测试"}}]}`)
	}))
	defer server.Close()

	repo := providerTestRepository(t)
	provider := NewNeteaseService(repo, server.Client())
	if err := provider.SaveConfig(NeteaseConfig{BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	music := NewMusicService(repo, t.TempDir())
	music.RegisterProvider(neteaseProvider, provider)
	track := model.MusicTrack{Title: "晴天", Source: neteaseProvider, ExternalID: "347230", Enabled: true}
	if err := repo.SaveTrack(&track); err != nil {
		t.Fatal(err)
	}
	cover, err := music.Cover(context.Background(), track.ID)
	if err != nil || cover.URL != "https://example.com/fetched-cover.jpg" {
		t.Fatalf("网易云空封面应按需补全: %+v, %v", cover, err)
	}
	stored, err := repo.TrackByID(track.ID)
	if err != nil || stored.Cover != cover.URL {
		t.Fatalf("补全封面未持久化: %+v, %v", stored, err)
	}
	if _, err := music.Cover(context.Background(), track.ID); err != nil {
		t.Fatal(err)
	}
	if foundCalls.Load() != 1 {
		t.Fatalf("已有封面不应重复请求上游，请求 %d 次", foundCalls.Load())
	}

	missing := model.MusicTrack{Title: "无封面", Source: neteaseProvider, ExternalID: "347231", Enabled: true}
	if err := repo.SaveTrack(&missing); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := music.Cover(context.Background(), missing.ID); err == nil {
			t.Fatal("上游无封面时应返回错误")
		}
	}
	if missingCalls.Load() != 1 {
		t.Fatalf("无封面负缓存未生效，请求 %d 次", missingCalls.Load())
	}
}

func testCoverPNG(t *testing.T) []byte {
	t.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	var output bytes.Buffer
	if err := png.Encode(&output, picture); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func testMP3WithCover(t *testing.T, picture []byte) []byte {
	t.Helper()
	body := append([]byte{0}, []byte("image/png")...)
	body = append(body, 0, 3, 0)
	body = append(body, picture...)
	frame := new(bytes.Buffer)
	frame.WriteString("APIC")
	if err := binary.Write(frame, binary.BigEndian, uint32(len(body))); err != nil {
		t.Fatal(err)
	}
	frame.Write([]byte{0, 0})
	frame.Write(body)
	size := frame.Len()
	header := []byte{'I', 'D', '3', 3, 0, 0, byte(size >> 21 & 0x7f), byte(size >> 14 & 0x7f), byte(size >> 7 & 0x7f), byte(size & 0x7f)}
	return append(header, frame.Bytes()...)
}
