package antispam

import (
	"sync"
	"time"
)

// Config holds anti-spam configuration
type Config struct {
	Enabled        bool          // Whether anti-spam is enabled
	MaxMessages    int           // Max messages allowed in the time window
	TimeWindow     time.Duration // Time window for rate limiting
	RepeatCooldown time.Duration // How long before the same message can be sent again
}

// DefaultConfig returns sensible defaults for anti-spam
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		MaxMessages:    5,
		TimeWindow:     10 * time.Second,
		RepeatCooldown: 30 * time.Second,
	}
}

// ConfigFromYAML creates a Config from YAML-loaded values
func ConfigFromYAML(enabled bool, maxMessages, timeWindowSeconds, repeatCooldownSeconds int) Config {
	cfg := DefaultConfig()
	cfg.Enabled = enabled
	if maxMessages > 0 {
		cfg.MaxMessages = maxMessages
	}
	if timeWindowSeconds > 0 {
		cfg.TimeWindow = time.Duration(timeWindowSeconds) * time.Second
	}
	if repeatCooldownSeconds > 0 {
		cfg.RepeatCooldown = time.Duration(repeatCooldownSeconds) * time.Second
	}
	return cfg
}

// Tracker tracks chat activity for a single player
type Tracker struct {
	mu             sync.Mutex
	config         Config
	enabled        bool                 // Cached enabled state
	messageTimes   []time.Time          // Timestamps of recent messages
	lastMessages   map[string]time.Time // message content -> last sent time
}

// NewTracker creates a new spam tracker with the given config
func NewTracker(config Config) *Tracker {
	return &Tracker{
		config:       config,
		enabled:      config.Enabled,
		messageTimes: make([]time.Time, 0, config.MaxMessages),
		lastMessages: make(map[string]time.Time),
	}
}

// CheckResult contains the result of a spam check
type CheckResult struct {
	Allowed       bool
	Reason        string
	WaitSeconds   int // How long to wait before trying again (if not allowed)
}

// Check determines if a message should be allowed
// Returns CheckResult indicating if the message is allowed and why not if blocked
func (t *Tracker) Check(message string) CheckResult {
	// If anti-spam is disabled, always allow
	if !t.enabled {
		return CheckResult{Allowed: true}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// Clean up old entries
	t.cleanup(now)

	// Check for duplicate message
	if lastTime, exists := t.lastMessages[message]; exists {
		elapsed := now.Sub(lastTime)
		if elapsed < t.config.RepeatCooldown {
			remaining := t.config.RepeatCooldown - elapsed
			return CheckResult{
				Allowed:     false,
				Reason:      "Please don't repeat the same message.",
				WaitSeconds: int(remaining.Seconds()) + 1,
			}
		}
	}

	// Check rate limit
	if len(t.messageTimes) >= t.config.MaxMessages {
		// Find when the oldest message will expire
		oldest := t.messageTimes[0]
		waitUntil := oldest.Add(t.config.TimeWindow)
		remaining := waitUntil.Sub(now)
		return CheckResult{
			Allowed:     false,
			Reason:      "You're sending messages too quickly. Please slow down.",
			WaitSeconds: int(remaining.Seconds()) + 1,
		}
	}

	// Message allowed - record it
	t.messageTimes = append(t.messageTimes, now)
	t.lastMessages[message] = now

	return CheckResult{Allowed: true}
}

// cleanup removes expired entries
func (t *Tracker) cleanup(now time.Time) {
	// Clean up old message times (outside the time window)
	cutoff := now.Add(-t.config.TimeWindow)
	newTimes := t.messageTimes[:0]
	for _, msgTime := range t.messageTimes {
		if msgTime.After(cutoff) {
			newTimes = append(newTimes, msgTime)
		}
	}
	t.messageTimes = newTimes

	// Clean up old last messages (outside repeat cooldown)
	repeatCutoff := now.Add(-t.config.RepeatCooldown)
	for msg, msgTime := range t.lastMessages {
		if msgTime.Before(repeatCutoff) {
			delete(t.lastMessages, msg)
		}
	}
}

// Reset clears all tracking data (useful for testing or admin reset)
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messageTimes = make([]time.Time, 0, t.config.MaxMessages)
	t.lastMessages = make(map[string]time.Time)
}
