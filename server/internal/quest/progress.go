package quest

import (
	"encoding/json"
	"sync"
)

// QuestStatus represents the status of an active quest
type QuestStatus string

const (
	QuestStatusActive    QuestStatus = "active"    // Quest in progress
	QuestStatusCompleted QuestStatus = "completed" // Objectives done, ready for turn-in
)

// ObjectiveProgress tracks progress on a single objective
type ObjectiveProgress struct {
	Current int `json:"current"` // Current count achieved
}

// QuestProgress tracks a player's progress on a specific quest
type QuestProgress struct {
	QuestID    string              `json:"quest_id"`
	Status     QuestStatus         `json:"status"`
	Objectives []ObjectiveProgress `json:"objectives"`
}

// PlayerQuestLog manages all quests for a player
type PlayerQuestLog struct {
	mu        sync.RWMutex
	Active    map[string]*QuestProgress `json:"active"`    // Currently active quests
	Completed map[string]bool           `json:"completed"` // Quest IDs that have been turned in
}

// NewPlayerQuestLog creates an empty quest log
func NewPlayerQuestLog() *PlayerQuestLog {
	return &PlayerQuestLog{
		Active:    make(map[string]*QuestProgress),
		Completed: make(map[string]bool),
	}
}

// ToJSON serializes quest log for database storage
func (pql *PlayerQuestLog) ToJSON() string {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	data, err := json.Marshal(pql)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// PlayerQuestLogFromJSON deserializes quest log from database
func PlayerQuestLogFromJSON(data string) (*PlayerQuestLog, error) {
	if data == "" || data == "{}" {
		return NewPlayerQuestLog(), nil
	}

	pql := &PlayerQuestLog{}
	if err := json.Unmarshal([]byte(data), pql); err != nil {
		return NewPlayerQuestLog(), err
	}

	// Initialize maps if nil (from old data)
	if pql.Active == nil {
		pql.Active = make(map[string]*QuestProgress)
	}
	if pql.Completed == nil {
		pql.Completed = make(map[string]bool)
	}

	return pql, nil
}

// StartQuest begins tracking a quest
func (pql *PlayerQuestLog) StartQuest(quest *Quest) error {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	// Initialize objective progress for each objective
	objectives := make([]ObjectiveProgress, len(quest.Objectives))
	for i := range objectives {
		objectives[i] = ObjectiveProgress{Current: 0}
	}

	pql.Active[quest.ID] = &QuestProgress{
		QuestID:    quest.ID,
		Status:     QuestStatusActive,
		Objectives: objectives,
	}

	return nil
}

// GetQuestProgress returns progress for a specific quest
func (pql *PlayerQuestLog) GetQuestProgress(questID string) (*QuestProgress, bool) {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	progress, exists := pql.Active[questID]
	return progress, exists
}

// GetActiveQuests returns all active quest IDs
func (pql *PlayerQuestLog) GetActiveQuests() []string {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	quests := make([]string, 0, len(pql.Active))
	for questID := range pql.Active {
		quests = append(quests, questID)
	}
	return quests
}

// HasActiveQuest checks if a quest is currently active
func (pql *PlayerQuestLog) HasActiveQuest(questID string) bool {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	_, exists := pql.Active[questID]
	return exists
}

// HasCompletedQuest checks if a quest was previously turned in
func (pql *PlayerQuestLog) HasCompletedQuest(questID string) bool {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	return pql.Completed[questID]
}

// UpdateKillProgress increments kill count for relevant quests
// target is the mob ID that was killed, name is the display name
func (pql *PlayerQuestLog) UpdateKillProgress(target string, name string) {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	for _, progress := range pql.Active {
		if progress.Status != QuestStatusActive {
			continue
		}
		// Note: We need the quest definition to check objectives
		// This will be called with quest context from the registry
	}
}

// UpdateKillProgressForQuest updates kill progress for a specific quest
func (pql *PlayerQuestLog) UpdateKillProgressForQuest(questID string, quest *Quest, killedMobID string) bool {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	progress, exists := pql.Active[questID]
	if !exists || progress.Status != QuestStatusActive {
		return false
	}

	updated := false
	for i, obj := range quest.Objectives {
		if obj.Type != QuestTypeKill {
			continue
		}
		// Empty target means any mob counts
		if obj.Target == "" || obj.Target == killedMobID {
			if progress.Objectives[i].Current < obj.Required {
				progress.Objectives[i].Current++
				updated = true
			}
		}
	}

	// Check if all objectives are complete
	if updated {
		pql.checkQuestCompletion(progress, quest)
	}

	return updated
}

// UpdateItemProgressForQuest updates item collection progress for a specific quest
func (pql *PlayerQuestLog) UpdateItemProgressForQuest(questID string, quest *Quest, itemID string) bool {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	progress, exists := pql.Active[questID]
	if !exists || progress.Status != QuestStatusActive {
		return false
	}

	updated := false
	for i, obj := range quest.Objectives {
		if obj.Type != QuestTypeFetch {
			continue
		}
		if obj.Target == itemID {
			if progress.Objectives[i].Current < obj.Required {
				progress.Objectives[i].Current++
				updated = true
			}
		}
	}

	if updated {
		pql.checkQuestCompletion(progress, quest)
	}

	return updated
}

// UpdateExploreProgressForQuest updates room exploration progress for a specific quest
func (pql *PlayerQuestLog) UpdateExploreProgressForQuest(questID string, quest *Quest, roomID string) bool {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	progress, exists := pql.Active[questID]
	if !exists || progress.Status != QuestStatusActive {
		return false
	}

	updated := false
	for i, obj := range quest.Objectives {
		if obj.Type != QuestTypeExplore {
			continue
		}
		if obj.Target == roomID {
			if progress.Objectives[i].Current < obj.Required {
				progress.Objectives[i].Current++
				updated = true
			}
		}
	}

	if updated {
		pql.checkQuestCompletion(progress, quest)
	}

	return updated
}

// UpdateCraftProgressForQuest updates crafting progress for a specific quest
func (pql *PlayerQuestLog) UpdateCraftProgressForQuest(questID string, quest *Quest, itemID string) bool {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	progress, exists := pql.Active[questID]
	if !exists || progress.Status != QuestStatusActive {
		return false
	}

	updated := false
	for i, obj := range quest.Objectives {
		if obj.Type != QuestTypeCraft {
			continue
		}
		if obj.Target == itemID {
			if progress.Objectives[i].Current < obj.Required {
				progress.Objectives[i].Current++
				updated = true
			}
		}
	}

	if updated {
		pql.checkQuestCompletion(progress, quest)
	}

	return updated
}

// UpdateCastProgressForQuest updates spell casting progress for a specific quest
func (pql *PlayerQuestLog) UpdateCastProgressForQuest(questID string, quest *Quest, spellID string) bool {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	progress, exists := pql.Active[questID]
	if !exists || progress.Status != QuestStatusActive {
		return false
	}

	updated := false
	for i, obj := range quest.Objectives {
		if obj.Type != QuestTypeCast {
			continue
		}
		// Empty target means any spell counts
		if obj.Target == "" || obj.Target == spellID {
			if progress.Objectives[i].Current < obj.Required {
				progress.Objectives[i].Current++
				updated = true
			}
		}
	}

	if updated {
		pql.checkQuestCompletion(progress, quest)
	}

	return updated
}

// checkQuestCompletion checks if all objectives are met and updates status
// Must be called with lock held
func (pql *PlayerQuestLog) checkQuestCompletion(progress *QuestProgress, quest *Quest) {
	for i, obj := range quest.Objectives {
		if progress.Objectives[i].Current < obj.Required {
			return // Not all objectives complete
		}
	}
	progress.Status = QuestStatusCompleted
}

// CanCompleteQuest checks if all objectives are met for a quest
func (pql *PlayerQuestLog) CanCompleteQuest(questID string, quest *Quest) bool {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	progress, exists := pql.Active[questID]
	if !exists {
		return false
	}

	// Check if marked as completed
	if progress.Status == QuestStatusCompleted {
		return true
	}

	// Double-check all objectives
	for i, obj := range quest.Objectives {
		if i >= len(progress.Objectives) {
			return false
		}
		if progress.Objectives[i].Current < obj.Required {
			return false
		}
	}

	return true
}

// TurnInQuest finalizes quest and moves to completed
func (pql *PlayerQuestLog) TurnInQuest(questID string, repeatable bool) error {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	// Remove from active
	delete(pql.Active, questID)

	// Add to completed only if not repeatable
	if !repeatable {
		pql.Completed[questID] = true
	}

	return nil
}

// AbandonQuest removes a quest from active without completing it
func (pql *PlayerQuestLog) AbandonQuest(questID string) error {
	pql.mu.Lock()
	defer pql.mu.Unlock()

	delete(pql.Active, questID)
	return nil
}

// GetCompletedQuests returns all completed quest IDs
func (pql *PlayerQuestLog) GetCompletedQuests() []string {
	pql.mu.RLock()
	defer pql.mu.RUnlock()

	quests := make([]string, 0, len(pql.Completed))
	for questID := range pql.Completed {
		quests = append(quests, questID)
	}
	return quests
}
