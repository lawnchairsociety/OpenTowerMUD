package tower

import "testing"

func TestScaleHP(t *testing.T) {
	tests := []struct {
		baseHP, floor, want int
	}{
		{100, 0, 100},   // City - no scaling
		{100, 1, 110},   // 10% increase
		{100, 5, 150},   // 50% increase
		{100, 10, 200},  // 100% increase
		{50, 10, 100},   // Different base
	}

	for _, tc := range tests {
		got := ScaleHP(tc.baseHP, tc.floor)
		if got != tc.want {
			t.Errorf("ScaleHP(%d, %d) = %d, want %d", tc.baseHP, tc.floor, got, tc.want)
		}
	}
}

func TestScaleDamage(t *testing.T) {
	tests := []struct {
		baseDmg, floor, want int
	}{
		{10, 0, 10},   // City - no scaling
		{10, 1, 10},   // 8% increase = 10.8 -> 10
		{100, 5, 140}, // 40% increase
		{100, 10, 180},// 80% increase
	}

	for _, tc := range tests {
		got := ScaleDamage(tc.baseDmg, tc.floor)
		if got != tc.want {
			t.Errorf("ScaleDamage(%d, %d) = %d, want %d", tc.baseDmg, tc.floor, got, tc.want)
		}
	}
}

func TestScaleXP(t *testing.T) {
	tests := []struct {
		baseXP, floor, want int
	}{
		{100, 0, 100},   // City - no scaling
		{100, 1, 114},   // 15% increase = 115, but int truncation gives 114
		{100, 10, 250},  // 150% increase
	}

	for _, tc := range tests {
		got := ScaleXP(tc.baseXP, tc.floor)
		if got != tc.want {
			t.Errorf("ScaleXP(%d, %d) = %d, want %d", tc.baseXP, tc.floor, got, tc.want)
		}
	}
}

func TestScaleGold(t *testing.T) {
	tests := []struct {
		baseGold, floor, want int
	}{
		{100, 0, 100},   // City - no scaling
		{100, 1, 112},   // 12% increase
		{100, 10, 220},  // 120% increase
	}

	for _, tc := range tests {
		got := ScaleGold(tc.baseGold, tc.floor)
		if got != tc.want {
			t.Errorf("ScaleGold(%d, %d) = %d, want %d", tc.baseGold, tc.floor, got, tc.want)
		}
	}
}

func TestGetMobTier(t *testing.T) {
	// Test with default 25-floor tower scaling
	tests := []struct {
		floor, want int
	}{
		{0, 0},   // City - safe
		{1, 1},   // Easy
		{6, 1},   // Easy (last)
		{7, 2},   // Medium
		{12, 2},  // Medium (last)
		{13, 3},  // Hard
		{18, 3},  // Hard (last)
		{19, 4},  // Elite
		{25, 4},  // Elite (boss floor)
	}

	for _, tc := range tests {
		got := GetMobTier(tc.floor)
		if got != tc.want {
			t.Errorf("GetMobTier(%d) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestGetMobTierFor100FloorTower(t *testing.T) {
	// Test with 100-floor unified tower scaling
	// Unified tower is endgame content with higher tier mobs (4-7)
	tests := []struct {
		floor, want int
	}{
		{0, 0},   // Base - no hostile mobs
		{1, 4},   // Elite (unified entry level)
		{25, 4},  // Elite (last)
		{26, 5},  // Veteran
		{50, 5},  // Veteran (last)
		{51, 6},  // Champion
		{75, 6},  // Champion (last)
		{76, 7},  // Legendary (The Architect's domain)
		{100, 7}, // Legendary (boss floor - The Architect)
	}

	for _, tc := range tests {
		got := GetMobTierForFloor(tc.floor, 100)
		if got != tc.want {
			t.Errorf("GetMobTierForFloor(%d, 100) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestGetLootTier(t *testing.T) {
	// Test with default 25-floor tower scaling
	tests := []struct {
		floor, want int
	}{
		{0, 0},   // City - none
		{1, 1},   // Common
		{5, 1},   // Common (last)
		{6, 2},   // Uncommon
		{10, 2},  // Uncommon (last)
		{11, 3},  // Rare
		{18, 3},  // Rare (last)
		{19, 4},  // Epic
		{24, 4},  // Epic (last)
		{25, 5},  // Legendary (boss floor)
	}

	for _, tc := range tests {
		got := GetLootTier(tc.floor)
		if got != tc.want {
			t.Errorf("GetLootTier(%d) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestIsBossFloor(t *testing.T) {
	// Test with default 25-floor tower - only floor 25 is the boss floor
	tests := []struct {
		floor  int
		isBoss bool
	}{
		{0, false},
		{1, false},
		{10, false},  // Not a boss floor in 25-floor tower
		{24, false},
		{25, true},   // Final boss floor
		{26, false},  // Beyond max floors
	}

	for _, tc := range tests {
		got := IsBossFloor(tc.floor)
		if got != tc.isBoss {
			t.Errorf("IsBossFloor(%d) = %v, want %v", tc.floor, got, tc.isBoss)
		}
	}
}

func TestIsBossFloorFor100FloorTower(t *testing.T) {
	// Test with 100-floor unified tower - only floor 100 is the boss floor
	tests := []struct {
		floor  int
		isBoss bool
	}{
		{0, false},
		{10, false},
		{50, false},
		{99, false},
		{100, true},  // Final boss floor
		{101, false},
	}

	for _, tc := range tests {
		got := IsBossFloorForTower(tc.floor, 100)
		if got != tc.isBoss {
			t.Errorf("IsBossFloorForTower(%d, 100) = %v, want %v", tc.floor, got, tc.isBoss)
		}
	}
}
