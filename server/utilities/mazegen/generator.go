package main

import (
	"math/rand"
)

// Direction represents a cardinal direction
type Direction int

const (
	North Direction = iota
	South
	East
	West
)

func (d Direction) Opposite() Direction {
	switch d {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	case West:
		return East
	}
	return North
}

func (d Direction) String() string {
	switch d {
	case North:
		return "north"
	case South:
		return "south"
	case East:
		return "east"
	case West:
		return "west"
	}
	return "unknown"
}

// AllDirections returns all four cardinal directions
func AllDirections() []Direction {
	return []Direction{North, South, East, West}
}

// Cell represents a single cell in the maze grid
type Cell struct {
	X, Y    int
	Visited bool
	Walls   map[Direction]bool // true = wall exists
	Type    CellType
	POI     POIType // Point of interest in this cell
}

// CellType represents the type of maze cell
type CellType int

const (
	CellPassage CellType = iota
	CellGate             // City gate connection
	CellTreasure         // Treasure vault
	CellMerchant         // Hidden merchant location
	CellLoreNPC          // Lore NPC location
	CellShortcut         // Secret shortcut endpoint
)

// POIType represents a point of interest
type POIType int

const (
	POINone POIType = iota
	POITreasure
	POIMerchant
	POILoreNPC
	POIShortcutA // One end of a shortcut pair
	POIShortcutB // Other end of a shortcut pair
)

// GateInfo holds information about a city gate
type GateInfo struct {
	CityID   string
	CityName string
	X, Y     int
}

// ShortcutPair represents a pair of connected shortcut rooms
type ShortcutPair struct {
	X1, Y1 int
	X2, Y2 int
}

// MazeGenerator generates a labyrinth using DFS recursive backtracker
type MazeGenerator struct {
	Width, Height int
	Grid          [][]*Cell
	Rand          *rand.Rand
	Gates         []GateInfo
	Shortcuts     []ShortcutPair

	// Counters for summary
	TreasureCount int
	MerchantCount int
	LoreNPCCount  int
	ShortcutCount int
	POICount      int
}

// NewMazeGenerator creates a new maze generator
func NewMazeGenerator(width, height int, seed int64) *MazeGenerator {
	mg := &MazeGenerator{
		Width:  width,
		Height: height,
		Grid:   make([][]*Cell, height),
		Rand:   rand.New(rand.NewSource(seed)),
	}

	// Initialize grid with all walls
	for y := 0; y < height; y++ {
		mg.Grid[y] = make([]*Cell, width)
		for x := 0; x < width; x++ {
			mg.Grid[y][x] = &Cell{
				X:       x,
				Y:       y,
				Visited: false,
				Walls: map[Direction]bool{
					North: true,
					South: true,
					East:  true,
					West:  true,
				},
				Type: CellPassage,
				POI:  POINone,
			}
		}
	}

	return mg
}

// Generate runs the DFS recursive backtracker algorithm
func (mg *MazeGenerator) Generate() {
	// Start from center
	startX, startY := mg.Width/2, mg.Height/2
	mg.carveFrom(startX, startY)
}

// carveFrom recursively carves passages using DFS
func (mg *MazeGenerator) carveFrom(x, y int) {
	cell := mg.Grid[y][x]
	cell.Visited = true

	// Shuffle directions for randomness
	dirs := mg.shuffledDirections()

	for _, dir := range dirs {
		nx, ny := mg.neighbor(x, y, dir)
		if mg.inBounds(nx, ny) && !mg.Grid[ny][nx].Visited {
			// Remove wall between current cell and neighbor
			cell.Walls[dir] = false
			mg.Grid[ny][nx].Walls[dir.Opposite()] = false

			// Recurse
			mg.carveFrom(nx, ny)
		}
	}
}

// shuffledDirections returns directions in random order
func (mg *MazeGenerator) shuffledDirections() []Direction {
	dirs := AllDirections()
	mg.Rand.Shuffle(len(dirs), func(i, j int) {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	})
	return dirs
}

// neighbor returns the coordinates of the neighbor in the given direction
func (mg *MazeGenerator) neighbor(x, y int, dir Direction) (int, int) {
	switch dir {
	case North:
		return x, y - 1
	case South:
		return x, y + 1
	case East:
		return x + 1, y
	case West:
		return x - 1, y
	}
	return x, y
}

// inBounds checks if coordinates are within the grid
func (mg *MazeGenerator) inBounds(x, y int) bool {
	return x >= 0 && x < mg.Width && y >= 0 && y < mg.Height
}

// PlaceGates places city gates at fixed positions around the perimeter
func (mg *MazeGenerator) PlaceGates() {
	// Gate positions as specified in the plan
	mg.Gates = []GateInfo{
		{CityID: "human", CityName: "Ironhaven", X: mg.Width / 2, Y: 0},           // North center
		{CityID: "elf", CityName: "Sylvanthal", X: mg.Width - 1, Y: mg.Height / 2}, // East center
		{CityID: "dwarf", CityName: "Khazad-Karn", X: mg.Width / 2, Y: mg.Height - 1}, // South center
		{CityID: "gnome", CityName: "Cogsworth", X: 0, Y: mg.Height / 2},           // West center
		{CityID: "orc", CityName: "Skullgar", X: 0, Y: 0},                          // Northwest corner
	}

	for _, gate := range mg.Gates {
		mg.Grid[gate.Y][gate.X].Type = CellGate
	}
}

