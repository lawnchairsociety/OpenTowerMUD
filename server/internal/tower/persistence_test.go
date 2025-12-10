package tower

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func TestSaveAndLoadTower(t *testing.T) {
	// Create a temp file for testing
	tmpDir := t.TempDir()
	towerFile := filepath.Join(tmpDir, "test_tower.yaml")

	// Create a tower with some floors
	tower := NewTower(12345)

	// Create floor 0 (city) manually
	cityFloor := NewFloor(0)
	townSquare := world.NewRoom("human_town_square", "Town Square", "A bustling plaza", world.RoomTypeCity)
	townSquare.Floor = 0
	townSquare.AddFeature("portal")
	cityFloor.AddRoom(townSquare)
	cityFloor.SetPortalRoom("human_town_square")

	towerEntrance := world.NewRoom("human_tower_entrance", "Tower Entrance", "The entrance to the endless tower", world.RoomTypeCity)
	towerEntrance.Floor = 0
	towerEntrance.AddFeature("stairs")
	cityFloor.AddRoom(towerEntrance)
	cityFloor.SetStairsUp("human_tower_entrance")

	// Connect rooms
	townSquare.AddExit("south", towerEntrance)
	towerEntrance.AddExit("north", townSquare)

	tower.SetFloor(0, cityFloor)

	// Generate floor 1
	floor1, err := tower.GetFloor(1)
	if err != nil {
		t.Fatalf("Failed to generate floor 1: %v", err)
	}

	// Save tower
	err = SaveTower(tower, towerFile)
	if err != nil {
		t.Fatalf("Failed to save tower: %v", err)
	}

	// Verify file exists
	if !TowerFileExists(towerFile) {
		t.Error("Tower file should exist after save")
	}

	// Load tower
	loadedTower, err := LoadTower(towerFile)
	if err != nil {
		t.Fatalf("Failed to load tower: %v", err)
	}

	// Verify seed
	if loadedTower.Seed != 12345 {
		t.Errorf("Seed = %d, want 12345", loadedTower.Seed)
	}

	// Verify highest floor
	if loadedTower.HighestFloor != tower.HighestFloor {
		t.Errorf("HighestFloor = %d, want %d", loadedTower.HighestFloor, tower.HighestFloor)
	}

	// Verify floor count
	if loadedTower.FloorCount() != tower.FloorCount() {
		t.Errorf("FloorCount = %d, want %d", loadedTower.FloorCount(), tower.FloorCount())
	}

	// Verify city floor rooms
	loadedCity := loadedTower.GetFloorIfExists(0)
	if loadedCity == nil {
		t.Fatal("City floor should exist")
	}

	loadedTownSquare := loadedCity.GetRoom("human_town_square")
	if loadedTownSquare == nil {
		t.Error("human_town_square should exist")
	} else {
		if loadedTownSquare.Name != "Town Square" {
			t.Errorf("human_town_square name = %s, want Town Square", loadedTownSquare.Name)
		}
		if !loadedTownSquare.HasFeature("portal") {
			t.Error("human_town_square should have portal feature")
		}
	}

	// Verify floor 1 room count
	loadedFloor1 := loadedTower.GetFloorIfExists(1)
	if loadedFloor1 == nil {
		t.Fatal("Floor 1 should exist")
	}
	if loadedFloor1.RoomCount() != floor1.RoomCount() {
		t.Errorf("Floor 1 room count = %d, want %d", loadedFloor1.RoomCount(), floor1.RoomCount())
	}

	// Verify exits are restored
	if loadedTownSquare != nil {
		southExit := loadedTownSquare.GetExit("south")
		if southExit == nil {
			t.Error("town_square should have south exit")
		}
	}
}

func TestTowerFileExists(t *testing.T) {
	// Non-existent file
	if TowerFileExists("/nonexistent/path/tower.yaml") {
		t.Error("TowerFileExists should return false for non-existent file")
	}

	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	if !TowerFileExists(tmpFile) {
		t.Error("TowerFileExists should return true for existing file")
	}
}

func TestLoadTowerNonExistent(t *testing.T) {
	_, err := LoadTower("/nonexistent/path/tower.yaml")
	if err == nil {
		t.Error("LoadTower should fail for non-existent file")
	}
}

func TestLoadTowerInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(tmpFile, []byte("this is not valid yaml: ["), 0644)

	_, err := LoadTower(tmpFile)
	if err == nil {
		t.Error("LoadTower should fail for invalid YAML")
	}
}

func TestRoomTypePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	towerFile := filepath.Join(tmpDir, "test_tower.yaml")

	tower := NewTower(42)

	// Create floor with different room types
	floor := NewFloor(1)
	floor.AddRoom(createTestRoom("corridor1", "Corridor", world.RoomTypeCorridor, 1))
	floor.AddRoom(createTestRoom("room1", "Room", world.RoomTypeRoom, 1))
	floor.AddRoom(createTestRoom("stairs1", "Stairs", world.RoomTypeStairs, 1))
	floor.AddRoom(createTestRoom("treasure1", "Treasure", world.RoomTypeTreasure, 1))
	floor.AddRoom(createTestRoom("boss1", "Boss", world.RoomTypeBoss, 1))
	tower.SetFloor(1, floor)

	// Save and load
	if err := SaveTower(tower, towerFile); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loaded, err := LoadTower(towerFile)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	loadedFloor := loaded.GetFloorIfExists(1)
	if loadedFloor == nil {
		t.Fatal("Floor 1 should exist")
	}

	// Check room types are preserved
	tests := []struct {
		id       string
		wantType world.RoomType
	}{
		{"corridor1", world.RoomTypeCorridor},
		{"room1", world.RoomTypeRoom},
		{"stairs1", world.RoomTypeStairs},
		{"treasure1", world.RoomTypeTreasure},
		{"boss1", world.RoomTypeBoss},
	}

	for _, tc := range tests {
		room := loadedFloor.GetRoom(tc.id)
		if room == nil {
			t.Errorf("Room %s not found", tc.id)
			continue
		}
		if room.Type != tc.wantType {
			t.Errorf("Room %s type = %v, want %v", tc.id, room.Type, tc.wantType)
		}
	}
}

func TestFeaturesPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	towerFile := filepath.Join(tmpDir, "test_tower.yaml")

	tower := NewTower(42)

	floor := NewFloor(1)
	room := world.NewRoom("test_room", "Test Room", "A test room", world.RoomTypeRoom)
	room.Floor = 1
	room.AddFeature("portal")
	room.AddFeature("stairs_up")
	room.AddFeature("treasure")
	floor.AddRoom(room)
	tower.SetFloor(1, floor)

	// Save and load
	if err := SaveTower(tower, towerFile); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loaded, err := LoadTower(towerFile)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	loadedRoom := loaded.GetRoom(1, "test_room")
	if loadedRoom == nil {
		t.Fatal("Room should exist")
	}

	if !loadedRoom.HasFeature("portal") {
		t.Error("Room should have portal feature")
	}
	if !loadedRoom.HasFeature("stairs_up") {
		t.Error("Room should have stairs_up feature")
	}
	if !loadedRoom.HasFeature("treasure") {
		t.Error("Room should have treasure feature")
	}
}

func createTestRoom(id, name string, roomType world.RoomType, floor int) *world.Room {
	room := world.NewRoom(id, name, "Test description", roomType)
	room.Floor = floor
	return room
}
