package tower

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
	"github.com/lawnchairsociety/opentowermud/server/internal/wfc"
)

// Tower represents the entire tower structure
type Tower struct {
	Seed         int64             // Base seed for generation
	Floors       map[int]*Floor    // All generated floors
	HighestFloor int               // Highest floor generated so far
	SavePath     string            // Path to save tower data
	mobSpawner   *MobSpawner       // Spawner for floor mobs
	lootSpawner  *LootSpawner      // Spawner for treasure loot
	mu           sync.RWMutex
}

// NewTower creates a new tower with the given seed
func NewTower(seed int64) *Tower {
	return &Tower{
		Seed:         seed,
		Floors:       make(map[int]*Floor),
		HighestFloor: 0,
	}
}

// GetFloor returns a floor by number, generating it if necessary
func (t *Tower) GetFloor(floorNum int) (*Floor, error) {
	t.mu.RLock()
	floor, exists := t.Floors[floorNum]
	t.mu.RUnlock()

	if exists {
		return floor, nil
	}

	// Generate the floor
	return t.generateFloor(floorNum)
}

// GetFloorIfExists returns a floor only if it already exists
func (t *Tower) GetFloorIfExists(floorNum int) *Floor {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Floors[floorNum]
}

// HasFloor returns true if the floor has been generated
func (t *Tower) HasFloor(floorNum int) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.Floors[floorNum]
	return exists
}

// SetFloor sets a floor directly (used for loading from persistence)
func (t *Tower) SetFloor(floorNum int, floor *Floor) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Floors[floorNum] = floor
	if floorNum > t.HighestFloor {
		t.HighestFloor = floorNum
	}
}

// GetRoom returns a room by floor and room ID
func (t *Tower) GetRoom(floorNum int, roomID string) *world.Room {
	floor := t.GetFloorIfExists(floorNum)
	if floor == nil {
		return nil
	}
	return floor.GetRoom(roomID)
}

// GetAllRooms returns all rooms across all floors
func (t *Tower) GetAllRooms() map[string]*world.Room {
	t.mu.RLock()
	defer t.mu.RUnlock()

	allRooms := make(map[string]*world.Room)
	for _, floor := range t.Floors {
		for id, room := range floor.GetRooms() {
			allRooms[id] = room
		}
	}
	return allRooms
}

// generateFloor creates a new floor using WFC
func (t *Tower) generateFloor(floorNum int) (*Floor, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check it wasn't generated while we were waiting for the lock
	if floor, exists := t.Floors[floorNum]; exists {
		return floor, nil
	}

	// Floor 0 is the city - it should be set externally
	if floorNum == 0 {
		return nil, fmt.Errorf("floor 0 (city) must be set explicitly, not generated")
	}

	// Create floor config
	config := wfc.DefaultFloorConfig(floorNum, t.Seed)

	// Generate the floor layout
	gen := wfc.NewGenerator(config)
	generatedFloor, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate floor %d: %w", floorNum, err)
	}

	// Convert WFC tiles to world rooms
	floor := t.convertToFloor(floorNum, generatedFloor)

	// Store the floor
	t.Floors[floorNum] = floor
	if floorNum > t.HighestFloor {
		t.HighestFloor = floorNum
	}

	// Connect stairs to adjacent floors if they exist
	t.connectStairsLocked(floorNum)

	// Create floor-specific RNG for reproducible spawning
	floorRNG := rand.New(rand.NewSource(t.Seed + int64(floorNum)*1000))

	// Spawn mobs on the floor if spawner is configured
	if t.mobSpawner != nil {
		t.mobSpawner.SpawnMobsOnFloor(floor, floorNum, floorRNG)
	}

	// Spawn loot in treasure/boss rooms if spawner is configured
	if t.lootSpawner != nil {
		t.lootSpawner.SpawnLootOnFloor(floor, floorNum, floorRNG)
	}

	// Lock stairs on boss floors - players must defeat boss to get key
	if IsBossFloor(floorNum) {
		t.lockStairsOnBossFloor(floor, floorNum)
	}

	// Lock treasure room entrances - players must use purchasable keys
	t.lockTreasureRooms(floor)

	// Spawn merchant on floors that have one
	SpawnMerchantOnFloor(floor, floorNum)

	// Auto-save if a save path is configured
	if t.SavePath != "" {
		// Save in a goroutine to not block floor generation
		go func() {
			if err := SaveTower(t, t.SavePath); err != nil {
				// Log but don't fail - floor generation succeeded
				fmt.Printf("Warning: failed to save tower: %v\n", err)
			}
		}()
	}

	return floor, nil
}

