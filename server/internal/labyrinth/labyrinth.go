package labyrinth

import (
	"math/rand"
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// GateInfo holds information about a city gate
type GateInfo struct {
	CityID   string
	CityName string
	RoomID   string
}

// ShortcutInfo holds information about a shortcut pair
type ShortcutInfo struct {
	RoomA string
	RoomB string
}

// Labyrinth represents the great labyrinth connecting all cities
type Labyrinth struct {
	Width     int
	Height    int
	Rooms     map[string]*world.Room
	Gates     []GateInfo              // City gates
	GateRooms map[string]string       // cityID -> gate room ID
	Shortcuts []ShortcutInfo          // Shortcut pairs
	mu        sync.RWMutex
}

// New creates a new empty labyrinth
func New() *Labyrinth {
	return &Labyrinth{
		Rooms:     make(map[string]*world.Room),
		GateRooms: make(map[string]string),
	}
}

// GetRoom returns a room by ID
func (l *Labyrinth) GetRoom(roomID string) *world.Room {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Rooms[roomID]
}

// GetRooms returns all rooms in the labyrinth
func (l *Labyrinth) GetRooms() map[string]*world.Room {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rooms := make(map[string]*world.Room, len(l.Rooms))
	for id, room := range l.Rooms {
		rooms[id] = room
	}
	return rooms
}

// RoomCount returns the number of rooms in the labyrinth
func (l *Labyrinth) RoomCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.Rooms)
}

// GetGateRoom returns the gate room ID for a city
func (l *Labyrinth) GetGateRoom(cityID string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.GateRooms[cityID]
}

// GetGateRoomForCity returns the gate room for a specific city
func (l *Labyrinth) GetGateRoomForCity(cityID string) *world.Room {
	l.mu.RLock()
	defer l.mu.RUnlock()
	roomID := l.GateRooms[cityID]
	if roomID == "" {
		return nil
	}
	return l.Rooms[roomID]
}

// GetAllGates returns all gate information
func (l *Labyrinth) GetAllGates() []GateInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	gates := make([]GateInfo, len(l.Gates))
	copy(gates, l.Gates)
	return gates
}

// FindRoom searches for a room by ID (returns nil if not found)
func (l *Labyrinth) FindRoom(roomID string) *world.Room {
	return l.GetRoom(roomID)
}

// IsLabyrinthRoom returns true if the room ID belongs to the labyrinth
func (l *Labyrinth) IsLabyrinthRoom(roomID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, exists := l.Rooms[roomID]
	return exists
}

// GetCityIDForGateRoom returns the city ID for a gate room, or empty string if not a gate
func (l *Labyrinth) GetCityIDForGateRoom(roomID string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, gate := range l.Gates {
		if gate.RoomID == roomID {
			return gate.CityID
		}
	}
	return ""
}

// addRoom adds a room to the labyrinth (internal use during loading)
func (l *Labyrinth) addRoom(room *world.Room) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Rooms[room.ID] = room
}

// SpawnMobs spawns mobs throughout the labyrinth using the provided mob configuration.
// Mobs are spawned in passage rooms (not gates or special NPC rooms).
// Returns the number of mobs spawned.
func (l *Labyrinth) SpawnMobs(mobConfig *npc.NPCsConfig, rng *rand.Rand) int {
	if mobConfig == nil {
		return 0
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	spawned := 0
	labyrinthTags := []string{"labyrinth"}

	for _, room := range l.Rooms {
		// Skip gate rooms (city entrances - safe zones)
		if room.Type == world.RoomTypeLabyrinthGate {
			continue
		}

		// Skip rooms with special features (merchant, lore_npc)
		if room.HasFeature("merchant") || room.HasFeature("lore_npc") {
			continue
		}

		// 40% chance to spawn 1-2 mobs in regular passage rooms
		if rng.Float64() < 0.4 {
			// Determine tier: mostly tier 1, occasionally tier 2
			tier := 1
			if rng.Float64() < 0.3 {
				tier = 2
			}

			// Spawn 1-2 mobs
			mobCount := 1 + rng.Intn(2)
			for i := 0; i < mobCount; i++ {
				mobDef := mobConfig.GetRandomMobForTierAndTags(tier, labyrinthTags, rng)
				if mobDef == nil {
					continue
				}

				// Create the mob
				mob := npc.NewNPC(
					mobDef.Name,
					mobDef.Description,
					mobDef.Level,
					mobDef.Health,
					mobDef.Damage,
					mobDef.Armor,
					mobDef.Experience,
					mobDef.Aggressive,
					mobDef.Attackable,
					room.ID,
					mobDef.RespawnMedian,
					mobDef.RespawnVariation,
				)

				// Copy loot table
				if len(mobDef.LootTable) > 0 {
					lootTable := make([]npc.LootEntry, len(mobDef.LootTable))
					for j, entry := range mobDef.LootTable {
						lootTable[j] = npc.LootEntry{
							ItemName:   entry.Item,
							DropChance: entry.Chance,
						}
					}
					mob.SetLootTable(lootTable)
				}

				// Set gold drop range
				mob.SetGoldDrop(mobDef.GoldMin, mobDef.GoldMax)

				room.AddNPC(mob)
				spawned++
			}
		}
	}

	return spawned
}
