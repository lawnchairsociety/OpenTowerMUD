package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

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

// TestExitsCommand tests the exits command showing available directions
func TestExitsCommand(serverAddr string) TestResult {
	const testName = "Exits Command"

	name := uniqueName("ExitsTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// We start in Town Square - check exits
	logAction(testName, "Checking exits command...")
	client.ClearMessages()
	client.SendCommand("exits")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Town Square has south (Market Street) and east (Temple) exits
	hasExitsHeader := strings.Contains(strings.ToLower(fullOutput), "exit")
	hasDirections := strings.Contains(strings.ToLower(fullOutput), "south") ||
		strings.Contains(strings.ToLower(fullOutput), "east") ||
		strings.Contains(strings.ToLower(fullOutput), "north") ||
		strings.Contains(strings.ToLower(fullOutput), "west")
	logResult(testName, hasExitsHeader, "Exits header shown")
	logResult(testName, hasDirections, "Directions displayed")

	if !hasExitsHeader {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Exits command failed. Got: %v", messages)}
	}
	if !hasDirections {
		return TestResult{Name: testName, Passed: false, Message: "No directions shown in exits"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Exits command shows available directions"}
}

// TestUnlockCommand tests unlocking doors (handles no locked door gracefully)
func TestUnlockCommand(serverAddr string) TestResult {
	const testName = "Unlock Command"

	name := uniqueName("UnlockTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Buy a treasure key from the general store
	navigateToGeneralStore(client)

	logAction(testName, "Buying treasure key...")
	client.ClearMessages()
	client.SendCommand("buy treasure key")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	boughtKey := strings.Contains(strings.ToLower(fullOutput), "key") ||
		strings.Contains(strings.ToLower(fullOutput), "purchase") ||
		strings.Contains(strings.ToLower(fullOutput), "bought")
	logResult(testName, boughtKey, "Bought treasure key")

	if !boughtKey {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to buy key. Got: %v", messages)}
	}

	// Try to unlock without a locked door (should fail gracefully)
	logAction(testName, "Testing unlock without locked door...")
	client.ClearMessages()
	client.SendCommand("unlock north")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should see error about no locked door or no exit
	noLockedDoor := strings.Contains(strings.ToLower(fullOutput), "not locked") ||
		strings.Contains(strings.ToLower(fullOutput), "no exit") ||
		strings.Contains(strings.ToLower(fullOutput), "no door") ||
		strings.Contains(strings.ToLower(fullOutput), "locked") ||
		strings.Contains(strings.ToLower(fullOutput), "exit")
	logResult(testName, noLockedDoor, "Correct response for no locked door")

	if !noLockedDoor {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Unlock should handle missing/unlocked doors. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Unlock command handles non-locked doors correctly"}
}

// TestUnlockNoArgs tests unlock command without direction argument
func TestUnlockNoArgs(serverAddr string) TestResult {
	const testName = "Unlock No Args"

	name := uniqueName("UnlockArgs")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try unlock without direction
	logAction(testName, "Testing unlock without args...")
	client.ClearMessages()
	client.SendCommand("unlock")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see usage message
	hasUsage := strings.Contains(strings.ToLower(fullOutput), "usage") ||
		strings.Contains(strings.ToLower(fullOutput), "direction") ||
		strings.Contains(strings.ToLower(fullOutput), "unlock what")
	logResult(testName, hasUsage, "Usage message shown")

	if !hasUsage {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Unlock should show usage. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Unlock command shows usage when missing direction"}
}

// TestMovementAliases tests single-letter movement commands (n, s, e, w)
func TestMovementAliases(serverAddr string) TestResult {
	const testName = "Movement Aliases"

	name := uniqueName("MoveAlias")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// We start in Town Square - try using 's' instead of 'south'
	logAction(testName, "Moving south using 's' alias...")
	client.ClearMessages()
	client.SendCommand("s")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	movedSouth := strings.Contains(fullOutput, "Market Street")
	logResult(testName, movedSouth, "Moved south with 's'")

	if !movedSouth {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("'s' command failed to move south. Got: %v", messages)}
	}

	// Now try 'n' to go back
	logAction(testName, "Moving north using 'n' alias...")
	client.ClearMessages()
	client.SendCommand("n")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	movedNorth := strings.Contains(fullOutput, "Town Square")
	logResult(testName, movedNorth, "Moved north with 'n'")

	if !movedNorth {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("'n' command failed to move north. Got: %v", messages)}
	}

	// Now try 'e' to go to Temple
	logAction(testName, "Moving east using 'e' alias...")
	client.ClearMessages()
	client.SendCommand("e")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	movedEast := strings.Contains(fullOutput, "Temple")
	logResult(testName, movedEast, "Moved east with 'e'")

	if !movedEast {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("'e' command failed to move east. Got: %v", messages)}
	}

	// Now try 'w' to go back
	logAction(testName, "Moving west using 'w' alias...")
	client.ClearMessages()
	client.SendCommand("w")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	movedWest := strings.Contains(fullOutput, "Town Square")
	logResult(testName, movedWest, "Moved west with 'w'")

	if !movedWest {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("'w' command failed to move west. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Movement aliases (n, s, e, w) work correctly"}
}