// convertToFloor converts WFC output to a Floor with world.Room objects
func (t *Tower) convertToFloor(floorNum int, generated *wfc.GeneratedFloor) *Floor {
	floor := NewFloor(floorNum)

	// Create a map of tile positions for looking up neighbors
	tileMap := make(map[string]*wfc.Tile)
	for _, tile := range generated.Tiles {
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		tileMap[key] = tile
	}

	// Create rooms from tiles
	roomMap := make(map[string]*world.Room) // key -> room for exit linking
	for _, tile := range generated.Tiles {
		roomID := wfc.GetRoomID(floorNum, tile.X, tile.Y)
		roomType := tileTypeToRoomType(tile.Type)
		name := generateRoomName(tile.Type, floorNum, tile.X, tile.Y)
		desc := generateRoomDescription(tile.Type, floorNum)

		room := world.NewRoom(roomID, name, desc, roomType)
		room.Floor = floorNum

		// Add features based on tile type
		switch tile.Type {
		case wfc.TileStairsUp:
			room.AddFeature("stairs_up")
		case wfc.TileStairsDown:
			room.AddFeature("stairs_down")
			room.AddFeature("portal") // Portal is at the stairs down (entry point)
		case wfc.TileTreasure:
			room.AddFeature("treasure")
		case wfc.TileBoss:
			room.AddFeature("boss")
		}

		floor.AddRoom(room)
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		roomMap[key] = room
	}

	// Set special rooms based on generator output
	if generated.StairsUpTile != nil {
		roomID := wfc.GetRoomID(floorNum, generated.StairsUpTile.X, generated.StairsUpTile.Y)
		floor.SetStairsUp(roomID)
	}
	if generated.StairsDownTile != nil {
		roomID := wfc.GetRoomID(floorNum, generated.StairsDownTile.X, generated.StairsDownTile.Y)
		floor.SetStairsDown(roomID)
		floor.SetPortalRoom(roomID) // Portal is at stairs down (entry point)
		// Add portal feature to the room if not already present
		if room := floor.GetRoom(roomID); room != nil && !room.HasFeature("portal") {
			room.AddFeature("portal")
		}
	}

	// Link rooms based on WFC connections
	for _, tile := range generated.Tiles {
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		room := roomMap[key]

		for _, dir := range wfc.AllDirections() {
			if !tile.HasConnection(dir) {
				continue
			}

			nx, ny := tile.X, tile.Y
			switch dir {
			case wfc.North:
				ny--
			case wfc.South:
				ny++
			case wfc.East:
				nx++
			case wfc.West:
				nx--
			}

			neighborKey := fmt.Sprintf("%d,%d", nx, ny)
			if neighbor, ok := roomMap[neighborKey]; ok {
				room.AddExit(dir.String(), neighbor)
			}
		}
	}

	return floor
}

// connectStairsLocked connects stairs between floors (caller must hold lock)
func (t *Tower) connectStairsLocked(floorNum int) {
	floor := t.Floors[floorNum]
	if floor == nil {
		return
	}

	// Connect to floor below
	if floorNum > 0 {
		belowFloor := t.Floors[floorNum-1]
		if belowFloor != nil {
			stairsUp := belowFloor.GetStairsUp()
			stairsDown := floor.GetStairsDown()
			if stairsUp != nil && stairsDown != nil {
				stairsUp.AddExit("up", stairsDown)
				stairsDown.AddExit("down", stairsUp)
			}
		}
	}

	// Connect to floor above
	if floorNum < t.HighestFloor {
		aboveFloor := t.Floors[floorNum+1]
		if aboveFloor != nil {
			stairsUp := floor.GetStairsUp()
			stairsDown := aboveFloor.GetStairsDown()
			if stairsUp != nil && stairsDown != nil {
				stairsUp.AddExit("up", stairsDown)
				stairsDown.AddExit("down", stairsUp)
			}
		}
	}
}

