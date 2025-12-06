package quest

import (
	"encoding/json"
	"testing"
)

func TestNewPlayerQuestLog(t *testing.T) {
	log := NewPlayerQuestLog()

	if log == nil {
		t.Fatal("NewPlayerQuestLog returned nil")
	}
	if log.Active == nil {
		t.Error("Active map is nil")
	}
	if log.Completed == nil {
		t.Error("Completed map is nil")
	}
	if len(log.Active) != 0 {
		t.Errorf("Active map should be empty, got %d items", len(log.Active))
	}
	if len(log.Completed) != 0 {
		t.Errorf("Completed map should be empty, got %d items", len(log.Completed))
	}
}

func TestStartQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID:   "test_quest",
		Name: "Test Quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 5},
			{Type: QuestTypeFetch, Target: "herb", Required: 3},
		},
	}

	err := log.StartQuest(quest)
	if err != nil {
		t.Fatalf("StartQuest returned error: %v", err)
	}

	if !log.HasActiveQuest("test_quest") {
		t.Error("Quest should be active after StartQuest")
	}

	progress, exists := log.GetQuestProgress("test_quest")
	if !exists {
		t.Fatal("GetQuestProgress should return quest")
	}

	if progress.Status != QuestStatusActive {
		t.Errorf("Quest status should be active, got %s", progress.Status)
	}

	if len(progress.Objectives) != 2 {
		t.Errorf("Should have 2 objectives, got %d", len(progress.Objectives))
	}

	for i, obj := range progress.Objectives {
		if obj.Current != 0 {
			t.Errorf("Objective %d should start at 0, got %d", i, obj.Current)
		}
	}
}

func TestUpdateKillProgressForQuest_SpecificTarget(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "kill_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 3},
		},
	}

	log.StartQuest(quest)

	// Kill target mob
	updated := log.UpdateKillProgressForQuest("kill_quest", quest, "rat")
	if !updated {
		t.Error("Should update progress when killing target mob")
	}

	progress, _ := log.GetQuestProgress("kill_quest")
	if progress.Objectives[0].Current != 1 {
		t.Errorf("Kill count should be 1, got %d", progress.Objectives[0].Current)
	}

	// Kill non-target mob
	updated = log.UpdateKillProgressForQuest("kill_quest", quest, "wolf")
	if updated {
		t.Error("Should not update progress when killing non-target mob")
	}

	if progress.Objectives[0].Current != 1 {
		t.Errorf("Kill count should still be 1, got %d", progress.Objectives[0].Current)
	}
}

func TestUpdateKillProgressForQuest_AnyTarget(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "kill_any_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "", Required: 3}, // Empty target = any mob
		},
	}

	log.StartQuest(quest)

	// Kill any mob
	log.UpdateKillProgressForQuest("kill_any_quest", quest, "rat")
	log.UpdateKillProgressForQuest("kill_any_quest", quest, "wolf")
	log.UpdateKillProgressForQuest("kill_any_quest", quest, "goblin")

	progress, _ := log.GetQuestProgress("kill_any_quest")
	if progress.Objectives[0].Current != 3 {
		t.Errorf("Kill count should be 3, got %d", progress.Objectives[0].Current)
	}

	// Quest should be completed
	if progress.Status != QuestStatusCompleted {
		t.Errorf("Quest should be completed, got %s", progress.Status)
	}
}

func TestUpdateItemProgressForQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "fetch_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeFetch, Target: "herb", Required: 2},
		},
	}

	log.StartQuest(quest)

	// Pick up target item
	updated := log.UpdateItemProgressForQuest("fetch_quest", quest, "herb")
	if !updated {
		t.Error("Should update progress when picking up target item")
	}

	progress, _ := log.GetQuestProgress("fetch_quest")
	if progress.Objectives[0].Current != 1 {
		t.Errorf("Item count should be 1, got %d", progress.Objectives[0].Current)
	}

	// Pick up non-target item
	updated = log.UpdateItemProgressForQuest("fetch_quest", quest, "potion")
	if updated {
		t.Error("Should not update progress when picking up non-target item")
	}
}

func TestUpdateExploreProgressForQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "explore_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeExplore, Target: "secret_room", Required: 1},
		},
	}

	log.StartQuest(quest)

	// Visit target room
	updated := log.UpdateExploreProgressForQuest("explore_quest", quest, "secret_room")
	if !updated {
		t.Error("Should update progress when visiting target room")
	}

	progress, _ := log.GetQuestProgress("explore_quest")
	if progress.Objectives[0].Current != 1 {
		t.Errorf("Visit count should be 1, got %d", progress.Objectives[0].Current)
	}

	// Quest should be completed
	if progress.Status != QuestStatusCompleted {
		t.Errorf("Quest should be completed, got %s", progress.Status)
	}

	// Visit non-target room
	updated = log.UpdateExploreProgressForQuest("explore_quest", quest, "other_room")
	if updated {
		t.Error("Should not update progress when visiting non-target room")
	}
}

func TestUpdateCraftProgressForQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "craft_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeCraft, Target: "sword", Required: 2},
		},
	}

	log.StartQuest(quest)

	// Craft target item
	updated := log.UpdateCraftProgressForQuest("craft_quest", quest, "sword")
	if !updated {
		t.Error("Should update progress when crafting target item")
	}

	progress, _ := log.GetQuestProgress("craft_quest")
	if progress.Objectives[0].Current != 1 {
		t.Errorf("Craft count should be 1, got %d", progress.Objectives[0].Current)
	}

	// Craft non-target item
	updated = log.UpdateCraftProgressForQuest("craft_quest", quest, "shield")
	if updated {
		t.Error("Should not update progress when crafting non-target item")
	}
}

func TestUpdateCastProgressForQuest_SpecificSpell(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "cast_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeCast, Target: "fireball", Required: 3},
		},
	}

	log.StartQuest(quest)

	// Cast target spell
	updated := log.UpdateCastProgressForQuest("cast_quest", quest, "fireball")
	if !updated {
		t.Error("Should update progress when casting target spell")
	}

	progress, _ := log.GetQuestProgress("cast_quest")
	if progress.Objectives[0].Current != 1 {
		t.Errorf("Cast count should be 1, got %d", progress.Objectives[0].Current)
	}

	// Cast non-target spell
	updated = log.UpdateCastProgressForQuest("cast_quest", quest, "heal")
	if updated {
		t.Error("Should not update progress when casting non-target spell")
	}
}

func TestUpdateCastProgressForQuest_AnySpell(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "cast_any_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeCast, Target: "", Required: 3}, // Empty = any spell
		},
	}

	log.StartQuest(quest)

	log.UpdateCastProgressForQuest("cast_any_quest", quest, "fireball")
	log.UpdateCastProgressForQuest("cast_any_quest", quest, "heal")
	log.UpdateCastProgressForQuest("cast_any_quest", quest, "lightning")

	progress, _ := log.GetQuestProgress("cast_any_quest")
	if progress.Objectives[0].Current != 3 {
		t.Errorf("Cast count should be 3, got %d", progress.Objectives[0].Current)
	}

	if progress.Status != QuestStatusCompleted {
		t.Errorf("Quest should be completed, got %s", progress.Status)
	}
}

func TestCanCompleteQuest_Incomplete(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "incomplete_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 5},
		},
	}

	log.StartQuest(quest)
	log.UpdateKillProgressForQuest("incomplete_quest", quest, "rat")
	log.UpdateKillProgressForQuest("incomplete_quest", quest, "rat")

	if log.CanCompleteQuest("incomplete_quest", quest) {
		t.Error("Should not be able to complete quest with incomplete objectives")
	}
}

func TestCanCompleteQuest_Complete(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "complete_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 2},
		},
	}

	log.StartQuest(quest)
	log.UpdateKillProgressForQuest("complete_quest", quest, "rat")
	log.UpdateKillProgressForQuest("complete_quest", quest, "rat")

	if !log.CanCompleteQuest("complete_quest", quest) {
		t.Error("Should be able to complete quest with all objectives met")
	}
}

func TestTurnInQuest_NonRepeatable(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID:         "non_repeatable",
		Repeatable: false,
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 1},
		},
	}

	log.StartQuest(quest)
	log.UpdateKillProgressForQuest("non_repeatable", quest, "rat")

	err := log.TurnInQuest("non_repeatable", quest.Repeatable)
	if err != nil {
		t.Fatalf("TurnInQuest returned error: %v", err)
	}

	if log.HasActiveQuest("non_repeatable") {
		t.Error("Quest should no longer be active after turn-in")
	}

	if !log.HasCompletedQuest("non_repeatable") {
		t.Error("Quest should be in completed list")
	}
}

func TestTurnInQuest_Repeatable(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID:         "repeatable",
		Repeatable: true,
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 1},
		},
	}

	log.StartQuest(quest)
	log.UpdateKillProgressForQuest("repeatable", quest, "rat")

	err := log.TurnInQuest("repeatable", quest.Repeatable)
	if err != nil {
		t.Fatalf("TurnInQuest returned error: %v", err)
	}

	if log.HasActiveQuest("repeatable") {
		t.Error("Quest should no longer be active after turn-in")
	}

	if log.HasCompletedQuest("repeatable") {
		t.Error("Repeatable quest should NOT be in completed list")
	}
}

