package quest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadQuestsFromYAML_ValidFile(t *testing.T) {
	// Create temp directory and file
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  test_quest:
    name: "Test Quest"
    description: "A test quest description"
    category: "side"
    giver_npc: "test_npc"
    turn_in_npc: "test_npc"
    objectives:
      - type: "kill"
        target: "rat"
        target_name: "Rat"
        required: 5
    rewards:
      gold: 100
      experience: 50
      items:
        - "sword"
    min_level: 1
    prereqs: []
    repeatable: false
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	if config == nil {
		t.Fatal("Config should not be nil")
	}

	if len(config.Quests) != 1 {
		t.Errorf("Should have 1 quest, got %d", len(config.Quests))
	}

	def, exists := config.Quests["test_quest"]
	if !exists {
		t.Fatal("test_quest should exist in config")
	}

	if def.Name != "Test Quest" {
		t.Errorf("Quest name mismatch: got %s, want Test Quest", def.Name)
	}
}

func TestLoadQuestsFromYAML_MissingFile(t *testing.T) {
	_, err := LoadQuestsFromYAML("/nonexistent/path/quests.yaml")
	if err == nil {
		t.Error("Should return error for missing file")
	}
}

func TestLoadQuestsFromYAML_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	// Invalid YAML
	yamlContent := `quests:
  test_quest:
    name: "Test Quest"
    objectives: [invalid yaml structure
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadQuestsFromYAML(questFile)
	if err == nil {
		t.Error("Should return error for malformed YAML")
	}
}

func TestGetQuestByID(t *testing.T) {
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {
				Name:        "Quest One",
				Description: "First quest",
				Category:    "main",
				GiverNPC:    "npc1",
				TurnInNPC:   "npc1",
				Objectives: []QuestObjectiveYAML{
					{Type: "kill", Target: "rat", TargetName: "Rat", Required: 3},
				},
				Rewards: QuestRewardYAML{Gold: 50},
			},
		},
	}

	quest, exists := config.GetQuestByID("quest1")
	if !exists {
		t.Fatal("Quest should exist")
	}

	if quest.ID != "quest1" {
		t.Errorf("Quest ID mismatch: got %s, want quest1", quest.ID)
	}
	if quest.Name != "Quest One" {
		t.Errorf("Quest Name mismatch: got %s, want Quest One", quest.Name)
	}

	_, exists = config.GetQuestByID("nonexistent")
	if exists {
		t.Error("Nonexistent quest should not exist")
	}
}

func TestGetAllQuests(t *testing.T) {
	config := &QuestsConfig{
		Quests: map[string]QuestDefinition{
			"quest1": {Name: "Quest One"},
			"quest2": {Name: "Quest Two"},
			"quest3": {Name: "Quest Three"},
		},
	}

	quests := config.GetAllQuests()
	if len(quests) != 3 {
		t.Errorf("Should have 3 quests, got %d", len(quests))
	}
}

func TestQuestFieldParsing_AllObjectiveTypes(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  kill_quest:
    name: "Kill Quest"
    objectives:
      - type: "kill"
        target: "mob"
        required: 5
  fetch_quest:
    name: "Fetch Quest"
    objectives:
      - type: "fetch"
        target: "item"
        required: 3
  delivery_quest:
    name: "Delivery Quest"
    objectives:
      - type: "delivery"
        target: "letter"
        required: 1
  explore_quest:
    name: "Explore Quest"
    objectives:
      - type: "explore"
        target: "room"
        required: 1
  craft_quest:
    name: "Craft Quest"
    objectives:
      - type: "craft"
        target: "sword"
        required: 2
  cast_quest:
    name: "Cast Quest"
    objectives:
      - type: "cast"
        target: "fireball"
        required: 10
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	testCases := []struct {
		questID      string
		expectedType QuestType
	}{
		{"kill_quest", QuestTypeKill},
		{"fetch_quest", QuestTypeFetch},
		{"delivery_quest", QuestTypeDelivery},
		{"explore_quest", QuestTypeExplore},
		{"craft_quest", QuestTypeCraft},
		{"cast_quest", QuestTypeCast},
	}

	for _, tc := range testCases {
		quest, exists := config.GetQuestByID(tc.questID)
		if !exists {
			t.Errorf("Quest %s should exist", tc.questID)
			continue
		}
		if len(quest.Objectives) == 0 {
			t.Errorf("Quest %s should have objectives", tc.questID)
			continue
		}
		if quest.Objectives[0].Type != tc.expectedType {
			t.Errorf("Quest %s objective type mismatch: got %s, want %s",
				tc.questID, quest.Objectives[0].Type, tc.expectedType)
		}
	}
}

func TestQuestFieldParsing_Rewards(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  reward_quest:
    name: "Reward Quest"
    objectives:
      - type: "kill"
        required: 1
    rewards:
      gold: 100
      experience: 200
      items:
        - "sword"
        - "shield"
      recipes:
        - "steel_sword"
      title: "Champion"
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	quest, _ := config.GetQuestByID("reward_quest")
	if quest.Rewards.Gold != 100 {
		t.Errorf("Gold mismatch: got %d, want 100", quest.Rewards.Gold)
	}
	if quest.Rewards.Experience != 200 {
		t.Errorf("Experience mismatch: got %d, want 200", quest.Rewards.Experience)
	}
	if len(quest.Rewards.Items) != 2 {
		t.Errorf("Items count mismatch: got %d, want 2", len(quest.Rewards.Items))
	}
	if len(quest.Rewards.Recipes) != 1 {
		t.Errorf("Recipes count mismatch: got %d, want 1", len(quest.Rewards.Recipes))
	}
	if quest.Rewards.Title != "Champion" {
		t.Errorf("Title mismatch: got %s, want Champion", quest.Rewards.Title)
	}
}

func TestQuestFieldParsing_Prerequisites(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  chain_quest:
    name: "Chain Quest"
    objectives:
      - type: "kill"
        required: 1
    prereqs:
      - "first_quest"
      - "second_quest"
    min_level: 10
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	quest, _ := config.GetQuestByID("chain_quest")
	if len(quest.Prereqs) != 2 {
		t.Errorf("Prereqs count mismatch: got %d, want 2", len(quest.Prereqs))
	}
	if quest.MinLevel != 10 {
		t.Errorf("MinLevel mismatch: got %d, want 10", quest.MinLevel)
	}
}

func TestQuestFieldParsing_ClassQuest(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  warrior_level_05:
    name: "Warrior Quest"
    category: "class"
    giver_npc: "trainer"
    turn_in_npc: "trainer"
    required_class: "warrior"
    required_class_level: 5
    objectives:
      - type: "kill"
        required: 10
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	quest, _ := config.GetQuestByID("warrior_level_05")
	if quest.Category != QuestCategoryClass {
		t.Errorf("Category mismatch: got %s, want class", quest.Category)
	}
	if quest.RequiredClass != "warrior" {
		t.Errorf("RequiredClass mismatch: got %s, want warrior", quest.RequiredClass)
	}
	if quest.RequiredClassLevel != 5 {
		t.Errorf("RequiredClassLevel mismatch: got %d, want 5", quest.RequiredClassLevel)
	}
	if !quest.IsClassQuest() {
		t.Error("IsClassQuest should return true")
	}
}

func TestQuestFieldParsing_CraftingQuest(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  blacksmithing_level_10:
    name: "Smithing Quest"
    category: "crafting"
    giver_npc: "smith"
    turn_in_npc: "smith"
    required_crafting_skill: "blacksmithing"
    required_crafting_level: 10
    objectives:
      - type: "craft"
        target: "sword"
        required: 3
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	quest, _ := config.GetQuestByID("blacksmithing_level_10")
	if quest.Category != QuestCategoryCrafting {
		t.Errorf("Category mismatch: got %s, want crafting", quest.Category)
	}
	if quest.RequiredCraftingSkill != "blacksmithing" {
		t.Errorf("RequiredCraftingSkill mismatch: got %s, want blacksmithing", quest.RequiredCraftingSkill)
	}
	if quest.RequiredCraftingLevel != 10 {
		t.Errorf("RequiredCraftingLevel mismatch: got %d, want 10", quest.RequiredCraftingLevel)
	}
	if !quest.IsCraftingQuest() {
		t.Error("IsCraftingQuest should return true")
	}
}

func TestQuestFieldParsing_QuestItems(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  delivery_quest:
    name: "Delivery Quest"
    objectives:
      - type: "delivery"
        target: "letter"
        required: 1
    quest_items:
      - "letter"
      - "package"
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	quest, _ := config.GetQuestByID("delivery_quest")
	if len(quest.QuestItems) != 2 {
		t.Errorf("QuestItems count mismatch: got %d, want 2", len(quest.QuestItems))
	}
	if !quest.HasQuestItems() {
		t.Error("HasQuestItems should return true")
	}
}

func TestQuestFieldParsing_Repeatable(t *testing.T) {
	tmpDir := t.TempDir()
	questFile := filepath.Join(tmpDir, "quests.yaml")

	yamlContent := `quests:
  repeatable_quest:
    name: "Repeatable Quest"
    objectives:
      - type: "kill"
        required: 1
    repeatable: true
  non_repeatable_quest:
    name: "Non-Repeatable Quest"
    objectives:
      - type: "kill"
        required: 1
    repeatable: false
