package antispam

import (
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	config := Config{
		Enabled:        true,
		MaxMessages:    3,
		TimeWindow:     1 * time.Second,
		RepeatCooldown: 30 * time.Second,
	}
	tracker := NewTracker(config)

	// First 3 messages should be allowed
	for i := 0; i < 3; i++ {
		result := tracker.Check("message " + string(rune('a'+i)))
		if !result.Allowed {
			t.Errorf("Message %d should be allowed", i+1)
		}
	}

	// 4th message should be blocked (rate limit)
	result := tracker.Check("message d")
	if result.Allowed {
		t.Error("4th message should be blocked by rate limit")
	}
	if result.Reason != "You're sending messages too quickly. Please slow down." {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}
}

func TestRepeatDetection(t *testing.T) {
	config := Config{
		Enabled:        true,
		MaxMessages:    10,
		TimeWindow:     10 * time.Second,
		RepeatCooldown: 1 * time.Second, // Short for testing
	}
	tracker := NewTracker(config)

	// First message should be allowed
	result := tracker.Check("hello world")
	if !result.Allowed {
		t.Error("First message should be allowed")
	}

	// Same message again should be blocked
	result = tracker.Check("hello world")
	if result.Allowed {
		t.Error("Repeat message should be blocked")
	}
	if result.Reason != "Please don't repeat the same message." {
		t.Errorf("Unexpected reason: %s", result.Reason)
	}

	// Different message should be allowed
	result = tracker.Check("different message")
	if !result.Allowed {
		t.Error("Different message should be allowed")
	}
}

func TestRepeatCooldownExpires(t *testing.T) {
	config := Config{
		Enabled:        true,
		MaxMessages:    10,
		TimeWindow:     10 * time.Second,
		RepeatCooldown: 50 * time.Millisecond, // Very short for testing
	}
	tracker := NewTracker(config)

	// First message
	result := tracker.Check("hello")
	if !result.Allowed {
		t.Error("First message should be allowed")
	}

	// Wait for cooldown to expire
	time.Sleep(60 * time.Millisecond)

	// Same message should now be allowed
	result = tracker.Check("hello")
	if !result.Allowed {
		t.Error("Message should be allowed after cooldown expires")
	}
}

func TestRateLimitExpires(t *testing.T) {
	config := Config{
		Enabled:        true,
		MaxMessages:    2,
		TimeWindow:     50 * time.Millisecond, // Very short for testing
		RepeatCooldown: 10 * time.Millisecond,
	}
	tracker := NewTracker(config)

	// Send 2 messages (hit limit)
	tracker.Check("a")
	tracker.Check("b")

	// 3rd should be blocked
	result := tracker.Check("c")
	if result.Allowed {
		t.Error("Should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed now
	result = tracker.Check("d")
	if !result.Allowed {
		t.Error("Should be allowed after rate limit window expires")
	}
}

func TestDisabled(t *testing.T) {
	config := Config{
		Enabled:        false,
		MaxMessages:    1,
		TimeWindow:     10 * time.Second,
		RepeatCooldown: 30 * time.Second,
	}
	tracker := NewTracker(config)

	// All messages should be allowed when disabled
	for i := 0; i < 10; i++ {
		result := tracker.Check("same message")
		if !result.Allowed {
			t.Errorf("Message %d should be allowed when antispam is disabled", i+1)
		}
	}
}

func TestReset(t *testing.T) {
	config := Config{
		Enabled:        true,
		MaxMessages:    2,
		TimeWindow:     10 * time.Second,
		RepeatCooldown: 30 * time.Second,
	}
	tracker := NewTracker(config)

	// Hit rate limit
	tracker.Check("a")
	tracker.Check("b")
	result := tracker.Check("c")
	if result.Allowed {
		t.Error("Should be rate limited")
	}

	// Reset
	tracker.Reset()

	// Should be allowed now
	result = tracker.Check("c")
	if !result.Allowed {
		t.Error("Should be allowed after reset")
	}
}
