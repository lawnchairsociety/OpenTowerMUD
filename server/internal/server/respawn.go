package server

import (
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// RespawnManager handles tracking and respawning dead NPCs
type RespawnManager struct {
	deadNPCs []*npc.NPC
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewRespawnManager creates a new respawn manager
func NewRespawnManager() *RespawnManager {
	return &RespawnManager{
		deadNPCs: make([]*npc.NPC, 0),
		stopChan: make(chan struct{}),
	}
}

// AddDeadNPC adds an NPC to the respawn queue
func (rm *RespawnManager) AddDeadNPC(n *npc.NPC) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Calculate respawn time
	n.CalculateRespawnTime()

	// Only add if respawn is enabled (median > 0)
	if n.GetRespawnMedian() > 0 {
		rm.deadNPCs = append(rm.deadNPCs, n)
	}
}

// Start begins the respawn checking loop
func (rm *RespawnManager) Start(respawnFunc func(*npc.NPC)) {
	go rm.checkRespawns(respawnFunc)
}

// Stop stops the respawn checking loop
func (rm *RespawnManager) Stop() {
	close(rm.stopChan)
}

// checkRespawns periodically checks for NPCs ready to respawn
func (rm *RespawnManager) checkRespawns(respawnFunc func(*npc.NPC)) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.processRespawns(respawnFunc)
		case <-rm.stopChan:
			return
		}
	}
}

// processRespawns checks all dead NPCs and respawns those ready
func (rm *RespawnManager) processRespawns(respawnFunc func(*npc.NPC)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	remaining := make([]*npc.NPC, 0)

	for _, deadNPC := range rm.deadNPCs {
		respawnTime := deadNPC.GetRespawnTime()

		// Check if it's time to respawn
		if now.After(respawnTime) || now.Equal(respawnTime) {
			// Call the respawn function (provided by server)
			respawnFunc(deadNPC)
		} else {
			// Keep in queue
			remaining = append(remaining, deadNPC)
		}
	}

	rm.deadNPCs = remaining
}

// GetDeadNPCCount returns the number of NPCs waiting to respawn
func (rm *RespawnManager) GetDeadNPCCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.deadNPCs)
}
