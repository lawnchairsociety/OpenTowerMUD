package world

import (
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// TowerInterface defines the methods needed from the tower package
// This avoids circular imports between world and tower packages
type TowerInterface interface {
	GetFloorPortalRoom(floorNum int) *Room
	GetFloorStairsRoom(floorNum int) *Room
	GetOrGenerateFloorPortalRoom(floorNum int) (*Room, error)
	HasFloor(floorNum int) bool
	FindRoom(roomID string) *Room
	GetAllRooms() map[string]*Room
}

// TowerManagerInterface defines methods for multi-tower management
// This is optional - World can work with either a single tower or a tower manager
// Methods use "ByString" suffix to accept string tower IDs (avoiding circular imports)
type TowerManagerInterface interface {
	// FindRoomByString searches all towers for a room by ID
	FindRoomByString(roomID string) *Room
	// FindRoomWithTowerID searches all towers for a room and returns both room and tower ID
	FindRoomWithTowerID(roomID string) (*Room, string)
	// GetAllCityRooms returns city rooms from all towers
	GetAllCityRooms() map[string]*Room
	// GetSpawnRoomByString returns the spawn room for a specific tower
	GetSpawnRoomByString(towerID string) *Room
	// GetFloorPortalRoomByString returns the portal room for a floor in a specific tower
	GetFloorPortalRoomByString(towerID string, floorNum int) *Room
	// IsInitializedByString returns true if a tower has been initialized
	IsInitializedByString(towerID string) bool
	// GetMaxFloorsForTower returns the max floors for a given tower
	GetMaxFloorsForTower(towerID string) int
}

type World struct {
	Rooms         map[string]*Room
	mu            sync.RWMutex
	worldFilePath string
	seed          int64
	readOnly      bool
	tower         TowerInterface         // Single tower (backward compatible)
	towerManager  TowerManagerInterface  // Multi-tower support (optional)
}

func NewWorld() *World {
	return &World{
		Rooms: make(map[string]*Room),
	}
}

// Initialize loads or generates a world using the default paths
func (w *World) Initialize(seed int64) {
	w.InitializeWithDirs(seed, "data/tower.yaml", "data/npcs", "data/mobs")
}

// InitializeWithPath loads or generates a world using a specific world file path
func (w *World) InitializeWithPath(seed int64, worldFilePath string) {
	w.InitializeWithDirs(seed, worldFilePath, "data/npcs", "data/mobs")
}

// InitializeWithDirs loads or generates a world using directory paths for NPCs and mobs.
func (w *World) InitializeWithDirs(seed int64, worldFilePath string, npcDirs ...string) {
	w.worldFilePath = worldFilePath
	w.seed = seed

	logger.Info("Initializing world", "seed", seed)

	// Load NPCs from YAML directories
	npcsConfig, err := npc.LoadNPCsFromDirectories(npcDirs...)
	if err != nil {
		logger.Warning("Failed to load NPCs/mobs", "error", err)
	} else {
		npcsByLocation := npcsConfig.GetNPCsByLocation()
		for location, npcs := range npcsByLocation {
			room := w.GetRoom(location)
			if room != nil {
				for _, n := range npcs {
					n.SetRoomID(room.ID)
					n.SetOriginalRoomID(room.ID)
					room.AddNPC(n)
					logger.Info("Added NPC to room", "npc", n.Name, "room", room.ID, "attackable", n.Attackable)
				}
			} else {
				logger.Warning("Room not found for NPC location", "location", location)
			}
		}
	}

	logger.Info("World initialized", "rooms", len(w.Rooms))
}

func (w *World) AddRoom(room *Room) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Rooms[room.ID] = room
}

func (w *World) GetRoom(id string) *Room {
	w.mu.RLock()
	room := w.Rooms[id]
	tower := w.tower
	towerManager := w.towerManager
	w.mu.RUnlock()

	if room != nil {
		return room
	}

	// Room not in base Rooms, check tower manager (multi-tower) first
	if towerManager != nil {
		if foundRoom := towerManager.FindRoomByString(id); foundRoom != nil {
			return foundRoom
		}
	}

	// Fall back to single tower
	if tower != nil {
		return tower.FindRoom(id)
	}
	return nil
}

