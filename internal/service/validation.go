package service

import (
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"homepage/internal/model"
)

func plainText(value string, multiline bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			if multiline && (r == '\n' || r == '\r' || r == '\t') {
				continue
			}
			return false
		}
	}
	return true
}

func validLength(value string, max int) bool {
	return utf8.RuneCountInString(value) <= max
}

func required(value string, max int) bool {
	return strings.TrimSpace(value) != "" && validLength(value, max) && plainText(value, false)
}

func httpURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func navigationURL(value string) bool {
	if httpURL(value) {
		return true
	}
	u, err := url.ParseRequestURI(value)
	return err == nil && u.Scheme == "mailto" && strings.Contains(u.Opaque, "@")
}

func resourceURL(value string, requiredValue bool) bool {
	if value == "" {
		return !requiredValue
	}
	if strings.HasPrefix(value, "/images/") || strings.HasPrefix(value, "/storage/images/") || strings.HasPrefix(value, "/storage/music/") {
		return !strings.Contains(value, "\\") && path.Clean(value) == value && !strings.Contains(value, "..")
	}
	return httpURL(value)
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func validateNavigation(item model.NavigationItem) error {
	if !required(item.Title, 100) || !navigationURL(item.URL) || !validLength(item.URL, 2048) {
		return invalid("导航标题或 URL 无效")
	}
	if !resourceURL(item.IconURL, false) || !validLength(item.Icon, 100) || !plainText(item.Icon, false) {
		return invalid("导航图标无效")
	}
	if item.Sort < 0 || item.Sort > 1000000 || !oneOf(item.OpenMode, "blank", "self") {
		return invalid("导航排序或打开方式无效")
	}
	return nil
}

func validateDevice(item model.Device) error {
	if !required(item.Name, 100) || !validLength(item.Description, 500) || !plainText(item.Description, true) {
		return invalid("设备名称或描述无效")
	}
	if !validLength(item.Icon, 100) || !plainText(item.Icon, false) || item.Sort < 0 || item.Sort > 1000000 {
		return invalid("设备图标或排序无效")
	}
	if !required(item.Category, 50) {
		return invalid("设备分类无效")
	}
	return nil
}

func validateTrack(item model.MusicTrack) error {
	if !required(item.Title, 200) || !validLength(item.Artist, 200) || !validLength(item.Album, 200) {
		return invalid("曲目标题、艺术家或专辑无效")
	}
	for _, value := range []string{item.Artist, item.Album, item.Source} {
		if !plainText(value, false) {
			return invalid("曲目信息包含控制字符")
		}
	}
	if !resourceURL(item.URL, true) || !resourceURL(item.Cover, false) || !validLength(item.URL, 2048) || !validLength(item.Cover, 2048) {
		return invalid("曲目或封面 URL 无效")
	}
	if !oneOf(item.Source, "local", "remote") || item.Sort < 0 || item.Sort > 1000000 {
		return invalid("曲目来源或排序无效")
	}
	return nil
}

func validateProfile(profile model.SiteProfile) error {
	if !required(profile.Name, 100) || !required(profile.Username, 100) || !validLength(profile.Description, 1000) || !plainText(profile.Description, true) {
		return invalid("资料名称、用户名或描述无效")
	}
	if !resourceURL(profile.Avatar, false) || !resourceURL(profile.CoverURL, false) {
		return invalid("头像或封面 URL 无效")
	}
	if !oneOf(profile.CoverMode, "current", "day", "night", "custom") || len(profile.Tags) > 20 {
		return invalid("封面模式或标签数量无效")
	}
	seen := make(map[string]struct{}, len(profile.Tags))
	for _, tag := range profile.Tags {
		if !required(tag, 30) {
			return invalid("资料标签无效")
		}
		if _, ok := seen[tag]; ok {
			return invalid("资料标签不能重复")
		}
		seen[tag] = struct{}{}
	}
	return nil
}

func validateSettings(settings model.Settings) error {
	if !required(settings.SiteTitle, 100) || !validLength(settings.ClockLabel, 100) || !plainText(settings.ClockLabel, false) {
		return invalid("站点标题或时钟标签无效")
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		return invalid("站点时区无效")
	}
	if _, err := time.LoadLocation(settings.Background.Timezone); err != nil {
		return invalid("背景时区无效")
	}
	if !oneOf(settings.TimeFormat, "12", "24") {
		return invalid("时间格式无效")
	}
	if !resourceURL(settings.Background.DayURL, true) || !resourceURL(settings.Background.NightURL, true) {
		return invalid("背景 URL 无效")
	}
	if !validClock(settings.Background.DayStart) || !validClock(settings.Background.NightStart) {
		return invalid("背景切换时间无效")
	}
	if settings.Background.DurationMS < 0 || settings.Background.DurationMS > 60000 {
		return invalid("背景切换时长无效")
	}
	if settings.GuestbookMaxLength < 1 || settings.GuestbookMaxLength > 10000 {
		return invalid("留言长度限制无效")
	}
	return nil
}

func validClock(value string) bool {
	parsed, err := time.Parse("15:04", value)
	return err == nil && parsed.Format("15:04") == value
}
