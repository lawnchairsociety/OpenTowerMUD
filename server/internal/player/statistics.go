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

	// Achievement tracking fields
	PortalsUsed           int             `json:"portals_used"`             // Portal usage count
	CitiesVisited         map[string]bool `json:"cities_visited"`           // city_id -> visited
	SecretRoomsFound      int             `json:"secret_rooms_found"`       // Secret room discovery count
	LabyrinthCompleted    bool            `json:"labyrinth_completed"`      // Successfully navigated labyrinth
	TotalPlayTimeSeconds  int64           `json:"total_play_time_seconds"`  // Total play time in seconds
	TowerClearsWithoutDeath map[string]int `json:"tower_clears_without_death"` // tower_id -> count of deathless clears

	mu sync.RWMutex
}

// NewPlayerStatistics creates a new statistics tracker.
func NewPlayerStatistics() *PlayerStatistics {
	return &PlayerStatistics{
		MobKills:                make(map[string]int),
		HighestFloor:            make(map[string]int),
		CitiesVisited:           make(map[string]bool),
		TowerClearsWithoutDeath: make(map[string]int),
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

// RecordPortalUsed increments portal usage count.
func (s *PlayerStatistics) RecordPortalUsed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PortalsUsed++
}

// RecordCityVisited marks a city as visited.
func (s *PlayerStatistics) RecordCityVisited(cityID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.CitiesVisited == nil {
		s.CitiesVisited = make(map[string]bool)
	}
	s.CitiesVisited[cityID] = true
}

// HasVisitedAllCities checks if all 5 racial cities have been visited.
func (s *PlayerStatistics) HasVisitedAllCities() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.CitiesVisited == nil {
		return false
	}
	cities := []string{"human", "elf", "dwarf", "gnome", "orc"}
	for _, city := range cities {
		if !s.CitiesVisited[city] {
			return false
		}
	}
	return true
}

// RecordSecretRoomFound increments secret room discovery count.
func (s *PlayerStatistics) RecordSecretRoomFound() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SecretRoomsFound++
}

// RecordLabyrinthCompleted marks the labyrinth as completed.
func (s *PlayerStatistics) RecordLabyrinthCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LabyrinthCompleted = true
}

// AddPlayTime adds seconds to total play time.
func (s *PlayerStatistics) AddPlayTime(seconds int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPlayTimeSeconds += seconds
}

// GetTotalPlayTimeHours returns total play time in hours.
func (s *PlayerStatistics) GetTotalPlayTimeHours() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return float64(s.TotalPlayTimeSeconds) / 3600.0
}

// RecordTowerClearWithoutDeath records a deathless tower clear.
func (s *PlayerStatistics) RecordTowerClearWithoutDeath(towerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.TowerClearsWithoutDeath == nil {
		s.TowerClearsWithoutDeath = make(map[string]int)
	}
	s.TowerClearsWithoutDeath[towerID]++
}

// HasDeathlessClear checks if player has any deathless tower clear.
func (s *PlayerStatistics) HasDeathlessClear() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.TowerClearsWithoutDeath == nil {
		return false
	}
	for _, count := range s.TowerClearsWithoutDeath {
		if count > 0 {
			return true
		}
	}
	return false
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
