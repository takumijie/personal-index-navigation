package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"homepage/internal/model"
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
