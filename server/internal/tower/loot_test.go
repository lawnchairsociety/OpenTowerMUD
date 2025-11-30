package tower

import (
	"math/rand"
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func TestNewLootSpawner(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"test_item": {Name: "Test Item", Tier: 1},
		},
	}

	spawner := NewLootSpawner(config)
	if spawner == nil {
		t.Fatal("expected spawner to be created")
	}
	if spawner.itemConfig == nil {
		t.Error("expected itemConfig to be set")
	}
}

func TestSpawnLootOnFloor_NoConfig(t *testing.T) {
	spawner := NewLootSpawner(nil)
	floor := NewFloor(1)
	rng := rand.New(rand.NewSource(12345))

	// Should not panic with nil config
	spawner.SpawnLootOnFloor(floor, 1, rng)
}

func TestSpawnLootOnFloor_CityFloor(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"test_item": {Name: "Test Item", Tier: 1, Value: 10, Weight: 1.0},
		},
	}
	spawner := NewLootSpawner(config)
	floor := NewFloor(0) // City floor

	room := world.NewRoom("test_room", "Test", "Test", world.RoomTypeTreasure)
	floor.AddRoom(room)

	rng := rand.New(rand.NewSource(12345))
	spawner.SpawnLootOnFloor(floor, 0, rng)

	// City floor should not get loot
	if len(room.Items) > 0 {
		t.Error("city floor should not spawn loot")
	}
}

func TestSpawnLootOnFloor_TreasureRoom(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"test_gem": {Name: "Test Gem", Tier: 1, Value: 50, Weight: 0.1, Type: "misc"},
		},
	}
	spawner := NewLootSpawner(config)
	floor := NewFloor(1)

	room := world.NewRoom("treasure_room", "Treasure Room", "Test", world.RoomTypeTreasure)
	floor.AddRoom(room)

	rng := rand.New(rand.NewSource(12345))
	spawner.SpawnLootOnFloor(floor, 1, rng)

	// Treasure room should have items (2-4 items + gold)
	itemCount := len(room.Items)
	if itemCount < 2 {
		t.Errorf("expected at least 2 items in treasure room, got %d", itemCount)
	}
}

func TestSpawnLootOnFloor_BossRoom(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"epic_item": {Name: "Epic Item", Tier: 2, Value: 200, Weight: 1.0, Type: "misc"},
		},
	}
	spawner := NewLootSpawner(config)
	floor := NewFloor(10) // Boss floor

	room := world.NewRoom("boss_room", "Boss Chamber", "Test", world.RoomTypeBoss)
	floor.AddRoom(room)

	rng := rand.New(rand.NewSource(12345))
	spawner.SpawnLootOnFloor(floor, 10, rng)

	// Boss room should have items (3-5 items + gold)
	itemCount := len(room.Items)
	if itemCount < 3 {
		t.Errorf("expected at least 3 items in boss room, got %d", itemCount)
	}
}

func TestSpawnLootOnFloor_RegularRoom(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"test_item": {Name: "Test Item", Tier: 1, Value: 10, Weight: 1.0, Type: "misc"},
		},
	}
	spawner := NewLootSpawner(config)
	floor := NewFloor(1)

	room := world.NewRoom("regular_room", "Chamber", "Test", world.RoomTypeRoom)
	floor.AddRoom(room)

	rng := rand.New(rand.NewSource(12345))
	spawner.SpawnLootOnFloor(floor, 1, rng)

	// Regular rooms should NOT get loot from loot spawner
	if len(room.Items) > 0 {
		t.Error("regular room should not get treasure loot")
	}
}

func TestSpawnLootInRoom_GoldAlwaysSpawns(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{}, // No items defined
	}
	spawner := NewLootSpawner(config)

	room := world.NewRoom("test_room", "Test", "Test", world.RoomTypeTreasure)
	rng := rand.New(rand.NewSource(12345))

	spawner.spawnLootInRoom(room, 1, rng, 0, 0) // 0 items requested

	// Gold should still spawn even with no items
	itemCount := len(room.Items)
	if itemCount == 0 {
		t.Error("expected gold to spawn even with no items")
	}

	// Check that spawned items are copper coins
	for _, item := range room.Items {
		if item.ID != "copper_coin" {
			t.Errorf("expected copper_coin, got %s", item.ID)
		}
	}
}

func TestGetRandomLootItem_TierFallback(t *testing.T) {
	config := &items.ItemsConfig{
		Items: map[string]items.ItemDefinition{
			"tier1_item": {Name: "Tier 1 Item", Tier: 1, Value: 10, Weight: 1.0},
			// No tier 2 items
		},
	}
	spawner := NewLootSpawner(config)
	rng := rand.New(rand.NewSource(12345))

	// Request tier 2 item, should fall back to tier 1
	item := spawner.getRandomLootItem(2, rng)
	if item == nil {
		t.Fatal("expected item to be returned with tier fallback")
	}
	if item.Name != "Tier 1 Item" {
		t.Errorf("expected 'Tier 1 Item', got '%s'", item.Name)
	}
}
