// Package balance provides Monte Carlo simulation tools for game balance testing.
package balance

import (
	"math/rand"
	"regexp"
	"strconv"
)

// CombatantStats represents a combatant's statistics for simulation
type CombatantStats struct {
	Name        string
	Level       int
	Health      int
	MaxHealth   int
	Armor       int    // For NPCs: flat armor reduction; For players: sum of equipped armor
	Damage      int    // For NPCs: flat damage dealt per hit
	DamageDice  string // For players: dice notation like "1d6+2"
	StrMod      int    // Player strength modifier for attack rolls and damage
	ArmorClass  int    // Target AC for attack rolls (10 + armor for NPCs)
}

// CasterStats extends CombatantStats with mana and spell information
type CasterStats struct {
	CombatantStats
	Mana         int    // Current/max mana
	SpellCost    int    // Mana cost per spell cast
	SpellDice    string // Spell damage dice (e.g., "1d4+1" for magic missile)
	CastingMod   int    // INT or WIS modifier for spell damage
}

// NewPlayerStats creates stats for a player combatant
func NewPlayerStats(name string, level, health, armor, strMod int, damageDice string) CombatantStats {
	return CombatantStats{
		Name:       name,
		Level:      level,
		Health:     health,
		MaxHealth:  health,
		Armor:      armor,
		DamageDice: damageDice,
		StrMod:     strMod,
		ArmorClass: 10 + armor, // Players have AC too (for future PvP or effects)
	}
}

// NewNPCStats creates stats for an NPC combatant
func NewNPCStats(name string, level, health, armor, damage int) CombatantStats {
	return CombatantStats{
		Name:       name,
		Level:      level,
		Health:     health,
		MaxHealth:  health,
		Armor:      armor,
		Damage:     damage,
		ArmorClass: 10 + armor,
	}
}

// NewCasterStats creates stats for a spellcasting player
func NewCasterStats(name string, level, health, armor, castingMod int, weaponDice string, mana, spellCost int, spellDice string) CasterStats {
	return CasterStats{
		CombatantStats: CombatantStats{
			Name:       name,
			Level:      level,
			Health:     health,
			MaxHealth:  health,
			Armor:      armor,
			DamageDice: weaponDice,
			StrMod:     0, // Casters typically don't use STR
			ArmorClass: 10 + armor,
		},
		Mana:       mana,
		SpellCost:  spellCost,
		SpellDice:  spellDice,
		CastingMod: castingMod,
	}
}

// CombatResult holds the outcome of a single combat simulation
type CombatResult struct {
	PlayerWon       bool
	Rounds          int
	PlayerHPRemain  int
	NPCHPRemain     int
	PlayerDamageIn  int // Total damage taken by player
	PlayerDamageOut int // Total damage dealt by player
}

// SimulationResult holds aggregated results from many combat simulations
type SimulationResult struct {
	Simulations      int
	PlayerWins       int
	NPCWins          int
	WinRate          float64
	AvgRounds        float64
	AvgPlayerHPLeft  float64 // Average HP remaining when player wins
	AvgDamageDealt   float64
	AvgDamageTaken   float64
	MinRounds        int
	MaxRounds        int
	PlayerStats      CombatantStats
	NPCStats         CombatantStats
}

// diceRegex matches dice notation like "1d6", "2d4+1", "1d8-2"
var diceRegex = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)

// rollDice parses and rolls dice notation, returning the result
func rollDice(notation string, extraBonus int) int {
	if notation == "" {
		return 1 // Minimum damage
	}

	matches := diceRegex.FindStringSubmatch(notation)
	if matches == nil {
		return 1
	}

	count, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])

	bonus := extraBonus
	if matches[3] != "" {
		notationBonus, _ := strconv.Atoi(matches[3])
		bonus += notationBonus
	}

	total := 0
	for i := 0; i < count; i++ {
		total += rand.Intn(sides) + 1
	}
	total += bonus

	if total < 1 {
		return 1 // Minimum damage
	}
	return total
}

// rollD20 rolls a d20
func rollD20() int {
	return rand.Intn(20) + 1
}

