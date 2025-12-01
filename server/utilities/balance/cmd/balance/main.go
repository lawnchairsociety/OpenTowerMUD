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
//	sweep      - Run a comprehensive balance sweep
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
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
  sweep     Run a comprehensive balance sweep

Examples:
  balance combat -player-hp=100 -player-str=12 -player-weapon="1d6" -npc-hp=50 -npc-damage=8 -npc-armor=2
  balance floors -player-level=5 -player-hp=150 -iterations=10000
  balance levels -npc-hp=80 -npc-damage=10 -npc-armor=2
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
