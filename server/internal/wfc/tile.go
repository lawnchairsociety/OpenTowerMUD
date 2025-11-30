package wfc

// TileType represents the type of tile in the WFC grid
type TileType int

const (
	TileEmpty      TileType = iota // No tile (unassigned)
	TileCorridor                   // Corridor - connects in 2-4 directions
	TileRoom                       // Room - open area with doors
	TileDeadEnd                    // Dead end - single connection
	TileTreasure                   // Treasure room - special loot
	TileBoss                       // Boss room - every 10th floor
	TileStairsUp                   // Stairs going up to the next floor
	TileStairsDown                 // Stairs coming down from the previous floor
)

// String returns the string representation of a TileType
func (t TileType) String() string {
	switch t {
	case TileEmpty:
		return "empty"
	case TileCorridor:
		return "corridor"
	case TileRoom:
		return "room"
	case TileDeadEnd:
		return "dead_end"
	case TileTreasure:
		return "treasure"
	case TileBoss:
		return "boss"
	case TileStairsUp:
		return "stairs_up"
	case TileStairsDown:
		return "stairs_down"
	default:
		return "unknown"
	}
}

// Direction represents a cardinal direction in the grid
type Direction int

const (
	North Direction = iota
	East
	South
	West
)

// String returns the string representation of a Direction
func (d Direction) String() string {
	switch d {
	case North:
		return "north"
	case East:
		return "east"
	case South:
		return "south"
	case West:
		return "west"
	default:
		return "unknown"
	}
}

// Opposite returns the opposite direction
func (d Direction) Opposite() Direction {
	switch d {
	case North:
		return South
	case East:
		return West
	case South:
		return North
	case West:
		return East
	default:
		return d
	}
}

// AllDirections returns all four cardinal directions
func AllDirections() []Direction {
	return []Direction{North, East, South, West}
}

// Tile represents a single cell in the WFC grid
type Tile struct {
	Type       TileType
	X, Y       int
	Connections map[Direction]bool // Which directions have exits
}

// NewTile creates a new tile at the given position
func NewTile(tileType TileType, x, y int) *Tile {
	return &Tile{
		Type:        tileType,
		X:           x,
		Y:           y,
		Connections: make(map[Direction]bool),
	}
}

// ConnectionCount returns the number of active connections
func (t *Tile) ConnectionCount() int {
	count := 0
	for _, connected := range t.Connections {
		if connected {
			count++
		}
	}
	return count
}

// HasConnection returns true if the tile has a connection in the given direction
func (t *Tile) HasConnection(dir Direction) bool {
	return t.Connections[dir]
}

// SetConnection sets the connection state for a direction
func (t *Tile) SetConnection(dir Direction, connected bool) {
	t.Connections[dir] = connected
}
