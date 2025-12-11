package tower

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// TowerManager manages multiple towers in the game world.
type TowerManager struct {
	towers     map[TowerID]*Tower
	dataDir    string
	worldDir   string
	mobConfig  *npc.NPCsConfig
	itemConfig *items.ItemsConfig
	mu         sync.RWMutex
}

// NewTowerManager creates a new tower manager.
func NewTowerManager(dataDir string) *TowerManager {
	return &TowerManager{
		towers:  make(map[TowerID]*Tower),
		dataDir: dataDir,
	}
}

// SetMobConfig sets the mob configuration for spawning.
func (m *TowerManager) SetMobConfig(config *npc.NPCsConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mobConfig = config
}

// SetItemConfig sets the item configuration for loot spawning.
func (m *TowerManager) SetItemConfig(config *items.ItemsConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.itemConfig = config
}

// SetWorldDir sets the directory for world state persistence.
func (m *TowerManager) SetWorldDir(worldDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.worldDir = worldDir
}

// InitializeTower initializes a specific tower by loading its city and floors.
func (m *TowerManager) InitializeTower(id TowerID, seed int64) error {
	theme := GetTheme(id)
	if theme == nil {
		return fmt.Errorf("unknown tower ID: %s", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create the tower
	t := NewTower(seed)
	t.SetTowerID(string(id))
	t.SetDataDir(m.dataDir)
	t.SetUseStaticFloors(true)
	t.SetMaxFloors(theme.MaxFloors)

	// Set up spawners with tag filtering
	if m.mobConfig != nil {
		spawner := NewMobSpawnerWithTags(m.mobConfig, theme.MobTags)
		t.SetMobSpawner(spawner)
	}
	if m.itemConfig != nil {
		t.SetItemConfig(m.itemConfig)
	}

	// Load city floor (floor 0)
	cityFloor, err := LoadAndCreateCity(theme.CityFile)
	if err != nil {
		return fmt.Errorf("failed to load city for tower %s: %w", id, err)
	}
	t.SetFloor(0, cityFloor)

	// Preload all static floors
	floorsLoaded, err := t.PreloadStaticFloors(theme.MaxFloors)
	if err != nil {
		return fmt.Errorf("failed to preload floors for tower %s: %w", id, err)
	}

	// Connect floor 1 to city if floors were loaded
	if floorsLoaded > 0 {
		if entranceRoom := cityFloor.GetRoom(theme.TowerEntrance); entranceRoom != nil {
			if err := t.ConnectFloorToCity(entranceRoom); err != nil {
				return fmt.Errorf("failed to connect floor 1 to city for tower %s: %w", id, err)
			}
		}
	}

	m.towers[id] = t
	return nil
}

// GetTower returns a tower by ID.
func (m *TowerManager) GetTower(id TowerID) *Tower {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.towers[id]
}

// GetTheme returns the theme for a tower ID.
func (m *TowerManager) GetTheme(id TowerID) *TowerTheme {
	return GetTheme(id)
}

// FindRoom searches all towers for a room by ID.
// Returns the room and its tower ID, or nil if not found.
func (m *TowerManager) FindRoom(roomID string) (*world.Room, TowerID) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, t := range m.towers {
		if room := t.FindRoom(roomID); room != nil {
			return room, id
		}
	}
	return nil, ""
}

// FindRoomInTower searches a specific tower for a room by ID.
func (m *TowerManager) FindRoomInTower(id TowerID, roomID string) *world.Room {
	m.mu.RLock()
	t := m.towers[id]
	m.mu.RUnlock()

	if t == nil {
		return nil
	}
	return t.FindRoom(roomID)
}

// GetFloorPortalRoom returns the portal room for a specific floor in a tower.
func (m *TowerManager) GetFloorPortalRoom(id TowerID, floorNum int) *world.Room {
	m.mu.RLock()
	t := m.towers[id]
	m.mu.RUnlock()

	if t == nil {
		return nil
	}
	return t.GetFloorPortalRoom(floorNum)
}

// GetCityRooms returns all city rooms for a specific tower.
func (m *TowerManager) GetCityRooms(id TowerID) map[string]*world.Room {
	m.mu.RLock()
	t := m.towers[id]
	m.mu.RUnlock()

	if t == nil {
		return nil
	}

	cityFloor := t.GetFloorIfExists(0)
	if cityFloor == nil {
		return nil
	}
	return cityFloor.GetRooms()
}

// GetAllCityRooms returns all city rooms from all initialized towers.
func (m *TowerManager) GetAllCityRooms() map[string]*world.Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allRooms := make(map[string]*world.Room)
	for _, t := range m.towers {
		cityFloor := t.GetFloorIfExists(0)
		if cityFloor == nil {
			continue
		}
		for id, room := range cityFloor.GetRooms() {
			allRooms[id] = room
		}
	}
	return allRooms
}

// GetSpawnRoom returns the spawn room for a specific tower.
func (m *TowerManager) GetSpawnRoom(id TowerID) *world.Room {
	theme := GetTheme(id)
	if theme == nil {
		return nil
	}

	m.mu.RLock()
	t := m.towers[id]
	m.mu.RUnlock()

	if t == nil {
		return nil
	}

	cityFloor := t.GetFloorIfExists(0)
	if cityFloor == nil {
		return nil
	}
	return cityFloor.GetRoom(theme.SpawnRoom)
}

