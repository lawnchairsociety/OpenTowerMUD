package server

import (
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/config"
)

// LoginRateLimiter tracks failed login attempts and enforces lockouts.
type LoginRateLimiter struct {
	mu                sync.Mutex
	attempts          map[string]*attemptInfo
	maxAttempts       int
	lockoutSeconds    int
	maxLockoutSeconds int
	cleanupInterval   time.Duration
	stopCleanup       chan struct{}
}

type attemptInfo struct {
	failedAttempts int
	lockedUntil    time.Time
	lockoutCount   int // Number of times locked out (for exponential backoff)
}

// NewLoginRateLimiter creates a new rate limiter with the given config.
func NewLoginRateLimiter(cfg config.RateLimitConfig) *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts:          make(map[string]*attemptInfo),
		maxAttempts:       cfg.MaxAttempts,
		lockoutSeconds:    cfg.LockoutSeconds,
		maxLockoutSeconds: cfg.MaxLockoutSeconds,
		cleanupInterval:   5 * time.Minute,
		stopCleanup:       make(chan struct{}),
	}

	// Use sensible defaults if not configured
	if rl.maxAttempts == 0 {
		rl.maxAttempts = 5
	}
	if rl.lockoutSeconds == 0 {
		rl.lockoutSeconds = 30
	}
	if rl.maxLockoutSeconds == 0 {
		rl.maxLockoutSeconds = 300
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Stop stops the cleanup goroutine.
func (rl *LoginRateLimiter) Stop() {
	close(rl.stopCleanup)
}

// IsLocked checks if the given IP is currently locked out.
// Returns true if locked, along with the remaining lockout duration.
func (rl *LoginRateLimiter) IsLocked(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	info, exists := rl.attempts[ip]
	if !exists {
		return false, 0
	}

	if time.Now().Before(info.lockedUntil) {
		return true, time.Until(info.lockedUntil)
	}

	return false, 0
}

// RecordFailure records a failed login attempt for the given IP.
// Returns true if the IP is now locked out, along with the lockout duration.
func (rl *LoginRateLimiter) RecordFailure(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	info, exists := rl.attempts[ip]
	if !exists {
		info = &attemptInfo{}
		rl.attempts[ip] = info
	}

	// If currently locked, extend the lockout (they're trying again while locked)
	if time.Now().Before(info.lockedUntil) {
		return true, time.Until(info.lockedUntil)
	}

	info.failedAttempts++

	// Check if we should lock them out
	if info.failedAttempts >= rl.maxAttempts {
		info.lockoutCount++
		// Exponential backoff: double the lockout each time, up to max
		lockoutDuration := time.Duration(rl.lockoutSeconds) * time.Second
		maxDuration := time.Duration(rl.maxLockoutSeconds) * time.Second
		for i := 1; i < info.lockoutCount; i++ {
			// Check before multiplication to prevent overflow
			if lockoutDuration >= maxDuration/2 {
				lockoutDuration = maxDuration
				break
			}
			lockoutDuration *= 2
		}
		// Final cap to ensure we never exceed max (handles edge cases)
		if lockoutDuration > maxDuration {
			lockoutDuration = maxDuration
		}
		info.lockedUntil = time.Now().Add(lockoutDuration)
		info.failedAttempts = 0 // Reset attempts for next round
		return true, lockoutDuration
	}

	return false, 0
}

// RecordSuccess records a successful login, clearing the failure count.
func (rl *LoginRateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, ip)
}

// GetAttempts returns the current failed attempt count for an IP.
func (rl *LoginRateLimiter) GetAttempts(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if info, exists := rl.attempts[ip]; exists {
		return info.failedAttempts
	}
	return 0
}

// cleanupLoop periodically removes expired entries.
func (rl *LoginRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCleanup:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes entries that are no longer locked and have no recent attempts.
func (rl *LoginRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Remove entries that have been unlocked for at least 10 minutes
	// and have no recent failed attempts
	cutoff := now.Add(-10 * time.Minute)

	for ip, info := range rl.attempts {
		if info.lockedUntil.Before(cutoff) && info.failedAttempts == 0 {
			delete(rl.attempts, ip)
		}
	}
}
