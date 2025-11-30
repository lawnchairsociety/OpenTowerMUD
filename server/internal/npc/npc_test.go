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
