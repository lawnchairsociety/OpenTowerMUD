package server

import (
	"sync"
	"testing"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/config"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// TestServer_ConcurrentClientAccess tests that multiple goroutines can safely
// access client data simultaneously
func TestServer_ConcurrentClientAccess(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":0", w, false)
	defer s.Shutdown()

	// Add some mock clients
	var wg sync.WaitGroup
	const numClients = 10

	// Concurrently access clients map
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Simulate concurrent reads (FindPlayer, GetOnlinePlayers)
			for j := 0; j < 100; j++ {
				_ = s.GetOnlinePlayers()
				_ = s.FindPlayer("TestPlayer")
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestServer_ConcurrentBroadcast tests that broadcasts are safe under concurrent access
func TestServer_ConcurrentBroadcast(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":0", w, false)
	defer s.Shutdown()

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Simulate concurrent broadcasts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.BroadcastMessage("Test message", nil)
			}
		}()
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestServer_ConcurrentRoomAccess tests concurrent access to room data
func TestServer_ConcurrentRoomAccess(t *testing.T) {
	w := world.NewWorld()

	// Create test rooms
	room1 := world.NewRoom("room1", "Room 1", "First room", world.RoomTypeRoom)
	room2 := world.NewRoom("room2", "Room 2", "Second room", world.RoomTypeRoom)
	w.AddRoom(room1)
	w.AddRoom(room2)

	var wg sync.WaitGroup
	const numGoroutines = 20

	// Concurrent room operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				// Simulate player movement between rooms
				playerName := "Player"
				if j%2 == 0 {
					room1.AddPlayer(playerName)
					room1.RemovePlayer(playerName)
				} else {
					room2.AddPlayer(playerName)
					room2.RemovePlayer(playerName)
				}
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestServer_ConcurrentWorldAccess tests concurrent access to world rooms
func TestServer_ConcurrentWorldAccess(t *testing.T) {
	w := world.NewWorld()

	// Create initial rooms
	for i := 0; i < 5; i++ {
		room := world.NewRoom("room"+string(rune('A'+i)), "Room", "Description", world.RoomTypeRoom)
		w.AddRoom(room)
	}

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Concurrent world reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = w.GetRoom("roomA")
				_ = w.GetRoom("roomB")
			}
		}()
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestServer_ConcurrentTickerSimulation tests simulated ticker operations
func TestServer_ConcurrentTickerSimulation(t *testing.T) {
	w := world.NewWorld()
	s := NewServer(":0", w, false)
	defer s.Shutdown()

	var wg sync.WaitGroup

	// Simulate regeneration ticker access
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = s.GetOnlinePlayers()
			time.Sleep(time.Millisecond)
		}
	}()

	// Simulate combat ticker access
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = s.GetOnlinePlayers()
			time.Sleep(time.Millisecond)
		}
	}()

	// Simulate client list modifications
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = s.GetOnlinePlayers()
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestRespawnManager_ConcurrentAccess tests concurrent access to respawn manager
func TestRespawnManager_ConcurrentAccess(t *testing.T) {
	rm := NewRespawnManager()

	// Start respawn manager with a no-op respawn function
	rm.Start(func(n *npc.NPC) {})
	defer rm.Stop()

	var wg sync.WaitGroup
	const numGoroutines = 5

	// Concurrent operations - check queue count concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Multiple reads/checks - respawn manager should handle safely
			for j := 0; j < 50; j++ {
				_ = rm.GetDeadNPCCount()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestLoginRateLimiter_ConcurrentAccess tests concurrent rate limiter operations
func TestLoginRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewLoginRateLimiter(defaultRateLimitConfig())
	defer rl.Stop()

	var wg sync.WaitGroup
	const numGoroutines = 20
	const numOps = 50

	// Concurrent rate limit checks and records
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1." + string(rune('0'+id%10))

			for j := 0; j < numOps; j++ {
				// Mix of operations
				switch j % 4 {
				case 0:
					rl.IsLocked(ip)
				case 1:
					rl.RecordFailure(ip)
				case 2:
					rl.GetAttempts(ip)
				case 3:
					rl.RecordSuccess(ip)
				}
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// TestConnLimiter_ConcurrentAccess tests concurrent connection limiter operations
func TestConnLimiter_ConcurrentAccess(t *testing.T) {
	cfg := config.ConnectionsConfig{
		MaxTotal: 100,
		MaxPerIP: 10,
	}
	cl := NewConnLimiter(cfg)

	var wg sync.WaitGroup
	const numGoroutines = 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1." + string(rune('0'+id%10))

			// Try to acquire and release
			for j := 0; j < 50; j++ {
				if cl.TryAcquire(ip) {
					time.Sleep(time.Microsecond)
					cl.Release(ip)
				}
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition or panic
}

// defaultRateLimitConfig returns default rate limit config for testing
func defaultRateLimitConfig() config.RateLimitConfig {
	return config.RateLimitConfig{
		MaxAttempts:       5,
		LockoutSeconds:    30,
		MaxLockoutSeconds: 300,
	}
}
