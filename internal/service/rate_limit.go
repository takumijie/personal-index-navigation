package service

import (
	"sync"
	"time"
)

type IPRateLimiter struct {
	mu          sync.Mutex
	window      time.Duration
	limit       int
	entries     map[string][]time.Time
	lastCleanup time.Time
}

func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{limit: limit, window: window, entries: make(map[string][]time.Time)}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
		for key, values := range l.entries {
			if len(values) == 0 || !values[len(values)-1].After(cutoff) {
				delete(l.entries, key)
			}
		}
		l.lastCleanup = now
	}
	entries := l.entries[ip]
	kept := entries[:0]
	for _, timestamp := range entries {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) >= l.limit {
		l.entries[ip] = kept
		return false
	}
	l.entries[ip] = append(kept, now)
	return true
}

func (l *IPRateLimiter) Reset(ip string) {
	l.mu.Lock()
	delete(l.entries, ip)
	l.mu.Unlock()
}
