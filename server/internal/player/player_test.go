package player

import (
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
)

// createTestPlayer creates a player with proper class initialization for testing
func createTestPlayer() *Player {
	p := &Player{
		Level:        1,
		Experience:   0,
		Health:       10,
		MaxHealth:    10,
		Mana:         0,
		MaxMana:      0,
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
		classLevels:  class.NewClassLevels(class.Warrior),
		activeClass:  class.Warrior,
	}
	return p
}

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
	p := createTestPlayer()
	p.Health = 5 // Damaged

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
	// Warrior: d10 hit die, average = 6, CON 10 = +0 mod
	// So HP gain = 6 per level, starting at 10, now 16
	if p.MaxHealth != 16 {
		t.Errorf("Expected MaxHealth 16 (10 base + 6 from level), got %d", p.MaxHealth)
	}
	// Warrior has 0 mana
	if p.MaxMana != 0 {
		t.Errorf("Expected MaxMana 0 (warrior), got %d", p.MaxMana)
	}
	// Should be fully restored
	if p.Health != p.MaxHealth {
		t.Errorf("Expected full health %d, got %d", p.MaxHealth, p.Health)
	}
}

func TestGainExperience_MultipleLevelUps(t *testing.T) {
	p := createTestPlayer()

	// Give enough XP to reach level 5 (1118 XP needed)
	levelUps := p.GainExperience(1200)

	if len(levelUps) != 4 {
		t.Errorf("Expected 4 level ups (1->5), got %d", len(levelUps))
	}
	if p.Level != 5 {
		t.Errorf("Expected level 5, got %d", p.Level)
	}
	// Warrior: 4 levels * 6 HP = 24 HP gain, starting at 10
	if p.MaxHealth != 34 {
		t.Errorf("Expected MaxHealth 34 (10 base + 4*6), got %d", p.MaxHealth)
	}
	// Warrior has 0 mana
	if p.MaxMana != 0 {
		t.Errorf("Expected MaxMana 0 (warrior), got %d", p.MaxMana)
	}
}

func TestGainExperience_NoLevelUp(t *testing.T) {
	p := createTestPlayer()

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
	if p.MaxHealth != 10 {
		t.Errorf("Expected MaxHealth 10, got %d", p.MaxHealth)
	}
}

func TestGainExperience_MaxLevelCap(t *testing.T) {
	p := createTestPlayer()
	p.Level = leveling.MaxPlayerLevel
	p.Experience = leveling.XPForLevel(leveling.MaxPlayerLevel)

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
	// With class system, a warrior gets d10 hit die (avg 6) per level
	// and 0 mana per level

	p := createTestPlayer()

	// Simulate leveling to 50
	for i := 1; i < leveling.MaxPlayerLevel; i++ {
		p.levelUp()
	}

	// Warrior: 10 base HP + 49 levels * 6 HP = 10 + 294 = 304
	// Plus +10% HP bonus at level 20:
	// At level 20: 10 + 19*6 = 124 HP, 10% bonus = 12 HP
	// Total: 304 + 12 = 316 HP
	baseHP := 10 + (leveling.MaxPlayerLevel-1)*6 // 304
	hpAt20 := 10 + 19*6                          // 124 HP at level 20
	level20Bonus := hpAt20 / 10                  // 10% bonus = 12
	expectedHP := baseHP + level20Bonus          // 316

	if p.MaxHealth != expectedHP {
		t.Errorf("Expected MaxHealth at level 50 = %d, got %d", expectedHP, p.MaxHealth)
	}
	// Warrior has 0 mana
	if p.MaxMana != 0 {
		t.Errorf("Expected MaxMana at level 50 = 0 (warrior), got %d", p.MaxMana)
	}
}

func TestLevelUpInfo(t *testing.T) {
	p := createTestPlayer()
	p.Health = 5 // Damaged

	levelUps := p.GainExperience(300)

	if len(levelUps) != 1 {
		t.Fatalf("Expected 1 level up, got %d", len(levelUps))
	}

	lu := levelUps[0]
	if lu.NewLevel != 2 {
		t.Errorf("Expected NewLevel 2, got %d", lu.NewLevel)
	}
	// Warrior: d10 avg = 6, CON 10 = +0 mod
	if lu.HPGain != 6 {
		t.Errorf("Expected HPGain 6 (warrior d10 avg), got %d", lu.HPGain)
	}
	// Warrior has 0 mana gain
	if lu.ManaGain != 0 {
		t.Errorf("Expected ManaGain 0 (warrior), got %d", lu.ManaGain)
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

func TestGetDiscoveredPortalsString(t *testing.T) {
	p := &Player{}

	// Initial state: only ground floor
	str := p.GetDiscoveredPortalsString()
	if str != "0" {
		t.Errorf("Expected '0', got '%s'", str)
	}

	// Add more floors
	p.DiscoverPortal(5)
	p.DiscoverPortal(3)

	str = p.GetDiscoveredPortalsString()
	if str != "0,3,5" {
		t.Errorf("Expected '0,3,5', got '%s'", str)
	}
}

func TestSetDiscoveredPortals(t *testing.T) {
	p := &Player{}

	// Set discovered portals from a list (simulating database load)
	p.SetDiscoveredPortals([]int{0, 2, 5, 10})

	// All should be discovered
	for _, floor := range []int{0, 2, 5, 10} {
		if !p.HasDiscoveredPortal(floor) {
			t.Errorf("Floor %d should be discovered after SetDiscoveredPortals", floor)
		}
	}

	// Floor 3 should NOT be discovered
	if p.HasDiscoveredPortal(3) {
		t.Error("Floor 3 should NOT be discovered")
	}
}

func TestSetDiscoveredPortals_AlwaysIncludesGroundFloor(t *testing.T) {
	p := &Player{}

	// Set discovered portals without ground floor
	p.SetDiscoveredPortals([]int{5, 10})

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

// ==================== Race Tests ====================

func TestPlayer_GetRace_Default(t *testing.T) {
	p := &Player{}
	// Default race should be empty (zero value)
	if p.GetRace() != "" {
		t.Errorf("Expected empty default race, got %q", p.GetRace())
	}
}

func TestPlayer_SetRace(t *testing.T) {
	p := &Player{}
	p.SetRace(race.Dwarf)
	if p.GetRace() != race.Dwarf {
		t.Errorf("Expected race 'dwarf', got %q", p.GetRace())
	}
}

func TestPlayer_GetRaceName(t *testing.T) {
	tests := []struct {
		raceVal  race.Race
		expected string
	}{
		{race.Human, "Human"},
		{race.Dwarf, "Dwarf"},
		{race.Elf, "Elf"},
		{race.Halfling, "Halfling"},
		{race.Gnome, "Gnome"},
		{race.HalfElf, "Half-Elf"},
		{race.HalfOrc, "Half-Orc"},
		{race.Race("invalid"), "Unknown"},
		{race.Race(""), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.raceVal), func(t *testing.T) {
			p := &Player{}
			p.SetRace(tt.raceVal)
			if got := p.GetRaceName(); got != tt.expected {
				t.Errorf("GetRaceName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
