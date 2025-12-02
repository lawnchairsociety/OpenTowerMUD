package spells

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStringToEffectType(t *testing.T) {
	tests := []struct {
		input    string
		expected EffectType
	}{
		{"heal", EffectHeal},
		{"damage", EffectDamage},
		{"heal_percent", EffectHealPercent},
		{"unknown", EffectHeal}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StringToEffectType(tt.input)
			if result != tt.expected {
				t.Errorf("StringToEffectType(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringToTargetType(t *testing.T) {
	tests := []struct {
		input    string
		expected TargetType
	}{
		{"self", TargetSelf},
		{"enemy", TargetEnemy},
		{"ally", TargetAlly},
		{"unknown", TargetSelf}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StringToTargetType(tt.input)
			if result != tt.expected {
				t.Errorf("StringToTargetType(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateSpellFromDefinition(t *testing.T) {
	def := SpellDefinition{
		Name:        "test heal",
		Description: "A test healing spell",
		ManaCost:    10,
		Cooldown:    5,
		Level:       1,
		Effects: []SpellEffectDefinition{
			{Type: "heal", Target: "self", Amount: 20},
		},
	}

	spell := CreateSpellFromDefinition("test_heal", def)

	if spell.ID != "test_heal" {
		t.Errorf("Expected ID 'test_heal', got '%s'", spell.ID)
	}
	if spell.Name != "test heal" {
		t.Errorf("Expected name 'test heal', got '%s'", spell.Name)
	}
	if spell.Description != "A test healing spell" {
		t.Errorf("Expected description 'A test healing spell', got '%s'", spell.Description)
	}
	if spell.ManaCost != 10 {
		t.Errorf("Expected mana cost 10, got %d", spell.ManaCost)
	}
	if spell.Cooldown != 5 {
		t.Errorf("Expected cooldown 5, got %d", spell.Cooldown)
	}
	if spell.Level != 1 {
		t.Errorf("Expected level 1, got %d", spell.Level)
	}
	if len(spell.Effects) != 1 {
		t.Errorf("Expected 1 effect, got %d", len(spell.Effects))
	}
	if spell.Effects[0].Type != EffectHeal {
		t.Errorf("Expected effect type EffectHeal, got %v", spell.Effects[0].Type)
	}
	if spell.Effects[0].Target != TargetSelf {
		t.Errorf("Expected target TargetSelf, got %v", spell.Effects[0].Target)
	}
	if spell.Effects[0].Amount != 20 {
		t.Errorf("Expected amount 20, got %d", spell.Effects[0].Amount)
	}
}

func TestSpellRegistry(t *testing.T) {
	registry := NewSpellRegistry()

	// Add a spell manually
	spell := &Spell{
		ID:          "test_spell",
		Name:        "test spell",
		Description: "A test spell",
		ManaCost:    5,
		Cooldown:    10,
		Level:       1,
		Effects: []SpellEffect{
			{Type: EffectDamage, Target: TargetEnemy, Amount: 10},
		},
	}

	registry.spells["test_spell"] = spell

	// Test GetSpell
	retrieved, exists := registry.GetSpell("test_spell")
	if !exists {
		t.Error("Expected spell to exist")
	}
	if retrieved.ID != "test_spell" {
		t.Errorf("Expected ID 'test_spell', got '%s'", retrieved.ID)
	}

	// Test non-existent spell
	_, exists = registry.GetSpell("nonexistent")
	if exists {
		t.Error("Expected spell to not exist")
	}

	// Test GetAllSpells
	all := registry.GetAllSpells()
	if len(all) != 1 {
		t.Errorf("Expected 1 spell, got %d", len(all))
	}
}

func TestDefaultStarterSpells(t *testing.T) {
	// With the class system, DefaultStarterSpells now returns empty
	// Spells are automatically available based on class levels
	starters := DefaultStarterSpells()
	if len(starters) != 0 {
		t.Errorf("Expected 0 starter spells (class-based system), got %d", len(starters))
	}
}

func TestGetSpellsForClass(t *testing.T) {
	registry := NewSpellRegistry()

	// Add some class-restricted spells
	registry.spells["mage_spell"] = &Spell{ID: "mage_spell", Level: 1, AllowedClasses: []string{"mage"}}
	registry.spells["cleric_spell"] = &Spell{ID: "cleric_spell", Level: 1, AllowedClasses: []string{"cleric"}}
	registry.spells["universal"] = &Spell{ID: "universal", Level: 1, AllowedClasses: []string{}} // Empty = all classes
	registry.spells["high_level_mage"] = &Spell{ID: "high_level_mage", Level: 10, AllowedClasses: []string{"mage"}}

	// Test mage at level 5 - should get mage_spell and universal
	mageSpells := registry.GetSpellsForClass("mage", 5)
	if len(mageSpells) != 2 {
		t.Errorf("Expected 2 spells for mage at level 5, got %d", len(mageSpells))
	}

	// Test cleric at level 5 - should get cleric_spell and universal
	clericSpells := registry.GetSpellsForClass("cleric", 5)
	if len(clericSpells) != 2 {
		t.Errorf("Expected 2 spells for cleric at level 5, got %d", len(clericSpells))
	}

	// Test mage at level 10 - should get 3 spells including high_level_mage
	mageLevel10 := registry.GetSpellsForClass("mage", 10)
	if len(mageLevel10) != 3 {
		t.Errorf("Expected 3 spells for mage at level 10, got %d", len(mageLevel10))
	}

	// Test warrior - should only get universal spell
	warriorSpells := registry.GetSpellsForClass("warrior", 10)
	if len(warriorSpells) != 1 {
		t.Errorf("Expected 1 spell for warrior, got %d", len(warriorSpells))
	}
}

func TestGetSpellsForClasses(t *testing.T) {
	registry := NewSpellRegistry()

	// Add some class-restricted spells
	registry.spells["mage_spell"] = &Spell{ID: "mage_spell", Level: 1, AllowedClasses: []string{"mage"}}
	registry.spells["cleric_spell"] = &Spell{ID: "cleric_spell", Level: 5, AllowedClasses: []string{"cleric"}}
	registry.spells["universal"] = &Spell{ID: "universal", Level: 1, AllowedClasses: []string{}}

	// Multiclass character: mage 10, cleric 3
	classLevels := map[string]int{"mage": 10, "cleric": 3}
	spells := registry.GetSpellsForClasses(classLevels)

	// Should get mage_spell (mage 1), universal (any 1), but NOT cleric_spell (requires cleric 5)
	if len(spells) != 2 {
		t.Errorf("Expected 2 spells for multiclass mage 10/cleric 3, got %d", len(spells))
	}

	// Multiclass character: mage 10, cleric 5
	classLevels2 := map[string]int{"mage": 10, "cleric": 5}
	spells2 := registry.GetSpellsForClasses(classLevels2)

	// Should get all 3 spells
	if len(spells2) != 3 {
		t.Errorf("Expected 3 spells for multiclass mage 10/cleric 5, got %d", len(spells2))
	}
}

func TestLoadSpellsFromYAML(t *testing.T) {
	// Create a temporary test YAML file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_spells.yaml")

	yamlContent := `spells:
  test_heal:
    name: "test heal"
    description: "Heals the caster"
    mana_cost: 10
    cooldown: 5
    level: 1
    effects:
      - type: "heal"
        target: "self"
        amount: 15

  test_damage:
    name: "test damage"
    description: "Damages an enemy"
    mana_cost: 8
    cooldown: 3
    level: 1
    effects:
      - type: "damage"
        target: "enemy"
        amount: 10
`

	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load the spells
	registry := NewSpellRegistry()
	if err := registry.LoadFromYAML(testFile); err != nil {
		t.Fatalf("Failed to load spells: %v", err)
	}

	// Verify spells were loaded
	all := registry.GetAllSpells()
	if len(all) != 2 {
		t.Errorf("Expected 2 spells, got %d", len(all))
	}

	// Check test_heal
	heal, exists := registry.GetSpell("test_heal")
	if !exists {
		t.Error("Expected test_heal to exist")
	} else {
		if heal.ManaCost != 10 {
			t.Errorf("Expected mana cost 10, got %d", heal.ManaCost)
		}
		if heal.Cooldown != 5 {
			t.Errorf("Expected cooldown 5, got %d", heal.Cooldown)
		}
		if !heal.IsSelfOnly() {
			t.Error("Expected test_heal to be self-only")
		}
	}

	// Check test_damage
	damage, exists := registry.GetSpell("test_damage")
	if !exists {
		t.Error("Expected test_damage to exist")
	} else {
		if damage.ManaCost != 8 {
			t.Errorf("Expected mana cost 8, got %d", damage.ManaCost)
		}
		if damage.RequiresTarget() == false {
			t.Error("Expected test_damage to require target")
		}
	}
}

func TestGetSpellsByLevel(t *testing.T) {
	registry := NewSpellRegistry()

	// Add spells of different levels
	registry.spells["level1_spell"] = &Spell{ID: "level1_spell", Level: 1}
	registry.spells["level2_spell"] = &Spell{ID: "level2_spell", Level: 2}
	registry.spells["level5_spell"] = &Spell{ID: "level5_spell", Level: 5}
	registry.spells["level10_spell"] = &Spell{ID: "level10_spell", Level: 10}

	// Test getting spells up to level 2
	level2Spells := registry.GetSpellsByLevel(2)
	if len(level2Spells) != 2 {
		t.Errorf("Expected 2 spells at level 2 or below, got %d", len(level2Spells))
	}

	// Test getting spells up to level 5
	level5Spells := registry.GetSpellsByLevel(5)
	if len(level5Spells) != 3 {
		t.Errorf("Expected 3 spells at level 5 or below, got %d", len(level5Spells))
	}

	// Test getting all spells
	allSpells := registry.GetSpellsByLevel(10)
	if len(allSpells) != 4 {
		t.Errorf("Expected 4 spells at level 10 or below, got %d", len(allSpells))
	}
}
