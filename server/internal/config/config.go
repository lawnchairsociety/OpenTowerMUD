package config

import (
	"os"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds server-wide configuration settings.
type ServerConfig struct {
	WebSocket   WebSocketConfig   `yaml:"websocket"`
	Password    PasswordConfig    `yaml:"password"`
	Connections ConnectionsConfig `yaml:"connections"`
	RateLimit   RateLimitConfig   `yaml:"rate_limit"`
}

// RateLimitConfig holds rate limiting settings for login attempts.
type RateLimitConfig struct {
	// MaxAttempts is the maximum login attempts before lockout.
	MaxAttempts int `yaml:"max_attempts"`

	// LockoutSeconds is the initial lockout duration in seconds.
	LockoutSeconds int `yaml:"lockout_seconds"`

	// MaxLockoutSeconds is the maximum lockout duration (for exponential backoff).
	MaxLockoutSeconds int `yaml:"max_lockout_seconds"`
}

// ConnectionsConfig holds connection limit settings.
type ConnectionsConfig struct {
	// MaxPerIP is the maximum concurrent connections allowed from a single IP address.
	// 0 means unlimited (not recommended).
	MaxPerIP int `yaml:"max_per_ip"`

	// MaxTotal is the maximum total concurrent connections to the server.
	// 0 means unlimited.
	MaxTotal int `yaml:"max_total"`
}

// PasswordConfig holds password validation settings.
type PasswordConfig struct {
	// MinLength is the minimum password length (default: 8)
	MinLength int `yaml:"min_length"`

	// RequireUppercase requires at least one uppercase letter
	RequireUppercase bool `yaml:"require_uppercase"`

	// RequireLowercase requires at least one lowercase letter
	RequireLowercase bool `yaml:"require_lowercase"`

	// RequireDigit requires at least one digit
	RequireDigit bool `yaml:"require_digit"`

	// RequireSpecial requires at least one special character
	RequireSpecial bool `yaml:"require_special"`
}

// WebSocketConfig holds WebSocket-specific settings.
type WebSocketConfig struct {
	// AllowedOrigins is a list of origins allowed to connect via WebSocket.
	// Empty list enforces same-origin policy.
	// Use "*" to allow all origins (not recommended for production).
	AllowedOrigins []string `yaml:"allowed_origins"`

	// MaxMessageSize is the maximum WebSocket message size in bytes.
	MaxMessageSize int64 `yaml:"max_message_size"`
}

// DefaultConfig returns a ServerConfig with secure defaults.
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		WebSocket: WebSocketConfig{
			AllowedOrigins: []string{}, // Same-origin only by default
			MaxMessageSize: 4096,
		},
		Password: PasswordConfig{
			MinLength:        8,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireDigit:     true,
			RequireSpecial:   false, // Not required by default for usability
		},
		Connections: ConnectionsConfig{
			MaxPerIP: 3,   // Default: 3 connections per IP
			MaxTotal: 100, // Default: 100 total connections
		},
		RateLimit: RateLimitConfig{
			MaxAttempts:       5,   // Default: 5 attempts before lockout
			LockoutSeconds:    30,  // Default: 30 second initial lockout
			MaxLockoutSeconds: 300, // Default: 5 minute max lockout
		},
	}
}

// LoadConfig loads server configuration from a YAML file.
// If the file doesn't exist or can't be parsed, returns default config.
func LoadConfig(path string) (*ServerConfig, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Use defaults if file doesn't exist
		}
		return config, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return DefaultConfig(), err
	}

	return config, nil
}

// IsOriginAllowed checks if the given origin is allowed based on the config.
// Returns true if:
// - AllowedOrigins contains "*" (allow all)
// - AllowedOrigins contains the exact origin
// - AllowedOrigins is empty and origin matches the request host (same-origin)
func (c *WebSocketConfig) IsOriginAllowed(origin, requestHost string) bool {
	// If no origins configured, enforce same-origin policy
	if len(c.AllowedOrigins) == 0 {
		return isSameOrigin(origin, requestHost)
	}

	for _, allowed := range c.AllowedOrigins {
		// Wildcard allows all origins
		if allowed == "*" {
			return true
		}
		// Exact match
		if allowed == origin {
			return true
		}
	}

	return false
}

// isSameOrigin checks if the origin matches the request host (same-origin policy).
func isSameOrigin(origin, requestHost string) bool {
	if origin == "" {
		return true // No origin header means same-origin (e.g., non-browser client)
	}

	// Extract host from origin URL (e.g., "http://localhost:3000" -> "localhost:3000")
	originHost := origin
	if idx := strings.Index(origin, "://"); idx != -1 {
		originHost = origin[idx+3:]
	}
	// Remove trailing slash if present
	originHost = strings.TrimSuffix(originHost, "/")

	return originHost == requestHost
}

// ValidatePassword checks if a password meets the configured requirements.
// Returns an error message describing what's wrong, or empty string if valid.
func (c *PasswordConfig) ValidatePassword(password string) string {
	// Check minimum length
	minLen := c.MinLength
	if minLen == 0 {
		minLen = 8 // Default if not set
	}
	if len(password) < minLen {
		return "Password must be at least " + itoa(minLen) + " characters."
	}

	// Check character requirements
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if c.RequireUppercase && !hasUpper {
		return "Password must contain at least one uppercase letter."
	}
	if c.RequireLowercase && !hasLower {
		return "Password must contain at least one lowercase letter."
	}
	if c.RequireDigit && !hasDigit {
		return "Password must contain at least one digit."
	}
	if c.RequireSpecial && !hasSpecial {
		return "Password must contain at least one special character."
	}

	return ""
}

// GetRequirementsText returns a human-readable description of password requirements.
func (c *PasswordConfig) GetRequirementsText() string {
	minLen := c.MinLength
	if minLen == 0 {
		minLen = 8
	}

	var parts []string
	parts = append(parts, "min "+itoa(minLen)+" chars")

	if c.RequireUppercase {
		parts = append(parts, "uppercase")
	}
	if c.RequireLowercase {
		parts = append(parts, "lowercase")
	}
	if c.RequireDigit {
		parts = append(parts, "digit")
	}
	if c.RequireSpecial {
		parts = append(parts, "special char")
	}

	return strings.Join(parts, ", ")
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
