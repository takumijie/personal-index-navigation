package database

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"homepage/internal/model"
)

func TestOpenEnablesWALAndSecuresDatabaseFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secure.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Row().Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("journal_mode = %q，预期 wal", journalMode)
	}
	for _, name := range []string{path, path + "-wal", path + "-shm"} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Errorf("%s 权限为 %o，预期 600", name, info.Mode().Perm())
		}
	}
}

func TestOpenRedactsSQLParametersFromLogs(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = writer
	db, openErr := Open(filepath.Join(t.TempDir(), "logging.db"))
	os.Stderr = originalStderr
	if openErr != nil {
		_ = writer.Close()
		_ = reader.Close()
		t.Fatal(openErr)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	const secret = "test-secret-that-must-not-appear"
	if err := db.Exec("SELECT * FROM missing_table WHERE secret = ?", secret).Error; err == nil {
		t.Fatal("测试 SQL 应返回错误")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	logged, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if len(logged) == 0 {
		t.Fatal("没有捕获到 SQL 错误日志")
	}
	if strings.Contains(string(logged), secret) {
		t.Fatalf("GORM 日志泄露 SQL 参数: %s", logged)
	}
}

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
	if err := db.Exec(`CREATE TABLE playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, sort INTEGER NOT NULL DEFAULT 0,
		enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE playlist_tracks (
		playlist_id INTEGER NOT NULL, track_id INTEGER NOT NULL, sort INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL, PRIMARY KEY(playlist_id, track_id)
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE site_settings (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`).Error; err != nil {
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

func TestPlaylistNameMigrationDeduplicatesAndConstrainsNames(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "playlists.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec(`CREATE TABLE playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, sort INTEGER NOT NULL DEFAULT 0,
		enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE playlist_tracks (
		playlist_id INTEGER NOT NULL, track_id INTEGER NOT NULL, sort INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL, PRIMARY KEY(playlist_id, track_id)
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE site_settings (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations(version) VALUES (1), (2)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO playlists(id, name, created_at, updated_at) VALUES
		(1, ' Daily ', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		(2, 'daily', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		(3, 'DAILY', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		(4, 'Other', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO playlist_tracks(playlist_id, track_id, created_at) VALUES (1, 1, CURRENT_TIMESTAMP), (2, 2, CURRENT_TIMESTAMP), (3, 3, CURRENT_TIMESTAMP), (4, 4, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var playlists []model.Playlist
	if err := db.Order("id").Find(&playlists).Error; err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 2 || playlists[0].ID != 1 || playlists[0].Name != "Daily" || playlists[1].ID != 4 {
		t.Fatalf("重复列表清理结果错误: %+v", playlists)
	}
	var relationCount int64
	if err := db.Table("playlist_tracks").Count(&relationCount).Error; err != nil || relationCount != 2 {
		t.Fatalf("重复列表关联应一并清理: count=%d, %v", relationCount, err)
	}
	if err := db.Exec("INSERT INTO playlists(name, created_at, updated_at) VALUES (' daily ', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)").Error; err == nil {
		t.Fatal("唯一索引不应允许空白和大小写变体重名")
	}
}
