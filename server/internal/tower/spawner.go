package tower

import (
	"fmt"
	"math/rand"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// MerchantFloorInterval is how often a merchant appears (every N floors)
const MerchantFloorInterval = 5

// PlayerCountGetter is an interface for getting the current online player count
type PlayerCountGetter interface {
	GetOnlinePlayerCount() int
}

// SpawnConfig holds configuration for dynamic spawn scaling
type SpawnConfig struct {
	DynamicSpawns       bool    // Enable dynamic spawn scaling
	TargetMobsPerPlayer float64 // Target number of available mobs per player
	BaseMobsPerFloor    float64 // Base number of mobs per floor (calculated from floor layout)
	MinSpawnMultiplier  float64 // Minimum spawn multiplier (default 1.0)
	MaxSpawnMultiplier  float64 // Maximum spawn multiplier (default 10.0)
}

// DefaultSpawnConfig returns the default spawn configuration
func DefaultSpawnConfig() SpawnConfig {
	return SpawnConfig{
		DynamicSpawns:       true,
		TargetMobsPerPlayer: 5.0,  // 5 mobs per player is a good target
		BaseMobsPerFloor:    15.0, // Average ~15 mobs per floor at base spawn rate
		MinSpawnMultiplier:  1.0,
		MaxSpawnMultiplier:  10.0,
	}
}

// MobSpawner handles spawning mobs on tower floors
type MobSpawner struct {
	mobConfig         *npc.NPCsConfig
	spawnConfig       SpawnConfig
	playerCountGetter PlayerCountGetter
}

// NewMobSpawner creates a new mob spawner with the given configuration
func NewMobSpawner(config *npc.NPCsConfig) *MobSpawner {
	return &MobSpawner{
		mobConfig:   config,
		spawnConfig: DefaultSpawnConfig(),
	}
}

// SetSpawnConfig sets the spawn configuration
func (s *MobSpawner) SetSpawnConfig(config SpawnConfig) {
	s.spawnConfig = config
}

// SetPlayerCountGetter sets the player count getter for dynamic spawning
func (s *MobSpawner) SetPlayerCountGetter(getter PlayerCountGetter) {
	s.playerCountGetter = getter
}

// GetSpawnMultiplier calculates the spawn multiplier based on player count
// and available floors. Returns 1.0 if dynamic spawns are disabled.
func (s *MobSpawner) GetSpawnMultiplier(floorsAvailable int) float64 {
	if !s.spawnConfig.DynamicSpawns || s.playerCountGetter == nil {
		return 1.0
	}

	playerCount := s.playerCountGetter.GetOnlinePlayerCount()
	if playerCount <= 0 {
		return 1.0
	}

	if floorsAvailable <= 0 {
		floorsAvailable = 1
	}

	// Calculate players per floor (assuming even distribution)
	playersPerFloor := float64(playerCount) / float64(floorsAvailable)

	// Target mobs needed = players per floor * target mobs per player
	targetMobs := playersPerFloor * s.spawnConfig.TargetMobsPerPlayer

	// Calculate multiplier
	multiplier := targetMobs / s.spawnConfig.BaseMobsPerFloor

	// Clamp to configured range
	if multiplier < s.spawnConfig.MinSpawnMultiplier {
		multiplier = s.spawnConfig.MinSpawnMultiplier
	}
	if multiplier > s.spawnConfig.MaxSpawnMultiplier {
		multiplier = s.spawnConfig.MaxSpawnMultiplier
	}

	return multiplier
}

// SpawnMobsOnFloor spawns appropriate mobs on a floor based on floor number
// Returns the list of spawned NPCs
func (s *MobSpawner) SpawnMobsOnFloor(floor *Floor, floorNum int, rng *rand.Rand) []*npc.NPC {
	return s.SpawnMobsOnFloorWithMultiplier(floor, floorNum, rng, 1.0)
}

// SpawnMobsOnFloorWithMultiplier spawns mobs with a spawn multiplier for dynamic scaling
// The multiplier increases the number of mobs spawned (e.g., 2.0 = double mobs)
func (s *MobSpawner) SpawnMobsOnFloorWithMultiplier(floor *Floor, floorNum int, rng *rand.Rand, multiplier float64) []*npc.NPC {
	if s.mobConfig == nil {
		return nil
	}

	tier := GetMobTier(floorNum)
	if tier == 0 {
		return nil // No mobs on floor 0 (city)
	}

	if multiplier < 1.0 {
		multiplier = 1.0
	}

	var spawned []*npc.NPC

	rooms := floor.GetRooms()
	for _, room := range rooms {
		// Determine what to spawn based on room type
		switch room.Type {
		case world.RoomTypeBoss:
			// Boss rooms get a boss mob (no scaling - always one boss)
			npcs := s.spawnBossInRoom(room, floorNum, tier, rng)
			spawned = append(spawned, npcs...)

		case world.RoomTypeTreasure:
			// Treasure rooms get 1-2 mobs guarding the loot (scaled)
			scaledMin, scaledMax := scaleSpawnRange(1, 2, multiplier)
			npcs := s.spawnGuardsInRoom(room, floorNum, tier, rng, scaledMin, scaledMax)
			spawned = append(spawned, npcs...)

		case world.RoomTypeRoom:
			// Regular rooms have a high chance to spawn 1-3 mobs (scaled)
			// With high multiplier, increase both chance and count
			spawnChance := 0.8 * multiplier
			if spawnChance > 1.0 {
				spawnChance = 1.0
			}
			if rng.Float64() < spawnChance {
				scaledMin, scaledMax := scaleSpawnRange(1, 3, multiplier)
				npcs := s.spawnMobsInRoom(room, floorNum, tier, rng, scaledMin, scaledMax)
				spawned = append(spawned, npcs...)
			}

		case world.RoomTypeCorridor:
			// Corridors have a moderate chance to spawn 1-2 mobs (scaled)
			spawnChance := 0.6 * multiplier
			if spawnChance > 1.0 {
				spawnChance = 1.0
			}
			if rng.Float64() < spawnChance {
				scaledMin, scaledMax := scaleSpawnRange(1, 2, multiplier)
				npcs := s.spawnMobsInRoom(room, floorNum, tier, rng, scaledMin, scaledMax)
				spawned = append(spawned, npcs...)
			}

		case world.RoomTypeStairs:
			// No mobs in stairway rooms (safe zones)
			continue
		}
	}

	return spawned
}

// scaleSpawnRange scales min/max spawn counts by a multiplier
// Returns scaled min and max values (at least 1 for min)
func scaleSpawnRange(min, max int, multiplier float64) (int, int) {
	scaledMin := int(float64(min) * multiplier)
	scaledMax := int(float64(max) * multiplier)
	if scaledMin < 1 {
		scaledMin = 1
	}
	if scaledMax < scaledMin {
		scaledMax = scaledMin
	}
	return scaledMin, scaledMax
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
	stairsRoom.SetDescription(fmt.Sprintf("%s A crusty old merchant has set up a small trading post here.", stairsRoom.GetBaseDescription()))

	return merchant
}

// SpawnAdditionalMobs spawns extra mobs on a floor based on the current mob count
// This is used for dynamic spawn scaling to add mobs when player count increases
// It only spawns in rooms that have fewer than the maximum expected mobs
func (s *MobSpawner) SpawnAdditionalMobs(floor *Floor, floorNum int, rng *rand.Rand, targetCount int) []*npc.NPC {
	if s.mobConfig == nil || floorNum == 0 {
		return nil
	}

	tier := GetMobTier(floorNum)
	if tier == 0 {
		return nil
	}

	// Count current alive mobs on the floor
	currentCount := 0
	rooms := floor.GetRooms()
	for _, room := range rooms {
		for _, n := range room.GetNPCs() {
			if n.GetHealth() > 0 && n.IsAttackable() {
				currentCount++
			}
		}
	}

	// Calculate how many new mobs to spawn
	mobsToSpawn := targetCount - currentCount
	if mobsToSpawn <= 0 {
		return nil
	}

	var spawned []*npc.NPC

	// Collect rooms that can accept more mobs (not boss, not stairs)
	var eligibleRooms []*world.Room
	for _, room := range rooms {
		if room.Type == world.RoomTypeBoss || room.Type == world.RoomTypeStairs {
			continue
		}
		// Count existing alive mobs in room
		aliveMobs := 0
		for _, n := range room.GetNPCs() {
			if n.GetHealth() > 0 && n.IsAttackable() {
				aliveMobs++
			}
		}
		// Only add rooms with fewer than 5 mobs (to prevent overcrowding)
		if aliveMobs < 5 {
			eligibleRooms = append(eligibleRooms, room)
		}
	}

	if len(eligibleRooms) == 0 {
		return nil
	}

	// Spawn mobs in random eligible rooms
	for i := 0; i < mobsToSpawn && len(eligibleRooms) > 0; i++ {
		// Pick a random room
		roomIdx := rng.Intn(len(eligibleRooms))
		room := eligibleRooms[roomIdx]

		// Spawn one mob
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

// CountAliveMobs counts the number of alive attackable mobs on a floor
func CountAliveMobs(floor *Floor) int {
	count := 0
	for _, room := range floor.GetRooms() {
		for _, n := range room.GetNPCs() {
			if n.GetHealth() > 0 && n.IsAttackable() {
				count++
			}
		}
	}
	return count
}
