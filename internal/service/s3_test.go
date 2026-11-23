package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"homepage/internal/database"
	"homepage/internal/model"
	"homepage/internal/repository"
)

func TestS3ConfigObjectsImportPlaybackAndLyrics(t *testing.T) {
	picture := testCoverPNG(t)
	audio := testMP3WithCover(t, picture)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("S3 请求缺少签名")
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>true</IsTruncated><NextContinuationToken>next-token</NextContinuationToken><Contents><Key>music/song.mp3</Key><Size>123</Size></Contents><Contents><Key>music/note.txt</Key><Size>4</Size></Contents></ListBucketResult>`)
		case r.Method == http.MethodGet && r.URL.Path == "/music/music/song.mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(audio)
		case r.Method == http.MethodGet && r.URL.Path == "/music/music/song.lrc":
			fmt.Fprint(w, "[00:00.00]S3 歌词")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	repo := providerTestRepository(t)
	provider := NewS3Service(repo, server.Client())
	initial, err := provider.ReadConfig()
	if err != nil || initial.AccessMode != "presigned" {
		t.Fatalf("未配置 S3 时应默认使用预签名模式: %+v, %v", initial, err)
	}
	config := S3Config{
		Endpoint: server.URL, Region: "auto", Bucket: "music", AccessKeyID: "access",
		SecretAccessKey: "secret", Prefix: "music", ForcePathStyle: true,
		PublicBaseURL: "https://cdn.example.com/audio", AccessMode: "public",
	}
	if err := provider.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	read, err := provider.ReadConfig()
	if err != nil || !read.HasSecret || read.AccessMode != "public" {
		t.Fatalf("S3 配置读取错误: %+v, %v", read, err)
	}
	config.SecretAccessKey = ""
	if err := provider.SaveConfig(config); err != nil {
		t.Fatalf("空白 Secret 不应覆盖原值: %v", err)
	}
	objects, err := provider.ListObjects(context.Background(), "", "")
	if err != nil || len(objects.Objects) != 1 || objects.Objects[0].Key != "music/song.mp3" || !objects.HasMore || objects.NextToken != "next-token" {
		t.Fatalf("S3 对象过滤或分页错误: %+v, %v", objects, err)
	}
	if _, _, err := provider.Import(context.Background(), "Bad", 0, []string{"other/song.mp3"}); err == nil {
		t.Fatal("配置前缀之外的对象不应允许导入")
	}
	playlist, duplicates, err := provider.Import(context.Background(), "S3 list", 0, []string{"music/song.mp3"})
	if err != nil || len(playlist.Tracks) != 1 || duplicates != 0 {
		t.Fatalf("S3 导入错误: %+v, %v", playlist, err)
	}
	if _, _, err := provider.Import(context.Background(), " s3 LIST ", 0, []string{"music/song.mp3"}); err == nil || err.Error() != "播放列表名称已存在" {
		t.Fatalf("S3 导入未阻止重名: %v", err)
	}
	playback, err := provider.Playback(context.Background(), playlist.Tracks[0])
	if err != nil || playback.URL != "https://cdn.example.com/audio/music/song.mp3" {
		t.Fatalf("公开播放地址错误: %+v, %v", playback, err)
	}
	lyrics, err := provider.Lyrics(context.Background(), playlist.Tracks[0])
	if err != nil || lyrics != "[00:00.00]S3 歌词" {
		t.Fatalf("S3 歌词错误: %q, %v", lyrics, err)
	}
	music := NewMusicService(repo, t.TempDir())
	music.RegisterProvider("s3", provider)
	cover, err := music.Cover(context.Background(), playlist.Tracks[0].ID)
	if err != nil || cover.ContentType != "image/png" || !bytes.Equal(cover.Data, picture) {
		t.Fatalf("S3 私有对象内嵌封面错误: %+v, %v", cover, err)
	}
	config.AccessMode = "presigned"
	config.SecretAccessKey = ""
	if err := provider.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Add(14 * time.Minute)
	playback, err = provider.Playback(context.Background(), model.MusicTrack{Source: "s3", ObjectKey: "music/song.mp3"})
	if err != nil || !strings.Contains(playback.URL, "X-Amz-Signature=") || playback.ExpiresAt == nil || playback.ExpiresAt.Before(before) {
		t.Fatalf("预签名播放地址错误: %+v, %v", playback, err)
	}
}

func TestS3AppendAndDeleteObject(t *testing.T) {
	deleteCalls := 0
	deleteStatus := http.StatusInternalServerError
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/music/music/song.mp3" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("S3 删除请求缺少签名")
		}
		deleteCalls++
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(deleteStatus)
		if deleteStatus == http.StatusNotFound {
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
		} else if deleteStatus >= http.StatusBadRequest {
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>InternalError</Code><Message>failed</Message></Error>`)
		}
	}))
	defer server.Close()
	repo := providerTestRepository(t)
	provider := NewS3Service(repo, server.Client())
	if err := provider.SaveConfig(S3Config{
		Endpoint: server.URL, Region: "auto", Bucket: "music", AccessKeyID: "access",
		SecretAccessKey: "secret", Prefix: "music", ForcePathStyle: true, AccessMode: "presigned",
	}); err != nil {
		t.Fatal(err)
	}
	first, _, err := provider.Import(context.Background(), "First", 0, []string{"music/song.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	second := model.Playlist{Name: "Second", Enabled: true}
	if err := repo.SavePlaylist(&second); err != nil {
		t.Fatal(err)
	}
	updated, imported, duplicates, err := provider.AppendToPlaylist(second.ID, []string{"music/song.mp3"})
	if err != nil || imported != 1 || duplicates != 0 || len(updated.Tracks) != 1 || updated.Tracks[0].ID != first.Tracks[0].ID {
		t.Fatalf("S3 追加或跨列表复用错误: imported=%d duplicates=%d %+v, %v", imported, duplicates, updated, err)
	}
	updated, imported, duplicates, err = provider.AppendToPlaylist(second.ID, []string{"music/song.mp3"})
	if err != nil || imported != 0 || duplicates != 1 || len(updated.Tracks) != 1 {
		t.Fatalf("S3 重复追加计数错误: imported=%d duplicates=%d %+v, %v", imported, duplicates, updated, err)
	}
	invalidSelections := [][]string{nil, {"music/song.mp3", "music/song.mp3"}, {"other/song.mp3"}}
	tooMany := make([]string, s3MaxObjects+1)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("music/%d.mp3", index)
	}
	invalidSelections = append(invalidSelections, tooMany)
	for _, keys := range invalidSelections {
		if _, _, _, err := provider.AppendToPlaylist(second.ID, keys); err == nil {
			t.Fatalf("非法 S3 选择应被拒绝: %v", keys)
		}
	}
	if _, _, _, err := provider.AppendToPlaylist(second.ID+1000, []string{"music/song.mp3"}); err == nil {
		t.Fatal("向不存在的列表追加 S3 对象应失败")
	}

	duplicate := model.MusicTrack{
		Title: "Historical duplicate", Source: "s3", ObjectKey: "music/song.mp3", Enabled: true,
		PlaylistIDs: []uint{second.ID},
	}
	if err := repo.SaveTrack(&duplicate); err != nil {
		t.Fatal(err)
	}
	if err := provider.DeleteObject(context.Background(), "music/song.mp3"); err == nil || !strings.Contains(err.Error(), "数据库关联已清理") {
		t.Fatalf("OSS 删除失败应返回可重试的部分失败错误: %v", err)
	}
	if deleteCalls == 0 {
		t.Fatal("未请求 OSS 删除")
	}
	tracks, err := repo.Tracks(false)
	if err != nil || len(tracks) != 0 {
		t.Fatalf("OSS 删除失败后数据库关联仍应已清理: %+v, %v", tracks, err)
	}
	for _, id := range []uint{first.ID, second.ID} {
		playlist, err := repo.PlaylistByID(id, false)
		if err != nil || len(playlist.Tracks) != 0 {
			t.Fatalf("S3 曲目的列表关联未全部清理: %+v, %v", playlist, err)
		}
	}
	deleteStatus = http.StatusNotFound
	if err := provider.DeleteObject(context.Background(), "music/song.mp3"); err != nil {
		t.Fatalf("对象不存在时应可幂等重试: %v", err)
	}
	if deleteCalls < 2 {
		t.Fatal("部分失败后重试未再次请求 OSS")
	}
	if err := provider.DeleteObject(context.Background(), "other/song.mp3"); err == nil {
		t.Fatal("删除配置前缀之外的对象应失败")
	}
}

