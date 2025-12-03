// Package balance provides Monte Carlo simulation tools for game balance testing.
package balance

import (
	"math"
	"math/rand"
)

// ClassConfig represents a player class for leveling simulation
type ClassConfig struct {
	Name        string
	HP          int    // Starting HP
	Armor       int    // Armor value
	StrMod      int    // Strength modifier (attack and damage)
	Weapon      string // Weapon damage dice
	BonusDmg    int    // Extra damage (sneak attack, etc.)
	RootLevel   int    // Level when class gets root ability (0 = never)
	StunLevel   int    // Level when class gets stun ability (0 = never)
	SpellDmg    string // Spell damage dice (for casters, replaces melee)
	SpellIntMod int    // INT modifier for spell damage
	Mana        int    // Starting mana pool
	SpellCost   int    // Mana cost per spell cast
	ManaRegen   int    // Mana regenerated per round
}

// DefaultClasses returns the game's class configurations for leveling simulation
func DefaultClasses() []ClassConfig {
	return []ClassConfig{
		// Melee classes: use weapon + STR mod
		// Name, HP, Armor, StrMod, Weapon, BonusDmg, RootLvl, StunLvl, SpellDmg, IntMod, Mana, SpellCost, ManaRegen
		{"Warrior", 110, 3, 1, "1d8", 0, 5, 3, "", 0, 0, 0, 0},   // Shield Bash at 3 (stun), Hamstring at 5 (root)
		{"Paladin", 105, 3, 1, "1d8", 0, 6, 4, "", 0, 0, 0, 0},   // Rebuke at 4 (stun), Commanding Presence at 6 (root)
		{"Ranger", 100, 2, 1, "1d8", 0, 2, 0, "", 0, 0, 0, 0},    // Ensnaring Strike at 2 (root), no stun
		{"Rogue", 95, 1, 2, "1d6", 2, 4, 2, "", 0, 0, 0, 0},      // Cheap Shot at 2 (stun), Hamstring at 4 (root)
		{"Cleric", 100, 2, 0, "1d6", 0, 3, 0, "", 0, 0, 0, 0},    // Command at 3 (root), no stun
		// Mage: uses spells instead of melee (Magic Missile 1d4+1, Flare 1d6+INT at lvl 1)
		// Average spell: 1d6 + 2 INT mod = ~5.5 damage, cost 6-8 mana, 100 mana pool, 2 mana/round regen
		{"Mage", 90, 0, -1, "1d4", 0, 3, 0, "1d6", 2, 100, 7, 2}, // Hold Person at 3 (root), uses spells
	}
}

// MobTypeConfig represents flee behavior for different mob types
type MobTypeConfig struct {
	Name       string
	FleeThresh float64 // HP % at which mob tries to flee (0 = never)
	FleeChance float64 // Chance per round to flee when below threshold
}

// DefaultMobTypes returns the game's mob type configurations
func DefaultMobTypes() []MobTypeConfig {
	return []MobTypeConfig{
		{"Beast", 0.15, 0.35},
		{"Humanoid", 0.12, 0.35},
		{"Giant", 0.10, 0.35},
		{"Demon", 0.05, 0.35},
		{"Undead", 0.0, 0.35},   // Never flees
		{"Construct", 0.0, 0.35}, // Never flees
	}
}

// LevelingResult holds results for a single class at a single level transition
type LevelingResult struct {
	ClassName   string
	FromLevel   int
	ToLevel     int
	WinRate     float64
	EscapeRate  float64
	DeathRate   float64
	AvgRounds   float64
	KillsNeeded float64 // Successful kills needed for XP
	TimePerKill float64 // Seconds per successful kill (including travel)
	TotalTime   float64 // Total minutes to level
	HasRoot     bool
	HasStun     bool
}

// LevelingSummary holds aggregated leveling results for a class
type LevelingSummary struct {
	ClassName     string
	LevelTimes    []float64 // Time in minutes for each level transition
	TotalTime     float64   // Total time to reach target level
	AvgWinRate    float64
	AvgEscapeRate float64
}

// XPConfig holds XP-related configuration
type XPConfig struct {
	XPPerKill       float64 // Average XP per mob kill
	TravelTime      float64 // Seconds between fights (finding next mob, regen)
	SecsPerRound    float64 // Seconds per combat round
	DeathPenaltySec float64 // Time lost per death (respawn, recovery)
}

