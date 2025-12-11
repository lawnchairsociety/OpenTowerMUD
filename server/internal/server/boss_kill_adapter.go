package server

import (
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
)

// BossKillAdapter adapts database.Database to tower.BossKillStore interface.
type BossKillAdapter struct {
	db *database.Database
}

// NewBossKillAdapter creates a new adapter wrapping the database.
func NewBossKillAdapter(db *database.Database) *BossKillAdapter {
	return &BossKillAdapter{db: db}
}

func (a *BossKillAdapter) RecordBossKill(towerID, playerName string) (bool, error) {
	return a.db.RecordBossKill(towerID, playerName)
}

func (a *BossKillAdapter) GetFirstKill(towerID string) (*tower.BossKillDB, error) {
	kill, err := a.db.GetFirstKill(towerID)
	if err != nil {
		return nil, err
	}
	if kill == nil {
		return nil, nil
	}
	return &tower.BossKillDB{
		ID:          kill.ID,
		TowerID:     kill.TowerID,
		PlayerName:  kill.PlayerName,
		KilledAt:    kill.KilledAt,
		IsFirstKill: kill.IsFirstKill,
	}, nil
}

func (a *BossKillAdapter) GetAllFirstKills() ([]tower.BossKillDB, error) {
	kills, err := a.db.GetAllFirstKills()
	if err != nil {
		return nil, err
	}
	result := make([]tower.BossKillDB, len(kills))
	for i, kill := range kills {
		result[i] = tower.BossKillDB{
			ID:          kill.ID,
			TowerID:     kill.TowerID,
			PlayerName:  kill.PlayerName,
			KilledAt:    kill.KilledAt,
			IsFirstKill: kill.IsFirstKill,
		}
	}
	return result, nil
}

func (a *BossKillAdapter) HasTowerBeenDefeated(towerID string) (bool, error) {
	return a.db.HasTowerBeenDefeated(towerID)
}

func (a *BossKillAdapter) HasPlayerKilledBoss(towerID, playerName string) (bool, error) {
	return a.db.HasPlayerKilledBoss(towerID, playerName)
}

func (a *BossKillAdapter) GetPlayerBossKills(playerName string) ([]tower.BossKillDB, error) {
	kills, err := a.db.GetPlayerBossKills(playerName)
	if err != nil {
		return nil, err
	}
	result := make([]tower.BossKillDB, len(kills))
	for i, kill := range kills {
		result[i] = tower.BossKillDB{
			ID:          kill.ID,
			TowerID:     kill.TowerID,
			PlayerName:  kill.PlayerName,
			KilledAt:    kill.KilledAt,
			IsFirstKill: kill.IsFirstKill,
		}
	}
	return result, nil
}

func (a *BossKillAdapter) GetTowerBossKills(towerID string) ([]tower.BossKillDB, error) {
	kills, err := a.db.GetTowerBossKills(towerID)
	if err != nil {
		return nil, err
	}
	result := make([]tower.BossKillDB, len(kills))
	for i, kill := range kills {
		result[i] = tower.BossKillDB{
			ID:          kill.ID,
			TowerID:     kill.TowerID,
			PlayerName:  kill.PlayerName,
			KilledAt:    kill.KilledAt,
			IsFirstKill: kill.IsFirstKill,
		}
	}
	return result, nil
}

func (a *BossKillAdapter) IsUnifiedTowerUnlocked() (bool, error) {
	return a.db.IsUnifiedTowerUnlocked()
}
