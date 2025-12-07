package tower

import (
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/wfc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func TestNewTower(t *testing.T) {
	tower := NewTower(42)

	if tower.Seed != 42 {
		t.Errorf("Seed = %d, want 42", tower.Seed)
	}
	if tower.Floors == nil {
		t.Error("Floors should not be nil")
	}
	if tower.HighestFloor != 0 {
		t.Errorf("HighestFloor = %d, want 0", tower.HighestFloor)
	}
}

func TestTowerSetAndGetFloor(t *testing.T) {
	tower := NewTower(42)
	floor := NewFloor(5)
	floor.AddRoom(world.NewRoom("test", "Test", "Test room", world.RoomTypeRoom))

	tower.SetFloor(5, floor)

	got := tower.GetFloorIfExists(5)
	if got != floor {
		t.Error("GetFloorIfExists did not return the set floor")
	}

	if tower.HighestFloor != 5 {
		t.Errorf("HighestFloor = %d, want 5", tower.HighestFloor)
	}
}

func TestTowerHasFloor(t *testing.T) {
	tower := NewTower(42)

	if tower.HasFloor(5) {
		t.Error("HasFloor(5) should be false before setting")
	}

	tower.SetFloor(5, NewFloor(5))

	if !tower.HasFloor(5) {
		t.Error("HasFloor(5) should be true after setting")
	}
}

func TestTowerGenerateFloor(t *testing.T) {
	tower := NewTower(42)

	// Floor 0 should fail (city must be set externally)
	_, err := tower.GetFloor(0)
	if err == nil {
		t.Error("GetFloor(0) should return error")
	}

	// Floor 1 should be generated
	floor, err := tower.GetFloor(1)
	if err != nil {
		t.Fatalf("GetFloor(1) failed: %v", err)
	}

	if floor.Number != 1 {
		t.Errorf("Floor number = %d, want 1", floor.Number)
	}

	if floor.RoomCount() < 10 {
		t.Errorf("Floor has too few rooms: %d", floor.RoomCount())
	}

	// Floor should have stairs
	if floor.StairsUpRoom == "" {
		t.Error("Floor should have StairsUpRoom set")
	}
}

func TestTowerGenerateFloorIdempotent(t *testing.T) {
	tower := NewTower(42)

	floor1, err := tower.GetFloor(1)
	if err != nil {
		t.Fatalf("First GetFloor(1) failed: %v", err)
	}

	floor2, err := tower.GetFloor(1)
	if err != nil {
		t.Fatalf("Second GetFloor(1) failed: %v", err)
	}

	if floor1 != floor2 {
		t.Error("GetFloor should return same floor on repeated calls")
	}
}

func TestTowerGetRoom(t *testing.T) {
	tower := NewTower(42)
	floor := NewFloor(5)
	room := world.NewRoom("test_room", "Test", "Test room", world.RoomTypeRoom)
	floor.AddRoom(room)
	tower.SetFloor(5, floor)

	got := tower.GetRoom(5, "test_room")
	if got != room {
		t.Error("GetRoom did not return correct room")
	}

	// Non-existent floor
	if tower.GetRoom(99, "test_room") != nil {
		t.Error("GetRoom should return nil for non-existent floor")
	}

	// Non-existent room
	if tower.GetRoom(5, "nonexistent") != nil {
		t.Error("GetRoom should return nil for non-existent room")
	}
}

func TestTowerGetAllRooms(t *testing.T) {
	tower := NewTower(42)

	floor1 := NewFloor(1)
	floor1.AddRoom(world.NewRoom("room1", "R1", "Room 1", world.RoomTypeRoom))
	floor1.AddRoom(world.NewRoom("room2", "R2", "Room 2", world.RoomTypeRoom))

	floor2 := NewFloor(2)
	floor2.AddRoom(world.NewRoom("room3", "R3", "Room 3", world.RoomTypeRoom))

	tower.SetFloor(1, floor1)
	tower.SetFloor(2, floor2)

	allRooms := tower.GetAllRooms()
	if len(allRooms) != 3 {
		t.Errorf("GetAllRooms returned %d rooms, want 3", len(allRooms))
	}
}

func TestTowerFloorCount(t *testing.T) {
	tower := NewTower(42)

	if tower.FloorCount() != 0 {
		t.Errorf("FloorCount = %d, want 0", tower.FloorCount())
	}

	tower.SetFloor(1, NewFloor(1))
	tower.SetFloor(5, NewFloor(5))

	if tower.FloorCount() != 2 {
		t.Errorf("FloorCount = %d, want 2", tower.FloorCount())
	}
}

func TestTowerConnectFloorToCity(t *testing.T) {
	tower := NewTower(42)

	// Generate floor 1
	floor1, err := tower.GetFloor(1)
	if err != nil {
		t.Fatalf("GetFloor(1) failed: %v", err)
	}

	// Create city entrance
	entrance := world.NewRoom("tower_entrance", "Tower Entrance", "The entrance to the tower", world.RoomTypeCity)

	// Connect
	err = tower.ConnectFloorToCity(entrance)
	if err != nil {
		t.Fatalf("ConnectFloorToCity failed: %v", err)
	}

	// Verify connections
	stairsDown := floor1.GetStairsDown()
	if stairsDown == nil {
		t.Fatal("Floor 1 should have stairs down")
	}

	// Check entrance has up exit
	upRoom := entrance.GetExit("up")
	if upRoom == nil {
		t.Error("Tower entrance should have 'up' exit")
	}

	// Check stairs has down exit
	downRoom := stairsDown.GetExit("down")
	if downRoom == nil {
		t.Error("Stairs should have 'down' exit")
	}
}

func TestTowerConnectFloorToCityNotGenerated(t *testing.T) {
	tower := NewTower(42)
	entrance := world.NewRoom("entrance", "Entrance", "Entrance", world.RoomTypeCity)

	err := tower.ConnectFloorToCity(entrance)
	if err == nil {
		t.Error("ConnectFloorToCity should fail when floor 1 not generated")
	}
}

func TestTowerStairsConnectBetweenFloors(t *testing.T) {
	tower := NewTower(42)

	// Generate floors 1, 2, 3
	floor1, _ := tower.GetFloor(1)
	floor2, _ := tower.GetFloor(2)
	floor3, _ := tower.GetFloor(3)

	// Check floor 2 connects to both floor 1 and floor 3
	stairs2Up := floor2.GetStairsUp()
	stairs2Down := floor2.GetStairsDown()

	if stairs2Up == nil || stairs2Down == nil {
		t.Fatal("Floor 2 should have stairs")
	}

	// Up from floor 2 should go to floor 3
	upExit := stairs2Up.GetExit("up")
	if upExit == nil {
		t.Error("Floor 2 stairs should have 'up' exit to floor 3")
	} else {
		upRoom := upExit.(*world.Room)
		if upRoom.Floor != 3 {
			t.Errorf("Up exit leads to floor %d, want 3", upRoom.Floor)
		}
	}

	// Down from floor 2 should go to floor 1
	downExit := stairs2Down.GetExit("down")
	if downExit == nil {
		t.Error("Floor 2 stairs should have 'down' exit to floor 1")
	} else {
		downRoom := downExit.(*world.Room)
		if downRoom.Floor != 1 {
			t.Errorf("Down exit leads to floor %d, want 1", downRoom.Floor)
		}
	}

	_ = floor1
	_ = floor3
}

func TestFloorHasSeparateStairs(t *testing.T) {
	tower := NewTower(42)

	// All floors should have separate rooms for up and down stairs
	for floorNum := 1; floorNum <= 3; floorNum++ {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		if floor.StairsUpRoom == floor.StairsDownRoom {
			t.Errorf("Floor %d should have separate rooms for up and down stairs, both are %s",
				floorNum, floor.StairsUpRoom)
		}

		upRoom := floor.GetStairsUp()
		downRoom := floor.GetStairsDown()

		if upRoom == nil {
			t.Errorf("Floor %d should have stairs up room", floorNum)
		}
		if downRoom == nil {
			t.Errorf("Floor %d should have stairs down room", floorNum)
		}

		if upRoom != nil && downRoom != nil && upRoom.ID == downRoom.ID {
			t.Errorf("Floor %d stairs up and down should be different rooms, both have ID %s",
				floorNum, upRoom.ID)
		}

		// Verify stairs up room has only stairs_up feature
		if upRoom != nil {
			if !upRoom.HasFeature("stairs_up") {
				t.Errorf("Floor %d stairs up room should have stairs_up feature", floorNum)
			}
			if upRoom.HasFeature("stairs_down") {
				t.Errorf("Floor %d stairs up room should NOT have stairs_down feature", floorNum)
			}
		}

		// Verify stairs down room has stairs_down and portal features
		if downRoom != nil {
			if !downRoom.HasFeature("stairs_down") {
				t.Errorf("Floor %d stairs down room should have stairs_down feature", floorNum)
			}
			if !downRoom.HasFeature("portal") {
				t.Errorf("Floor %d stairs down room should have portal feature", floorNum)
			}
			if downRoom.HasFeature("stairs_up") {
				t.Errorf("Floor %d stairs down room should NOT have stairs_up feature", floorNum)
			}
		}
	}
}

func TestTileTypeToRoomType(t *testing.T) {
	tests := []struct {
		tileType wfc.TileType
		want     world.RoomType
	}{
		{wfc.TileCorridor, world.RoomTypeCorridor},
		{wfc.TileRoom, world.RoomTypeRoom},
		{wfc.TileDeadEnd, world.RoomTypeRoom},
		{wfc.TileStairsUp, world.RoomTypeStairs},
		{wfc.TileStairsDown, world.RoomTypeStairs},
		{wfc.TileTreasure, world.RoomTypeTreasure},
		{wfc.TileBoss, world.RoomTypeBoss},
	}

	for _, tc := range tests {
		got := tileTypeToRoomType(tc.tileType)
		if got != tc.want {
			t.Errorf("tileTypeToRoomType(%v) = %v, want %v", tc.tileType, got, tc.want)
		}
	}
}

// TestTowerGenerateFloorAtScale tests floor generation at high floor numbers
func TestTowerGenerateFloorAtScale(t *testing.T) {
	tower := NewTower(12345)

	// Test floors at various scales: 10, 25, 50
	testFloors := []int{10, 25, 50}

	for _, floorNum := range testFloors {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		// Verify floor was created correctly
		if floor.Number != floorNum {
			t.Errorf("Floor number = %d, want %d", floor.Number, floorNum)
		}

		// Verify reasonable room count (20-50 rooms expected)
		roomCount := floor.RoomCount()
		if roomCount < 10 || roomCount > 100 {
			t.Errorf("Floor %d has unexpected room count: %d", floorNum, roomCount)
		}

		// Verify stairs exist
		if floor.StairsUpRoom == "" {
			t.Errorf("Floor %d should have StairsUpRoom set", floorNum)
		}
		if floor.StairsDownRoom == "" {
			t.Errorf("Floor %d should have StairsDownRoom set", floorNum)
		}

		// Verify boss floors have boss rooms
		if IsBossFloor(floorNum) {
			foundBoss := false
			for _, room := range floor.GetRooms() {
				if room.Type == world.RoomTypeBoss {
					foundBoss = true
					break
				}
			}
			if !foundBoss {
				t.Errorf("Boss floor %d should have a boss room", floorNum)
			}
		}

		// Verify scaling is applied correctly
		expectedTier := GetMobTier(floorNum)
		if expectedTier < 1 || expectedTier > 4 {
			t.Errorf("GetMobTier(%d) = %d, expected 1-4", floorNum, expectedTier)
		}

		expectedLootTier := GetLootTier(floorNum)
		if expectedLootTier < 1 || expectedLootTier > 5 {
			t.Errorf("GetLootTier(%d) = %d, expected 1-5", floorNum, expectedLootTier)
		}
	}

	// Verify highest floor tracking
	if tower.HighestFloor != 50 {
		t.Errorf("HighestFloor = %d, want 50", tower.HighestFloor)
	}

	// Verify floor count
	if tower.FloorCount() != 3 {
		t.Errorf("FloorCount = %d, want 3", tower.FloorCount())
	}
}

// TestTowerFloorConnectivity tests that all rooms on a floor are connected
func TestTowerFloorConnectivity(t *testing.T) {
	tower := NewTower(99999)

	floor, err := tower.GetFloor(5)
	if err != nil {
		t.Fatalf("GetFloor(5) failed: %v", err)
	}

	rooms := floor.GetRooms()
	if len(rooms) == 0 {
		t.Fatal("Floor has no rooms")
	}

	// BFS to verify all rooms are reachable from stairs room
	stairsRoom := floor.GetStairsDown()
	if stairsRoom == nil {
		t.Fatal("Floor has no stairs room")
	}

	visited := make(map[string]bool)
	queue := []*world.Room{stairsRoom}
	visited[stairsRoom.ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check all exits
		for _, dir := range []string{"north", "south", "east", "west"} {
			exit := current.GetExit(dir)
			if exit == nil {
				continue
			}
			nextRoom := exit.(*world.Room)
			// Only follow exits to rooms on the same floor
			if nextRoom.Floor == floor.Number && !visited[nextRoom.ID] {
				visited[nextRoom.ID] = true
				queue = append(queue, nextRoom)
			}
		}
	}

	// Verify all rooms are reachable
	for id := range rooms {
		if !visited[id] {
			t.Errorf("Room %s is not reachable from stairs room", id)
		}
	}
}

// TestTowerScalingFormulas tests the scaling formulas at various floor levels
func TestTowerScalingFormulas(t *testing.T) {
	tests := []struct {
		floor        int
		expectedTier int
		expectedLoot int
		isBossFloor  bool
	}{
		{1, 1, 1, false},
		{5, 1, 1, false},
		{6, 2, 2, false},
		{10, 2, 2, true},
		{11, 3, 3, false},
		{20, 3, 3, true},
		{21, 4, 4, false},
		{30, 4, 4, true},
		{31, 4, 5, false},
		{50, 4, 5, true},
	}

	for _, tc := range tests {
		if got := GetMobTier(tc.floor); got != tc.expectedTier {
			t.Errorf("GetMobTier(%d) = %d, want %d", tc.floor, got, tc.expectedTier)
		}

		if got := GetLootTier(tc.floor); got != tc.expectedLoot {
			t.Errorf("GetLootTier(%d) = %d, want %d", tc.floor, got, tc.expectedLoot)
		}

		if got := IsBossFloor(tc.floor); got != tc.isBossFloor {
			t.Errorf("IsBossFloor(%d) = %v, want %v", tc.floor, got, tc.isBossFloor)
		}
	}
}

// TestTowerDeterministicGeneration tests that same seed produces same floors
func TestTowerDeterministicGeneration(t *testing.T) {
	seed := int64(777)

	tower1 := NewTower(seed)
	tower2 := NewTower(seed)

	floor1a, _ := tower1.GetFloor(1)
	floor1b, _ := tower2.GetFloor(1)

	// Room counts should be identical
	if floor1a.RoomCount() != floor1b.RoomCount() {
		t.Errorf("Same seed produced different room counts: %d vs %d",
			floor1a.RoomCount(), floor1b.RoomCount())
	}

	// Stairs room IDs should be identical
	if floor1a.StairsUpRoom != floor1b.StairsUpRoom {
		t.Errorf("Same seed produced different stairs: %s vs %s",
			floor1a.StairsUpRoom, floor1b.StairsUpRoom)
	}
}

// BenchmarkFloorGeneration benchmarks floor generation performance
func BenchmarkFloorGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tower := NewTower(int64(i))
		_, err := tower.GetFloor(1)
		if err != nil {
			b.Fatalf("Floor generation failed: %v", err)
		}
	}
}

