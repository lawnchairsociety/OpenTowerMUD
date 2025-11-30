// Package spells provides spell definitions and effect handling for the magic system.
package spells

// EffectType represents the type of effect a spell has.
type EffectType string

const (
	EffectHeal        EffectType = "heal"
	EffectDamage      EffectType = "damage"
	EffectHealPercent EffectType = "heal_percent"
	EffectStun        EffectType = "stun"
)

// TargetType represents what a spell can target.
type TargetType string

const (
	TargetSelf       TargetType = "self"
	TargetEnemy      TargetType = "enemy"
	TargetAlly       TargetType = "ally"
	TargetRoomEnemy  TargetType = "room_enemy" // All attackable NPCs in the room
)

// SpellEffect represents a single effect that a spell applies.
type SpellEffect struct {
	Type   EffectType
	Target TargetType
	Amount int    // Flat amount or percentage depending on EffectType (legacy, used as fallback)
	Dice   string // Dice notation for effect (e.g., "1d6", "2d4+2") - used with ability modifier
}

// Spell represents a castable spell with its properties.
type Spell struct {
	ID          string
	Name        string
	Description string
	ManaCost    int
	Cooldown    int // Seconds (0 = no cooldown)
	Effects     []SpellEffect
	Level       int // Minimum level to learn
}

// RequiresTarget returns true if the spell can ONLY target enemies/allies (no self effects).
func (s *Spell) RequiresTarget() bool {
	hasSelfEffect := false
	hasTargetEffect := false
	for _, effect := range s.Effects {
		if effect.Target == TargetSelf {
			hasSelfEffect = true
		}
		if effect.Target == TargetEnemy || effect.Target == TargetAlly {
			hasTargetEffect = true
		}
	}
	// Only requires target if it has no self effects
	return hasTargetEffect && !hasSelfEffect
}

// CanTargetAlly returns true if the spell has ally-targeted effects.
func (s *Spell) CanTargetAlly() bool {
	for _, effect := range s.Effects {
		if effect.Target == TargetAlly {
			return true
		}
	}
	return false
}

// CanTargetEnemy returns true if the spell has enemy-targeted effects.
func (s *Spell) CanTargetEnemy() bool {
	for _, effect := range s.Effects {
		if effect.Target == TargetEnemy {
			return true
		}
	}
	return false
}

// CanTargetSelf returns true if the spell has self-targeted effects.
func (s *Spell) CanTargetSelf() bool {
	for _, effect := range s.Effects {
		if effect.Target == TargetSelf {
			return true
		}
	}
	return false
}

// IsSelfOnly returns true if the spell only affects the caster.
func (s *Spell) IsSelfOnly() bool {
	for _, effect := range s.Effects {
		if effect.Target != TargetSelf {
			return false
		}
	}
	return true
}

// HasDamageEffect returns true if the spell deals damage.
func (s *Spell) HasDamageEffect() bool {
	for _, effect := range s.Effects {
		if effect.Type == EffectDamage {
			return true
		}
	}
	return false
}

// HasHealEffect returns true if the spell heals.
func (s *Spell) HasHealEffect() bool {
	for _, effect := range s.Effects {
		if effect.Type == EffectHeal || effect.Type == EffectHealPercent {
			return true
		}
	}
	return false
}

// GetDamageAmount returns the total damage the spell deals.
func (s *Spell) GetDamageAmount() int {
	total := 0
	for _, effect := range s.Effects {
		if effect.Type == EffectDamage {
			total += effect.Amount
		}
	}
	return total
}

// CanTargetRoomEnemies returns true if the spell affects all enemies in the room.
func (s *Spell) CanTargetRoomEnemies() bool {
	for _, effect := range s.Effects {
		if effect.Target == TargetRoomEnemy {
			return true
		}
	}
	return false
}

// HasStunEffect returns true if the spell has a stun effect.
func (s *Spell) HasStunEffect() bool {
	for _, effect := range s.Effects {
		if effect.Type == EffectStun {
			return true
		}
	}
	return false
}
