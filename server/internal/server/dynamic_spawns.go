package server

import (
	"math/rand"
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
)

// DynamicSpawnManager handles periodic mob spawning based on player count
type DynamicSpawnManager struct {
	tower             *tower.Tower
	playerCountGetter func() int
	stopChan          chan struct{}
	mu                sync.Mutex

	// Configuration
	enabled             bool
	checkIntervalSec    int
	targetMobsPerPlayer float64
	baseMobsPerFloor    float64
}

// NewDynamicSpawnManager creates a new dynamic spawn manager
func NewDynamicSpawnManager(t *tower.Tower, playerCountGetter func() int) *DynamicSpawnManager {
	return &DynamicSpawnManager{
		tower:               t,
		playerCountGetter:   playerCountGetter,
		stopChan:            make(chan struct{}),
		enabled:             true,
		checkIntervalSec:    30, // Check every 30 seconds
		targetMobsPerPlayer: 5.0,
		baseMobsPerFloor:    15.0,
	}
}

// SetEnabled enables or disables dynamic spawning
func (dsm *DynamicSpawnManager) SetEnabled(enabled bool) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.enabled = enabled
}

// SetCheckInterval sets how often to check for spawn needs (in seconds)
func (dsm *DynamicSpawnManager) SetCheckInterval(seconds int) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.checkIntervalSec = seconds
}

// SetTargetMobsPerPlayer sets the target number of mobs per player
func (dsm *DynamicSpawnManager) SetTargetMobsPerPlayer(target float64) {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	dsm.targetMobsPerPlayer = target
}

// Start begins the dynamic spawn checking loop
func (dsm *DynamicSpawnManager) Start() {
	go dsm.spawnLoop()
	logger.Info("Dynamic spawn manager started")
}

// Stop stops the dynamic spawn checking loop
func (dsm *DynamicSpawnManager) Stop() {
	close(dsm.stopChan)
	logger.Info("Dynamic spawn manager stopped")
}

// spawnLoop periodically checks and spawns mobs based on player count
func (dsm *DynamicSpawnManager) spawnLoop() {
	dsm.mu.Lock()
	interval := dsm.checkIntervalSec
	dsm.mu.Unlock()

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dsm.checkAndSpawn()
		case <-dsm.stopChan:
			return
		}
	}
}

// checkAndSpawn checks if we need to spawn more mobs and does so if necessary
func (dsm *DynamicSpawnManager) checkAndSpawn() {
	dsm.mu.Lock()
	enabled := dsm.enabled
	targetPerPlayer := dsm.targetMobsPerPlayer
	baseMobsPerFloor := dsm.baseMobsPerFloor
	dsm.mu.Unlock()

	if !enabled || dsm.tower == nil {
		return
	}

	playerCount := dsm.playerCountGetter()
	if playerCount <= 0 {
		return
	}

	// Get all generated floors
	highestFloor := dsm.tower.GetHighestFloor()
	if highestFloor <= 0 {
		return
	}

	// Calculate target mobs per floor based on player count
	// Players are assumed to spread across floors
	floorsAvailable := highestFloor // floors 1 to highestFloor
	playersPerFloor := float64(playerCount) / float64(floorsAvailable)
	targetMobsPerFloor := playersPerFloor * targetPerPlayer

	// Calculate spawn multiplier
	multiplier := targetMobsPerFloor / baseMobsPerFloor
	if multiplier < 1.0 {
		multiplier = 1.0
	}
	if multiplier > 10.0 {
		multiplier = 10.0
	}

	// Target count based on multiplier
	targetCount := int(baseMobsPerFloor * multiplier)

	totalSpawned := 0
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Check each floor and spawn if needed
	for floorNum := 1; floorNum <= highestFloor; floorNum++ {
		floor := dsm.tower.GetFloorIfExists(floorNum)
		if floor == nil {
			continue
		}

		// Count current mobs
		currentMobs := tower.CountAliveMobs(floor)

		// If we have fewer than target, spawn more
		if currentMobs < targetCount {
			mobsNeeded := targetCount - currentMobs
			// Don't spawn too many at once (cap at 5 per check)
			if mobsNeeded > 5 {
				mobsNeeded = 5
			}

			spawned := dsm.spawnMobsOnFloor(floor, floorNum, rng, mobsNeeded)
			totalSpawned += len(spawned)
		}
	}

	if totalSpawned > 0 {
		logger.Debug("Dynamic spawner added mobs",
			"spawned", totalSpawned,
			"players", playerCount,
			"multiplier", multiplier,
			"target_per_floor", targetCount)
	}
}

// spawnMobsOnFloor spawns additional mobs on a floor
func (dsm *DynamicSpawnManager) spawnMobsOnFloor(floor *tower.Floor, floorNum int, rng *rand.Rand, count int) []*npc.NPC {
	// Get the mob spawner from the tower
	spawner := dsm.tower.GetMobSpawner()
	if spawner == nil {
		return nil
	}

	return spawner.SpawnAdditionalMobs(floor, floorNum, rng, tower.CountAliveMobs(floor)+count)
}
