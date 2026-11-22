package repository

import (
	"testing"

	"homepage/internal/model"
)

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
}

func TestDecodeSettingsExplicitCardsOverrideDefaults(t *testing.T) {
	data := `{"cards":{"profile":false,"navigation":true,"devices":false,"terminal":true,"clock":false},` +
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
