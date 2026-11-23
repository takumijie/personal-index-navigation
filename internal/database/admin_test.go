package database

import (
	"path/filepath"
	"strings"
	"testing"

	"homepage/internal/config"
	"homepage/internal/model"
	"homepage/internal/repository"
	"homepage/internal/service"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func openAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "admin.db"))
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
	return db
}

func TestSeedRequiresPasswordOnlyForEmptyAdminTable(t *testing.T) {
	db := openAdminTestDB(t)
	cfg := config.Default()
	if err := Seed(db, cfg); err == nil {
		t.Fatal("空库缺少初始管理员密码时应失败")
	}
	var count int64
	if err := db.Model(&model.AdminUser{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("初始化失败后管理员数量 = %d，预期 0", count)
	}

	cfg.Admin.Password = "initial-password"
	if err := Seed(db, cfg); err != nil {
		t.Fatal(err)
	}
	var before model.AdminUser
	if err := db.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	cfg.Admin.Username = "replacement"
	cfg.Admin.Password = "replacement-password"
	if err := Seed(db, cfg); err != nil {
		t.Fatal(err)
	}
	var after model.AdminUser
	if err := db.First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Username != before.Username || after.PasswordHash != before.PasswordHash || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("重复 Seed 修改了已有管理员: before=%+v after=%+v", before, after)
	}
	if bcrypt.CompareHashAndPassword([]byte(after.PasswordHash), []byte("initial-password")) != nil {
		t.Fatal("重复 Seed 后初始密码应仍然有效")
	}

	cfg.Admin.Password = ""
	if err := Seed(db, cfg); err != nil {
		t.Fatalf("已有管理员时应允许无密码启动: %v", err)
	}
}

func TestSeedAdminValidatesUnicodeAndBcryptLength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "eight ASCII runes", password: "12345678", valid: true},
		{name: "eight Chinese runes", password: "密码密码密码密码", valid: true},
		{name: "72 bytes", password: strings.Repeat("a", 72), valid: true},
		{name: "seven Chinese runes", password: "密码密码密码密", valid: false},
		{name: "73 bytes", password: strings.Repeat("a", 73), valid: false},
		{name: "25 Chinese runes exceed bytes", password: strings.Repeat("密", 25), valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openAdminTestDB(t)
			err := seedAdmin(db, config.Admin{Username: "admin", Password: test.password})
			if test.valid && err != nil {
				t.Fatalf("合法密码初始化失败: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("非法密码不应创建管理员")
			}
		})
	}
}

func TestResetAdminPasswordChangesPasswordAndInvalidatesSessions(t *testing.T) {
	db := openAdminTestDB(t)
	cfg := config.Default()
	cfg.Admin.Password = "old-password"
	if err := Seed(db, cfg); err != nil {
		t.Fatal(err)
	}
	auth := service.NewAuthService(repository.New(db))
	oldToken, _, err := auth.Login(cfg.Admin.Username, cfg.Admin.Password, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	cfg.Admin.Username = "ignored-during-reset"
	cfg.Admin.Password = "new-password"
	if err := ResetAdminPassword(db, cfg.Admin); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Session(oldToken); err == nil {
		t.Fatal("重置密码后旧会话应失效")
	}
	if _, _, err := auth.Login("admin", "old-password", "127.0.0.1"); err == nil {
		t.Fatal("重置密码后旧密码不应能登录")
	}
	newToken, info, err := auth.Login("admin", "new-password", "127.0.0.1")
	if err != nil {
		t.Fatalf("新密码无法登录: %v", err)
	}
	if info.Username != "admin" {
		t.Fatalf("重置密码不应修改管理员用户名: %q", info.Username)
	}
	if _, err := auth.Session(newToken); err != nil {
		t.Fatalf("新会话应有效: %v", err)
	}
}

func TestResetAdminPasswordRejectsInvalidPasswordWithoutChanges(t *testing.T) {
	db := openAdminTestDB(t)
	cfg := config.Default()
	cfg.Admin.Password = "original-password"
	if err := Seed(db, cfg); err != nil {
		t.Fatal(err)
	}
	var before model.AdminUser
	if err := db.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	session := model.AdminSession{TokenHash: "existing-session", AdminUserID: before.ID}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	for _, password := range []string{"", "short", strings.Repeat("a", 73)} {
		if err := ResetAdminPassword(db, config.Admin{Username: "admin", Password: password}); err == nil {
			t.Fatalf("非法新密码 %q 应失败", password)
		}
	}
	var after model.AdminUser
	if err := db.First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.PasswordHash != before.PasswordHash || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("非法新密码不应修改管理员")
	}
	var count int64
	if err := db.Model(&model.AdminSession{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("非法新密码不应删除会话，数量 = %d", count)
	}
}
