package stats

import "testing"

func TestModifier(t *testing.T) {
	// D&D formula: floor((score - 10) / 2)
	// Uses floor division so odd scores round down (toward negative infinity)
	tests := []struct {
		score    int
		expected int
	}{
		{1, -5},   // Very low
		{6, -2},   // Low
		{7, -2},   // Low (floor division: -3/2 = -2)
		{8, -1},   // Below average
		{9, -1},   // Below average (floor division: -1/2 = -1)
		{10, 0},   // Average
		{11, 0},   // Average
		{12, 1},   // Above average
		{13, 1},   // Above average
		{14, 2},   // Good
		{15, 2},   // Good
		{16, 3},   // Very good
		{17, 3},   // Very good
		{18, 4},   // Excellent
		{19, 4},   // Excellent
		{20, 5},   // Exceptional
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := Modifier(tt.score)
			if result != tt.expected {
				t.Errorf("Modifier(%d) = %d, expected %d", tt.score, result, tt.expected)
			}
		})
	}
}

func TestNewDefaultScores(t *testing.T) {
	scores := NewDefaultScores()

	if scores.Strength != 10 {
		t.Errorf("Default Strength = %d, expected 10", scores.Strength)
	}
	if scores.Dexterity != 10 {
		t.Errorf("Default Dexterity = %d, expected 10", scores.Dexterity)
	}
	if scores.Constitution != 10 {
		t.Errorf("Default Constitution = %d, expected 10", scores.Constitution)
	}
	if scores.Intelligence != 10 {
		t.Errorf("Default Intelligence = %d, expected 10", scores.Intelligence)
	}
	if scores.Wisdom != 10 {
		t.Errorf("Default Wisdom = %d, expected 10", scores.Wisdom)
	}
	if scores.Charisma != 10 {
		t.Errorf("Default Charisma = %d, expected 10", scores.Charisma)
	}
}

func TestNewScores(t *testing.T) {
	scores := NewScores(15, 14, 13, 12, 10, 8)

	if scores.Strength != 15 {
		t.Errorf("Strength = %d, expected 15", scores.Strength)
	}
	if scores.Dexterity != 14 {
		t.Errorf("Dexterity = %d, expected 14", scores.Dexterity)
	}
	if scores.Constitution != 13 {
		t.Errorf("Constitution = %d, expected 13", scores.Constitution)
	}
	if scores.Intelligence != 12 {
		t.Errorf("Intelligence = %d, expected 12", scores.Intelligence)
	}
	if scores.Wisdom != 10 {
		t.Errorf("Wisdom = %d, expected 10", scores.Wisdom)
	}
	if scores.Charisma != 8 {
		t.Errorf("Charisma = %d, expected 8", scores.Charisma)
	}
}

func TestAbilityScoreModifiers(t *testing.T) {
	scores := NewScores(15, 14, 13, 12, 10, 8)

	if scores.StrengthMod() != 2 {
		t.Errorf("StrengthMod() = %d, expected 2", scores.StrengthMod())
	}
	if scores.DexterityMod() != 2 {
		t.Errorf("DexterityMod() = %d, expected 2", scores.DexterityMod())
	}
	if scores.ConstitutionMod() != 1 {
		t.Errorf("ConstitutionMod() = %d, expected 1", scores.ConstitutionMod())
	}
	if scores.IntelligenceMod() != 1 {
		t.Errorf("IntelligenceMod() = %d, expected 1", scores.IntelligenceMod())
	}
	if scores.WisdomMod() != 0 {
		t.Errorf("WisdomMod() = %d, expected 0", scores.WisdomMod())
	}
	if scores.CharismaMod() != -1 {
		t.Errorf("CharismaMod() = %d, expected -1", scores.CharismaMod())
	}
}

func TestStandardArray(t *testing.T) {
	expected := []int{15, 14, 13, 12, 10, 8}
	if len(StandardArray) != len(expected) {
		t.Errorf("StandardArray length = %d, expected %d", len(StandardArray), len(expected))
	}
	for i, v := range expected {
		if StandardArray[i] != v {
			t.Errorf("StandardArray[%d] = %d, expected %d", i, StandardArray[i], v)
		}
	}
}

func TestAbilityNames(t *testing.T) {
	expected := []string{"Strength", "Dexterity", "Constitution", "Intelligence", "Wisdom", "Charisma"}
	if len(AbilityNames) != len(expected) {
		t.Errorf("AbilityNames length = %d, expected %d", len(AbilityNames), len(expected))
	}
	for i, v := range expected {
		if AbilityNames[i] != v {
			t.Errorf("AbilityNames[%d] = %s, expected %s", i, AbilityNames[i], v)
		}
	}
}
