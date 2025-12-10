package quest

import (
	"sync"
)

// QuestRegistry holds all loaded quest definitions
type QuestRegistry struct {
	mu          sync.RWMutex
	quests      map[string]*Quest   // questID -> Quest
	questsByNPC map[string][]*Quest // npcID -> quests they give
}

// NewQuestRegistry creates a new registry
func NewQuestRegistry() *QuestRegistry {
	return &QuestRegistry{
		quests:      make(map[string]*Quest),
		questsByNPC: make(map[string][]*Quest),
	}
}

// LoadFromConfig populates registry from QuestsConfig
func (r *QuestRegistry) LoadFromConfig(config *QuestsConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing data
	r.quests = make(map[string]*Quest)
	r.questsByNPC = make(map[string][]*Quest)

	// Load all quests
	for id, def := range config.Quests {
		quest := createQuestFromDefinition(id, &def)
		r.quests[id] = quest

		// Index by giver NPC
		if quest.GiverNPC != "" {
			r.questsByNPC[quest.GiverNPC] = append(r.questsByNPC[quest.GiverNPC], quest)
		}
	}
}

// GetQuest returns a quest by ID
func (r *QuestRegistry) GetQuest(id string) (*Quest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	quest, exists := r.quests[id]
	return quest, exists
}

// GetQuestsForNPC returns all quests an NPC can give
func (r *QuestRegistry) GetQuestsForNPC(npcID string) []*Quest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	quests := r.questsByNPC[npcID]
	if quests == nil {
		return []*Quest{}
	}

	// Return a copy to prevent modification
	result := make([]*Quest, len(quests))
	copy(result, quests)
	return result
}

// PlayerQuestState contains player state needed for filtering quests
type PlayerQuestState struct {
	Level           int
	ActiveClass     string
	ClassLevels     map[string]int // class -> level
	CraftingSkills  map[string]int // skill -> level
	CompletedQuests map[string]bool
	ActiveQuests    map[string]bool
}

// GetAvailableQuestsForPlayer returns quests player can accept from an NPC
func (r *QuestRegistry) GetAvailableQuestsForPlayer(npcID string, state *PlayerQuestState) []*Quest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allQuests := r.questsByNPC[npcID]
	if allQuests == nil {
		return []*Quest{}
	}

	available := make([]*Quest, 0)

	for _, quest := range allQuests {
		if r.isQuestAvailable(quest, state) {
			available = append(available, quest)
		}
	}

	return available
}

// isQuestAvailable checks if a player can accept a quest
func (r *QuestRegistry) isQuestAvailable(quest *Quest, state *PlayerQuestState) bool {
	// Check if already active
	if state.ActiveQuests[quest.ID] {
		return false
	}

	// Check if already completed (and not repeatable)
	if state.CompletedQuests[quest.ID] && !quest.Repeatable {
		return false
	}

	// Check minimum level
	if quest.MinLevel > 0 && state.Level < quest.MinLevel {
		return false
	}

	// Check prerequisites
	for _, prereqID := range quest.Prereqs {
		if !state.CompletedQuests[prereqID] {
			return false
		}
	}

	// Check class requirements (class quests only available for active class)
	if quest.RequiredClass != "" {
		if state.ActiveClass != quest.RequiredClass {
			return false
		}
		classLevel := state.ClassLevels[quest.RequiredClass]
		if classLevel < quest.RequiredClassLevel {
			return false
		}
	}

	// Check crafting requirements
	if quest.RequiredCraftingSkill != "" {
		skillLevel := state.CraftingSkills[quest.RequiredCraftingSkill]
		if skillLevel < quest.RequiredCraftingLevel {
			return false
		}
	}

	return true
}

// GetAllQuests returns all registered quests
func (r *QuestRegistry) GetAllQuests() []*Quest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	quests := make([]*Quest, 0, len(r.quests))
	for _, quest := range r.quests {
		quests = append(quests, quest)
	}
	return quests
}

// GetQuestCount returns the number of registered quests
func (r *QuestRegistry) GetQuestCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.quests)
}

// Count returns the number of registered quests (alias for GetQuestCount)
func (r *QuestRegistry) Count() int {
	return r.GetQuestCount()
}

// LoadFromYAML loads quests from a YAML file
func (r *QuestRegistry) LoadFromYAML(filename string) error {
	config, err := LoadQuestsFromYAML(filename)
	if err != nil {
		return err
	}
	r.LoadFromConfig(config)
	return nil
}

// LoadFromDirectory loads quests from all YAML files in a directory
func (r *QuestRegistry) LoadFromDirectory(dir string) error {
	config, err := LoadQuestsFromDirectory(dir)
	if err != nil {
		return err
	}
	r.LoadFromConfig(config)
	return nil
}
