package world

// RoomType represents the category of a room
type RoomType int

const (
	RoomTypeCity     RoomType = iota // Safe city rooms (ground floor)
	RoomTypeCorridor                 // Tower corridors
	RoomTypeRoom                     // General tower rooms
	RoomTypeStairs                   // Stairway rooms (up/down)
	RoomTypeTreasure                 // Treasure/loot rooms
	RoomTypeBoss                     // Boss rooms (every 10 floors)
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
	default:
		return "unknown"
	}
}

// IsSafe returns true if the room type is generally safe
func (t RoomType) IsSafe() bool {
	return t == RoomTypeCity
}

// GetDangerLevel returns a danger rating from 0 (safe) to 5 (very dangerous)
func (t RoomType) GetDangerLevel() int {
	switch t {
	case RoomTypeCity:
		return 0
	case RoomTypeCorridor:
		return 2
	case RoomTypeRoom:
		return 3
	case RoomTypeTreasure:
		return 3
	case RoomTypeStairs:
		return 1
	case RoomTypeBoss:
		return 5
	default:
		return 0
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
	default:
		return RoomTypeCity, false
	}
}
