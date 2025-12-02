package class

import "fmt"

// Point buy constants
const (
	PointBuyTotal    = 27 // Total points to spend
	PointBuyMinScore = 8  // Minimum stat score
	PointBuyMaxScore = 15 // Maximum stat score (before racial bonuses)
)

// PointCost returns the point cost for a given ability score
// Based on D&D 5e point buy costs
func PointCost(score int) int {
	switch score {
	case 8:
		return 0
	case 9:
		return 1
	case 10:
		return 2
	case 11:
		return 3
	case 12:
		return 4
	case 13:
		return 5
	case 14:
		return 7
	case 15:
		return 9
	default:
		return -1 // Invalid score
	}
}

// PointBuyAllocation represents a complete stat allocation
type PointBuyAllocation struct {
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
}

// TotalCost calculates the total point cost of an allocation
func (a *PointBuyAllocation) TotalCost() int {
	return PointCost(a.Strength) +
		PointCost(a.Dexterity) +
		PointCost(a.Constitution) +
		PointCost(a.Intelligence) +
		PointCost(a.Wisdom) +
		PointCost(a.Charisma)
}

// IsValid returns true if the allocation is valid
func (a *PointBuyAllocation) IsValid() (bool, string) {
	// Check all scores are in valid range
	stats := map[string]int{
		"Strength":     a.Strength,
		"Dexterity":    a.Dexterity,
		"Constitution": a.Constitution,
		"Intelligence": a.Intelligence,
		"Wisdom":       a.Wisdom,
		"Charisma":     a.Charisma,
	}

	for name, score := range stats {
		if score < PointBuyMinScore {
			return false, fmt.Sprintf("%s cannot be below %d", name, PointBuyMinScore)
		}
		if score > PointBuyMaxScore {
			return false, fmt.Sprintf("%s cannot be above %d", name, PointBuyMaxScore)
		}
		if PointCost(score) < 0 {
			return false, fmt.Sprintf("%s has invalid value %d", name, score)
		}
	}

	// Check total cost
	cost := a.TotalCost()
	if cost > PointBuyTotal {
		return false, fmt.Sprintf("allocation costs %d points, but only %d available", cost, PointBuyTotal)
	}

	return true, ""
}

// RemainingPoints returns how many points are left to spend
func (a *PointBuyAllocation) RemainingPoints() int {
	return PointBuyTotal - a.TotalCost()
}

// DefaultAllocation returns the default allocation (all 10s)
func DefaultAllocation() *PointBuyAllocation {
	return &PointBuyAllocation{
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
	}
}

// RecommendedAllocation returns the recommended stat allocation for a class
func RecommendedAllocation(c Class) *PointBuyAllocation {
	switch c {
	case Warrior:
		return &PointBuyAllocation{
			Strength:     15,
			Dexterity:    12,
			Constitution: 14,
			Intelligence: 8,
			Wisdom:       10,
			Charisma:     8,
		}
	case Mage:
		return &PointBuyAllocation{
			Strength:     8,
			Dexterity:    12,
			Constitution: 14,
			Intelligence: 15,
			Wisdom:       13,
			Charisma:     8,
		}
	case Cleric:
		return &PointBuyAllocation{
			Strength:     10,
			Dexterity:    10,
			Constitution: 14,
			Intelligence: 8,
			Wisdom:       15,
			Charisma:     10,
		}
	case Rogue:
		return &PointBuyAllocation{
			Strength:     8,
			Dexterity:    15,
			Constitution: 12,
			Intelligence: 14,
			Wisdom:       10,
			Charisma:     8,
		}
	case Ranger:
		return &PointBuyAllocation{
			Strength:     10,
			Dexterity:    15,
			Constitution: 13,
			Intelligence: 8,
			Wisdom:       13,
			Charisma:     8,
		}
	case Paladin:
		return &PointBuyAllocation{
			Strength:     15,
			Dexterity:    8,
			Constitution: 12,
			Intelligence: 8,
			Wisdom:       10,
			Charisma:     14,
		}
	default:
		return DefaultAllocation()
	}
}

// PointCostTable returns a formatted string showing the point cost table
func PointCostTable() string {
	return `Score | Cost | Modifier
------+------+---------
   8  |   0  |    -1
   9  |   1  |    -1
  10  |   2  |    +0
  11  |   3  |    +0
  12  |   4  |    +1
  13  |   5  |    +1
  14  |   7  |    +2
  15  |   9  |    +2`
}
