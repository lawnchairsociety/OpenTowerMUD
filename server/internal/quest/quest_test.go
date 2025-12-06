package quest

import (
	"testing"
)

func TestQuestTypeConstants(t *testing.T) {
	tests := []struct {
		questType QuestType
		expected  string
	}{
		{QuestTypeKill, "kill"},
		{QuestTypeFetch, "fetch"},
		{QuestTypeDelivery, "delivery"},
		{QuestTypeExplore, "explore"},
		{QuestTypeCraft, "craft"},
		{QuestTypeCast, "cast"},
	}

	for _, tt := range tests {
		if string(tt.questType) != tt.expected {
			t.Errorf("QuestType constant mismatch: got %s, want %s", tt.questType, tt.expected)
		}
	}
}

func TestQuestCategoryConstants(t *testing.T) {
	tests := []struct {
		category QuestCategory
		expected string
	}{
		{QuestCategoryMain, "main"},
		{QuestCategorySide, "side"},
		{QuestCategoryClass, "class"},
		{QuestCategoryCrafting, "crafting"},
	}

	for _, tt := range tests {
		if string(tt.category) != tt.expected {
			t.Errorf("QuestCategory constant mismatch: got %s, want %s", tt.category, tt.expected)
		}
	}
}

func TestQuestCreation(t *testing.T) {
	quest := &Quest{
		ID:          "test_quest",
		Name:        "Test Quest",
		Description: "A test quest",
		Category:    QuestCategorySide,
		GiverNPC:    "test_npc",
		TurnInNPC:   "test_npc",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", TargetName: "Rat", Required: 5},
		},
		Rewards: QuestReward{
			Gold:       100,
			Experience: 50,
			Items:      []string{"sword"},
		},
		MinLevel:   1,
		Prereqs:    []string{},
		Repeatable: false,
	}

	if quest.ID != "test_quest" {
		t.Errorf("Quest ID mismatch: got %s, want test_quest", quest.ID)
	}
	if quest.Name != "Test Quest" {
		t.Errorf("Quest Name mismatch: got %s, want Test Quest", quest.Name)
	}
	if len(quest.Objectives) != 1 {
		t.Errorf("Quest Objectives length mismatch: got %d, want 1", len(quest.Objectives))
	}
}

func TestQuestIsClassQuest(t *testing.T) {
	classQuest := &Quest{
		ID:                 "warrior_level_05",
		Category:           QuestCategoryClass,
		RequiredClass:      "warrior",
		RequiredClassLevel: 5,
	}

	if !classQuest.IsClassQuest() {
		t.Error("Expected IsClassQuest to return true for class quest")
	}

	sideQuest := &Quest{
		ID:       "side_quest",
		Category: QuestCategorySide,
	}

	if sideQuest.IsClassQuest() {
		t.Error("Expected IsClassQuest to return false for side quest")
	}

	// Class category but no required class
	incompleteClassQuest := &Quest{
		ID:       "incomplete",
		Category: QuestCategoryClass,
	}

	if incompleteClassQuest.IsClassQuest() {
		t.Error("Expected IsClassQuest to return false when RequiredClass is empty")
	}
}

func TestQuestIsCraftingQuest(t *testing.T) {
	craftingQuest := &Quest{
		ID:                    "blacksmithing_level_10",
		Category:              QuestCategoryCrafting,
		RequiredCraftingSkill: "blacksmithing",
		RequiredCraftingLevel: 10,
	}

	if !craftingQuest.IsCraftingQuest() {
		t.Error("Expected IsCraftingQuest to return true for crafting quest")
	}

	sideQuest := &Quest{
		ID:       "side_quest",
		Category: QuestCategorySide,
	}

	if sideQuest.IsCraftingQuest() {
		t.Error("Expected IsCraftingQuest to return false for side quest")
	}
}

func TestQuestHasQuestItems(t *testing.T) {
	deliveryQuest := &Quest{
		ID:         "delivery_quest",
		QuestItems: []string{"letter"},
	}

	if !deliveryQuest.HasQuestItems() {
		t.Error("Expected HasQuestItems to return true when QuestItems is not empty")
	}

	normalQuest := &Quest{
		ID:         "normal_quest",
		QuestItems: []string{},
	}

	if normalQuest.HasQuestItems() {
		t.Error("Expected HasQuestItems to return false when QuestItems is empty")
	}

	nilQuest := &Quest{
		ID: "nil_quest",
	}

	if nilQuest.HasQuestItems() {
		t.Error("Expected HasQuestItems to return false when QuestItems is nil")
	}
}

func TestQuestHasPrereqs(t *testing.T) {
	questWithPrereqs := &Quest{
		ID:      "quest_with_prereqs",
		Prereqs: []string{"prereq_quest_1", "prereq_quest_2"},
	}

	if !questWithPrereqs.HasPrereqs() {
		t.Error("Expected HasPrereqs to return true when Prereqs is not empty")
	}

	questWithoutPrereqs := &Quest{
		ID:      "quest_without_prereqs",
		Prereqs: []string{},
	}

	if questWithoutPrereqs.HasPrereqs() {
		t.Error("Expected HasPrereqs to return false when Prereqs is empty")
	}
}

func TestQuestGetObjectiveCount(t *testing.T) {
	quest := &Quest{
		ID: "multi_objective",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 5},
			{Type: QuestTypeFetch, Target: "herb", Required: 3},
			{Type: QuestTypeExplore, Target: "room_1", Required: 1},
		},
	}

	if quest.GetObjectiveCount() != 3 {
		t.Errorf("GetObjectiveCount mismatch: got %d, want 3", quest.GetObjectiveCount())
	}

	emptyQuest := &Quest{
		ID:         "empty_objectives",
		Objectives: []QuestObjective{},
	}

	if emptyQuest.GetObjectiveCount() != 0 {
		t.Errorf("GetObjectiveCount mismatch for empty: got %d, want 0", emptyQuest.GetObjectiveCount())
	}
}

func TestQuestObjective(t *testing.T) {
	objective := QuestObjective{
		Type:       QuestTypeKill,
		Target:     "giant_rat",
		TargetName: "Giant Rat",
		Required:   5,
	}

	if objective.Type != QuestTypeKill {
		t.Errorf("Objective Type mismatch: got %s, want kill", objective.Type)
	}
	if objective.Target != "giant_rat" {
		t.Errorf("Objective Target mismatch: got %s, want giant_rat", objective.Target)
	}
	if objective.TargetName != "Giant Rat" {
		t.Errorf("Objective TargetName mismatch: got %s, want Giant Rat", objective.TargetName)
	}
	if objective.Required != 5 {
		t.Errorf("Objective Required mismatch: got %d, want 5", objective.Required)
	}
}

func TestQuestReward(t *testing.T) {
	reward := QuestReward{
		Gold:       100,
		Experience: 200,
		Items:      []string{"sword", "shield"},
		Recipes:    []string{"steel_sword"},
		Title:      "Champion",
	}

	if reward.Gold != 100 {
		t.Errorf("Reward Gold mismatch: got %d, want 100", reward.Gold)
	}
	if reward.Experience != 200 {
		t.Errorf("Reward Experience mismatch: got %d, want 200", reward.Experience)
	}
	if len(reward.Items) != 2 {
		t.Errorf("Reward Items length mismatch: got %d, want 2", len(reward.Items))
	}
	if len(reward.Recipes) != 1 {
		t.Errorf("Reward Recipes length mismatch: got %d, want 1", len(reward.Recipes))
	}
	if reward.Title != "Champion" {
		t.Errorf("Reward Title mismatch: got %s, want Champion", reward.Title)
	}
}
