package leveling

import "math"

// Leveling constants
const (
	MaxPlayerLevel = 50
	HPPerLevel     = 10
	ManaPerLevel   = 5
)

// XPForLevel returns the total XP required to reach a given level.
// Uses polynomial curve: 100 * level^1.5
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	return int(100 * math.Pow(float64(level), 1.5))
}

// XPToNextLevel returns XP needed from current level to next level.
func XPToNextLevel(currentLevel int) int {
	if currentLevel >= MaxPlayerLevel {
		return 0
	}
	return XPForLevel(currentLevel+1) - XPForLevel(currentLevel)
}

// LevelUpInfo contains information about a level-up event
type LevelUpInfo struct {
	NewLevel int
	HPGain   int
	ManaGain int
}
