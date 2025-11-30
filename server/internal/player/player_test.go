package player

import (
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
)

func TestXPForLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{1, 0},       // Level 1 requires 0 XP
		{2, 282},     // 100 * 2^1.5 = 282
		{3, 519},     // 100 * 3^1.5 = 519
		{5, 1118},    // 100 * 5^1.5 = 1118
		{10, 3162},   // 100 * 10^1.5 = 3162
		{20, 8944},   // 100 * 20^1.5 = 8944
		{50, 35355},  // 100 * 50^1.5 = 35355
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := leveling.XPForLevel(tt.level)
			if result != tt.expected {
				t.Errorf("XPForLevel(%d) = %d, want %d", tt.level, result, tt.expected)
			}
		})
	}
}

func TestXPForLevel_EdgeCases(t *testing.T) {
	// Level 0 or below should return 0
	if leveling.XPForLevel(0) != 0 {
		t.Errorf("XPForLevel(0) should be 0, got %d", leveling.XPForLevel(0))
	}
	if leveling.XPForLevel(-1) != 0 {
		t.Errorf("XPForLevel(-1) should be 0, got %d", leveling.XPForLevel(-1))
	}
}

func TestXPToNextLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{1, 282},  // XP from level 1 to 2: 282 - 0 = 282
		{2, 237},  // XP from level 2 to 3: 519 - 282 = 237
		{10, 486}, // XP from level 10 to 11
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := leveling.XPToNextLevel(tt.level)
			if result != tt.expected {
				t.Errorf("XPToNextLevel(%d) = %d, want %d", tt.level, result, tt.expected)
			}
		})
	}
}

func TestXPToNextLevel_AtMaxLevel(t *testing.T) {
	// At max level, XP to next should be 0
	result := leveling.XPToNextLevel(leveling.MaxPlayerLevel)
	if result != 0 {
		t.Errorf("XPToNextLevel(%d) should be 0 at max level, got %d", leveling.MaxPlayerLevel, result)
	}
}

func TestGainExperience_SingleLevelUp(t *testing.T) {
	p := &Player{
		Level:      1,
		Experience: 0,
		Health:     50,
		MaxHealth:  100,
		Mana:       30,
		MaxMana:    100,
	}

	// Give enough XP to reach level 2 (283 XP needed)
	levelUps := p.GainExperience(300)

	if len(levelUps) != 1 {
		t.Errorf("Expected 1 level up, got %d", len(levelUps))
	}
	if p.Level != 2 {
		t.Errorf("Expected level 2, got %d", p.Level)
	}
	if p.Experience != 300 {
		t.Errorf("Expected 300 XP, got %d", p.Experience)
	}
	// Should have gained HP and Mana
	if p.MaxHealth != 110 {
		t.Errorf("Expected MaxHealth 110, got %d", p.MaxHealth)
	}
	if p.MaxMana != 105 {
		t.Errorf("Expected MaxMana 105, got %d", p.MaxMana)
	}
	// Should be fully restored
	if p.Health != p.MaxHealth {
		t.Errorf("Expected full health %d, got %d", p.MaxHealth, p.Health)
	}
	if p.Mana != p.MaxMana {
		t.Errorf("Expected full mana %d, got %d", p.MaxMana, p.Mana)
	}
}

func TestGainExperience_MultipleLevelUps(t *testing.T) {
	p := &Player{
		Level:      1,
		Experience: 0,
		Health:     100,
		MaxHealth:  100,
		Mana:       100,
		MaxMana:    100,
	}

	// Give enough XP to reach level 5 (1118 XP needed)
	levelUps := p.GainExperience(1200)

	if len(levelUps) != 4 {
		t.Errorf("Expected 4 level ups (1->5), got %d", len(levelUps))
	}
	if p.Level != 5 {
		t.Errorf("Expected level 5, got %d", p.Level)
	}
	// 4 levels * 10 HP = 40 HP gain
	if p.MaxHealth != 140 {
		t.Errorf("Expected MaxHealth 140, got %d", p.MaxHealth)
	}
	// 4 levels * 5 Mana = 20 Mana gain
	if p.MaxMana != 120 {
		t.Errorf("Expected MaxMana 120, got %d", p.MaxMana)
	}
}

