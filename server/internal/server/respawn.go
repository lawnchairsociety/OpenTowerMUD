package server

import (
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
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
		logger.Debug("NPC added to respawn queue",
			"npc", n.GetName(),
			"respawn_time", n.GetRespawnTime().Format(time.RFC3339),
			"queue_size", len(rm.deadNPCs))
	} else {
		logger.Debug("NPC not added to respawn queue (respawn disabled)",
			"npc", n.GetName())
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
	respawnedCount := 0

	for _, deadNPC := range rm.deadNPCs {
		respawnTime := deadNPC.GetRespawnTime()

		// Check if it's time to respawn
		if now.After(respawnTime) || now.Equal(respawnTime) {
			logger.Debug("NPC respawning from queue",
				"npc", deadNPC.GetName(),
				"scheduled_time", respawnTime.Format(time.RFC3339),
				"actual_time", now.Format(time.RFC3339))
			// Call the respawn function (provided by server)
			respawnFunc(deadNPC)
			respawnedCount++
		} else {
			// Keep in queue
			remaining = append(remaining, deadNPC)
		}
	}

	if respawnedCount > 0 || len(rm.deadNPCs) > 0 {
		logger.Debug("Respawn queue processed",
			"respawned", respawnedCount,
			"remaining", len(remaining))
	}

	rm.deadNPCs = remaining
}

// GetDeadNPCCount returns the number of NPCs waiting to respawn
func (rm *RespawnManager) GetDeadNPCCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.deadNPCs)
}
