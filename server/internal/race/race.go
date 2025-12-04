// Package race defines the player race system for OpenTowerMUD.
package race

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Race represents a player race
type Race string

const (
	Human    Race = "human"
	Dwarf    Race = "dwarf"
	Elf      Race = "elf"
	Halfling Race = "halfling"
	Gnome    Race = "gnome"
	HalfElf  Race = "half-elf"
	HalfOrc  Race = "half-orc"
)

// IsValid returns true if the race is a valid race
func (r Race) IsValid() bool {
	_, exists := globalConfig.Races[string(r)]
	return exists
}

// String returns the display name of the race
func (r Race) String() string {
	if def, exists := globalConfig.Races[string(r)]; exists {
		return def.Name
	}
	return "Unknown"
}

// ParseRace parses a string into a Race, case-insensitive
func ParseRace(s string) (Race, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	// Handle alternate spellings
	switch normalized {
	case "halfelf":
		normalized = "half-elf"
	case "halforc":
		normalized = "half-orc"
	}

	if _, exists := globalConfig.Races[normalized]; exists {
		return Race(normalized), nil
	}
	return "", fmt.Errorf("unknown race: %s", s)
}

// RaceDefinition represents a race definition from the YAML file
type RaceDefinition struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Size        string         `yaml:"size"`
	StatBonuses map[string]int `yaml:"stat_bonuses"`
	Abilities   []string       `yaml:"abilities"`
}

// RacesConfig represents the structure of the races.yaml file
type RacesConfig struct {
	Races map[string]*RaceDefinition `yaml:"races"`
}

// globalConfig holds the loaded race configuration
var globalConfig = &RacesConfig{
	Races: make(map[string]*RaceDefinition),
}

// LoadRacesFromYAML loads race definitions from a YAML file
func LoadRacesFromYAML(filename string) (*RacesConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read races file: %w", err)
	}

	var config RacesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse races YAML: %w", err)
	}

	// Initialize empty stat_bonuses maps if nil
	for _, def := range config.Races {
		if def.StatBonuses == nil {
			def.StatBonuses = make(map[string]int)
		}
	}

	// Store globally for access by Race methods
	globalConfig = &config

	return &config, nil
}

// SetGlobalConfig sets the global race configuration (used after loading)
func SetGlobalConfig(config *RacesConfig) {
	if config != nil {
		globalConfig = config
	}
}

// AllRaces returns all valid races from the loaded configuration
func AllRaces() []Race {
	// Return in a consistent order
	order := []string{"human", "dwarf", "elf", "halfling", "gnome", "half-elf", "half-orc"}
	var races []Race
	for _, id := range order {
		if _, exists := globalConfig.Races[id]; exists {
			races = append(races, Race(id))
		}
	}
	return races
}

// GetDefinition returns the definition for a race
func GetDefinition(r Race) *Definition {
	def, exists := globalConfig.Races[string(r)]
	if !exists {
		return nil
	}
	// Convert RaceDefinition to Definition for backwards compatibility
	return &Definition{
		Name:        r,
		Description: def.Description,
		StatBonuses: def.StatBonuses,
		Abilities:   def.Abilities,
		Size:        def.Size,
	}
}

// Definition contains the static definition for a race (backwards compatibility)
type Definition struct {
	Name        Race
	Description string
	StatBonuses map[string]int // e.g., {"CON": 2, "CHA": -2}
	Abilities   []string       // Flavor text abilities
	Size        string         // "Medium" or "Small"
}

