package namefilter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_NilConfig(t *testing.T) {
	nf := New(nil)

	if nf.enabled {
		t.Error("Filter should be disabled when config is nil")
	}

	// Should allow any name when disabled
	result := nf.Check("admin")
	if !result.Allowed {
		t.Error("Should allow any name when filter is disabled")
	}
}

func TestNew_DisabledConfig(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		BannedWords: []string{"admin"},
		BannedNames: []string{"root"},
	}

	nf := New(cfg)

	if nf.enabled {
		t.Error("Filter should be disabled")
	}

	// Should allow banned words when disabled
	result := nf.Check("admin")
	if !result.Allowed {
		t.Error("Should allow banned words when filter is disabled")
	}

	result = nf.Check("root")
	if !result.Allowed {
		t.Error("Should allow banned names when filter is disabled")
	}
}

func TestCheck_BannedWords(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		BannedWords: []string{"admin", "moderator", "gm"},
	}

	nf := New(cfg)

	tests := []struct {
		name    string
		allowed bool
	}{
		{"admin", false},             // Exact match
		{"Admin", false},             // Case insensitive
		{"ADMIN", false},             // All caps
		{"superadmin", false},        // Contains banned word
		{"adminuser", false},         // Contains banned word
		{"theadmin123", false},       // Contains banned word
		{"moderator", false},         // Another banned word
		{"gamemoderator", false},     // Contains banned word
		{"gm", false},                // Short banned word
		{"gmmaster", false},          // Contains banned word
		{"player", true},             // Valid name
		{"knight", true},             // Valid name
		{"wizard", true},             // Valid name
		{"admi", true},               // Partial, doesn't contain "admin"
		{"mod", true},                // Partial, doesn't contain "moderator"
	}

	for _, tc := range tests {
		result := nf.Check(tc.name)
		if result.Allowed != tc.allowed {
			t.Errorf("Check(%q) = %v, want %v", tc.name, result.Allowed, tc.allowed)
		}
		if !tc.allowed && result.Reason == "" {
			t.Errorf("Check(%q) should have a rejection reason", tc.name)
		}
	}
}

func TestCheck_BannedNames(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		BannedNames: []string{"root", "system", "null"},
	}

	nf := New(cfg)

	tests := []struct {
		name    string
		allowed bool
	}{
		{"root", false},       // Exact match
		{"Root", false},       // Case insensitive
		{"ROOT", false},       // All caps
		{"system", false},     // Exact match
		{"null", false},       // Exact match
		{"rootuser", true},    // Not exact match (banned names are exact only)
		{"superroot", true},   // Not exact match
		{"systemic", true},    // Not exact match
		{"nullify", true},     // Not exact match
		{"player", true},      // Valid name
	}

	for _, tc := range tests {
		result := nf.Check(tc.name)
		if result.Allowed != tc.allowed {
			t.Errorf("Check(%q) = %v, want %v", tc.name, result.Allowed, tc.allowed)
		}
	}
}

func TestCheck_BothWordsAndNames(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		BannedWords: []string{"admin"},
		BannedNames: []string{"root"},
	}

	nf := New(cfg)

	// Banned word (partial match)
	result := nf.Check("superadmin")
	if result.Allowed {
		t.Error("Should reject name containing banned word")
	}

	// Banned name (exact match only)
	result = nf.Check("root")
	if result.Allowed {
		t.Error("Should reject exact banned name")
	}

	// Banned name substring should be allowed (not partial match)
	result = nf.Check("rootling")
	if !result.Allowed {
		t.Error("Should allow name that only contains banned name as substring")
	}
}

func TestCheck_EmptyBannedEntries(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		BannedWords: []string{"admin", "", "moderator"}, // Empty string in list
		BannedNames: []string{"", "root"},               // Empty string in list
	}

	nf := New(cfg)

	// Should still work with empty entries filtered out
	result := nf.Check("admin")
	if result.Allowed {
		t.Error("Should reject banned word")
	}

	result = nf.Check("player")
	if !result.Allowed {
		t.Error("Should allow valid name")
	}
}

func TestCheck_ReasonMessages(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		BannedWords: []string{"admin"},
		BannedNames: []string{"root"},
	}

	nf := New(cfg)

	// Check banned word reason
	result := nf.Check("superadmin")
	if result.Allowed {
		t.Fatal("Should reject banned word")
	}
	if result.Reason == "" {
		t.Error("Should provide reason for rejection")
	}

	// Check banned name reason
	result = nf.Check("root")
	if result.Allowed {
		t.Fatal("Should reject banned name")
	}
	if result.Reason == "" {
		t.Error("Should provide reason for rejection")
	}

	// Valid name should have no reason
	result = nf.Check("player")
	if !result.Allowed {
		t.Fatal("Should allow valid name")
	}
	if result.Reason != "" {
		t.Error("Should not have reason for allowed name")
	}
}

func TestIsEnabled(t *testing.T) {
	// Enabled filter
	cfg := &Config{Enabled: true}
	nf := New(cfg)
	if !nf.IsEnabled() {
		t.Error("IsEnabled should return true for enabled filter")
	}

	// Disabled filter
	cfg = &Config{Enabled: false}
	nf = New(cfg)
	if nf.IsEnabled() {
		t.Error("IsEnabled should return false for disabled filter")
	}

	// Nil config
	nf = New(nil)
	if nf.IsEnabled() {
		t.Error("IsEnabled should return false for nil config")
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "name_filter.yaml")

	// Create a test config file
	configContent := `enabled: true
banned_words:
  - admin
  - moderator
banned_names:
  - root
  - system
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if !cfg.Enabled {
		t.Error("Config should be enabled")
	}

	if len(cfg.BannedWords) != 2 {
		t.Errorf("Expected 2 banned words, got %d", len(cfg.BannedWords))
	}

	if len(cfg.BannedNames) != 2 {
		t.Errorf("Expected 2 banned names, got %d", len(cfg.BannedNames))
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig should return error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should return error for invalid YAML")
	}
}

// TestCheck_RealWorldScenarios tests common scenarios for name filtering
func TestCheck_RealWorldScenarios(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		BannedWords: []string{
			"admin",
			"moderator",
			"gamemaster",
			"staff",
			"developer",
			"owner",
			"operator",
			"sysop",
			"gm",
		},
		BannedNames: []string{
			"god",
			"jesus",
			"satan",
		},
	}

	nf := New(cfg)

	// Valid fantasy names
	validNames := []string{
		"Gandalf",
		"Aragorn",
		"Frodo",
		"Legolas",
		"DragonSlayer",
		"ShadowHunter",
		"IronFist",
		"MysticMage",
		"player123",
		"xXWarriorXx",
	}

	for _, name := range validNames {
		result := nf.Check(name)
		if !result.Allowed {
			t.Errorf("Valid name %q should be allowed", name)
		}
	}

	// Invalid impersonation attempts
	invalidNames := []string{
		"Admin",
		"GameAdmin",
		"SuperAdmin",
		"Moderator",
		"HeadModerator",
		"GM_Player",
		"PlayerGM",
		"Gamemaster",
		"Staff_Helper",
		"Developer123",
	}

	for _, name := range invalidNames {
		result := nf.Check(name)
		if result.Allowed {
			t.Errorf("Invalid name %q should be rejected", name)
		}
	}
}
