package server

import (
	"net/http"
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/config"
)

func TestConnLimiter_PerIPLimit(t *testing.T) {
	limiter := NewConnLimiter(config.ConnectionsConfig{
		MaxPerIP: 2,
		MaxTotal: 100,
	})

	// First two connections from same IP should succeed
	if !limiter.TryAcquire("192.168.1.1") {
		t.Error("first connection should be allowed")
	}
	if !limiter.TryAcquire("192.168.1.1") {
		t.Error("second connection should be allowed")
	}

	// Third connection from same IP should fail
	if limiter.TryAcquire("192.168.1.1") {
		t.Error("third connection from same IP should be rejected")
	}

	// Connection from different IP should succeed
	if !limiter.TryAcquire("192.168.1.2") {
		t.Error("connection from different IP should be allowed")
	}

	// Release one connection
	limiter.Release("192.168.1.1")

	// Now should be able to connect again
	if !limiter.TryAcquire("192.168.1.1") {
		t.Error("connection should be allowed after release")
	}
}

func TestConnLimiter_TotalLimit(t *testing.T) {
	limiter := NewConnLimiter(config.ConnectionsConfig{
		MaxPerIP: 10,
		MaxTotal: 3,
	})

	// First 3 connections should succeed
	if !limiter.TryAcquire("192.168.1.1") {
		t.Error("first connection should be allowed")
	}
	if !limiter.TryAcquire("192.168.1.2") {
		t.Error("second connection should be allowed")
	}
	if !limiter.TryAcquire("192.168.1.3") {
		t.Error("third connection should be allowed")
	}

	// Fourth connection should fail (total limit)
	if limiter.TryAcquire("192.168.1.4") {
		t.Error("fourth connection should be rejected due to total limit")
	}

	// Release one
	limiter.Release("192.168.1.1")

	// Now should be able to connect
	if !limiter.TryAcquire("192.168.1.4") {
		t.Error("connection should be allowed after release")
	}
}

func TestConnLimiter_Unlimited(t *testing.T) {
	limiter := NewConnLimiter(config.ConnectionsConfig{
		MaxPerIP: 0, // Unlimited
		MaxTotal: 0, // Unlimited
	})

	// Should allow many connections
	for i := 0; i < 100; i++ {
		if !limiter.TryAcquire("192.168.1.1") {
			t.Errorf("connection %d should be allowed when unlimited", i)
		}
	}
}

func TestConnLimiter_GetStats(t *testing.T) {
	limiter := NewConnLimiter(config.ConnectionsConfig{
		MaxPerIP: 10,
		MaxTotal: 100,
	})

	limiter.TryAcquire("192.168.1.1")
	limiter.TryAcquire("192.168.1.1")
	limiter.TryAcquire("192.168.1.2")

	total, uniqueIPs := limiter.GetStats()

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}

	if uniqueIPs != 2 {
		t.Errorf("expected 2 unique IPs, got %d", uniqueIPs)
	}
}

func TestConnLimiter_GetIPCount(t *testing.T) {
	limiter := NewConnLimiter(config.ConnectionsConfig{
		MaxPerIP: 10,
		MaxTotal: 100,
	})

	limiter.TryAcquire("192.168.1.1")
	limiter.TryAcquire("192.168.1.1")
	limiter.TryAcquire("192.168.1.2")

	if count := limiter.GetIPCount("192.168.1.1"); count != 2 {
		t.Errorf("expected count 2 for IP 192.168.1.1, got %d", count)
	}

	if count := limiter.GetIPCount("192.168.1.2"); count != 1 {
		t.Errorf("expected count 1 for IP 192.168.1.2, got %d", count)
	}

	if count := limiter.GetIPCount("192.168.1.3"); count != 0 {
		t.Errorf("expected count 0 for unknown IP, got %d", count)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"[::1]:12345", "::1"},
		{"localhost:4000", "localhost"},
		{"192.168.1.1", "192.168.1.1"}, // No port
	}

	for _, tt := range tests {
		result := extractIP(tt.input)
		if result != tt.expected {
			t.Errorf("extractIP(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetRealIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			xff:        "203.0.113.50",
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			xff:        "203.0.113.50, 70.41.3.18, 150.172.238.178",
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.50", // First IP is the client
		},
		{
			name:       "X-Real-IP",
			xri:        "203.0.113.50",
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			xff:        "203.0.113.50",
			xri:        "198.51.100.25",
			remoteAddr: "10.0.0.1:12345",
			expected:   "203.0.113.50",
		},
		{
			name:       "No headers - use RemoteAddr",
			remoteAddr: "192.168.1.100:54321",
			expected:   "192.168.1.100",
		},
		{
			name:       "Empty X-Forwarded-For falls back to RemoteAddr",
			xff:        "",
			remoteAddr: "192.168.1.100:54321",
			expected:   "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			result := getRealIP(req)
			if result != tt.expected {
				t.Errorf("getRealIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}
