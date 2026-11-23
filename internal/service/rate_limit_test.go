package service

import (
	"fmt"
	"testing"
	"time"
)

func TestIPRateLimiterCleansExpiredEntries(t *testing.T) {
	limiter := NewIPRateLimiter(2, time.Minute)
	now := time.Now()
	for index := range 100 {
		limiter.entries[fmt.Sprintf("stale-%d", index)] = []time.Time{now.Add(-2 * time.Minute)}
	}
	limiter.entries["active"] = []time.Time{now}

	if !limiter.Allow("new-ip") {
		t.Fatal("新 IP 应被允许")
	}
	for index := range 100 {
		key := fmt.Sprintf("stale-%d", index)
		if _, exists := limiter.entries[key]; exists {
			t.Fatalf("过期 IP %s 未被清理", key)
		}
	}
	if _, exists := limiter.entries["active"]; !exists {
		t.Fatal("活跃 IP 不应被清理")
	}
}

func TestIPRateLimiterStillLimitsRecentRequests(t *testing.T) {
	limiter := NewIPRateLimiter(2, time.Minute)
	if !limiter.Allow("ip") {
		t.Fatal("第一次请求应被允许")
	}
	if !limiter.Allow("ip") {
		t.Fatal("第二次请求应被允许")
	}
	if limiter.Allow("ip") {
		t.Fatal("超过窗口额度的请求应被限制")
	}
	if !limiter.Allow("other-ip") {
		t.Fatal("其他 IP 不应受到影响")
	}
}

func TestPublicMusicRateLimitsAreIndependent(t *testing.T) {
	music := NewMusicService(nil, "")
	for range 120 {
		if !music.AllowPlayback("public-ip") {
			t.Fatal("播放额度内请求不应被限制")
		}
	}
	if music.AllowPlayback("public-ip") {
		t.Fatal("播放请求超过额度后应被限制")
	}
	if !music.Allow("public-ip") {
		t.Fatal("后台 Provider 额度不应被公开播放请求消耗")
	}
	if !music.AllowLyrics("public-ip") {
		t.Fatal("歌词额度不应被播放请求消耗")
	}
	if !music.AllowCover("public-ip") {
		t.Fatal("封面额度不应被播放请求消耗")
	}
	if !music.AllowPlayback("other-ip") {
		t.Fatal("其他 IP 的播放额度不应受到影响")
	}
}