func (w *World) GetStartingRoom() *Room {
	// Try the new prefixed human town square first, fall back to legacy name
	if room := w.GetRoom("human_town_square"); room != nil {
		return room
	}
	return w.GetRoom("town_square")
}

// GetStartingRoomForTower returns the spawn room for a specific tower
func (w *World) GetStartingRoomForTower(towerID string) *Room {
	w.mu.RLock()
	towerManager := w.towerManager
	w.mu.RUnlock()

	if towerManager != nil {
		return towerManager.GetSpawnRoomByString(towerID)
	}

	// Fall back to default starting room
	return w.GetStartingRoom()
}

func (w *World) GetAllRooms() []*Room {
	w.mu.RLock()
	defer w.mu.RUnlock()

	rooms := make([]*Room, 0, len(w.Rooms))
	for _, room := range w.Rooms {
		rooms = append(rooms, room)
	}

	// Include tower rooms
	if w.tower != nil {
		towerRooms := w.tower.GetAllRooms()
		for _, room := range towerRooms {
			rooms = append(rooms, room)
		}
	}

	return rooms
}

// SetReadOnly sets whether the world is in read-only mode
func (w *World) SetReadOnly(readOnly bool) {
	w.readOnly = readOnly
}

// IsReadOnly returns whether the world is in read-only mode
func (w *World) IsReadOnly() bool {
	return w.readOnly
}

// GetRoomCount returns the total number of rooms in the world
func (w *World) GetRoomCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.Rooms)
}

// GetSeed returns the world seed
func (w *World) GetSeed() int64 {
	return w.seed
}

// SetTower sets the tower interface for floor-based navigation
func (w *World) SetTower(tower TowerInterface) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tower = tower
}

// GetTower returns the tower interface
func (w *World) GetTower() TowerInterface {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.tower
}

// SetTowerManager sets the tower manager for multi-tower support
func (w *World) SetTowerManager(tm TowerManagerInterface) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.towerManager = tm
}

// GetTowerManager returns the tower manager
func (w *World) GetTowerManager() TowerManagerInterface {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.towerManager
}

// FindRoomWithTowerID searches all towers for a room and returns both room and tower ID.
// Returns empty string for tower ID if room not found or tower manager not set.
func (w *World) FindRoomWithTowerID(roomID string) (*Room, string) {
	w.mu.RLock()
	tm := w.towerManager
	w.mu.RUnlock()

	if tm == nil {
		return nil, ""
	}
	return tm.FindRoomWithTowerID(roomID)
}

// GetMaxFloorsForTower returns the max floors for a given tower.
// Returns 0 if tower manager not set or tower not found.
func (w *World) GetMaxFloorsForTower(towerID string) int {
	w.mu.RLock()
	tm := w.towerManager
	w.mu.RUnlock()

	if tm == nil {
		return 0
	}
	return tm.GetMaxFloorsForTower(towerID)
}

// GetFloorPortalRoom returns the portal room for a specific floor
// Returns nil if the floor doesn't exist or has no portal room
func (w *World) GetFloorPortalRoom(floorNum int) *Room {
	w.mu.RLock()
	tower := w.tower
	w.mu.RUnlock()

	if tower == nil {
		// No tower set, check if floor 0 (use starting room as portal)
		if floorNum == 0 {
			return w.GetStartingRoom()
		}
		return nil
	}

	return tower.GetFloorPortalRoom(floorNum)
}

// GetFloorStairsRoom returns the stairs room for a specific floor
// Returns nil if the floor doesn't exist or has no stairs room
func (w *World) GetFloorStairsRoom(floorNum int) *Room {
	w.mu.RLock()
	tower := w.tower
	w.mu.RUnlock()

	if tower == nil {
		return nil
	}

	return tower.GetFloorStairsRoom(floorNum)
}

// GetOrGenerateFloorPortalRoom returns the portal room for a floor, generating the floor if needed
func (w *World) GetOrGenerateFloorPortalRoom(floorNum int) (*Room, error) {
	w.mu.RLock()
	tower := w.tower
	w.mu.RUnlock()

	if tower == nil {
		// No tower set, check if floor 0 (use starting room as portal)
		if floorNum == 0 {
			return w.GetStartingRoom(), nil
		}
		return nil, nil
	}

	return tower.GetOrGenerateFloorPortalRoom(floorNum)
}
