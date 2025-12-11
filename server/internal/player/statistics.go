package player

import (
	"encoding/json"
	"sync"
)

// PlayerStatistics tracks player activity for the companion website.
type PlayerStatistics struct {
	TotalKills       int            `json:"total_kills"`
	MobKills         map[string]int `json:"mob_kills"`          // mob_id -> count
	HighestFloor     map[string]int `json:"highest_floor"`      // tower_id -> floor
	GoldAccumulated  int64          `json:"gold_accumulated"`   // Lifetime gold earned
	QuestsCompleted  int            `json:"quests_completed"`
	Deaths           int            `json:"deaths"`
	DamageDealt      int64          `json:"damage_dealt"`
	DamageTaken      int64          `json:"damage_taken"`
	ItemsCrafted     int            `json:"items_crafted"`
	SpellsCast       int            `json:"spells_cast"`
	DistanceTraveled int            `json:"distance_traveled"` // Room moves
	mu               sync.RWMutex
}

// NewPlayerStatistics creates a new statistics tracker.
func NewPlayerStatistics() *PlayerStatistics {
	return &PlayerStatistics{
		MobKills:     make(map[string]int),
		HighestFloor: make(map[string]int),
	}
}

// RecordKill increments kill counts.
func (s *PlayerStatistics) RecordKill(mobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalKills++
	if s.MobKills == nil {
		s.MobKills = make(map[string]int)
	}
	s.MobKills[mobID]++
}

// RecordFloorReached updates highest floor if this is higher.
func (s *PlayerStatistics) RecordFloorReached(towerID string, floor int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.HighestFloor == nil {
		s.HighestFloor = make(map[string]int)
	}
	if floor > s.HighestFloor[towerID] {
		s.HighestFloor[towerID] = floor
	}
}

// RecordGoldEarned adds to lifetime gold earned.
func (s *PlayerStatistics) RecordGoldEarned(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GoldAccumulated += int64(amount)
}

// RecordQuestCompleted increments quest completion count.
func (s *PlayerStatistics) RecordQuestCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.QuestsCompleted++
}

// RecordDeath increments death count.
func (s *PlayerStatistics) RecordDeath() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Deaths++
}

// RecordDamageDealt adds to total damage dealt.
func (s *PlayerStatistics) RecordDamageDealt(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DamageDealt += int64(amount)
}

// RecordDamageTaken adds to total damage taken.
func (s *PlayerStatistics) RecordDamageTaken(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DamageTaken += int64(amount)
}

// RecordItemCrafted increments items crafted count.
func (s *PlayerStatistics) RecordItemCrafted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ItemsCrafted++
}

// RecordSpellCast increments spells cast count.
func (s *PlayerStatistics) RecordSpellCast() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SpellsCast++
}

// RecordMove increments distance traveled.
func (s *PlayerStatistics) RecordMove() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DistanceTraveled++
}

// GetTotalKills returns total kill count.
func (s *PlayerStatistics) GetTotalKills() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalKills
}

// GetHighestFloor returns highest floor reached for a tower.
func (s *PlayerStatistics) GetHighestFloor(towerID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.HighestFloor == nil {
		return 0
	}
	return s.HighestFloor[towerID]
}

// ToJSON serializes statistics to JSON.
func (s *PlayerStatistics) ToJSON() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FromJSON deserializes statistics from JSON.
func (s *PlayerStatistics) FromJSON(jsonStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), s)
}
