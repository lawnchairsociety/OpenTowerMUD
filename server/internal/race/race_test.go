package race

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRaceIsValid(t *testing.T) {
	tests := []struct {
		race  Race
		valid bool
	}{
		{Human, true},
		{Dwarf, true},
		{Elf, true},
		{Gnome, true},
		{Orc, true},
		{Race("invalid"), false},
		{Race(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.race), func(t *testing.T) {
			if got := tt.race.IsValid(); got != tt.valid {
				t.Errorf("Race(%q).IsValid() = %v, want %v", tt.race, got, tt.valid)
			}
		})
	}
}

func TestRaceString(t *testing.T) {
	tests := []struct {
		race     Race
		expected string
	}{
		{Human, "Human"},
		{Dwarf, "Dwarf"},
		{Elf, "Elf"},
		{Gnome, "Gnome"},
		{Orc, "Orc"},
		{Race("invalid"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.race), func(t *testing.T) {
			if got := tt.race.String(); got != tt.expected {
				t.Errorf("Race(%q).String() = %q, want %q", tt.race, got, tt.expected)
			}
		})
	}
}

func TestParseRace(t *testing.T) {
	tests := []struct {
		input    string
		expected Race
		hasError bool
	}{
		{"human", Human, false},
		{"Human", Human, false},
		{"HUMAN", Human, false},
		{"dwarf", Dwarf, false},
		{"elf", Elf, false},
		{"gnome", Gnome, false},
		{"orc", Orc, false},
		{"Orc", Orc, false},
		{"invalid", Race(""), true},
		{"", Race(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseRace(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ParseRace(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseRace(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ParseRace(%q) = %q, want %q", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestAllRaces(t *testing.T) {
	races := AllRaces()

	// Should return all 5 races
	if len(races) != 5 {
		t.Errorf("AllRaces() returned %d races, want 5", len(races))
	}

	// Should be in consistent order
	expected := []Race{Human, Dwarf, Elf, Gnome, Orc}
	for i, r := range races {
		if r != expected[i] {
			t.Errorf("AllRaces()[%d] = %q, want %q", i, r, expected[i])
		}
	}
}

func TestGetDefinition(t *testing.T) {
	// Test valid race
	def := GetDefinition(Dwarf)
	if def == nil {
		t.Fatal("GetDefinition(Dwarf) returned nil")
	}
	if def.Name != Dwarf {
		t.Errorf("Definition.Name = %q, want %q", def.Name, Dwarf)
	}
	if def.Size != "Medium" {
		t.Errorf("Definition.Size = %q, want 'Medium'", def.Size)
	}

	// Test stat bonuses for dwarf
	if bonus, ok := def.StatBonuses["CON"]; !ok || bonus != 2 {
		t.Errorf("Dwarf CON bonus = %d, want 2", bonus)
	}
	if bonus, ok := def.StatBonuses["CHA"]; !ok || bonus != -2 {
		t.Errorf("Dwarf CHA bonus = %d, want -2", bonus)
	}

	// Test invalid race
	def = GetDefinition(Race("invalid"))
	if def != nil {
		t.Error("GetDefinition(invalid) should return nil")
	}
}

func TestDefinition_ApplyStatBonuses(t *testing.T) {
	tests := []struct {
		race                                          Race
		str, dex, con, int_, wis, cha                 int
		expStr, expDex, expCon, expInt, expWis, expCha int
	}{
		// Dwarf: +2 CON, -2 CHA
		{Dwarf, 10, 10, 10, 10, 10, 10, 10, 10, 12, 10, 10, 8},
		// Elf: +2 DEX, -2 CON
		{Elf, 10, 10, 10, 10, 10, 10, 10, 12, 8, 10, 10, 10},
		// Gnome: +2 CON, -2 STR
		{Gnome, 10, 10, 10, 10, 10, 10, 8, 10, 12, 10, 10, 10},
		// Orc: +2 STR, -2 INT
		{Orc, 10, 10, 10, 10, 10, 10, 12, 10, 10, 8, 10, 10},
		// Human: no bonuses (human gets +1 choice at creation)
		{Human, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(string(tt.race), func(t *testing.T) {
			def := GetDefinition(tt.race)
			if def == nil {
				t.Fatalf("GetDefinition(%q) returned nil", tt.race)
			}
			str, dex, con, int_, wis, cha := def.ApplyStatBonuses(
				tt.str, tt.dex, tt.con, tt.int_, tt.wis, tt.cha,
			)
			if str != tt.expStr || dex != tt.expDex || con != tt.expCon ||
				int_ != tt.expInt || wis != tt.expWis || cha != tt.expCha {
				t.Errorf("ApplyStatBonuses() = (%d,%d,%d,%d,%d,%d), want (%d,%d,%d,%d,%d,%d)",
					str, dex, con, int_, wis, cha,
					tt.expStr, tt.expDex, tt.expCon, tt.expInt, tt.expWis, tt.expCha)
			}
		})
	}
}

func TestDefinition_GetStatBonusesString(t *testing.T) {
	tests := []struct {
		race     Race
		expected string
	}{
		{Dwarf, "+2 CON, -2 CHA"},
		{Elf, "+2 DEX, -2 CON"},
		{Gnome, "-2 STR, +2 CON"}, // Order is STR, DEX, CON, INT, WIS, CHA
		{Orc, "+2 STR, -2 INT"},
		{Human, "+1 to one ability (your choice)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.race), func(t *testing.T) {
			def := GetDefinition(tt.race)
			if def == nil {
				t.Fatalf("GetDefinition(%q) returned nil", tt.race)
			}
			got := def.GetStatBonusesString()
			if got != tt.expected {
				t.Errorf("GetStatBonusesString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefinition_GetAbilitiesString(t *testing.T) {
	def := GetDefinition(Dwarf)
	if def == nil {
		t.Fatal("GetDefinition(Dwarf) returned nil")
	}
	abilities := def.GetAbilitiesString()
	if abilities == "" || abilities == "None" {
		t.Error("Dwarf should have abilities")
	}
}

func TestDefinition_HasStatBonus(t *testing.T) {
	tests := []struct {
		race     Race
		expected bool
	}{
		{Dwarf, true},
		{Elf, true},
		{Orc, true},
		{Human, false}, // Human gets choice at creation, not fixed bonuses
	}

	for _, tt := range tests {
		t.Run(string(tt.race), func(t *testing.T) {
			def := GetDefinition(tt.race)
			if def == nil {
				t.Fatalf("GetDefinition(%q) returned nil", tt.race)
			}
			if got := def.HasStatBonus(); got != tt.expected {
				t.Errorf("HasStatBonus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadRacesFromYAML(t *testing.T) {
	// Create a temporary YAML file for testing
	content := `races:
  test-race:
    name: "Test Race"
    description: "A test race for unit testing"
    size: "Medium"
    stat_bonuses:
      STR: 1
      DEX: -1
    abilities:
      - "Test Ability"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_races.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Save original global config
	originalConfig := globalConfig

	// Load the test config
	config, err := LoadRacesFromYAML(tmpFile)
	if err != nil {
		t.Fatalf("LoadRacesFromYAML() error: %v", err)
	}

	// Verify loaded config
	if config == nil {
		t.Fatal("LoadRacesFromYAML() returned nil config")
	}
	if len(config.Races) != 1 {
		t.Errorf("Expected 1 race, got %d", len(config.Races))
	}

	testRace, ok := config.Races["test-race"]
	if !ok {
		t.Fatal("test-race not found in loaded config")
	}
	if testRace.Name != "Test Race" {
		t.Errorf("Name = %q, want 'Test Race'", testRace.Name)
	}
	if testRace.StatBonuses["STR"] != 1 {
		t.Errorf("STR bonus = %d, want 1", testRace.StatBonuses["STR"])
	}
	if testRace.StatBonuses["DEX"] != -1 {
		t.Errorf("DEX bonus = %d, want -1", testRace.StatBonuses["DEX"])
	}

	// Verify global config was set
	if !Race("test-race").IsValid() {
		t.Error("test-race should be valid after loading")
	}

	// Restore original config
	globalConfig = originalConfig
}

func TestLoadRacesFromYAML_FileNotFound(t *testing.T) {
	_, err := LoadRacesFromYAML("/nonexistent/path/races.yaml")
	if err == nil {
		t.Error("LoadRacesFromYAML() should return error for nonexistent file")
	}
}

func TestLoadRacesFromYAML_InvalidYAML(t *testing.T) {
	// Create a temporary invalid YAML file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	_, err := LoadRacesFromYAML(tmpFile)
	if err == nil {
		t.Error("LoadRacesFromYAML() should return error for invalid YAML")
	}
}

func TestSetGlobalConfig(t *testing.T) {
	// Save original
	original := globalConfig

	// Test setting nil (should be no-op)
	SetGlobalConfig(nil)
	if globalConfig != original {
		t.Error("SetGlobalConfig(nil) should not change global config")
	}

	// Test setting valid config
	newConfig := &RacesConfig{
		Races: map[string]*RaceDefinition{
			"custom": {Name: "Custom"},
		},
	}
	SetGlobalConfig(newConfig)
	if globalConfig != newConfig {
		t.Error("SetGlobalConfig() did not set global config")
	}

	// Restore original
	globalConfig = original
}

func TestSmallRaces(t *testing.T) {
	smallRaces := []Race{Gnome}
	for _, r := range smallRaces {
		def := GetDefinition(r)
		if def == nil {
			t.Fatalf("GetDefinition(%q) returned nil", r)
		}
		if def.Size != "Small" {
			t.Errorf("%s Size = %q, want 'Small'", r, def.Size)
		}
	}
}

func TestMediumRaces(t *testing.T) {
	mediumRaces := []Race{Human, Dwarf, Elf, Orc}
	for _, r := range mediumRaces {
		def := GetDefinition(r)
		if def == nil {
			t.Fatalf("GetDefinition(%q) returned nil", r)
		}
		if def.Size != "Medium" {
			t.Errorf("%s Size = %q, want 'Medium'", r, def.Size)
		}
	}
}
