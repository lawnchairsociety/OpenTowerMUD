package quest

import (
	"testing"
)

func TestNewQuestRegistry(t *testing.T) {
	registry := NewQuestRegistry()

	if registry == nil {
		t.Fatal("NewQuestRegistry returned nil")
	}
	if registry.quests == nil {
		t.Error("quests map should be initialized")
	}
	if registry.questsByNPC == nil {
		t.Error("questsByNPC map should be initialized")
	}
}

func TestLoadFromConfig(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {
				Name:      "Quest One",
				GiverNPC:  "npc1",
				TurnInNPC: "npc1",
			},
			"quest2": {
				Name:      "Quest Two",
				GiverNPC:  "npc1",
				TurnInNPC: "npc2",
			},
			"quest3": {
				Name:      "Quest Three",
				GiverNPC:  "npc2",
				TurnInNPC: "npc2",
			},
		},
	}

	registry.LoadFromConfig(config)

	if registry.GetQuestCount() != 3 {
		t.Errorf("Should have 3 quests, got %d", registry.GetQuestCount())
	}

	// npc1 gives quest1 and quest2
	npc1Quests := registry.GetQuestsForNPC("npc1")
	if len(npc1Quests) != 2 {
		t.Errorf("npc1 should give 2 quests, got %d", len(npc1Quests))
	}

	// npc2 gives quest3
	npc2Quests := registry.GetQuestsForNPC("npc2")
	if len(npc2Quests) != 1 {
		t.Errorf("npc2 should give 1 quest, got %d", len(npc2Quests))
	}
}

func TestGetQuest(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"test_quest": {
				Name:        "Test Quest",
				Description: "A test quest",
			},
		},
	}

	registry.LoadFromConfig(config)

	quest, exists := registry.GetQuest("test_quest")
	if !exists {
		t.Fatal("Quest should exist")
	}
	if quest.Name != "Test Quest" {
		t.Errorf("Quest name mismatch: got %s, want Test Quest", quest.Name)
	}

	_, exists = registry.GetQuest("nonexistent")
	if exists {
		t.Error("Nonexistent quest should not exist")
	}
}

func TestGetQuestsForNPC(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {Name: "Quest 1", GiverNPC: "npc1"},
			"quest2": {Name: "Quest 2", GiverNPC: "npc1"},
		},
	}

	registry.LoadFromConfig(config)

	quests := registry.GetQuestsForNPC("npc1")
	if len(quests) != 2 {
		t.Errorf("Should have 2 quests, got %d", len(quests))
	}

	// Unknown NPC should return empty slice
	quests = registry.GetQuestsForNPC("unknown_npc")
	if len(quests) != 0 {
		t.Errorf("Unknown NPC should have 0 quests, got %d", len(quests))
	}
}

func TestGetAvailableQuestsForPlayer_FiltersByLevel(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"low_level":  {Name: "Low Level", GiverNPC: "npc", MinLevel: 1},
			"mid_level":  {Name: "Mid Level", GiverNPC: "npc", MinLevel: 10},
			"high_level": {Name: "High Level", GiverNPC: "npc", MinLevel: 20},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           5,
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    make(map[string]bool),
		ClassLevels:     make(map[string]int),
		CraftingSkills:  make(map[string]int),
	}

	available := registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 1 {
		t.Errorf("Level 5 player should see 1 quest, got %d", len(available))
	}
	if available[0].Name != "Low Level" {
		t.Errorf("Available quest should be Low Level, got %s", available[0].Name)
	}

	state.Level = 15
	available = registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 2 {
		t.Errorf("Level 15 player should see 2 quests, got %d", len(available))
	}
}

func TestGetAvailableQuestsForPlayer_FiltersByPrereqs(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"first_quest":  {Name: "First Quest", GiverNPC: "npc"},
			"second_quest": {Name: "Second Quest", GiverNPC: "npc", Prereqs: []string{"first_quest"}},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    make(map[string]bool),
		ClassLevels:     make(map[string]int),
		CraftingSkills:  make(map[string]int),
	}

	// Without prereq complete
	available := registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 1 {
		t.Errorf("Should only see first quest without prereq, got %d", len(available))
	}

	// Complete prereq
	state.CompletedQuests["first_quest"] = true
	available = registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 1 {
		t.Errorf("Should see second quest after completing prereq, got %d", len(available))
	}
	if available[0].Name != "Second Quest" {
		t.Errorf("Available quest should be Second Quest, got %s", available[0].Name)
	}
}

func TestGetAvailableQuestsForPlayer_FiltersByClass(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"warrior_quest": {
				Name:               "Warrior Quest",
				GiverNPC:           "trainer",
				Category:           "class",
				RequiredClass:      "warrior",
				RequiredClassLevel: 5,
			},
			"mage_quest": {
				Name:               "Mage Quest",
				GiverNPC:           "trainer",
				Category:           "class",
				RequiredClass:      "mage",
				RequiredClassLevel: 5,
			},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		ActiveClass:     "warrior",
		ClassLevels:     map[string]int{"warrior": 10, "mage": 3},
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    make(map[string]bool),
		CraftingSkills:  make(map[string]int),
	}

	available := registry.GetAvailableQuestsForPlayer("trainer", state)
	if len(available) != 1 {
		t.Errorf("Warrior should see 1 quest, got %d", len(available))
	}
	if available[0].Name != "Warrior Quest" {
		t.Errorf("Should see Warrior Quest, got %s", available[0].Name)
	}

	// Switch to mage (but level too low)
	state.ActiveClass = "mage"
	available = registry.GetAvailableQuestsForPlayer("trainer", state)
	if len(available) != 0 {
		t.Errorf("Mage with level 3 should see 0 quests, got %d", len(available))
	}

	// Level up mage
	state.ClassLevels["mage"] = 5
	available = registry.GetAvailableQuestsForPlayer("trainer", state)
	if len(available) != 1 {
		t.Errorf("Mage with level 5 should see 1 quest, got %d", len(available))
	}
}

