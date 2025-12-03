// balance is a Monte Carlo simulator for testing game balance in OpenTowerMUD.
//
// Usage:
//
//	balance [command] [options]
//
// Commands:
//
//	combat     - Simulate combat between a player and NPC
//	floors     - Test player performance across tower floors
//	levels     - Test how player level affects combat outcomes
//	spells     - Test spell/CC effectiveness
//	leveling   - Test time to level by class with mob flee mechanics
//	sweep      - Run a comprehensive balance sweep
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/utilities/balance"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "combat":
		runCombatSim()
	case "floors":
		runFloorSim()
	case "levels":
		runLevelSim()
	case "spells":
		runSpellSim()
	case "leveling":
		runLevelingSim()
	case "sweep":
		runSweep()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`OpenTowerMUD Balance Simulator

A Monte Carlo simulator for testing game balance.

Usage: balance <command> [options]

Commands:
  combat    Simulate combat between a player and NPC
  floors    Test player performance across tower floors
  levels    Test how player level affects combat outcomes
  spells    Test spell/CC effectiveness against fleeing mobs
  leveling  Test time to level by class with mob flee mechanics
  sweep     Run a comprehensive balance sweep

Examples:
  balance combat -player-hp=100 -player-str=12 -player-weapon="1d6" -npc-hp=50 -npc-damage=8 -npc-armor=2
  balance floors -player-level=5 -player-hp=150 -iterations=10000
  balance levels -npc-hp=80 -npc-damage=10 -npc-armor=2
  balance spells -flee-threshold=0.2
  balance leveling -start-level=1 -end-level=5
  balance sweep

Use "balance <command> -h" for more information about a command.`)
}

func runCombatSim() {
	fs := flag.NewFlagSet("combat", flag.ExitOnError)

	// Player stats
	playerHP := fs.Int("player-hp", 100, "Player health")
	playerArmor := fs.Int("player-armor", 0, "Player armor (damage reduction)")
	playerStr := fs.Int("player-str", 10, "Player strength score")
	playerWeapon := fs.String("player-weapon", "1d4", "Player weapon damage dice (e.g., '1d6', '2d4+1')")
	playerLevel := fs.Int("player-level", 1, "Player level")

	// NPC stats
	npcHP := fs.Int("npc-hp", 50, "NPC health")
	npcDamage := fs.Int("npc-damage", 5, "NPC damage per hit")
	npcArmor := fs.Int("npc-armor", 0, "NPC armor (damage reduction and AC bonus)")
	npcLevel := fs.Int("npc-level", 1, "NPC level")
	npcName := fs.String("npc-name", "Mob", "NPC name")

	// Simulation options
	iterations := fs.Int("iterations", 10000, "Number of simulations to run")

	fs.Parse(os.Args[2:])

	// Calculate STR modifier: (score - 10) / 2
	strMod := (*playerStr - 10) / 2

	player := balance.NewPlayerStats("Player", *playerLevel, *playerHP, *playerArmor, strMod, *playerWeapon)
	npc := balance.NewNPCStats(*npcName, *npcLevel, *npcHP, *npcArmor, *npcDamage)

	fmt.Println("=== Combat Simulation ===")
	fmt.Println()
	fmt.Printf("Player: Level %d, %d HP, %d Armor, STR %d (mod %+d), Weapon: %s\n",
		player.Level, player.Health, player.Armor, *playerStr, strMod, *playerWeapon)
	fmt.Printf("NPC:    Level %d, %d HP, %d Armor (AC %d), %d Damage\n",
		npc.Level, npc.Health, npc.Armor, npc.ArmorClass, npc.Damage)
	fmt.Printf("Iterations: %d\n", *iterations)
	fmt.Println()

	result := balance.RunSimulation(player, npc, *iterations)

	printSimulationResult(result)
}