func TestGainExperience_NoLevelUp(t *testing.T) {
	p := &Player{
		Level:      1,
		Experience: 0,
		Health:     100,
		MaxHealth:  100,
		Mana:       100,
		MaxMana:    100,
	}

	// Give XP that's not enough to level up (need 283 for level 2)
	levelUps := p.GainExperience(100)

	if len(levelUps) != 0 {
		t.Errorf("Expected 0 level ups, got %d", len(levelUps))
	}
	if p.Level != 1 {
		t.Errorf("Expected level 1, got %d", p.Level)
	}
	if p.Experience != 100 {
		t.Errorf("Expected 100 XP, got %d", p.Experience)
	}
	// Stats should not change
	if p.MaxHealth != 100 {
		t.Errorf("Expected MaxHealth 100, got %d", p.MaxHealth)
	}
}

func TestGainExperience_MaxLevelCap(t *testing.T) {
	p := &Player{
		Level:      leveling.MaxPlayerLevel,
		Experience: leveling.XPForLevel(leveling.MaxPlayerLevel),
		Health:     100,
		MaxHealth:  100,
		Mana:       100,
		MaxMana:    100,
	}

	// Try to gain more XP at max level
	levelUps := p.GainExperience(10000)

	if len(levelUps) != 0 {
		t.Errorf("Expected 0 level ups at max level, got %d", len(levelUps))
	}
	if p.Level != leveling.MaxPlayerLevel {
		t.Errorf("Expected level %d (max), got %d", leveling.MaxPlayerLevel, p.Level)
	}
	// XP should still accumulate
	expectedXP := leveling.XPForLevel(leveling.MaxPlayerLevel) + 10000
	if p.Experience != expectedXP {
		t.Errorf("Expected %d XP, got %d", expectedXP, p.Experience)
	}
}

func TestStatGrowth(t *testing.T) {
	// Verify HP and Mana growth constants are correct
	if leveling.HPPerLevel != 10 {
		t.Errorf("Expected HPPerLevel = 10, got %d", leveling.HPPerLevel)
	}
	if leveling.ManaPerLevel != 5 {
		t.Errorf("Expected ManaPerLevel = 5, got %d", leveling.ManaPerLevel)
	}

	// Verify a level 50 character has expected stats
	// Base: 100 HP, 100 Mana
	// Level 50: 49 level-ups * 10 HP = 490 HP, 49 * 5 Mana = 245 Mana
	p := &Player{
		Level:     1,
		MaxHealth: 100,
		MaxMana:   100,
		Health:    100,
		Mana:      100,
	}

	// Simulate leveling to 50
	for i := 1; i < leveling.MaxPlayerLevel; i++ {
		p.levelUp()
	}

	expectedHP := 100 + (leveling.MaxPlayerLevel-1)*leveling.HPPerLevel     // 100 + 49*10 = 590
	expectedMana := 100 + (leveling.MaxPlayerLevel-1)*leveling.ManaPerLevel // 100 + 49*5 = 345

	if p.MaxHealth != expectedHP {
		t.Errorf("Expected MaxHealth at level 50 = %d, got %d", expectedHP, p.MaxHealth)
	}
	if p.MaxMana != expectedMana {
		t.Errorf("Expected MaxMana at level 50 = %d, got %d", expectedMana, p.MaxMana)
	}
}

func TestLevelUpInfo(t *testing.T) {
	p := &Player{
		Level:      1,
		Experience: 0,
		Health:     50,
		MaxHealth:  100,
		Mana:       30,
		MaxMana:    100,
	}

	levelUps := p.GainExperience(300)

	if len(levelUps) != 1 {
		t.Fatalf("Expected 1 level up, got %d", len(levelUps))
	}

	lu := levelUps[0]
	if lu.NewLevel != 2 {
		t.Errorf("Expected NewLevel 2, got %d", lu.NewLevel)
	}
	if lu.HPGain != leveling.HPPerLevel {
		t.Errorf("Expected HPGain %d, got %d", leveling.HPPerLevel, lu.HPGain)
	}
	if lu.ManaGain != leveling.ManaPerLevel {
		t.Errorf("Expected ManaGain %d, got %d", leveling.ManaPerLevel, lu.ManaGain)
	}
}