func TestGetAvailableQuestsForPlayer_FiltersByCraftingSkill(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"smithing_quest": {
				Name:                  "Smithing Quest",
				GiverNPC:              "smith",
				Category:              "crafting",
				RequiredCraftingSkill: "blacksmithing",
				RequiredCraftingLevel: 10,
			},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		CraftingSkills:  map[string]int{"blacksmithing": 5},
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    make(map[string]bool),
		ClassLevels:     make(map[string]int),
	}

	// Skill too low
	available := registry.GetAvailableQuestsForPlayer("smith", state)
	if len(available) != 0 {
		t.Errorf("Should see 0 quests with skill 5, got %d", len(available))
	}

	// Skill high enough
	state.CraftingSkills["blacksmithing"] = 10
	available = registry.GetAvailableQuestsForPlayer("smith", state)
	if len(available) != 1 {
		t.Errorf("Should see 1 quest with skill 10, got %d", len(available))
	}
}

func TestGetAvailableQuestsForPlayer_ExcludesCompletedQuests(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"one_time_quest": {Name: "One Time", GiverNPC: "npc", Repeatable: false},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    make(map[string]bool),
		ClassLevels:     make(map[string]int),
		CraftingSkills:  make(map[string]int),
	}

	// Not completed yet
	available := registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 1 {
		t.Errorf("Should see quest before completion, got %d", len(available))
	}

	// Mark as completed
	state.CompletedQuests["one_time_quest"] = true
	available = registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 0 {
		t.Errorf("Should not see completed non-repeatable quest, got %d", len(available))
	}
}

func TestGetAvailableQuestsForPlayer_AllowsRepeatableQuests(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"repeatable_quest": {Name: "Repeatable", GiverNPC: "npc", Repeatable: true},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		CompletedQuests: map[string]bool{"repeatable_quest": true},
		ActiveQuests:    make(map[string]bool),
		ClassLevels:     make(map[string]int),
		CraftingSkills:  make(map[string]int),
	}

	// Should still be available even after completion
	available := registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 1 {
		t.Errorf("Repeatable quest should still be available, got %d", len(available))
	}
}

func TestGetAvailableQuestsForPlayer_ExcludesActiveQuests(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {Name: "Quest 1", GiverNPC: "npc"},
		},
	}

	registry.LoadFromConfig(config)

	state := &PlayerQuestState{
		Level:           10,
		CompletedQuests: make(map[string]bool),
		ActiveQuests:    map[string]bool{"quest1": true},
		ClassLevels:     make(map[string]int),
		CraftingSkills:  make(map[string]int),
	}

	available := registry.GetAvailableQuestsForPlayer("npc", state)
	if len(available) != 0 {
		t.Errorf("Should not see quest already in progress, got %d", len(available))
	}
}

func TestRegistryGetAllQuests(t *testing.T) {
	registry := NewQuestRegistry()
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {Name: "Quest 1"},
			"quest2": {Name: "Quest 2"},
			"quest3": {Name: "Quest 3"},
		},
	}

	registry.LoadFromConfig(config)

	quests := registry.GetAllQuests()
	if len(quests) != 3 {
		t.Errorf("Should have 3 quests, got %d", len(quests))
	}
}

func TestGetQuestCount(t *testing.T) {
	registry := NewQuestRegistry()

	if registry.GetQuestCount() != 0 {
		t.Error("Empty registry should have 0 quests")
	}

	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {Name: "Quest 1"},
			"quest2": {Name: "Quest 2"},
		},
	}

	registry.LoadFromConfig(config)

	if registry.GetQuestCount() != 2 {
		t.Errorf("Should have 2 quests, got %d", registry.GetQuestCount())
	}
}

func TestLoadFromConfig_ClearsExistingData(t *testing.T) {
	registry := NewQuestRegistry()

	// First load
	config1 := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"old_quest": {Name: "Old Quest", GiverNPC: "old_npc"},
		},
	}
	registry.LoadFromConfig(config1)

	// Second load should replace
	config2 := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"new_quest": {Name: "New Quest", GiverNPC: "new_npc"},
		},
	}
	registry.LoadFromConfig(config2)

	if registry.GetQuestCount() != 1 {
		t.Errorf("Should have 1 quest after reload, got %d", registry.GetQuestCount())
	}

	_, exists := registry.GetQuest("old_quest")
	if exists {
		t.Error("Old quest should not exist after reload")
	}

	_, exists = registry.GetQuest("new_quest")
	if !exists {
		t.Error("New quest should exist after reload")
	}
}