func runFloorSim() {
	fs := flag.NewFlagSet("floors", flag.ExitOnError)

	// Player stats
	playerHP := fs.Int("player-hp", 100, "Player health")
	playerArmor := fs.Int("player-armor", 0, "Player armor")
	playerStr := fs.Int("player-str", 10, "Player strength score")
	playerWeapon := fs.String("player-weapon", "1d6", "Player weapon damage dice")
	playerLevel := fs.Int("player-level", 1, "Player level")

	// Base mob stats (will be scaled per floor)
	baseMobHP := fs.Int("mob-hp", 30, "Base mob health (before scaling)")
	baseMobDamage := fs.Int("mob-damage", 5, "Base mob damage (before scaling)")
	baseMobArmor := fs.Int("mob-armor", 1, "Base mob armor")

	// Floor range
	startFloor := fs.Int("start-floor", 1, "Starting floor")
	endFloor := fs.Int("end-floor", 20, "Ending floor")
	stepFloor := fs.Int("step", 1, "Floor step")

	iterations := fs.Int("iterations", 5000, "Iterations per floor")

	fs.Parse(os.Args[2:])

	strMod := (*playerStr - 10) / 2

	player := balance.NewPlayerStats("Player", *playerLevel, *playerHP, *playerArmor, strMod, *playerWeapon)
	baseMob := balance.NewNPCStats("Mob", 1, *baseMobHP, *baseMobArmor, *baseMobDamage)

	// Build floor list
	floors := make([]int, 0)
	for f := *startFloor; f <= *endFloor; f += *stepFloor {
		floors = append(floors, f)
	}

	fmt.Println("=== Floor Scaling Simulation ===")
	fmt.Println()
	fmt.Printf("Player: Level %d, %d HP, %d Armor, STR %d, Weapon: %s\n",
		*playerLevel, *playerHP, *playerArmor, *playerStr, *playerWeapon)
	fmt.Printf("Base Mob: %d HP, %d Damage, %d Armor\n",
		*baseMobHP, *baseMobDamage, *baseMobArmor)
	fmt.Printf("Testing floors %d-%d (step %d), %d iterations each\n",
		*startFloor, *endFloor, *stepFloor, *iterations)
	fmt.Println()

	results := balance.RunFloorScalingSim(player, baseMob, floors, *iterations)

	fmt.Println("Floor | Win Rate | Avg Rounds | Avg HP Left | Avg Damage Taken")
	fmt.Println("------+----------+------------+-------------+-----------------")
	for _, r := range results {
		fmt.Printf("%5d | %6.1f%% | %10.1f | %11.1f | %15.1f\n",
			r.Floor, r.WinRate, r.AvgRounds, r.AvgHPLeft, r.AvgDamageTaken)
	}
}

func runLevelSim() {
	fs := flag.NewFlagSet("levels", flag.ExitOnError)

	// Base player stats
	baseStr := fs.Int("player-str", 10, "Base player strength score")
	playerArmor := fs.Int("player-armor", 0, "Player armor")
	playerWeapon := fs.String("player-weapon", "1d6", "Player weapon damage dice")

	// Fixed NPC stats
	npcHP := fs.Int("npc-hp", 80, "NPC health")
	npcDamage := fs.Int("npc-damage", 10, "NPC damage")
	npcArmor := fs.Int("npc-armor", 2, "NPC armor")

	// Level range
	startLevel := fs.Int("start-level", 1, "Starting player level")
	endLevel := fs.Int("end-level", 20, "Ending player level")

	iterations := fs.Int("iterations", 5000, "Iterations per level")

	fs.Parse(os.Args[2:])

	strMod := (*baseStr - 10) / 2

	basePlayer := balance.NewPlayerStats("Player", 1, 100, *playerArmor, strMod, *playerWeapon)
	npc := balance.NewNPCStats("Mob", 5, *npcHP, *npcArmor, *npcDamage)

	levels := make([]int, 0)
	for l := *startLevel; l <= *endLevel; l++ {
		levels = append(levels, l)
	}

	fmt.Println("=== Level Progression Simulation ===")
	fmt.Println()
	fmt.Printf("Base Player: STR %d, Armor %d, Weapon: %s\n",
		*baseStr, *playerArmor, *playerWeapon)
	fmt.Printf("NPC: %d HP, %d Damage, %d Armor (AC %d)\n",
		*npcHP, *npcDamage, *npcArmor, 10+*npcArmor)
	fmt.Printf("Testing levels %d-%d, %d iterations each\n",
		*startLevel, *endLevel, *iterations)
	fmt.Println()

	results := balance.RunLevelProgressionSim(basePlayer, npc, levels, *iterations)

	fmt.Println("Level | Win Rate | Avg Rounds | Avg HP Left")
	fmt.Println("------+----------+------------+------------")
	for _, r := range results {
		fmt.Printf("%5d | %6.1f%% | %10.1f | %10.1f\n",
			r.PlayerLevel, r.WinRate, r.AvgRounds, r.AvgHPLeft)
	}
}