// DefaultXPConfig returns the default XP configuration
func DefaultXPConfig() XPConfig {
	return XPConfig{
		XPPerKill:       11.0,
		TravelTime:      30.0,
		SecsPerRound:    3.0,
		DeathPenaltySec: 120.0, // 2 minutes lost per death
	}
}

// FloorConfig holds floor spawn configuration
type FloorConfig struct {
	MinRooms            int     // Minimum rooms per floor
	MaxRooms            int     // Maximum rooms per floor
	RoomMobChance       float64 // Chance of mobs in regular rooms
	RoomMobMin          int     // Min mobs per regular room
	RoomMobMax          int     // Max mobs per regular room
	CorridorMobChance   float64 // Chance of mobs in corridors
	CorridorMobMin      int     // Min mobs per corridor
	CorridorMobMax      int     // Max mobs per corridor
	RoomToCorridorRatio float64 // Ratio of rooms to corridors
	RespawnTimeSec      float64 // Mob respawn time in seconds
	RoomTraversalSec    float64 // Time to move between rooms
	PlayersOnline       int     // Number of players online competing for mobs
	FloorsAvailable     int     // Number of floors players spread across
	DynamicSpawns       bool    // Enable dynamic spawn scaling based on player count
	TargetMobsPerPlayer float64 // Target mobs per player when dynamic spawns enabled
}

// DefaultFloorConfig returns the default floor configuration based on game data
func DefaultFloorConfig() FloorConfig {
	return FloorConfig{
		MinRooms:            20,
		MaxRooms:            50,
		RoomMobChance:       0.80, // 80% chance in regular rooms
		RoomMobMin:          1,
		RoomMobMax:          3,
		CorridorMobChance:   0.60, // 60% chance in corridors
		CorridorMobMin:      1,
		CorridorMobMax:      2,
		RoomToCorridorRatio: 0.6,   // ~60% rooms, 40% corridors
		RespawnTimeSec:      120.0, // Tier 1 mobs respawn in 2 minutes
		RoomTraversalSec:    5.0,   // 5 seconds to move between rooms
		PlayersOnline:       1,     // Solo play by default
		FloorsAvailable:     5,     // Players spread across floors 1-5
		DynamicSpawns:       false, // Static spawns by default
		TargetMobsPerPlayer: 30.0,  // Target 30 mobs per player when dynamic
	}
}

// CalculateFloorMobs calculates expected number of mobs on a floor
func CalculateFloorMobs(config FloorConfig) float64 {
	avgRooms := float64(config.MinRooms+config.MaxRooms) / 2.0

	// Split into room types (excluding stairs, boss, treasure which are few)
	regularRooms := avgRooms * config.RoomToCorridorRatio * 0.85 // 85% are regular rooms
	corridors := avgRooms * (1 - config.RoomToCorridorRatio) * 0.85

	// Calculate expected mobs from regular rooms
	avgRoomMobs := float64(config.RoomMobMin+config.RoomMobMax) / 2.0
	roomMobs := regularRooms * config.RoomMobChance * avgRoomMobs

	// Calculate expected mobs from corridors
	avgCorridorMobs := float64(config.CorridorMobMin+config.CorridorMobMax) / 2.0
	corridorMobs := corridors * config.CorridorMobChance * avgCorridorMobs

	return roomMobs + corridorMobs
}

// CalculateEffectiveMobsPerPlayer calculates mobs available per player accounting for competition
func CalculateEffectiveMobsPerPlayer(config FloorConfig) float64 {
	// If dynamic spawns enabled, return target mobs per player
	if config.DynamicSpawns {
		return config.TargetMobsPerPlayer
	}

	totalMobsPerFloor := CalculateFloorMobs(config)

	if config.PlayersOnline <= 1 {
		return totalMobsPerFloor
	}

	// Players spread across available floors
	// Assume even distribution for simplicity
	playersPerFloor := float64(config.PlayersOnline) / float64(config.FloorsAvailable)
	if playersPerFloor < 1 {
		playersPerFloor = 1
	}

	// Each player gets a share of mobs on their floor
	mobsPerPlayer := totalMobsPerFloor / playersPerFloor

	return mobsPerPlayer
}