func TestJSONSerialization(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "test_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 5},
		},
	}

	log.StartQuest(quest)
	log.UpdateKillProgressForQuest("test_quest", quest, "rat")
	log.UpdateKillProgressForQuest("test_quest", quest, "rat")
	log.UpdateKillProgressForQuest("test_quest", quest, "rat")

	// Add a completed quest
	log.Completed["old_quest"] = true

	jsonStr := log.ToJSON()
	if jsonStr == "" || jsonStr == "{}" {
		t.Error("ToJSON should return non-empty JSON")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v", err)
	}

	// Deserialize
	restored, err := PlayerQuestLogFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("PlayerQuestLogFromJSON returned error: %v", err)
	}

	// Verify active quest
	if !restored.HasActiveQuest("test_quest") {
		t.Error("Restored log should have active quest")
	}

	progress, exists := restored.GetQuestProgress("test_quest")
	if !exists {
		t.Fatal("Restored log should have quest progress")
	}

	if progress.Objectives[0].Current != 3 {
		t.Errorf("Restored kill count should be 3, got %d", progress.Objectives[0].Current)
	}

	// Verify completed quest
	if !restored.HasCompletedQuest("old_quest") {
		t.Error("Restored log should have completed quest")
	}
}

func TestJSONDeserialization_Empty(t *testing.T) {
	log, err := PlayerQuestLogFromJSON("")
	if err != nil {
		t.Fatalf("Empty string should not error: %v", err)
	}
	if log == nil {
		t.Fatal("Should return valid log for empty string")
	}
	if log.Active == nil || log.Completed == nil {
		t.Error("Maps should be initialized")
	}

	log2, err := PlayerQuestLogFromJSON("{}")
	if err != nil {
		t.Fatalf("Empty object should not error: %v", err)
	}
	if log2 == nil {
		t.Fatal("Should return valid log for empty object")
	}
}

func TestGetActiveQuests(t *testing.T) {
	log := NewPlayerQuestLog()

	quest1 := &Quest{ID: "quest1", Objectives: []QuestObjective{{Type: QuestTypeKill, Required: 1}}}
	quest2 := &Quest{ID: "quest2", Objectives: []QuestObjective{{Type: QuestTypeKill, Required: 1}}}

	log.StartQuest(quest1)
	log.StartQuest(quest2)

	active := log.GetActiveQuests()
	if len(active) != 2 {
		t.Errorf("Should have 2 active quests, got %d", len(active))
	}

	// Check both IDs are present
	found1, found2 := false, false
	for _, id := range active {
		if id == "quest1" {
			found1 = true
		}
		if id == "quest2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Both quest IDs should be in active list")
	}
}

func TestGetCompletedQuests(t *testing.T) {
	log := NewPlayerQuestLog()
	log.Completed["quest1"] = true
	log.Completed["quest2"] = true

	completed := log.GetCompletedQuests()
	if len(completed) != 2 {
		t.Errorf("Should have 2 completed quests, got %d", len(completed))
	}
}

func TestAbandonQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{ID: "abandon_test", Objectives: []QuestObjective{{Type: QuestTypeKill, Required: 5}}}

	log.StartQuest(quest)
	if !log.HasActiveQuest("abandon_test") {
		t.Fatal("Quest should be active")
	}

	err := log.AbandonQuest("abandon_test")
	if err != nil {
		t.Fatalf("AbandonQuest returned error: %v", err)
	}

	if log.HasActiveQuest("abandon_test") {
		t.Error("Quest should not be active after abandoning")
	}

	if log.HasCompletedQuest("abandon_test") {
		t.Error("Abandoned quest should not be in completed list")
	}
}

func TestProgressDoesNotExceedRequired(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "capped_quest",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 2},
		},
	}

	log.StartQuest(quest)

	// Kill more than required
	for i := 0; i < 10; i++ {
		log.UpdateKillProgressForQuest("capped_quest", quest, "rat")
	}

	progress, _ := log.GetQuestProgress("capped_quest")
	if progress.Objectives[0].Current != 2 {
		t.Errorf("Progress should cap at required amount (2), got %d", progress.Objectives[0].Current)
	}
}

func TestMultiObjectiveQuest(t *testing.T) {
	log := NewPlayerQuestLog()
	quest := &Quest{
		ID: "multi_obj",
		Objectives: []QuestObjective{
			{Type: QuestTypeKill, Target: "rat", Required: 2},
			{Type: QuestTypeFetch, Target: "herb", Required: 1},
		},
	}

	log.StartQuest(quest)

	// Complete first objective
	log.UpdateKillProgressForQuest("multi_obj", quest, "rat")
	log.UpdateKillProgressForQuest("multi_obj", quest, "rat")

	progress, _ := log.GetQuestProgress("multi_obj")
	if progress.Status == QuestStatusCompleted {
		t.Error("Quest should not be completed with only first objective done")
	}

	// Complete second objective
	log.UpdateItemProgressForQuest("multi_obj", quest, "herb")

	progress, _ = log.GetQuestProgress("multi_obj")
	if progress.Status != QuestStatusCompleted {
		t.Error("Quest should be completed with all objectives done")
	}
}
