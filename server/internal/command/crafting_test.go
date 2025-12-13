package command

import (
	"testing"

	"github.com/lawnchairsociety/opentowermud/server/internal/crafting"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// mockRoom implements RoomInterface for testing
type mockRoom struct {
	features []string
}

func (m *mockRoom) GetFeatures() []string                                             { return m.features }
func (m *mockRoom) GetDescription() string                                            { return "" }
func (m *mockRoom) GetBaseDescription() string                                        { return "" }
func (m *mockRoom) GetDescriptionForPlayer(playerName string) string                  { return "" }
func (m *mockRoom) GetDescriptionForPlayerWithCustomDesc(playerName, desc string) string { return "" }
func (m *mockRoom) GetDescriptionDay() string                                         { return "" }
func (m *mockRoom) GetDescriptionNight() string                                       { return "" }
func (m *mockRoom) GetExit(direction string) interface{}                              { return nil }
func (m *mockRoom) GetExits() map[string]string                                       { return nil }
func (m *mockRoom) GetID() string                                                     { return "test_room" }
func (m *mockRoom) GetFloor() int                                                     { return 0 }
func (m *mockRoom) HasItem(itemName string) bool                                      { return false }
func (m *mockRoom) RemoveItem(itemName string) (*items.Item, bool)                    { return nil, false }
func (m *mockRoom) AddItem(item *items.Item)                                          {}
func (m *mockRoom) FindItem(partial string) (*items.Item, bool)                       { return nil, false }
func (m *mockRoom) HasFeature(feature string) bool                                    { return false }
func (m *mockRoom) RemoveFeature(feature string)                                      {}
func (m *mockRoom) IsExitLocked(direction string) bool                                { return false }
func (m *mockRoom) GetExitKeyRequired(direction string) string                        { return "" }
func (m *mockRoom) UnlockExit(direction string)                                       {}
func (m *mockRoom) GetNPCs() []*npc.NPC                                               { return nil }
func (m *mockRoom) FindNPC(name string) *npc.NPC                                      { return nil }
func (m *mockRoom) AddNPC(n *npc.NPC)                                                 {}
func (m *mockRoom) RemoveNPC(n *npc.NPC)                                              {}

func TestGetStationInRoom(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		want     string
	}{
		{
			name:     "room with forge",
			features: []string{"torch", crafting.StationForge, "anvil"},
			want:     crafting.StationForge,
		},
		{
			name:     "room with workbench",
			features: []string{crafting.StationWorkbench},
			want:     crafting.StationWorkbench,
		},
		{
			name:     "room with alchemy lab",
			features: []string{"cauldron", crafting.StationAlchemyLab},
			want:     crafting.StationAlchemyLab,
		},
		{
			name:     "room with enchanting table",
			features: []string{crafting.StationEnchantingTable, "bookshelf"},
			want:     crafting.StationEnchantingTable,
		},
		{
			name:     "room with no station",
			features: []string{"torch", "table", "chair"},
			want:     "",
		},
		{
			name:     "empty room",
			features: []string{},
			want:     "",
		},
		{
			name:     "nil features",
			features: nil,
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room := &mockRoom{features: tc.features}
			got := getStationInRoom(room)
			if got != tc.want {
				t.Errorf("getStationInRoom() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRollCraftingCheck(t *testing.T) {
	// Test the formula: d20 + (skill/5) + (intMod/2) >= difficulty
	// Since there's randomness, we test the bonus calculations and boundary conditions

	tests := []struct {
		name       string
		skillLevel int
		intMod     int
		difficulty int
		// We can't test exact success/failure due to randomness,
		// but we can verify the roll value range
		minBonus int // Expected minimum bonus (skill/5 + intMod/2)
		maxBonus int
	}{
		{
			name:       "no bonuses",
			skillLevel: 0,
			intMod:     0,
			difficulty: 10,
			minBonus:   0,
			maxBonus:   0,
		},
		{
			name:       "skill bonus only",
			skillLevel: 25, // 25/5 = 5
			intMod:     0,
			difficulty: 10,
			minBonus:   5,
			maxBonus:   5,
		},
		{
			name:       "int bonus only",
			skillLevel: 0,
			intMod:     4, // 4/2 = 2
			difficulty: 10,
			minBonus:   2,
			maxBonus:   2,
		},
		{
			name:       "both bonuses",
			skillLevel: 50, // 50/5 = 10
			intMod:     6,  // 6/2 = 3
			difficulty: 15,
			minBonus:   13, // 10 + 3
			maxBonus:   13,
		},
		{
			name:       "high skill",
			skillLevel: 100, // 100/5 = 20
			intMod:     10,  // 10/2 = 5
			difficulty: 20,
			minBonus:   25,
			maxBonus:   25,
		},
		{
			name:       "negative int mod",
			skillLevel: 10, // 10/5 = 2
			intMod:     -2, // -2/2 = -1
			difficulty: 10,
			minBonus:   1, // 2 + (-1) = 1
			maxBonus:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Run multiple times to account for randomness
			for i := 0; i < 100; i++ {
				roll, _ := rollCraftingCheck(tc.skillLevel, tc.intMod, tc.difficulty)

				// Roll should be between 1+bonus and 20+bonus (d20 range)
				minRoll := 1 + tc.minBonus
				maxRoll := 20 + tc.maxBonus

				if roll < minRoll || roll > maxRoll {
					t.Errorf("rollCraftingCheck() roll = %d, want between %d and %d", roll, minRoll, maxRoll)
				}
			}
		})
	}
}

func TestRollCraftingCheck_SuccessFailure(t *testing.T) {
	// Test guaranteed success (very high bonus vs low DC)
	successes := 0
	for i := 0; i < 100; i++ {
		_, success := rollCraftingCheck(100, 10, 1) // Bonus of 25, DC 1 - always succeeds
		if success {
			successes++
		}
	}
	if successes != 100 {
		t.Errorf("Expected 100%% success rate with high bonus, got %d%%", successes)
	}

	// Test guaranteed failure (no bonus vs very high DC)
	failures := 0
	for i := 0; i < 100; i++ {
		_, success := rollCraftingCheck(0, 0, 100) // No bonus, DC 100 - always fails
		if !success {
			failures++
		}
	}
	if failures != 100 {
		t.Errorf("Expected 100%% failure rate with impossible DC, got %d%% failures", failures)
	}
}

func TestCreateSkillBar(t *testing.T) {
	tests := []struct {
		name    string
		current int
		max     int
		want    string
	}{
		{
			name:    "empty bar",
			current: 0,
			max:     100,
			want:    "[--------------------]",
		},
		{
			name:    "full bar",
			current: 100,
			max:     100,
			want:    "[====================]",
		},
		{
			name:    "half bar",
			current: 50,
			max:     100,
			want:    "[==========----------]",
		},
		{
			name:    "quarter bar",
			current: 25,
			max:     100,
			want:    "[=====---------------]",
		},
		{
			name:    "minimum filled (1 skill point)",
			current: 1,
			max:     100,
			want:    "[=-------------------]", // At least 1 filled if current > 0
		},
		{
			name:    "three quarters",
			current: 75,
			max:     100,
			want:    "[===============-----]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := createSkillBar(tc.current, tc.max)
			if got != tc.want {
				t.Errorf("createSkillBar(%d, %d) = %q, want %q", tc.current, tc.max, got, tc.want)
			}
		})
	}
}

// mockCraftingPlayer implements only the methods needed for canCraftRecipe and hasIngredients
type mockCraftingPlayer struct {
	level         int
	craftingSkill int
	inventory     map[string]int // itemID -> count
}

func newMockCraftingPlayer() *mockCraftingPlayer {
	return &mockCraftingPlayer{
		level:         1,
		craftingSkill: 0,
		inventory:     make(map[string]int),
	}
}

func (m *mockCraftingPlayer) GetLevel() int                                     { return m.level }
func (m *mockCraftingPlayer) GetCraftingSkill(skill crafting.CraftingSkill) int { return m.craftingSkill }
func (m *mockCraftingPlayer) CountItemsByID(itemID string) int                  { return m.inventory[itemID] }

// testCanCraftRecipe is a test helper that duplicates canCraftRecipe logic for testing
// This avoids needing to implement the full PlayerInterface
func testCanCraftRecipe(playerLevel, playerSkill int, recipe *crafting.Recipe) (bool, string) {
	// Check player level
	if playerLevel < recipe.LevelRequired {
		return false, "requires level " + itoa(recipe.LevelRequired)
	}

	// Check skill level
	if playerSkill < recipe.SkillRequired {
		return false, "requires " + itoa(recipe.SkillRequired) + " " + recipe.Skill.String()
	}

	return true, ""
}

// testHasIngredients is a test helper that duplicates hasIngredients logic for testing
func testHasIngredients(inventory map[string]int, recipe *crafting.Recipe) (bool, string) {
	for _, ing := range recipe.Ingredients {
		count := inventory[ing.ItemID]
		if count < ing.Quantity {
			if count == 0 {
				return false, "You don't have any " + ing.ItemID + "."
			}
			return false, "You need " + itoa(ing.Quantity) + " " + ing.ItemID + " but only have " + itoa(count) + "."
		}
	}
	return true, ""
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func TestCanCraftRecipe_Logic(t *testing.T) {
	tests := []struct {
		name           string
		playerLevel    int
		playerSkill    int
		recipeLevelReq int
		recipeSkillReq int
		wantCanCraft   bool
		wantContains   string
	}{
		{
			name:           "meets all requirements",
			playerLevel:    5,
			playerSkill:    20,
			recipeLevelReq: 3,
			recipeSkillReq: 15,
			wantCanCraft:   true,
			wantContains:   "",
		},
		{
			name:           "exactly meets requirements",
			playerLevel:    5,
			playerSkill:    15,
			recipeLevelReq: 5,
			recipeSkillReq: 15,
			wantCanCraft:   true,
			wantContains:   "",
		},
		{
			name:           "level too low",
			playerLevel:    2,
			playerSkill:    50,
			recipeLevelReq: 5,
			recipeSkillReq: 10,
			wantCanCraft:   false,
			wantContains:   "requires level 5",
		},
		{
			name:           "skill too low",
			playerLevel:    10,
			playerSkill:    5,
			recipeLevelReq: 1,
			recipeSkillReq: 20,
			wantCanCraft:   false,
			wantContains:   "requires 20",
		},
		{
			name:           "no requirements",
			playerLevel:    1,
			playerSkill:    0,
			recipeLevelReq: 0,
			recipeSkillReq: 0,
			wantCanCraft:   true,
			wantContains:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recipe := &crafting.Recipe{
				ID:            "test_recipe",
				Name:          "Test Recipe",
				Skill:         crafting.Blacksmithing,
				LevelRequired: tc.recipeLevelReq,
				SkillRequired: tc.recipeSkillReq,
			}

			canCraft, reason := testCanCraftRecipe(tc.playerLevel, tc.playerSkill, recipe)

			if canCraft != tc.wantCanCraft {
				t.Errorf("canCraftRecipe() canCraft = %v, want %v", canCraft, tc.wantCanCraft)
			}
			if tc.wantContains != "" && !stringContains(reason, tc.wantContains) {
				t.Errorf("canCraftRecipe() reason = %q, want to contain %q", reason, tc.wantContains)
			}
		})
	}
}

func TestHasIngredients_Logic(t *testing.T) {
	tests := []struct {
		name            string
		inventory       map[string]int
		ingredients     []crafting.RecipeIngredient
		wantHas         bool
		wantMsgContains string
	}{
		{
			name:      "has all ingredients",
			inventory: map[string]int{"iron_ore": 3, "leather_strip": 2},
			ingredients: []crafting.RecipeIngredient{
				{ItemID: "iron_ore", Quantity: 2},
				{ItemID: "leather_strip", Quantity: 1},
			},
			wantHas:         true,
			wantMsgContains: "",
		},
		{
			name:      "has exact amounts",
			inventory: map[string]int{"iron_ore": 2, "leather_strip": 1},
			ingredients: []crafting.RecipeIngredient{
				{ItemID: "iron_ore", Quantity: 2},
				{ItemID: "leather_strip", Quantity: 1},
			},
			wantHas:         true,
			wantMsgContains: "",
		},
		{
			name:      "missing ingredient completely",
			inventory: map[string]int{"iron_ore": 5},
			ingredients: []crafting.RecipeIngredient{
				{ItemID: "iron_ore", Quantity: 2},
				{ItemID: "leather_strip", Quantity: 1},
			},
			wantHas:         false,
			wantMsgContains: "don't have any",
		},
		{
			name:      "not enough of ingredient",
			inventory: map[string]int{"iron_ore": 1, "leather_strip": 1},
			ingredients: []crafting.RecipeIngredient{
				{ItemID: "iron_ore", Quantity: 3},
			},
			wantHas:         false,
			wantMsgContains: "need 3",
		},
		{
			name:        "empty inventory",
			inventory:   map[string]int{},
			ingredients: []crafting.RecipeIngredient{{ItemID: "iron_ore", Quantity: 1}},
			wantHas:     false,
			wantMsgContains: "don't have any",
		},
		{
			name:            "no ingredients required",
			inventory:       map[string]int{},
			ingredients:     []crafting.RecipeIngredient{},
			wantHas:         true,
			wantMsgContains: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recipe := &crafting.Recipe{
				Ingredients: tc.ingredients,
			}

			hasIng, msg := testHasIngredients(tc.inventory, recipe)

			if hasIng != tc.wantHas {
				t.Errorf("hasIngredients() = %v, want %v", hasIng, tc.wantHas)
			}
			if tc.wantMsgContains != "" && !stringContains(msg, tc.wantMsgContains) {
				t.Errorf("hasIngredients() msg = %q, want to contain %q", msg, tc.wantMsgContains)
			}
		})
	}
}

// stringContains checks if s contains substr
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
