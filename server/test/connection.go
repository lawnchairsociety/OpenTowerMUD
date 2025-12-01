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
