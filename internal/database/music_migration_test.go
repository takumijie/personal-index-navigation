package database

import (
	"path/filepath"
	"testing"

	"homepage/internal/model"
)

func TestMusicSourceMigrationPreservesTracks(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec(`CREATE TABLE music_tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, artist TEXT NOT NULL DEFAULT '',
		album TEXT NOT NULL DEFAULT '', cover TEXT NOT NULL DEFAULT '', url TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT 'local', sort INTEGER NOT NULL DEFAULT 0,
		enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations(version) VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO music_tracks(title, artist, url, source, sort, enabled, created_at, updated_at) VALUES ('Legacy', 'Artist', 'https://example.com/song.mp3', 'remote', 3, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)").Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var track model.MusicTrack
	if err := db.First(&track).Error; err != nil {
		t.Fatal(err)
	}
	if track.Title != "Legacy" || track.Source != "remote" || track.Sort != 3 || !track.Enabled || track.ExternalID != "" || track.LyricPath != "" {
		t.Fatalf("旧曲目迁移结果错误: %+v", track)
	}
}
