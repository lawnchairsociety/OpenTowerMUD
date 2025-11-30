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
	tests := []struct {
		floor, want int
	}{
		{0, 0},   // City - safe
		{1, 1},   // Easy
		{5, 1},   // Easy
		{6, 2},   // Medium
		{10, 2},  // Medium
		{11, 3},  // Hard
		{20, 3},  // Hard
		{21, 4},  // Elite
		{50, 4},  // Elite
	}

	for _, tc := range tests {
		got := GetMobTier(tc.floor)
		if got != tc.want {
			t.Errorf("GetMobTier(%d) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestGetMobTierName(t *testing.T) {
	tests := []struct {
		tier int
		want string
	}{
		{0, "Safe"},
		{1, "Easy"},
		{2, "Medium"},
		{3, "Hard"},
		{4, "Elite"},
		{99, "Unknown"},
	}

	for _, tc := range tests {
		got := GetMobTierName(tc.tier)
		if got != tc.want {
			t.Errorf("GetMobTierName(%d) = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestGetLootTier(t *testing.T) {
	tests := []struct {
		floor, want int
	}{
		{0, 0},   // City - none
		{1, 1},   // Common
		{5, 1},   // Common
		{6, 2},   // Uncommon
		{10, 2},  // Uncommon
		{11, 3},  // Rare
		{20, 3},  // Rare
		{21, 4},  // Epic
		{30, 4},  // Epic
		{31, 5},  // Legendary
		{50, 5},  // Legendary
	}

	for _, tc := range tests {
		got := GetLootTier(tc.floor)
		if got != tc.want {
			t.Errorf("GetLootTier(%d) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestGetLootTierName(t *testing.T) {
	tests := []struct {
		tier int
		want string
	}{
		{0, "None"},
		{1, "Common"},
		{2, "Uncommon"},
		{3, "Rare"},
		{4, "Epic"},
		{5, "Legendary"},
		{99, "Unknown"},
	}

	for _, tc := range tests {
		got := GetLootTierName(tc.tier)
		if got != tc.want {
			t.Errorf("GetLootTierName(%d) = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestRecommendedLevel(t *testing.T) {
	tests := []struct {
		floor, want int
	}{
		{0, 1},   // City
		{1, 1},   // Floor 1
		{2, 2},   // Floor 2
		{10, 6},  // Floor 10
		{20, 11}, // Floor 20
	}

	for _, tc := range tests {
		got := RecommendedLevel(tc.floor)
		if got != tc.want {
			t.Errorf("RecommendedLevel(%d) = %d, want %d", tc.floor, got, tc.want)
		}
	}
}

func TestIsBossFloor(t *testing.T) {
	tests := []struct {
		floor  int
		isBoss bool
	}{
		{0, false},
		{1, false},
		{9, false},
		{10, true},
		{11, false},
		{20, true},
		{100, true},
	}

	for _, tc := range tests {
		got := IsBossFloor(tc.floor)
		if got != tc.isBoss {
			t.Errorf("IsBossFloor(%d) = %v, want %v", tc.floor, got, tc.isBoss)
		}
	}
}