// ConnectFloorToCity connects floor 1's stairs down to the tower entrance in the city
func (t *Tower) ConnectFloorToCity(towerEntranceRoom *world.Room) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	floor1 := t.Floors[1]
	if floor1 == nil {
		return fmt.Errorf("floor 1 not generated yet")
	}

	stairsDown := floor1.GetStairsDown()
	if stairsDown == nil {
		return fmt.Errorf("floor 1 has no stairs down room")
	}

	// Connect bidirectionally
	towerEntranceRoom.AddExit("up", stairsDown)
	stairsDown.AddExit("down", towerEntranceRoom)

	return nil
}

// GetHighestFloor returns the highest floor number that has been generated
func (t *Tower) GetHighestFloor() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.HighestFloor
}

// FloorCount returns the number of generated floors
func (t *Tower) FloorCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.Floors)
}

// SetSavePath sets the path for auto-saving the tower
func (t *Tower) SetSavePath(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.SavePath = path
}

// SetMobSpawner sets the mob spawner for floor generation
func (t *Tower) SetMobSpawner(spawner *MobSpawner) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.mobSpawner = spawner
}

// SetMobConfig is a convenience method to create and set a mob spawner from config
func (t *Tower) SetMobConfig(config *npc.NPCsConfig) {
	t.SetMobSpawner(NewMobSpawner(config))
}

// SetLootSpawner sets the loot spawner for floor generation
func (t *Tower) SetLootSpawner(spawner *LootSpawner) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lootSpawner = spawner
}

// SetItemConfig is a convenience method to create and set a loot spawner from config
func (t *Tower) SetItemConfig(config *items.ItemsConfig) {
	t.SetLootSpawner(NewLootSpawner(config))
}

// GetFloorPortalRoom returns the portal room for a specific floor
// Returns nil if the floor doesn't exist or has no portal room
func (t *Tower) GetFloorPortalRoom(floorNum int) *world.Room {
	t.mu.RLock()
	floor, exists := t.Floors[floorNum]
	t.mu.RUnlock()

	if !exists {
		return nil
	}

	return floor.GetPortalRoom()
}

// GetFloorStairsRoom returns the stairs room for a specific floor (going up)
// Returns nil if the floor doesn't exist or has no stairs room
func (t *Tower) GetFloorStairsRoom(floorNum int) *world.Room {
	t.mu.RLock()
	floor, exists := t.Floors[floorNum]
	t.mu.RUnlock()

	if !exists {
		return nil
	}

	return floor.GetStairsUp()
}

// GetOrGenerateFloorPortalRoom returns the portal room for a floor, generating it if needed
func (t *Tower) GetOrGenerateFloorPortalRoom(floorNum int) (*world.Room, error) {
	// Use GetFloor which generates on demand
	floor, err := t.GetFloor(floorNum)
	if err != nil {
		return nil, err
	}
	if floor == nil {
		return nil, nil
	}
	return floor.GetPortalRoom(), nil
}

// FindRoom searches all floors for a room by ID
// Returns nil if the room is not found on any floor
func (t *Tower) FindRoom(roomID string) *world.Room {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, floor := range t.Floors {
		if room := floor.GetRoom(roomID); room != nil {
			return room
		}
	}
	return nil
}

// tileTypeToRoomType converts WFC tile type to world room type
func tileTypeToRoomType(tt wfc.TileType) world.RoomType {
	switch tt {
	case wfc.TileCorridor:
		return world.RoomTypeCorridor
	case wfc.TileRoom:
		return world.RoomTypeRoom
	case wfc.TileDeadEnd:
		return world.RoomTypeRoom // Dead ends are just rooms
	case wfc.TileStairsUp, wfc.TileStairsDown:
		return world.RoomTypeStairs
	case wfc.TileTreasure:
		return world.RoomTypeTreasure
	case wfc.TileBoss:
		return world.RoomTypeBoss
	default:
		return world.RoomTypeRoom
	}
}

