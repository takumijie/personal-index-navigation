package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"homepage/internal/model"

	"gorm.io/gorm"
)

func TestDeleteMusicRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	uploads := NewUploadService(root, 1<<20, 10<<20)
	validName := "0123456789abcdef0123456789abcdef.mp3"
	sentinel := filepath.Join(root, "sentinel.mp3")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalidPaths := []string{
		"/storage/music/../sentinel.mp3",
		"/storage/music/nested/" + validName,
		`/storage/music\` + validName,
		"/storage/music/" + validName + "?download=1",
		"https://example.com/storage/music/" + validName,
		"/storage/lyrics/0123456789abcdef0123456789abcdef.lrc",
		"/storage/music/not-an-upload.mp3",
	}
	for _, rawURL := range invalidPaths {
		if err := uploads.DeleteMusic(rawURL); err == nil {
			t.Errorf("非法路径应被拒绝: %q", rawURL)
		}
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "keep" {
		t.Fatalf("非法删除影响了哨兵文件: %q, %v", data, err)
	}
}

func TestContentDeleteTrackCleansResourcesAndAssociations(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	content := NewContentService(repo, NewUploadService(root, 1<<20, 10<<20))
	musicDir := filepath.Join(root, "music")
	if err := os.MkdirAll(musicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	audioURL := "/storage/music/0123456789abcdef0123456789abcdef.mp3"
	audioPath := filepath.Join(musicDir, "0123456789abcdef0123456789abcdef.mp3")
	writeAudio := func() {
		t.Helper()
		if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	saveTrack := func(track model.MusicTrack) model.MusicTrack {
		t.Helper()
		if track.PlaylistIDs == nil {
			track.PlaylistIDs = []uint{}
		}
		if err := repo.SaveTrack(&track); err != nil {
			t.Fatal(err)
		}
		return track
	}
	first := model.Playlist{Name: "First", Enabled: true}
	second := model.Playlist{Name: "Second", Enabled: true}
	if err := repo.SavePlaylist(&first); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePlaylist(&second); err != nil {
		t.Fatal(err)
	}

	writeAudio()
	local := saveTrack(model.MusicTrack{Title: "Local", URL: audioURL, Source: "local", Enabled: true, PlaylistIDs: []uint{first.ID, second.ID}})
	if err := content.DeleteTrack(local.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(audioPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("本地音频未删除: %v", err)
	}
	if _, err := repo.TrackByID(local.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("本地曲目未删除: %v", err)
	}
	for _, id := range []uint{first.ID, second.ID} {
		playlist, err := repo.PlaylistByID(id, false)
		if err != nil || len(playlist.Tracks) != 0 {
			t.Fatalf("播放列表关联未清理: %+v, %v", playlist, err)
		}
	}

	writeAudio()
	original := saveTrack(model.MusicTrack{Title: "Original", URL: audioURL, Source: "local", Enabled: true})
	shared := saveTrack(model.MusicTrack{Title: "Shared", URL: audioURL, Source: "local", Enabled: true})
	if err := content.DeleteTrack(shared.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(audioPath); err != nil {
		t.Fatalf("共享 URL 的音频不应删除: %v", err)
	}
	if err := os.Remove(audioPath); err != nil {
		t.Fatal(err)
	}
	if err := content.DeleteTrack(original.ID); err != nil {
		t.Fatalf("文件缺失时应可幂等删除曲目: %v", err)
	}

	writeAudio()
	remote := saveTrack(model.MusicTrack{Title: "Remote", URL: audioURL, Source: "remote", Enabled: true})
	netease := saveTrack(model.MusicTrack{Title: "Netease", URL: audioURL, Source: "netease", ExternalID: "123", Enabled: true})
	for _, track := range []model.MusicTrack{remote, netease} {
		if err := content.DeleteTrack(track.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(audioPath); err != nil {
			t.Fatalf("%s 来源不应删除本地文件: %v", track.Source, err)
		}
	}

	s3Track := saveTrack(model.MusicTrack{Title: "S3", Source: "s3", ObjectKey: "music/song.mp3", Enabled: true})
	if err := content.DeleteTrack(s3Track.ID); err == nil {
		t.Fatal("通用接口不应删除 S3 曲目")
	}
	if _, err := repo.TrackByID(s3Track.ID); err != nil {
		t.Fatalf("被拒绝的 S3 曲目不应改变: %v", err)
	}

	invalid := saveTrack(model.MusicTrack{Title: "Invalid", URL: "/storage/music/../sentinel.mp3", Source: "local", Enabled: true})
	if err := content.DeleteTrack(invalid.ID); err == nil {
		t.Fatal("非法本地路径不应删除数据库记录")
	}
	if _, err := repo.TrackByID(invalid.ID); err != nil {
		t.Fatalf("非法路径删除失败后曲目应保留: %v", err)
	}

	failedURL := "/storage/music/fedcba9876543210fedcba9876543210.flac"
	if err := os.Mkdir(filepath.Join(musicDir, "fedcba9876543210fedcba9876543210.flac"), 0o755); err != nil {
		t.Fatal(err)
	}
	failed := saveTrack(model.MusicTrack{Title: "Failed", URL: failedURL, Source: "local", Enabled: true})
	if err := content.DeleteTrack(failed.ID); err == nil {
		t.Fatal("非普通文件删除应失败")
	}
	if _, err := repo.TrackByID(failed.ID); err != nil {
		t.Fatalf("文件删除失败后数据库记录应保留: %v", err)
	}
}
