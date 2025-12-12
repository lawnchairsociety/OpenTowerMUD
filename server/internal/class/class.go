// Package class defines the player class system for OpenTowerMUD.
package class

import (
	"fmt"
	"strings"
)

// Class represents a player class
type Class string

const (
	Warrior Class = "warrior"
	Mage    Class = "mage"
	Cleric  Class = "cleric"
	Rogue   Class = "rogue"
	Ranger  Class = "ranger"
	Paladin Class = "paladin"
)

// AllClasses returns all valid classes
func AllClasses() []Class {
	return []Class{Warrior, Mage, Cleric, Rogue, Ranger, Paladin}
}

// IsValid returns true if the class is a valid class
func (c Class) IsValid() bool {
	switch c {
	case Warrior, Mage, Cleric, Rogue, Ranger, Paladin:
		return true
	default:
		return false
	}
}

// String returns the display name of the class
func (c Class) String() string {
	switch c {
	case Warrior:
		return "Warrior"
	case Mage:
		return "Mage"
	case Cleric:
		return "Cleric"
	case Rogue:
		return "Rogue"
	case Ranger:
		return "Ranger"
	case Paladin:
		return "Paladin"
	default:
		return "Unknown"
	}
}

// ParseClass parses a string into a Class, case-insensitive
func ParseClass(s string) (Class, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "warrior":
		return Warrior, nil
	case "mage":
		return Mage, nil
	case "cleric":
		return Cleric, nil
	case "rogue":
		return Rogue, nil
	case "ranger":
		return Ranger, nil
	case "paladin":
		return Paladin, nil
	default:
		return "", fmt.Errorf("unknown class: %s", s)
	}
}

// Definition contains the static definition for a class
type Definition struct {
	Name        Class
	Description string
	HitDie      int    // e.g., 10 for d10
	PrimaryStat string // e.g., "STR"

	// Proficiencies
	ArmorProficiencies  []ArmorType
	WeaponProficiencies []WeaponType

	// Starting stats
	StartingHP   int // Base HP before CON modifier
	StartingMana int // Base mana before casting stat modifier

	// Per-level gains
	ManaPerLevel int // Base mana gain per level (before modifier)

	// Multiclass requirements
	MulticlassRequirements map[string]int // stat name -> minimum value
}

// ArmorType represents a category of armor proficiency
type ArmorType string

const (
	ArmorNone   ArmorType = "none"
	ArmorLight  ArmorType = "light"
	ArmorMedium ArmorType = "medium"
	ArmorHeavy  ArmorType = "heavy"
	ArmorShield ArmorType = "shield"
)

// WeaponType represents a category of weapon proficiency
type WeaponType string

const (
	WeaponSimple  WeaponType = "simple"
	WeaponMartial WeaponType = "martial"
	WeaponFinesse WeaponType = "finesse"
	WeaponRanged  WeaponType = "ranged"
)

// Definitions contains all class definitions
var Definitions = map[Class]*Definition{
	Warrior: {
		Name:                "Warrior",
		Description:         "Master of arms and armor, front-line fighter",
		HitDie:              10,
		PrimaryStat:         "STR",
		ArmorProficiencies:  []ArmorType{ArmorLight, ArmorMedium, ArmorHeavy, ArmorShield},
		WeaponProficiencies: []WeaponType{WeaponSimple, WeaponMartial},
		StartingHP:          10,
		StartingMana:        0,
		ManaPerLevel:        0,
		MulticlassRequirements: map[string]int{
			"STR": 13,
		},
	},
	Mage: {
		Name:                "Mage",
		Description:         "Master of arcane magic, glass cannon",
		HitDie:              6,
		PrimaryStat:         "INT",
		ArmorProficiencies:  []ArmorType{ArmorNone},
		WeaponProficiencies: []WeaponType{WeaponSimple}, // Limited to daggers, staves
		StartingHP:          8,
		StartingMana:        20,
		ManaPerLevel:        5,
		MulticlassRequirements: map[string]int{
			"INT": 13,
		},
	},
	Cleric: {
		Name:                "Cleric",
		Description:         "Divine healer and support, armored caster",
		HitDie:              8,
		PrimaryStat:         "WIS",
		ArmorProficiencies:  []ArmorType{ArmorLight, ArmorMedium, ArmorShield},
		WeaponProficiencies: []WeaponType{WeaponSimple}, // Blunt only
		StartingHP:          8,
		StartingMana:        15,
		ManaPerLevel:        4,
		MulticlassRequirements: map[string]int{
			"WIS": 13,
		},
	},
	Rogue: {
		Name:                "Rogue",
		Description:         "Stealth, critical hits, high single-target damage",
		HitDie:              8,
		PrimaryStat:         "DEX",
		ArmorProficiencies:  []ArmorType{ArmorLight},
		WeaponProficiencies: []WeaponType{WeaponSimple, WeaponFinesse},
		StartingHP:          8,
		StartingMana:        10,
		ManaPerLevel:        2,
		MulticlassRequirements: map[string]int{
			"DEX": 13,
		},
	},
	Ranger: {
		Name:                "Ranger",
		Description:         "Ranged combat specialist, wilderness tracker",
		HitDie:              10,
		PrimaryStat:         "DEX",
		ArmorProficiencies:  []ArmorType{ArmorLight, ArmorMedium},
		WeaponProficiencies: []WeaponType{WeaponSimple, WeaponMartial, WeaponRanged},
		StartingHP:          10,
		StartingMana:        10,
		ManaPerLevel:        3,
		MulticlassRequirements: map[string]int{
			"DEX": 13,
			"WIS": 13,
		},
	},
	Paladin: {
		Name:                "Paladin",
		Description:         "Holy warrior, tank with healing",
		HitDie:              10,
		PrimaryStat:         "STR",
		ArmorProficiencies:  []ArmorType{ArmorLight, ArmorMedium, ArmorHeavy, ArmorShield},
		WeaponProficiencies: []WeaponType{WeaponSimple, WeaponMartial},
		StartingHP:          10,
		StartingMana:        10,
		ManaPerLevel:        3,
		MulticlassRequirements: map[string]int{
			"STR": 13,
			"CHA": 13,
		},
	},
}

// GetDefinition returns the definition for a class
func GetDefinition(c Class) *Definition {
	return Definitions[c]
}

// HasArmorProficiency checks if a class has proficiency with an armor type
func (d *Definition) HasArmorProficiency(armorType ArmorType) bool {
	for _, prof := range d.ArmorProficiencies {
		if prof == armorType {
			return true
		}
	}
	return false
}

// HasWeaponProficiency checks if a class has proficiency with a weapon type
func (d *Definition) HasWeaponProficiency(weaponType WeaponType) bool {
	for _, prof := range d.WeaponProficiencies {
		if prof == weaponType {
			return true
		}
	}
	return false
}

// CanMulticlassInto checks if a player with given stats can multiclass into this class
func (d *Definition) CanMulticlassInto(stats map[string]int) bool {
	for stat, required := range d.MulticlassRequirements {
		if stats[stat] < required {
			return false
		}
	}
	return true
}

// GetMulticlassRequirementsString returns a human-readable string of requirements
func (d *Definition) GetMulticlassRequirementsString() string {
	if len(d.MulticlassRequirements) == 0 {
		return "None"
	}
	var reqs []string
	for stat, val := range d.MulticlassRequirements {
		reqs = append(reqs, fmt.Sprintf("%s %d+", stat, val))
	}
	return strings.Join(reqs, ", ")
}
