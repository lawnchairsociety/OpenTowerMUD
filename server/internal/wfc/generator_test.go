package wfc

import "testing"

func TestDefaultFloorConfig(t *testing.T) {
	tests := []struct {
		floorNum      int
		expectBoss    bool
		minTreasure   int
	}{
		{1, false, 1},
		{5, false, 2},
		{10, true, 3},
		{20, true, 3}, // Capped at 3
		{15, false, 3},
	}

	for _, tc := range tests {
		cfg := DefaultFloorConfig(tc.floorNum, 42)

		if cfg.FloorNumber != tc.floorNum {
			t.Errorf("Floor %d: FloorNumber = %d", tc.floorNum, cfg.FloorNumber)
		}

		if cfg.IsBossFloor != tc.expectBoss {
			t.Errorf("Floor %d: IsBossFloor = %v, want %v", tc.floorNum, cfg.IsBossFloor, tc.expectBoss)
		}

		if cfg.TreasureCount < tc.minTreasure {
			t.Errorf("Floor %d: TreasureCount = %d, want >= %d", tc.floorNum, cfg.TreasureCount, tc.minTreasure)
		}

		if cfg.MinRooms != 20 {
			t.Errorf("Floor %d: MinRooms = %d, want 20", tc.floorNum, cfg.MinRooms)
		}

		if cfg.MaxRooms != 50 {
			t.Errorf("Floor %d: MaxRooms = %d, want 50", tc.floorNum, cfg.MaxRooms)
		}
	}
}

func TestGeneratorGenerate(t *testing.T) {
	config := DefaultFloorConfig(1, 42)
	config.MinRooms = 10 // Lower for faster tests
	config.MaxRooms = 20

	gen := NewGenerator(config)
	floor, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if floor == nil {
		t.Fatal("Generate() returned nil floor")
	}

	if floor.FloorNumber != 1 {
		t.Errorf("FloorNumber = %d, want 1", floor.FloorNumber)
	}

	if len(floor.Tiles) < config.MinRooms {
		t.Errorf("Too few tiles: %d < %d", len(floor.Tiles), config.MinRooms)
	}

	if len(floor.Tiles) > config.MaxRooms {
		t.Errorf("Too many tiles: %d > %d", len(floor.Tiles), config.MaxRooms)
	}

	// Floor 1 should have stairs up
	if floor.StairsUpTile == nil {
		t.Error("Floor 1 should have StairsUpTile")
	}
}

func TestGeneratorBossFloor(t *testing.T) {
	config := DefaultFloorConfig(10, 42)
	config.MinRooms = 10
	config.MaxRooms = 20

	gen := NewGenerator(config)
	floor, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if !config.IsBossFloor {
		t.Error("Floor 10 should be a boss floor")
	}

	if floor.BossTile == nil {
		t.Error("Boss floor should have BossTile")
	}

	if floor.BossTile.Type != TileBoss {
		t.Errorf("BossTile type = %s, want boss", floor.BossTile.Type)
	}
}

func TestGeneratorDeterministic(t *testing.T) {
	config1 := DefaultFloorConfig(5, 42)
	config1.MinRooms = 10
	config1.MaxRooms = 20

	config2 := DefaultFloorConfig(5, 42)
	config2.MinRooms = 10
	config2.MaxRooms = 20

	gen1 := NewGenerator(config1)
	gen2 := NewGenerator(config2)

	floor1, err1 := gen1.Generate()
	floor2, err2 := gen2.Generate()

	if (err1 == nil) != (err2 == nil) {
		t.Fatalf("Different error states: %v vs %v", err1, err2)
	}

	if err1 != nil {
		return
	}

	if len(floor1.Tiles) != len(floor2.Tiles) {
		t.Errorf("Different tile counts: %d vs %d", len(floor1.Tiles), len(floor2.Tiles))
	}
}

func TestGeneratorTreasureRooms(t *testing.T) {
	config := DefaultFloorConfig(15, 42)
	config.MinRooms = 15
	config.MaxRooms = 30
	config.TreasureCount = 2

	gen := NewGenerator(config)
	floor, err := gen.Generate()

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if len(floor.TreasureTiles) == 0 {
		t.Error("Should have at least one treasure tile")
	}

	// Count treasure tiles
	treasureCount := 0
	for _, tile := range floor.Tiles {
		if tile.Type == TileTreasure {
			treasureCount++
		}
	}

	if treasureCount == 0 {
		t.Error("No treasure tiles found in floor tiles")
	}
}

func TestGetRoomID(t *testing.T) {
	tests := []struct {
		floor, x, y int
		want        string
	}{
		{0, 0, 0, "floor0_0_0"},
		{1, 5, 10, "floor1_5_10"},
		{42, 7, 3, "floor42_7_3"},
	}

	for _, tc := range tests {
		if got := GetRoomID(tc.floor, tc.x, tc.y); got != tc.want {
			t.Errorf("GetRoomID(%d, %d, %d) = %q, want %q", tc.floor, tc.x, tc.y, got, tc.want)
		}
	}
}

func TestGeneratorCalculateGridSize(t *testing.T) {
	// Small room count
	config := DefaultFloorConfig(1, 42)
	config.MinRooms = 10
	config.MaxRooms = 20
	gen := NewGenerator(config)

	size := gen.calculateGridSize()
	if size < 8 {
		t.Errorf("Grid size too small: %d < 8", size)
	}

	// Large room count
	config2 := DefaultFloorConfig(1, 42)
	config2.MinRooms = 40
	config2.MaxRooms = 60
	gen2 := NewGenerator(config2)

	size2 := gen2.calculateGridSize()
	if size2 > 15 {
		t.Errorf("Grid size too large: %d > 15", size2)
	}
}
