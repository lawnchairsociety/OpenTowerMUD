package stats

import (
	"math/rand"
	"regexp"
	"strconv"
)

// D20 rolls a 20-sided die (1-20)
func D20() int {
	return rand.Intn(20) + 1
}

// D12 rolls a 12-sided die (1-12)
func D12() int {
	return rand.Intn(12) + 1
}

// D10 rolls a 10-sided die (1-10)
func D10() int {
	return rand.Intn(10) + 1
}

// D8 rolls an 8-sided die (1-8)
func D8() int {
	return rand.Intn(8) + 1
}

// D6 rolls a 6-sided die (1-6)
func D6() int {
	return rand.Intn(6) + 1
}

// D4 rolls a 4-sided die (1-4)
func D4() int {
	return rand.Intn(4) + 1
}

// D100 rolls a 100-sided die (1-100), used for percentage checks
func D100() int {
	return rand.Intn(100) + 1
}

// Roll rolls n dice with the specified number of sides and returns the total
func Roll(n, sides int) int {
	total := 0
	for i := 0; i < n; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}

// RollWithBonus rolls n dice with the specified number of sides and adds a bonus
func RollWithBonus(n, sides, bonus int) int {
	return Roll(n, sides) + bonus
}

// diceNotationRegex matches dice notation like "1d6", "2d4+1", "1d8-2"
var diceNotationRegex = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)

// ParseDice parses dice notation and returns the roll result
// Supports formats: "1d6", "2d4", "1d8+2", "2d6-1"
// Returns 0 if the notation is invalid
func ParseDice(notation string) int {
	if notation == "" {
		return 0
	}

	matches := diceNotationRegex.FindStringSubmatch(notation)
	if matches == nil {
		return 0
	}

	count, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])

	bonus := 0
	if matches[3] != "" {
		bonus, _ = strconv.Atoi(matches[3])
	}

	return RollWithBonus(count, sides, bonus)
}

// ParseDiceWithBonus parses dice notation and adds an extra bonus (e.g., STR modifier)
// Returns 0 if the notation is invalid
func ParseDiceWithBonus(notation string, extraBonus int) int {
	if notation == "" {
		return 0
	}

	matches := diceNotationRegex.FindStringSubmatch(notation)
	if matches == nil {
		return 0
	}

	count, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])

	bonus := extraBonus
	if matches[3] != "" {
		notationBonus, _ := strconv.Atoi(matches[3])
		bonus += notationBonus
	}

	return RollWithBonus(count, sides, bonus)
}