// ApplyStatBonuses applies racial stat bonuses to the given ability scores
// Returns the modified stats (str, dex, con, int, wis, cha)
func (d *Definition) ApplyStatBonuses(str, dex, con, int_, wis, cha int) (int, int, int, int, int, int) {
	if bonus, ok := d.StatBonuses["STR"]; ok {
		str += bonus
	}
	if bonus, ok := d.StatBonuses["DEX"]; ok {
		dex += bonus
	}
	if bonus, ok := d.StatBonuses["CON"]; ok {
		con += bonus
	}
	if bonus, ok := d.StatBonuses["INT"]; ok {
		int_ += bonus
	}
	if bonus, ok := d.StatBonuses["WIS"]; ok {
		wis += bonus
	}
	if bonus, ok := d.StatBonuses["CHA"]; ok {
		cha += bonus
	}
	return str, dex, con, int_, wis, cha
}

// GetStatBonusesString returns a human-readable string of stat bonuses
func (d *Definition) GetStatBonusesString() string {
	if len(d.StatBonuses) == 0 {
		if d.Name == Human {
			return "+1 to one ability (your choice)"
		}
		return "None"
	}

	var bonuses []string
	// Order matters for consistent display
	statOrder := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	for _, stat := range statOrder {
		if bonus, ok := d.StatBonuses[stat]; ok {
			if bonus > 0 {
				bonuses = append(bonuses, fmt.Sprintf("+%d %s", bonus, stat))
			} else {
				bonuses = append(bonuses, fmt.Sprintf("%d %s", bonus, stat))
			}
		}
	}
	return strings.Join(bonuses, ", ")
}

// GetAbilitiesString returns a human-readable string of racial abilities
func (d *Definition) GetAbilitiesString() string {
	if len(d.Abilities) == 0 {
		return "None"
	}
	return strings.Join(d.Abilities, ", ")
}

// HasStatBonus returns true if this race has any stat bonuses to apply
// Note: Human returns false here since their bonus is chosen at creation
func (d *Definition) HasStatBonus() bool {
	return len(d.StatBonuses) > 0
}

// init initializes with default races in case YAML isn't loaded
// This provides fallback data for testing and development
func init() {
	globalConfig = &RacesConfig{
		Races: map[string]*RaceDefinition{
			"human": {
				Name:        "Human",
				Description: "Versatile and ambitious, humans are the most adaptable of all races.",
				Size:        "Medium",
				StatBonuses: map[string]int{},
				Abilities:   []string{"Versatile: +1 to one ability score of your choice"},
			},
			"dwarf": {
				Name:        "Dwarf",
				Description: "Stout and sturdy, dwarves are known for their resilience.",
				Size:        "Medium",
				StatBonuses: map[string]int{"CON": 2, "CHA": -2},
				Abilities:   []string{"Darkvision", "Poison Resistance"},
			},
			"elf": {
				Name:        "Elf",
				Description: "Graceful and long-lived, elves possess keen senses.",
				Size:        "Medium",
				StatBonuses: map[string]int{"DEX": 2, "CON": -2},
				Abilities:   []string{"Low-light Vision", "Sleep Immunity"},
			},
			"halfling": {
				Name:        "Halfling",
				Description: "Small but brave, halflings are nimble and lucky.",
				Size:        "Small",
				StatBonuses: map[string]int{"DEX": 2, "STR": -2},
				Abilities:   []string{"Fearless", "Lucky"},
			},
			"gnome": {
				Name:        "Gnome",
				Description: "Curious and inventive, gnomes are small but hardy.",
				Size:        "Small",
				StatBonuses: map[string]int{"CON": 2, "STR": -2},
				Abilities:   []string{"Low-light Vision", "Illusion Resistance"},
			},
			"half-elf": {
				Name:        "Half-Elf",
				Description: "Combining the best of both worlds, half-elves are diplomatic.",
				Size:        "Medium",
				StatBonuses: map[string]int{},
				Abilities:   []string{"Sleep Immunity", "Adaptable"},
			},
			"half-orc": {
				Name:        "Half-Orc",
				Description: "Strong and fierce, half-orcs combine human ambition with orcish might.",
				Size:        "Medium",
				StatBonuses: map[string]int{"STR": 2, "INT": -2, "CHA": -2},
				Abilities:   []string{"Darkvision", "Ferocity"},
			},
		},
	}
}