// ==================== Portal Discovery Tests ====================

func TestDiscoverPortal_GroundFloorAlwaysAvailable(t *testing.T) {
	p := &Player{}

	// Ground floor (0) should always be available even without discovering
	if !p.HasDiscoveredPortal(0) {
		t.Error("Ground floor (0) should always be available")
	}
}

func TestDiscoverPortal_NewFloor(t *testing.T) {
	p := &Player{}

	// Floor 5 should not be discovered initially
	if p.HasDiscoveredPortal(5) {
		t.Error("Floor 5 should not be discovered initially")
	}

	// Discover floor 5
	p.DiscoverPortal(5)

	if !p.HasDiscoveredPortal(5) {
		t.Error("Floor 5 should be discovered after DiscoverPortal(5)")
	}
}

func TestDiscoverPortal_MultipleFLoors(t *testing.T) {
	p := &Player{}

	// Discover multiple floors in random order
	p.DiscoverPortal(3)
	p.DiscoverPortal(7)
	p.DiscoverPortal(1)
	p.DiscoverPortal(10)

	// All should be discovered (plus ground floor 0)
	for _, floor := range []int{0, 1, 3, 7, 10} {
		if !p.HasDiscoveredPortal(floor) {
			t.Errorf("Floor %d should be discovered", floor)
		}
	}

	// Floor 5 should NOT be discovered
	if p.HasDiscoveredPortal(5) {
		t.Error("Floor 5 should NOT be discovered")
	}
}

func TestGetDiscoveredPortals_Sorted(t *testing.T) {
	p := &Player{}

	// Discover floors in random order
	p.DiscoverPortal(7)
	p.DiscoverPortal(3)
	p.DiscoverPortal(10)
	p.DiscoverPortal(1)

	floors := p.GetDiscoveredPortals()

	// Should be sorted: 0, 1, 3, 7, 10
	expected := []int{0, 1, 3, 7, 10}
	if len(floors) != len(expected) {
		t.Fatalf("Expected %d floors, got %d", len(expected), len(floors))
	}
	for i, floor := range floors {
		if floor != expected[i] {
			t.Errorf("Expected floor %d at position %d, got %d", expected[i], i, floor)
		}
	}
}

func TestGetVisitedPortalsString(t *testing.T) {
	p := &Player{}

	// Initial state: only ground floor
	str := p.GetVisitedPortalsString()
	if str != "0" {
		t.Errorf("Expected '0', got '%s'", str)
	}

	// Add more floors
	p.DiscoverPortal(5)
	p.DiscoverPortal(3)

	str = p.GetVisitedPortalsString()
	if str != "0,3,5" {
		t.Errorf("Expected '0,3,5', got '%s'", str)
	}
}

func TestSetVisitedPortals(t *testing.T) {
	p := &Player{}

	// Set visited portals from a list (simulating database load)
	p.SetVisitedPortals([]int{0, 2, 5, 10})

	// All should be discovered
	for _, floor := range []int{0, 2, 5, 10} {
		if !p.HasDiscoveredPortal(floor) {
			t.Errorf("Floor %d should be discovered after SetVisitedPortals", floor)
		}
	}

	// Floor 3 should NOT be discovered
	if p.HasDiscoveredPortal(3) {
		t.Error("Floor 3 should NOT be discovered")
	}
}

func TestSetVisitedPortals_AlwaysIncludesGroundFloor(t *testing.T) {
	p := &Player{}

	// Set visited portals without ground floor
	p.SetVisitedPortals([]int{5, 10})

	// Ground floor should still be available
	if !p.HasDiscoveredPortal(0) {
		t.Error("Ground floor (0) should always be available")
	}
}

func TestDiscoverPortal_Idempotent(t *testing.T) {
	p := &Player{}

	// Discover floor 5 multiple times
	p.DiscoverPortal(5)
	p.DiscoverPortal(5)
	p.DiscoverPortal(5)

	// Should only appear once in the list
	floors := p.GetDiscoveredPortals()
	count := 0
	for _, f := range floors {
		if f == 5 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Floor 5 should appear exactly once, appeared %d times", count)
	}
}
