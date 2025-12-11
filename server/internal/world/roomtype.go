package world

// RoomType represents the category of a room
type RoomType int

const (
	RoomTypeCity          RoomType = iota // Safe city rooms (ground floor)
	RoomTypeCorridor                      // Tower corridors
	RoomTypeRoom                          // General tower rooms
	RoomTypeStairs                        // Stairway rooms (up/down)
	RoomTypeTreasure                      // Treasure/loot rooms
	RoomTypeBoss                          // Boss rooms (every 10 floors)
	RoomTypeLabyrinth                     // Labyrinth passages
	RoomTypeLabyrinthGate                 // City gate rooms in the labyrinth
)

// String returns the string representation of a RoomType
func (t RoomType) String() string {
	switch t {
	case RoomTypeCity:
		return "city"
	case RoomTypeCorridor:
		return "corridor"
	case RoomTypeRoom:
		return "room"
	case RoomTypeStairs:
		return "stairs"
	case RoomTypeTreasure:
		return "treasure"
	case RoomTypeBoss:
		return "boss"
	case RoomTypeLabyrinth:
		return "labyrinth"
	case RoomTypeLabyrinthGate:
		return "labyrinth_gate"
	default:
		return "unknown"
	}
}

// ParseRoomType converts a string to a RoomType
func ParseRoomType(s string) (RoomType, bool) {
	switch s {
	case "city":
		return RoomTypeCity, true
	case "corridor":
		return RoomTypeCorridor, true
	case "room":
		return RoomTypeRoom, true
	case "stairs":
		return RoomTypeStairs, true
	case "treasure":
		return RoomTypeTreasure, true
	case "boss":
		return RoomTypeBoss, true
	case "labyrinth":
		return RoomTypeLabyrinth, true
	case "labyrinth_gate":
		return RoomTypeLabyrinthGate, true
	default:
		return RoomTypeCity, false
	}
}
