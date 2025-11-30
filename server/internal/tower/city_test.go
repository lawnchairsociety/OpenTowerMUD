package tower

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCityFromYAML(t *testing.T) {
	// Find the data directory
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	cityFile := filepath.Join(dataDir, "city_rooms.yaml")
	config, err := LoadCityFromYAML(cityFile)
	if err != nil {
		t.Fatalf("LoadCityFromYAML failed: %v", err)
	}

	if config == nil {
		t.Fatal("Config is nil")
	}

	// Should have 16 rooms (10 original + 6 castle rooms)
	if len(config.Rooms) != 16 {
		t.Errorf("Expected 16 rooms, got %d", len(config.Rooms))
	}

	// Check for required rooms
	requiredRooms := []string{
		"town_square",
		"north_gate",
		"temple",
		"barracks",
		"market_street",
		"armory",
		"general_store",
		"training_hall",
		"tavern",
		"tower_entrance",
		"castle_gate",
		"guard_post",
		"castle_hall",
		"throne_room",
		"royal_library",
		"castle_courtyard",
	}

	for _, roomID := range requiredRooms {
		if _, ok := config.Rooms[roomID]; !ok {
			t.Errorf("Missing required room: %s", roomID)
		}
	}
}

func TestCreateCityFloor(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	cityFile := filepath.Join(dataDir, "city_rooms.yaml")
	config, err := LoadCityFromYAML(cityFile)
	if err != nil {
		t.Fatalf("LoadCityFromYAML failed: %v", err)
	}

	floor, err := CreateCityFloor(config)
	if err != nil {
		t.Fatalf("CreateCityFloor failed: %v", err)
	}

	// Should be floor 0
	if floor.Number != 0 {
		t.Errorf("Floor number = %d, want 0", floor.Number)
	}

	// Should have 16 rooms (10 original + 6 castle rooms)
	if floor.RoomCount() != 16 {
		t.Errorf("Room count = %d, want 16", floor.RoomCount())
	}

	// Should be marked as city
	if !floor.IsCity() {
		t.Error("Floor should be marked as city")
	}
}

func TestCityFloorRoomConnections(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	floor, err := LoadAndCreateCity(filepath.Join(dataDir, "city_rooms.yaml"))
	if err != nil {
		t.Fatalf("LoadAndCreateCity failed: %v", err)
	}

	// Check town square has correct exits
	townSquare := floor.GetRoom("town_square")
	if townSquare == nil {
		t.Fatal("Town square not found")
	}

	exits := townSquare.GetExits()
	expectedExits := map[string]string{
		"north": "North Gate",
		"south": "Market Street",
		"east":  "Temple of Light",
		"west":  "Castle Gate",
	}

	for dir, expectedName := range expectedExits {
		if exits[dir] != expectedName {
			t.Errorf("Town square exit %s = %q, want %q", dir, exits[dir], expectedName)
		}
	}
}

func TestCityFloorFeatures(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	floor, err := LoadAndCreateCity(filepath.Join(dataDir, "city_rooms.yaml"))
	if err != nil {
		t.Fatalf("LoadAndCreateCity failed: %v", err)
	}

	// Town square should have portal
	townSquare := floor.GetRoom("town_square")
	if !townSquare.HasFeature("portal") {
		t.Error("Town square should have portal feature")
	}

	// Temple should have altar
	temple := floor.GetRoom("temple")
	if !temple.HasFeature("altar") {
		t.Error("Temple should have altar feature")
	}

	// Tower entrance should have stairs_up
	entrance := floor.GetRoom("tower_entrance")
	if !entrance.HasFeature("stairs_up") {
		t.Error("Tower entrance should have stairs_up feature")
	}
}

