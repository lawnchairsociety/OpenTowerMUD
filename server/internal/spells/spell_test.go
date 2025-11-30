package spells

import "testing"

func TestSpell_RequiresTarget(t *testing.T) {
	tests := []struct {
		name     string
		spell    Spell
		expected bool
	}{
		{
			name: "Self only spell",
			spell: Spell{
				ID:   "heal",
				Name: "heal",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 10},
				},
			},
			expected: false,
		},
		{
			name: "Enemy target spell",
			spell: Spell{
				ID:   "flare",
				Name: "flare",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: true,
		},
		{
			name: "Mixed spell with self effect",
			spell: Spell{
				ID:   "mixed",
				Name: "mixed",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 5},
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: false, // Has self effect, so target is optional
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spell.RequiresTarget()
			if result != tt.expected {
				t.Errorf("RequiresTarget() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpell_IsSelfOnly(t *testing.T) {
	tests := []struct {
		name     string
		spell    Spell
		expected bool
	}{
		{
			name: "Self only spell",
			spell: Spell{
				ID:   "heal",
				Name: "heal",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 10},
				},
			},
			expected: true,
		},
		{
			name: "Enemy target spell",
			spell: Spell{
				ID:   "flare",
				Name: "flare",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: false,
		},
		{
			name: "Multiple self effects",
			spell: Spell{
				ID:   "regen",
				Name: "regen",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 5},
					{Type: EffectHealPercent, Target: TargetSelf, Amount: 10},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spell.IsSelfOnly()
			if result != tt.expected {
				t.Errorf("IsSelfOnly() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpell_HasDamageEffect(t *testing.T) {
	tests := []struct {
		name     string
		spell    Spell
		expected bool
	}{
		{
			name: "Heal spell",
			spell: Spell{
				ID:   "heal",
				Name: "heal",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 10},
				},
			},
			expected: false,
		},
		{
			name: "Damage spell",
			spell: Spell{
				ID:   "flare",
				Name: "flare",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spell.HasDamageEffect()
			if result != tt.expected {
				t.Errorf("HasDamageEffect() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpell_HasHealEffect(t *testing.T) {
	tests := []struct {
		name     string
		spell    Spell
		expected bool
	}{
		{
			name: "Heal spell",
			spell: Spell{
				ID:   "heal",
				Name: "heal",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 10},
				},
			},
			expected: true,
		},
		{
			name: "Percent heal spell",
			spell: Spell{
				ID:   "regen",
				Name: "regen",
				Effects: []SpellEffect{
					{Type: EffectHealPercent, Target: TargetSelf, Amount: 5},
				},
			},
			expected: true,
		},
		{
			name: "Damage spell",
			spell: Spell{
				ID:   "flare",
				Name: "flare",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spell.HasHealEffect()
			if result != tt.expected {
				t.Errorf("HasHealEffect() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpell_GetDamageAmount(t *testing.T) {
	tests := []struct {
		name     string
		spell    Spell
		expected int
	}{
		{
			name: "Single damage effect",
			spell: Spell{
				ID:   "flare",
				Name: "flare",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: 5,
		},
		{
			name: "Multiple damage effects",
			spell: Spell{
				ID:   "fireball",
				Name: "fireball",
				Effects: []SpellEffect{
					{Type: EffectDamage, Target: TargetEnemy, Amount: 10},
					{Type: EffectDamage, Target: TargetEnemy, Amount: 5},
				},
			},
			expected: 15,
		},
		{
			name: "No damage effects",
			spell: Spell{
				ID:   "heal",
				Name: "heal",
				Effects: []SpellEffect{
					{Type: EffectHeal, Target: TargetSelf, Amount: 10},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.spell.GetDamageAmount()
			if result != tt.expected {
				t.Errorf("GetDamageAmount() = %d, want %d", result, tt.expected)
			}
		})
	}
}
