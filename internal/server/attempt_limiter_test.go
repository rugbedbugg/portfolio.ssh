package server

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAttemptLimiterEnforcesIndependentIPWindows(t *testing.T) {
	now := time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)
	limiter := newAttemptLimiter(2, time.Minute, 16, func() time.Time { return now })

	if !limiter.Allow("192.0.2.10") || !limiter.Allow("192.0.2.10") {
		t.Fatal("limiter rejected an attempt within the per-IP window limit")
	}
	if limiter.Allow("192.0.2.10") {
		t.Fatal("limiter accepted an attempt beyond the per-IP window limit")
	}
	if !limiter.Allow("192.0.2.11") {
		t.Fatal("one IP exhausted another IP's independent attempt limit")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("192.0.2.10") {
		t.Fatal("limiter did not reset an IP after its attempt window elapsed")
	}
}

func TestAttemptLimiterEvictsLeastRecentlySeenAddressAtCapacity(t *testing.T) {
	now := time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)
	limiter := newAttemptLimiter(10, time.Hour, 2, func() time.Time { return now })

	limiter.Allow("192.0.2.10")
	now = now.Add(time.Second)
	limiter.Allow("192.0.2.11")
	now = now.Add(time.Second)
	limiter.Allow("192.0.2.10")
	now = now.Add(time.Second)
	limiter.Allow("192.0.2.12")

	if len(limiter.entries) != 2 {
		t.Fatalf("tracked attempt addresses = %d, want hard cap 2", len(limiter.entries))
	}
	if _, ok := limiter.entries["192.0.2.11"]; ok {
		t.Fatal("least recently seen address was not evicted")
	}
	for _, retained := range []string{"192.0.2.10", "192.0.2.12"} {
		if _, ok := limiter.entries[retained]; !ok {
			t.Fatalf("recent address %s was evicted", retained)
		}
	}
}

func TestAttemptLimiterCleansExpiredAddressState(t *testing.T) {
	now := time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)
	limiter := newAttemptLimiter(2, time.Minute, 16, func() time.Time { return now })

	limiter.Allow("192.0.2.10")
	limiter.Allow("192.0.2.11")
	now = now.Add(time.Minute)
	limiter.Allow("192.0.2.12")

	if len(limiter.entries) != 1 {
		t.Fatalf("tracked attempt addresses after expiry cleanup = %d, want 1", len(limiter.entries))
	}
	if _, ok := limiter.entries["192.0.2.12"]; !ok {
		t.Fatal("current address missing after expiry cleanup")
	}
}

func TestAttemptLimiterSerializesConcurrentAttempts(t *testing.T) {
	const limit = 25
	limiter := newAttemptLimiter(limit, time.Minute, 16, func() time.Time {
		return time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)
	})

	var accepted atomic.Int32
	var workers sync.WaitGroup
	for range 100 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if limiter.Allow("192.0.2.10") {
				accepted.Add(1)
			}
		}()
	}
	workers.Wait()

	if got := accepted.Load(); got != limit {
		t.Fatalf("accepted concurrent attempts = %d, want %d", got, limit)
	}
}

func TestConnectionAttemptLimitCallbackRejectsBeforeSSHHandshake(t *testing.T) {
	limiter := newAttemptLimiter(1, time.Minute, 16, func() time.Time {
		return time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)
	})
	callback := connectionAttemptLimitCallback(limiter)

	firstClient, firstServer := net.Pipe()
	t.Cleanup(func() { _ = firstClient.Close(); _ = firstServer.Close() })
	first := &remoteAddressConn{Conn: firstServer, remote: &net.TCPAddr{IP: net.ParseIP("198.51.100.8"), Port: 49152}}
	if got := callback(nil, first); got != first {
		t.Fatalf("first connection callback returned %T, want original connection", got)
	}

	secondClient, secondServer := net.Pipe()
	t.Cleanup(func() { _ = secondClient.Close(); _ = secondServer.Close() })
	second := &remoteAddressConn{Conn: secondServer, remote: &net.TCPAddr{IP: net.ParseIP("198.51.100.8"), Port: 49153}}
	if got := callback(nil, second); got != nil {
		t.Fatalf("rate-limited callback returned %T, want nil pre-handshake rejection", got)
	}
}

type remoteAddressConn struct {
	net.Conn
	remote net.Addr
}

func (c *remoteAddressConn) RemoteAddr() net.Addr { return c.remote }
