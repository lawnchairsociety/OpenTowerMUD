package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if len(cfg.WebSocket.AllowedOrigins) != 0 {
		t.Errorf("expected empty allowed origins by default, got %v", cfg.WebSocket.AllowedOrigins)
	}

	if cfg.WebSocket.MaxMessageSize != 4096 {
		t.Errorf("expected max message size 4096, got %d", cfg.WebSocket.MaxMessageSize)
	}
}

func TestLoadConfig_FileNotExists(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")

	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected default config for missing file, got nil")
	}

	// Should return defaults
	if len(cfg.WebSocket.AllowedOrigins) != 0 {
		t.Errorf("expected empty allowed origins by default")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "server.yaml")

	content := `
websocket:
  allowed_origins:
    - "https://example.com"
    - "http://localhost:3000"
  max_message_size: 8192
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.WebSocket.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(cfg.WebSocket.AllowedOrigins))
	}

	if cfg.WebSocket.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("expected first origin 'https://example.com', got %s", cfg.WebSocket.AllowedOrigins[0])
	}

	if cfg.WebSocket.MaxMessageSize != 8192 {
		t.Errorf("expected max message size 8192, got %d", cfg.WebSocket.MaxMessageSize)
	}
}

func TestIsOriginAllowed_EmptyList_SameOrigin(t *testing.T) {
	cfg := WebSocketConfig{
		AllowedOrigins: []string{},
	}

	// Same origin (no Origin header)
	if !cfg.IsOriginAllowed("", "localhost:4000") {
		t.Error("expected empty origin to be allowed (same-origin)")
	}

	// Same origin (matching host)
	if !cfg.IsOriginAllowed("http://localhost:4000", "localhost:4000") {
		t.Error("expected matching origin to be allowed (same-origin)")
	}

	// Different origin should be rejected
	if cfg.IsOriginAllowed("http://evil.com", "localhost:4000") {
		t.Error("expected different origin to be rejected (same-origin policy)")
	}
}

func TestIsOriginAllowed_Wildcard(t *testing.T) {
	cfg := WebSocketConfig{
		AllowedOrigins: []string{"*"},
	}

	// Wildcard allows everything
	if !cfg.IsOriginAllowed("http://anything.com", "localhost:4000") {
		t.Error("expected wildcard to allow any origin")
	}

	if !cfg.IsOriginAllowed("", "localhost:4000") {
		t.Error("expected wildcard to allow empty origin")
	}
}

func TestIsOriginAllowed_ExactMatch(t *testing.T) {
	cfg := WebSocketConfig{
		AllowedOrigins: []string{
			"https://example.com",
			"http://localhost:3000",
		},
	}

	// Exact matches
	if !cfg.IsOriginAllowed("https://example.com", "localhost:4000") {
		t.Error("expected exact match to be allowed")
	}

	if !cfg.IsOriginAllowed("http://localhost:3000", "localhost:4000") {
		t.Error("expected exact match to be allowed")
	}

	// Non-matching origin
	if cfg.IsOriginAllowed("http://evil.com", "localhost:4000") {
		t.Error("expected non-matching origin to be rejected")
	}

	// Partial match should not work
	if cfg.IsOriginAllowed("https://example.com:8080", "localhost:4000") {
		t.Error("expected partial match to be rejected")
	}
}

func TestIsSameOrigin(t *testing.T) {
	tests := []struct {
		origin      string
		requestHost string
		expected    bool
	}{
		{"", "localhost:4000", true},                                // No origin header
		{"http://localhost:4000", "localhost:4000", true},           // HTTP match
		{"https://localhost:4000", "localhost:4000", true},          // HTTPS match
		{"http://localhost:4000/", "localhost:4000", true},          // Trailing slash
		{"http://example.com", "localhost:4000", false},             // Different host
		{"http://localhost:3000", "localhost:4000", false},          // Different port
		{"ws://localhost:4000", "localhost:4000", true},             // WebSocket scheme
	}

	for _, tt := range tests {
		result := isSameOrigin(tt.origin, tt.requestHost)
		if result != tt.expected {
			t.Errorf("isSameOrigin(%q, %q) = %v, want %v",
				tt.origin, tt.requestHost, result, tt.expected)
		}
	}
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   PasswordConfig
		password string
		wantErr  bool
	}{
		{
			name:     "valid password with all requirements",
			config:   PasswordConfig{MinLength: 8, RequireUppercase: true, RequireLowercase: true, RequireDigit: true},
			password: "Password1",
			wantErr:  false,
		},
		{
			name:     "too short",
			config:   PasswordConfig{MinLength: 8},
			password: "Pass1",
			wantErr:  true,
		},
		{
			name:     "missing uppercase",
			config:   PasswordConfig{MinLength: 8, RequireUppercase: true},
			password: "password1",
			wantErr:  true,
		},
		{
			name:     "missing lowercase",
			config:   PasswordConfig{MinLength: 8, RequireLowercase: true},
			password: "PASSWORD1",
			wantErr:  true,
		},
		{
			name:     "missing digit",
			config:   PasswordConfig{MinLength: 8, RequireDigit: true},
			password: "Password",
			wantErr:  true,
		},
		{
			name:     "missing special char",
			config:   PasswordConfig{MinLength: 8, RequireSpecial: true},
			password: "Password1",
			wantErr:  true,
		},
		{
			name:     "valid with special char",
			config:   PasswordConfig{MinLength: 8, RequireSpecial: true},
			password: "Password1!",
			wantErr:  false,
		},
		{
			name:     "minimal requirements only",
			config:   PasswordConfig{MinLength: 4},
			password: "test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ValidatePassword(tt.password)
			gotErr := result != ""
			if gotErr != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v (msg: %s)",
					tt.password, gotErr, tt.wantErr, result)
			}
		})
	}
}

func TestGetRequirementsText(t *testing.T) {
	cfg := PasswordConfig{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   false,
	}

	text := cfg.GetRequirementsText()

	if text == "" {
		t.Error("expected non-empty requirements text")
	}

	if !strings.Contains(text, "8") {
		t.Errorf("expected text to contain '8', got %s", text)
	}

	if !strings.Contains(text, "uppercase") {
		t.Errorf("expected text to contain 'uppercase', got %s", text)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{8, "8"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
