package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"homepage/internal/model"
	"homepage/internal/repository"

	"gorm.io/gorm"
)

func TestLyricResourcePathsAndRegularFiles(t *testing.T) {
	root := t.TempDir()
	uploads := NewUploadService(root, 1<<20, 10<<20)
	validName := "0123456789abcdef0123456789abcdef.lrc"
	validURL := "/storage/lyrics/" + validName
	sentinel := filepath.Join(root, "sentinel.lrc")
	writeResourceFile(t, sentinel)
	invalidPaths := []string{
		"/storage/lyrics/../sentinel.lrc",
		"/storage/lyrics/nested/" + validName,
		`/storage/lyrics\` + validName,
		validURL + "?download=1",
		"https://example.com" + validURL,
		"/storage/lyrics/0123456789ABCDEF0123456789ABCDEF.lrc",
		"/storage/lyrics/not-an-upload.lrc",
		"/storage/music/0123456789abcdef0123456789abcdef.mp3",
	}
	for _, resource := range invalidPaths {
		if err := uploads.DeleteLyrics(resource); err == nil {
			t.Errorf("非法歌词路径应被拒绝: %q", resource)
		}
		if _, err := regularLocalResourcePath(root, "lyrics", resource); err == nil {
			t.Errorf("非法歌词读取路径应被拒绝: %q", resource)
		}
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "resource" {
		t.Fatalf("非法歌词操作影响了哨兵文件: %q, %v", data, err)
	}

	lyricPath := filepath.Join(root, "lyrics", validName)
	writeResourceFile(t, lyricPath)
	repo := providerTestRepository(t)
	track := saveResourceTrack(t, repo, model.MusicTrack{
		Title: "Local", Source: "local", URL: "/storage/music/test.wav", LyricPath: validURL, Enabled: true,
	})
	music := NewMusicService(repo, root)
	lyrics, err := music.Lyrics(context.Background(), track.ID)
	if err != nil || lyrics.Text != "resource" {
		t.Fatalf("系统生成歌词读取失败: %+v, %v", lyrics, err)
	}
	if err := os.Remove(lyricPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lyricPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uploads.DeleteLyrics(validURL); err == nil {
		t.Fatal("歌词目录不应作为普通文件删除")
	}
	if _, err := music.localLyrics(validURL); err == nil {
		t.Fatal("歌词目录不应作为普通文件读取")
	}
	if err := os.Remove(lyricPath); err != nil {
		t.Fatal(err)
	}
	if _, err := music.localLyrics(validURL); err != nil {
		t.Fatalf("缺失歌词应返回空内容: %v", err)
	}
	if err := uploads.DeleteLyrics(validURL); err != nil {
		t.Fatalf("缺失歌词删除应幂等: %v", err)
	}

	linkName := "fedcba9876543210fedcba9876543210.lrc"
	linkPath := filepath.Join(root, "lyrics", linkName)
	if err := os.Symlink(sentinel, linkPath); err != nil {
		t.Logf("当前 Windows 环境不可创建符号链接，Lstat 分支由目录用例覆盖: %v", err)
	} else {
		linkURL := "/storage/lyrics/" + linkName
		if err := uploads.DeleteLyrics(linkURL); err == nil {
			t.Fatal("歌词符号链接应被拒绝")
		}
		if _, err := music.localLyrics(linkURL); err == nil {
			t.Fatal("歌词符号链接不应被读取")
		}
		if _, err := os.Stat(sentinel); err != nil {
			t.Fatalf("符号链接操作不应影响目标: %v", err)
		}
	}
}

func TestTrackLyricPathValidation(t *testing.T) {
	repo := providerTestRepository(t)
	root := t.TempDir()
	content := NewContentService(repo, NewUploadService(root, 1<<20, 10<<20))
	validURL := "/storage/lyrics/0123456789abcdef0123456789abcdef.lrc"
	track := model.MusicTrack{Title: "Local", Source: "local", URL: "/storage/music/test.wav", LyricPath: validURL, Enabled: true}
	for _, invalidPath := range []string{
		"/storage/lyrics/../secret.lrc", "/storage/lyrics/nested/0123456789abcdef0123456789abcdef.lrc",
		`/storage/lyrics\0123456789abcdef0123456789abcdef.lrc`, validURL + "?x=1", "https://example.com/song.lrc",
	} {
		candidate := track
		candidate.LyricPath = invalidPath
		if _, err := content.CreateTrack(candidate); err == nil {
			t.Errorf("创建时不应接受非法歌词路径: %q", invalidPath)
		}
	}
	created, err := content.CreateTrack(track)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lyrics", "0123456789abcdef0123456789abcdef.lrc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := content.UpdateTrack(created.ID, created); err == nil {
		t.Fatal("更新时不应接受歌词目录")
	}
	created.LyricPath = "/storage/lyrics/not-an-upload.lrc"
	if _, err := content.UpdateTrack(created.ID, created); err == nil {
		t.Fatal("更新时不应接受非法歌词路径")
	}
	stored, err := repo.TrackByID(created.ID)
	if err != nil || stored.LyricPath != validURL {
		t.Fatalf("非法更新不应改变歌词路径: %+v, %v", stored, err)
	}
}

func TestDeleteTrackSharedLyricsCoversAndSourceIsolation(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	uploads := NewUploadService(root, 1<<20, 10<<20)
	content := NewContentService(repo, uploads)
	audioURL := "/storage/music/0123456789abcdef0123456789abcdef.mp3"
	lyricURL := "/storage/lyrics/fedcba9876543210fedcba9876543210.lrc"
	audioPath := filepath.Join(root, "music", "0123456789abcdef0123456789abcdef.mp3")
	lyricPath := filepath.Join(root, "lyrics", "fedcba9876543210fedcba9876543210.lrc")
	writeResourceFile(t, audioPath)
	writeResourceFile(t, lyricPath)
	first := saveResourceTrack(t, repo, model.MusicTrack{Title: "First", Source: "local", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	second := saveResourceTrack(t, repo, model.MusicTrack{Title: "Second", Source: "local", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	firstCover := filepath.Join(root, "covers", fmt.Sprintf("%d-123.img", first.ID))
	secondCover := filepath.Join(root, "covers", fmt.Sprintf("%d-123.img", second.ID))
	lookalikes := []string{
		filepath.Join(root, "covers", fmt.Sprintf("%d-123.img.bak", first.ID)),
		filepath.Join(root, "covers", fmt.Sprintf("%d--123.img", first.ID)),
		filepath.Join(root, "covers", fmt.Sprintf("%d-abc.img", first.ID)),
		filepath.Join(root, "covers", "999999-123.img"),
	}
	for _, name := range append([]string{firstCover, secondCover}, lookalikes...) {
		writeResourceFile(t, name)
	}
	if err := content.DeleteTrack(first.ID); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, audioPath, true)
	assertResourceExists(t, lyricPath, true)
	assertResourceExists(t, firstCover, false)
	assertResourceExists(t, secondCover, true)
	if err := content.DeleteTrack(second.ID); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, audioPath, false)
	assertResourceExists(t, lyricPath, false)
	assertResourceExists(t, secondCover, false)
	for _, name := range lookalikes {
		assertResourceExists(t, name, true)
	}

	writeResourceFile(t, audioPath)
	writeResourceFile(t, lyricPath)
	local := saveResourceTrack(t, repo, model.MusicTrack{Title: "Local", Source: "local", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	remote := saveResourceTrack(t, repo, model.MusicTrack{Title: "Remote", Source: "remote", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	if err := content.DeleteTrack(local.ID); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, audioPath, true)
	assertResourceExists(t, lyricPath, true)
	if err := content.DeleteTrack(remote.ID); err != nil {
		t.Fatal(err)
	}
	netease := saveResourceTrack(t, repo, model.MusicTrack{Title: "Netease", Source: "netease", ExternalID: "123", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	if err := content.DeleteTrack(netease.ID); err != nil {
		t.Fatal(err)
	}
	s3Track := saveResourceTrack(t, repo, model.MusicTrack{Title: "S3", Source: "s3", ObjectKey: "music/song.mp3", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	if err := content.DeleteTrack(s3Track.ID); err == nil {
		t.Fatal("通用删除不应处理 S3 曲目")
	}
	assertResourceExists(t, audioPath, true)
	assertResourceExists(t, lyricPath, true)

	third := saveResourceTrack(t, repo, model.MusicTrack{Title: "Missing", Source: "local", URL: audioURL, LyricPath: lyricURL, Enabled: true})
	if err := content.DeleteTrack(third.ID); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, audioPath, true)
	assertResourceExists(t, lyricPath, true)
	if err := repo.DeleteTrack(s3Track.ID); err != nil {
		t.Fatal(err)
	}
	if err := content.DeleteTrack(third.ID + 100000); err == nil {
		t.Fatal("不存在的曲目不应删除成功")
	}
}

func TestDeleteTrackCleanupFailureDoesNotRestoreDatabase(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	content := NewContentService(repo, NewUploadService(root, 1<<20, 10<<20))
	lyricName := "0123456789abcdef0123456789abcdef.lrc"
	if err := os.MkdirAll(filepath.Join(root, "lyrics", lyricName), 0o755); err != nil {
		t.Fatal(err)
	}
	track := saveResourceTrack(t, repo, model.MusicTrack{
		Title: "Local", Source: "local", URL: "/storage/music/fedcba9876543210fedcba9876543210.mp3",
		LyricPath: "/storage/lyrics/" + lyricName, Enabled: true,
	})
	if err := content.DeleteTrack(track.ID); err != nil {
		t.Fatalf("附属歌词清理失败不应回滚数据库删除: %v", err)
	}
	if _, err := repo.TrackByID(track.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("曲目数据库记录应已删除: %v", err)
	}
	assertResourceExists(t, filepath.Join(root, "lyrics", lyricName), true)
}

func TestUpdateTrackCleansOnlyUnsharedOldResources(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	content := NewContentService(repo, NewUploadService(root, 1<<20, 10<<20))
	oldAudio := "/storage/music/0123456789abcdef0123456789abcdef.mp3"
	oldLyric := "/storage/lyrics/11111111111111111111111111111111.lrc"
	oldAudioPath := filepath.Join(root, "music", "0123456789abcdef0123456789abcdef.mp3")
	oldLyricPath := filepath.Join(root, "lyrics", "11111111111111111111111111111111.lrc")
	writeResourceFile(t, oldAudioPath)
	writeResourceFile(t, oldLyricPath)
	first := saveResourceTrack(t, repo, model.MusicTrack{Title: "First", Source: "local", URL: oldAudio, LyricPath: oldLyric, Enabled: true})
	second := saveResourceTrack(t, repo, model.MusicTrack{Title: "Second", Source: "local", URL: oldAudio, LyricPath: oldLyric, Enabled: true})
	newAudio := "/storage/music/22222222222222222222222222222222.flac"
	newLyric := "/storage/lyrics/33333333333333333333333333333333.lrc"
	updated := first
	updated.URL, updated.LyricPath = newAudio, newLyric
	if _, err := content.UpdateTrack(first.ID, updated); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, oldAudioPath, true)
	assertResourceExists(t, oldLyricPath, true)
	updated = second
	updated.URL, updated.LyricPath = newAudio, newLyric
	if _, err := content.UpdateTrack(second.ID, updated); err != nil {
		t.Fatal(err)
	}
	assertResourceExists(t, oldAudioPath, false)
	assertResourceExists(t, oldLyricPath, false)

	failedAudio := "/storage/music/44444444444444444444444444444444.mp3"
	failedLyric := "/storage/lyrics/55555555555555555555555555555555.lrc"
	failedAudioPath := filepath.Join(root, "music", "44444444444444444444444444444444.mp3")
	failedLyricPath := filepath.Join(root, "lyrics", "55555555555555555555555555555555.lrc")
	writeResourceFile(t, failedAudioPath)
	writeResourceFile(t, failedLyricPath)
	failed := saveResourceTrack(t, repo, model.MusicTrack{Title: "Failed", Source: "local", URL: failedAudio, LyricPath: failedLyric, Enabled: true})
	updated = failed
	updated.URL, updated.LyricPath = newAudio, newLyric
	updated.PlaylistIDs = []uint{999999}
	if _, err := content.UpdateTrack(failed.ID, updated); err == nil {
		t.Fatal("非法播放列表关联应使数据库更新失败")
	}
	assertResourceExists(t, failedAudioPath, true)
	assertResourceExists(t, failedLyricPath, true)
	stored, err := repo.TrackByID(failed.ID)
	if err != nil || stored.URL != failedAudio || stored.LyricPath != failedLyric {
		t.Fatalf("更新失败后旧记录不应改变: %+v, %v", stored, err)
	}
}

func TestDeleteTrackRemoveFailureCanRetry(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	uploads := NewUploadService(root, 1<<20, 10<<20)
	content := NewContentService(repo, uploads)
	audioURL := "/storage/music/0123456789abcdef0123456789abcdef.mp3"
	audioPath := filepath.Join(root, "music", "0123456789abcdef0123456789abcdef.mp3")
	writeResourceFile(t, audioPath)
	track := saveResourceTrack(t, repo, model.MusicTrack{Title: "Retry", Source: "local", URL: audioURL, Enabled: true})
	uploads.remove = func(string) error { return errors.New("injected remove failure") }
	if err := content.DeleteTrack(track.ID); err == nil {
		t.Fatal("音频删除失败应返回错误")
	}
	if _, err := repo.TrackByID(track.ID); err != nil {
		t.Fatalf("删除失败后数据库记录应保留: %v", err)
	}
	assertResourceExists(t, audioPath, true)
	uploads.remove = os.Remove
	if err := content.DeleteTrack(track.ID); err != nil {
		t.Fatalf("恢复文件删除后重试应成功: %v", err)
	}
	assertResourceExists(t, audioPath, false)
	if _, err := repo.TrackByID(track.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("重试成功后数据库记录应删除: %v", err)
	}
}

func TestTrackResourceAuditIsReadOnly(t *testing.T) {
	root := t.TempDir()
	repo := providerTestRepository(t)
	content := NewContentService(repo, NewUploadService(root, 1<<20, 10<<20))
	uniqueAudio := "/storage/music/0123456789abcdef0123456789abcdef.mp3"
	sharedAudio := "/storage/music/11111111111111111111111111111111.flac"
	sharedLyric := "/storage/lyrics/22222222222222222222222222222222.lrc"
	first := saveResourceTrack(t, repo, model.MusicTrack{Title: "First", Source: "local", URL: uniqueAudio, LyricPath: sharedLyric, Enabled: true})
	second := saveResourceTrack(t, repo, model.MusicTrack{Title: "Second", Source: "local", URL: sharedAudio, LyricPath: sharedLyric, Enabled: true})
	saveResourceTrack(t, repo, model.MusicTrack{Title: "Third", Source: "remote", URL: sharedAudio, Enabled: true})
	files := []string{
		filepath.Join(root, "music", "0123456789abcdef0123456789abcdef.mp3"),
		filepath.Join(root, "music", "11111111111111111111111111111111.flac"),
		filepath.Join(root, "lyrics", "22222222222222222222222222222222.lrc"),
		filepath.Join(root, "music", "33333333333333333333333333333333.mp3"),
		filepath.Join(root, "lyrics", "44444444444444444444444444444444.lrc"),
		filepath.Join(root, "covers", "999999-123.img"),
		filepath.Join(root, "covers", fmt.Sprintf("%d-123.img", first.ID)),
		filepath.Join(root, "music", "not-an-upload.mp3"),
		filepath.Join(root, "lyrics", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA.lrc"),
		filepath.Join(root, "covers", fmt.Sprintf("%d-123-extra.img", first.ID)),
	}
	for _, name := range files {
		writeResourceFile(t, name)
	}
	nonRegular := filepath.Join(root, "music", "55555555555555555555555555555555.mp3")
	if err := os.MkdirAll(nonRegular, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := content.AuditTrackResources()
	if err != nil {
		t.Fatal(err)
	}
	if report.Scanned != len(files)+1 || report.Referenced != 2 || len(report.Orphans) != 3 || len(report.Shared) != 2 || len(report.Invalid) != 3 || len(report.NonRegular) != 1 {
		t.Fatalf("审计分类计数错误: %+v", report)
	}
	for _, item := range report.Shared {
		if len(item.TrackIDs) != 2 || !slices.Contains(item.TrackIDs, second.ID) {
			t.Errorf("共享引用曲目列表错误: %+v", item)
		}
	}
	for _, name := range append(files, nonRegular) {
		assertResourceExists(t, name, true)
	}
}

func saveResourceTrack(t *testing.T, repo *repository.Repository, track model.MusicTrack) model.MusicTrack {
	t.Helper()
	if err := repo.SaveTrack(&track); err != nil {
		t.Fatal(err)
	}
	return track
}

func writeResourceFile(t *testing.T, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("resource"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertResourceExists(t *testing.T, name string, expected bool) {
	t.Helper()
	_, err := os.Lstat(name)
	if expected && err != nil {
		t.Fatalf("资源应保留 %s: %v", name, err)
	}
	if !expected && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("资源应删除 %s: %v", name, err)
	}
}
