package repository

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"homepage/internal/database"
	"homepage/internal/model"

	"gorm.io/gorm"
)

func TestPlaylistAndTrackAssociationsUseConstantQueries(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "playlist.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	playlists := []model.Playlist{
		{Name: "Second", Sort: 20, Enabled: true},
		{Name: "First", Sort: 10, Enabled: true},
		{Name: "Hidden", Sort: 0, Enabled: false},
		{Name: "Empty", Sort: 30, Enabled: true},
	}
	for i := range playlists {
		if err := db.Create(&playlists[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	tracks := []model.MusicTrack{
		{Title: "Shared", Source: "remote", URL: "https://example.com/shared.mp3", Sort: 10, Enabled: true},
		{Title: "Other", Source: "remote", URL: "https://example.com/other.mp3", Sort: 20, Enabled: true},
		{Title: "Disabled", Source: "remote", URL: "https://example.com/disabled.mp3", Sort: 0, Enabled: false},
	}
	for i := range tracks {
		if err := db.Create(&tracks[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	relations := []model.PlaylistTrack{
		{PlaylistID: playlists[1].ID, TrackID: tracks[1].ID, Sort: 2},
		{PlaylistID: playlists[1].ID, TrackID: tracks[0].ID, Sort: 1},
		{PlaylistID: playlists[1].ID, TrackID: tracks[2].ID, Sort: 0},
		{PlaylistID: playlists[0].ID, TrackID: tracks[1].ID, Sort: 0},
		{PlaylistID: playlists[2].ID, TrackID: tracks[0].ID, Sort: 0},
	}
	for i := range relations {
		if err := db.Create(&relations[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	expectedPlaylists := make([]model.Playlist, 0)
	if err := db.Where("enabled = ?", true).Order("sort, id").Find(&expectedPlaylists).Error; err != nil {
		t.Fatal(err)
	}
	for i := range expectedPlaylists {
		expectedPlaylists[i], err = repo.PlaylistByID(expectedPlaylists[i].ID, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	expectedTracks := make([]model.MusicTrack, 0)
	if err := db.Where("enabled = ?", true).Order("sort, id").Find(&expectedTracks).Error; err != nil {
		t.Fatal(err)
	}
	for i := range expectedTracks {
		expectedTracks[i].PlaylistIDs = make([]uint, 0)
		if err := db.Model(&model.PlaylistTrack{}).Where("track_id = ?", expectedTracks[i].ID).
			Order("playlist_id").Pluck("playlist_id", &expectedTracks[i].PlaylistIDs).Error; err != nil {
			t.Fatal(err)
		}
	}

	queries := 0
	const callbackName = "test:count_association_queries"
	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(*gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	actualPlaylists, err := repo.Playlists(true, true)
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 {
		t.Fatalf("播放列表查询应固定为 2 次，实际为 %d", queries)
	}
	assertSameJSON(t, expectedPlaylists, actualPlaylists)

	queries = 0
	actualTracks, err := repo.Tracks(true)
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 {
		t.Fatalf("曲目关联查询应固定为 2 次，实际为 %d", queries)
	}
	assertSameJSON(t, expectedTracks, actualTracks)
}

func assertSameJSON(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	if string(expectedJSON) != string(actualJSON) {
		t.Fatalf("批量查询结果与逐条查询不一致:\n期望: %s\n实际: %s", expectedJSON, actualJSON)
	}
}

func TestAppendPlaylistTracksRollsBackOnFailure(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "playlist.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	playlist := model.Playlist{Name: "Rollback", Enabled: true}
	if err := repo.SavePlaylist(&playlist); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_second_track BEFORE INSERT ON playlist_tracks
		WHEN NEW.sort >= 1 BEGIN SELECT RAISE(ABORT, 'forced failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	tracks := []model.MusicTrack{
		{Title: "One", Source: "netease", ExternalID: "1", Enabled: true},
		{Title: "Two", Source: "netease", ExternalID: "2", Enabled: true},
	}
	if _, _, _, err := repo.AppendPlaylistTracks(playlist.ID, tracks); err == nil {
		t.Fatal("第二条关联失败时应返回错误")
	}
	var trackCount, relationCount int64
	if err := db.Model(&model.MusicTrack{}).Count(&trackCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.PlaylistTrack{}).Count(&relationCount).Error; err != nil {
		t.Fatal(err)
	}
	if trackCount != 0 || relationCount != 0 {
		t.Fatalf("追加失败未整体回滚: tracks=%d, relations=%d", trackCount, relationCount)
	}
}
