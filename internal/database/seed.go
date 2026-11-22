package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"homepage/internal/config"
	"homepage/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const dayBackground = "/images/background-day.webp"
const nightBackground = "/images/background-night.webp"
const defaultAvatar = "/images/avatar.webp"

func Seed(db *gorm.DB, cfg config.Config) error {
	if err := seedAdmin(db, cfg.Admin); err != nil {
		return err
	}
	var count int64
	if err := db.Model(&model.SiteProfile{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	settings := model.Settings{
		SiteTitle:  "个人主页",
		Timezone:   "Asia/Shanghai",
		TimeFormat: "24",
		ClockLabel: "本地时间",
		Background: model.Background{
			DayURL: dayBackground, NightURL: nightBackground,
			DayStart: "06:00", NightStart: "18:00", Timezone: "Asia/Shanghai", DurationMS: 1000,
		},
		Cards:                model.CardSettings{Profile: true, Navigation: true, Devices: true, Terminal: true, Clock: true},
		Loader:               model.LoaderSettings{Theme: "dark", SoundEnabled: false},
		Terminal:             model.TerminalSettings{FastfetchLoadingMS: 2000},
		GuestbookAutoPublish: false,
		GuestbookMaxLength:   500,
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		profile := model.SiteProfile{
			Name: "Takaki Toono", Username: "takaki_toono", Description: "A quieter place on the internet.\nExploring the world, one line at a time.",
			Avatar: defaultAvatar, CoverMode: "current",
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
		for i, tag := range []string{"开发者", "音乐", "生活"} {
			if err := tx.Create(&model.ProfileTag{ProfileID: profile.ID, Tag: tag, Sort: i}).Error; err != nil {
				return err
			}
		}
		navigation := []model.NavigationItem{
			{Title: "Blog", URL: "https://example.com", Icon: "BookOpen", Sort: 0, Enabled: true, OpenMode: "blank"},
			{Title: "GitHub", URL: "https://github.com", Icon: "Github", Sort: 1, Enabled: true, OpenMode: "blank"},
			{Title: "Photos", URL: "https://example.com", Icon: "Camera", Sort: 2, Enabled: true, OpenMode: "blank"},
			{Title: "Email", URL: "mailto:hello@example.com", Icon: "Mail", Sort: 3, Enabled: true, OpenMode: "self"},
		}
		if err := tx.Create(&navigation).Error; err != nil {
			return err
		}
		devices := []model.Device{
			{Name: "ThinkPad X1", Description: "Linux Laptop", Icon: "Laptop", Category: "computer", Sort: 0, Enabled: true},
			{Name: "Keychron K2", Description: "Mechanical Keyboard", Icon: "Keyboard", Category: "peripheral", Sort: 1, Enabled: true},
			{Name: "iPhone 15", Description: "Mobile", Icon: "Smartphone", Category: "mobile", Sort: 2, Enabled: true},
			{Name: "Sony WH-1000XM5", Description: "Wireless Headphones", Icon: "Headphones", Category: "peripheral", Sort: 3, Enabled: true},
		}
		if err := tx.Create(&devices).Error; err != nil {
			return err
		}
		playlist := model.Playlist{Name: "默认播放列表", Sort: 0, Enabled: true}
		if err := tx.Create(&playlist).Error; err != nil {
			return err
		}
		return tx.Create(&model.SiteSettings{Data: string(data)}).Error
	})
}

func seedAdmin(db *gorm.DB, cfg config.Admin) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成管理员密码哈希: %w", err)
	}
	var user model.AdminUser
	err = db.Order("id").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&model.AdminUser{Username: cfg.Username, PasswordHash: string(hash)}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&user).Updates(map[string]any{
		"username": cfg.Username, "password_hash": string(hash), "updated_at": time.Now(),
	}).Error
}
