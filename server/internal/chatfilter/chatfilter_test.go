package chatfilter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_NilConfig(t *testing.T) {
	cf := New(nil)
	if cf.IsEnabled() {
		t.Error("expected filter to be disabled with nil config")
	}
}

func TestNew_DisabledConfig(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		Mode:        ModeReplace,
		BannedWords: []string{"badword"},
	}
	cf := New(cfg)
	if cf.IsEnabled() {
		t.Error("expected filter to be disabled")
	}
}

func TestNew_EnabledConfig(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"badword"},
	}
	cf := New(cfg)
	if !cf.IsEnabled() {
		t.Error("expected filter to be enabled")
	}
	if !cf.IsReplaceMode() {
		t.Error("expected REPLACE mode")
	}
}

func TestCheck_DisabledFilter(t *testing.T) {
	cf := New(&Config{Enabled: false, BannedWords: []string{"badword"}})
	result := cf.Check("this is a badword test")
	if result.Violated {
		t.Error("disabled filter should not flag violations")
	}
	if result.Filtered != "this is a badword test" {
		t.Error("disabled filter should return message unchanged")
	}
}

func TestCheck_NoViolation(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"badword"},
	})
	result := cf.Check("this is a clean message")
	if result.Violated {
		t.Error("should not flag clean message")
	}
	if len(result.MatchedWords) != 0 {
		t.Error("should have no matched words")
	}
}

func TestCheck_ReplaceMode(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"badword"},
	})
	result := cf.Check("this is a badword test")
	if !result.Violated {
		t.Error("should flag violation")
	}
	if result.Filtered != "this is a ******* test" {
		t.Errorf("expected 'this is a ******* test', got '%s'", result.Filtered)
	}
	if len(result.MatchedWords) != 1 || result.MatchedWords[0] != "badword" {
		t.Error("should have matched 'badword'")
	}
}

func TestCheck_BlockMode(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeBlock,
		BannedWords: []string{"badword"},
	})
	result := cf.Check("this is a badword test")
	if !result.Violated {
		t.Error("should flag violation")
	}
	// In BLOCK mode, filtered message is unchanged (message gets blocked entirely)
	if result.Filtered != "this is a badword test" {
		t.Errorf("BLOCK mode should not modify message, got '%s'", result.Filtered)
	}
}

func TestCheck_CaseInsensitive(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"badword"},
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"BADWORD", "*******"},
		{"BadWord", "*******"},
		{"badword", "*******"},
		{"BaDwOrD", "*******"},
	}

	for _, test := range tests {
		result := cf.Check(test.input)
		if !result.Violated {
			t.Errorf("should flag '%s'", test.input)
		}
		if result.Filtered != test.expected {
			t.Errorf("input '%s': expected '%s', got '%s'", test.input, test.expected, result.Filtered)
		}
	}
}

func TestCheck_WordBoundary(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"bad"},
	})

	// Should match "bad" as a whole word
	result := cf.Check("this is bad")
	if !result.Violated {
		t.Error("should flag 'bad' as whole word")
	}

	// Should NOT match "bad" inside "badger"
	result = cf.Check("look at the badger")
	if result.Violated {
		t.Error("should not flag partial word match 'badger'")
	}

	// Should NOT match "bad" inside "notbad"
	result = cf.Check("notbad")
	if result.Violated {
		t.Error("should not flag partial word match 'notbad'")
	}
}

func TestCheck_MultipleWords(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"bad", "ugly"},
	})
	result := cf.Check("this is bad and ugly")
	if !result.Violated {
		t.Error("should flag violation")
	}
	if result.Filtered != "this is *** and ****" {
		t.Errorf("expected 'this is *** and ****', got '%s'", result.Filtered)
	}
	if len(result.MatchedWords) != 2 {
		t.Errorf("should have matched 2 words, got %d", len(result.MatchedWords))
	}
}

func TestCheck_MultipleOccurrences(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"bad"},
	})
	result := cf.Check("bad bad bad")
	if !result.Violated {
		t.Error("should flag violation")
	}
	if result.Filtered != "*** *** ***" {
		t.Errorf("expected '*** *** ***', got '%s'", result.Filtered)
	}
}

func TestCheck_EmptyBannedWords(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{},
	})
	result := cf.Check("anything goes")
	if result.Violated {
		t.Error("should not flag with empty banned words list")
	}
}

func TestCheck_PreservesPunctuation(t *testing.T) {
	cf := New(&Config{
		Enabled:     true,
		Mode:        ModeReplace,
		BannedWords: []string{"bad"},
	})
	result := cf.Check("this is bad!")
	if !result.Violated {
		t.Error("should flag violation")
	}
	if result.Filtered != "this is ***!" {
		t.Errorf("expected 'this is ***!', got '%s'", result.Filtered)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chat_filter.yaml")

	content := `enabled: true
mode: BLOCK
banned_words:
  - word1
  - word2
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.Enabled {
		t.Error("expected enabled to be true")
	}
	if cfg.Mode != ModeBlock {
		t.Errorf("expected mode BLOCK, got %s", cfg.Mode)
	}
	if len(cfg.BannedWords) != 2 {
		t.Errorf("expected 2 banned words, got %d", len(cfg.BannedWords))
	}
}

func TestLoadConfig_DefaultMode(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chat_filter.yaml")

	content := `enabled: true
banned_words:
  - word1
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Mode != ModeReplace {
		t.Errorf("expected default mode REPLACE, got %s", cfg.Mode)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestModeHelpers(t *testing.T) {
	cfReplace := New(&Config{Enabled: true, Mode: ModeReplace})
	if !cfReplace.IsReplaceMode() {
		t.Error("expected IsReplaceMode to be true")
	}
	if cfReplace.IsBlockMode() {
		t.Error("expected IsBlockMode to be false")
	}

	cfBlock := New(&Config{Enabled: true, Mode: ModeBlock})
	if cfBlock.IsReplaceMode() {
		t.Error("expected IsReplaceMode to be false")
	}
	if !cfBlock.IsBlockMode() {
		t.Error("expected IsBlockMode to be true")
	}
}