// CalculateDynamicSpawnMultiplier calculates how much to increase spawns for player count
func CalculateDynamicSpawnMultiplier(config FloorConfig) float64 {
	if !config.DynamicSpawns || config.PlayersOnline <= 1 {
		return 1.0
	}

	baseMobsPerFloor := CalculateFloorMobs(FloorConfig{
		MinRooms:            config.MinRooms,
		MaxRooms:            config.MaxRooms,
		RoomMobChance:       config.RoomMobChance,
		RoomMobMin:          config.RoomMobMin,
		RoomMobMax:          config.RoomMobMax,
		CorridorMobChance:   config.CorridorMobChance,
		CorridorMobMin:      config.CorridorMobMin,
		CorridorMobMax:      config.CorridorMobMax,
		RoomToCorridorRatio: config.RoomToCorridorRatio,
	})

	// How many mobs we need total across all floors
	playersPerFloor := float64(config.PlayersOnline) / float64(config.FloorsAvailable)
	targetMobsPerFloor := config.TargetMobsPerPlayer * playersPerFloor

	// Multiplier needed to achieve target
	multiplier := targetMobsPerFloor / baseMobsPerFloor
	if multiplier < 1.0 {
		multiplier = 1.0 // Never reduce spawns below base
	}

	return multiplier
}

// CalculateEffectiveRespawnRate calculates respawn rate per player accounting for competition
func CalculateEffectiveRespawnRate(config FloorConfig) float64 {
	// Respawns per second on one floor
	totalMobsPerFloor := CalculateFloorMobs(config)
	respawnsPerSecond := totalMobsPerFloor / config.RespawnTimeSec

	if config.PlayersOnline <= 1 {
		return respawnsPerSecond
	}

	// Players compete for respawns on shared floors
	playersPerFloor := float64(config.PlayersOnline) / float64(config.FloorsAvailable)
	if playersPerFloor < 1 {
		playersPerFloor = 1
	}

	// Your share of respawns
	return respawnsPerSecond / playersPerFloor
}

// FloorLevelingResult holds results for leveling simulation with floor mechanics
type FloorLevelingResult struct {
	LevelingResult
	MobsOnFloor      float64 // Average mobs available on floor
	MobsKilledPerRun float64 // Mobs killed before needing to wait/move
	RespawnWaitTime  float64 // Time spent waiting for respawns (seconds)
	SearchTime       float64 // Time spent searching for mobs (seconds)
	FloorsVisited    float64 // Number of floors visited to level
}

// XPToNextLevel calculates XP needed to go from level to level+1
// Formula: 100 * level^1.5
func XPToNextLevel(level int) int {
	nextLevelXP := int(100 * math.Pow(float64(level+1), 1.5))
	currentLevelXP := int(100 * math.Pow(float64(level), 1.5))
	return nextLevelXP - currentLevelXP
}

// simulateCombatWithFlee runs a single combat with flee mechanics
func simulateCombatWithFlee(player ClassConfig, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, hasRoot, hasStun bool) (won bool, escaped bool, rounds int, playerHPLeft int) {

	playerHP := player.HP
	playerMana := player.Mana
	npcHP := mobHP
	npcMaxHP := mobHP
	npcAC := 10 + mobArmor
	rooted := false
	stunRounds := 0

	// Determine if this is a spell caster (has spell damage defined)
	isSpellCaster := player.SpellDmg != ""

	for playerHP > 0 && npcHP > 0 {
		rounds++

		// Regenerate mana each round
		if isSpellCaster {
			playerMana += player.ManaRegen
			if playerMana > player.Mana {
				playerMana = player.Mana
			}
		}

		// Apply stun at start of combat if available
		if hasStun && rounds == 1 {
			stunRounds = 1 // 1 round stun
		}

		// Player attacks
		if isSpellCaster && playerMana >= player.SpellCost {
			// Cast spell - auto-hit (like Magic Missile)
			playerMana -= player.SpellCost
			damage := rollDice(player.SpellDmg, player.SpellIntMod)
			// Spells ignore armor (magical damage)
			if damage < 1 {
				damage = 1
			}
			npcHP -= damage
		} else {
			// Melee attack (or out of mana fallback)
			attackRoll := rollD20() + player.StrMod
			if attackRoll >= npcAC {
				damage := rollDice(player.Weapon, player.StrMod) + player.BonusDmg
				actualDmg := damage - mobArmor
				if actualDmg < 1 {
					actualDmg = 1
				}
				npcHP -= actualDmg
			}
		}

		if npcHP <= 0 {
			return true, false, rounds, playerHP
		}

		// Check if mob wants to flee
		hpPercent := float64(npcHP) / float64(npcMaxHP)
		if fleeThresh > 0 && hpPercent <= fleeThresh && !rooted {
			if rand.Float64() < fleeChance {
				// Mob tries to flee
				if hasRoot {
					// Player can root it
					rooted = true
				} else {
					// Mob escapes
					return false, true, rounds, playerHP
				}
			}
		}

		// Mob attacks (if not stunned)
		if stunRounds > 0 {
			stunRounds--
		} else {
			actualDmg := mobDmg - player.Armor
			if actualDmg < 1 {
				actualDmg = 1
			}
			playerHP -= actualDmg
		}
	}

	if playerHP <= 0 {
		return false, false, rounds, 0
	}
	return true, false, rounds, playerHP
}

