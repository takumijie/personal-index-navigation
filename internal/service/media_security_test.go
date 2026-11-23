package service

import (
	"context"
	"errors"
	"testing"

	"homepage/internal/model"
)

type staticPlaybackProvider struct {
	url string
}

func (p staticPlaybackProvider) Playback(context.Context, model.MusicTrack) (Playback, error) {
	return Playback{URL: p.url}, nil
}

func (p staticPlaybackProvider) Lyrics(context.Context, model.MusicTrack) (string, error) {
	return "", nil
}

func TestMediaPolicyRejectsInsecureBrowserURLsWithoutRewriting(t *testing.T) {
	repo := providerTestRepository(t)
	remote := model.MusicTrack{
		Title: "Remote", Source: "remote", URL: "http://media.example.com/song.mp3",
		Cover: "http://media.example.com/cover.jpg", Enabled: true,
	}
	if err := repo.SaveTrack(&remote); err != nil {
		t.Fatal(err)
	}
	music := NewMusicService(repo, t.TempDir())
	if _, err := music.Playback(context.Background(), remote.ID); !hasErrorCode(err, "playback_unavailable") {
		t.Fatalf("默认策略应明确拒绝 HTTP 音频，实际错误: %v", err)
	}
	if _, err := music.Cover(context.Background(), remote.ID); !hasErrorCode(err, "not_found") {
		t.Fatalf("HTTP 封面应使用无封面回退，实际错误: %v", err)
	}

	music.SetMediaPolicy(MediaPolicy{AllowInsecureHTTP: true})
	playback, err := music.Playback(context.Background(), remote.ID)
	if err != nil || playback.URL != remote.URL {
		t.Fatalf("显式兼容模式应原样返回 HTTP 地址: %+v, %v", playback, err)
	}
	cover, err := music.Cover(context.Background(), remote.ID)
	if err != nil || cover.URL != remote.Cover {
		t.Fatalf("显式兼容模式应原样返回 HTTP 封面: %+v, %v", cover, err)
	}
}

func TestMediaPolicyChecksProviderPlaybackAndRemoteUpdate(t *testing.T) {
	repo := providerTestRepository(t)
	track := model.MusicTrack{Title: "Netease", Source: "netease", ExternalID: "123", Enabled: true}
	if err := repo.SaveTrack(&track); err != nil {
		t.Fatal(err)
	}
	music := NewMusicService(repo, t.TempDir())
	music.RegisterProvider("netease", staticPlaybackProvider{url: "http://media.example.com/song.mp3"})
	if _, err := music.Playback(context.Background(), track.ID); !hasErrorCode(err, "playback_unavailable") {
		t.Fatalf("Provider 返回 HTTP 地址时应明确不可播放: %v", err)
	}
	music.SetMediaPolicy(MediaPolicy{AllowInsecureHTTP: true})
	playback, err := music.Playback(context.Background(), track.ID)
	if err != nil || playback.URL != "http://media.example.com/song.mp3" {
		t.Fatalf("兼容模式不应改写 Provider 地址: %+v, %v", playback, err)
	}

	remote := model.MusicTrack{Title: "Remote", Source: "remote", URL: "https://media.example.com/old.mp3", Enabled: true}
	if err := repo.SaveTrack(&remote); err != nil {
		t.Fatal(err)
	}
	content := NewContentService(repo, nil)
	updated := remote
	updated.URL = "http://media.example.com/new.mp3"
	if _, err := content.UpdateTrack(remote.ID, updated); !hasErrorCode(err, "playback_unavailable") {
		t.Fatalf("Remote 保存应拒绝不安全地址: %v", err)
	}
	stored, err := repo.TrackByID(remote.ID)
	if err != nil || stored.URL != remote.URL {
		t.Fatalf("拒绝保存后原地址不应变化: %+v, %v", stored, err)
	}
	content.SetMediaPolicy(MediaPolicy{AllowInsecureHTTP: true})
	if _, err := content.UpdateTrack(remote.ID, updated); err != nil {
		t.Fatalf("兼容模式应允许保存 Remote 地址: %v", err)
	}
}

func hasErrorCode(err error, code string) bool {
	var serviceError *Error
	return errors.As(err, &serviceError) && serviceError.Code == code
}