`

	if err := os.WriteFile(questFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadQuestsFromYAML(questFile)
	if err != nil {
		t.Fatalf("LoadQuestsFromYAML returned error: %v", err)
	}

	repeatable, _ := config.GetQuestByID("repeatable_quest")
	if !repeatable.Repeatable {
		t.Error("repeatable_quest should be repeatable")
	}

	nonRepeatable, _ := config.GetQuestByID("non_repeatable_quest")
	if nonRepeatable.Repeatable {
		t.Error("non_repeatable_quest should not be repeatable")
	}
}

func TestParseQuestType(t *testing.T) {
	tests := []struct {
		input    string
		expected QuestType
	}{
		{"kill", QuestTypeKill},
		{"fetch", QuestTypeFetch},
		{"delivery", QuestTypeDelivery},
		{"explore", QuestTypeExplore},
		{"craft", QuestTypeCraft},
		{"cast", QuestTypeCast},
		{"unknown", QuestTypeKill}, // Default fallback
		{"", QuestTypeKill},        // Empty string fallback
	}

	for _, tt := range tests {
		result := parseQuestType(tt.input)
		if result != tt.expected {
			t.Errorf("parseQuestType(%s): got %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseQuestCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected QuestCategory
	}{
		{"main", QuestCategoryMain},
		{"side", QuestCategorySide},
		{"class", QuestCategoryClass},
		{"crafting", QuestCategoryCrafting},
		{"unknown", QuestCategorySide}, // Default fallback
		{"", QuestCategorySide},        // Empty string fallback
	}

	for _, tt := range tests {
		result := parseQuestCategory(tt.input)
		if result != tt.expected {
			t.Errorf("parseQuestCategory(%s): got %s, want %s", tt.input, result, tt.expected)
		}
	}
}
