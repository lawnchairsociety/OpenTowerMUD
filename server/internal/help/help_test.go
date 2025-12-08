package help

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndGetTopic(t *testing.T) {
	// Create temp test file
	content := `
topics:
  look:
    aliases:
      - look
      - l
    text: |
      LOOK command help text
  attack:
    aliases:
      - attack
      - kill
      - k
    text: |
      ATTACK command help text
general_help: |
  General help text
admin_help: |
  Admin help text
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "help.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	h, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load help: %v", err)
	}

	// Test getting topic by primary alias
	text := h.GetTopic("look")
	if text != "LOOK command help text" {
		t.Errorf("Expected 'LOOK command help text', got %q", text)
	}

	// Test getting topic by secondary alias
	text = h.GetTopic("l")
	if text != "LOOK command help text" {
		t.Errorf("Expected 'LOOK command help text', got %q", text)
	}

	// Test case insensitivity
	text = h.GetTopic("LOOK")
	if text != "LOOK command help text" {
		t.Errorf("Expected 'LOOK command help text', got %q", text)
	}

	// Test unknown topic
	text = h.GetTopic("unknown")
	if text != "" {
		t.Errorf("Expected empty string for unknown topic, got %q", text)
	}
}

func TestGetHelpText(t *testing.T) {
	content := `
topics:
  look:
    aliases:
      - look
    text: |
      LOOK help
general_help: |
  General help
admin_help: |
  Admin help
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "help.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	h, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load help: %v", err)
	}

	// Test general help for non-admin
	text := h.GetHelpText("", false)
	if text != "General help" {
		t.Errorf("Expected 'General help', got %q", text)
	}

	// Test general help for admin (includes admin section)
	text = h.GetHelpText("", true)
	expected := "General help\nAdmin help"
	if text != expected {
		t.Errorf("Expected %q, got %q", expected, text)
	}

	// Test specific topic
	text = h.GetHelpText("look", false)
	if text != "LOOK help" {
		t.Errorf("Expected 'LOOK help', got %q", text)
	}

	// Test unknown topic
	text = h.GetHelpText("unknown", false)
	if text == "" || text == "LOOK help" {
		t.Errorf("Expected 'no help' message, got %q", text)
	}
}

func TestLoadError(t *testing.T) {
	// Test loading non-existent file
	_, err := Load("/nonexistent/path/help.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test loading invalid YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte("not: valid: yaml: content:"), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	_, err = Load(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