// BenchmarkMultiFloorGeneration benchmarks generating multiple floors
func BenchmarkMultiFloorGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tower := NewTower(int64(i))
		for floor := 1; floor <= 10; floor++ {
			_, err := tower.GetFloor(floor)
			if err != nil {
				b.Fatalf("Floor %d generation failed: %v", floor, err)
			}
		}
	}
}

// TestTowerManyFloors tests generating many floors to verify stability
func TestTowerManyFloors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping many-floors test in short mode")
	}

	tower := NewTower(54321)

	// Generate 20 floors to test stability
	for floorNum := 1; floorNum <= 20; floorNum++ {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		// Basic sanity checks
		if floor.Number != floorNum {
			t.Errorf("Floor %d: Number = %d", floorNum, floor.Number)
		}
		if floor.RoomCount() < 10 {
			t.Errorf("Floor %d: Too few rooms (%d)", floorNum, floor.RoomCount())
		}
		if floor.StairsUpRoom == "" {
			t.Errorf("Floor %d: Missing StairsUpRoom", floorNum)
		}
	}

	// Verify all floors were tracked
	if tower.FloorCount() != 20 {
		t.Errorf("FloorCount = %d, want 20", tower.FloorCount())
	}
	if tower.HighestFloor != 20 {
		t.Errorf("HighestFloor = %d, want 20", tower.HighestFloor)
	}

	// Verify total room count is reasonable (20 floors * ~30 rooms = ~600)
	allRooms := tower.GetAllRooms()
	if len(allRooms) < 200 || len(allRooms) > 2000 {
		t.Errorf("Total rooms = %d, expected 200-2000", len(allRooms))
	}
}

