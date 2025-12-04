package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// TestRoomPersistence tests that player room persists across logout/login
func TestRoomPersistence(serverAddr string) TestResult {
	const testName = "Room Persistence"

	name := uniqueName("RoomPersist")
	logAction(testName, fmt.Sprintf("Creating account '%s'...", name))

	// Create account
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}

	time.Sleep(300 * time.Millisecond)

	// Verify we start in Town Square
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	if !strings.Contains(fullOutput, "Town Square") {
		client.Close()
		return TestResult{Name: testName, Passed: false, Message: "Not starting in Town Square"}
	}
	logResult(testName, true, "Starting in Town Square")

	// Navigate to General Store (south, east from Town Square)
	logAction(testName, "Navigating to General Store...")
	navigateToGeneralStore(client)

	// Verify we're in General Store
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	if !strings.Contains(fullOutput, "General Store") {
		client.Close()
		return TestResult{Name: testName, Passed: false, Message: "Failed to navigate to General Store"}
	}
	logResult(testName, true, "In General Store")

	// Wait for auto-save
	logAction(testName, "Waiting for auto-save...")
	time.Sleep(500 * time.Millisecond)

	// Disconnect
	client.Close()
	time.Sleep(500 * time.Millisecond)

	// Login again
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

	// Check our location
	client2.ClearMessages()
	client2.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages = client2.GetMessages()
	fullOutput = strings.Join(messages, " ")

	inGeneralStore := strings.Contains(fullOutput, "General Store")
	logResult(testName, inGeneralStore, "Player returned to General Store")

	if !inGeneralStore {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Room not persisted - expected General Store, got: %s", fullOutput[:min(len(fullOutput), 100)])}
	}

	return TestResult{Name: testName, Passed: true, Message: "Player room persists across logout/login"}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
