package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

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

	// Navigate to Tower Entrance
	navigateToTowerEntrance(client)

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

// TestScoreCommand tests the score command
func TestScoreCommand(serverAddr string) TestResult {
	const testName = "Score Command"

	name := uniqueName("ScoreTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Checking score...")
	client.ClearMessages()
	client.SendCommand("score")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasScore := strings.Contains(fullOutput, "Level") || strings.Contains(fullOutput, "HP") ||
		strings.Contains(fullOutput, "Gold") || strings.Contains(fullOutput, "XP")
	logResult(testName, hasScore, "Score displayed")

	if !hasScore {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Score command failed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Score command displays player summary"}
}
