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

// GetMobTier returns the mob tier for a given floor (assumes 25-floor tower)
// Used to determine which mobs can spawn on each floor
func GetMobTier(floor int) int {
	return GetMobTierForFloor(floor, 25)
}

// GetMobTierForFloor returns the mob tier based on floor and tower max floors.
// For 25-floor towers, tiers are compressed (1-4). For 100-floor unified tower, starts at tier 4.
func GetMobTierForFloor(floor int, maxFloors int) int {
	if floor <= 0 {
		return 0 // City - no hostile mobs
	}

	if maxFloors <= 25 {
		// Compressed tiers for 25-floor racial towers
		switch {
		case floor <= 6:
			return 1 // Easy mobs
		case floor <= 12:
			return 2 // Medium mobs
		case floor <= 18:
			return 3 // Hard mobs
		default:
			return 4 // Elite mobs
		}
	}

	// Unified tower (100 floors) - endgame content starts at tier 4
	switch {
	case floor <= 25:
		return 4 // Elite mobs (unified entry level)
	case floor <= 50:
		return 5 // Veteran mobs
	case floor <= 75:
		return 6 // Champion mobs
	default:
		return 7 // Legendary mobs (The Blighted One's domain)
	}
}

// GetLootTier returns the loot tier for a given floor (assumes 25-floor tower)
// Higher tiers have better loot drops
func GetLootTier(floor int) int {
	return GetLootTierForFloor(floor, 25)
}

// GetLootTierForFloor returns the loot tier based on floor and tower max floors.
func GetLootTierForFloor(floor int, maxFloors int) int {
	if floor <= 0 {
		return 0 // City - no loot
	}

	if maxFloors <= 25 {
		// Compressed tiers for 25-floor racial towers
		switch {
		case floor <= 5:
			return 1 // Common loot
		case floor <= 10:
			return 2 // Uncommon loot
		case floor <= 18:
			return 3 // Rare loot
		case floor <= 24:
			return 4 // Epic loot
		default:
			return 5 // Legendary loot (final boss)
		}
	}

	// Unified tower (100 floors) - better loot throughout
	switch {
	case floor <= 25:
		return 3 // Rare loot (unified entry level)
	case floor <= 50:
		return 4 // Epic loot
	case floor <= 75:
		return 5 // Legendary loot
	default:
		return 5 // Legendary loot (The Blighted One's domain)
	}
}

// IsBossFloor returns true if the floor number is the boss floor for the tower.
// For single-tower compatibility, defaults to checking if it's a 25-floor boss floor.
func IsBossFloor(floor int) bool {
	return IsBossFloorForTower(floor, 25)
}

// IsBossFloorForTower returns true if this floor is the final boss floor.
func IsBossFloorForTower(floor int, maxFloors int) bool {
	return floor > 0 && floor == maxFloors
}
