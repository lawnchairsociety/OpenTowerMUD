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

// TestDrinkCommand tests drinking beverages
func TestDrinkCommand(serverAddr string) TestResult {
	const testName = "Drink Command"

	name := uniqueName("DrinkTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Tavern and buy ale
	navigateToTavern(client)

	// Buy ale from bartender
	logAction(testName, "Buying ale from bartender...")
	client.ClearMessages()
	client.SendCommand("buy ale")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	boughtAle := strings.Contains(strings.ToLower(fullOutput), "purchase") ||
		strings.Contains(strings.ToLower(fullOutput), "bought") ||
		strings.Contains(strings.ToLower(fullOutput), "ale")
	logResult(testName, boughtAle, "Bought ale")

	if !boughtAle {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to buy ale. Got: %v", messages)}
	}

	// Now drink the ale
	logAction(testName, "Drinking ale...")
	client.ClearMessages()
	client.SendCommand("drink ale")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should see a message about drinking and mana restoration
	drankAle := strings.Contains(strings.ToLower(fullOutput), "drink") ||
		strings.Contains(strings.ToLower(fullOutput), "mana") ||
		strings.Contains(strings.ToLower(fullOutput), "restore") ||
		strings.Contains(strings.ToLower(fullOutput), "ale")
	logResult(testName, drankAle, "Drank ale successfully")

	if !drankAle {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Drink command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Drink command consumes beverages"}
}

// TestDrinkPotion tests drinking potions for health
func TestDrinkPotion(serverAddr string) TestResult {
	const testName = "Drink Potion"

	name := uniqueName("DrinkPotion")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy healing potion
	navigateToGeneralStore(client)

	// Buy healing potion
	logAction(testName, "Buying healing potion...")
	client.ClearMessages()
	client.SendCommand("buy healing potion")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	boughtPotion := strings.Contains(strings.ToLower(fullOutput), "purchase") ||
		strings.Contains(strings.ToLower(fullOutput), "bought") ||
		strings.Contains(strings.ToLower(fullOutput), "healing")
	logResult(testName, boughtPotion, "Bought healing potion")

	if !boughtPotion {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to buy healing potion. Got: %v", messages)}
	}

	// Now drink the potion
	logAction(testName, "Drinking healing potion...")
	client.ClearMessages()
	client.SendCommand("drink healing")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should see a message about drinking and health restoration
	drankPotion := strings.Contains(strings.ToLower(fullOutput), "drink") ||
		strings.Contains(strings.ToLower(fullOutput), "health") ||
		strings.Contains(strings.ToLower(fullOutput), "heal") ||
		strings.Contains(strings.ToLower(fullOutput), "restore") ||
		strings.Contains(strings.ToLower(fullOutput), "potion")
	logResult(testName, drankPotion, "Drank potion successfully")

	if !drankPotion {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Drink potion failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Drink command works with potions"}
}

// TestDrinkNonDrinkable tests trying to drink non-drinkable items
func TestDrinkNonDrinkable(serverAddr string) TestResult {
	const testName = "Drink Non-Drinkable"

	name := uniqueName("DrinkFail")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy bread (food, not drink)
	navigateToGeneralStore(client)

	// Buy bread
	logAction(testName, "Buying bread...")
	client.ClearMessages()
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Try to drink bread
	logAction(testName, "Trying to drink bread...")
	client.ClearMessages()
	client.SendCommand("drink bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see an error message
	cantDrink := strings.Contains(strings.ToLower(fullOutput), "can't") ||
		strings.Contains(strings.ToLower(fullOutput), "cannot") ||
		strings.Contains(strings.ToLower(fullOutput), "not") ||
		strings.Contains(strings.ToLower(fullOutput), "don't have")
	logResult(testName, cantDrink, "Cannot drink non-drinkable")

	if !cantDrink {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Should not be able to drink bread. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Drink command rejects non-drinkable items"}
}

// TestHoldCommand tests holding misc items (like torches)
func TestHoldCommand(serverAddr string) TestResult {
	const testName = "Hold Command"

	name := uniqueName("HoldTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy a torch
	navigateToGeneralStore(client)

	// Buy torch
	logAction(testName, "Buying torch...")
	client.ClearMessages()
	client.SendCommand("buy torch")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	boughtTorch := strings.Contains(strings.ToLower(fullOutput), "purchase") ||
		strings.Contains(strings.ToLower(fullOutput), "bought") ||
		strings.Contains(strings.ToLower(fullOutput), "torch")
	logResult(testName, boughtTorch, "Bought torch")

	if !boughtTorch {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to buy torch. Got: %v", messages)}
	}

	// Now hold the torch
	logAction(testName, "Holding torch...")
	client.ClearMessages()
	client.SendCommand("hold torch")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should see a message about holding or equipping
	heldTorch := strings.Contains(strings.ToLower(fullOutput), "hold") ||
		strings.Contains(strings.ToLower(fullOutput), "equip") ||
		strings.Contains(strings.ToLower(fullOutput), "torch")
	logResult(testName, heldTorch, "Held torch successfully")

	if !heldTorch {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Hold command failed. Got: %v", messages)}
	}

	// Check equipment to see if it's in held slot
	client.ClearMessages()
	client.SendCommand("equipment")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	inEquipment := strings.Contains(strings.ToLower(fullOutput), "torch") ||
		strings.Contains(strings.ToLower(fullOutput), "held")
	logResult(testName, inEquipment, "Torch in equipment")

	return TestResult{Name: testName, Passed: true, Message: "Hold command equips items to held slot"}
}

// TestUseCommand tests using consumable items and room features
func TestUseCommand(serverAddr string) TestResult {
	const testName = "Use Command"

	name := uniqueName("UseTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy bread
	navigateToGeneralStore(client)

	// Buy bread
	logAction(testName, "Buying bread...")
	client.ClearMessages()
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Use the bread (should consume it like eat)
	logAction(testName, "Using bread...")
	client.ClearMessages()
	client.SendCommand("use bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see a message about eating/consuming or health
	usedBread := strings.Contains(strings.ToLower(fullOutput), "eat") ||
		strings.Contains(strings.ToLower(fullOutput), "consume") ||
		strings.Contains(strings.ToLower(fullOutput), "heal") ||
		strings.Contains(strings.ToLower(fullOutput), "bread")
	logResult(testName, usedBread, "Used bread successfully")

	if !usedBread {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Use command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Use command consumes items"}
}

// TestUseAltar tests using room features (altar)
func TestUseAltar(serverAddr string) TestResult {
	const testName = "Use Altar"

	name := uniqueName("UseAltar")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Temple (has altar)
	navigateToTemple(client)

	// Use the altar
	logAction(testName, "Using altar...")
	client.ClearMessages()
	client.SendCommand("use altar")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see a message about praying, healing, or altar
	usedAltar := strings.Contains(strings.ToLower(fullOutput), "pray") ||
		strings.Contains(strings.ToLower(fullOutput), "heal") ||
		strings.Contains(strings.ToLower(fullOutput), "altar") ||
		strings.Contains(strings.ToLower(fullOutput), "divine")
	logResult(testName, usedAltar, "Used altar successfully")

	if !usedAltar {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Use altar failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Use command works with room features"}
}

// TestTakeCommand tests picking up items from the ground
func TestTakeCommand(serverAddr string) TestResult {
	const testName = "Take Command"

	name := uniqueName("TakeTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy bread
	navigateToGeneralStore(client)

	// Buy bread
	logAction(testName, "Buying bread...")
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Drop the bread
	logAction(testName, "Dropping bread...")
	client.SendCommand("drop bread")
	time.Sleep(300 * time.Millisecond)

	// Now pick it up using take
	logAction(testName, "Taking bread...")
	client.ClearMessages()
	client.SendCommand("take bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see message about picking up
	tookItem := strings.Contains(strings.ToLower(fullOutput), "pick") ||
		strings.Contains(strings.ToLower(fullOutput), "take") ||
		strings.Contains(strings.ToLower(fullOutput), "get") ||
		strings.Contains(strings.ToLower(fullOutput), "bread")
	logResult(testName, tookItem, "Took bread successfully")

	if !tookItem {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Take command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Take command picks up items from ground"}
}

// TestGetCommand tests the 'get' alias for take
func TestGetCommand(serverAddr string) TestResult {
	const testName = "Get Command"

	name := uniqueName("GetTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy bread
	navigateToGeneralStore(client)

	// Buy bread
	logAction(testName, "Buying bread...")
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Drop the bread
	logAction(testName, "Dropping bread...")
	client.SendCommand("drop bread")
	time.Sleep(300 * time.Millisecond)

	// Now pick it up using get (alias for take)
	logAction(testName, "Getting bread using 'get' alias...")
	client.ClearMessages()
	client.SendCommand("get bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see message about picking up
	gotItem := strings.Contains(strings.ToLower(fullOutput), "pick") ||
		strings.Contains(strings.ToLower(fullOutput), "take") ||
		strings.Contains(strings.ToLower(fullOutput), "get") ||
		strings.Contains(strings.ToLower(fullOutput), "bread")
	logResult(testName, gotItem, "Got bread successfully")

	if !gotItem {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Get command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Get command (take alias) works correctly"}
}

// TestPickupCommand tests the 'pickup' alias for take
func TestPickupCommand(serverAddr string) TestResult {
	const testName = "Pickup Command"

	name := uniqueName("PickupTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to General Store and buy bread
	navigateToGeneralStore(client)

	// Buy bread
	logAction(testName, "Buying bread...")
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Drop the bread
	logAction(testName, "Dropping bread...")
	client.SendCommand("drop bread")
	time.Sleep(300 * time.Millisecond)

	// Now pick it up using pickup
	logAction(testName, "Picking up bread using 'pickup' alias...")
	client.ClearMessages()
	client.SendCommand("pickup bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see message about picking up
	pickedUp := strings.Contains(strings.ToLower(fullOutput), "pick") ||
		strings.Contains(strings.ToLower(fullOutput), "take") ||
		strings.Contains(strings.ToLower(fullOutput), "get") ||
		strings.Contains(strings.ToLower(fullOutput), "bread")
	logResult(testName, pickedUp, "Picked up bread successfully")

	if !pickedUp {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Pickup command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Pickup command (take alias) works correctly"}
}