// SimulateCombat runs a single combat between player and NPC
// Returns the result including who won, rounds taken, and HP remaining
func SimulateCombat(player, npc CombatantStats) CombatResult {
	result := CombatResult{}

	playerHP := player.Health
	npcHP := npc.Health
	round := 0
	maxRounds := 1000 // Safety limit

	for playerHP > 0 && npcHP > 0 && round < maxRounds {
		round++

		// Player attacks first
		attackRoll := rollD20() + player.StrMod
		if attackRoll >= npc.ArmorClass {
			// Hit! Roll damage
			var damage int
			if player.DamageDice != "" {
				damage = rollDice(player.DamageDice, player.StrMod)
			} else {
				damage = player.Damage + player.StrMod
			}

			// Apply NPC armor reduction
			actualDamage := damage - npc.Armor
			if actualDamage < 1 {
				actualDamage = 1
			}

			npcHP -= actualDamage
			result.PlayerDamageOut += actualDamage
		}

		// Check if NPC died
		if npcHP <= 0 {
			break
		}

		// NPC attacks - roll d20 + level vs player AC
		playerAC := 10 + player.Armor
		npcAttackRoll := rollD20() + npc.Level
		if npcAttackRoll >= playerAC {
			// Hit! Deal damage
			npcDamage := npc.Damage

			// Apply player armor reduction
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

// RunSimulation runs multiple combat simulations and returns aggregated results
func RunSimulation(player, npc CombatantStats, iterations int) SimulationResult {
	result := SimulationResult{
		Simulations: iterations,
		PlayerStats: player,
		NPCStats:    npc,
		MinRounds:   999999,
		MaxRounds:   0,
	}

	totalRounds := 0
	totalPlayerHPLeft := 0
	totalDamageDealt := 0
	totalDamageTaken := 0

	for i := 0; i < iterations; i++ {
		combat := SimulateCombat(player, npc)

		if combat.PlayerWon {
			result.PlayerWins++
			totalPlayerHPLeft += combat.PlayerHPRemain
		} else {
			result.NPCWins++
		}

		totalRounds += combat.Rounds
		totalDamageDealt += combat.PlayerDamageOut
		totalDamageTaken += combat.PlayerDamageIn

		if combat.Rounds < result.MinRounds {
			result.MinRounds = combat.Rounds
		}
		if combat.Rounds > result.MaxRounds {
			result.MaxRounds = combat.Rounds
		}
	}

	result.WinRate = float64(result.PlayerWins) / float64(iterations) * 100
	result.AvgRounds = float64(totalRounds) / float64(iterations)
	result.AvgDamageDealt = float64(totalDamageDealt) / float64(iterations)
	result.AvgDamageTaken = float64(totalDamageTaken) / float64(iterations)

	if result.PlayerWins > 0 {
		result.AvgPlayerHPLeft = float64(totalPlayerHPLeft) / float64(result.PlayerWins)
	}

	return result
}

// SimulateCasterCombat runs a single combat between a spellcasting player and NPC
// Casters alternate between spells (when mana available) and weapon attacks
// Spells auto-hit (no roll needed), making them more reliable than weapon attacks
func SimulateCasterCombat(caster CasterStats, npc CombatantStats) CombatResult {
	result := CombatResult{}

	playerHP := caster.Health
	playerMana := caster.Mana
	npcHP := npc.Health
	round := 0
	maxRounds := 1000

	for playerHP > 0 && npcHP > 0 && round < maxRounds {
		round++

		// Player attacks - use spell if mana available, otherwise weapon
		if playerMana >= caster.SpellCost && caster.SpellDice != "" {
			// Cast spell - auto-hit!
			spellDamage := rollDice(caster.SpellDice, caster.CastingMod)
			actualDamage := spellDamage - npc.Armor
			if actualDamage < 1 {
				actualDamage = 1
			}
			npcHP -= actualDamage
			result.PlayerDamageOut += actualDamage
			playerMana -= caster.SpellCost
		} else {
			// Weapon attack - must roll to hit
			attackRoll := rollD20() + caster.StrMod
			if attackRoll >= npc.ArmorClass {
				var damage int
				if caster.DamageDice != "" {
					damage = rollDice(caster.DamageDice, caster.StrMod)
				} else {
					damage = 1 + caster.StrMod
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

		// NPC attacks - roll d20 + level vs player AC
		playerAC := 10 + caster.Armor
		npcAttackRoll := rollD20() + npc.Level
		if npcAttackRoll >= playerAC {
			// Hit! Deal damage
			npcDamage := npc.Damage
			actualDamage := npcDamage - caster.Armor
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

// RunCasterSimulation runs multiple caster combat simulations and returns aggregated results
func RunCasterSimulation(caster CasterStats, npc CombatantStats, iterations int) SimulationResult {
	result := SimulationResult{
		Simulations: iterations,
		PlayerStats: caster.CombatantStats,
		NPCStats:    npc,
		MinRounds:   999999,
		MaxRounds:   0,
	}

	totalRounds := 0
	totalPlayerHPLeft := 0
	totalDamageDealt := 0
	totalDamageTaken := 0

	for i := 0; i < iterations; i++ {
		combat := SimulateCasterCombat(caster, npc)

		if combat.PlayerWon {
			result.PlayerWins++
			totalPlayerHPLeft += combat.PlayerHPRemain
		} else {
			result.NPCWins++
		}

		totalRounds += combat.Rounds
		totalDamageDealt += combat.PlayerDamageOut
		totalDamageTaken += combat.PlayerDamageIn

		if combat.Rounds < result.MinRounds {
			result.MinRounds = combat.Rounds
		}
		if combat.Rounds > result.MaxRounds {
			result.MaxRounds = combat.Rounds
		}
	}

	result.WinRate = float64(result.PlayerWins) / float64(iterations) * 100
	result.AvgRounds = float64(totalRounds) / float64(iterations)
	result.AvgDamageDealt = float64(totalDamageDealt) / float64(iterations)
	result.AvgDamageTaken = float64(totalDamageTaken) / float64(iterations)

	if result.PlayerWins > 0 {
		result.AvgPlayerHPLeft = float64(totalPlayerHPLeft) / float64(result.PlayerWins)
	}

	return result
}

// FloorScalingResult holds results for combat across multiple floors
type FloorScalingResult struct {
	Floor         int
	WinRate       float64
	AvgRounds     float64
	AvgHPLeft     float64
	AvgDamageTaken float64
}

// RunFloorScalingSim tests a player against increasingly difficult mobs
func RunFloorScalingSim(player CombatantStats, baseMob CombatantStats, floors []int, iterations int) []FloorScalingResult {
	results := make([]FloorScalingResult, 0, len(floors))

	for _, floor := range floors {
		// Scale mob stats by floor (matching tower spawner logic)
		scaledMob := ScaleMobForFloor(baseMob, floor)

		simResult := RunSimulation(player, scaledMob, iterations)

		results = append(results, FloorScalingResult{
			Floor:          floor,
			WinRate:        simResult.WinRate,
			AvgRounds:      simResult.AvgRounds,
			AvgHPLeft:      simResult.AvgPlayerHPLeft,
			AvgDamageTaken: simResult.AvgDamageTaken,
		})
	}

	return results
}

// ScaleMobForFloor scales a base mob's stats for a given floor
// This should match the scaling in tower/spawner.go
func ScaleMobForFloor(base CombatantStats, floor int) CombatantStats {
	scaled := base

	// Determine tier based on floor (matches tower logic)
	tier := 1
	switch {
	case floor >= 21:
		tier = 4
	case floor >= 11:
		tier = 3
	case floor >= 6:
		tier = 2
	default:
		tier = 1
	}

	// Scale stats by tier (rough approximation)
	tierMultiplier := float64(tier)
	scaled.Health = int(float64(base.Health) * tierMultiplier)
	scaled.MaxHealth = scaled.Health
	scaled.Damage = int(float64(base.Damage) * tierMultiplier)
	scaled.Level = base.Level + (tier - 1) * 5

	// Also scale by floor within tier
	floorBonus := float64(floor) * 0.1
	scaled.Health = int(float64(scaled.Health) * (1 + floorBonus))
	scaled.MaxHealth = scaled.Health
	scaled.Damage = int(float64(scaled.Damage) * (1 + floorBonus * 0.5))

	scaled.ArmorClass = 10 + scaled.Armor

	return scaled
}

// LevelProgressionResult holds results for testing player progression
type LevelProgressionResult struct {
	PlayerLevel int
	WinRate     float64
	AvgRounds   float64
	AvgHPLeft   float64
}

// RunLevelProgressionSim tests different player levels against a fixed mob
func RunLevelProgressionSim(basePlayer CombatantStats, mob CombatantStats, levels []int, iterations int) []LevelProgressionResult {
	results := make([]LevelProgressionResult, 0, len(levels))

	for _, level := range levels {
		scaledPlayer := ScalePlayerForLevel(basePlayer, level)

		simResult := RunSimulation(scaledPlayer, mob, iterations)

		results = append(results, LevelProgressionResult{
			PlayerLevel: level,
			WinRate:     simResult.WinRate,
			AvgRounds:   simResult.AvgRounds,
			AvgHPLeft:   simResult.AvgPlayerHPLeft,
		})
	}

	return results
}

// ScalePlayerForLevel scales a base player's stats for a given level
// Uses actual game HP formula: StartingHP + (level-1) * avg_hit_die
// Warrior: 10 base + 6 per level (d10 avg)
// Mage: 6 base + 4 per level (d6 avg)
// Default assumes warrior-like progression
func ScalePlayerForLevel(base CombatantStats, level int) CombatantStats {
	scaled := base

	// HP scales using actual game formula:
	// StartingHP (default 10 for warrior) + (level-1) * avg_hit_die (6 for d10)
	baseHP := 10 // Warrior starting HP
	hpPerLevel := 6 // Average of d10 hit die
	scaled.Health = baseHP + (level-1)*hpPerLevel
	scaled.MaxHealth = scaled.Health
	scaled.Level = level

	// Ability scores might increase with level (simplified)
	// Every 4 levels, +1 to STR mod
	scaled.StrMod = base.StrMod + (level-1)/4

	return scaled
}