// PlacePOIs places points of interest throughout the labyrinth
func (mg *MazeGenerator) PlacePOIs() {
	// Find dead-end cells (cells with only one exit)
	deadEnds := mg.findDeadEnds()

	// Shuffle dead ends for random placement
	mg.Rand.Shuffle(len(deadEnds), func(i, j int) {
		deadEnds[i], deadEnds[j] = deadEnds[j], deadEnds[i]
	})

	// Place treasure vaults (5-10)
	treasureCount := 5 + mg.Rand.Intn(6)
	for i := 0; i < treasureCount && i < len(deadEnds); i++ {
		cell := deadEnds[i]
		if cell.Type == CellPassage && cell.POI == POINone {
			cell.POI = POITreasure
			mg.TreasureCount++
			mg.POICount++
		}
	}

	// Remove used dead ends
	deadEnds = deadEnds[treasureCount:]

	// Place hidden merchants (3-5)
	merchantCount := 3 + mg.Rand.Intn(3)
	for i := 0; i < merchantCount && i < len(deadEnds); i++ {
		cell := deadEnds[i]
		if cell.Type == CellPassage && cell.POI == POINone {
			cell.POI = POIMerchant
			mg.MerchantCount++
			mg.POICount++
		}
	}

	// Remove used dead ends
	deadEnds = deadEnds[merchantCount:]

	// Place lore NPCs (5-8)
	loreCount := 5 + mg.Rand.Intn(4)
	for i := 0; i < loreCount && i < len(deadEnds); i++ {
		cell := deadEnds[i]
		if cell.Type == CellPassage && cell.POI == POINone {
			cell.POI = POILoreNPC
			mg.LoreNPCCount++
			mg.POICount++
		}
	}

	// Place secret shortcuts (2-3 pairs)
	// Find cells that are far apart for shortcuts
	mg.placeShortcuts(2 + mg.Rand.Intn(2))
}

// findDeadEnds finds all cells with only one exit (excluding gates)
func (mg *MazeGenerator) findDeadEnds() []*Cell {
	var deadEnds []*Cell

	for y := 0; y < mg.Height; y++ {
		for x := 0; x < mg.Width; x++ {
			cell := mg.Grid[y][x]
			if cell.Type == CellGate {
				continue
			}

			// Count exits (no wall = exit)
			exits := 0
			for _, dir := range AllDirections() {
				if !cell.Walls[dir] {
					exits++
				}
			}

			if exits == 1 {
				deadEnds = append(deadEnds, cell)
			}
		}
	}

	return deadEnds
}

// placeShortcuts places pairs of shortcut rooms that connect distant parts of the maze
func (mg *MazeGenerator) placeShortcuts(count int) {
	// Find cells in different quadrants for shortcuts
	type quadrantCell struct {
		cell     *Cell
		quadrant int
	}

	var candidates []quadrantCell

	for y := 0; y < mg.Height; y++ {
		for x := 0; x < mg.Width; x++ {
			cell := mg.Grid[y][x]
			if cell.Type != CellPassage || cell.POI != POINone {
				continue
			}

			// Determine quadrant (0=NW, 1=NE, 2=SW, 3=SE)
			quadrant := 0
			if x >= mg.Width/2 {
				quadrant += 1
			}
			if y >= mg.Height/2 {
				quadrant += 2
			}

			candidates = append(candidates, quadrantCell{cell: cell, quadrant: quadrant})
		}
	}

	// Group by quadrant
	quadrants := make([][]*Cell, 4)
	for _, qc := range candidates {
		quadrants[qc.quadrant] = append(quadrants[qc.quadrant], qc.cell)
	}

	// Shuffle each quadrant
	for i := range quadrants {
		mg.Rand.Shuffle(len(quadrants[i]), func(a, b int) {
			quadrants[i][a], quadrants[i][b] = quadrants[i][b], quadrants[i][a]
		})
	}

	// Create shortcuts between opposite quadrants
	pairs := [][2]int{{0, 3}, {1, 2}} // NW-SE, NE-SW

	for i := 0; i < count && i < len(pairs); i++ {
		q1, q2 := pairs[i][0], pairs[i][1]
		if len(quadrants[q1]) > 0 && len(quadrants[q2]) > 0 {
			cell1 := quadrants[q1][0]
			cell2 := quadrants[q2][0]

			cell1.POI = POIShortcutA
			cell2.POI = POIShortcutB

			mg.Shortcuts = append(mg.Shortcuts, ShortcutPair{
				X1: cell1.X, Y1: cell1.Y,
				X2: cell2.X, Y2: cell2.Y,
			})

			mg.ShortcutCount++
			mg.POICount += 2

			// Remove used cells
			quadrants[q1] = quadrants[q1][1:]
			quadrants[q2] = quadrants[q2][1:]
		}
	}
}

// RoomCount returns the total number of rooms in the maze
func (mg *MazeGenerator) RoomCount() int {
	return mg.Width * mg.Height
}