// TestBossFloorLockedStairs tests that boss floors have locked stairs
func TestBossFloorLockedStairs(t *testing.T) {
	tower := NewTower(12345)

	// Floor 10 is a boss floor
	floor, err := tower.GetFloor(10)
	if err != nil {
		t.Fatalf("GetFloor(10) failed: %v", err)
	}

	// Verify floor 10 is indeed a boss floor
	if !IsBossFloor(10) {
		t.Fatal("Floor 10 should be a boss floor")
	}

	// Get the stairs room
	stairsRoom := floor.GetStairsUp()
	if stairsRoom == nil {
		t.Fatal("Boss floor should have stairs room")
	}

	// Verify the "up" exit is locked
	if !stairsRoom.IsExitLocked("up") {
		t.Error("Stairs 'up' exit should be locked on boss floors")
	}

	// Verify the correct key is required
	expectedKeyID := GetBossKeyID(10)
	actualKeyID := stairsRoom.GetExitKeyRequired("up")
	if actualKeyID != expectedKeyID {
		t.Errorf("Expected key %s, got %s", expectedKeyID, actualKeyID)
	}

	// Verify locked_door feature is present
	if !stairsRoom.HasFeature("locked_door") {
		t.Error("Stairs room should have 'locked_door' feature")
	}
}

