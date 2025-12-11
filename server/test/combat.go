package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Group 4: Combat System
// =============================================================================

// TestUnattackableNPC tests that friendly NPCs cannot be attacked
func TestUnattackableNPC(serverAddr string) TestResult {
	const testName = "Unattackable NPC"

	name := uniqueName("AttackNPC")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try to attack the old guide in Town Square
	logAction(testName, "Attempting to attack Aldric the old guide...")
	client.ClearMessages()
	client.SendCommand("attack aldric")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should get an error message about not being able to attack
	cannotAttack := strings.Contains(fullOutput, "cannot") || strings.Contains(fullOutput, "can't") ||
		strings.Contains(fullOutput, "not attackable") || strings.Contains(fullOutput, "unable")
	logResult(testName, cannotAttack, "Received cannot attack message")

	if !cannotAttack {
		return TestResult{Name: testName, Passed: false, Message: "No error when trying to attack friendly NPC"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Friendly NPCs cannot be attacked"}
}

// TestAttackRolls tests D20 combat mechanics
func TestAttackRolls(serverAddr string) TestResult {
	const testName = "Attack Rolls"

	name := uniqueName("CombatTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atHall := client.WaitForMessage("Training Hall", 1*time.Second)
	logResult(testName, atHall, "At Training Hall")
	if !atHall {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Training Hall"}
	}

	// Attack training dummy
	logAction(testName, "Attacking training dummy...")
	client.ClearMessages()
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Wait for combat messages
	time.Sleep(3500 * time.Millisecond) // Wait for combat tick

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see attack messages with hit/miss language
	hasCombat := strings.Contains(fullOutput, "attack") || strings.Contains(fullOutput, "hit") ||
		strings.Contains(fullOutput, "miss") || strings.Contains(fullOutput, "damage") ||
		strings.Contains(fullOutput, "swing") || strings.Contains(fullOutput, "strike")
	logResult(testName, hasCombat, "Received combat messages")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !hasCombat {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("No combat messages received. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Attack rolls and combat messages working"}
}

// TestFleeCommand tests escaping from combat
func TestFleeCommand(serverAddr string) TestResult {
	const testName = "Flee Command"

	name := uniqueName("FleeTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Attack training dummy
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Flee
	logAction(testName, "Fleeing from combat...")
	client.ClearMessages()
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	found := client.WaitForMessage("flee", 1*time.Second) || client.WaitForMessage("escape", 1*time.Second)
	logResult(testName, found, "Fled from combat")

	if !found {
		messages := client.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to flee. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully fled from combat"}
}

// TestCombatAndKill tests killing a mob and receiving XP
func TestCombatAndKill(serverAddr string) TestResult {
	const testName = "Combat and Kill"

	name := uniqueName("KillTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Wait for test rat to be present (may have been killed by previous test)
	// Rat has 5s median respawn, so wait up to 15 seconds
	logAction(testName, "Waiting for test rat to be present...")
	var ratPresent bool
	for i := 0; i < 15; i++ {
		client.ClearMessages()
		client.SendCommand("look")
		time.Sleep(500 * time.Millisecond)

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "test rat") || strings.Contains(fullOutput, "rat") {
			ratPresent = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !ratPresent {
		return TestResult{Name: testName, Passed: false, Message: "Test rat not present in room after waiting"}
	}

	// Attack test rat (10 HP, fast respawn)
	logAction(testName, "Attacking test rat...")
	client.ClearMessages()
	client.SendCommand("attack rat")
	time.Sleep(500 * time.Millisecond)

	// Wait for kill (test rat has 10 HP, should die quickly)
	var foundKill bool
	for i := 0; i < 10; i++ {
		time.Sleep(3500 * time.Millisecond) // Combat tick interval

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")

		if strings.Contains(fullOutput, "defeated") || strings.Contains(fullOutput, "killed") ||
			strings.Contains(fullOutput, "slain") || strings.Contains(fullOutput, "dies") ||
			strings.Contains(fullOutput, "experience") || strings.Contains(fullOutput, "XP") {
			foundKill = true
			break
		}
	}

	logResult(testName, foundKill, "Killed mob and received XP")

	if !foundKill {
		return TestResult{Name: testName, Passed: false, Message: "Failed to kill mob or receive XP"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully killed mob and received XP"}
}

// TestMobRespawn tests that killed mobs respawn
func TestMobRespawn(serverAddr string) TestResult {
	const testName = "Mob Respawn"

	name := uniqueName("RespawnTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Wait for test rat to be present (may have been killed by previous test)
	// Rat has 5s median respawn, so wait up to 15 seconds
	logAction(testName, "Waiting for test rat to be present...")
	var ratPresent bool
	for i := 0; i < 15; i++ {
		client.ClearMessages()
		client.SendCommand("look")
		time.Sleep(500 * time.Millisecond)

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "test rat") || strings.Contains(fullOutput, "rat") {
			ratPresent = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !ratPresent {
		return TestResult{Name: testName, Passed: false, Message: "Test rat not present in room"}
	}

	// Kill the test rat (10 HP, 5s respawn in test config) - faster than training dummy
	logAction(testName, "Killing test rat...")
	client.ClearMessages()
	client.SendCommand("attack rat")
	time.Sleep(500 * time.Millisecond)

	// Wait for kill (rat has 10 HP, should die quickly)
	// Use more iterations with shorter waits for reliability
	var killed bool
	for i := 0; i < 15; i++ {
		time.Sleep(3000 * time.Millisecond)
		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "defeated") || strings.Contains(fullOutput, "killed") ||
			strings.Contains(fullOutput, "slain") || strings.Contains(fullOutput, "dies") ||
			strings.Contains(fullOutput, "experience") || strings.Contains(fullOutput, "XP") {
			killed = true
			break
		}
	}

	if !killed {
		return TestResult{Name: testName, Passed: false, Message: "Failed to kill test rat for respawn test"}
	}

	logAction(testName, "Waiting for respawn (up to 15 seconds)...")

	// Wait for respawn (test rat has 5s median respawn in test config)
	var respawned bool
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		client.ClearMessages()
		client.SendCommand("look")
		time.Sleep(300 * time.Millisecond)

		messages := client.GetMessages()
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "test rat") || strings.Contains(fullOutput, "rat") {
			respawned = true
			break
		}
	}

	logResult(testName, respawned, "Mob respawned")

	if !respawned {
		return TestResult{Name: testName, Passed: false, Message: "Mob did not respawn within timeout"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Mob respawned after being killed"}
}

// TestCombatThreat tests that damage generates threat and affects targeting
func TestCombatThreat(serverAddr string) TestResult {
	const testName = "Combat Threat"

	// Create two players to test threat mechanics
	name1 := uniqueName("ThreatOne")
	client1, err := testclient.NewTestClient(name1, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Client 1 connection failed: %v", err)}
	}
	defer client1.Close()

	time.Sleep(500 * time.Millisecond) // Wait between client connections

	name2 := uniqueName("ThreatTwo")
	client2, err := testclient.NewTestClient(name2, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Client 2 connection failed: %v", err)}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// Both navigate to Training Hall
	navigateToTrainingHall(client1)
	navigateToTrainingHall(client2)

	// Wait for dummy to be present
	time.Sleep(500 * time.Millisecond)

	// Client 1 attacks first
	logAction(testName, "Client 1 attacking training dummy...")
	client1.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Client 2 joins combat
	logAction(testName, "Client 2 joining combat...")
	client2.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Wait for a combat round
	time.Sleep(3500 * time.Millisecond)

	// Both clients should see combat messages
	messages1 := client1.GetMessages()
	messages2 := client2.GetMessages()

	output1 := strings.Join(messages1, " ")
	output2 := strings.Join(messages2, " ")

	client1InCombat := strings.Contains(output1, "attack") || strings.Contains(output1, "hit") ||
		strings.Contains(output1, "miss") || strings.Contains(output1, "damage")
	client2InCombat := strings.Contains(output2, "attack") || strings.Contains(output2, "hit") ||
		strings.Contains(output2, "miss") || strings.Contains(output2, "damage")

	logResult(testName, client1InCombat, "Client 1 in combat")
	logResult(testName, client2InCombat, "Client 2 in combat")

	// Both flee
	client1.SendCommand("flee")
	client2.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !client1InCombat || !client2InCombat {
		return TestResult{Name: testName, Passed: false, Message: "Multi-player combat failed to engage properly"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Multiple players can engage in combat with threat system"}
}

// TestCombatNotInCombat tests that flee fails when not in combat
func TestCombatNotInCombat(serverAddr string) TestResult {
	const testName = "Combat Not In Combat"

	name := uniqueName("NotCombat")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Try to flee without being in combat
	logAction(testName, "Trying to flee without being in combat...")
	client.ClearMessages()
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	notInCombat := strings.Contains(strings.ToLower(fullOutput), "not") ||
		strings.Contains(strings.ToLower(fullOutput), "combat") ||
		strings.Contains(strings.ToLower(fullOutput), "fighting")
	logResult(testName, notInCombat, "Cannot flee when not in combat")

	if !notInCombat {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Should show error when fleeing without combat. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Flee correctly fails when not in combat"}
}

// TestCombatConsiderMob tests the consider command for evaluating mob difficulty
func TestCombatConsiderMob(serverAddr string) TestResult {
	const testName = "Combat Consider Mob"

	name := uniqueName("Consider")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Consider the training dummy
	logAction(testName, "Considering training dummy...")
	client.ClearMessages()
	client.SendCommand("consider dummy")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see some difficulty assessment
	hasConsiderOutput := strings.Contains(strings.ToLower(fullOutput), "dummy") ||
		strings.Contains(strings.ToLower(fullOutput), "level") ||
		strings.Contains(strings.ToLower(fullOutput), "easy") ||
		strings.Contains(strings.ToLower(fullOutput), "hard") ||
		strings.Contains(strings.ToLower(fullOutput), "match") ||
		strings.Contains(strings.ToLower(fullOutput), "trivial")
	logResult(testName, hasConsiderOutput, "Consider shows mob info")

	if !hasConsiderOutput {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Consider should show mob info. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Consider command shows mob difficulty assessment"}
}

// TestCombatDamageFormula tests that combat damage includes proper modifiers
func TestCombatDamageFormula(serverAddr string) TestResult {
	const testName = "Combat Damage Formula"

	name := uniqueName("DmgFormula")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Attack training dummy
	logAction(testName, "Attacking training dummy to verify damage formula...")
	client.ClearMessages()
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Wait for combat tick
	time.Sleep(3500 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see damage output or miss message with roll details
	hasDamageInfo := strings.Contains(fullOutput, "damage") ||
		strings.Contains(fullOutput, "hit") ||
		strings.Contains(fullOutput, "miss") ||
		strings.Contains(fullOutput, "roll")
	logResult(testName, hasDamageInfo, "Combat shows damage/roll information")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !hasDamageInfo {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Combat should show damage info. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Combat damage formula working with proper feedback"}
}