func TestCityFloorPortalAndStairs(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	floor, err := LoadAndCreateCity(filepath.Join(dataDir, "city_rooms.yaml"))
	if err != nil {
		t.Fatalf("LoadAndCreateCity failed: %v", err)
	}

	// Portal room should be set
	if floor.PortalRoom != "town_square" {
		t.Errorf("PortalRoom = %q, want %q", floor.PortalRoom, "town_square")
	}

	// Stairs up should be set
	if floor.StairsUpRoom != "tower_entrance" {
		t.Errorf("StairsUpRoom = %q, want %q", floor.StairsUpRoom, "tower_entrance")
	}

	// GetPortalRoom should return the room
	portalRoom := floor.GetPortalRoom()
	if portalRoom == nil {
		t.Error("GetPortalRoom returned nil")
	} else if portalRoom.ID != "town_square" {
		t.Errorf("Portal room ID = %q, want %q", portalRoom.ID, "town_square")
	}

	// GetStairsUp should return the room
	stairsRoom := floor.GetStairsUp()
	if stairsRoom == nil {
		t.Error("GetStairsUp returned nil")
	} else if stairsRoom.ID != "tower_entrance" {
		t.Errorf("Stairs room ID = %q, want %q", stairsRoom.ID, "tower_entrance")
	}
}

func TestValidateCityFloor(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	floor, err := LoadAndCreateCity(filepath.Join(dataDir, "city_rooms.yaml"))
	if err != nil {
		t.Fatalf("LoadAndCreateCity failed: %v", err)
	}

	err = ValidateCityFloor(floor)
	if err != nil {
		t.Errorf("ValidateCityFloor failed: %v", err)
	}
}

func TestValidateCityFloorErrors(t *testing.T) {
	// Test nil floor
	err := ValidateCityFloor(nil)
	if err == nil {
		t.Error("ValidateCityFloor should fail for nil floor")
	}

	// Test wrong floor number
	wrongFloor := NewFloor(5)
	err = ValidateCityFloor(wrongFloor)
	if err == nil {
		t.Error("ValidateCityFloor should fail for non-zero floor")
	}
}

func TestGetSpawnRoom(t *testing.T) {
	if GetSpawnRoom() != "town_square" {
		t.Errorf("GetSpawnRoom() = %q, want %q", GetSpawnRoom(), "town_square")
	}
}

func TestGetTowerEntranceRoom(t *testing.T) {
	if GetTowerEntranceRoom() != "tower_entrance" {
		t.Errorf("GetTowerEntranceRoom() = %q, want %q", GetTowerEntranceRoom(), "tower_entrance")
	}
}

func TestHasFeature(t *testing.T) {
	features := []string{"portal", "altar", "stairs"}

	if !hasFeature(features, "portal") {
		t.Error("hasFeature should return true for 'portal'")
	}

	if !hasFeature(features, "altar") {
		t.Error("hasFeature should return true for 'altar'")
	}

	if hasFeature(features, "nonexistent") {
		t.Error("hasFeature should return false for 'nonexistent'")
	}

	if hasFeature(nil, "portal") {
		t.Error("hasFeature should return false for nil slice")
	}
}

func TestLoadCityFromYAMLNotFound(t *testing.T) {
	_, err := LoadCityFromYAML("/nonexistent/path/city.yaml")
	if err == nil {
		t.Error("LoadCityFromYAML should fail for non-existent file")
	}
}

func TestCreateCityFloorNilConfig(t *testing.T) {
	_, err := CreateCityFloor(nil)
	if err == nil {
		t.Error("CreateCityFloor should fail for nil config")
	}
}

func TestCreateCityFloorEmptyConfig(t *testing.T) {
	config := &CityConfig{Rooms: make(map[string]CityRoomDef)}
	_, err := CreateCityFloor(config)
	if err == nil {
		t.Error("CreateCityFloor should fail for empty config")
	}
}

// findDataDir looks for the data directory
func findDataDir() string {
	// Try relative paths from test location
	candidates := []string{
		"../../data",
		"../../../data",
		"data",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "city_rooms.yaml")); err == nil {
			return candidate
		}
	}

	return ""
}