// TestNonBossFloorNoLockedStairs tests that non-boss floors have unlocked stairs
func TestNonBossFloorNoLockedStairs(t *testing.T) {
	tower := NewTower(12345)

	// Floor 5 is not a boss floor
	floor, err := tower.GetFloor(5)
	if err != nil {
		t.Fatalf("GetFloor(5) failed: %v", err)
	}

	// Verify floor 5 is not a boss floor
	if IsBossFloor(5) {
		t.Fatal("Floor 5 should not be a boss floor")
	}

	// Get the stairs room
	stairsRoom := floor.GetStairsUp()
	if stairsRoom == nil {
		t.Fatal("Floor should have stairs room")
	}

	// Verify the "up" exit is NOT locked
	if stairsRoom.IsExitLocked("up") {
		t.Error("Stairs 'up' exit should NOT be locked on non-boss floors")
	}
}

// TestGetBossKeyID tests the key ID generation format
func TestGetBossKeyID(t *testing.T) {
	tests := []struct {
		floorNum int
		expected string
	}{
		{10, "boss_key_floor_10"},
		{20, "boss_key_floor_20"},
		{30, "boss_key_floor_30"},
		{50, "boss_key_floor_50"},
		{100, "boss_key_floor_100"},
	}

	for _, tc := range tests {
		got := GetBossKeyID(tc.floorNum)
		if got != tc.expected {
			t.Errorf("GetBossKeyID(%d) = %s, want %s", tc.floorNum, got, tc.expected)
		}
	}
}

