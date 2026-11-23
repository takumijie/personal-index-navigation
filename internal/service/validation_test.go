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
		HomeButton:         model.HomeButtonSettings{Theme: "classic"},
		Loader:             model.LoaderSettings{Theme: "dark"},
		GuestbookMaxLength: 280,
	}
}

func TestValidateSettingsHomeBranding(t *testing.T) {
	settings := validSettings()
	for _, theme := range []string{"classic", "glass", "terminal", "aurora", "image"} {
		candidate := settings
		candidate.FaviconURL = "/storage/images/favicon.ico"
		candidate.HomeButton.Theme = theme
		if theme == "image" {
			candidate.HomeButton.ImageURL = "https://example.com/brand.webp"
		}
		if err := validateSettings(candidate); err != nil {
			t.Fatalf("主页按钮主题 %s 应有效: %v", theme, err)
		}
	}

	invalidCases := []struct {
		name   string
		mutate func(*model.Settings)
	}{
		{name: "图标协议", mutate: func(candidate *model.Settings) { candidate.FaviconURL = "javascript:alert(1)" }},
		{name: "按钮主题", mutate: func(candidate *model.Settings) { candidate.HomeButton.Theme = "unknown" }},
		{name: "图片主题缺少图片", mutate: func(candidate *model.Settings) { candidate.HomeButton.Theme = "image" }},
		{name: "按钮图片地址", mutate: func(candidate *model.Settings) {
			candidate.HomeButton.Theme = "image"
			candidate.HomeButton.ImageURL = "images/brand.png"
		}},
	}
	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := settings
			testCase.mutate(&candidate)
			if err := validateSettings(candidate); err == nil {
				t.Fatal("无效的网站图标或主页按钮设置应被拒绝")
			}
		})
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
