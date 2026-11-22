package repository

import (
	"path/filepath"
	"testing"

	"homepage/internal/database"
	"homepage/internal/model"
)

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
