package server

import (
	"net"
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/config"
)

// ConnLimiter tracks and limits connections per IP and total.
type ConnLimiter struct {
	mu          sync.Mutex
	ipCounts    map[string]int
	totalCount  int
	maxPerIP    int
	maxTotal    int
}

// NewConnLimiter creates a new connection limiter with the given config.
func NewConnLimiter(cfg config.ConnectionsConfig) *ConnLimiter {
	return &ConnLimiter{
		ipCounts: make(map[string]int),
		maxPerIP: cfg.MaxPerIP,
		maxTotal: cfg.MaxTotal,
	}
}

// TryAcquire attempts to acquire a connection slot for the given IP.
// Returns true if the connection is allowed, false if it would exceed limits.
func (c *ConnLimiter) TryAcquire(ip string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check total limit
	if c.maxTotal > 0 && c.totalCount >= c.maxTotal {
		return false
	}

	// Check per-IP limit
	if c.maxPerIP > 0 && c.ipCounts[ip] >= c.maxPerIP {
		return false
	}

	// Acquire the slot
	c.ipCounts[ip]++
	c.totalCount++
	return true
}

// Release releases a connection slot for the given IP.
func (c *ConnLimiter) Release(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ipCounts[ip] > 0 {
		c.ipCounts[ip]--
		if c.ipCounts[ip] == 0 {
			delete(c.ipCounts, ip)
		}
	}
	if c.totalCount > 0 {
		c.totalCount--
	}
}

// GetStats returns the current connection stats.
func (c *ConnLimiter) GetStats() (totalCount int, ipCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.totalCount, len(c.ipCounts)
}

// GetIPCount returns the current connection count for a specific IP.
func (c *ConnLimiter) GetIPCount(ip string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ipCounts[ip]
}

// extractIP extracts the IP address from a remote address string (ip:port format).
func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr // Return as-is if can't split
	}
	return host
}