// TestMultipleBossFloorsLocked tests that all boss floors have locked stairs
func TestMultipleBossFloorsLocked(t *testing.T) {
	tower := NewTower(99999)

	bossFloors := []int{10, 20, 30}

	for _, floorNum := range bossFloors {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		stairsRoom := floor.GetStairsUp()
		if stairsRoom == nil {
			t.Fatalf("Floor %d: Missing stairs room", floorNum)
		}

		if !stairsRoom.IsExitLocked("up") {
			t.Errorf("Floor %d: Stairs 'up' should be locked", floorNum)
		}

		expectedKeyID := GetBossKeyID(floorNum)
		actualKeyID := stairsRoom.GetExitKeyRequired("up")
		if actualKeyID != expectedKeyID {
			t.Errorf("Floor %d: Expected key %s, got %s", floorNum, expectedKeyID, actualKeyID)
		}
	}
}

// TestTreasureRoomsLocked tests that treasure room entrances are locked
func TestTreasureRoomsLocked(t *testing.T) {
	tower := NewTower(12345)

	// Generate a floor
	floor, err := tower.GetFloor(5)
	if err != nil {
		t.Fatalf("GetFloor(5) failed: %v", err)
	}

	// Find treasure rooms
	var treasureRooms []*world.Room
	for _, room := range floor.GetRooms() {
		if room.Type == world.RoomTypeTreasure {
			treasureRooms = append(treasureRooms, room)
		}
	}

	if len(treasureRooms) == 0 {
		t.Skip("No treasure rooms generated on this floor")
	}

	// For each treasure room, verify there's a locked entrance
	for _, treasureRoom := range treasureRooms {
		// Find rooms that have exits to this treasure room
		foundLockedEntrance := false
		for _, room := range floor.GetRooms() {
			if room.ID == treasureRoom.ID {
				continue
			}
			for _, dir := range []string{"north", "south", "east", "west"} {
				exit := room.GetExit(dir)
				if exit == nil {
					continue
				}
				exitRoom, ok := exit.(*world.Room)
				if !ok {
					continue
				}
				if exitRoom.ID == treasureRoom.ID {
					// This room has an exit to the treasure room
					if room.IsExitLocked(dir) {
						foundLockedEntrance = true
						// Verify correct key required
						if room.GetExitKeyRequired(dir) != TreasureKeyID {
							t.Errorf("Treasure room entrance requires wrong key: %s", room.GetExitKeyRequired(dir))
						}
					}
				}
			}
		}
		if !foundLockedEntrance {
			t.Errorf("Treasure room %s has no locked entrance", treasureRoom.ID)
		}
	}
}

