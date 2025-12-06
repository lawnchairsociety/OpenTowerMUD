package npc

import (
	"testing"
	"time"
)

func TestCalculateRespawnTime(t *testing.T) {
	tests := []struct {
		name             string
		respawnMedian    int
		respawnVariation int
		expectRespawn    bool
	}{
		{
			name:             "Respawn enabled with variation",
			respawnMedian:    60,
			respawnVariation: 10,
			expectRespawn:    true,
		},
		{
			name:             "Respawn enabled without variation",
			respawnMedian:    120,
			respawnVariation: 0,
			expectRespawn:    true,
		},
		{
			name:             "Respawn disabled",
			respawnMedian:    0,
			respawnVariation: 0,
			expectRespawn:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			npc := NewNPC(
				"test goblin",
				"A test goblin",
				1,
				20,
				3,
				0,
				10,
				true,
				true,
				"test_room",
				tt.respawnMedian,
				tt.respawnVariation,
			)

			before := time.Now()
			respawnTime := npc.CalculateRespawnTime()
			after := time.Now()

			if tt.expectRespawn {
				// Check that respawn time is in the future
				if !respawnTime.After(before) {
					t.Errorf("Expected respawn time to be after death time, got %v", respawnTime)
				}

				// Check that respawn time is within expected range
				expectedMin := before.Add(time.Duration(tt.respawnMedian-tt.respawnVariation) * time.Second)
				expectedMax := after.Add(time.Duration(tt.respawnMedian+tt.respawnVariation) * time.Second)

				if respawnTime.Before(expectedMin) || respawnTime.After(expectedMax) {
					t.Errorf("Respawn time %v not within expected range [%v, %v]", respawnTime, expectedMin, expectedMax)
				}

				// Verify death time was set
				if npc.DeathTime.IsZero() {
					t.Error("Expected death time to be set")
				}
			} else {
				// Respawn disabled - should return zero time
				if !respawnTime.IsZero() {
					t.Errorf("Expected zero respawn time for disabled respawn, got %v", respawnTime)
				}
			}
		})
	}
}

func TestNPCReset(t *testing.T) {
	npc := NewNPC(
		"test orc",
		"A test orc",
		3,
		40,
		8,
		2,
		30,
		true,
		true,
		"test_room",
		180,
		30,
	)

	// Damage the NPC and put it in combat
	npc.TakeDamage(20)
	npc.StartCombat("player1")
	npc.CalculateRespawnTime()

	// Verify NPC is damaged and in combat
	if npc.Health == npc.MaxHealth {
		t.Error("Expected NPC to be damaged")
	}
	if !npc.InCombat {
		t.Error("Expected NPC to be in combat")
	}
	if len(npc.Targets) == 0 {
		t.Error("Expected NPC to have targets")
	}
	if npc.DeathTime.IsZero() {
		t.Error("Expected death time to be set")
	}

	// Reset the NPC
	npc.Reset()

	// Verify NPC is back to full health and not in combat
	if npc.Health != npc.MaxHealth {
		t.Errorf("Expected health to be %d, got %d", npc.MaxHealth, npc.Health)
	}
	if npc.InCombat {
		t.Error("Expected NPC to not be in combat")
	}
	if len(npc.Targets) != 0 {
		t.Errorf("Expected no targets, got %d", len(npc.Targets))
	}
	if !npc.DeathTime.IsZero() {
		t.Error("Expected death time to be cleared")
	}
	if !npc.RespawnTime.IsZero() {
		t.Error("Expected respawn time to be cleared")
	}
}

func TestGetRespawnFields(t *testing.T) {
	npc := NewNPC(
		"test bat",
		"A test bat",
		2,
		12,
		4,
		0,
		12,
		true,
		true,
		"cave_room",
		120,
		30,
	)

	if npc.GetRespawnMedian() != 120 {
		t.Errorf("Expected respawn median 120, got %d", npc.GetRespawnMedian())
	}

	if npc.GetRespawnVariation() != 30 {
		t.Errorf("Expected respawn variation 30, got %d", npc.GetRespawnVariation())
	}

	if npc.GetOriginalRoomID() != "cave_room" {
		t.Errorf("Expected original room 'cave_room', got '%s'", npc.GetOriginalRoomID())
	}
}

