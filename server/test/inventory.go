package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

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

	// Navigate to General Store
	logAction(testName, "Navigating to General Store...")
	navigateToGeneralStore(client)

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
	navigateToGeneralStore(client)

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
	navigateToGeneralStore(client)

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
	navigateToGeneralStore(client)

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