func TestS3DeleteObjectDatabaseFailureSkipsOSS(t *testing.T) {
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	db, err := database.Open(filepath.Join(t.TempDir(), "s3-delete.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(db)
	provider := NewS3Service(repo, server.Client())
	if err := provider.SaveConfig(S3Config{
		Endpoint: server.URL, Region: "auto", Bucket: "music", AccessKeyID: "access",
		SecretAccessKey: "secret", Prefix: "music", ForcePathStyle: true, AccessMode: "presigned",
	}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	track := model.MusicTrack{Title: "S3", Source: "s3", ObjectKey: "music/song.mp3", Enabled: true, PlaylistIDs: []uint{}}
	if err := repo.SaveTrack(&track); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_s3_track_delete BEFORE DELETE ON music_tracks BEGIN SELECT RAISE(FAIL, 'forced delete failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := provider.DeleteObject(context.Background(), "music/song.mp3"); err == nil {
		t.Fatal("数据库失败应返回错误")
	}
	if deleteCalls != 0 {
		t.Fatal("数据库失败时不应请求 OSS")
	}
	if _, err := repo.TrackByID(track.ID); err != nil {
		t.Fatalf("数据库删除事务失败后曲目应保留: %v", err)
	}
}

func TestS3ClearSecretPersistenceAndRecovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	repo := providerTestRepository(t)
	provider := NewS3Service(repo, server.Client())
	config := S3Config{
		Endpoint: server.URL, Region: "auto", Bucket: "music", AccessKeyID: "access",
		SecretAccessKey: "old-secret", Prefix: "music", ForcePathStyle: true, AccessMode: "presigned",
	}
	if err := provider.SaveConfig(config); err != nil {
		t.Fatal(err)
	}

	config.SecretAccessKey = "must-not-be-persisted"
	config.ClearSecret = true
	if err := provider.SaveConfig(config); err != nil {
		t.Fatalf("显式清除 Secret 时应允许保存不完整配置: %v", err)
	}
	var stored S3Config
	if err := repo.ProviderConfig(s3Provider, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.SecretAccessKey != "" {
		t.Fatalf("显式清除后 Secret 仍被持久化: %q", stored.SecretAccessKey)
	}

	provider = NewS3Service(repo, server.Client())
	read, err := provider.ReadConfig()
	if err != nil || read.HasSecret {
		t.Fatalf("重新构造服务后不应恢复 Secret: %+v, %v", read, err)
	}
	encoded, err := json.Marshal(read)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("secret_access_key")) {
		t.Fatalf("读取 DTO 不应包含 Secret 字段: %s", encoded)
	}

	operations := map[string]func() error{
		"test": func() error { return provider.Test(context.Background()) },
		"list": func() error {
			_, err := provider.ListObjects(context.Background(), "", "")
			return err
		},
		"playback": func() error {
			_, err := provider.Playback(context.Background(), model.MusicTrack{Source: s3Provider, ObjectKey: "music/song.mp3"})
			return err
		},
	}
	for name, operation := range operations {
		err := operation()
		var serviceErr *Error
		if !errors.As(err, &serviceErr) || serviceErr.Code != "provider_not_configured" {
			t.Fatalf("清除 Secret 后 %s 应明确报告未完整配置: %v", name, err)
		}
	}

	config.SecretAccessKey = "new-secret"
	config.ClearSecret = false
	if err := provider.SaveConfig(config); err != nil {
		t.Fatalf("重新填写 Secret 后应可保存: %v", err)
	}
	if err := repo.ProviderConfig(s3Provider, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.SecretAccessKey != "new-secret" {
		t.Fatalf("重新填写的新 Secret 未持久化: %q", stored.SecretAccessKey)
	}
	read, err = provider.ReadConfig()
	if err != nil || !read.HasSecret {
		t.Fatalf("重新填写后 has_secret 应恢复: %+v, %v", read, err)
	}
	if err := provider.Test(context.Background()); err != nil {
		t.Fatalf("重新填写 Secret 后运行操作应恢复: %v", err)
	}
}
