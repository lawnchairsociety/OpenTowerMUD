package tower

import (
	"math/rand"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// LootSpawner handles spawning loot in treasure rooms
type LootSpawner struct {
	itemConfig *items.ItemsConfig
}

// NewLootSpawner creates a new loot spawner with the given configuration
func NewLootSpawner(config *items.ItemsConfig) *LootSpawner {
	return &LootSpawner{
		itemConfig: config,
	}
}

// SpawnLootOnFloor spawns loot in treasure rooms based on floor number
func (s *LootSpawner) SpawnLootOnFloor(floor *Floor, floorNum int, rng *rand.Rand) {
	if s.itemConfig == nil {
		return
	}

	lootTier := GetLootTier(floorNum)
	if lootTier == 0 {
		return // No loot on floor 0 (city)
	}

	rooms := floor.GetRooms()
	for _, room := range rooms {
		switch room.Type {
		case world.RoomTypeTreasure:
			// Treasure rooms get 2-4 items
			s.spawnLootInRoom(room, lootTier, rng, 2, 4)

		case world.RoomTypeBoss:
			// Boss rooms get 3-5 items of higher tier
			bossTier := lootTier
			if bossTier < 5 {
				bossTier++ // Boss loot is one tier higher
			}
			s.spawnLootInRoom(room, bossTier, rng, 3, 5)
		}
	}
}

// spawnLootInRoom spawns random loot items in a room
func (s *LootSpawner) spawnLootInRoom(room *world.Room, tier int, rng *rand.Rand, minItems, maxItems int) {
	count := minItems + rng.Intn(maxItems-minItems+1)

	for i := 0; i < count; i++ {
		item := s.getRandomLootItem(tier, rng)
		if item != nil {
			room.AddItem(item)
		}
	}

	// Always add some gold (copper coins) based on tier
	goldCount := tier + rng.Intn(tier*2+1)
	for i := 0; i < goldCount; i++ {
		coin := items.NewItem("copper coin", "A shiny copper coin", 0.01, items.Misc, 1)
		coin.ID = "copper_coin"
		room.AddItem(coin)
	}
}

// getRandomLootItem returns a random item appropriate for the tier
func (s *LootSpawner) getRandomLootItem(tier int, rng *rand.Rand) *items.Item {
	return s.itemConfig.GetRandomItemForTier(tier, rng)
}
