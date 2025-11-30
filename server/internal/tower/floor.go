package tower

import (
	"fmt"
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// Floor represents a single floor of the tower
type Floor struct {
	Number         int                    // Floor number (0 = city, 1+ = tower floors)
	Rooms          map[string]*world.Room // All rooms on this floor
	StairsUpRoom   string                 // Room ID with stairs going up (empty for top floor)
	StairsDownRoom string                 // Room ID with stairs going down (empty for floor 0)
	PortalRoom     string                 // Room ID with portal (for fast travel)
	Generated      time.Time              // When this floor was generated
	mu             sync.RWMutex
}

// NewFloor creates a new floor
func NewFloor(number int) *Floor {
	return &Floor{
		Number:    number,
		Rooms:     make(map[string]*world.Room),
		Generated: time.Now(),
	}
}

// AddRoom adds a room to the floor
func (f *Floor) AddRoom(room *world.Room) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Rooms[room.ID] = room
}

// GetRoom returns a room by ID
func (f *Floor) GetRoom(roomID string) *world.Room {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.Rooms[roomID]
}

// GetRooms returns all rooms on this floor
func (f *Floor) GetRooms() map[string]*world.Room {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rooms := make(map[string]*world.Room, len(f.Rooms))
	for id, room := range f.Rooms {
		rooms[id] = room
	}
	return rooms
}

// RoomCount returns the number of rooms on this floor
func (f *Floor) RoomCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.Rooms)
}

// GetStairsUp returns the room with stairs going up
func (f *Floor) GetStairsUp() *world.Room {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.StairsUpRoom == "" {
		return nil
	}
	return f.Rooms[f.StairsUpRoom]
}

// GetStairsDown returns the room with stairs going down
func (f *Floor) GetStairsDown() *world.Room {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.StairsDownRoom == "" {
		return nil
	}
	return f.Rooms[f.StairsDownRoom]
}

// GetPortalRoom returns the room with the portal
func (f *Floor) GetPortalRoom() *world.Room {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.PortalRoom == "" {
		return nil
	}
	return f.Rooms[f.PortalRoom]
}

// SetStairsUp sets the room ID that has stairs going up
func (f *Floor) SetStairsUp(roomID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StairsUpRoom = roomID
}

// SetStairsDown sets the room ID that has stairs going down
func (f *Floor) SetStairsDown(roomID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StairsDownRoom = roomID
}

// SetPortalRoom sets the room ID that has the portal
func (f *Floor) SetPortalRoom(roomID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PortalRoom = roomID
}

// IsBossFloor returns true if this is a boss floor (every 10th floor)
func (f *Floor) IsBossFloor() bool {
	return f.Number > 0 && f.Number%10 == 0
}

// IsCity returns true if this is the ground floor (city)
func (f *Floor) IsCity() bool {
	return f.Number == 0
}

// GetDifficultyMultiplier returns the difficulty scaling factor for this floor
func (f *Floor) GetDifficultyMultiplier() float64 {
	if f.Number <= 0 {
		return 1.0
	}
	// Gradual scaling: 10% increase per floor
	return 1.0 + float64(f.Number)*0.1
}

// String returns a string representation of the floor
func (f *Floor) String() string {
	if f.IsCity() {
		return "Ground Floor (City)"
	}
	if f.IsBossFloor() {
		return fmt.Sprintf("Floor %d (Boss)", f.Number)
	}
	return fmt.Sprintf("Floor %d", f.Number)
}
