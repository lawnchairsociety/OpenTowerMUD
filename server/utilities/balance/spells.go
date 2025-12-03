// Package balance provides Monte Carlo simulation tools for game balance testing.
package balance

import "math/rand"

// SpellEffect represents a spell's crowd control effect
type SpellEffect struct {
	Name     string
	Type     string // "stun" or "root"
	Duration int    // Duration in combat rounds (1 round = 3 seconds)
	Cooldown int    // Cooldown in combat rounds
	ManaCost int    // Mana cost
}

// CombatConfig holds configuration for advanced combat simulation
type CombatConfig struct {
	PlayerSpells   []SpellEffect // Spells the player can use
	NPCFleeThresh  float64       // HP percentage at which NPC flees (0.0-1.0, 0 = never)
	NPCFleeChance  float64       // Chance per round to flee when below threshold (0.0-1.0)
	TicksPerRound  int           // Combat ticks per round (default 1)
	MaxMana        int           // Player's max mana
	ManaPerRound   int           // Mana regenerated per round
}

// DefaultCombatConfig returns a default combat configuration
func DefaultCombatConfig() CombatConfig {
	return CombatConfig{
		NPCFleeThresh: 0.0,
		NPCFleeChance: 0.35, // 35% chance to flee per round when below threshold
		TicksPerRound: 1,
		MaxMana:       100,
		ManaPerRound:  2,
	}
}

// CCCombatResult holds the outcome of a combat simulation with CC
type CCCombatResult struct {
	PlayerWon        bool
	NPCFled          bool // True if NPC escaped
	Rounds           int
	PlayerHPRemain   int
	NPCHPRemain      int
	PlayerDamageIn   int
	PlayerDamageOut  int
	StunRoundsUsed   int // Rounds where NPC was stunned
	RootRoundsUsed   int // Rounds where NPC was rooted (prevented flee)
	SpellsCast       int // Total spells cast
	FleeAttempts     int // Times NPC tried to flee
	FleePrevented    int // Times flee was prevented by root
}

// CCSimulationResult holds aggregated results from CC combat simulations
type CCSimulationResult struct {
	Simulations       int
	PlayerWins        int
	NPCWins           int
	NPCEscapes        int
	WinRate           float64
	EscapeRate        float64
	AvgRounds         float64
	AvgPlayerHPLeft   float64
	AvgDamageDealt    float64
	AvgDamageTaken    float64
	AvgStunRounds     float64
	AvgRootRounds     float64
	AvgSpellsCast     float64
	AvgFleeAttempts   float64
	AvgFleePrevented  float64
	DamagePrevented   float64 // Estimated damage prevented by stuns
	Config            CombatConfig
}

// spellState tracks cooldowns and active effects during combat
type spellState struct {
	cooldowns    map[string]int // Spell name -> rounds until ready
	stunRemain   int            // Rounds of stun remaining on NPC
	rootRemain   int            // Rounds of root remaining on NPC
	currentMana  int
}