// SimulateLeveling runs leveling simulations for a class at a specific level transition
func SimulateLeveling(class ClassConfig, fromLevel int, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, iterations int) LevelingResult {

	result := LevelingResult{
		ClassName: class.Name,
		FromLevel: fromLevel,
		ToLevel:   fromLevel + 1,
		HasRoot:   fromLevel >= class.RootLevel && class.RootLevel > 0,
		HasStun:   fromLevel >= class.StunLevel && class.StunLevel > 0,
	}

	wins := 0
	escapes := 0
	deaths := 0
	totalRounds := 0

	for i := 0; i < iterations; i++ {
		won, escaped, rounds, _ := simulateCombatWithFlee(
			class, mobHP, mobDmg, mobArmor,
			fleeThresh, fleeChance,
			result.HasRoot, result.HasStun,
		)

		totalRounds += rounds
		if won {
			wins++
		} else if escaped {
			escapes++
		} else {
			deaths++
		}
	}

	result.WinRate = float64(wins) / float64(iterations) * 100
	result.EscapeRate = float64(escapes) / float64(iterations) * 100
	result.DeathRate = float64(deaths) / float64(iterations) * 100
	result.AvgRounds = float64(totalRounds) / float64(iterations)

	// Calculate time to level
	xpNeeded := XPToNextLevel(fromLevel)
	result.KillsNeeded = float64(xpNeeded) / xpConfig.XPPerKill

	// Time per successful kill
	combatTime := result.AvgRounds * xpConfig.SecsPerRound
	result.TimePerKill = combatTime + xpConfig.TravelTime

	// Effective kills rate (accounting for escapes and deaths)
	killRate := result.WinRate / 100
	if killRate > 0 {
		attemptsNeeded := result.KillsNeeded / killRate

		// Time from combat attempts
		attemptTime := attemptsNeeded * result.TimePerKill

		// Add death penalty time
		expectedDeaths := attemptsNeeded * (result.DeathRate / 100)
		deathTime := expectedDeaths * xpConfig.DeathPenaltySec

		result.TotalTime = (attemptTime + deathTime) / 60 // Convert to minutes
	} else {
		result.TotalTime = 9999 // Essentially infinite if 0% win rate
	}

	return result
}

