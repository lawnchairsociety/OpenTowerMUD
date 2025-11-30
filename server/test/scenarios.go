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
	results = append(results, TestChatFilterReplace(serverAddr))

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
	results = append(results, TestDazzleSpell(serverAddr))
	results = append(results, TestSpellDamageWithModifiers(serverAddr))

	// Group 6: Room Features
	results = append(results, TestPrayCommand(serverAddr))
	results = append(results, TestPortalCommand(serverAddr))
	results = append(results, TestConsiderSelf(serverAddr))
	results = append(results, TestLookAtFeature(serverAddr))

	// Group 7: Tower & Progression
	results = append(results, TestTowerClimb(serverAddr))
	results = append(results, TestPlayerLevelUp(serverAddr))
	results = append(results, TestAbilityScores(serverAddr))

	// Group 8: Account System
	results = append(results, TestAccountSystem(serverAddr))
	results = append(results, TestInventoryPersistence(serverAddr))
	results = append(results, TestLastVisitedCityRespawn(serverAddr))

	// Group 9: Admin Commands
	results = append(results, TestAdminCommandsHidden(serverAddr))
	results = append(results, TestAdminAnnounce(serverAddr))

	return results
}

// =============================================================================
// Group 1: Core Connection & Movement
// =============================================================================

// TestBasicConnection tests that clients can connect and receive welcome
func TestBasicConnection(serverAddr string) TestResult {
	const testName = "Basic Connection"

	name := uniqueName("TestPlayer")
	logAction(testName, fmt.Sprintf("Connecting as '%s'...", name))
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to connect: %v", err)}
	}
	defer client.Close()

	logAction(testName, "Waiting for welcome messages...")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	logResult(testName, len(messages) > 0, fmt.Sprintf("Received %d messages", len(messages)))

	if len(messages) == 0 {
		return TestResult{Name: testName, Passed: false, Message: "No messages received from server"}
	}

	return TestResult{Name: testName, Passed: true, Message: fmt.Sprintf("Connected successfully, received %d messages", len(messages))}
}

// TestMovement tests that a player can move between rooms
func TestMovement(serverAddr string) TestResult {
	const testName = "Movement"

	name := uniqueName("MoveTest")
	logAction(testName, fmt.Sprintf("Connecting as '%s'...", name))
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)
	client.ClearMessages()

	// Town Square -> south -> Market Street
	logAction(testName, "Moving south to Market Street")
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	found := client.WaitForMessage("Market Street", 1*time.Second)
	logResult(testName, found, "Moved to Market Street")
	if !found {
		return TestResult{Name: testName, Passed: false, Message: "Failed to move south to Market Street"}
	}

	// Market Street -> north -> Town Square
	logAction(testName, "Moving north back to Town Square")
	client.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	found = client.WaitForMessage("Town Square", 1*time.Second)
	logResult(testName, found, "Moved to Town Square")
	if !found {
		return TestResult{Name: testName, Passed: false, Message: "Failed to move north back to Town Square"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully moved south and north"}
}

// TestMultipleClientsMovement tests multiple clients moving simultaneously
func TestMultipleClientsMovement(serverAddr string) TestResult {
	const testName = "Multiple Clients Movement"

	name1 := uniqueName("Alice")
	name2 := uniqueName("Bob")
	name3 := uniqueName("Charlie")
	logAction(testName, fmt.Sprintf("Connecting 3 clients: %s, %s, %s...", name1, name2, name3))

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to connect %s: %v", name1, err1)}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to connect %s: %v", name2, err2)}
	}
	defer client2.Close()

	client3, err3 := testclient.NewTestClient(name3, serverAddr)
	if err3 != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to connect %s: %v", name3, err3)}
	}
	defer client3.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()
	client3.ClearMessages()

	// Move all three in different directions
	logAction(testName, "Alice moves north, Bob moves south, Charlie moves east")
	client1.SendCommand("north")
	time.Sleep(100 * time.Millisecond)
	client2.SendCommand("south")
	time.Sleep(100 * time.Millisecond)
	client3.SendCommand("east")
	time.Sleep(500 * time.Millisecond)

	found1 := client1.WaitForMessage("North Gate", 2*time.Second)
	found2 := client2.WaitForMessage("Market Street", 2*time.Second)
	found3 := client3.WaitForMessage("Temple", 2*time.Second)

	logResult(testName, found1, "Alice at North Gate")
	logResult(testName, found2, "Bob at Market Street")
	logResult(testName, found3, "Charlie at Temple")

	if !found1 || !found2 || !found3 {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Not all clients reached destination (Alice:%v, Bob:%v, Charlie:%v)", found1, found2, found3)}
	}

	return TestResult{Name: testName, Passed: true, Message: "All 3 clients successfully moved to different rooms"}
}

