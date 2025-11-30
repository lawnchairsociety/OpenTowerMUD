package stats

// AbilityScores holds the six core D&D-style ability scores
type AbilityScores struct {
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
}

// StandardArray is the default set of values for new characters
var StandardArray = []int{15, 14, 13, 12, 10, 8}

// AbilityNames in order for character creation
var AbilityNames = []string{"Strength", "Dexterity", "Constitution", "Intelligence", "Wisdom", "Charisma"}

// Modifier calculates the D&D-style modifier using floor division
// Formula: floor((score - 10) / 2)
// Examples: 8=-1, 9=-1, 10=0, 11=0, 12=+1, 14=+2, 16=+3, 18=+4
func Modifier(score int) int {
	diff := score - 10
	if diff >= 0 {
		return diff / 2
	}
	// Floor division for negative numbers
	return (diff - 1) / 2
}

// NewDefaultScores returns ability scores with all values at 10
func NewDefaultScores() *AbilityScores {
	return &AbilityScores{
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
	}
}

// NewScores creates ability scores from individual values
func NewScores(str, dex, con, int_, wis, cha int) *AbilityScores {
	return &AbilityScores{
		Strength:     str,
		Dexterity:    dex,
		Constitution: con,
		Intelligence: int_,
		Wisdom:       wis,
		Charisma:     cha,
	}
}

// StrengthMod returns the strength modifier
func (a *AbilityScores) StrengthMod() int {
	return Modifier(a.Strength)
}

// DexterityMod returns the dexterity modifier
func (a *AbilityScores) DexterityMod() int {
	return Modifier(a.Dexterity)
}

// ConstitutionMod returns the constitution modifier
func (a *AbilityScores) ConstitutionMod() int {
	return Modifier(a.Constitution)
}

// IntelligenceMod returns the intelligence modifier
func (a *AbilityScores) IntelligenceMod() int {
	return Modifier(a.Intelligence)
}

// WisdomMod returns the wisdom modifier
func (a *AbilityScores) WisdomMod() int {
	return Modifier(a.Wisdom)
}

// CharismaMod returns the charisma modifier
func (a *AbilityScores) CharismaMod() int {
	return Modifier(a.Charisma)
}