func runSweep() {
	fmt.Println("=== Comprehensive Balance Sweep ===")
	fmt.Println()
	fmt.Println("Running standard balance checks...")
	fmt.Println()

	iterations := 10000

	// Test 1: Level 1 player vs Tier 1 mob
	fmt.Println("--- Test 1: Level 1 Player vs Floor 1 Mob ---")
	player1 := balance.NewPlayerStats("Player", 1, 100, 0, 0, "1d4") // Unarmed, STR 10
	mob1 := balance.NewNPCStats("Test Rat", 1, 20, 0, 3)
	result1 := balance.RunSimulation(player1, mob1, iterations)
	printSimulationResult(result1)
	assessBalance("Level 1 vs Tier 1", result1.WinRate)
	fmt.Println()

	// Test 2: Level 5 player with weapon vs Floor 5 mob
	fmt.Println("--- Test 2: Level 5 Player (Equipped) vs Floor 5 Mob ---")
	player2 := balance.NewPlayerStats("Player", 5, 140, 3, 1, "1d8") // STR 12, leather armor, longsword
	mob2 := balance.NewNPCStats("Tower Goblin", 5, 60, 2, 8)
	result2 := balance.RunSimulation(player2, mob2, iterations)
	printSimulationResult(result2)
	assessBalance("Level 5 vs Floor 5", result2.WinRate)
	fmt.Println()

	// Test 3: Level 10 player vs Floor 10 boss
	fmt.Println("--- Test 3: Level 10 Player vs Floor 10 Boss ---")
	player3 := balance.NewPlayerStats("Player", 10, 190, 6, 2, "1d10+1") // STR 14, good gear
	mob3 := balance.NewNPCStats("Floor 10 Boss", 12, 200, 5, 20)
	result3 := balance.RunSimulation(player3, mob3, iterations)
	printSimulationResult(result3)
	assessBalance("Level 10 vs Boss", result3.WinRate)
	fmt.Println()

	// Test 4: Undergeared check - ensure weak players struggle
	fmt.Println("--- Test 4: Undergeared Player (Balance Check) ---")
	player4 := balance.NewPlayerStats("Player", 5, 140, 0, 0, "1d4") // No gear
	result4 := balance.RunSimulation(player4, mob2, iterations)
	printSimulationResult(result4)
	if result4.WinRate > 30 {
		fmt.Println("WARNING: Undergeared players may have it too easy")
	} else {
		fmt.Println("OK: Gear matters for combat success")
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println("Target win rates:")
	fmt.Println("  - Same-level content: 60-80%")
	fmt.Println("  - Boss fights: 30-50%")
	fmt.Println("  - Undergeared: <30%")
}

func printSimulationResult(r balance.SimulationResult) {
	fmt.Printf("Results (%d simulations):\n", r.Simulations)
	fmt.Printf("  Win Rate:      %.1f%% (%d wins, %d losses)\n", r.WinRate, r.PlayerWins, r.NPCWins)
	fmt.Printf("  Avg Rounds:    %.1f (min: %d, max: %d)\n", r.AvgRounds, r.MinRounds, r.MaxRounds)
	fmt.Printf("  Avg HP Left:   %.1f (when winning)\n", r.AvgPlayerHPLeft)
	fmt.Printf("  Avg Damage In: %.1f\n", r.AvgDamageTaken)
	fmt.Printf("  Avg Damage Out:%.1f\n", r.AvgDamageDealt)
}

func assessBalance(context string, winRate float64) {
	var assessment string
	switch {
	case winRate < 30:
		assessment = "TOO HARD"
	case winRate < 50:
		assessment = "CHALLENGING"
	case winRate < 70:
		assessment = "BALANCED"
	case winRate < 85:
		assessment = "EASY"
	default:
		assessment = "TOO EASY"
	}

	// Color-code if terminal supports it
	color := ""
	reset := ""
	if isTerminal() {
		switch assessment {
		case "TOO HARD":
			color = "\033[31m" // Red
		case "CHALLENGING":
			color = "\033[33m" // Yellow
		case "BALANCED":
			color = "\033[32m" // Green
		case "EASY":
			color = "\033[33m" // Yellow
		case "TOO EASY":
			color = "\033[31m" // Red
		}
		reset = "\033[0m"
	}

	fmt.Printf("Assessment: %s%s%s\n", color, assessment, reset)
}

func isTerminal() bool {
	// Simple check - could be improved
	return os.Getenv("TERM") != "" && !strings.Contains(os.Getenv("TERM"), "dumb")
}

func runSpellSim() {
	fs := flag.NewFlagSet("spells", flag.ExitOnError)

	// Player stats
	playerHP := fs.Int("player-hp", 140, "Player health")
	playerArmor := fs.Int("player-armor", 3, "Player armor")
	playerStr := fs.Int("player-str", 12, "Player strength score")
	playerWeapon := fs.String("player-weapon", "1d8", "Player weapon damage dice")

	// NPC stats
	npcHP := fs.Int("npc-hp", 60, "NPC health")
	npcDamage := fs.Int("npc-damage", 8, "NPC damage per hit")
	npcArmor := fs.Int("npc-armor", 2, "NPC armor")

	// CC options
	fleeThreshold := fs.Float64("flee-threshold", 0.2, "NPC flee threshold (0.0-1.0, e.g. 0.2 = flee at 20% HP)")

	// Simulation options
	iterations := fs.Int("iterations", 10000, "Number of simulations per spell")

	fs.Parse(os.Args[2:])

	strMod := (*playerStr - 10) / 2

	player := balance.NewPlayerStats("Player", 5, *playerHP, *playerArmor, strMod, *playerWeapon)
	npc := balance.NewNPCStats("Mob", 5, *npcHP, *npcArmor, *npcDamage)

	fmt.Println("=== Spell/CC Balance Simulation ===")
	fmt.Println()
	fmt.Printf("Player: %d HP, %d Armor, STR %d (mod %+d), Weapon: %s\n",
		*playerHP, *playerArmor, *playerStr, strMod, *playerWeapon)
	fmt.Printf("NPC:    %d HP, %d Damage, %d Armor (AC %d)\n",
		*npcHP, *npcDamage, *npcArmor, 10+*npcArmor)
	fmt.Printf("Flee Threshold: %.0f%% HP\n", *fleeThreshold*100)
	fmt.Printf("Iterations: %d per spell\n", *iterations)
	fmt.Println()

	// Run baseline without any spells
	baseConfig := balance.DefaultCombatConfig()
	baseConfig.NPCFleeThresh = *fleeThreshold
	baseResult := balance.RunCCSimulation(player, npc, baseConfig, *iterations)

	fmt.Println("--- Baseline (No Spells) ---")
	fmt.Printf("Win Rate:    %.1f%%\n", baseResult.WinRate)
	fmt.Printf("Escape Rate: %.1f%%\n", baseResult.EscapeRate)
	fmt.Printf("Avg Rounds:  %.1f\n", baseResult.AvgRounds)
	fmt.Printf("Avg Damage Taken: %.1f\n", baseResult.AvgDamageTaken)
	fmt.Println()

	// Run all spell comparisons
	results := balance.RunSpellBalanceSweep(player, npc, *fleeThreshold, *iterations)

	// Sort by spell type then by win rate improvement
	sort.Slice(results, func(i, j int) bool {
		// Group stuns together, roots together
		spellsMap := balance.PredefinedSpells()
		var iType, jType string
		for _, s := range spellsMap {
			if s.Name == results[i].SpellName {
				iType = s.Type
			}
			if s.Name == results[j].SpellName {
				jType = s.Type
			}
		}
		if iType != jType {
			return iType < jType // "root" < "stun" alphabetically
		}
		return results[i].WinRateImprove > results[j].WinRateImprove
	})

	// Print stun results
	fmt.Println("=== STUN SPELLS (Prevent NPC Attacks) ===")
	fmt.Println()
	fmt.Println("Spell                        | Win Rate | Improve | Dmg Prevented | Casts/Fight")
	fmt.Println("-----------------------------+----------+---------+---------------+------------")
	for _, r := range results {
		spell := findSpell(r.SpellName)
		if spell.Type != "stun" {
			continue
		}
		fmt.Printf("%-28s | %6.1f%% | %+6.1f%% | %13.1f | %10.1f\n",
			r.SpellName, r.SpellWinRate, r.WinRateImprove, r.DamagePrevented, r.AvgSpellsCast)
	}
	fmt.Println()

	// Print root results
	fmt.Println("=== ROOT SPELLS (Prevent NPC Fleeing) ===")
	fmt.Println()
	fmt.Println("Spell                        | Win Rate | Improve | Escape Rate | Reduction | Casts/Fight")
	fmt.Println("-----------------------------+----------+---------+-------------+-----------+------------")
	for _, r := range results {
		spell := findSpell(r.SpellName)
		if spell.Type != "root" {
			continue
		}
		fmt.Printf("%-28s | %6.1f%% | %+6.1f%% | %9.1f%% | %+8.1f%% | %10.1f\n",
			r.SpellName, r.SpellWinRate, r.WinRateImprove, r.SpellEscapeRate, -r.EscapeReduction, r.AvgSpellsCast)
	}
	fmt.Println()

	// Assessment
	fmt.Println("=== Balance Assessment ===")
	fmt.Println()

	// Check for overpowered spells
	for _, r := range results {
		if r.WinRateImprove > 20 {
			fmt.Printf("WARNING: %s may be OVERPOWERED (+%.1f%% win rate)\n", r.SpellName, r.WinRateImprove)
		}
	}

	// Check for underpowered spells
	for _, r := range results {
		if r.WinRateImprove < 2 && r.EscapeReduction < 5 {
			fmt.Printf("WARNING: %s may be UNDERPOWERED (minimal impact)\n", r.SpellName)
		}
	}

	// Check if root spells are effective at preventing escapes
	for _, r := range results {
		spell := findSpell(r.SpellName)
		if spell.Type == "root" && r.EscapeReduction < baseResult.EscapeRate*0.5 {
			fmt.Printf("NOTE: %s only prevents %.0f%% of escapes (may need longer duration)\n",
				r.SpellName, (r.EscapeReduction/baseResult.EscapeRate)*100)
		}
	}

	// Overall assessment
	fmt.Println()
	if baseResult.EscapeRate > 30 {
		fmt.Println("HIGH ESCAPE RATE: Mobs flee often - root spells are valuable")
	} else if baseResult.EscapeRate > 10 {
		fmt.Println("MODERATE ESCAPE RATE: Root spells provide situational value")
	} else {
		fmt.Println("LOW ESCAPE RATE: Mobs rarely flee - root spells have limited value")
	}
}

func findSpell(name string) balance.SpellEffect {
	spells := balance.PredefinedSpells()
	for _, s := range spells {
		if s.Name == name {
			return s
		}
	}
	return balance.SpellEffect{}
}

func runLevelingSim() {
	fs := flag.NewFlagSet("leveling", flag.ExitOnError)

	// Level range
	startLevel := fs.Int("start-level", 1, "Starting player level")
	endLevel := fs.Int("end-level", 5, "Target player level")

	// Mob stats (Tier 1 defaults)
	mobHP := fs.Int("mob-hp", 17, "Average mob health")
	mobDmg := fs.Int("mob-damage", 4, "Average mob damage")
	mobArmor := fs.Int("mob-armor", 0, "Average mob armor")

	// Flee mechanics
	fleeThreshold := fs.Float64("flee-threshold", 0.13, "Average flee threshold (0.0-1.0)")
	fleeChance := fs.Float64("flee-chance", 0.35, "Flee chance per round when below threshold")

	// XP config
	xpPerKill := fs.Float64("xp-per-kill", 11.0, "Average XP per mob kill")
	travelTime := fs.Float64("travel-time", 30.0, "Seconds between fights (simple mode)")
	deathPenalty := fs.Float64("death-penalty", 120.0, "Seconds lost per death")

	// Floor config
	useFloorMechanics := fs.Bool("floor-mechanics", false, "Use realistic floor spawn/respawn mechanics")
	respawnTime := fs.Float64("respawn-time", 120.0, "Mob respawn time in seconds")
	roomTraversal := fs.Float64("room-traversal", 5.0, "Seconds to move between rooms")
	minRooms := fs.Int("min-rooms", 20, "Minimum rooms per floor")
	maxRooms := fs.Int("max-rooms", 50, "Maximum rooms per floor")
	playersOnline := fs.Int("players", 1, "Number of players online competing for mobs")
	floorsAvailable := fs.Int("floors-available", 5, "Number of floors players spread across")
	dynamicSpawns := fs.Bool("dynamic-spawns", false, "Scale spawn density with player count")
	targetMobsPerPlayer := fs.Float64("target-mobs", 30.0, "Target mobs per player with dynamic spawns")

	// Options
	detailed := fs.Bool("detailed", false, "Show detailed per-level breakdown")
	byMobType := fs.Bool("by-mob-type", false, "Show results by mob type")
	iterations := fs.Int("iterations", 10000, "Number of simulations per test")

	fs.Parse(os.Args[2:])

	xpConfig := balance.XPConfig{
		XPPerKill:       *xpPerKill,
		TravelTime:      *travelTime,
		SecsPerRound:    3.0,
		DeathPenaltySec: *deathPenalty,
	}

	floorConfig := balance.FloorConfig{
		MinRooms:            *minRooms,
		MaxRooms:            *maxRooms,
		RoomMobChance:       0.80,
		RoomMobMin:          1,
		RoomMobMax:          3,
		CorridorMobChance:   0.60,
		CorridorMobMin:      1,
		CorridorMobMax:      2,
		RoomToCorridorRatio: 0.6,
		RespawnTimeSec:      *respawnTime,
		RoomTraversalSec:    *roomTraversal,
		PlayersOnline:       *playersOnline,
		FloorsAvailable:     *floorsAvailable,
		DynamicSpawns:       *dynamicSpawns,
		TargetMobsPerPlayer: *targetMobsPerPlayer,
	}

	classes := balance.DefaultClasses()

	fmt.Println("=== Leveling Time Simulation ===")
	fmt.Println()
	fmt.Printf("Mob stats: %d HP, %d damage, %d armor\n", *mobHP, *mobDmg, *mobArmor)
	fmt.Printf("Flee mechanics: %.0f%% threshold, %.0f%% chance/round\n", *fleeThreshold*100, *fleeChance*100)
	if *useFloorMechanics {
		mobsPerFloor := balance.CalculateFloorMobs(floorConfig)
		mobsPerPlayer := balance.CalculateEffectiveMobsPerPlayer(floorConfig)
		fmt.Printf("Floor: %d-%d rooms, ~%.0f base mobs/floor, %.0fs respawn, %.0fs/room\n",
			*minRooms, *maxRooms, mobsPerFloor, *respawnTime, *roomTraversal)
		if *playersOnline > 1 {
			playersPerFloor := float64(*playersOnline) / float64(*floorsAvailable)
			if *dynamicSpawns {
				multiplier := balance.CalculateDynamicSpawnMultiplier(floorConfig)
				fmt.Printf("Dynamic spawns: %d players, %.1fx spawn multiplier, ~%.0f mobs/player\n",
					*playersOnline, multiplier, mobsPerPlayer)
			} else {
				fmt.Printf("Competition: %d players across %d floors (%.1f players/floor, ~%.0f mobs/player)\n",
					*playersOnline, *floorsAvailable, playersPerFloor, mobsPerPlayer)
			}
		}
	} else {
		fmt.Printf("XP: %.0f per kill, %.0f sec travel, %.0f sec death penalty\n",
			*xpPerKill, *travelTime, *deathPenalty)
	}
	fmt.Printf("Simulating levels %d -> %d\n", *startLevel, *endLevel)
	fmt.Println()

	if *byMobType && !*useFloorMechanics {
		// Show results by mob type (only in simple mode)
		mobTypes := balance.DefaultMobTypes()

		fmt.Println("=== Leveling by Mob Type (Level 1 -> 2) ===")
		fmt.Println()
		fmt.Println("Class    | Beast    | Humanoid | Giant    | Demon    | Undead   |")
		fmt.Println("---------|----------|----------|----------|----------|----------|")

		for _, class := range classes {
			results := balance.RunMobTypeLevelingSim(class, mobTypes, *mobHP, *mobDmg, *mobArmor,
				xpConfig, 1, *iterations)

			fmt.Printf("%-8s |", class.Name)
			for _, mobType := range mobTypes {
				if mobType.Name == "Construct" {
					continue // Skip construct (same as undead)
				}
				r := results[mobType.Name]
				fmt.Printf(" %5.0f min |", r.TotalTime)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if *useFloorMechanics {
		// Use floor-based simulation
		if *detailed {
			results := balance.RunDetailedFloorLevelingSimulation(classes, *mobHP, *mobDmg, *mobArmor,
				*fleeThreshold, *fleeChance, xpConfig, floorConfig, *startLevel, *endLevel, *iterations)

			fmt.Println("=== Detailed Leveling Breakdown (with Floor Mechanics) ===")
			fmt.Println()
			fmt.Println("Level  | Class    | Win%  | Escape% | Floors | Search | Respawn | Time    | CC")
			fmt.Println("-------|----------|-------|---------|--------|--------|---------|---------|----")

			currentLevel := 0
			for _, r := range results {
				if r.FromLevel != currentLevel {
					if currentLevel != 0 {
						fmt.Println()
					}
					currentLevel = r.FromLevel
				}

				cc := ""
				if r.HasRoot && r.HasStun {
					cc = "R+S"
				} else if r.HasRoot {
					cc = "R"
				} else if r.HasStun {
					cc = "S"
				} else {
					cc = "-"
				}

				fmt.Printf("%d -> %d | %-8s | %5.1f | %6.1f%% | %6.1f | %5.0fs | %6.0fs | %5.0f min | %s\n",
					r.FromLevel, r.ToLevel, r.ClassName, r.WinRate, r.EscapeRate,
					r.FloorsVisited, r.SearchTime, r.RespawnWaitTime, r.TotalTime, cc)
			}
			fmt.Println()
		}

		// Floor-based summary
		summaries := balance.RunFloorLevelingSimulation(classes, *mobHP, *mobDmg, *mobArmor,
			*fleeThreshold, *fleeChance, xpConfig, floorConfig, *startLevel, *endLevel, *iterations)

		fmt.Println("=== Time to Level Summary (with Floor Mechanics) ===")
		fmt.Println()

		// Build header
		header := "Class    |"
		divider := "---------|"
		for lvl := *startLevel; lvl < *endLevel; lvl++ {
			header += fmt.Sprintf(" %d->%d  |", lvl, lvl+1)
			divider += "-------|"
		}
		header += " TOTAL   | Floors | Search | Respawn"
		divider += "---------|--------|--------|--------"

		fmt.Println(header)
		fmt.Println(divider)

		for _, s := range summaries {
			line := fmt.Sprintf("%-8s |", s.ClassName)
			for _, t := range s.LevelTimes {
				line += fmt.Sprintf(" %5.0f |", t)
			}
			line += fmt.Sprintf(" %5.0f min | %5.1f | %5.0fs | %5.0fs",
				s.TotalTime, s.AvgFloorsPerLvl, s.AvgSearchTime, s.AvgRespawnWait)
			fmt.Println(line)
		}

		fmt.Println()
		fmt.Println("Legend: Floors = avg floors visited per level, Search = time finding mobs, Respawn = wait time")
	} else {
		// Simple mode
		if *detailed {
			results := balance.RunDetailedLevelingSimulation(classes, *mobHP, *mobDmg, *mobArmor,
				*fleeThreshold, *fleeChance, xpConfig, *startLevel, *endLevel, *iterations)

			fmt.Println("=== Detailed Leveling Breakdown ===")
			fmt.Println()
			fmt.Println("Level  | Class    | Win%  | Escape% | Death% | Kills | Time    | CC Available")
			fmt.Println("-------|----------|-------|---------|--------|-------|---------|-------------")

			currentLevel := 0
			for _, r := range results {
				if r.FromLevel != currentLevel {
					if currentLevel != 0 {
						fmt.Println()
					}
					currentLevel = r.FromLevel
				}

				cc := ""
				if r.HasRoot && r.HasStun {
					cc = "root+stun"
				} else if r.HasRoot {
					cc = "root"
				} else if r.HasStun {
					cc = "stun"
				} else {
					cc = "none"
				}

				fmt.Printf("%d -> %d | %-8s | %5.1f | %6.1f%% | %5.1f%% | %5.0f | %5.0f min | %s\n",
					r.FromLevel, r.ToLevel, r.ClassName, r.WinRate, r.EscapeRate, r.DeathRate,
					r.KillsNeeded, r.TotalTime, cc)
			}
			fmt.Println()
		}

		// Simple summary
		summaries := balance.RunLevelingSimulation(classes, *mobHP, *mobDmg, *mobArmor,
			*fleeThreshold, *fleeChance, xpConfig, *startLevel, *endLevel, *iterations)

		fmt.Println("=== Time to Level Summary ===")
		fmt.Println()

		// Build header
		header := "Class    |"
		divider := "---------|"
		for lvl := *startLevel; lvl < *endLevel; lvl++ {
			header += fmt.Sprintf(" %d->%d  |", lvl, lvl+1)
			divider += "-------|"
		}
		header += " TOTAL   | Avg Win% | Avg Esc%"
		divider += "---------|----------|----------"

		fmt.Println(header)
		fmt.Println(divider)

		for _, s := range summaries {
			line := fmt.Sprintf("%-8s |", s.ClassName)
			for _, t := range s.LevelTimes {
				line += fmt.Sprintf(" %5.0f |", t)
			}
			line += fmt.Sprintf(" %5.0f min | %6.1f%% | %6.1f%%", s.TotalTime, s.AvgWinRate, s.AvgEscapeRate)
			fmt.Println(line)
		}
	}

	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Times include travel between fights and death penalties")
	fmt.Println("  - Escape% shows mobs that fled (no XP gained)")
	fmt.Println("  - Classes gain root/stun abilities at different levels:")
	fmt.Println("    Ranger: root@2 | Rogue: stun@2, root@4 | Warrior: stun@3, root@5")
	fmt.Println("    Cleric: root@3 | Mage: root@3 | Paladin: stun@4, root@6")
	if *useFloorMechanics {
		fmt.Println()
		fmt.Println("Floor mechanics enabled:")
		fmt.Printf("  - ~%.0f mobs per floor, %.0fs respawn time\n",
			balance.CalculateFloorMobs(floorConfig), *respawnTime)
		fmt.Println("  - Search time increases as floor empties")
		fmt.Println("  - May wait for respawns or move to new floor")
	}
}