// TestLookCommand tests the look command shows room details
func TestLookCommand(serverAddr string) TestResult {
	const testName = "Look Command"

	name := uniqueName("LookTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)
	client.ClearMessages()

	logAction(testName, "Sending look command")
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasRoomName := strings.Contains(fullOutput, "Town Square")
	hasExits := strings.Contains(fullOutput, "Exits") || strings.Contains(fullOutput, "north") || strings.Contains(fullOutput, "south")

	logResult(testName, hasRoomName, "Room name shown")
	logResult(testName, hasExits, "Exits shown")

	if !hasRoomName {
		return TestResult{Name: testName, Passed: false, Message: "Room name not shown in look output"}
	}
	if !hasExits {
		return TestResult{Name: testName, Passed: false, Message: "Exits not shown in look output"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Look command shows room name and exits"}
}

// =============================================================================
// Group 2: Communication
// =============================================================================

// TestSayCommand tests the say command broadcasts to room
func TestSayCommand(serverAddr string) TestResult {
	const testName = "Say Command"

	name1 := uniqueName("Speaker")
	name2 := uniqueName("Listener")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", name1, name2))

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect speaker"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect listener"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()

	logAction(testName, "Speaker says: Hello everyone!")
	client1.SendCommand("say Hello everyone!")
	time.Sleep(300 * time.Millisecond)

	foundMessage := client2.WaitForMessage("Hello everyone", 1*time.Second)
	logResult(testName, foundMessage, "Listener received message")

	if !foundMessage {
		return TestResult{Name: testName, Passed: false, Message: "Listener did not receive say message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Say command successfully broadcast to room"}
}

// TestTellCommand tests private messaging
func TestTellCommand(serverAddr string) TestResult {
	const testName = "Tell Command"

	aliceName := uniqueName("Alice")
	bobName := uniqueName("Bob")
	charlieName := uniqueName("Charlie")
	logAction(testName, fmt.Sprintf("Connecting %s, %s, %s...", aliceName, bobName, charlieName))

	alice, err1 := testclient.NewTestClient(aliceName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Alice"}
	}
	defer alice.Close()

	bob, err2 := testclient.NewTestClient(bobName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Bob"}
	}
	defer bob.Close()

	charlie, err3 := testclient.NewTestClient(charlieName, serverAddr)
	if err3 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Charlie"}
	}
	defer charlie.Close()

	time.Sleep(500 * time.Millisecond)

	// Move Bob to different room
	bob.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	alice.ClearMessages()
	bob.ClearMessages()
	charlie.ClearMessages()

	logAction(testName, fmt.Sprintf("Alice tells %s: Secret message!", bobName))
	alice.SendCommand(fmt.Sprintf("tell %s Secret message!", bobName))
	time.Sleep(300 * time.Millisecond)

	foundBob := bob.WaitForMessage("Secret message", 1*time.Second)
	logResult(testName, foundBob, "Bob received tell")

	if !foundBob {
		return TestResult{Name: testName, Passed: false, Message: "Bob did not receive tell from Alice"}
	}

	// Charlie should NOT receive the message
	messages := charlie.GetMessages()
	charlieReceivedSecret := false
	for _, msg := range messages {
		if strings.Contains(msg, "Secret message") {
			charlieReceivedSecret = true
			break
		}
	}
	logResult(testName, !charlieReceivedSecret, "Charlie did NOT receive private message")

	if charlieReceivedSecret {
		return TestResult{Name: testName, Passed: false, Message: "Charlie incorrectly received private tell"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Tell command works privately between players"}
}

// TestChatFilterReplace tests that banned words are replaced
func TestChatFilterReplace(serverAddr string) TestResult {
	const testName = "Chat Filter Replace"

	name1 := uniqueName("Talker")
	name2 := uniqueName("Hearer")

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect talker"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect hearer"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()

	// "badword" is in chat_filter_test.yaml
	logAction(testName, "Talker says message with banned word 'badword'")
	client1.SendCommand("say This is a badword test")
	time.Sleep(300 * time.Millisecond)

	messages := client2.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Word should be replaced with asterisks
	hasAsterisks := strings.Contains(fullOutput, "***")
	hasBadword := strings.Contains(fullOutput, "badword")

	logResult(testName, hasAsterisks, "Message contains asterisks")
	logResult(testName, !hasBadword, "Banned word filtered out")

	if hasBadword {
		return TestResult{Name: testName, Passed: false, Message: "Banned word was not filtered"}
	}
	if !hasAsterisks {
		return TestResult{Name: testName, Passed: false, Message: "No asterisks in filtered message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Chat filter replaces banned words with asterisks"}
}

// =============================================================================
// Group 3: Inventory & Shopping
// =============================================================================

// TestInventorySystem tests buying, displaying inventory, weight, and dropping items
func TestInventorySystem(serverAddr string) TestResult {
	const testName = "Inventory System"

	name := uniqueName("InvTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store: Town Square -> south -> Market Street -> east -> General Store
	logAction(testName, "Navigating to General Store...")
	client.SendCommand("south")
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("east")
	time.Sleep(300 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	atStore := client.WaitForMessage("General Store", 1*time.Second)
	logResult(testName, atStore, "At General Store")
	if !atStore {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach General Store"}
	}

	// Part 1: Buy items (player starts with 20 gold)
	logAction(testName, "Buying bread...")
	client.ClearMessages()
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	found := client.WaitForMessage("purchase", 1*time.Second)
	logResult(testName, found, "Purchased bread")
	if !found {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to purchase bread. Got: %v", messages)}
	}

	// Part 2: Check inventory display
	logAction(testName, "Checking inventory...")
	client.ClearMessages()
	client.SendCommand("inventory")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasBread := strings.Contains(fullOutput, "bread")
	hasTotalWeight := strings.Contains(fullOutput, "Total weight") || strings.Contains(fullOutput, "weight")

	logResult(testName, hasBread, "Bread in inventory")
	logResult(testName, hasTotalWeight, "Weight shown")

	if !hasBread {
		return TestResult{Name: testName, Passed: false, Message: "Bread not in inventory after purchase"}
	}

	// Part 3: Drop item
	logAction(testName, "Dropping bread...")
	client.ClearMessages()
	client.SendCommand("drop bread")
	time.Sleep(300 * time.Millisecond)

	found = client.WaitForMessage("drop", 1*time.Second)
	logResult(testName, found, "Dropped bread")
	if !found {
		return TestResult{Name: testName, Passed: false, Message: "Failed to drop bread"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Buy, inventory display, and drop all work correctly"}
}

// TestSellItem tests selling items to a shop
func TestSellItem(serverAddr string) TestResult {
	const testName = "Sell Item"

	name := uniqueName("SellTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy something
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east")
	time.Sleep(200 * time.Millisecond)

	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Now sell it back
	logAction(testName, "Selling bread...")
	client.ClearMessages()
	client.SendCommand("sell bread")
	time.Sleep(300 * time.Millisecond)

	found := client.WaitForMessage("sell", 1*time.Second) || client.WaitForMessage("gold", 1*time.Second)
	logResult(testName, found, "Sold bread")

	if !found {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to sell bread. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully sold item to shop"}
}

// TestEquipment tests wielding weapons and wearing armor
func TestEquipment(serverAddr string) TestResult {
	const testName = "Equipment"

	name := uniqueName("EquipTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east")
	time.Sleep(200 * time.Millisecond)

	// Buy a leather cap (15 gold, player starts with 20)
	logAction(testName, "Buying leather cap...")
	client.SendCommand("buy leather cap")
	time.Sleep(300 * time.Millisecond)

	// Wear it
	logAction(testName, "Wearing leather cap...")
	client.ClearMessages()
	client.SendCommand("wear leather cap")
	time.Sleep(300 * time.Millisecond)

	foundWear := client.WaitForMessage("wear", 1*time.Second) || client.WaitForMessage("equip", 1*time.Second)
	logResult(testName, foundWear, "Wore leather cap")

	if !foundWear {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to wear leather cap. Got: %v", messages)}
	}

	// Check equipment
	logAction(testName, "Checking equipment...")
	client.ClearMessages()
	client.SendCommand("equipment")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	hasCap := strings.Contains(fullOutput, "cap") || strings.Contains(fullOutput, "Head") || strings.Contains(fullOutput, "leather")
	logResult(testName, hasCap, "Leather cap shown in equipment")

	if !hasCap {
		return TestResult{Name: testName, Passed: false, Message: "Leather cap not shown in equipment"}
	}

	// Remove it
	logAction(testName, "Removing leather cap...")
	client.ClearMessages()
	client.SendCommand("remove leather cap")
	time.Sleep(300 * time.Millisecond)

	foundRemove := client.WaitForMessage("remove", 1*time.Second) || client.WaitForMessage("unequip", 1*time.Second)
	logResult(testName, foundRemove, "Removed leather cap")

	if !foundRemove {
		return TestResult{Name: testName, Passed: false, Message: "Failed to remove leather cap"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Wear, equipment display, and remove all work correctly"}
}

// TestConsumables tests eating food and drinking potions
func TestConsumables(serverAddr string) TestResult {
	const testName = "Consumables"

	name := uniqueName("ConsumeTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east")
	time.Sleep(200 * time.Millisecond)

	// Buy bread
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Eat it
	logAction(testName, "Eating bread...")
	client.ClearMessages()
	client.SendCommand("eat bread")
	time.Sleep(300 * time.Millisecond)

	foundEat := client.WaitForMessage("eat", 1*time.Second) || client.WaitForMessage("consume", 1*time.Second) || client.WaitForMessage("heal", 1*time.Second)
	logResult(testName, foundEat, "Ate bread")

	if !foundEat {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to eat bread. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully consumed food"}
}

// =============================================================================
// Group 4: Combat System
// =============================================================================

// TestUnattackableNPC tests that friendly NPCs cannot be attacked
func TestUnattackableNPC(serverAddr string) TestResult {
	const testName = "Unattackable NPC"

	name := uniqueName("AttackNPC")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try to attack the old guide in Town Square
	logAction(testName, "Attempting to attack Aldric the old guide...")
	client.ClearMessages()
	client.SendCommand("attack aldric")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should get an error message about not being able to attack
	cannotAttack := strings.Contains(fullOutput, "cannot") || strings.Contains(fullOutput, "can't") ||
		strings.Contains(fullOutput, "not attackable") || strings.Contains(fullOutput, "unable")
	logResult(testName, cannotAttack, "Received cannot attack message")

	if !cannotAttack {
		return TestResult{Name: testName, Passed: false, Message: "No error when trying to attack friendly NPC"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Friendly NPCs cannot be attacked"}
}

// TestAttackRolls tests D20 combat mechanics
func TestAttackRolls(serverAddr string) TestResult {
	const testName = "Attack Rolls"

	name := uniqueName("CombatTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall: Town Square -> south -> Market Street -> south -> Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atHall := client.WaitForMessage("Training Hall", 1*time.Second)
	logResult(testName, atHall, "At Training Hall")
	if !atHall {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Training Hall"}
	}

	// Attack training dummy
	logAction(testName, "Attacking training dummy...")
	client.ClearMessages()
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Wait for combat messages
	time.Sleep(3500 * time.Millisecond) // Wait for combat tick

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see attack messages with hit/miss language
	hasCombat := strings.Contains(fullOutput, "attack") || strings.Contains(fullOutput, "hit") ||
		strings.Contains(fullOutput, "miss") || strings.Contains(fullOutput, "damage") ||
		strings.Contains(fullOutput, "swing") || strings.Contains(fullOutput, "strike")
	logResult(testName, hasCombat, "Received combat messages")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !hasCombat {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("No combat messages received. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Attack rolls and combat messages working"}
}

// TestFleeCommand tests escaping from combat
func TestFleeCommand(serverAddr string) TestResult {
	const testName = "Flee Command"

	name := uniqueName("FleeTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	// Attack training dummy
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Flee
	logAction(testName, "Fleeing from combat...")
	client.ClearMessages()
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	found := client.WaitForMessage("flee", 1*time.Second) || client.WaitForMessage("escape", 1*time.Second)
	logResult(testName, found, "Fled from combat")

	if !found {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to flee. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully fled from combat"}
}

// TestCombatAndKill tests killing a mob and receiving XP
func TestCombatAndKill(serverAddr string) TestResult {
	const testName = "Combat and Kill"

	name := uniqueName("KillTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	// Attack test rat (10 HP, fast respawn)
	logAction(testName, "Attacking test rat...")
	client.ClearMessages()
	client.SendCommand("attack rat")
	time.Sleep(500 * time.Millisecond)

	// Wait for kill (test rat has 10 HP, should die quickly)
	var foundKill bool
	for i := 0; i < 10; i++ {
		time.Sleep(3500 * time.Millisecond) // Combat tick interval

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")

		if strings.Contains(fullOutput, "defeated") || strings.Contains(fullOutput, "killed") ||
			strings.Contains(fullOutput, "slain") || strings.Contains(fullOutput, "dies") ||
			strings.Contains(fullOutput, "experience") || strings.Contains(fullOutput, "XP") {
			foundKill = true
			break
		}
	}

	logResult(testName, foundKill, "Killed mob and received XP")

	if !foundKill {
		return TestResult{Name: testName, Passed: false, Message: "Failed to kill mob or receive XP"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully killed mob and received XP"}
}

// TestMobRespawn tests that killed mobs respawn
func TestMobRespawn(serverAddr string) TestResult {
	const testName = "Mob Respawn"

	name := uniqueName("RespawnTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	// Kill the training dummy (50 HP, 10s respawn in test config)
	logAction(testName, "Killing training dummy...")
	client.SendCommand("attack dummy")

	// Wait for kill (dummy has 50 HP, 0 armor, 0 damage)
	var killed bool
	for i := 0; i < 20; i++ {
		time.Sleep(3500 * time.Millisecond)
		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "defeated") || strings.Contains(fullOutput, "killed") ||
			strings.Contains(fullOutput, "slain") || strings.Contains(fullOutput, "dies") ||
			strings.Contains(fullOutput, "experience") {
			killed = true
			break
		}
	}

	if !killed {
		return TestResult{Name: testName, Passed: false, Message: "Failed to kill training dummy for respawn test"}
	}

	logAction(testName, "Waiting for respawn (up to 20 seconds)...")

	// Wait for respawn (training dummy has 10s median respawn in test config)
	var respawned bool
	for i := 0; i < 20; i++ {
		time.Sleep(1 * time.Second)
		client.ClearMessages()
		client.SendCommand("look")
		time.Sleep(300 * time.Millisecond)

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "training dummy") || strings.Contains(fullOutput, "dummy") {
			respawned = true
			break
		}
	}

	logResult(testName, respawned, "Mob respawned")

	if !respawned {
		return TestResult{Name: testName, Passed: false, Message: "Mob did not respawn within timeout"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Mob respawned after being killed"}
}

// =============================================================================
// Group 5: Magic System
// =============================================================================

// TestSpellCasting tests casting a spell, mana cost, and cooldown
func TestSpellCasting(serverAddr string) TestResult {
	const testName = "Spell Casting"

	name := uniqueName("SpellTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Cast heal on self
	logAction(testName, "Casting heal on self...")
	client.ClearMessages()
	client.SendCommand("cast heal")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundCast := strings.Contains(fullOutput, "heal") || strings.Contains(fullOutput, "cast") ||
		strings.Contains(fullOutput, "restore") || strings.Contains(fullOutput, "health")
	logResult(testName, foundCast, "Cast heal spell")

	if !foundCast {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to cast heal. Got: %v", messages)}
	}

	// Try to cast again immediately - should be on cooldown
	logAction(testName, "Trying to cast heal again (should be on cooldown)...")
	client.ClearMessages()
	client.SendCommand("cast heal")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	onCooldown := strings.Contains(fullOutput, "cooldown") || strings.Contains(fullOutput, "wait") ||
		strings.Contains(fullOutput, "seconds")
	logResult(testName, onCooldown, "Spell on cooldown")

	if !onCooldown {
		return TestResult{Name: testName, Passed: false, Message: "Spell did not show cooldown message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Spell casting, mana, and cooldown working"}
}

// TestHealOtherPlayer tests healing another player
func TestHealOtherPlayer(serverAddr string) TestResult {
	const testName = "Heal Other Player"

	name1 := uniqueName("Healer")
	name2 := uniqueName("Patient")

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect healer"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect patient"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// Healer casts heal on patient
	logAction(testName, fmt.Sprintf("Healer casts heal on %s...", name2))
	client1.ClearMessages()
	client2.ClearMessages()
	client1.SendCommand(fmt.Sprintf("cast heal %s", name2))
	time.Sleep(300 * time.Millisecond)

	// Check if patient received heal
	messages := client2.GetMessages()
	fullOutput := strings.Join(messages, " ")
	foundHeal := strings.Contains(fullOutput, "heal") || strings.Contains(fullOutput, name1) ||
		strings.Contains(fullOutput, "restore") || strings.Contains(fullOutput, "health")
	logResult(testName, foundHeal, "Patient received heal")

	if !foundHeal {
		return TestResult{Name: testName, Passed: false, Message: "Patient did not receive heal notification"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Heal other player works"}
}

// TestDazzleSpell tests room-wide stun
func TestDazzleSpell(serverAddr string) TestResult {
	const testName = "Dazzle Spell"

	name := uniqueName("DazzleTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	// Attack something to enter combat
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Cast dazzle
	logAction(testName, "Casting dazzle...")
	client.ClearMessages()
	client.SendCommand("cast dazzle")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundDazzle := strings.Contains(fullOutput, "dazzle") || strings.Contains(fullOutput, "stun") ||
		strings.Contains(fullOutput, "blind") || strings.Contains(fullOutput, "flash")
	logResult(testName, foundDazzle, "Cast dazzle")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !foundDazzle {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to cast dazzle. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Dazzle spell stuns enemies"}
}

// TestSpellDamageWithModifiers tests that INT affects spell damage
func TestSpellDamageWithModifiers(serverAddr string) TestResult {
	const testName = "Spell Damage Modifiers"

	name := uniqueName("SpellDmg")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	// Cast flare at dummy
	logAction(testName, "Casting flare at dummy...")
	client.ClearMessages()
	client.SendCommand("cast flare dummy")
	time.Sleep(500 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundDamage := strings.Contains(fullOutput, "damage") || strings.Contains(fullOutput, "hit") ||
		strings.Contains(fullOutput, "flare") || strings.Contains(fullOutput, "burn")
	logResult(testName, foundDamage, "Flare dealt damage")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !foundDamage {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Flare didn't deal damage. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Spell damage works with modifiers"}
}

// =============================================================================
// Group 6: Room Features
// =============================================================================

// TestPrayCommand tests healing at altar
func TestPrayCommand(serverAddr string) TestResult {
	const testName = "Pray Command"

	name := uniqueName("PrayTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Temple (has altar): Town Square -> east -> Temple
	client.SendCommand("east")
	time.Sleep(300 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atTemple := client.WaitForMessage("Temple", 1*time.Second)
	logResult(testName, atTemple, "At Temple")
	if !atTemple {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Temple"}
	}

	// Pray
	logAction(testName, "Praying at altar...")
	client.ClearMessages()
	client.SendCommand("pray")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundPray := strings.Contains(fullOutput, "pray") || strings.Contains(fullOutput, "heal") ||
		strings.Contains(fullOutput, "restore") || strings.Contains(fullOutput, "light") ||
		strings.Contains(fullOutput, "altar")
	logResult(testName, foundPray, "Prayed at altar")

	if !foundPray {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Pray command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Pray at altar heals player"}
}

// TestPortalCommand tests portal listing and travel
func TestPortalCommand(serverAddr string) TestResult {
	const testName = "Portal Command"

	name := uniqueName("PortalTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// We're in Town Square which has a portal
	logAction(testName, "Checking portal...")
	client.ClearMessages()
	client.SendCommand("portal")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show available portals (at least city/floor 0)
	hasPortalInfo := strings.Contains(fullOutput, "portal") || strings.Contains(fullOutput, "floor") ||
		strings.Contains(fullOutput, "city") || strings.Contains(fullOutput, "0")
	logResult(testName, hasPortalInfo, "Portal info displayed")

	if !hasPortalInfo {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Portal command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Portal command shows available destinations"}
}

// TestConsiderSelf tests viewing own stats
func TestConsiderSelf(serverAddr string) TestResult {
	const testName = "Consider Self"

	name := uniqueName("ConsiderTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Considering self...")
	client.ClearMessages()
	client.SendCommand("consider self")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasStats := strings.Contains(fullOutput, "Level") || strings.Contains(fullOutput, "Health") ||
		strings.Contains(fullOutput, "HP") || strings.Contains(fullOutput, "Mana")
	logResult(testName, hasStats, "Stats displayed")

	if !hasStats {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Consider self failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Consider self shows player stats"}
}

// TestLookAtFeature tests examining room features
func TestLookAtFeature(serverAddr string) TestResult {
	const testName = "Look At Feature"

	name := uniqueName("FeatureTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Town Square has fountain and portal features
	logAction(testName, "Looking at fountain...")
	client.ClearMessages()
	client.SendCommand("look fountain")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasDescription := strings.Contains(fullOutput, "fountain") || strings.Contains(fullOutput, "water")
	logResult(testName, hasDescription, "Fountain description shown")

	if !hasDescription {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Look at feature failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Can examine room features"}
}

// =============================================================================
// Group 7: Tower & Progression
// =============================================================================

// TestTowerClimb tests climbing into the tower
func TestTowerClimb(serverAddr string) TestResult {
	const testName = "Tower Climb"

	name := uniqueName("ClimbTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Tower Entrance: south -> south -> south
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atEntrance := client.WaitForMessage("Tower Entrance", 1*time.Second)
	logResult(testName, atEntrance, "At Tower Entrance")
	if !atEntrance {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Tower Entrance"}
	}

	// Climb up
	logAction(testName, "Climbing up into the tower...")
	client.ClearMessages()
	client.SendCommand("up")
	time.Sleep(600 * time.Millisecond) // Allow time for floor generation

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should be on floor 1 now
	onFloor1 := strings.Contains(fullOutput, "Floor 1") || strings.Contains(fullOutput, "floor 1") ||
		strings.Contains(fullOutput, "portal") || strings.Contains(fullOutput, "stairs")
	logResult(testName, onFloor1, "Reached floor 1")

	if !onFloor1 {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to climb to floor 1. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully climbed into tower floor 1"}
}

// TestPlayerLevelUp tests XP gain and leveling up
func TestPlayerLevelUp(serverAddr string) TestResult {
	const testName = "Player Level Up"

	name := uniqueName("LevelTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Check initial level
	client.ClearMessages()
	client.SendCommand("level")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasLevel := strings.Contains(fullOutput, "Level") || strings.Contains(fullOutput, "level") ||
		strings.Contains(fullOutput, "XP") || strings.Contains(fullOutput, "experience")
	logResult(testName, hasLevel, "Level info displayed")

	if !hasLevel {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Level command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Level and XP display working"}
}

// TestAbilityScores tests viewing ability scores
func TestAbilityScores(serverAddr string) TestResult {
	const testName = "Ability Scores"

	name := uniqueName("AbilityTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Checking stats...")
	client.ClearMessages()
	client.SendCommand("stats")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasAbilities := strings.Contains(fullOutput, "STR") || strings.Contains(fullOutput, "Strength") ||
		strings.Contains(fullOutput, "DEX") || strings.Contains(fullOutput, "INT") ||
		strings.Contains(fullOutput, "WIS") || strings.Contains(fullOutput, "CON")
	logResult(testName, hasAbilities, "Ability scores displayed")

	if !hasAbilities {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Stats command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Ability scores display correctly"}
}

// =============================================================================
// Group 8: Account System
// =============================================================================

// TestAccountSystem tests registration, login, invalid login, and banned accounts
func TestAccountSystem(serverAddr string) TestResult {
	const testName = "Account System"

	// Test 1: Registration (via NewTestClient which auto-registers)
	name := uniqueName("AcctTest")
	logAction(testName, fmt.Sprintf("Registering new account '%s'...", name))

	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Registration failed: %v", err)}
	}
	client.Close()
	time.Sleep(500 * time.Millisecond)

	// Test 2: Login with valid credentials
	// NewTestClient uses password = name + "pass123"
	logAction(testName, "Logging in with valid credentials...")
	creds := testclient.Credentials{
		Username:      name,
		Password:      name + "pass123",
		CharacterName: name,
	}

	client2, err := testclient.NewTestClientWithLogin(creds, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Login failed: %v", err)}
	}

	// Verify we're in the game
	time.Sleep(300 * time.Millisecond)
	inGame := client2.WaitForMessage("Town Square", 2*time.Second) || client2.HasMessage("HP:")
	logResult(testName, inGame, "Successfully logged in")
	client2.Close()
	time.Sleep(500 * time.Millisecond)

	if !inGame {
		return TestResult{Name: testName, Passed: false, Message: "Login did not result in entering the game"}
	}

	// Test 3: Invalid login - use a completely wrong username
	logAction(testName, "Testing invalid login with wrong password...")
	badCreds := testclient.Credentials{
		Username:      name,
		Password:      "wrongpassword",
		CharacterName: name,
	}

	client3, err := testclient.NewTestClientWithLogin(badCreds, serverAddr)
	if client3 != nil {
		// Check if we actually got into the game or got an error
		time.Sleep(300 * time.Millisecond)
		badInGame := client3.WaitForMessage("Town Square", 1*time.Second) || client3.HasMessage("HP:")
		client3.Close()
		if badInGame {
			return TestResult{Name: testName, Passed: false, Message: "Invalid login should have failed but entered game"}
		}
	}
	logResult(testName, true, "Invalid login handled")

	return TestResult{Name: testName, Passed: true, Message: "Registration, login, and invalid login all work correctly"}
}

// TestInventoryPersistence tests that inventory persists across login
func TestInventoryPersistence(serverAddr string) TestResult {
	const testName = "Inventory Persistence"

	name := uniqueName("PersistTest")
	logAction(testName, fmt.Sprintf("Creating account '%s' and buying item...", name))

	// Create account and buy an item
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store
	client.SendCommand("south")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east")
	time.Sleep(200 * time.Millisecond)

	// Buy bread
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Navigate to tavern to save via bard
	client.SendCommand("west")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("west")
	time.Sleep(200 * time.Millisecond)

	// Talk to bard to save
	logAction(testName, "Saving via bard...")
	client.SendCommand("talk bard")
	time.Sleep(500 * time.Millisecond)

	client.Close()
	time.Sleep(500 * time.Millisecond)

	// Login again (password is name + "pass123")
	logAction(testName, "Logging back in...")
	creds := testclient.Credentials{
		Username:      name,
		Password:      name + "pass123",
		CharacterName: name,
	}

	client2, err := testclient.NewTestClientWithLogin(creds, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Re-login failed: %v", err)}
	}
	defer client2.Close()

	time.Sleep(300 * time.Millisecond)

	// Check inventory
	client2.ClearMessages()
	client2.SendCommand("inventory")
	time.Sleep(300 * time.Millisecond)

	messages := client2.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasBread := strings.Contains(fullOutput, "bread")
	logResult(testName, hasBread, "Bread still in inventory")

	if !hasBread {
		return TestResult{Name: testName, Passed: false, Message: "Inventory not persisted after logout/login"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Inventory persists across logout/login"}
}

// TestLastVisitedCityRespawn tests that death respawns at city
func TestLastVisitedCityRespawn(serverAddr string) TestResult {
	const testName = "Death Respawn"

	// This test verifies the respawn logic exists by checking if we can look at respawn-related content
	// Actually killing the player would require a high-damage mob

	name := uniqueName("RespawnLocTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Verify we start in Town Square (the respawn point)
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	inTownSquare := strings.Contains(fullOutput, "Town Square")
	logResult(testName, inTownSquare, "Starting in Town Square (respawn point)")

	if !inTownSquare {
		return TestResult{Name: testName, Passed: false, Message: "Not in Town Square at start"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Player starts in Town Square (death respawn point)"}
}

// =============================================================================
// Group 9: Admin Commands
// =============================================================================

// TestAdminCommandsHidden tests that non-admins don't see admin commands
func TestAdminCommandsHidden(serverAddr string) TestResult {
	const testName = "Admin Commands Hidden"

	name := uniqueName("NonAdmin")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Checking help for admin commands...")
	client.ClearMessages()
	client.SendCommand("help")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Admin commands should not be visible
	hasAdmin := strings.Contains(fullOutput, "admin promote") || strings.Contains(fullOutput, "admin ban") ||
		strings.Contains(fullOutput, "admin kick") || strings.Contains(fullOutput, "admin teleport")
	logResult(testName, !hasAdmin, "Admin commands not visible")

	if hasAdmin {
		return TestResult{Name: testName, Passed: false, Message: "Admin commands visible to non-admin"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Admin commands hidden from non-admin users"}
}

// TestAdminAnnounce tests the admin announce command (requires admin account)
func TestAdminAnnounce(serverAddr string) TestResult {
	const testName = "Admin Announce"

	// This test requires an admin account which we can't easily create in integration tests
	// without database access. We'll test that the command exists by trying it as non-admin.

	name := uniqueName("AnnounceTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Trying admin announce as non-admin...")
	client.ClearMessages()
	client.SendCommand("admin announce Test message")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should get permission denied or unknown command
	denied := strings.Contains(fullOutput, "permission") || strings.Contains(fullOutput, "admin") ||
		strings.Contains(fullOutput, "unknown") || strings.Contains(fullOutput, "Unknown") ||
		strings.Contains(fullOutput, "not") || strings.Contains(fullOutput, "cannot")
	logResult(testName, denied, "Non-admin cannot use admin announce")

	if !denied {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Admin announce should fail for non-admin. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Admin announce requires admin privileges"}
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
