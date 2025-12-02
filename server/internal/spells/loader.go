package spells

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SpellEffectDefinition represents a spell effect in the YAML file.
type SpellEffectDefinition struct {
	Type     string `yaml:"type"`
	Target   string `yaml:"target"`
	Amount   int    `yaml:"amount"`              // Flat amount (legacy, used as fallback)
	Dice     string `yaml:"dice,omitempty"`      // Dice notation e.g. "1d6", "2d4+2"
	Duration int    `yaml:"duration,omitempty"`  // Duration in seconds for timed effects
	BuffType string `yaml:"buff_type,omitempty"` // Type of buff/debuff (ac, hit, damage, taken)
}

// SpellDefinition represents a spell definition from the YAML file.
type SpellDefinition struct {
	Name           string                  `yaml:"name"`
	Description    string                  `yaml:"description"`
	ManaCost       int                     `yaml:"mana_cost"`
	Cooldown       int                     `yaml:"cooldown"`
	Level          int                     `yaml:"level"`
	Effects        []SpellEffectDefinition `yaml:"effects"`
	AllowedClasses []string                `yaml:"allowed_classes,omitempty"` // Classes that can learn this spell
}

// SpellsConfig represents the structure of the spells.yaml file.
type SpellsConfig struct {
	StarterSpells []string                   `yaml:"starter_spells"`
	Spells        map[string]SpellDefinition `yaml:"spells"`
}

// SpellRegistry holds all loaded spells and provides lookup.
type SpellRegistry struct {
	spells        map[string]*Spell
	starterSpells []string
}

// NewSpellRegistry creates a new empty spell registry.
func NewSpellRegistry() *SpellRegistry {
	return &SpellRegistry{
		spells:        make(map[string]*Spell),
		starterSpells: []string{},
	}
}

// LoadSpellsFromYAML loads spell definitions from a YAML file.
func LoadSpellsFromYAML(filename string) (*SpellsConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read spells file: %w", err)
	}

	var config SpellsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse spells YAML: %w", err)
	}

	return &config, nil
}

// StringToEffectType converts a string to an EffectType.
func StringToEffectType(s string) EffectType {
	switch s {
	case "heal":
		return EffectHeal
	case "damage":
		return EffectDamage
	case "heal_percent":
		return EffectHealPercent
	case "stun":
		return EffectStun
	case "buff":
		return EffectBuff
	case "debuff":
		return EffectDebuff
	case "poison":
		return EffectPoison
	case "stealth":
		return EffectStealth
	case "root":
		return EffectRoot
	case "execute":
		return EffectExecute
	case "smite":
		return EffectSmite
	case "resurrect":
		return EffectResurrect
	case "cleanse":
		return EffectCleanse
	case "multi_attack":
		return EffectMultiAttack
	default:
		return EffectHeal
	}
}

// StringToTargetType converts a string to a TargetType.
func StringToTargetType(s string) TargetType {
	switch s {
	case "self":
		return TargetSelf
	case "enemy":
		return TargetEnemy
	case "ally":
		return TargetAlly
	case "room_enemy":
		return TargetRoomEnemy
	case "room_ally":
		return TargetRoomAlly
	case "dead_ally":
		return TargetDeadAlly
	default:
		return TargetSelf
	}
}

// StringToBuffType converts a string to a BuffType.
func StringToBuffType(s string) BuffType {
	switch s {
	case "ac":
		return BuffAC
	case "hit":
		return BuffHit
	case "damage":
		return BuffDamage
	case "taken":
		return BuffTaken
	case "mana":
		return BuffMana
	case "regen":
		return BuffRegen
	default:
		return BuffAC
	}
}

// CreateSpellFromDefinition creates a Spell from a SpellDefinition.
func CreateSpellFromDefinition(id string, def SpellDefinition) *Spell {
	effects := make([]SpellEffect, len(def.Effects))
	for i, e := range def.Effects {
		effects[i] = SpellEffect{
			Type:     StringToEffectType(e.Type),
			Target:   StringToTargetType(e.Target),
			Amount:   e.Amount,
			Dice:     e.Dice,
			Duration: e.Duration,
			BuffType: StringToBuffType(e.BuffType),
		}
	}

	return &Spell{
		ID:             id,
		Name:           def.Name,
		Description:    def.Description,
		ManaCost:       def.ManaCost,
		Cooldown:       def.Cooldown,
		Level:          def.Level,
		Effects:        effects,
		AllowedClasses: def.AllowedClasses,
	}
}

// LoadFromYAML loads spells from a YAML file into the registry.
func (r *SpellRegistry) LoadFromYAML(filename string) error {
	config, err := LoadSpellsFromYAML(filename)
	if err != nil {
		return err
	}

	for id, def := range config.Spells {
		r.spells[id] = CreateSpellFromDefinition(id, def)
	}

	// Load starter spells from config
	r.starterSpells = config.StarterSpells

	return nil
}

// GetSpell returns a spell by its ID.
func (r *SpellRegistry) GetSpell(id string) (*Spell, bool) {
	spell, exists := r.spells[id]
	return spell, exists
}

// GetAllSpells returns all spells in the registry.
func (r *SpellRegistry) GetAllSpells() map[string]*Spell {
	return r.spells
}

// GetSpellsByLevel returns all spells at or below a given level.
func (r *SpellRegistry) GetSpellsByLevel(maxLevel int) []*Spell {
	var result []*Spell
	for _, spell := range r.spells {
		if spell.Level <= maxLevel {
			result = append(result, spell)
		}
	}
	return result
}

// GetSpellsForClass returns all spells available to a specific class at or below a given level.
func (r *SpellRegistry) GetSpellsForClass(className string, maxLevel int) []*Spell {
	var result []*Spell
	for _, spell := range r.spells {
		if spell.Level <= maxLevel && spell.IsAllowedForClass(className) {
			result = append(result, spell)
		}
	}
	return result
}

// GetSpellsForClasses returns all spells available to any of the given classes and levels.
// classLevels is a map of class name to class level.
func (r *SpellRegistry) GetSpellsForClasses(classLevels map[string]int) []*Spell {
	var result []*Spell
	seen := make(map[string]bool)
	for _, spell := range r.spells {
		if seen[spell.ID] {
			continue
		}
		// Check if any class can use this spell at their level
		for className, level := range classLevels {
			if spell.Level <= level && spell.IsAllowedForClass(className) {
				result = append(result, spell)
				seen[spell.ID] = true
				break
			}
		}
	}
	return result
}

// GetStarterSpells returns the IDs of starter spells from config.
// Deprecated: With the class system, spells are now automatically available based on class levels.
// This method is kept for backwards compatibility but returns an empty slice by default.
func (r *SpellRegistry) GetStarterSpells() []string {
	return r.starterSpells
}

// DefaultStarterSpells returns the IDs of spells new characters used to start with.
// Deprecated: With the class system, spells are now automatically available based on class levels.
// Use SpellRegistry.GetSpellsForClasses() instead.
func DefaultStarterSpells() []string {
	return []string{} // Empty - spells are now class-based
}
