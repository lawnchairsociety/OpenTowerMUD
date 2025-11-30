package spells

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SpellEffectDefinition represents a spell effect in the YAML file.
type SpellEffectDefinition struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
	Amount int    `yaml:"amount"`            // Flat amount (legacy, used as fallback)
	Dice   string `yaml:"dice,omitempty"`    // Dice notation e.g. "1d6", "2d4+2"
}

// SpellDefinition represents a spell definition from the YAML file.
type SpellDefinition struct {
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description"`
	ManaCost    int                     `yaml:"mana_cost"`
	Cooldown    int                     `yaml:"cooldown"`
	Level       int                     `yaml:"level"`
	Effects     []SpellEffectDefinition `yaml:"effects"`
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
	default:
		return TargetSelf
	}
}

// CreateSpellFromDefinition creates a Spell from a SpellDefinition.
func CreateSpellFromDefinition(id string, def SpellDefinition) *Spell {
	effects := make([]SpellEffect, len(def.Effects))
	for i, e := range def.Effects {
		effects[i] = SpellEffect{
			Type:   StringToEffectType(e.Type),
			Target: StringToTargetType(e.Target),
			Amount: e.Amount,
			Dice:   e.Dice,
		}
	}

	return &Spell{
		ID:          id,
		Name:        def.Name,
		Description: def.Description,
		ManaCost:    def.ManaCost,
		Cooldown:    def.Cooldown,
		Level:       def.Level,
		Effects:     effects,
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

// GetStarterSpells returns the IDs of spells new characters should start with.
func (r *SpellRegistry) GetStarterSpells() []string {
	if len(r.starterSpells) == 0 {
		// Fallback to defaults if not loaded from YAML
		return []string{"heal", "flare", "dazzle"}
	}
	return r.starterSpells
}

// DefaultStarterSpells returns the IDs of spells new characters should start with.
// Deprecated: Use SpellRegistry.GetStarterSpells() instead.
func DefaultStarterSpells() []string {
	return []string{"heal", "flare", "dazzle"}
}
