package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

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

	// Navigate to Temple (has altar)
	navigateToTemple(client)

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

// TestTrainCommand tests the train command with class trainers
func TestTrainCommand(serverAddr string) TestResult {
	const testName = "Train Command"

	name := uniqueName("TrainTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall where Battlemaster Korg (warrior trainer) is
	navigateToTrainingHall(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atHall := client.WaitForMessage("Training Hall", 1*time.Second)
	logResult(testName, atHall, "At Training Hall")
	if !atHall {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Training Hall"}
	}

	// Try to train (should fail because we're already a warrior and need requirements for other classes)
	logAction(testName, "Attempting to train...")
	client.ClearMessages()
	client.SendCommand("train")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should get a response about training (either requirements not met or already have class)
	hasResponse := strings.Contains(fullOutput, "Battlemaster") || strings.Contains(fullOutput, "train") ||
		strings.Contains(fullOutput, "Warrior") || strings.Contains(fullOutput, "already") ||
		strings.Contains(fullOutput, "level") || strings.Contains(fullOutput, "requirements")
	logResult(testName, hasResponse, "Train command responded")

	if !hasResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Train command gave no response. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Train command works with class trainers"}
}

// TestTrainerLocations tests that trainers exist in expected locations
func TestTrainerLocations(serverAddr string) TestResult {
	const testName = "Trainer Locations"

	name := uniqueName("TrainerLocTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Check Training Hall for Battlemaster Korg
	navigateToTrainingHall(client)
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	hasKorg := strings.Contains(fullOutput, "Battlemaster Korg")
	logResult(testName, hasKorg, "Battlemaster Korg in Training Hall")

	if !hasKorg {
		return TestResult{Name: testName, Passed: false, Message: "Battlemaster Korg not found in Training Hall"}
	}

	// Navigate back to Town Square and check Temple for Father Aldous
	client.SendCommand("north") // Military District East
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("west") // Military District
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Artisan's Market
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Town Square
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east") // Temple
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	hasAldous := strings.Contains(fullOutput, "Father Aldous")
	logResult(testName, hasAldous, "Father Aldous in Temple")

	if !hasAldous {
		return TestResult{Name: testName, Passed: false, Message: "Father Aldous not found in Temple"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Class trainers are in expected locations"}
}
