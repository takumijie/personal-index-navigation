package service

import (
	"testing"

	"homepage/internal/model"
)

func validSettings() model.Settings {
	return model.Settings{
		SiteTitle:  "Personal Space",
		Timezone:   "UTC",
		TimeFormat: "24",
		ClockLabel: "LOCAL",
		Background: model.Background{
			DayURL: "https://example.com/day.webp", NightURL: "https://example.com/night.webp",
			DayStart: "06:00", NightStart: "18:00", Timezone: "UTC",
		},
		Loader:             model.LoaderSettings{Theme: "dark"},
		GuestbookMaxLength: 280,
	}
}

func TestValidateSettingsFastfetchLoadingRange(t *testing.T) {
	settings := validSettings()
	for _, value := range []int{0, 2000, 10000} {
		settings.Terminal.FastfetchLoadingMS = value
		if err := validateSettings(settings); err != nil {
			t.Fatalf("停留时长 %d 应有效: %v", value, err)
		}
	}
	for _, value := range []int{-1, 10001} {
		settings.Terminal.FastfetchLoadingMS = value
		if err := validateSettings(settings); err == nil {
			t.Fatalf("停留时长 %d 应被拒绝", value)
		}
	}
}

func TestValidateSettingsDurationAndGuestbookRanges(t *testing.T) {
	settings := validSettings()
	for _, value := range []int{0, 10000, 60000} {
		candidate := settings
		candidate.Background.DurationMS = value
		if err := validateSettings(candidate); err != nil {
			t.Fatalf("背景切换时长 %d 应有效: %v", value, err)
		}
	}
	for _, value := range []int{-1, 60001} {
		candidate := settings
		candidate.Background.DurationMS = value
		if err := validateSettings(candidate); err == nil {
			t.Fatalf("背景切换时长 %d 应被拒绝", value)
		}
	}
	for _, value := range []int{1, 2000, 10000} {
		candidate := settings
		candidate.GuestbookMaxLength = value
		if err := validateSettings(candidate); err != nil {
			t.Fatalf("留言长度限制 %d 应有效: %v", value, err)
		}
	}
	for _, value := range []int{0, 10001} {
		candidate := settings
		candidate.GuestbookMaxLength = value
		if err := validateSettings(candidate); err == nil {
			t.Fatalf("留言长度限制 %d 应被拒绝", value)
		}
	}
}