// SimulateCombatWithCC runs a single combat with crowd control effects
func SimulateCombatWithCC(player, npc CombatantStats, config CombatConfig) CCCombatResult {
	result := CCCombatResult{}

	playerHP := player.Health
	npcHP := npc.Health
	npcMaxHP := npc.Health
	round := 0
	maxRounds := 1000

	state := spellState{
		cooldowns:   make(map[string]int),
		currentMana: config.MaxMana,
	}

	for playerHP > 0 && npcHP > 0 && round < maxRounds {
		round++

		// Regenerate mana
		state.currentMana += config.ManaPerRound
		if state.currentMana > config.MaxMana {
			state.currentMana = config.MaxMana
		}

		// Reduce cooldowns
		for spell := range state.cooldowns {
			if state.cooldowns[spell] > 0 {
				state.cooldowns[spell]--
			}
		}

		// Check if NPC wants to flee (before player acts)
		npcWantsToFlee := false
		if config.NPCFleeThresh > 0 {
			hpPercent := float64(npcHP) / float64(npcMaxHP)
			if hpPercent <= config.NPCFleeThresh {
				npcWantsToFlee = true
			}
		}

		// Player decides whether to use a spell
		// AI: Use stun if available and NPC not stunned
		// AI: Use root if NPC wants to flee and not rooted
		spellUsed := false
		for _, spell := range config.PlayerSpells {
			if state.cooldowns[spell.Name] > 0 {
				continue // On cooldown
			}
			if state.currentMana < spell.ManaCost {
				continue // Not enough mana
			}

			shouldUse := false
			switch spell.Type {
			case "stun":
				// Use stun if NPC isn't already stunned
				if state.stunRemain <= 0 {
					shouldUse = true
				}
			case "root":
				// Use root if NPC wants to flee and isn't rooted
				if npcWantsToFlee && state.rootRemain <= 0 {
					shouldUse = true
				}
			}

			if shouldUse {
				state.currentMana -= spell.ManaCost
				state.cooldowns[spell.Name] = spell.Cooldown
				result.SpellsCast++
				spellUsed = true

				switch spell.Type {
				case "stun":
					state.stunRemain = spell.Duration
				case "root":
					state.rootRemain = spell.Duration
				}
				break // Only cast one spell per round
			}
		}

		// Player attacks
		if !spellUsed { // Can attack if didn't cast a spell (simplified)
			attackRoll := rollD20() + player.StrMod
			if attackRoll >= npc.ArmorClass {
				var damage int
				if player.DamageDice != "" {
					damage = rollDice(player.DamageDice, player.StrMod)
				} else {
					damage = player.Damage + player.StrMod
				}

				actualDamage := damage - npc.Armor
				if actualDamage < 1 {
					actualDamage = 1
				}

				npcHP -= actualDamage
				result.PlayerDamageOut += actualDamage
			}
		}

		// Check if NPC died
		if npcHP <= 0 {
			break
		}

		// NPC turn: check flee, then attack
		if npcWantsToFlee && rand.Float64() < config.NPCFleeChance {
			result.FleeAttempts++
			if state.rootRemain > 0 {
				// Rooted, can't flee
				result.FleePrevented++
			} else {
				// NPC escapes!
				result.NPCFled = true
				break
			}
		}

		// Decrement root (after flee check)
		if state.rootRemain > 0 {
			state.rootRemain--
			result.RootRoundsUsed++
		}

		// NPC attacks (if not stunned)
		if state.stunRemain > 0 {
			state.stunRemain--
			result.StunRoundsUsed++
			// NPC is stunned, skip attack
		} else {
			npcDamage := npc.Damage
			actualDamage := npcDamage - player.Armor
			if actualDamage < 1 {
				actualDamage = 1
			}
			playerHP -= actualDamage
			result.PlayerDamageIn += actualDamage
		}
	}

	result.Rounds = round
	result.PlayerHPRemain = playerHP
	result.NPCHPRemain = npcHP
	result.PlayerWon = playerHP > 0 && npcHP <= 0

	return result
}

// RunCCSimulation runs multiple CC combat simulations and returns aggregated results
func RunCCSimulation(player, npc CombatantStats, config CombatConfig, iterations int) CCSimulationResult {
	result := CCSimulationResult{
		Simulations: iterations,
		Config:      config,
	}

	totalRounds := 0
	totalPlayerHPLeft := 0
	totalDamageDealt := 0
	totalDamageTaken := 0
	totalStunRounds := 0
	totalRootRounds := 0
	totalSpellsCast := 0
	totalFleeAttempts := 0
	totalFleePrevented := 0

	for i := 0; i < iterations; i++ {
		combat := SimulateCombatWithCC(player, npc, config)

		if combat.PlayerWon {
			result.PlayerWins++
			totalPlayerHPLeft += combat.PlayerHPRemain
		} else if combat.NPCFled {
			result.NPCEscapes++
		} else {
			result.NPCWins++
		}

		totalRounds += combat.Rounds
		totalDamageDealt += combat.PlayerDamageOut
		totalDamageTaken += combat.PlayerDamageIn
		totalStunRounds += combat.StunRoundsUsed
		totalRootRounds += combat.RootRoundsUsed
		totalSpellsCast += combat.SpellsCast
		totalFleeAttempts += combat.FleeAttempts
		totalFleePrevented += combat.FleePrevented
	}

	result.WinRate = float64(result.PlayerWins) / float64(iterations) * 100
	result.EscapeRate = float64(result.NPCEscapes) / float64(iterations) * 100
	result.AvgRounds = float64(totalRounds) / float64(iterations)
	result.AvgDamageDealt = float64(totalDamageDealt) / float64(iterations)
	result.AvgDamageTaken = float64(totalDamageTaken) / float64(iterations)
	result.AvgStunRounds = float64(totalStunRounds) / float64(iterations)
	result.AvgRootRounds = float64(totalRootRounds) / float64(iterations)
	result.AvgSpellsCast = float64(totalSpellsCast) / float64(iterations)
	result.AvgFleeAttempts = float64(totalFleeAttempts) / float64(iterations)
	result.AvgFleePrevented = float64(totalFleePrevented) / float64(iterations)

	// Estimate damage prevented by stuns (avg stun rounds * npc damage)
	result.DamagePrevented = result.AvgStunRounds * float64(npc.Damage)

	if result.PlayerWins > 0 {
		result.AvgPlayerHPLeft = float64(totalPlayerHPLeft) / float64(result.PlayerWins)
	}

	return result
}

