package tower

import (
	"sync"
	"time"
)

// BossKill represents a recorded boss kill.
type BossKill struct {
	ID          int64
	TowerID     TowerID
	PlayerName  string
	KilledAt    time.Time
	IsFirstKill bool
}

// BossKillDB represents a boss kill record from the database (using string tower ID).
type BossKillDB struct {
	ID          int64
	TowerID     string
	PlayerName  string
	KilledAt    time.Time
	IsFirstKill bool
}

// BossKillStore defines the interface for boss kill persistence.
type BossKillStore interface {
	RecordBossKill(towerID, playerName string) (bool, error)
	GetFirstKill(towerID string) (*BossKillDB, error)
	GetAllFirstKills() ([]BossKillDB, error)
	HasTowerBeenDefeated(towerID string) (bool, error)
	HasPlayerKilledBoss(towerID, playerName string) (bool, error)
	GetPlayerBossKills(playerName string) ([]BossKillDB, error)
	GetTowerBossKills(towerID string) ([]BossKillDB, error)
	IsUnifiedTowerUnlocked() (bool, error)
}

// BossTracker tracks boss defeats across all towers.
type BossTracker struct {
	store           BossKillStore
	mu              sync.RWMutex
	defeatedTowers  map[TowerID]bool // Cache of defeated towers
	unifiedUnlocked bool             // Cached unlock state
}

// NewBossTracker creates a new boss tracker.
func NewBossTracker(store BossKillStore) (*BossTracker, error) {
	bt := &BossTracker{
		store:          store,
		defeatedTowers: make(map[TowerID]bool),
	}

	// Load initial state from database
	if err := bt.loadState(); err != nil {
		return nil, err
	}

	return bt, nil
}

// loadState loads the current state from the database into the cache.
func (bt *BossTracker) loadState() error {
	// Load which towers have been defeated
	for _, towerID := range AllRacialTowers {
		defeated, err := bt.store.HasTowerBeenDefeated(string(towerID))
		if err != nil {
			return err
		}
		bt.defeatedTowers[towerID] = defeated
	}

	// Check unified tower state
	defeated, err := bt.store.HasTowerBeenDefeated(string(TowerUnified))
	if err != nil {
		return err
	}
	bt.defeatedTowers[TowerUnified] = defeated

	// Check if unified is unlocked
	unlocked, err := bt.store.IsUnifiedTowerUnlocked()
	if err != nil {
		return err
	}
	bt.unifiedUnlocked = unlocked

	return nil
}

// RecordKill records a boss kill and returns whether it was the first kill.
func (bt *BossTracker) RecordKill(towerID TowerID, playerName string) (bool, error) {
	isFirstKill, err := bt.store.RecordBossKill(string(towerID), playerName)
	if err != nil {
		return false, err
	}

	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Update cache
	bt.defeatedTowers[towerID] = true

	// Check if this unlocks the unified tower
	if isFirstKill && towerID != TowerUnified {
		allDefeated := true
		for _, id := range AllRacialTowers {
			if !bt.defeatedTowers[id] {
				allDefeated = false
				break
			}
		}
		if allDefeated {
			bt.unifiedUnlocked = true
		}
	}

	return isFirstKill, nil
}

// GetFirstKill returns the first kill record for a tower.
func (bt *BossTracker) GetFirstKill(towerID TowerID) (*BossKill, error) {
	dbKill, err := bt.store.GetFirstKill(string(towerID))
	if err != nil {
		return nil, err
	}
	if dbKill == nil {
		return nil, nil
	}
	return &BossKill{
		ID:          dbKill.ID,
		TowerID:     TowerID(dbKill.TowerID),
		PlayerName:  dbKill.PlayerName,
		KilledAt:    dbKill.KilledAt,
		IsFirstKill: dbKill.IsFirstKill,
	}, nil
}

// GetAllFirstKills returns all first kill records.
func (bt *BossTracker) GetAllFirstKills() ([]BossKill, error) {
	dbKills, err := bt.store.GetAllFirstKills()
	if err != nil {
		return nil, err
	}
	kills := make([]BossKill, len(dbKills))
	for i, dbKill := range dbKills {
		kills[i] = BossKill{
			ID:          dbKill.ID,
			TowerID:     TowerID(dbKill.TowerID),
			PlayerName:  dbKill.PlayerName,
			KilledAt:    dbKill.KilledAt,
			IsFirstKill: dbKill.IsFirstKill,
		}
	}
	return kills, nil
}

// HasBeenDefeated returns true if a tower boss has ever been defeated.
func (bt *BossTracker) HasBeenDefeated(towerID TowerID) bool {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.defeatedTowers[towerID]
}

// HasPlayerKilled returns true if a player has killed a specific tower boss.
func (bt *BossTracker) HasPlayerKilled(towerID TowerID, playerName string) (bool, error) {
	return bt.store.HasPlayerKilledBoss(string(towerID), playerName)
}

// GetPlayerKills returns all boss kills for a player.
func (bt *BossTracker) GetPlayerKills(playerName string) ([]BossKill, error) {
	dbKills, err := bt.store.GetPlayerBossKills(playerName)
	if err != nil {
		return nil, err
	}
	kills := make([]BossKill, len(dbKills))
	for i, dbKill := range dbKills {
		kills[i] = BossKill{
			ID:          dbKill.ID,
			TowerID:     TowerID(dbKill.TowerID),
			PlayerName:  dbKill.PlayerName,
			KilledAt:    dbKill.KilledAt,
			IsFirstKill: dbKill.IsFirstKill,
		}
	}
	return kills, nil
}

// GetTowerKills returns all kills for a specific tower boss.
func (bt *BossTracker) GetTowerKills(towerID TowerID) ([]BossKill, error) {
	dbKills, err := bt.store.GetTowerBossKills(string(towerID))
	if err != nil {
		return nil, err
	}
	kills := make([]BossKill, len(dbKills))
	for i, dbKill := range dbKills {
		kills[i] = BossKill{
			ID:          dbKill.ID,
			TowerID:     TowerID(dbKill.TowerID),
			PlayerName:  dbKill.PlayerName,
			KilledAt:    dbKill.KilledAt,
			IsFirstKill: dbKill.IsFirstKill,
		}
	}
	return kills, nil
}

// GetDefeatedTowerCount returns the number of unique towers defeated.
func (bt *BossTracker) GetDefeatedTowerCount() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	count := 0
	for _, defeated := range bt.defeatedTowers {
		if defeated {
			count++
		}
	}
	return count
}

// GetDefeatedRacialTowerCount returns the number of racial towers defeated.
func (bt *BossTracker) GetDefeatedRacialTowerCount() int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	count := 0
	for _, towerID := range AllRacialTowers {
		if bt.defeatedTowers[towerID] {
			count++
		}
	}
	return count
}

// IsUnifiedUnlocked returns true if the unified tower is unlocked.
func (bt *BossTracker) IsUnifiedUnlocked() bool {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.unifiedUnlocked
}

// GetDefeatedTowers returns a list of all defeated tower IDs.
func (bt *BossTracker) GetDefeatedTowers() []TowerID {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var defeated []TowerID
	for towerID, isDefeated := range bt.defeatedTowers {
		if isDefeated {
			defeated = append(defeated, towerID)
		}
	}
	return defeated
}

// GetUndefeatedRacialTowers returns a list of racial towers not yet defeated.
func (bt *BossTracker) GetUndefeatedRacialTowers() []TowerID {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	var undefeated []TowerID
	for _, towerID := range AllRacialTowers {
		if !bt.defeatedTowers[towerID] {
			undefeated = append(undefeated, towerID)
		}
	}
	return undefeated
}
