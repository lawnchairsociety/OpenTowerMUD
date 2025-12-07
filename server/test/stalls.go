package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Group 12: Player Stalls
// =============================================================================

// TestStallOpen tests opening a player stall in the city
func TestStallOpen(serverAddr string) TestResult {
	const testName = "Stall Open"

	name := uniqueName("StallOpen")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Should be in Town Square (floor 0) at start
	logAction(testName, "Opening stall in city...")
	client.ClearMessages()
	client.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see confirmation of stall opening
	stallOpened := strings.Contains(strings.ToLower(fullOutput), "open") &&
		(strings.Contains(strings.ToLower(fullOutput), "stall") || strings.Contains(strings.ToLower(fullOutput), "business"))
	logResult(testName, stallOpened, "Stall opened")

	if !stallOpened {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to open stall. Got: %v", messages)}
	}

	// Close the stall
	client.SendCommand("stall close")
	time.Sleep(200 * time.Millisecond)

	return TestResult{Name: testName, Passed: true, Message: "Successfully opened stall in city"}
}

// TestStallClose tests closing a player stall
func TestStallClose(serverAddr string) TestResult {
	const testName = "Stall Close"

	name := uniqueName("StallClose")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Open stall first
	client.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	// Now close it
	logAction(testName, "Closing stall...")
	client.ClearMessages()
	client.SendCommand("stall close")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	stallClosed := strings.Contains(strings.ToLower(fullOutput), "close") ||
		strings.Contains(strings.ToLower(fullOutput), "closed")
	logResult(testName, stallClosed, "Stall closed")

	if !stallClosed {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to close stall. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully closed stall"}
}

// TestStallAddItem tests adding an item to a player stall
func TestStallAddItem(serverAddr string) TestResult {
	const testName = "Stall Add Item"

	name := uniqueName("StallAdd")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Go to store and buy an item
	navigateToGeneralStore(client)
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Go back to Town Square
	client.SendCommand("west")  // Back to Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Back to Town Square
	time.Sleep(200 * time.Millisecond)

	// Add item to stall
	logAction(testName, "Adding bread to stall for 5 gold...")
	client.ClearMessages()
	client.SendCommand("stall add bread 5")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	itemAdded := strings.Contains(strings.ToLower(fullOutput), "add") ||
		strings.Contains(strings.ToLower(fullOutput), "stall") ||
		strings.Contains(strings.ToLower(fullOutput), "5 gold")
	logResult(testName, itemAdded, "Item added to stall")

	if !itemAdded {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to add item to stall. Got: %v", messages)}
	}

	// Clean up - close stall (returns items)
	client.SendCommand("stall close")
	time.Sleep(200 * time.Millisecond)

	return TestResult{Name: testName, Passed: true, Message: "Successfully added item to stall"}
}

// TestStallRemoveItem tests removing an item from a player stall
func TestStallRemoveItem(serverAddr string) TestResult {
	const testName = "Stall Remove Item"

	name := uniqueName("StallRemove")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Go to store and buy an item
	navigateToGeneralStore(client)
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Go back to Town Square
	client.SendCommand("west")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	// Add item to stall
	client.SendCommand("stall add bread 5")
	time.Sleep(300 * time.Millisecond)

	// Remove item from stall
	logAction(testName, "Removing bread from stall...")
	client.ClearMessages()
	client.SendCommand("stall remove bread")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	itemRemoved := strings.Contains(strings.ToLower(fullOutput), "remove") ||
		strings.Contains(strings.ToLower(fullOutput), "returned") ||
		strings.Contains(strings.ToLower(fullOutput), "inventory")
	logResult(testName, itemRemoved, "Item removed from stall")

	if !itemRemoved {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to remove item from stall. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully removed item from stall"}
}

// TestStallList tests listing items in a player stall
func TestStallList(serverAddr string) TestResult {
	const testName = "Stall List"

	name := uniqueName("StallList")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Go to store and buy items
	navigateToGeneralStore(client)
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Go back to Town Square
	client.SendCommand("west")
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	// Add items to stall
	client.SendCommand("stall add bread 5")
	time.Sleep(300 * time.Millisecond)

	// List stall contents
	logAction(testName, "Listing stall contents...")
	client.ClearMessages()
	client.SendCommand("stall list")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasListing := strings.Contains(strings.ToLower(fullOutput), "bread") ||
		strings.Contains(strings.ToLower(fullOutput), "stall") ||
		strings.Contains(strings.ToLower(fullOutput), "5 gold")
	logResult(testName, hasListing, "Stall listing shown")

	// Clean up
	client.SendCommand("stall close")
	time.Sleep(200 * time.Millisecond)

	if !hasListing {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to list stall. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully listed stall contents"}
}

// TestStallBrowse tests browsing another player's stall
func TestStallBrowse(serverAddr string) TestResult {
	const testName = "Stall Browse"

	// Create seller
	sellerName := uniqueName("Seller")
	seller, err := testclient.NewTestClient(sellerName, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Seller connection failed: %v", err)}
	}
	defer seller.Close()

	time.Sleep(300 * time.Millisecond)

	// Seller buys item and sets up stall
	navigateToGeneralStore(seller)
	seller.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Go back to Town Square
	seller.SendCommand("west")
	time.Sleep(200 * time.Millisecond)
	seller.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	seller.SendCommand("stall add bread 10")
	time.Sleep(300 * time.Millisecond)
	seller.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	// Create buyer
	buyerName := uniqueName("Buyer")
	buyer, err := testclient.NewTestClient(buyerName, serverAddr)
	if err != nil {
		seller.Close()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Buyer connection failed: %v", err)}
	}
	defer buyer.Close()

	time.Sleep(300 * time.Millisecond)

	// Buyer browses seller's stall
	logAction(testName, "Buyer browsing seller's stall...")
	buyer.ClearMessages()
	buyer.SendCommand(fmt.Sprintf("browse %s", sellerName))
	time.Sleep(300 * time.Millisecond)

	messages := buyer.GetMessages()
	fullOutput := strings.Join(messages, " ")

	canBrowse := strings.Contains(strings.ToLower(fullOutput), "bread") ||
		strings.Contains(strings.ToLower(fullOutput), "stall") ||
		strings.Contains(strings.ToLower(fullOutput), "10 gold") ||
		strings.Contains(strings.ToLower(fullOutput), sellerName)
	logResult(testName, canBrowse, "Can browse seller's stall")

	// Clean up
	seller.SendCommand("stall close")
	time.Sleep(200 * time.Millisecond)

	if !canBrowse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to browse stall. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully browsed another player's stall"}
}

// TestStallPurchase tests purchasing an item from another player's stall
func TestStallPurchase(serverAddr string) TestResult {
	const testName = "Stall Purchase"

	// Create seller
	sellerName := uniqueName("Seller")
	seller, err := testclient.NewTestClient(sellerName, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Seller connection failed: %v", err)}
	}
	defer seller.Close()

	time.Sleep(300 * time.Millisecond)

	// Seller buys item and sets up stall
	navigateToGeneralStore(seller)
	seller.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Go back to Town Square
	seller.SendCommand("west")
	time.Sleep(200 * time.Millisecond)
	seller.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	seller.SendCommand("stall add bread 5")
	time.Sleep(300 * time.Millisecond)
	seller.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	// Create buyer
	buyerName := uniqueName("Buyer")
	buyer, err := testclient.NewTestClient(buyerName, serverAddr)
	if err != nil {
		seller.Close()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Buyer connection failed: %v", err)}
	}
	defer buyer.Close()

	time.Sleep(300 * time.Millisecond)

	// Buyer purchases from seller's stall
	logAction(testName, "Buyer purchasing bread from seller...")
	buyer.ClearMessages()
	buyer.SendCommand(fmt.Sprintf("purchase bread from %s", sellerName))
	time.Sleep(300 * time.Millisecond)

	messages := buyer.GetMessages()
	fullOutput := strings.Join(messages, " ")

	purchased := strings.Contains(strings.ToLower(fullOutput), "purchase") ||
		strings.Contains(strings.ToLower(fullOutput), "bought") ||
		strings.Contains(strings.ToLower(fullOutput), "gold")
	logResult(testName, purchased, "Purchased item")

	// Check seller received notification
	sellerMessages := seller.GetMessages()
	sellerOutput := strings.Join(sellerMessages, " ")
	sellerNotified := strings.Contains(strings.ToLower(sellerOutput), "sold") ||
		strings.Contains(strings.ToLower(sellerOutput), "purchase") ||
		strings.Contains(strings.ToLower(sellerOutput), buyerName)
	logResult(testName, sellerNotified, "Seller notified of sale")

	// Clean up
	seller.SendCommand("stall close")
	time.Sleep(200 * time.Millisecond)

	if !purchased {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to purchase from stall. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully purchased item from player stall"}
}

// TestStallCloseOnRoomChange tests that stall closes when player leaves room
func TestStallCloseOnRoomChange(serverAddr string) TestResult {
	const testName = "Stall Close On Room Change"

	name := uniqueName("StallMove")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Open stall in Town Square
	client.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	// Move to another room
	logAction(testName, "Moving to another room with stall open...")
	client.ClearMessages()
	client.SendCommand("south") // Move to Market Street
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	stallClosed := strings.Contains(strings.ToLower(fullOutput), "close") ||
		strings.Contains(strings.ToLower(fullOutput), "stall") ||
		strings.Contains(strings.ToLower(fullOutput), "returned")
	logResult(testName, stallClosed, "Stall closed on room change")

	if !stallClosed {
		// Check if stall is still open
		client.ClearMessages()
		client.SendCommand("stall list")
		time.Sleep(200 * time.Millisecond)
		listMessages := client.GetMessages()
		listOutput := strings.Join(listMessages, " ")
		if strings.Contains(strings.ToLower(listOutput), "not open") || strings.Contains(strings.ToLower(listOutput), "open a stall") {
			return TestResult{Name: testName, Passed: true, Message: "Stall automatically closed on room change"}
		}
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Stall may not have closed on room change. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Stall closed automatically when player moved"}
}

// TestStallRequiresCity tests that stalls can only be opened in the city (floor 0)
func TestStallRequiresCity(serverAddr string) TestResult {
	const testName = "Stall Requires City"

	name := uniqueName("StallCity")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to tower entrance and go up to floor 1
	navigateToTowerEntrance(client)
	client.SendCommand("up")
	time.Sleep(500 * time.Millisecond)

	// Try to open stall outside city
	logAction(testName, "Trying to open stall outside city...")
	client.ClearMessages()
	client.SendCommand("stall open")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	rejected := strings.Contains(strings.ToLower(fullOutput), "city") ||
		strings.Contains(strings.ToLower(fullOutput), "cannot") ||
		strings.Contains(strings.ToLower(fullOutput), "only") ||
		strings.Contains(strings.ToLower(fullOutput), "floor 0")
	logResult(testName, rejected, "Stall rejected outside city")

	if !rejected {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Stall should not open outside city. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Stall correctly requires city location"}
}