// generateRoomName creates a name for a room based on its type
func generateRoomName(tt wfc.TileType, floor, x, y int) string {
	switch tt {
	case wfc.TileCorridor:
		return fmt.Sprintf("Tower Corridor (Floor %d)", floor)
	case wfc.TileRoom:
		return fmt.Sprintf("Tower Chamber (Floor %d)", floor)
	case wfc.TileDeadEnd:
		return fmt.Sprintf("Dead End (Floor %d)", floor)
	case wfc.TileStairsUp:
		return fmt.Sprintf("Ascending Stairway (Floor %d)", floor)
	case wfc.TileStairsDown:
		return fmt.Sprintf("Descending Stairway (Floor %d)", floor)
	case wfc.TileTreasure:
		return fmt.Sprintf("Treasure Room (Floor %d)", floor)
	case wfc.TileBoss:
		return fmt.Sprintf("Boss Chamber (Floor %d)", floor)
	default:
		return fmt.Sprintf("Unknown Room (Floor %d)", floor)
	}
}

// generateRoomDescription creates a description for a room based on its type
func generateRoomDescription(tt wfc.TileType, floor int) string {
	switch tt {
	case wfc.TileCorridor:
		return "A narrow stone corridor stretches before you. Torches flicker on the walls, casting dancing shadows."
	case wfc.TileRoom:
		return "You stand in a chamber within the tower. The ancient stone walls are cold to the touch."
	case wfc.TileDeadEnd:
		return "The passage ends here in a small alcove. Dust motes drift in the dim light."
	case wfc.TileStairsUp:
		return "A spiral staircase ascends into the darkness above. The stone steps are worn smooth by countless travelers."
	case wfc.TileStairsDown:
		return "A spiral staircase descends from above. A shimmering portal offers quick travel to floors you've visited."
	case wfc.TileTreasure:
		return "This chamber holds the remnants of some forgotten hoard. Glittering objects catch the torchlight."
	case wfc.TileBoss:
		return "An ominous presence fills this grand chamber. The air is thick with danger."
	default:
		return "You are in a room within the tower."
	}
}

// lockStairsOnBossFloor locks the exit from the stairs room on boss floors
// Players must defeat the boss to get the key
func (t *Tower) lockStairsOnBossFloor(floor *Floor, floorNum int) {
	stairsRoom := floor.GetStairsUp()
	if stairsRoom == nil {
		return
	}

	keyID := GetBossKeyID(floorNum)

	// Lock the "up" exit from the stairs room
	stairsRoom.LockExit("up", keyID)

	// Add a "locked_door" feature so the room description can mention it
	stairsRoom.AddFeature("locked_door")
}

// GetBossKeyID returns the key ID for a given boss floor
func GetBossKeyID(floorNum int) string {
	return fmt.Sprintf("boss_key_floor_%d", floorNum)
}

// TreasureKeyID is the constant key ID for all treasure room doors
const TreasureKeyID = "treasure_key"

// lockTreasureRooms locks all treasure room entrances on the floor
// Players must use a purchasable treasure key to unlock them
func (t *Tower) lockTreasureRooms(floor *Floor) {
	rooms := floor.GetRooms()
	for _, room := range rooms {
		if room.Type != world.RoomTypeTreasure {
			continue
		}

		// Find all exits leading INTO the treasure room and lock the approach
		// We need to lock from the adjacent room's side
		for _, adjacentRoom := range rooms {
			if adjacentRoom.ID == room.ID {
				continue
			}
			// Check each direction from adjacent room
			for _, dir := range []string{"north", "south", "east", "west"} {
				exit := adjacentRoom.GetExit(dir)
				if exit == nil {
					continue
				}
				exitRoom, ok := exit.(*world.Room)
				if !ok {
					continue
				}
				if exitRoom.ID == room.ID {
					// This adjacent room has an exit to the treasure room - lock it
					adjacentRoom.LockExit(dir, TreasureKeyID)
				}
			}
		}
	}
}