// TestTreasureKeyID tests the treasure key constant
func TestTreasureKeyID(t *testing.T) {
	if TreasureKeyID != "treasure_key" {
		t.Errorf("TreasureKeyID = %s, want treasure_key", TreasureKeyID)
	}
}

// TestHasMerchant tests which floors should have merchants
func TestHasMerchant(t *testing.T) {
	tests := []struct {
		floor    int
		expected bool
	}{
		{0, false},  // City - no merchant (has proper shop)
		{1, false},  // Not a merchant floor
		{4, false},  // Not a merchant floor
		{5, true},   // Merchant floor (5 % 5 == 0)
		{6, false},  // Not a merchant floor
		{10, true},  // Merchant floor (10 % 5 == 0)
		{15, true},  // Merchant floor
		{20, true},  // Merchant floor
		{25, true},  // Merchant floor
		{13, false}, // Not a merchant floor
	}

	for _, tc := range tests {
		got := HasMerchant(tc.floor)
		if got != tc.expected {
			t.Errorf("HasMerchant(%d) = %v, want %v", tc.floor, got, tc.expected)
		}
	}
}

// TestMerchantSpawnsOnCorrectFloors tests that merchants spawn on correct floors
func TestMerchantSpawnsOnCorrectFloors(t *testing.T) {
	tower := NewTower(12345)

	// Test floor 5 - should have merchant
	floor5, err := tower.GetFloor(5)
	if err != nil {
		t.Fatalf("GetFloor(5) failed: %v", err)
	}

	// Merchant spawns in the portal room (entry point)
	portalRoom5 := floor5.GetPortalRoom()
	if portalRoom5 == nil {
		t.Fatal("Floor 5 should have portal room")
	}

	if !portalRoom5.HasFeature("merchant") {
		t.Error("Floor 5 portal room should have 'merchant' feature")
	}

	// Check for the merchant NPC
	var foundMerchant bool
	for _, npc := range portalRoom5.NPCs {
		if npc.Name == "crusty old merchant" {
			foundMerchant = true
			break
		}
	}
	if !foundMerchant {
		t.Error("Floor 5 should have crusty old merchant NPC")
	}

	// Test floor 3 - should NOT have merchant
	floor3, err := tower.GetFloor(3)
	if err != nil {
		t.Fatalf("GetFloor(3) failed: %v", err)
	}

	portalRoom3 := floor3.GetPortalRoom()
	if portalRoom3 == nil {
		t.Fatal("Floor 3 should have portal room")
	}

	if portalRoom3.HasFeature("merchant") {
		t.Error("Floor 3 portal room should NOT have 'merchant' feature")
	}
}

