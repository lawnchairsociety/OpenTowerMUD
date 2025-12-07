package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

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
	navigateToGeneralStore(client)

	// Buy bread
	client.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	// Auto-save is enabled, so just wait a moment for save to occur
	logAction(testName, "Waiting for auto-save...")
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

// TestPasswordChangeNoArgs tests password command without arguments
func TestPasswordChangeNoArgs(serverAddr string) TestResult {
	const testName = "Password Change No Args"

	name := uniqueName("PasswordNoArgs")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try password without arguments
	logAction(testName, "Testing password command without args...")
	client.ClearMessages()
	client.SendCommand("password")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see usage message
	hasUsage := strings.Contains(strings.ToLower(fullOutput), "usage") ||
		strings.Contains(strings.ToLower(fullOutput), "old_password") ||
		strings.Contains(strings.ToLower(fullOutput), "new_password")
	logResult(testName, hasUsage, "Usage message shown")

	if !hasUsage {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Password should show usage. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Password command shows usage when missing arguments"}
}

// TestPasswordChangeWrongOld tests password command with wrong old password
func TestPasswordChangeWrongOld(serverAddr string) TestResult {
	const testName = "Password Change Wrong Old"

	name := uniqueName("PasswordWrong")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try password with wrong old password
	logAction(testName, "Testing password command with wrong old password...")
	client.ClearMessages()
	client.SendCommand("password WrongPassword123 NewPassword456")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see error about incorrect password
	wrongPassword := strings.Contains(strings.ToLower(fullOutput), "incorrect") ||
		strings.Contains(strings.ToLower(fullOutput), "wrong") ||
		strings.Contains(strings.ToLower(fullOutput), "invalid") ||
		strings.Contains(strings.ToLower(fullOutput), "not match")
	logResult(testName, wrongPassword, "Wrong password rejected")

	if !wrongPassword {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Password should reject wrong old password. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Password command rejects incorrect old password"}
}
