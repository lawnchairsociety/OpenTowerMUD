package server

import (
	"sync"
	"testing"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

func TestRespawnManagerAddDeadNPC(t *testing.T) {
	rm := NewRespawnManager()

	// Create an NPC with respawn enabled
	npc1 := npc.NewNPC(
		"goblin",
		"A goblin",
		1,
		20,
		3,
		0,
		10,
		true,
		true,
		"room1",
		10, // 10 second respawn
		2,  // +/- 2 seconds
	)

	// Create an NPC with respawn disabled
	npc2 := npc.NewNPC(
		"merchant",
		"A merchant",
		5,
		50,
		8,
		2,
		0,
		false,
		false,
		"room2",
		0, // respawn disabled
		0,
	)

	// Add both NPCs
	rm.AddDeadNPC(npc1)
	rm.AddDeadNPC(npc2)

	// Only npc1 should be in the queue (npc2 has respawn disabled)
	count := rm.GetDeadNPCCount()
	if count != 1 {
		t.Errorf("Expected 1 NPC in respawn queue, got %d", count)
	}

	// Verify npc1 has a respawn time set
	if npc1.GetRespawnTime().IsZero() {
		t.Error("Expected npc1 to have respawn time set")
	}

	// Verify npc2 does not have a respawn time
	if !npc2.GetRespawnTime().IsZero() {
		t.Error("Expected npc2 to not have respawn time set")
	}
}

func TestRespawnManagerProcessRespawns(t *testing.T) {
	rm := NewRespawnManager()

	// Track respawned NPCs
	var mu sync.Mutex
	respawned := make([]*npc.NPC, 0)

	respawnFunc := func(n *npc.NPC) {
		mu.Lock()
		respawned = append(respawned, n)
		mu.Unlock()
	}

	// Create NPC with very short respawn time (1 second)
	quickNPC := npc.NewNPC(
		"quick goblin",
		"A quickly respawning goblin",
		1,
		20,
		3,
		0,
		10,
		true,
		true,
		"room1",
		1, // 1 second respawn
		0, // no variation
	)

	// Create NPC with longer respawn time (10 seconds)
	slowNPC := npc.NewNPC(
		"slow troll",
		"A slowly respawning troll",
		7,
		100,
		18,
		4,
		120,
		true,
		true,
		"room2",
		10, // 10 second respawn
		0,  // no variation
	)

	// Add both NPCs
	rm.AddDeadNPC(quickNPC)
	rm.AddDeadNPC(slowNPC)

	// Should have 2 NPCs in queue
	if rm.GetDeadNPCCount() != 2 {
		t.Errorf("Expected 2 NPCs in queue, got %d", rm.GetDeadNPCCount())
	}

	// Wait for quick NPC to be ready (1.5 seconds)
	time.Sleep(1500 * time.Millisecond)

	// Process respawns
	rm.processRespawns(respawnFunc)

	// Check that only quickNPC was respawned
	mu.Lock()
	if len(respawned) != 1 {
		t.Errorf("Expected 1 respawned NPC, got %d", len(respawned))
	} else if respawned[0].GetName() != "quick goblin" {
		t.Errorf("Expected 'quick goblin' to respawn, got '%s'", respawned[0].GetName())
	}
	mu.Unlock()

	// Should have 1 NPC left in queue (slowNPC)
	if rm.GetDeadNPCCount() != 1 {
		t.Errorf("Expected 1 NPC remaining in queue, got %d", rm.GetDeadNPCCount())
	}

	// Wait for slow NPC to be ready (additional 9 seconds)
	time.Sleep(9 * time.Second)

	// Process respawns again
	rm.processRespawns(respawnFunc)

	// Check that slowNPC was respawned
	mu.Lock()
	if len(respawned) != 2 {
		t.Errorf("Expected 2 total respawned NPCs, got %d", len(respawned))
	} else if respawned[1].GetName() != "slow troll" {
		t.Errorf("Expected 'slow troll' to respawn, got '%s'", respawned[1].GetName())
	}
	mu.Unlock()

	// Should have 0 NPCs left in queue
	if rm.GetDeadNPCCount() != 0 {
		t.Errorf("Expected 0 NPCs remaining in queue, got %d", rm.GetDeadNPCCount())
	}
}

func TestRespawnManagerStartStop(t *testing.T) {
	rm := NewRespawnManager()

	// Track if respawn function was called
	var mu sync.Mutex
	called := false

	respawnFunc := func(n *npc.NPC) {
		mu.Lock()
		called = true
		mu.Unlock()
	}

	// Start the respawn manager
	rm.Start(respawnFunc)

	// Add an NPC with very short respawn time
	quickNPC := npc.NewNPC(
		"test npc",
		"A test NPC",
		1,
		10,
		1,
		0,
		5,
		true,
		true,
		"room1",
		1, // 1 second respawn
		0,
	)
	rm.AddDeadNPC(quickNPC)

	// Wait for respawn to process
	time.Sleep(6 * time.Second) // Wait long enough for ticker + respawn time

	// Stop the manager
	rm.Stop()

	// Check if respawn function was called
	mu.Lock()
	wasCalled := called
	mu.Unlock()

	if !wasCalled {
		t.Error("Expected respawn function to be called")
	}

	// Verify queue is empty
	if rm.GetDeadNPCCount() != 0 {
		t.Errorf("Expected empty queue after respawn, got %d", rm.GetDeadNPCCount())
	}
}
