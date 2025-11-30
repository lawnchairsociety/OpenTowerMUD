package tower

import (
	"math/rand"
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func createTestMobConfig() *npc.NPCsConfig {
	return &npc.NPCsConfig{
		NPCs: map[string]npc.NPCDefinition{
			"goblin": {
				Name:        "goblin",
				Description: "A small goblin",
				Level:       1,
				Health:      20,
				Damage:      3,
				Armor:       0,
				Experience:  10,
				Aggressive:  true,
				Attackable:  true,
				Tier:        1,
				LootTable:   []npc.LootEntryYAML{{Item: "coin", Chance: 50}},
			},
			"orc": {
				Name:        "orc",
				Description: "A brutish orc",
				Level:       3,
				Health:      40,
				Damage:      8,
				Armor:       2,
				Experience:  30,
				Aggressive:  true,
				Attackable:  true,
				Tier:        2,
				LootTable:   []npc.LootEntryYAML{{Item: "sword", Chance: 25}},
			},
			"goblin_king": {
				Name:        "Goblin King",
				Description: "The goblin ruler",
				Level:       5,
				Health:      150,
				Damage:      12,
				Armor:       3,
				Experience:  150,
				Aggressive:  true,
				Attackable:  true,
				Tier:        1,
				Boss:        true,
				LootTable:   []npc.LootEntryYAML{{Item: "crown", Chance: 100}},
			},
		},
	}
}

func TestNewMobSpawner(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	if spawner == nil {
		t.Fatal("NewMobSpawner returned nil")
	}
	if spawner.mobConfig != config {
		t.Error("mobConfig not set correctly")
	}
}

func TestSpawnMobsOnFloor_NoSpawnerConfig(t *testing.T) {
	spawner := NewMobSpawner(nil)
	floor := NewFloor(1)
	rng := rand.New(rand.NewSource(42))

	result := spawner.SpawnMobsOnFloor(floor, 1, rng)
	if result != nil {
		t.Error("Should return nil when no config")
	}
}

func TestSpawnMobsOnFloor_CityFloor(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)
	floor := NewFloor(0)
	rng := rand.New(rand.NewSource(42))

	result := spawner.SpawnMobsOnFloor(floor, 0, rng)
	if len(result) != 0 {
		t.Errorf("Should not spawn mobs on city floor, got %d", len(result))
	}
}

func TestSpawnMobsOnFloor_BossRoom(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	floor := NewFloor(1)
	bossRoom := world.NewRoom("boss1", "Boss Chamber", "A boss room", world.RoomTypeBoss)
	bossRoom.Floor = 1
	floor.AddRoom(bossRoom)

	rng := rand.New(rand.NewSource(42))
	spawned := spawner.SpawnMobsOnFloor(floor, 1, rng)

	// Should have spawned a boss
	if len(spawned) == 0 {
		t.Error("Should have spawned a boss")
		return
	}

	// Check it's the boss mob
	foundBoss := false
	for _, mob := range spawned {
		if mob.GetName() == "Goblin King" {
			foundBoss = true
			break
		}
	}
	if !foundBoss {
		t.Error("Should have spawned the Goblin King boss")
	}
}

func TestSpawnMobsOnFloor_TreasureRoom(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	floor := NewFloor(1)
	treasureRoom := world.NewRoom("treasure1", "Treasure Room", "A treasure room", world.RoomTypeTreasure)
	treasureRoom.Floor = 1
	floor.AddRoom(treasureRoom)

	rng := rand.New(rand.NewSource(42))
	spawned := spawner.SpawnMobsOnFloor(floor, 1, rng)

	// Treasure rooms should have 1-2 guards
	if len(spawned) < 1 || len(spawned) > 2 {
		t.Errorf("Treasure room should have 1-2 guards, got %d", len(spawned))
	}
}

func TestSpawnMobsOnFloor_StairsRoom(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	floor := NewFloor(1)
	stairsRoom := world.NewRoom("stairs1", "Stairway", "Stairs", world.RoomTypeStairs)
	stairsRoom.Floor = 1
	floor.AddRoom(stairsRoom)

	rng := rand.New(rand.NewSource(42))
	spawned := spawner.SpawnMobsOnFloor(floor, 1, rng)

	// No mobs in stairs rooms
	if len(spawned) != 0 {
		t.Errorf("Stairs room should have no mobs, got %d", len(spawned))
	}
}