// SpellBalanceResult holds comparison results for spell effectiveness
type SpellBalanceResult struct {
	SpellName        string
	BaseWinRate      float64 // Win rate without the spell
	SpellWinRate     float64 // Win rate with the spell
	WinRateImprove   float64 // Percentage point improvement
	BaseEscapeRate   float64 // Escape rate without spell
	SpellEscapeRate  float64 // Escape rate with spell
	EscapeReduction  float64 // How much escapes were reduced
	DamagePrevented  float64 // Avg damage prevented (for stuns)
	AvgSpellsCast    float64 // How often the spell was used
}

// CompareSpellEffectiveness compares combat with and without a specific spell
func CompareSpellEffectiveness(player, npc CombatantStats, spell SpellEffect, fleeThreshold float64, iterations int) SpellBalanceResult {
	result := SpellBalanceResult{
		SpellName: spell.Name,
	}

	// Run without the spell
	baseConfig := DefaultCombatConfig()
	baseConfig.NPCFleeThresh = fleeThreshold
	baseResult := RunCCSimulation(player, npc, baseConfig, iterations)

	result.BaseWinRate = baseResult.WinRate
	result.BaseEscapeRate = baseResult.EscapeRate

	// Run with the spell
	spellConfig := DefaultCombatConfig()
	spellConfig.NPCFleeThresh = fleeThreshold
	spellConfig.PlayerSpells = []SpellEffect{spell}
	spellResult := RunCCSimulation(player, npc, spellConfig, iterations)

	result.SpellWinRate = spellResult.WinRate
	result.WinRateImprove = spellResult.WinRate - baseResult.WinRate
	result.SpellEscapeRate = spellResult.EscapeRate
	result.EscapeReduction = baseResult.EscapeRate - spellResult.EscapeRate
	result.DamagePrevented = spellResult.DamagePrevented
	result.AvgSpellsCast = spellResult.AvgSpellsCast

	return result
}

// PredefinedSpells returns the game's actual spell definitions for testing
func PredefinedSpells() map[string]SpellEffect {
	// Convert seconds to rounds (3 seconds per round)
	secToRounds := func(sec int) int {
		return (sec + 2) / 3 // Round up
	}

	return map[string]SpellEffect{
		// Stuns
		"cheap_shot": {
			Name:     "Cheap Shot (Rogue)",
			Type:     "stun",
			Duration: secToRounds(3),
			Cooldown: secToRounds(30),
			ManaCost: 10,
		},
		"shield_bash": {
			Name:     "Shield Bash (Warrior)",
			Type:     "stun",
			Duration: secToRounds(4),
			Cooldown: secToRounds(30),
			ManaCost: 10,
		},
		"rebuke": {
			Name:     "Rebuke (Paladin)",
			Type:     "stun",
			Duration: secToRounds(3),
			Cooldown: secToRounds(30),
			ManaCost: 12,
		},
		// Roots
		"ensnaring_strike": {
			Name:     "Ensnaring Strike (Ranger)",
			Type:     "root",
			Duration: secToRounds(30),
			Cooldown: secToRounds(30),
			ManaCost: 12,
		},
		"hold_person": {
			Name:     "Hold Person (Mage)",
			Type:     "root",
			Duration: secToRounds(30),
			Cooldown: secToRounds(45),
			ManaCost: 15,
		},
		"command": {
			Name:     "Command (Cleric)",
			Type:     "root",
			Duration: secToRounds(20),
			Cooldown: secToRounds(30),
			ManaCost: 12,
		},
		"hamstring_rogue": {
			Name:     "Hamstring (Rogue)",
			Type:     "root",
			Duration: secToRounds(20),
			Cooldown: secToRounds(45),
			ManaCost: 12,
		},
		"commanding_presence": {
			Name:     "Commanding Presence (Paladin)",
			Type:     "root",
			Duration: secToRounds(20),
			Cooldown: secToRounds(45),
			ManaCost: 15,
		},
		"hamstring_warrior": {
			Name:     "Hamstring (Warrior)",
			Type:     "root",
			Duration: secToRounds(25),
			Cooldown: secToRounds(45),
			ManaCost: 12,
		},
	}
}

// RunSpellBalanceSweep tests all predefined spells and returns comparison results
func RunSpellBalanceSweep(player, npc CombatantStats, fleeThreshold float64, iterations int) []SpellBalanceResult {
	spells := PredefinedSpells()
	results := make([]SpellBalanceResult, 0, len(spells))

	for _, spell := range spells {
		result := CompareSpellEffectiveness(player, npc, spell, fleeThreshold, iterations)
		results = append(results, result)
	}

	return results
}
