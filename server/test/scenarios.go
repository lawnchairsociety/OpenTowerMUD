package test

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
	_ "modernc.org/sqlite"
)

// uniqueCounter provides unique IDs for test players within a single run
var uniqueCounter uint64

// uniqueName generates a unique name by appending a counter suffix
func uniqueName(base string) string {
	counter := atomic.AddUint64(&uniqueCounter, 1)
	return fmt.Sprintf("%s%d", base, counter)
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

	// Group 1: Core Connection & Movement
	results = append(results, TestBasicConnection(serverAddr))
	results = append(results, TestMovement(serverAddr))
	results = append(results, TestMultipleClientsMovement(serverAddr))
	results = append(results, TestLookCommand(serverAddr))

	// Group 2: Communication
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

	// Group 3: Inventory & Shopping
	results = append(results, TestInventorySystem(serverAddr))
	results = append(results, TestSellItem(serverAddr))
	results = append(results, TestEquipment(serverAddr))
	results = append(results, TestConsumables(serverAddr))

	// Group 4: Combat System (non-killing tests first)
	results = append(results, TestUnattackableNPC(serverAddr))
	results = append(results, TestAttackRolls(serverAddr))
	results = append(results, TestFleeCommand(serverAddr))
	results = append(results, TestCombatAndKill(serverAddr))
	results = append(results, TestMobRespawn(serverAddr))

	// Group 5: Magic System
	results = append(results, TestSpellCasting(serverAddr))
	results = append(results, TestHealOtherPlayer(serverAddr))
	results = append(results, TestBlessSpell(serverAddr))
	results = append(results, TestSpellDamageWithModifiers(serverAddr))

	// Group 6: Room Features
	results = append(results, TestPrayCommand(serverAddr))
	results = append(results, TestPortalCommand(serverAddr))
	results = append(results, TestConsiderSelf(serverAddr))
	results = append(results, TestLookAtFeature(serverAddr))
	results = append(results, TestTrainCommand(serverAddr))
	results = append(results, TestTrainerLocations(serverAddr))

	// Group 7: Tower & Progression
	results = append(results, TestTowerClimb(serverAddr))
	results = append(results, TestPlayerLevelUp(serverAddr))
	results = append(results, TestAbilityScores(serverAddr))
	results = append(results, TestScoreCommand(serverAddr))
	results = append(results, TestClassCommand(serverAddr))
	results = append(results, TestStartingEquipment(serverAddr))
	results = append(results, TestLookAtPlayer(serverAddr))

	// Group 8: Account System
	results = append(results, TestAccountSystem(serverAddr))
	results = append(results, TestInventoryPersistence(serverAddr))
	results = append(results, TestLastVisitedCityRespawn(serverAddr))

	// Group 9: Admin Commands
	results = append(results, TestAdminCommandsHidden(serverAddr))
	results = append(results, TestAdminAnnounce(serverAddr))

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
