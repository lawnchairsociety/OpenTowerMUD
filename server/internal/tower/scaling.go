package tower

// Scaling contains difficulty scaling formulas for tower floors

// ScaleHP calculates scaled HP for a mob on a given floor
// Formula: base_hp * (1 + floor * 0.1)
func ScaleHP(baseHP, floor int) int {
	if floor <= 0 {
		return baseHP
	}
	multiplier := 1.0 + float64(floor)*0.1
	return int(float64(baseHP) * multiplier)
}

// ScaleDamage calculates scaled damage for a mob on a given floor
// Formula: base_damage * (1 + floor * 0.08)
func ScaleDamage(baseDamage, floor int) int {
	if floor <= 0 {
		return baseDamage
	}
	multiplier := 1.0 + float64(floor)*0.08
	return int(float64(baseDamage) * multiplier)
}

// ScaleXP calculates scaled XP reward for a mob on a given floor
// Formula: base_xp * (1 + floor * 0.15)
func ScaleXP(baseXP, floor int) int {
	if floor <= 0 {
		return baseXP
	}
	multiplier := 1.0 + float64(floor)*0.15
	return int(float64(baseXP) * multiplier)
}

// ScaleGold calculates scaled gold drop for a mob on a given floor
// Formula: base_gold * (1 + floor * 0.12)
func ScaleGold(baseGold, floor int) int {
	if floor <= 0 {
		return baseGold
	}
	multiplier := 1.0 + float64(floor)*0.12
	return int(float64(baseGold) * multiplier)
}

// GetMobTier returns the mob tier for a given floor
// Used to determine which mobs can spawn on each floor
func GetMobTier(floor int) int {
	switch {
	case floor <= 0:
		return 0 // City - no hostile mobs
	case floor <= 5:
		return 1 // Easy mobs (rats, goblins)
	case floor <= 10:
		return 2 // Medium mobs (orcs, spiders)
	case floor <= 20:
		return 3 // Hard mobs (knights, trolls)
	default:
		return 4 // Elite mobs
	}
}

// GetMobTierName returns a human-readable name for the mob tier
func GetMobTierName(tier int) string {
	switch tier {
	case 0:
		return "Safe"
	case 1:
		return "Easy"
	case 2:
		return "Medium"
	case 3:
		return "Hard"
	case 4:
		return "Elite"
	default:
		return "Unknown"
	}
}

// GetLootTier returns the loot tier for a given floor
// Higher tiers have better loot drops
func GetLootTier(floor int) int {
	switch {
	case floor <= 0:
		return 0 // City - no loot
	case floor <= 5:
		return 1 // Common loot
	case floor <= 10:
		return 2 // Uncommon loot
	case floor <= 20:
		return 3 // Rare loot
	case floor <= 30:
		return 4 // Epic loot
	default:
		return 5 // Legendary loot
	}
}

// GetLootTierName returns a human-readable name for the loot tier
func GetLootTierName(tier int) string {
	switch tier {
	case 0:
		return "None"
	case 1:
		return "Common"
	case 2:
		return "Uncommon"
	case 3:
		return "Rare"
	case 4:
		return "Epic"
	case 5:
		return "Legendary"
	default:
		return "Unknown"
	}
}

// RecommendedLevel returns the recommended player level for a floor
func RecommendedLevel(floor int) int {
	if floor <= 0 {
		return 1
	}
	// Roughly 1 level per 2 floors
	return 1 + (floor / 2)
}

// IsBossFloor returns true if the floor number is a boss floor
func IsBossFloor(floor int) bool {
	return floor > 0 && floor%10 == 0
}
