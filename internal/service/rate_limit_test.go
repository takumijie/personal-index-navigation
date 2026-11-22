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
	if !limiter.Allow("ip") || !limiter.Allow("ip") || limiter.Allow("ip") {
		t.Fatal("限流器未按窗口内请求次数限制 IP")
	}
}