// TestMerchantNPCProperties tests the merchant NPC has correct properties
func TestMerchantNPCProperties(t *testing.T) {
	tower := NewTower(54321)

	floor, err := tower.GetFloor(10) // Floor 10 should have merchant
	if err != nil {
		t.Fatalf("GetFloor(10) failed: %v", err)
	}

	// Merchant spawns in the portal room (entry point)
	portalRoom := floor.GetPortalRoom()
	if portalRoom == nil {
		t.Fatal("Floor should have portal room")
	}

	var merchant *npc.NPC
	for _, n := range portalRoom.NPCs {
		if n.Name == "crusty old merchant" {
			merchant = n
			break
		}
	}

	if merchant == nil {
		t.Fatal("Merchant NPC not found")
	}

	// Verify merchant is not attackable
	if merchant.Attackable {
		t.Error("Merchant should not be attackable")
	}

	// Verify merchant is not aggressive
	if merchant.Aggressive {
		t.Error("Merchant should not be aggressive")
	}
}

// TestFloorConnectivityMultipleSeeds tests floor connectivity across different seeds
// This ensures the WFC algorithm consistently produces connected floors
func TestFloorConnectivityMultipleSeeds(t *testing.T) {
	seeds := []int64{1, 42, 123, 456, 789, 1000, 9999, 12345, 54321, 99999}

	for _, seed := range seeds {
		tower := NewTower(seed)

		// Test floors 1, 5, and 10 (regular, mid, and boss)
		for _, floorNum := range []int{1, 5, 10} {
			floor, err := tower.GetFloor(floorNum)
			if err != nil {
				t.Fatalf("Seed %d, Floor %d: GetFloor failed: %v", seed, floorNum, err)
			}

			// Verify connectivity from entrance (stairs down)
			if !verifyFloorConnectivity(floor) {
				t.Errorf("Seed %d, Floor %d: Not all rooms are connected from entrance", seed, floorNum)
			}
		}
	}
}

// TestAllRoomsReachableFromEntrance verifies BFS traversal from entrance reaches all rooms
func TestAllRoomsReachableFromEntrance(t *testing.T) {
	tower := NewTower(77777)

	for floorNum := 1; floorNum <= 5; floorNum++ {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		entrance := floor.GetStairsDown()
		if entrance == nil {
			t.Fatalf("Floor %d: No entrance (stairs down) room", floorNum)
		}

		rooms := floor.GetRooms()
		visited := bfsTraverseFloor(entrance, floor.Number)

		unreachable := []string{}
		for id := range rooms {
			if !visited[id] {
				unreachable = append(unreachable, id)
			}
		}

		if len(unreachable) > 0 {
			t.Errorf("Floor %d: %d rooms unreachable from entrance: %v",
				floorNum, len(unreachable), unreachable)
		}
	}
}

// TestSpecialRoomsReachable verifies stairs up, boss room, and treasure rooms are reachable
func TestSpecialRoomsReachable(t *testing.T) {
	tower := NewTower(88888)

	// Test a boss floor (10)
	floor, err := tower.GetFloor(10)
	if err != nil {
		t.Fatalf("GetFloor(10) failed: %v", err)
	}

	entrance := floor.GetStairsDown()
	if entrance == nil {
		t.Fatal("Floor 10: No entrance room")
	}

	visited := bfsTraverseFloor(entrance, floor.Number)

	// Verify stairs up is reachable
	stairsUp := floor.GetStairsUp()
	if stairsUp != nil && !visited[stairsUp.ID] {
		t.Error("Stairs up room is not reachable from entrance")
	}

	// Verify boss room is reachable (if exists)
	for _, room := range floor.GetRooms() {
		if room.Type == world.RoomTypeBoss && !visited[room.ID] {
			t.Errorf("Boss room %s is not reachable from entrance", room.ID)
		}
	}

	// Verify treasure rooms are reachable
	for _, room := range floor.GetRooms() {
		if room.Type == world.RoomTypeTreasure && !visited[room.ID] {
			t.Errorf("Treasure room %s is not reachable from entrance", room.ID)
		}
	}
}

