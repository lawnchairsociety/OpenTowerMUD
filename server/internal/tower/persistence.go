package tower

import (
	"fmt"
	"os"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
	"gopkg.in/yaml.v3"
)

// TowerData represents the serialized tower structure for persistence
type TowerData struct {
	Seed         int64       `yaml:"seed"`
	HighestFloor int         `yaml:"highest_floor"`
	SavedAt      time.Time   `yaml:"saved_at"`
	Floors       []FloorData `yaml:"floors"`
}

// FloorData represents a serialized floor
type FloorData struct {
	Number         int        `yaml:"number"`
	StairsUpRoom   string     `yaml:"stairs_up_room"`
	StairsDownRoom string     `yaml:"stairs_down_room"`
	PortalRoom     string     `yaml:"portal_room"`
	GeneratedAt    time.Time  `yaml:"generated_at"`
	Rooms          []RoomData `yaml:"rooms"`
}

// RoomData represents a serialized room
type RoomData struct {
	ID               string            `yaml:"id"`
	Name             string            `yaml:"name"`
	Description      string            `yaml:"description"`
	DescriptionDay   string            `yaml:"description_day,omitempty"`
	DescriptionNight string            `yaml:"description_night,omitempty"`
	Type             string            `yaml:"type"`
	Features         []string          `yaml:"features,omitempty"`
	Floor            int               `yaml:"floor"`
	Exits            map[string]string `yaml:"exits"` // direction -> room_id
}

// SaveTower saves the tower state to a YAML file
func SaveTower(t *Tower, filename string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data := TowerData{
		Seed:         t.Seed,
		HighestFloor: t.HighestFloor,
		SavedAt:      time.Now(),
		Floors:       make([]FloorData, 0, len(t.Floors)),
	}

	// Serialize each floor
	for floorNum, floor := range t.Floors {
		floorData := serializeFloor(floorNum, floor)
		data.Floors = append(data.Floors, floorData)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("failed to marshal tower data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write tower file: %w", err)
	}

	return nil
}

// serializeFloor converts a Floor to FloorData
func serializeFloor(floorNum int, floor *Floor) FloorData {
	floor.mu.RLock()
	defer floor.mu.RUnlock()

	floorData := FloorData{
		Number:         floorNum,
		StairsUpRoom:   floor.StairsUpRoom,
		StairsDownRoom: floor.StairsDownRoom,
		PortalRoom:     floor.PortalRoom,
		GeneratedAt:    floor.Generated,
		Rooms:          make([]RoomData, 0, len(floor.Rooms)),
	}

	// Serialize each room
	for _, room := range floor.Rooms {
		roomData := serializeRoom(room)
		floorData.Rooms = append(floorData.Rooms, roomData)
	}

	return floorData
}

// serializeRoom converts a Room to RoomData
func serializeRoom(room *world.Room) RoomData {
	exits := make(map[string]string)
	for dir, exitRoom := range room.Exits {
		if exitRoom != nil {
			exits[dir] = exitRoom.ID
		}
	}

	return RoomData{
		ID:               room.ID,
		Name:             room.Name,
		Description:      room.Description,
		DescriptionDay:   room.DescriptionDay,
		DescriptionNight: room.DescriptionNight,
		Type:             room.Type.String(),
		Features:         room.Features,
		Floor:            room.Floor,
		Exits:            exits,
	}
}

// LoadTower loads a tower from a YAML file
func LoadTower(filename string) (*Tower, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read tower file: %w", err)
	}

	var towerData TowerData
	if err := yaml.Unmarshal(data, &towerData); err != nil {
		return nil, fmt.Errorf("failed to parse tower YAML: %w", err)
	}

	// Create tower
	tower := NewTower(towerData.Seed)
	tower.HighestFloor = towerData.HighestFloor

	// Collect all exit data for second pass
	var allExits []pendingExits

	// Deserialize each floor
	for _, floorData := range towerData.Floors {
		floor, exits, err := deserializeFloorWithExits(floorData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize floor %d: %w", floorData.Number, err)
		}
		tower.Floors[floorData.Number] = floor
		allExits = append(allExits, exits...)
	}

	// Second pass: link room exits across all floors
	linkRoomExits(tower, allExits)

	return tower, nil
}

// deserializeFloorWithExits converts FloorData back to a Floor and collects exit data
func deserializeFloorWithExits(data FloorData) (*Floor, []pendingExits, error) {
	floor := NewFloor(data.Number)
	floor.StairsUpRoom = data.StairsUpRoom
	floor.StairsDownRoom = data.StairsDownRoom
	floor.PortalRoom = data.PortalRoom
	floor.Generated = data.GeneratedAt

	var exits []pendingExits

	// Deserialize rooms
	for _, roomData := range data.Rooms {
		room, err := deserializeRoom(roomData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to deserialize room %s: %w", roomData.ID, err)
		}
		floor.Rooms[room.ID] = room

		// Collect exit data for second pass
		if len(roomData.Exits) > 0 {
			exits = append(exits, pendingExits{
				roomID: room.ID,
				exits:  roomData.Exits,
			})
		}
	}

	return floor, exits, nil
}

// deserializeRoom converts RoomData back to a Room
func deserializeRoom(data RoomData) (*world.Room, error) {
	roomType, _ := world.ParseRoomType(data.Type)

	room := world.NewRoom(data.ID, data.Name, data.Description, roomType)
	room.DescriptionDay = data.DescriptionDay
	room.DescriptionNight = data.DescriptionNight
	room.Floor = data.Floor

	// Add features
	for _, feature := range data.Features {
		room.AddFeature(feature)
	}

	// Note: Exits are linked in a second pass after all rooms are created

	return room, nil
}

// pendingExits stores exit data during deserialization for later linking
type pendingExits struct {
	roomID string
	exits  map[string]string // direction -> target room ID
}

// linkRoomExits links all room exits after rooms are created
func linkRoomExits(tower *Tower, exitData []pendingExits) {
	// Build a map of all rooms by ID
	allRooms := make(map[string]*world.Room)
	for _, floor := range tower.Floors {
		for roomID, room := range floor.Rooms {
			allRooms[roomID] = room
		}
	}

	// Link exits using the stored exit data
	for _, pending := range exitData {
		room := allRooms[pending.roomID]
		if room == nil {
			continue
		}
		for direction, targetID := range pending.exits {
			targetRoom := allRooms[targetID]
			if targetRoom != nil {
				room.AddExit(direction, targetRoom)
			}
		}
	}
}

// TowerFileExists checks if a tower save file exists
func TowerFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
