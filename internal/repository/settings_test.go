package repository

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"homepage/internal/database"
	"homepage/internal/model"
)

func TestSettingsVersionMigrationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE site_settings (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
		INSERT INTO schema_migrations(version) VALUES (1), (2), (3);
		INSERT INTO site_settings(data, created_at, updated_at) VALUES ('{"site_title":"legacy","clock_label":"legacy"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	migrated, err := repo.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Version != 1 || migrated.SiteTitle != "legacy" {
		t.Fatalf("旧设置迁移版本不正确: %+v", migrated)
	}

	migrated.SiteTitle = "updated"
	migrated.ClockLabel = ""
	updated, err := repo.SaveSettings(migrated)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("保存后版本应为 2，实际为 %d", updated.Version)
	}
	stale := migrated
	stale.Version = 1
	stale.SiteTitle = "stale"
	if _, err := repo.SaveSettings(stale); !errors.Is(err, ErrSettingsConflict) {
		t.Fatalf("旧版本保存应冲突，实际为 %v", err)
	}
	current, err := repo.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != 2 || current.SiteTitle != "updated" || current.ClockLabel != "" {
		t.Fatalf("冲突保存覆盖了新值或未保留显式空字符串: %+v", current)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reopenedSQL, err := reopened.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopenedSQL.Close() })
	persisted, err := New(reopened).Settings()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Version != 2 || persisted.SiteTitle != "updated" {
		t.Fatalf("数据库重开后版本或设置未保留: %+v", persisted)
	}
}

func TestSettingsConcurrentCompareAndSwap(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings-concurrent.db"))
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
	if err := db.Create(&model.SiteSettings{Data: `{"site_title":"initial"}`, Version: 1}).Error; err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	first, err := repo.Settings()
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Settings()
	if err != nil {
		t.Fatal(err)
	}
	first.SiteTitle = "first"
	second.SiteTitle = "second"

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, snapshot := range []model.Settings{first, second} {
		wg.Add(1)
		go func(settings model.Settings) {
			defer wg.Done()
			<-start
			_, err := repo.SaveSettings(settings)
			results <- err
		}(snapshot)
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded, conflicted := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrSettingsConflict):
			conflicted++
		default:
			t.Fatalf("并发保存返回意外错误: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("并发保存应仅一个成功，成功 %d，冲突 %d", succeeded, conflicted)
	}
	current, err := repo.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != 2 || (current.SiteTitle != "first" && current.SiteTitle != "second") {
		t.Fatalf("并发保存结果不正确: %+v", current)
	}
}

func TestDecodeSettingsLegacyDataKeepsCardsEnabled(t *testing.T) {
	legacy := `{"site_title":"个人主页","timezone":"Asia/Shanghai","time_format":"24","clock_label":"本地时间",` +
		`"background":{"day_url":"/images/background-day.webp","night_url":"/images/background-night.webp",` +
		`"day_start":"06:00","night_start":"18:00","timezone":"Asia/Shanghai","duration_ms":1000},` +
		`"guestbook_auto_publish":false,"guestbook_max_length":500}`

	settings, err := decodeSettings(legacy)
	if err != nil {
		t.Fatalf("decodeSettings 返回错误: %v", err)
	}
	cards := settings.Cards
	if !cards.Profile || !cards.Navigation || !cards.Devices || !cards.Terminal || !cards.Clock {
		t.Fatalf("旧数据缺少 cards 字段时应默认全开，实际为 %+v", cards)
	}
	if settings.SiteTitle != "个人主页" || settings.Background.DayStart != "06:00" {
		t.Fatalf("旧字段解析不正确: %+v", settings)
	}
	if settings.Loader != (model.LoaderSettings{Theme: "dark"}) {
		t.Fatalf("旧数据缺少 loader 字段时应使用默认主题且关闭音效，实际为 %+v", settings.Loader)
	}
	if settings.Terminal.FastfetchLoadingMS != 2000 {
		t.Fatalf("旧数据缺少 terminal 字段时应使用默认停留时长，实际为 %+v", settings.Terminal)
	}
	if !settings.Music.LyricsEnabled || settings.Music.PlaylistID != 0 {
		t.Fatalf("旧数据缺少 music 字段时应默认开启歌词并自动选列表，实际为 %+v", settings.Music)
	}
	if settings.FaviconURL != "" || settings.HomeButton != (model.HomeButtonSettings{Theme: "classic"}) {
		t.Fatalf("旧数据缺少图标和主页按钮字段时应使用默认外观，实际为 %+v", settings)
	}
}

func TestDecodeSettingsExplicitCardsOverrideDefaults(t *testing.T) {
	data := `{"cards":{"profile":false,"navigation":true,"devices":false,"terminal":true,"clock":false},` +
		`"terminal":{"fastfetch_loading_ms":3500},` +
		`"music":{"lyrics_enabled":false,"playlist_id":12},` +
		`"loader":{"theme":"beige","sound_enabled":true,"watermark_title":"MY SPACE","badge_image":"/storage/images/badge.png"}}`

	settings, err := decodeSettings(data)
	if err != nil {
		t.Fatalf("decodeSettings 返回错误: %v", err)
	}
	cards := settings.Cards
	if cards.Profile || !cards.Navigation || cards.Devices || !cards.Terminal || cards.Clock {
		t.Fatalf("显式 cards 配置应覆盖默认值，实际为 %+v", cards)
	}
	if settings.Loader.Theme != "beige" || !settings.Loader.SoundEnabled ||
		settings.Loader.WatermarkTitle != "MY SPACE" || settings.Loader.BadgeImage != "/storage/images/badge.png" {
		t.Fatalf("loader 字段解析不正确: %+v", settings.Loader)
	}
	if settings.Terminal.FastfetchLoadingMS != 3500 {
		t.Fatalf("显式 terminal 停留时长应保留，实际为 %+v", settings.Terminal)
	}
	if settings.Music.LyricsEnabled || settings.Music.PlaylistID != 12 {
		t.Fatalf("显式 music 配置应覆盖默认值，实际为 %+v", settings.Music)
	}
}

func TestDecodeSettingsPartialCardsKeepsRemainingEnabled(t *testing.T) {
	settings, err := decodeSettings(`{"cards":{"terminal":false}}`)
	if err != nil {
		t.Fatalf("decodeSettings 返回错误: %v", err)
	}
	cards := settings.Cards
	if cards.Terminal {
		t.Fatal("显式关闭的 terminal 应为 false")
	}
	if !cards.Profile || !cards.Navigation || !cards.Devices || !cards.Clock {
		t.Fatalf("未指定的卡片应保持默认开启，实际为 %+v", cards)
	}
}

func TestDecodeSettingsPartialLoaderKeepsDefaults(t *testing.T) {
	settings, err := decodeSettings(`{"loader":{"watermark_title":"MY SPACE"}}`)
	if err != nil {
		t.Fatalf("decodeSettings 返回错误: %v", err)
	}
	if settings.Loader.Theme != "dark" || settings.Loader.SoundEnabled {
		t.Fatalf("部分 loader 数据应保持主题和音效默认值，实际为 %+v", settings.Loader)
	}
	if settings.Loader.WatermarkTitle != "MY SPACE" {
		t.Fatalf("已有 loader 字段不应被默认值覆盖，实际为 %+v", settings.Loader)
	}
}