// RunLevelingSimulation runs leveling simulation for all classes across multiple level transitions
func RunLevelingSimulation(classes []ClassConfig, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, startLevel, endLevel, iterations int) []LevelingSummary {

	summaries := make([]LevelingSummary, 0, len(classes))

	for _, class := range classes {
		summary := LevelingSummary{
			ClassName:  class.Name,
			LevelTimes: make([]float64, 0, endLevel-startLevel),
		}

		totalWinRate := 0.0
		totalEscapeRate := 0.0
		levelCount := 0

		for lvl := startLevel; lvl < endLevel; lvl++ {
			result := SimulateLeveling(class, lvl, mobHP, mobDmg, mobArmor,
				fleeThresh, fleeChance, xpConfig, iterations)

			summary.LevelTimes = append(summary.LevelTimes, result.TotalTime)
			summary.TotalTime += result.TotalTime
			totalWinRate += result.WinRate
			totalEscapeRate += result.EscapeRate
			levelCount++
		}

		if levelCount > 0 {
			summary.AvgWinRate = totalWinRate / float64(levelCount)
			summary.AvgEscapeRate = totalEscapeRate / float64(levelCount)
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

// RunDetailedLevelingSimulation returns detailed results for each class and level
func RunDetailedLevelingSimulation(classes []ClassConfig, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, startLevel, endLevel, iterations int) []LevelingResult {

	results := make([]LevelingResult, 0)

	for _, class := range classes {
		for lvl := startLevel; lvl < endLevel; lvl++ {
			result := SimulateLeveling(class, lvl, mobHP, mobDmg, mobArmor,
				fleeThresh, fleeChance, xpConfig, iterations)
			results = append(results, result)
		}
	}

	return results
}

// RunMobTypeLevelingSim runs leveling simulations against different mob types
func RunMobTypeLevelingSim(class ClassConfig, mobTypes []MobTypeConfig, mobHP, mobDmg, mobArmor int,
	xpConfig XPConfig, level, iterations int) map[string]LevelingResult {

	results := make(map[string]LevelingResult)

	for _, mobType := range mobTypes {
		result := SimulateLeveling(class, level, mobHP, mobDmg, mobArmor,
			mobType.FleeThresh, mobType.FleeChance, xpConfig, iterations)
		result.ClassName = class.Name + " vs " + mobType.Name
		results[mobType.Name] = result
	}

	return results
}

// SimulateLevelingWithFloor runs leveling simulation accounting for floor spawn/respawn mechanics
func SimulateLevelingWithFloor(class ClassConfig, fromLevel int, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, floorConfig FloorConfig, iterations int) FloorLevelingResult {

	// First get basic combat stats
	baseResult := SimulateLeveling(class, fromLevel, mobHP, mobDmg, mobArmor,
		fleeThresh, fleeChance, xpConfig, iterations)

	result := FloorLevelingResult{
		LevelingResult: baseResult,
	}

	// Calculate mobs available per player (accounting for competition)
	result.MobsOnFloor = CalculateEffectiveMobsPerPlayer(floorConfig)

	// Calculate how many mobs we can kill accounting for win rate
	effectiveKillRate := baseResult.WinRate / 100
	if effectiveKillRate <= 0 {
		effectiveKillRate = 0.01 // Avoid division by zero
	}

	// XP needed and kills required
	xpNeeded := XPToNextLevel(fromLevel)
	killsNeeded := float64(xpNeeded) / xpConfig.XPPerKill

	// Combat time per fight (successful or not)
	combatTimeSec := baseResult.AvgRounds * xpConfig.SecsPerRound

	// Average rooms per floor
	avgRooms := float64(floorConfig.MinRooms+floorConfig.MaxRooms) / 2.0

	// Calculate effective respawn rate per player
	respawnsPerSecPerPlayer := CalculateEffectiveRespawnRate(floorConfig)

	// Simulate the leveling process
	totalTimeSec := 0.0
	totalKills := 0.0
	floorsVisited := 1.0
	totalSearchTime := 0.0
	totalRespawnWait := 0.0

	// Track mobs available to this player on current floor
	mobsRemaining := result.MobsOnFloor
	floorStartTime := 0.0

	for totalKills < killsNeeded {
		// Check if we've run out of mobs
		if mobsRemaining < 1 {
			// Calculate respawns available to us since floor start
			timeSinceFloorStart := totalTimeSec - floorStartTime
			respawnedForUs := timeSinceFloorStart * respawnsPerSecPerPlayer

			if respawnedForUs >= 1 {
				// Some mobs have respawned for us
				mobsRemaining = math.Min(respawnedForUs, result.MobsOnFloor)
				floorStartTime = totalTimeSec
			} else {
				// Wait for respawn or move to new floor
				timeToNextFloor := avgRooms * floorConfig.RoomTraversalSec * 0.5

				// Time until we get 1 respawn
				timeToRespawn := (1.0 - respawnedForUs) / respawnsPerSecPerPlayer
				if respawnsPerSecPerPlayer <= 0 {
					timeToRespawn = 9999 // Never respawns
				}

				if timeToNextFloor < timeToRespawn {
					// Faster to go to new floor
					totalTimeSec += timeToNextFloor
					floorsVisited++
					mobsRemaining = result.MobsOnFloor
					floorStartTime = totalTimeSec
				} else {
					// Wait for respawn
					totalTimeSec += timeToRespawn
					totalRespawnWait += timeToRespawn
					mobsRemaining = 1
				}
				continue
			}
		}

		// Calculate search time based on mob density
		// With competition, effective density is lower
		mobDensity := mobsRemaining / avgRooms
		if mobDensity > 1 {
			mobDensity = 1
		}
		if mobDensity < 0.01 {
			mobDensity = 0.01 // Minimum density to avoid infinite search
		}

		// Higher density = less search time
		roomsToSearch := 1.0 / mobDensity
		if roomsToSearch > avgRooms {
			roomsToSearch = avgRooms
		}
		searchTime := roomsToSearch * floorConfig.RoomTraversalSec
		totalSearchTime += searchTime
		totalTimeSec += searchTime

		// Attempt combat
		totalTimeSec += combatTimeSec

		// Determine outcome
		roll := rand.Float64() * 100
		if roll < baseResult.WinRate {
			// Won - got the kill
			totalKills++
			mobsRemaining--
		} else if roll < baseResult.WinRate+baseResult.EscapeRate {
			// Mob escaped - no kill, mob removed from floor
			mobsRemaining--
		} else {
			// Died - death penalty
			totalTimeSec += xpConfig.DeathPenaltySec
		}
	}

	result.SearchTime = totalSearchTime
	result.RespawnWaitTime = totalRespawnWait
	result.FloorsVisited = floorsVisited
	result.MobsKilledPerRun = killsNeeded / floorsVisited
	result.TotalTime = totalTimeSec / 60 // Convert to minutes

	return result
}

// FloorLevelingSummary holds aggregated floor leveling results
type FloorLevelingSummary struct {
	ClassName       string
	LevelTimes      []float64 // Time in minutes for each level transition
	TotalTime       float64   // Total time to reach target level
	AvgWinRate      float64
	AvgEscapeRate   float64
	AvgFloorsPerLvl float64   // Average floors visited per level
	AvgSearchTime   float64   // Average search time per level (seconds)
	AvgRespawnWait  float64   // Average respawn wait per level (seconds)
}

// RunFloorLevelingSimulation runs leveling simulation with floor mechanics for all classes
func RunFloorLevelingSimulation(classes []ClassConfig, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, floorConfig FloorConfig,
	startLevel, endLevel, iterations int) []FloorLevelingSummary {

	summaries := make([]FloorLevelingSummary, 0, len(classes))

	for _, class := range classes {
		summary := FloorLevelingSummary{
			ClassName:  class.Name,
			LevelTimes: make([]float64, 0, endLevel-startLevel),
		}

		totalWinRate := 0.0
		totalEscapeRate := 0.0
		totalFloors := 0.0
		totalSearch := 0.0
		totalRespawn := 0.0
		levelCount := 0

		for lvl := startLevel; lvl < endLevel; lvl++ {
			result := SimulateLevelingWithFloor(class, lvl, mobHP, mobDmg, mobArmor,
				fleeThresh, fleeChance, xpConfig, floorConfig, iterations)

			summary.LevelTimes = append(summary.LevelTimes, result.TotalTime)
			summary.TotalTime += result.TotalTime
			totalWinRate += result.WinRate
			totalEscapeRate += result.EscapeRate
			totalFloors += result.FloorsVisited
			totalSearch += result.SearchTime
			totalRespawn += result.RespawnWaitTime
			levelCount++
		}

		if levelCount > 0 {
			summary.AvgWinRate = totalWinRate / float64(levelCount)
			summary.AvgEscapeRate = totalEscapeRate / float64(levelCount)
			summary.AvgFloorsPerLvl = totalFloors / float64(levelCount)
			summary.AvgSearchTime = totalSearch / float64(levelCount)
			summary.AvgRespawnWait = totalRespawn / float64(levelCount)
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

// RunDetailedFloorLevelingSimulation returns detailed results with floor mechanics
func RunDetailedFloorLevelingSimulation(classes []ClassConfig, mobHP, mobDmg, mobArmor int,
	fleeThresh, fleeChance float64, xpConfig XPConfig, floorConfig FloorConfig,
	startLevel, endLevel, iterations int) []FloorLevelingResult {

	results := make([]FloorLevelingResult, 0)

	for _, class := range classes {
		for lvl := startLevel; lvl < endLevel; lvl++ {
			result := SimulateLevelingWithFloor(class, lvl, mobHP, mobDmg, mobArmor,
				fleeThresh, fleeChance, xpConfig, floorConfig, iterations)
			results = append(results, result)
		}
	}

	return results
}
