package tower

import (
	"fmt"
	"math/rand"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// MerchantFloorInterval is how often a merchant appears (every N floors)
const MerchantFloorInterval = 5

// MobSpawner handles spawning mobs on tower floors
type MobSpawner struct {
	mobConfig *npc.NPCsConfig
}

// NewMobSpawner creates a new mob spawner with the given configuration
func NewMobSpawner(config *npc.NPCsConfig) *MobSpawner {
	return &MobSpawner{
		mobConfig: config,
	}
}

// SpawnMobsOnFloor spawns appropriate mobs on a floor based on floor number
// Returns the list of spawned NPCs
func (s *MobSpawner) SpawnMobsOnFloor(floor *Floor, floorNum int, rng *rand.Rand) []*npc.NPC {
	if s.mobConfig == nil {
		return nil
	}

	tier := GetMobTier(floorNum)
	if tier == 0 {
		return nil // No mobs on floor 0 (city)
	}

	var spawned []*npc.NPC

	rooms := floor.GetRooms()
	for _, room := range rooms {
		// Determine what to spawn based on room type
		switch room.Type {
		case world.RoomTypeBoss:
			// Boss rooms get a boss mob
			npcs := s.spawnBossInRoom(room, floorNum, tier, rng)
			spawned = append(spawned, npcs...)

		case world.RoomTypeTreasure:
			// Treasure rooms get 1-2 mobs guarding the loot
			npcs := s.spawnGuardsInRoom(room, floorNum, tier, rng, 1, 2)
			spawned = append(spawned, npcs...)

		case world.RoomTypeRoom:
			// Regular rooms have a chance to spawn 0-2 mobs
			if rng.Float32() < 0.4 { // 40% chance of mobs
				npcs := s.spawnMobsInRoom(room, floorNum, tier, rng, 0, 2)
				spawned = append(spawned, npcs...)
			}

		case world.RoomTypeCorridor:
			// Corridors have a lower chance to spawn 0-1 mobs
			if rng.Float32() < 0.2 { // 20% chance of mobs
				npcs := s.spawnMobsInRoom(room, floorNum, tier, rng, 0, 1)
				spawned = append(spawned, npcs...)
			}

		case world.RoomTypeStairs:
			// No mobs in stairway rooms (safe zones)
			continue
		}
	}

	return spawned
}

// spawnBossInRoom spawns a boss mob in the room
func (s *MobSpawner) spawnBossInRoom(room *world.Room, floorNum, tier int, rng *rand.Rand) []*npc.NPC {
	bossDef := s.mobConfig.GetRandomBossForTier(tier, rng)
	if bossDef == nil {
		return nil
	}

	boss := s.createScaledMob(bossDef, room.ID, floorNum)
	// Mark as boss so it drops the floor key
	boss.SetBoss(floorNum)
	room.AddNPC(boss)
	return []*npc.NPC{boss}
}

// spawnGuardsInRoom spawns guard mobs in treasure rooms
func (s *MobSpawner) spawnGuardsInRoom(room *world.Room, floorNum, tier int, rng *rand.Rand, minMobs, maxMobs int) []*npc.NPC {
	count := minMobs + rng.Intn(maxMobs-minMobs+1)
	if count == 0 {
		return nil
	}

	var spawned []*npc.NPC
	for i := 0; i < count; i++ {
		mobDef := s.mobConfig.GetRandomMobForTier(tier, rng)
		if mobDef == nil {
			continue
		}

		mob := s.createScaledMob(mobDef, room.ID, floorNum)
		room.AddNPC(mob)
		spawned = append(spawned, mob)
	}

	return spawned
}

// spawnMobsInRoom spawns random mobs in a room
func (s *MobSpawner) spawnMobsInRoom(room *world.Room, floorNum, tier int, rng *rand.Rand, minMobs, maxMobs int) []*npc.NPC {
	count := minMobs + rng.Intn(maxMobs-minMobs+1)
	if count == 0 {
		return nil
	}

	var spawned []*npc.NPC
	for i := 0; i < count; i++ {
		mobDef := s.mobConfig.GetRandomMobForTier(tier, rng)
		if mobDef == nil {
			continue
		}

		mob := s.createScaledMob(mobDef, room.ID, floorNum)
		room.AddNPC(mob)
		spawned = append(spawned, mob)
	}

	return spawned
}

// createScaledMob creates an NPC from a definition with floor-scaled stats
func (s *MobSpawner) createScaledMob(def *npc.NPCDefinition, roomID string, floorNum int) *npc.NPC {
	// Apply floor scaling to stats
	scaledHP := ScaleHP(def.Health, floorNum)
	scaledDamage := ScaleDamage(def.Damage, floorNum)
	scaledXP := ScaleXP(def.Experience, floorNum)

	// Calculate scaled level (roughly 1 level per 2 floors + base level)
	scaledLevel := def.Level + (floorNum / 2)

	mob := npc.NewNPC(
		def.Name,
		def.Description,
		scaledLevel,
		scaledHP,
		scaledDamage,
		def.Armor,
		scaledXP,
		def.Aggressive,
		def.Attackable,
		roomID,
		def.RespawnMedian,
		def.RespawnVariation,
	)

	// Copy loot table for percentage-based drops
	if len(def.LootTable) > 0 {
		lootTable := make([]npc.LootEntry, len(def.LootTable))
		for i, entry := range def.LootTable {
			lootTable[i] = npc.LootEntry{
				ItemName:   entry.Item,
				DropChance: entry.Chance,
			}
		}
		mob.SetLootTable(lootTable)
	}

	return mob
}

// HasMerchant returns true if the given floor should have a merchant
func HasMerchant(floorNum int) bool {
	// Merchants appear every MerchantFloorInterval floors starting at floor 5
	// (not on floor 0 which is the city with a proper shop)
	return floorNum > 0 && floorNum%MerchantFloorInterval == 0
}

// SpawnMerchantOnFloor adds a merchant NPC to the portal room if this floor has one
func SpawnMerchantOnFloor(floor *Floor, floorNum int) *npc.NPC {
	if !HasMerchant(floorNum) {
		return nil
	}

	// Find the portal room (where players arrive via portal or stairs down)
	portalRoom := floor.GetPortalRoom()
	if portalRoom == nil {
		// Fallback to stairs down if no portal room
		portalRoom = floor.GetStairsDown()
	}
	if portalRoom == nil {
		return nil
	}
	stairsRoom := portalRoom

	// Create the merchant NPC
	merchant := npc.NewNPC(
		"crusty old merchant",
		"A weathered old man with a heavy pack full of supplies. His prices are steep, but he's the only trader brave enough to venture this deep into the tower.",
		1,   // Level (doesn't matter - not attackable)
		100, // Health (doesn't matter)
		0,   // Damage (doesn't matter)
		0,   // Armor (doesn't matter)
		0,   // XP (doesn't matter)
		false, // Not aggressive
		false, // Can't be attacked
		stairsRoom.ID,
		0, // No respawn (permanent)
		0,
	)

	// Set the merchant's shop inventory (consumables at 150% markup)
	merchant.SetShopInventory([]npc.ShopItem{
		{ItemName: "bread", Price: 3},           // 150% of 2
		{ItemName: "apple", Price: 5},           // 150% of 3
		{ItemName: "water", Price: 3},           // 150% of 2
		{ItemName: "bandage", Price: 8},         // 150% of 5
		{ItemName: "healing_potion", Price: 75}, // 150% of 50
		{ItemName: "mana_potion", Price: 75},    // 150% of 50
	})

	// Add the merchant to the room
	stairsRoom.AddNPC(merchant)

	// Add the "merchant" feature to the room so commands can detect it
	stairsRoom.AddFeature("merchant")

	// Update room description to mention the merchant
	stairsRoom.Description = fmt.Sprintf("%s A crusty old merchant has set up a small trading post here.", stairsRoom.Description)

	return merchant
}
