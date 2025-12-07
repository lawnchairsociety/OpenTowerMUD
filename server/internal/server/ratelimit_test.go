package server

import (
	"testing"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/config"
)

func TestLoginRateLimiter_Basic(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       3,
		LockoutSeconds:    1,
		MaxLockoutSeconds: 10,
	})
	defer rl.Stop()

	ip := "192.168.1.1"

	// First 2 failures should not trigger lockout
	locked, _ := rl.RecordFailure(ip)
	if locked {
		t.Error("first failure should not trigger lockout")
	}

	locked, _ = rl.RecordFailure(ip)
	if locked {
		t.Error("second failure should not trigger lockout")
	}

	// Third failure should trigger lockout
	locked, duration := rl.RecordFailure(ip)
	if !locked {
		t.Error("third failure should trigger lockout")
	}
	if duration < 1*time.Second {
		t.Errorf("lockout duration should be at least 1 second, got %v", duration)
	}

	// Should be locked now
	isLocked, _ := rl.IsLocked(ip)
	if !isLocked {
		t.Error("IP should be locked")
	}
}

func TestLoginRateLimiter_SuccessClears(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       3,
		LockoutSeconds:    1,
		MaxLockoutSeconds: 10,
	})
	defer rl.Stop()

	ip := "192.168.1.1"

	// Record 2 failures
	rl.RecordFailure(ip)
	rl.RecordFailure(ip)

	// Success should clear the counter
	rl.RecordSuccess(ip)

	// Should need 3 more failures to trigger lockout
	locked, _ := rl.RecordFailure(ip)
	if locked {
		t.Error("first failure after success should not trigger lockout")
	}

	locked, _ = rl.RecordFailure(ip)
	if locked {
		t.Error("second failure after success should not trigger lockout")
	}
}

func TestLoginRateLimiter_ExponentialBackoff(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       1, // Lock after 1 attempt for faster testing
		LockoutSeconds:    1,
		MaxLockoutSeconds: 10,
	})
	defer rl.Stop()

	ip := "192.168.1.1"

	// First lockout should be ~1 second
	_, duration1 := rl.RecordFailure(ip)
	if duration1 < 1*time.Second || duration1 > 2*time.Second {
		t.Errorf("first lockout should be ~1 second, got %v", duration1)
	}

	// Wait for lockout to expire
	time.Sleep(duration1 + 100*time.Millisecond)

	// Second lockout should be ~2 seconds (doubled)
	_, duration2 := rl.RecordFailure(ip)
	if duration2 < 2*time.Second || duration2 > 3*time.Second {
		t.Errorf("second lockout should be ~2 seconds, got %v", duration2)
	}

	// Wait for lockout to expire
	time.Sleep(duration2 + 100*time.Millisecond)

	// Third lockout should be ~4 seconds (doubled again)
	_, duration3 := rl.RecordFailure(ip)
	if duration3 < 4*time.Second || duration3 > 5*time.Second {
		t.Errorf("third lockout should be ~4 seconds, got %v", duration3)
	}
}

func TestLoginRateLimiter_MaxLockout(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       1,
		LockoutSeconds:    1,
		MaxLockoutSeconds: 2, // Cap at 2 seconds
	})
	defer rl.Stop()

	ip := "192.168.1.1"

	// First lockout: 1 second
	rl.RecordFailure(ip)
	time.Sleep(1100 * time.Millisecond)

	// Second lockout: would be 2 seconds (doubled), but capped at 2
	_, duration2 := rl.RecordFailure(ip)
	if duration2 > 2100*time.Millisecond {
		t.Errorf("lockout should be capped at 2 seconds, got %v", duration2)
	}

	time.Sleep(duration2 + 100*time.Millisecond)

	// Third lockout: would be 4 seconds, but capped at 2
	_, duration3 := rl.RecordFailure(ip)
	if duration3 > 2100*time.Millisecond {
		t.Errorf("lockout should still be capped at 2 seconds, got %v", duration3)
	}
}

func TestLoginRateLimiter_MultipleIPs(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       2,
		LockoutSeconds:    1,
		MaxLockoutSeconds: 10,
	})
	defer rl.Stop()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Lock out IP1
	rl.RecordFailure(ip1)
	rl.RecordFailure(ip1)

	locked1, _ := rl.IsLocked(ip1)
	if !locked1 {
		t.Error("IP1 should be locked")
	}

	// IP2 should not be affected
	locked2, _ := rl.IsLocked(ip2)
	if locked2 {
		t.Error("IP2 should not be locked")
	}

	// IP2 can still fail once without lockout
	locked, _ := rl.RecordFailure(ip2)
	if locked {
		t.Error("first failure for IP2 should not trigger lockout")
	}
}

func TestLoginRateLimiter_GetAttempts(t *testing.T) {
	rl := NewLoginRateLimiter(config.RateLimitConfig{
		MaxAttempts:       5,
		LockoutSeconds:    30,
		MaxLockoutSeconds: 300,
	})
	defer rl.Stop()

	ip := "192.168.1.1"

	if count := rl.GetAttempts(ip); count != 0 {
		t.Errorf("expected 0 attempts, got %d", count)
	}

	rl.RecordFailure(ip)
	if count := rl.GetAttempts(ip); count != 1 {
		t.Errorf("expected 1 attempt, got %d", count)
	}

	rl.RecordFailure(ip)
	rl.RecordFailure(ip)
	if count := rl.GetAttempts(ip); count != 3 {
		t.Errorf("expected 3 attempts, got %d", count)
	}
}