func TestCreateScaledMob_Scaling(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	goblinDef := config.NPCs["goblin"]

	// Test floor 1 scaling
	mob1 := spawner.createScaledMob(&goblinDef, "room1", 1)
	expectedHP1 := ScaleHP(20, 1) // 20 * 1.1 = 22
	if mob1.GetMaxHealth() != expectedHP1 {
		t.Errorf("Floor 1 HP = %d, want %d", mob1.GetMaxHealth(), expectedHP1)
	}

	// Test floor 10 scaling
	mob10 := spawner.createScaledMob(&goblinDef, "room1", 10)
	expectedHP10 := ScaleHP(20, 10) // 20 * 2.0 = 40
	if mob10.GetMaxHealth() != expectedHP10 {
		t.Errorf("Floor 10 HP = %d, want %d", mob10.GetMaxHealth(), expectedHP10)
	}

	// Test level scaling
	expectedLevel10 := 1 + (10 / 2) // 1 + 5 = 6
	if mob10.GetLevel() != expectedLevel10 {
		t.Errorf("Floor 10 Level = %d, want %d", mob10.GetLevel(), expectedLevel10)
	}
}

func TestGetMobsByTier(t *testing.T) {
	config := createTestMobConfig()

	tier1 := config.GetMobsByTier(1)
	if len(tier1) != 1 { // goblin only (goblin_king is a boss)
		t.Errorf("Tier 1 should have 1 mob, got %d", len(tier1))
	}

	tier2 := config.GetMobsByTier(2)
	if len(tier2) != 1 { // orc
		t.Errorf("Tier 2 should have 1 mob, got %d", len(tier2))
	}
}

func TestGetBossesByTier(t *testing.T) {
	config := createTestMobConfig()

	bosses := config.GetBossesByTier(1)
	if len(bosses) != 1 { // goblin_king
		t.Errorf("Tier 1 should have 1 boss, got %d", len(bosses))
	}
	if bosses[0].Name != "Goblin King" {
		t.Errorf("Boss name = %s, want Goblin King", bosses[0].Name)
	}
}

func TestGetRandomMobForTier(t *testing.T) {
	config := createTestMobConfig()
	rng := rand.New(rand.NewSource(42))

	// Should get a tier 1 mob for tier 1
	mob := config.GetRandomMobForTier(1, rng)
	if mob == nil {
		t.Fatal("Should return a mob for tier 1")
	}
	if mob.Tier != 1 {
		t.Errorf("Mob tier = %d, want 1", mob.Tier)
	}

	// Should fall back to lower tier if requested tier has no mobs
	mob3 := config.GetRandomMobForTier(3, rng) // No tier 3 mobs
	if mob3 == nil {
		t.Fatal("Should fall back to lower tier")
	}
	if mob3.Tier > 3 {
		t.Errorf("Should not return higher tier, got tier %d", mob3.Tier)
	}
}

// TestSpawnBoss_MarkedAsBoss tests that spawned bosses are marked with IsBoss and Floor
func TestSpawnBoss_MarkedAsBoss(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	floor := NewFloor(10)
	bossRoom := world.NewRoom("boss10", "Boss Chamber", "A boss room", world.RoomTypeBoss)
	bossRoom.Floor = 10
	floor.AddRoom(bossRoom)

	rng := rand.New(rand.NewSource(42))
	spawned := spawner.SpawnMobsOnFloor(floor, 10, rng)

	if len(spawned) == 0 {
		t.Fatal("Should have spawned a boss")
	}

	// Find the boss NPC
	var boss *npc.NPC
	for _, mob := range spawned {
		if mob.GetName() == "Goblin King" {
			boss = mob
			break
		}
	}

	if boss == nil {
		t.Fatal("Should have spawned the Goblin King boss")
	}

	// Verify boss is marked as boss
	if !boss.GetIsBoss() {
		t.Error("Boss should have IsBoss = true")
	}

	// Verify floor is set correctly
	if boss.GetFloor() != 10 {
		t.Errorf("Boss floor = %d, want 10", boss.GetFloor())
	}
}

// TestSpawnRegularMob_NotMarkedAsBoss tests that regular mobs are not marked as bosses
func TestSpawnRegularMob_NotMarkedAsBoss(t *testing.T) {
	config := createTestMobConfig()
	spawner := NewMobSpawner(config)

	floor := NewFloor(1)
	treasureRoom := world.NewRoom("treasure1", "Treasure Room", "A treasure room", world.RoomTypeTreasure)
	treasureRoom.Floor = 1
	floor.AddRoom(treasureRoom)

	rng := rand.New(rand.NewSource(42))
	spawned := spawner.SpawnMobsOnFloor(floor, 1, rng)

	if len(spawned) == 0 {
		t.Skip("No mobs spawned in treasure room (RNG variance)")
	}

	// None of the mobs should be marked as bosses
	for _, mob := range spawned {
		if mob.GetIsBoss() {
			t.Errorf("Regular mob %s should not be marked as boss", mob.GetName())
		}
	}
}