// TestBidirectionalConnections verifies all room connections are bidirectional
func TestBidirectionalConnections(t *testing.T) {
	tower := NewTower(33333)

	floor, err := tower.GetFloor(3)
	if err != nil {
		t.Fatalf("GetFloor(3) failed: %v", err)
	}

	rooms := floor.GetRooms()
	opposites := map[string]string{
		"north": "south",
		"south": "north",
		"east":  "west",
		"west":  "east",
	}

	for _, room := range rooms {
		for dir, opposite := range opposites {
			exit := room.GetExit(dir)
			if exit == nil {
				continue
			}
			neighbor, ok := exit.(*world.Room)
			if !ok || neighbor.Floor != floor.Number {
				continue // Skip cross-floor connections
			}

			// Verify the neighbor has a reverse connection
			reverseExit := neighbor.GetExit(opposite)
			if reverseExit == nil {
				t.Errorf("Room %s has %s exit to %s, but %s has no %s exit back",
					room.ID, dir, neighbor.ID, neighbor.ID, opposite)
				continue
			}
			reverseRoom, ok := reverseExit.(*world.Room)
			if !ok {
				continue
			}
			if reverseRoom.ID != room.ID {
				t.Errorf("Room %s has %s exit to %s, but %s's %s exit goes to %s instead of back",
					room.ID, dir, neighbor.ID, neighbor.ID, opposite, reverseRoom.ID)
			}
		}
	}
}

// TestNoIsolatedRoomClusters verifies there are no isolated clusters of rooms
func TestNoIsolatedRoomClusters(t *testing.T) {
	tower := NewTower(44444)

	for floorNum := 1; floorNum <= 3; floorNum++ {
		floor, err := tower.GetFloor(floorNum)
		if err != nil {
			t.Fatalf("GetFloor(%d) failed: %v", floorNum, err)
		}

		rooms := floor.GetRooms()
		if len(rooms) == 0 {
			t.Fatalf("Floor %d has no rooms", floorNum)
		}

		// Start from any room and count reachable rooms
		var startRoom *world.Room
		for _, r := range rooms {
			startRoom = r
			break
		}

		visited := bfsTraverseFloor(startRoom, floor.Number)

		if len(visited) != len(rooms) {
			t.Errorf("Floor %d: Found %d connected rooms but floor has %d total rooms (isolated cluster detected)",
				floorNum, len(visited), len(rooms))
		}
	}
}

// verifyFloorConnectivity checks if all rooms on a floor are reachable from the entrance
func verifyFloorConnectivity(floor *Floor) bool {
	rooms := floor.GetRooms()
	if len(rooms) == 0 {
		return true
	}

	entrance := floor.GetStairsDown()
	if entrance == nil {
		return false
	}

	visited := bfsTraverseFloor(entrance, floor.Number)
	return len(visited) == len(rooms)
}

// bfsTraverseFloor performs BFS from a starting room and returns visited room IDs
func bfsTraverseFloor(start *world.Room, floorNum int) map[string]bool {
	visited := make(map[string]bool)
	queue := []*world.Room{start}
	visited[start.ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dir := range []string{"north", "south", "east", "west"} {
			exit := current.GetExit(dir)
			if exit == nil {
				continue
			}
			nextRoom, ok := exit.(*world.Room)
			if !ok {
				continue
			}
			// Only traverse rooms on the same floor
			if nextRoom.Floor != floorNum {
				continue
			}
			if visited[nextRoom.ID] {
				continue
			}
			visited[nextRoom.ID] = true
			queue = append(queue, nextRoom)
		}
	}

	return visited
}
