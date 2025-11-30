package tower

import (
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func TestNewFloor(t *testing.T) {
	floor := NewFloor(5)

	if floor.Number != 5 {
		t.Errorf("Number = %d, want 5", floor.Number)
	}
	if floor.Rooms == nil {
		t.Error("Rooms should not be nil")
	}
	if floor.Generated.IsZero() {
		t.Error("Generated time should be set")
	}
}

func TestFloorAddAndGetRoom(t *testing.T) {
	floor := NewFloor(1)
	room := world.NewRoom("test_room", "Test Room", "A test room", world.RoomTypeRoom)

	floor.AddRoom(room)

	got := floor.GetRoom("test_room")
	if got != room {
		t.Error("GetRoom did not return the added room")
	}

	// Non-existent room
	if floor.GetRoom("nonexistent") != nil {
		t.Error("GetRoom should return nil for non-existent room")
	}
}

func TestFloorGetRooms(t *testing.T) {
	floor := NewFloor(1)
	room1 := world.NewRoom("room1", "Room 1", "First room", world.RoomTypeRoom)
	room2 := world.NewRoom("room2", "Room 2", "Second room", world.RoomTypeRoom)

	floor.AddRoom(room1)
	floor.AddRoom(room2)

	rooms := floor.GetRooms()
	if len(rooms) != 2 {
		t.Errorf("GetRooms returned %d rooms, want 2", len(rooms))
	}
}

func TestFloorRoomCount(t *testing.T) {
	floor := NewFloor(1)

	if floor.RoomCount() != 0 {
		t.Errorf("RoomCount = %d, want 0", floor.RoomCount())
	}

	floor.AddRoom(world.NewRoom("r1", "R1", "Room 1", world.RoomTypeRoom))
	floor.AddRoom(world.NewRoom("r2", "R2", "Room 2", world.RoomTypeRoom))

	if floor.RoomCount() != 2 {
		t.Errorf("RoomCount = %d, want 2", floor.RoomCount())
	}
}

func TestFloorStairs(t *testing.T) {
	floor := NewFloor(5)
	stairsRoom := world.NewRoom("stairs", "Stairway", "A stairway", world.RoomTypeStairs)

	floor.AddRoom(stairsRoom)
	floor.SetStairsUp("stairs")
	floor.SetStairsDown("stairs")

	if floor.GetStairsUp() != stairsRoom {
		t.Error("GetStairsUp did not return correct room")
	}
	if floor.GetStairsDown() != stairsRoom {
		t.Error("GetStairsDown did not return correct room")
	}

	// Empty stairs
	emptyFloor := NewFloor(1)
	if emptyFloor.GetStairsUp() != nil {
		t.Error("GetStairsUp should return nil when not set")
	}
	if emptyFloor.GetStairsDown() != nil {
		t.Error("GetStairsDown should return nil when not set")
	}
}

func TestFloorPortal(t *testing.T) {
	floor := NewFloor(5)
	portalRoom := world.NewRoom("portal", "Portal Room", "A portal room", world.RoomTypeStairs)

	floor.AddRoom(portalRoom)
	floor.SetPortalRoom("portal")

	if floor.GetPortalRoom() != portalRoom {
		t.Error("GetPortalRoom did not return correct room")
	}

	// Empty portal
	emptyFloor := NewFloor(1)
	if emptyFloor.GetPortalRoom() != nil {
		t.Error("GetPortalRoom should return nil when not set")
	}
}

func TestFloorIsBossFloor(t *testing.T) {
	tests := []struct {
		floorNum int
		isBoss   bool
	}{
		{0, false},  // City
		{1, false},
		{5, false},
		{10, true},  // Boss
		{15, false},
		{20, true},  // Boss
		{30, true},  // Boss
		{100, true}, // Boss
	}

	for _, tc := range tests {
		floor := NewFloor(tc.floorNum)
		if floor.IsBossFloor() != tc.isBoss {
			t.Errorf("Floor %d: IsBossFloor() = %v, want %v", tc.floorNum, floor.IsBossFloor(), tc.isBoss)
		}
	}
}

func TestFloorIsCity(t *testing.T) {
	cityFloor := NewFloor(0)
	if !cityFloor.IsCity() {
		t.Error("Floor 0 should be city")
	}

	towerFloor := NewFloor(1)
	if towerFloor.IsCity() {
		t.Error("Floor 1 should not be city")
	}
}

func TestFloorGetDifficultyMultiplier(t *testing.T) {
	tests := []struct {
		floorNum   int
		wantMulti  float64
	}{
		{0, 1.0},
		{1, 1.1},
		{5, 1.5},
		{10, 2.0},
		{20, 3.0},
	}

	for _, tc := range tests {
		floor := NewFloor(tc.floorNum)
		got := floor.GetDifficultyMultiplier()
		if got != tc.wantMulti {
			t.Errorf("Floor %d: GetDifficultyMultiplier() = %v, want %v", tc.floorNum, got, tc.wantMulti)
		}
	}
}

func TestFloorString(t *testing.T) {
	tests := []struct {
		floorNum int
		want     string
	}{
		{0, "Ground Floor (City)"},
		{1, "Floor 1"},
		{5, "Floor 5"},
		{10, "Floor 10 (Boss)"},
		{20, "Floor 20 (Boss)"},
	}

	for _, tc := range tests {
		floor := NewFloor(tc.floorNum)
		if floor.String() != tc.want {
			t.Errorf("Floor %d: String() = %q, want %q", tc.floorNum, floor.String(), tc.want)
		}
	}
}
