package database

import (
	"database/sql"
	"time"
)

// BossKill represents a recorded boss kill.
type BossKill struct {
	ID          int64
	TowerID     string
	PlayerName  string
	KilledAt    time.Time
	IsFirstKill bool
}

// RecordBossKill records a boss kill in the database.
// Returns whether this was the first kill for this tower.
func (d *Database) RecordBossKill(towerID, playerName string) (bool, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// Check if this tower has been defeated before
	var count int
	err = tx.QueryRow(`SELECT COUNT(*) FROM boss_kills WHERE tower_id = ?`, towerID).Scan(&count)
	if err != nil {
		return false, err
	}

	isFirstKill := count == 0

	// Record the kill
	_, err = tx.Exec(`
		INSERT INTO boss_kills (tower_id, player_name, killed_at, is_first_kill)
		VALUES (?, ?, ?, ?)
	`, towerID, playerName, time.Now(), isFirstKill)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return isFirstKill, nil
}

// GetFirstKill returns the first kill record for a tower, or nil if never defeated.
func (d *Database) GetFirstKill(towerID string) (*BossKill, error) {
	row := d.db.QueryRow(`
		SELECT id, tower_id, player_name, killed_at, is_first_kill
		FROM boss_kills
		WHERE tower_id = ? AND is_first_kill = 1
		LIMIT 1
	`, towerID)

	kill := &BossKill{}
	err := row.Scan(&kill.ID, &kill.TowerID, &kill.PlayerName, &kill.KilledAt, &kill.IsFirstKill)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return kill, nil
}

// GetAllFirstKills returns all first kill records.
func (d *Database) GetAllFirstKills() ([]BossKill, error) {
	rows, err := d.db.Query(`
		SELECT id, tower_id, player_name, killed_at, is_first_kill
		FROM boss_kills
		WHERE is_first_kill = 1
		ORDER BY killed_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kills []BossKill
	for rows.Next() {
		var kill BossKill
		if err := rows.Scan(&kill.ID, &kill.TowerID, &kill.PlayerName, &kill.KilledAt, &kill.IsFirstKill); err != nil {
			return nil, err
		}
		kills = append(kills, kill)
	}
	return kills, rows.Err()
}

// HasTowerBeenDefeated returns true if a tower boss has ever been defeated.
func (d *Database) HasTowerBeenDefeated(towerID string) (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM boss_kills WHERE tower_id = ?`, towerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasPlayerKilledBoss returns true if a player has killed a specific tower boss.
func (d *Database) HasPlayerKilledBoss(towerID, playerName string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM boss_kills
		WHERE tower_id = ? AND player_name = ?
	`, towerID, playerName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetPlayerBossKills returns all boss kills for a player.
func (d *Database) GetPlayerBossKills(playerName string) ([]BossKill, error) {
	rows, err := d.db.Query(`
		SELECT id, tower_id, player_name, killed_at, is_first_kill
		FROM boss_kills
		WHERE player_name = ?
		ORDER BY killed_at ASC
	`, playerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kills []BossKill
	for rows.Next() {
		var kill BossKill
		if err := rows.Scan(&kill.ID, &kill.TowerID, &kill.PlayerName, &kill.KilledAt, &kill.IsFirstKill); err != nil {
			return nil, err
		}
		kills = append(kills, kill)
	}
	return kills, rows.Err()
}

// GetTowerBossKills returns all kills for a specific tower boss.
func (d *Database) GetTowerBossKills(towerID string) ([]BossKill, error) {
	rows, err := d.db.Query(`
		SELECT id, tower_id, player_name, killed_at, is_first_kill
		FROM boss_kills
		WHERE tower_id = ?
		ORDER BY killed_at ASC
	`, towerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kills []BossKill
	for rows.Next() {
		var kill BossKill
		if err := rows.Scan(&kill.ID, &kill.TowerID, &kill.PlayerName, &kill.KilledAt, &kill.IsFirstKill); err != nil {
			return nil, err
		}
		kills = append(kills, kill)
	}
	return kills, rows.Err()
}

// GetDefeatedTowerCount returns the number of unique towers that have been defeated.
func (d *Database) GetDefeatedTowerCount() (int, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(DISTINCT tower_id) FROM boss_kills
	`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetDefeatedRacialTowerCount returns the number of racial towers defeated.
func (d *Database) GetDefeatedRacialTowerCount() (int, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(DISTINCT tower_id) FROM boss_kills
		WHERE tower_id IN ('human', 'elf', 'dwarf', 'gnome', 'orc')
	`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IsUnifiedTowerUnlocked returns true if all 5 racial tower bosses have been defeated.
func (d *Database) IsUnifiedTowerUnlocked() (bool, error) {
	count, err := d.GetDefeatedRacialTowerCount()
	if err != nil {
		return false, err
	}
	return count >= 5, nil
}

// GetBossKillLeaderboard returns players sorted by number of boss kills.
func (d *Database) GetBossKillLeaderboard(limit int) ([]struct {
	PlayerName string
	KillCount  int
}, error) {
	rows, err := d.db.Query(`
		SELECT player_name, COUNT(*) as kills
		FROM boss_kills
		GROUP BY player_name
		ORDER BY kills DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		PlayerName string
		KillCount  int
	}
	for rows.Next() {
		var entry struct {
			PlayerName string
			KillCount  int
		}
		if err := rows.Scan(&entry.PlayerName, &entry.KillCount); err != nil {
			return nil, err
		}
		results = append(results, entry)
	}
	return results, rows.Err()
}
