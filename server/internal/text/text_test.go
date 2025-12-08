package text

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndGetText(t *testing.T) {
	content := `
welcome:
  banner: |
    =====================================
        Welcome to Test MUD!
    =====================================

guide:
  greeting: "Hello, %s!"
  tower: "The tower is tall, %s says."
  combat: "Fight well!"
  save: "Your progress is saved."
  shop: "Buy stuff with %d gold."
  portal: "Travel fast!"
  quests: "Do quests!"
  commands: "Type commands!"

bard:
  song: "The %s sings for %s!"

classes:
  abilities:
    warrior: "Strong attacks"
    mage: "Magic spells"
  welcome:
    warrior: "Welcome, warrior!"
    mage: "Welcome, mage!"
  trainer_accept:
    warrior: "%s welcomes %s to the warrior path."
    mage: "%s teaches %s magic."
  trainer_reject:
    warrior: "You lack strength."
    mage: "You lack intelligence."
  stat_recommendations:
    warrior: "STR 15, CON 14"
    mage: "INT 15, CON 14"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "text.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	txt, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load text: %v", err)
	}

	// Test welcome banner
	banner := txt.GetWelcomeBanner()
	if !strings.Contains(banner, "Welcome to Test MUD") {
		t.Errorf("Expected banner to contain 'Welcome to Test MUD', got %q", banner)
	}

	// Test guide greeting
	greeting := txt.GetGuideGreeting()
	if greeting != "Hello, %s!" {
		t.Errorf("Expected 'Hello, %%s!', got %q", greeting)
	}

	// Test guide topics
	tower := txt.GetGuideTopic("tower")
	if tower != "The tower is tall, %s says." {
		t.Errorf("Expected tower topic, got %q", tower)
	}

	// Test unknown topic
	unknown := txt.GetGuideTopic("unknown")
	if unknown != "" {
		t.Errorf("Expected empty string for unknown topic, got %q", unknown)
	}

	// Test bard song
	song := txt.GetBardSong()
	if song != "The %s sings for %s!" {
		t.Errorf("Expected bard song, got %q", song)
	}

	// Test class abilities
	abilities := txt.GetClassAbilities("warrior")
	if abilities != "Strong attacks" {
		t.Errorf("Expected 'Strong attacks', got %q", abilities)
	}

	// Test unknown class abilities
	unknownAbilities := txt.GetClassAbilities("unknown")
	if !strings.Contains(unknownAbilities, "No special abilities defined.") {
		t.Errorf("Expected fallback to contain 'No special abilities defined.', got %q", unknownAbilities)
	}

	// Test class welcome
	welcome := txt.GetClassWelcome("mage")
	if welcome != "Welcome, mage!" {
		t.Errorf("Expected 'Welcome, mage!', got %q", welcome)
	}

	// Test trainer accept
	accept := txt.GetTrainerAccept("warrior")
	if accept != "%s welcomes %s to the warrior path." {
		t.Errorf("Expected trainer accept, got %q", accept)
	}

	// Test trainer reject
	reject := txt.GetTrainerReject("mage")
	if reject != "You lack intelligence." {
		t.Errorf("Expected 'You lack intelligence.', got %q", reject)
	}

	// Test stat recommendations
	stats := txt.GetStatRecommendation("warrior")
	if !strings.Contains(stats, "STR 15") {
		t.Errorf("Expected stats to contain 'STR 15', got %q", stats)
	}
}

func TestLoadError(t *testing.T) {
	// Test loading non-existent file
	_, err := Load("/nonexistent/path/text.yaml")
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

func TestCaseInsensitivity(t *testing.T) {
	content := `
welcome:
  banner: "Welcome!"
guide:
  greeting: "Hi"
  tower: "Tower"
  combat: "Combat"
  save: "Save"
  shop: "Shop"
  portal: "Portal"
  quests: "Quests"
  commands: "Commands"
bard:
  song: "Song"
classes:
  abilities:
    warrior: "Warrior abilities"
  welcome:
    warrior: "Warrior welcome"
  trainer_accept:
    warrior: "Accepted"
  trainer_reject:
    warrior: "Rejected"
  stat_recommendations:
    warrior: "Stats"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "text.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	txt, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load text: %v", err)
	}

	// Test case insensitivity for class lookups
	if txt.GetClassAbilities("WARRIOR") != "Warrior abilities" {
		t.Error("Expected case-insensitive class abilities lookup")
	}
	if txt.GetClassAbilities("Warrior") != "Warrior abilities" {
		t.Error("Expected case-insensitive class abilities lookup")
	}
}
