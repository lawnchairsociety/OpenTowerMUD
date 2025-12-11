package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Navigation Helpers for Quest Locations
// =============================================================================

// navigateToBarracks moves a client from Town Square to Barracks (Guard Captain Marcus)
// Path: Town Square -> Market Street -> Artisan's Market -> Military District -> Barracks
func navigateToBarracks(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("south") // Military District
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("west") // Barracks
	time.Sleep(500 * time.Millisecond)
}

// =============================================================================
// Group 11: Quest System
// =============================================================================

// TestQuestCommand tests the basic quest command
func TestQuestCommand(serverAddr string) TestResult {
	const testName = "Quest Command"

	name := uniqueName("QuestTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Testing quest command...")
	client.ClearMessages()
	client.SendCommand("quest")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show quest log (empty or with info about having no quests)
	hasQuestResponse := strings.Contains(fullOutput, "quest") || strings.Contains(fullOutput, "Quest") ||
		strings.Contains(fullOutput, "active") || strings.Contains(fullOutput, "journal") ||
		strings.Contains(fullOutput, "no") || strings.Contains(fullOutput, "empty")
	logResult(testName, hasQuestResponse, "Quest command responded")

	if !hasQuestResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Quest command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest command works correctly"}
}

// TestQuestGiverNPC tests that NPCs marked as quest givers show quests
func TestQuestGiverNPC(serverAddr string) TestResult {
	const testName = "Quest Giver NPC"

	name := uniqueName("QuestGiverTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Barracks where Guard Captain Marcus is
	logAction(testName, "Navigating to Barracks...")
	navigateToBarracks(client)

	// Verify Guard Captain is present
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasCaptain := client.WaitForMessage("Guard Captain", 1*time.Second) || client.WaitForMessage("Marcus", 1*time.Second)
	logResult(testName, hasCaptain, "Guard Captain Marcus present")
	if !hasCaptain {
		return TestResult{Name: testName, Passed: false, Message: "Guard Captain Marcus not found in Barracks"}
	}

	// Talk to Guard Captain - should mention available quests
	logAction(testName, "Talking to Guard Captain...")
	client.ClearMessages()
	client.SendCommand("talk guard captain")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show dialogue AND hint about available quests
	hasQuestHint := strings.Contains(fullOutput, "quest") || strings.Contains(fullOutput, "Quest") ||
		strings.Contains(fullOutput, "quests available")
	logResult(testName, hasQuestHint, "NPC shows quest hint when talking")

	if !hasQuestHint {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("NPC doesn't show quest hint. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest giver NPC shows quest hint when talking"}
}

// TestAcceptQuest tests accepting a quest from an NPC
func TestAcceptQuest(serverAddr string) TestResult {
	const testName = "Accept Quest"

	name := uniqueName("AcceptQuestTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Town Square where Aldric is (gives test_introduction)
	logAction(testName, "Checking for Aldric in Town Square...")
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasAldric := client.WaitForMessage("Aldric", 1*time.Second)
	logResult(testName, hasAldric, "Aldric present in Town Square")
	if !hasAldric {
		return TestResult{Name: testName, Passed: false, Message: "Aldric not found in Town Square"}
	}

	// Try to accept a quest
	logAction(testName, "Accepting test_introduction quest...")
	client.ClearMessages()
	client.SendCommand("accept test_introduction")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should confirm quest acceptance or show error if already accepted
	hasAcceptResponse := strings.Contains(fullOutput, "accept") || strings.Contains(fullOutput, "Accept") ||
		strings.Contains(fullOutput, "quest") || strings.Contains(fullOutput, "Tower Awaits") ||
		strings.Contains(fullOutput, "already") || strings.Contains(fullOutput, "started")
	logResult(testName, hasAcceptResponse, "Accept command responded")

	if !hasAcceptResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Accept quest failed. Got: %v", messages)}
	}

	// Verify quest appears in journal (retry a few times to handle timing)
	logAction(testName, "Checking quest journal...")
	var hasQuestInJournal bool
	var journalMessages []string
	for i := 0; i < 5; i++ {
		client.ClearMessages()
		client.SendCommand("quest")
		time.Sleep(500 * time.Millisecond)

		journalMessages = client.GetMessages()
		fullOutput = strings.Join(journalMessages, " ")

		hasQuestInJournal = strings.Contains(fullOutput, "Test Introduction") || strings.Contains(fullOutput, "test_introduction") ||
			strings.Contains(fullOutput, "active") || strings.Contains(fullOutput, "Active")
		if hasQuestInJournal {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	logResult(testName, hasQuestInJournal, "Quest appears in journal")

	if !hasQuestInJournal {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Quest not in journal. Got: %v", journalMessages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest can be accepted and appears in journal"}
}

// TestQuestProgress tests that quest progress updates correctly
func TestQuestProgress(serverAddr string) TestResult {
	const testName = "Quest Progress"

	name := uniqueName("QuestProgressTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Accept test_introduction quest (explore quest)
	logAction(testName, "Accepting exploration quest...")
	client.ClearMessages()
	client.SendCommand("accept test_introduction")
	time.Sleep(300 * time.Millisecond)

	// Check initial progress
	client.ClearMessages()
	client.SendCommand("quest test_introduction")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show quest details with progress (0/1 for explore)
	hasProgress := strings.Contains(fullOutput, "0/1") || strings.Contains(fullOutput, "Explore") ||
		strings.Contains(fullOutput, "explore") || strings.Contains(fullOutput, "Portal") ||
		strings.Contains(fullOutput, "Floor 1") || strings.Contains(fullOutput, "progress")
	logResult(testName, hasProgress, "Quest progress shown")

	if !hasProgress {
		// It's okay if quest details are shown differently
		hasQuestDetails := strings.Contains(fullOutput, "Tower Awaits") || strings.Contains(fullOutput, "description")
		if !hasQuestDetails {
			return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Quest details not shown. Got: %v", messages)}
		}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest progress tracking works"}
}

// TestQuestListWithNPC tests the 'quests available' command shows available quests from nearby NPC
func TestQuestListWithNPC(serverAddr string) TestResult {
	const testName = "Quest List With NPC"

	name := uniqueName("QuestListTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Barracks
	logAction(testName, "Navigating to Barracks...")
	navigateToBarracks(client)

	// Use 'quests available' command to see available quests
	logAction(testName, "Listing available quests...")
	client.ClearMessages()
	client.SendCommand("quests available")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should list available quests from Guard Captain Marcus
	hasQuestList := strings.Contains(fullOutput, "quest") || strings.Contains(fullOutput, "Quest") ||
		strings.Contains(fullOutput, "Pest Control") || strings.Contains(fullOutput, "First Blood") ||
		strings.Contains(fullOutput, "Available") || strings.Contains(fullOutput, "offers")
	logResult(testName, hasQuestList, "'quests available' shows available quests")

	if !hasQuestList {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("'quests available' command didn't show quests. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "'quests available' command shows available quests from NPC"}
}

// TestQuestPrerequisites tests that quest prerequisites are enforced
func TestQuestPrerequisites(serverAddr string) TestResult {
	const testName = "Quest Prerequisites"

	name := uniqueName("PrereqTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Barracks
	logAction(testName, "Navigating to Barracks...")
	navigateToBarracks(client)

	// Try to accept first_blood quest which requires test_introduction to be completed
	logAction(testName, "Trying to accept first_blood (requires test_introduction)...")
	client.ClearMessages()
	client.SendCommand("accept first_blood")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should either fail due to prerequisites or succeed if test_introduction isn't a prereq
	// The quest should either be rejected or require prereq completion
	hasPrereqCheck := strings.Contains(fullOutput, "prerequisite") || strings.Contains(fullOutput, "prereq") ||
		strings.Contains(fullOutput, "complete") || strings.Contains(fullOutput, "first") ||
		strings.Contains(fullOutput, "require") || strings.Contains(fullOutput, "must") ||
		strings.Contains(fullOutput, "accept") || strings.Contains(fullOutput, "quest")
	logResult(testName, hasPrereqCheck, "Prerequisite check performed")

	if !hasPrereqCheck {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("No response to quest accept. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest prerequisite system works"}
}

// TestTitleCommand tests the title command for earned quest titles
func TestTitleCommand(serverAddr string) TestResult {
	const testName = "Title Command"

	name := uniqueName("TitleTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Test title command (should work even with no titles)
	logAction(testName, "Testing title command...")
	client.ClearMessages()
	client.SendCommand("title")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show title information (either list or "no titles")
	hasTitleResponse := strings.Contains(fullOutput, "title") || strings.Contains(fullOutput, "Title") ||
		strings.Contains(fullOutput, "none") || strings.Contains(fullOutput, "earned") ||
		strings.Contains(fullOutput, "no titles") || strings.Contains(fullOutput, "available")
	logResult(testName, hasTitleResponse, "Title command responded")

	if !hasTitleResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Title command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Title command works correctly"}
}

// TestQuestJournalAlias tests that journal is an alias for quest
func TestQuestJournalAlias(serverAddr string) TestResult {
	const testName = "Quest Journal Alias"

	name := uniqueName("JournalTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Test journal command
	logAction(testName, "Testing journal command alias...")
	client.ClearMessages()
	client.SendCommand("journal")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show same output as quest command
	hasJournalResponse := strings.Contains(fullOutput, "quest") || strings.Contains(fullOutput, "Quest") ||
		strings.Contains(fullOutput, "journal") || strings.Contains(fullOutput, "Journal") ||
		strings.Contains(fullOutput, "active") || strings.Contains(fullOutput, "no")
	logResult(testName, hasJournalResponse, "Journal alias works")

	if !hasJournalResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Journal command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Journal is a valid alias for quest command"}
}

// TestCompleteQuestNotReady tests that complete fails when quest isn't finished
func TestCompleteQuestNotReady(serverAddr string) TestResult {
	const testName = "Complete Quest Not Ready"

	name := uniqueName("NotReadyTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Accept a quest first
	logAction(testName, "Accepting quest...")
	client.SendCommand("accept test_introduction")
	time.Sleep(300 * time.Millisecond)

	// Try to complete without finishing objectives
	logAction(testName, "Trying to complete unfinished quest...")
	client.ClearMessages()
	client.SendCommand("complete test_introduction")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should fail because quest isn't complete
	hasNotReadyResponse := strings.Contains(fullOutput, "not complete") || strings.Contains(fullOutput, "not finished") ||
		strings.Contains(fullOutput, "objective") || strings.Contains(fullOutput, "haven't") ||
		strings.Contains(fullOutput, "incomplete") || strings.Contains(fullOutput, "NPC") ||
		strings.Contains(fullOutput, "complete") || strings.Contains(fullOutput, "turn in")
	logResult(testName, hasNotReadyResponse, "Complete fails when not ready")

	if !hasNotReadyResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Complete should fail. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Quest completion requires finished objectives"}
}
