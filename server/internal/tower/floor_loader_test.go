package tower

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func TestLoadFloorFromYAML(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "floor_1.yaml")

	yamlContent := `
floor: 1
tower: human
generated_seed: 43
stairs_up: human_f1_r10_4
stairs_down: human_f1_r13_6
portal_room: human_f1_r13_6
rooms:
  human_f1_r10_4:
    name: Ascending Stairway (Floor 1)
    description: A spiral staircase ascends into the darkness above.
    type: stairs_up
    features:
      - stairs_up
    exits:
      west: human_f1_r9_4
  human_f1_r9_4:
    name: Tower Chamber (Floor 1)
    description: You stand in a chamber within the tower.
    type: room
    exits:
      east: human_f1_r10_4
      south: human_f1_r9_5
  human_f1_r9_5:
    name: Tower Corridor (Floor 1)
    description: A narrow stone corridor stretches before you.
    type: corridor
    exits:
      north: human_f1_r9_4
  human_f1_r13_6:
    name: Descending Stairway (Floor 1)
    description: A spiral staircase descends from above.
    type: stairs_down
    features:
      - stairs_down
      - portal
    exits: {}
`

	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load the floor
	floor, err := LoadFloorFromYAML(testFile)
	if err != nil {
		t.Fatalf("LoadFloorFromYAML failed: %v", err)
	}

	// Verify floor metadata
	if floor.Number != 1 {
		t.Errorf("Expected floor number 1, got %d", floor.Number)
	}

	// Verify room count
	if floor.RoomCount() != 4 {
		t.Errorf("Expected 4 rooms, got %d", floor.RoomCount())
	}

	// Verify stairs up room
	stairsUp := floor.GetStairsUp()
	if stairsUp == nil {
		t.Error("Expected stairs up room to be set")
	} else {
		if stairsUp.ID != "human_f1_r10_4" {
			t.Errorf("Expected stairs up room ID 'human_f1_r10_4', got '%s'", stairsUp.ID)
		}
		if !stairsUp.HasFeature("stairs_up") {
			t.Error("Expected stairs up room to have 'stairs_up' feature")
		}
		if stairsUp.Type != world.RoomTypeStairs {
			t.Errorf("Expected stairs room type, got %v", stairsUp.Type)
		}
	}

	// Verify stairs down room
	stairsDown := floor.GetStairsDown()
	if stairsDown == nil {
		t.Error("Expected stairs down room to be set")
	} else {
		if stairsDown.ID != "human_f1_r13_6" {
			t.Errorf("Expected stairs down room ID 'human_f1_r13_6', got '%s'", stairsDown.ID)
		}
		if !stairsDown.HasFeature("stairs_down") {
			t.Error("Expected stairs down room to have 'stairs_down' feature")
		}
		if !stairsDown.HasFeature("portal") {
			t.Error("Expected stairs down room to have 'portal' feature")
		}
	}

	// Verify portal room
	portalRoom := floor.GetPortalRoom()
	if portalRoom == nil {
		t.Error("Expected portal room to be set")
	} else if portalRoom.ID != "human_f1_r13_6" {
		t.Errorf("Expected portal room ID 'human_f1_r13_6', got '%s'", portalRoom.ID)
	}

	// Verify exits are linked correctly
	room := floor.GetRoom("human_f1_r10_4")
	if room == nil {
		t.Fatal("Expected room 'human_f1_r10_4' to exist")
	}

	westExit := room.GetExit("west")
	if westExit == nil {
		t.Error("Expected 'west' exit from stairs up room")
	} else {
		westRoom, ok := westExit.(*world.Room)
		if !ok {
			t.Error("Expected west exit to be a *world.Room")
		} else if westRoom.ID != "human_f1_r9_4" {
			t.Errorf("Expected west exit to lead to 'human_f1_r9_4', got '%s'", westRoom.ID)
		}
	}

	// Verify corridor room type
	corridor := floor.GetRoom("human_f1_r9_5")
	if corridor == nil {
		t.Fatal("Expected corridor room to exist")
	}
	if corridor.Type != world.RoomTypeCorridor {
		t.Errorf("Expected corridor room type, got %v", corridor.Type)
	}
}

func TestLoadFloorFromYAML_FileNotFound(t *testing.T) {
	_, err := LoadFloorFromYAML("/nonexistent/path/floor_1.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestFloorFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	// File doesn't exist yet
	if FloorFileExists(testFile) {
		t.Error("Expected FloorFileExists to return false for nonexistent file")
	}

	// Create the file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Now it exists
	if !FloorFileExists(testFile) {
		t.Error("Expected FloorFileExists to return true for existing file")
	}
}

func TestParseRoomType(t *testing.T) {
	tests := []struct {
		input    string
		expected world.RoomType
	}{
		{"corridor", world.RoomTypeCorridor},
		{"room", world.RoomTypeRoom},
		{"dead_end", world.RoomTypeRoom},
		{"stairs_up", world.RoomTypeStairs},
		{"stairs_down", world.RoomTypeStairs},
		{"treasure", world.RoomTypeTreasure},
		{"boss", world.RoomTypeBoss},
		{"unknown", world.RoomTypeRoom},
	}

	for _, tt := range tests {
		got := parseRoomType(tt.input)
		if got != tt.expected {
			t.Errorf("parseRoomType(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