func TestNPCDialogue(t *testing.T) {
	npc := NewNPC(
		"test merchant",
		"A test merchant",
		5,
		50,
		0,
		0,
		0,
		false,
		false,
		"market_room",
		0,
		0,
	)

	// Test NPC with no dialogue
	if dialogue := npc.GetDialogue(); dialogue != "" {
		t.Errorf("Expected empty dialogue for NPC without dialogue, got '%s'", dialogue)
	}

	// Set dialogue
	dialogueLines := []string{
		"Welcome to my shop!",
		"What would you like to buy?",
		"Come again soon!",
	}
	npc.SetDialogue(dialogueLines)

	// Test that GetDialogue returns one of the lines
	dialogue := npc.GetDialogue()
	found := false
	for _, line := range dialogueLines {
		if dialogue == line {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetDialogue returned unexpected value: '%s'", dialogue)
	}
}

func TestNPCDialogueFromDefinition(t *testing.T) {
	def := NPCDefinition{
		Name:        "shopkeeper",
		Description: "A friendly shopkeeper",
		Level:       4,
		Health:      40,
		Dialogue: []string{
			"Hello there!",
			"How can I help you?",
		},
	}

	npc := CreateNPCFromDefinition(def, "shop_room")

	// Verify dialogue was copied
	dialogue := npc.GetDialogue()
	if dialogue != "Hello there!" && dialogue != "How can I help you?" {
		t.Errorf("Expected dialogue from definition, got '%s'", dialogue)
	}
}

func TestRollLoot(t *testing.T) {
	t.Run("Regular mob with loot table", func(t *testing.T) {
		npc := NewNPC(
			"test goblin",
			"A test goblin",
			1, 20, 3, 0, 10,
			true, true,
			"test_room",
			120, 30,
		)

		// Set a loot table with various chances
		npc.SetLootTable([]LootEntry{
			{ItemName: "copper coin", DropChance: 100.0}, // Always drops
			{ItemName: "rare gem", DropChance: 0.0},      // Never drops
		})

		// Run multiple times to test probability
		gotCopperCoin := false
		gotRareGem := false
		for i := 0; i < 100; i++ {
			loot := npc.RollLoot()
			for _, item := range loot {
				if item == "copper coin" {
					gotCopperCoin = true
				}
				if item == "rare gem" {
					gotRareGem = true
				}
			}
		}

		if !gotCopperCoin {
			t.Error("Expected copper coin to drop with 100% chance")
		}
		if gotRareGem {
			t.Error("Did not expect rare gem to drop with 0% chance")
		}
	})

	t.Run("Boss drops everything", func(t *testing.T) {
		npc := NewNPC(
			"test boss",
			"A test boss",
			10, 200, 25, 5, 500,
			true, true,
			"boss_room",
			900, 180,
		)
		npc.SetBoss(10)

		// Set loot table with low chances
		npc.SetLootTable([]LootEntry{
			{ItemName: "boss crown", DropChance: 1.0},     // Normally 1% chance
			{ItemName: "epic sword", DropChance: 5.0},     // Normally 5% chance
			{ItemName: "ancient key", DropChance: 10.0},   // Normally 10% chance
		})

		// Bosses should drop everything regardless of chance
		loot := npc.RollLoot()

		if len(loot) != 3 {
			t.Errorf("Expected boss to drop all 3 items, got %d", len(loot))
		}

		// Check all items are present
		hasItems := map[string]bool{
			"boss crown": false,
			"epic sword": false,
			"ancient key": false,
		}
		for _, item := range loot {
			hasItems[item] = true
		}

		for item, has := range hasItems {
			if !has {
				t.Errorf("Expected boss to drop %s", item)
			}
		}
	})

	t.Run("NPC with no loot table drops nothing", func(t *testing.T) {
		npc := NewNPC(
			"no loot mob",
			"A mob with no loot",
			2, 30, 5, 1, 20,
			true, true,
			"test_room",
			120, 30,
		)

		// Don't set LootTable - should drop nothing
		for i := 0; i < 10; i++ {
			loot := npc.RollLoot()
			if len(loot) > 0 {
				t.Error("Expected no loot from mob without loot table")
			}
		}
	})
}

func TestLootTableFromDefinition(t *testing.T) {
	def := NPCDefinition{
		Name:        "test mob",
		Description: "A test mob",
		Level:       3,
		Health:      30,
		LootTable: []LootEntryYAML{
			{Item: "gold coin", Chance: 50.0},
			{Item: "healing potion", Chance: 25.0},
		},
	}

	npc := CreateNPCFromDefinition(def, "test_room")

	// Verify loot table was set
	lootTable := npc.GetLootTable()
	if len(lootTable) != 2 {
		t.Errorf("Expected 2 loot entries, got %d", len(lootTable))
	}

	if lootTable[0].ItemName != "gold coin" || lootTable[0].DropChance != 50.0 {
		t.Errorf("First loot entry incorrect: %+v", lootTable[0])
	}

	if lootTable[1].ItemName != "healing potion" || lootTable[1].DropChance != 25.0 {
		t.Errorf("Second loot entry incorrect: %+v", lootTable[1])
	}
}

// ==================== Quest Giver Tests ====================

func TestNPCQuestGiver_Basic(t *testing.T) {
	npc := NewNPC(
		"quest_giver",
		"A quest giving NPC",
		5, 50, 0, 0, 0,
		false, false,
		"town_square",
		0, 0,
	)

	// Initially not a quest giver
	if npc.IsQuestGiver() {
		t.Error("New NPC should not be a quest giver initially")
	}

	// Set as quest giver with quests
	npc.SetGivesQuests([]string{"quest_001", "quest_002"})

	if !npc.IsQuestGiver() {
		t.Error("NPC should be a quest giver after setting quests")
	}

	quests := npc.GetGivesQuests()
	if len(quests) != 2 {
		t.Errorf("Expected 2 quests, got %d", len(quests))
	}
}

func TestNPCQuestGiver_CanGiveQuest(t *testing.T) {
	npc := NewNPC(
		"quest_giver",
		"A quest giving NPC",
		5, 50, 0, 0, 0,
		false, false,
		"town_square",
		0, 0,
	)

	npc.SetGivesQuests([]string{"pest_control", "herb_gathering"})

	if !npc.CanGiveQuest("pest_control") {
		t.Error("NPC should be able to give pest_control quest")
	}
	if !npc.CanGiveQuest("herb_gathering") {
		t.Error("NPC should be able to give herb_gathering quest")
	}
	if npc.CanGiveQuest("unknown_quest") {
		t.Error("NPC should not be able to give unknown_quest")
	}
}

func TestNPCQuestGiver_TurnInQuests(t *testing.T) {
	npc := NewNPC(
		"quest_receiver",
		"An NPC that accepts quest turn-ins",
		5, 50, 0, 0, 0,
		false, false,
		"town_hall",
		0, 0,
	)

	npc.SetTurnInQuests([]string{"delivery_quest", "fetch_quest"})

	turnInQuests := npc.GetTurnInQuests()
	if len(turnInQuests) != 2 {
		t.Errorf("Expected 2 turn-in quests, got %d", len(turnInQuests))
	}

	if !npc.CanTurnInQuest("delivery_quest") {
		t.Error("NPC should accept delivery_quest turn-in")
	}
	if !npc.CanTurnInQuest("fetch_quest") {
		t.Error("NPC should accept fetch_quest turn-in")
	}
	if npc.CanTurnInQuest("unknown_quest") {
		t.Error("NPC should not accept unknown_quest turn-in")
	}
}

func TestNPCQuestGiver_HasQuestInteraction(t *testing.T) {
	npc := NewNPC(
		"test_npc",
		"A test NPC",
		5, 50, 0, 0, 0,
		false, false,
		"test_room",
		0, 0,
	)

	// No quest interaction initially
	if npc.HasQuestInteraction() {
		t.Error("NPC without quests should not have quest interaction")
	}

	// Set gives quests
	npc.SetGivesQuests([]string{"quest_1"})
	if !npc.HasQuestInteraction() {
		t.Error("NPC with gives_quests should have quest interaction")
	}

	// Reset and set turn-in only
	npc2 := NewNPC(
		"test_npc2",
		"Another test NPC",
		5, 50, 0, 0, 0,
		false, false,
		"test_room",
		0, 0,
	)
	npc2.SetTurnInQuests([]string{"quest_2"})
	if !npc2.HasQuestInteraction() {
		t.Error("NPC with turn_in_quests should have quest interaction")
	}
}

func TestNPCQuestGiverFromDefinition(t *testing.T) {
	def := NPCDefinition{
		Name:         "Guard Captain",
		Description:  "The captain of the city guard",
		Level:        10,
		Health:       100,
		QuestGiver:   true,
		GivesQuests:  []string{"pest_control", "bandit_hunt"},
		TurnInQuests: []string{"pest_control", "bandit_hunt"},
	}

	npc := CreateNPCFromDefinition(def, "barracks")

	if !npc.IsQuestGiver() {
		t.Error("NPC should be a quest giver from definition")
	}

	quests := npc.GetGivesQuests()
	if len(quests) != 2 {
		t.Errorf("Expected 2 quests, got %d", len(quests))
	}

	if !npc.CanGiveQuest("pest_control") {
		t.Error("NPC should be able to give pest_control quest")
	}
	if !npc.CanGiveQuest("bandit_hunt") {
		t.Error("NPC should be able to give bandit_hunt quest")
	}

	if !npc.CanTurnInQuest("pest_control") {
		t.Error("NPC should accept pest_control turn-in")
	}
}

func TestNPCQuestGiver_SetQuestGiverFlag(t *testing.T) {
	npc := NewNPC(
		"test_npc",
		"A test NPC",
		5, 50, 0, 0, 0,
		false, false,
		"test_room",
		0, 0,
	)

	// Set flag manually (without quests - should still not be "quest giver")
	npc.SetQuestGiver(true)

	// IsQuestGiver requires both flag AND quests
	if npc.IsQuestGiver() {
		t.Error("NPC should not be quest giver without quests even with flag set")
	}

	// Now add quests
	npc.SetGivesQuests([]string{"quest_1"})
	if !npc.IsQuestGiver() {
		t.Error("NPC should be quest giver with flag and quests")
	}
}

func TestNPCQuestGiver_GetQuestsReturnsCopy(t *testing.T) {
	npc := NewNPC(
		"test_npc",
		"A test NPC",
		5, 50, 0, 0, 0,
		false, false,
		"test_room",
		0, 0,
	)

	npc.SetGivesQuests([]string{"quest_1", "quest_2"})

	// Get quests and modify the returned slice
	quests := npc.GetGivesQuests()
	quests[0] = "modified"

	// Original should be unchanged
	originalQuests := npc.GetGivesQuests()
	if originalQuests[0] == "modified" {
		t.Error("GetGivesQuests should return a copy, not the original slice")
	}
}

func TestNPCQuestGiver_EmptyQuests(t *testing.T) {
	npc := NewNPC(
		"test_npc",
		"A test NPC",
		5, 50, 0, 0, 0,
		false, false,
		"test_room",
		0, 0,
	)

	// Set empty quest list
	npc.SetGivesQuests([]string{})
	npc.SetQuestGiver(true)

	// Should not be quest giver with empty quests
	if npc.IsQuestGiver() {
		t.Error("NPC should not be quest giver with empty quests list")
	}

	if npc.CanGiveQuest("any") {
		t.Error("NPC with no quests should not be able to give any quest")
	}
}
