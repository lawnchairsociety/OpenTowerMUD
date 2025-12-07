package test

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
	_ "modernc.org/sqlite"
)

// uniqueCounter provides unique IDs for test players within a single run
var uniqueCounter uint64

// uniqueName generates a unique name by appending a letter-based suffix
// Character names can only contain letters (no numbers), so we convert
// the counter to a base-26 letter sequence (a, b, ..., z, aa, ab, ...)
func uniqueName(base string) string {
	counter := atomic.AddUint64(&uniqueCounter, 1)
	suffix := counterToLetters(counter)
	return base + suffix
}

// counterToLetters converts a number to a letter sequence (1=a, 2=b, ..., 26=z, 27=aa, 28=ab, ...)
func counterToLetters(n uint64) string {
	if n == 0 {
		return "a"
	}
	result := ""
	for n > 0 {
		n-- // Make it 0-indexed
		result = string(rune('a'+(n%26))) + result
		n /= 26
	}
	return result
}

// Verbose controls whether detailed logging is shown during tests
var Verbose = false

// TestResult represents the result of a test
type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

// logAction logs a test action when verbose mode is enabled
func logAction(testName, action string) {
	if Verbose {
		fmt.Printf("  [%s] %s\n", testName, action)
	}
}

// logResult logs an expected vs actual result when verbose mode is enabled
func logResult(testName string, success bool, detail string) {
	if Verbose {
		status := "OK"
		if !success {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %s: %s\n", testName, status, detail)
	}
}

// =============================================================================
// Navigation Helpers
// =============================================================================

// navigateToTrainingHall moves a client from Town Square to Training Hall
// Path: Town Square -> Market Street -> Artisan's Market -> Military District
//       -> Military District East -> Training Hall
func navigateToTrainingHall(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Military District
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east") // Military District East
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Training Hall
	time.Sleep(200 * time.Millisecond)
}

// navigateToTowerEntrance moves a client from Town Square to Tower Entrance
// Path: Town Square -> Market Street -> Artisan's Market -> Military District
//       -> Tower Entrance
func navigateToTowerEntrance(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Military District
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south") // Tower Entrance
	time.Sleep(200 * time.Millisecond)
}

// navigateToGeneralStore moves a client from Town Square to General Store
// Path: Town Square -> Market Street -> General Store
func navigateToGeneralStore(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east") // General Store
	time.Sleep(200 * time.Millisecond)
}

// navigateToTavern moves a client from Town Square to Tavern
// Path: Town Square -> Market Street -> Tavern
func navigateToTavern(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("west") // Tavern
	time.Sleep(200 * time.Millisecond)
}

// navigateToTemple moves a client from Town Square to Temple
// Path: Town Square -> Temple
func navigateToTemple(client *testclient.TestClient) {
	client.SendCommand("east") // Temple
	time.Sleep(200 * time.Millisecond)
}

// =============================================================================
// Test Runner
// =============================================================================

// RunAllTests runs all integration tests
func RunAllTests(serverAddr string) []TestResult {
	results := make([]TestResult, 0)

	// Group 1: Connection & Account System
	results = append(results, TestBasicConnection(serverAddr))
	results = append(results, TestMovement(serverAddr))
	results = append(results, TestMultipleClientsMovement(serverAddr))
	results = append(results, TestLookCommand(serverAddr))
	results = append(results, TestExitsCommand(serverAddr))
	results = append(results, TestUnlockCommand(serverAddr))
	results = append(results, TestUnlockNoArgs(serverAddr))
	results = append(results, TestMovementAliases(serverAddr))
	results = append(results, TestAccountSystem(serverAddr))
	results = append(results, TestInventoryPersistence(serverAddr))
	results = append(results, TestRoomPersistence(serverAddr))
	results = append(results, TestLastVisitedCityRespawn(serverAddr))
	results = append(results, TestPasswordChangeNoArgs(serverAddr))
	results = append(results, TestPasswordChangeWrongOld(serverAddr))

	// Group 2: Communication & Social
	results = append(results, TestSayCommand(serverAddr))
	results = append(results, TestTellCommand(serverAddr))
	results = append(results, TestShoutCommand(serverAddr))
	results = append(results, TestEmoteCommand(serverAddr))
	results = append(results, TestGiveItem(serverAddr))
	results = append(results, TestGiveGold(serverAddr))
	results = append(results, TestGiveRequiresSameRoom(serverAddr))
	results = append(results, TestChatFilterReplace(serverAddr))
	results = append(results, TestAntispamRateLimit(serverAddr))
	results = append(results, TestAntispamRepeatMessage(serverAddr))
	results = append(results, TestIgnoreCommand(serverAddr))
	results = append(results, TestIgnoreTell(serverAddr))
	results = append(results, TestUnignoreCommand(serverAddr))
	results = append(results, TestReportCommand(serverAddr))
	results = append(results, TestIgnoreList(serverAddr))
	results = append(results, TestWhoCommand(serverAddr))
	results = append(results, TestWhoMultiplePlayers(serverAddr))

	// Group 3: Character Info & Stats
	results = append(results, TestScoreCommand(serverAddr))
	results = append(results, TestClassCommand(serverAddr))
	results = append(results, TestRaceCommand(serverAddr))
	results = append(results, TestRacesCommand(serverAddr))
	results = append(results, TestAbilityScores(serverAddr))
	results = append(results, TestStartingEquipment(serverAddr))
	results = append(results, TestLookAtPlayer(serverAddr))
	results = append(results, TestConsiderSelf(serverAddr))
	results = append(results, TestSkillsCommand(serverAddr))
	results = append(results, TestTitleCommand(serverAddr))
	results = append(results, TestPlayerLevelUp(serverAddr))

	// Group 4: Inventory & Shopping
	results = append(results, TestInventorySystem(serverAddr))
	results = append(results, TestSellItem(serverAddr))
	results = append(results, TestEquipment(serverAddr))
	results = append(results, TestConsumables(serverAddr))
	results = append(results, TestDrinkCommand(serverAddr))
	results = append(results, TestDrinkPotion(serverAddr))
	results = append(results, TestDrinkNonDrinkable(serverAddr))
	results = append(results, TestHoldCommand(serverAddr))
	results = append(results, TestUseCommand(serverAddr))
	results = append(results, TestUseAltar(serverAddr))
	results = append(results, TestTakeCommand(serverAddr))
	results = append(results, TestGetCommand(serverAddr))
	results = append(results, TestPickupCommand(serverAddr))

	// Group 5: Combat System
	results = append(results, TestUnattackableNPC(serverAddr))
	results = append(results, TestAttackRolls(serverAddr))
	results = append(results, TestFleeCommand(serverAddr))
	results = append(results, TestCombatAndKill(serverAddr))
	results = append(results, TestMobRespawn(serverAddr))
	results = append(results, TestCombatThreat(serverAddr))
	results = append(results, TestCombatNotInCombat(serverAddr))
	results = append(results, TestCombatConsiderMob(serverAddr))
	results = append(results, TestCombatDamageFormula(serverAddr))

	// Group 6: Magic System
	results = append(results, TestSpellCasting(serverAddr))
	results = append(results, TestHealOtherPlayer(serverAddr))
	results = append(results, TestBlessSpell(serverAddr))
	results = append(results, TestSpellDamageWithModifiers(serverAddr))

	// Group 7: Room Features & Tower
	results = append(results, TestPrayCommand(serverAddr))
	results = append(results, TestPortalCommand(serverAddr))
	results = append(results, TestLookAtFeature(serverAddr))
	results = append(results, TestTowerClimb(serverAddr))

	// Group 8: NPCs & Training
	results = append(results, TestTrainCommand(serverAddr))
	results = append(results, TestTrainerLocations(serverAddr))
	results = append(results, TestLearnFromTrainer(serverAddr))
	results = append(results, TestCraftingTrainerLocations(serverAddr))
	results = append(results, TestTimeCommand(serverAddr))
	results = append(results, TestSleepCommand(serverAddr))
	results = append(results, TestWakeCommand(serverAddr))
	results = append(results, TestWakeWhenNotSleeping(serverAddr))
	results = append(results, TestStandCommand(serverAddr))
	results = append(results, TestStandWhenStanding(serverAddr))

	// Group 9: Crafting System
	results = append(results, TestCraftingStationForge(serverAddr))
	results = append(results, TestCraftingStationWorkbench(serverAddr))
	results = append(results, TestCraftingStationAlchemyLab(serverAddr))
	results = append(results, TestCraftingStationEnchantingTable(serverAddr))
	results = append(results, TestCraftWithoutStation(serverAddr))
	results = append(results, TestCraftWithoutMaterials(serverAddr))
	results = append(results, TestBuyCraftingMaterials(serverAddr))
	results = append(results, TestCraftRecipeInfo(serverAddr))
	results = append(results, TestCraftingSkillPersistence(serverAddr))

	// Group 10: Quest System
	results = append(results, TestQuestCommand(serverAddr))
	results = append(results, TestQuestJournalAlias(serverAddr))
	results = append(results, TestQuestGiverNPC(serverAddr))
	results = append(results, TestQuestListWithNPC(serverAddr))
	results = append(results, TestAcceptQuest(serverAddr))
	results = append(results, TestQuestProgress(serverAddr))
	results = append(results, TestQuestPrerequisites(serverAddr))
	results = append(results, TestCompleteQuestNotReady(serverAddr))

	// Group 11: Admin Commands
	results = append(results, TestAdminCommandsHidden(serverAddr))
	results = append(results, TestAdminAnnounce(serverAddr))

	// Group 12: Player Stalls
	results = append(results, TestStallOpen(serverAddr))
	results = append(results, TestStallClose(serverAddr))
	results = append(results, TestStallAddItem(serverAddr))
	results = append(results, TestStallRemoveItem(serverAddr))
	results = append(results, TestStallList(serverAddr))
	results = append(results, TestStallBrowse(serverAddr))
	results = append(results, TestStallPurchase(serverAddr))
	results = append(results, TestStallCloseOnRoomChange(serverAddr))
	results = append(results, TestStallRequiresCity(serverAddr))

	return results
}

// testEntry holds a test function and its name
type testEntry struct {
	Name string
	Func func(string) TestResult
}

// getAllTests returns all test entries in order
func getAllTests() []testEntry {
	return []testEntry{
		// Group 1: Connection & Account System
		{"Basic Connection", TestBasicConnection},
		{"Movement", TestMovement},
		{"Multiple Clients Movement", TestMultipleClientsMovement},
		{"Look Command", TestLookCommand},
		{"Exits Command", TestExitsCommand},
		{"Unlock Command", TestUnlockCommand},
		{"Unlock No Args", TestUnlockNoArgs},
		{"Movement Aliases", TestMovementAliases},
		{"Account System", TestAccountSystem},
		{"Inventory Persistence", TestInventoryPersistence},
		{"Room Persistence", TestRoomPersistence},
		{"Death Respawn", TestLastVisitedCityRespawn},
		{"Password Change No Args", TestPasswordChangeNoArgs},
		{"Password Change Wrong Old", TestPasswordChangeWrongOld},

		// Group 2: Communication & Social
		{"Say Command", TestSayCommand},
		{"Tell Command", TestTellCommand},
		{"Shout Command", TestShoutCommand},
		{"Emote Command", TestEmoteCommand},
		{"Give Item", TestGiveItem},
		{"Give Gold", TestGiveGold},
		{"Give Requires Same Room", TestGiveRequiresSameRoom},
		{"Chat Filter Replace", TestChatFilterReplace},
		{"Antispam Rate Limit", TestAntispamRateLimit},
		{"Antispam Repeat Message", TestAntispamRepeatMessage},
		{"Ignore Command", TestIgnoreCommand},
		{"Ignore Tell", TestIgnoreTell},
		{"Unignore Command", TestUnignoreCommand},
		{"Report Command", TestReportCommand},
		{"Ignore List", TestIgnoreList},
		{"Who Command", TestWhoCommand},
		{"Who Multiple Players", TestWhoMultiplePlayers},

		// Group 3: Character Info & Stats
		{"Score Command", TestScoreCommand},
		{"Class Command", TestClassCommand},
		{"Race Command", TestRaceCommand},
		{"Races Command", TestRacesCommand},
		{"Ability Scores", TestAbilityScores},
		{"Starting Equipment", TestStartingEquipment},
		{"Look At Player", TestLookAtPlayer},
		{"Consider Self", TestConsiderSelf},
		{"Skills Command", TestSkillsCommand},
		{"Title Command", TestTitleCommand},
		{"Player Level Up", TestPlayerLevelUp},

		// Group 4: Inventory & Shopping
		{"Inventory System", TestInventorySystem},
		{"Sell Item", TestSellItem},
		{"Equipment", TestEquipment},
		{"Consumables", TestConsumables},
		{"Drink Command", TestDrinkCommand},
		{"Drink Potion", TestDrinkPotion},
		{"Drink Non-Drinkable", TestDrinkNonDrinkable},
		{"Hold Command", TestHoldCommand},
		{"Use Command", TestUseCommand},
		{"Use Altar", TestUseAltar},
		{"Take Command", TestTakeCommand},
		{"Get Command", TestGetCommand},
		{"Pickup Command", TestPickupCommand},

		// Group 5: Combat System
		{"Unattackable NPC", TestUnattackableNPC},
		{"Attack Rolls", TestAttackRolls},
		{"Flee Command", TestFleeCommand},
		{"Combat and Kill", TestCombatAndKill},
		{"Mob Respawn", TestMobRespawn},
		{"Combat Threat", TestCombatThreat},
		{"Combat Not In Combat", TestCombatNotInCombat},
		{"Combat Consider Mob", TestCombatConsiderMob},
		{"Combat Damage Formula", TestCombatDamageFormula},

		// Group 6: Magic System
		{"Spell Casting", TestSpellCasting},
		{"Heal Other Player", TestHealOtherPlayer},
		{"Bless Spell", TestBlessSpell},
		{"Spell Damage Modifiers", TestSpellDamageWithModifiers},

		// Group 7: Room Features & Tower
		{"Pray Command", TestPrayCommand},
		{"Portal Command", TestPortalCommand},
		{"Look At Feature", TestLookAtFeature},
		{"Tower Climb", TestTowerClimb},

		// Group 8: NPCs & Training
		{"Train Command", TestTrainCommand},
		{"Trainer Locations", TestTrainerLocations},
		{"Learn From Trainer", TestLearnFromTrainer},
		{"Crafting Trainer Locations", TestCraftingTrainerLocations},
		{"Time Command", TestTimeCommand},
		{"Sleep Command", TestSleepCommand},
		{"Wake Command", TestWakeCommand},
		{"Wake When Not Sleeping", TestWakeWhenNotSleeping},
		{"Stand Command", TestStandCommand},
		{"Stand When Standing", TestStandWhenStanding},

		// Group 9: Crafting System
		{"Crafting Station - Forge", TestCraftingStationForge},
		{"Crafting Station - Workbench", TestCraftingStationWorkbench},
		{"Crafting Station - Alchemy Lab", TestCraftingStationAlchemyLab},
		{"Crafting Station - Enchanting Table", TestCraftingStationEnchantingTable},
		{"Craft Without Station", TestCraftWithoutStation},
		{"Craft Without Materials", TestCraftWithoutMaterials},
		{"Buy Crafting Materials", TestBuyCraftingMaterials},
		{"Craft Recipe Info", TestCraftRecipeInfo},
		{"Crafting Skill Persistence", TestCraftingSkillPersistence},

		// Group 10: Quest System
		{"Quest Command", TestQuestCommand},
		{"Quest Journal Alias", TestQuestJournalAlias},
		{"Quest Giver NPC", TestQuestGiverNPC},
		{"Quest List With NPC", TestQuestListWithNPC},
		{"Accept Quest", TestAcceptQuest},
		{"Quest Progress", TestQuestProgress},
		{"Quest Prerequisites", TestQuestPrerequisites},
		{"Complete Quest Not Ready", TestCompleteQuestNotReady},

		// Group 11: Admin Commands
		{"Admin Commands Hidden", TestAdminCommandsHidden},
		{"Admin Announce", TestAdminAnnounce},

		// Group 12: Player Stalls
		{"Stall Open", TestStallOpen},
		{"Stall Close", TestStallClose},
		{"Stall Add Item", TestStallAddItem},
		{"Stall Remove Item", TestStallRemoveItem},
		{"Stall List", TestStallList},
		{"Stall Browse", TestStallBrowse},
		{"Stall Purchase", TestStallPurchase},
		{"Stall Close On Room Change", TestStallCloseOnRoomChange},
		{"Stall Requires City", TestStallRequiresCity},
	}
}

// GetTestNames returns the names of all available tests
func GetTestNames() []string {
	tests := getAllTests()
	names := make([]string, len(tests))
	for i, t := range tests {
		names[i] = t.Name
	}
	return names
}

// RunFilteredTests runs only tests whose names contain the filter string (case-insensitive)
func RunFilteredTests(serverAddr string, filter string) []TestResult {
	results := make([]TestResult, 0)
	filterLower := strings.ToLower(filter)

	for _, t := range getAllTests() {
		if strings.Contains(strings.ToLower(t.Name), filterLower) {
			results = append(results, t.Func(serverAddr))
		}
	}

	return results
}

// PrintResults prints all test results in a formatted way
func PrintResults(results []TestResult) {
	passed := 0
	failed := 0

	fmt.Println("============================================================")
	fmt.Println("Integration Test Results")
	fmt.Println("============================================================")
	fmt.Println()

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		fmt.Printf("[%s] %s: %s\n", status, r.Name, r.Message)
	}

	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Total: %d | Passed: %d | Failed: %d\n", len(results), passed, failed)
	fmt.Println("------------------------------------------------------------")
}

// connectToTestDB opens a connection to the test database
func connectToTestDB() (*sql.DB, error) {
	return sql.Open("sqlite", "data/test/players_test.db")
}