// GetInitializedTowers returns a list of all initialized tower IDs.
func (m *TowerManager) GetInitializedTowers() []TowerID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]TowerID, 0, len(m.towers))
	for id := range m.towers {
		ids = append(ids, id)
	}
	return ids
}

// IsInitialized returns true if a tower has been initialized.
func (m *TowerManager) IsInitialized(id TowerID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.towers[id]
	return exists
}

// GetMobSpawner returns the mob spawner for a specific tower.
func (m *TowerManager) GetMobSpawner(id TowerID) *MobSpawner {
	m.mu.RLock()
	t := m.towers[id]
	m.mu.RUnlock()

	if t == nil {
		return nil
	}
	return t.GetMobSpawner()
}

// ==================== world.TowerManagerInterface implementation ====================
// These methods use string tower IDs to match the interface and avoid circular imports

// FindRoomByString searches all towers for a room by ID (implements TowerManagerInterface)
func (m *TowerManager) FindRoomByString(roomID string) *world.Room {
	room, _ := m.FindRoom(roomID)
	return room
}

// FindRoomWithTowerID searches all towers for a room and returns both room and tower ID
func (m *TowerManager) FindRoomWithTowerID(roomID string) (*world.Room, string) {
	room, towerID := m.FindRoom(roomID)
	return room, string(towerID)
}

// GetMaxFloorsForTower returns the max floors for a given tower
func (m *TowerManager) GetMaxFloorsForTower(towerID string) int {
	theme := GetTheme(TowerID(towerID))
	if theme == nil {
		return 0
	}
	return theme.MaxFloors
}

// GetSpawnRoomByString returns the spawn room for a tower (implements TowerManagerInterface)
func (m *TowerManager) GetSpawnRoomByString(towerID string) *world.Room {
	return m.GetSpawnRoom(TowerID(towerID))
}

// GetFloorPortalRoomByString returns the portal room for a floor (implements TowerManagerInterface)
func (m *TowerManager) GetFloorPortalRoomByString(towerID string, floorNum int) *world.Room {
	return m.GetFloorPortalRoom(TowerID(towerID), floorNum)
}

// IsInitializedByString returns true if a tower is initialized (implements TowerManagerInterface)
func (m *TowerManager) IsInitializedByString(towerID string) bool {
	return m.IsInitialized(TowerID(towerID))
}

// ==================== World State Persistence ====================

// getTowerStateFile returns the file path for a tower's world state file.
func (m *TowerManager) getTowerStateFile(id TowerID) string {
	return filepath.Join(m.worldDir, string(id)+"_world.yaml")
}

// SaveAllTowers saves the world state of all towers to the world directory.
// Returns the number of towers saved and any error encountered.
func (m *TowerManager) SaveAllTowers() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.worldDir == "" {
		return 0, fmt.Errorf("world directory not configured")
	}

	// Ensure world directory exists
	if err := os.MkdirAll(m.worldDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create world directory: %w", err)
	}

	saved := 0
	for id, t := range m.towers {
		stateFile := m.getTowerStateFile(id)
		if err := SaveTower(t, stateFile); err != nil {
			return saved, fmt.Errorf("failed to save tower %s: %w", id, err)
		}
		saved++
	}
	return saved, nil
}

// SaveTowerState saves a single tower's world state.
func (m *TowerManager) SaveTowerState(id TowerID) error {
	m.mu.RLock()
	t := m.towers[id]
	worldDir := m.worldDir
	m.mu.RUnlock()

	if t == nil {
		return fmt.Errorf("tower %s not initialized", id)
	}
	if worldDir == "" {
		return fmt.Errorf("world directory not configured")
	}

	// Ensure world directory exists
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		return fmt.Errorf("failed to create world directory: %w", err)
	}

	stateFile := filepath.Join(worldDir, string(id)+"_world.yaml")
	return SaveTower(t, stateFile)
}

// LoadTowerState loads a tower's world state from the world directory.
// Returns true if state was loaded, false if no state file exists.
func (m *TowerManager) LoadTowerState(id TowerID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.worldDir == "" {
		return false, nil // No world dir configured, nothing to load
	}

	stateFile := m.getTowerStateFile(id)
	if !TowerFileExists(stateFile) {
		return false, nil // No saved state exists
	}

	loadedTower, err := LoadTower(stateFile)
	if err != nil {
		return false, fmt.Errorf("failed to load tower state from %s: %w", stateFile, err)
	}

	// Merge loaded state into existing tower (preserving city and config)
	if existingTower, exists := m.towers[id]; exists {
		// Copy loaded floor data (except floor 0 which is the city)
		for floorNum, floor := range loadedTower.Floors {
			if floorNum > 0 {
				existingTower.SetFloor(floorNum, floor)
			}
		}
		if loadedTower.HighestFloor > existingTower.HighestFloor {
			existingTower.HighestFloor = loadedTower.HighestFloor
		}
	} else {
		// No existing tower, use loaded tower directly
		m.towers[id] = loadedTower
	}

	return true, nil
}

// HasSavedState returns true if a tower has a saved world state file.
func (m *TowerManager) HasSavedState(id TowerID) bool {
	m.mu.RLock()
	worldDir := m.worldDir
	m.mu.RUnlock()

	if worldDir == "" {
		return false
	}
	return TowerFileExists(filepath.Join(worldDir, string(id)+"_world.yaml"))
}
