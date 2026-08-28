package server

import (
	"net"
	"sync"
	"time"

	"charm.land/ssh"
)

const maxTrackedAttemptAddresses = 4096

type attemptEntry struct {
	windowStart time.Time
	lastSeen    time.Time
	attempts    int
}

type attemptLimiter struct {
	mu         sync.Mutex
	entries    map[string]*attemptEntry
	limit      int
	window     time.Duration
	maxEntries int
	now        func() time.Time
}

func newAttemptLimiter(limit int, window time.Duration, maxEntries int, now func() time.Time) *attemptLimiter {
	if maxEntries < 1 {
		maxEntries = 1
	}
	if now == nil {
		now = time.Now
	}
	return &attemptLimiter{
		entries:    make(map[string]*attemptEntry),
		limit:      limit,
		window:     window,
		maxEntries: maxEntries,
		now:        now,
	}
}

func (l *attemptLimiter) Allow(ip string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.removeExpired(now)
	entry, ok := l.entries[ip]
	if !ok {
		if len(l.entries) >= l.maxEntries {
			l.evictLeastRecentlySeen()
		}
		entry = &attemptEntry{windowStart: now}
		l.entries[ip] = entry
	} else if now.Before(entry.windowStart) || !now.Before(entry.windowStart.Add(l.window)) {
		entry.windowStart = now
		entry.attempts = 0
	}

	entry.lastSeen = now
	entry.attempts++
	return entry.attempts <= l.limit
}

func (l *attemptLimiter) removeExpired(now time.Time) {
	for ip, entry := range l.entries {
		if now.Before(entry.lastSeen) || !now.Before(entry.lastSeen.Add(l.window)) {
			delete(l.entries, ip)
		}
	}
}

func (l *attemptLimiter) evictLeastRecentlySeen() {
	var oldestIP string
	var oldestTime time.Time
	for ip, entry := range l.entries {
		if oldestIP == "" || entry.lastSeen.Before(oldestTime) || entry.lastSeen.Equal(oldestTime) && ip < oldestIP {
			oldestIP = ip
			oldestTime = entry.lastSeen
		}
	}
	if oldestIP != "" {
		delete(l.entries, oldestIP)
	}
}

func connectionAttemptLimitCallback(limiter *attemptLimiter) ssh.ConnCallback {
	return func(_ ssh.Context, conn net.Conn) net.Conn {
		if conn == nil || !limiter.Allow(remoteIP(conn.RemoteAddr())) {
			return nil
		}
		return conn
	}
}
