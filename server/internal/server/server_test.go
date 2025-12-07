package server

import (
	"sync"
	"testing"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// TestServer_Shutdown_CalledTwice tests that calling Shutdown() twice doesn't panic
func TestServer_Shutdown_CalledTwice(t *testing.T) {
	// Create a minimal server
	w := world.NewWorld()
	s := NewServer(":0", w, false)

	// First shutdown should work
	s.Shutdown()

	// Second shutdown should not panic (protected by sync.Once)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Second Shutdown() call panicked: %v", r)
		}
	}()

	s.Shutdown()
}

// TestServer_Shutdown_Concurrent tests that concurrent Shutdown() calls are safe
func TestServer_Shutdown_Concurrent(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":0", w, false)

	var wg sync.WaitGroup
	panicCount := 0
	var mu sync.Mutex

	// Try to shutdown from multiple goroutines simultaneously
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					panicCount++
					mu.Unlock()
				}
			}()
			s.Shutdown()
		}()
	}

	wg.Wait()

	if panicCount > 0 {
		t.Errorf("Concurrent Shutdown() calls caused %d panics", panicCount)
	}
}

// TestServer_NewServer_Defaults tests that NewServer creates a server with correct defaults
func TestServer_NewServer_Defaults(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":4000", w, false)

	if s.address != ":4000" {
		t.Errorf("Expected address :4000, got %s", s.address)
	}

	if s.world != w {
		t.Error("World not set correctly")
	}

	if s.pilgrimMode {
		t.Error("Pilgrim mode should be false by default")
	}

	if s.clients == nil {
		t.Error("Clients map should be initialized")
	}

	if s.shutdown == nil {
		t.Error("Shutdown channel should be initialized")
	}

	if s.gameClock == nil {
		t.Error("Game clock should be initialized")
	}

	if s.respawnManager == nil {
		t.Error("Respawn manager should be initialized")
	}

	// Clean up
	s.Shutdown()
}

// TestServer_PilgrimMode tests that pilgrim mode is set correctly
func TestServer_PilgrimMode(t *testing.T) {
	w := world.NewWorld()

	// Test with pilgrim mode enabled
	s := NewServer(":4000", w, true)
	if !s.IsPilgrimMode() {
		t.Error("Pilgrim mode should be enabled")
	}
	s.Shutdown()

	// Test with pilgrim mode disabled
	s2 := NewServer(":4000", w, false)
	if s2.IsPilgrimMode() {
		t.Error("Pilgrim mode should be disabled")
	}
	s2.Shutdown()
}

// TestServer_GetUptime tests that uptime is tracked correctly
func TestServer_GetUptime(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":4000", w, false)
	defer s.Shutdown()

	// Uptime should be very small initially
	uptime := s.GetUptime()
	if uptime < 0 {
		t.Error("Uptime should be non-negative")
	}

	// Wait a bit and check uptime increased
	time.Sleep(50 * time.Millisecond)
	uptime2 := s.GetUptime()
	if uptime2 <= uptime {
		t.Error("Uptime should increase over time")
	}
}

// TestServer_GetWorldRoomCount tests room count reporting
func TestServer_GetWorldRoomCount(t *testing.T) {
	w := world.NewWorld()

	// Add some rooms
	room1 := world.NewRoom("room1", "Room 1", "First room", world.RoomTypeRoom)
	room2 := world.NewRoom("room2", "Room 2", "Second room", world.RoomTypeRoom)
	w.AddRoom(room1)
	w.AddRoom(room2)

	s := NewServer(":4000", w, false)
	defer s.Shutdown()

	count := s.GetWorldRoomCount()
	if count != 2 {
		t.Errorf("Expected 2 rooms, got %d", count)
	}
}

// TestServer_GetOnlinePlayers_Empty tests online player list when empty
func TestServer_GetOnlinePlayers_Empty(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":4000", w, false)
	defer s.Shutdown()

	players := s.GetOnlinePlayers()
	if len(players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(players))
	}
}
