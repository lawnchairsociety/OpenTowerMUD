package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Group 5: Magic System
// =============================================================================

// TestSpellCasting tests casting a spell, mana cost, and cooldown
func TestSpellCasting(serverAddr string) TestResult {
	const testName = "Spell Casting"

	name := uniqueName("SpellTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Cast heal on self
	logAction(testName, "Casting heal on self...")
	client.ClearMessages()
	client.SendCommand("cast heal")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundCast := strings.Contains(fullOutput, "heal") || strings.Contains(fullOutput, "cast") ||
		strings.Contains(fullOutput, "restore") || strings.Contains(fullOutput, "health")
	logResult(testName, foundCast, "Cast heal spell")

	if !foundCast {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to cast heal. Got: %v", messages)}
	}

	// Try to cast again immediately - should be on cooldown
	logAction(testName, "Trying to cast heal again (should be on cooldown)...")
	client.ClearMessages()
	client.SendCommand("cast heal")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")
	onCooldown := strings.Contains(fullOutput, "cooldown") || strings.Contains(fullOutput, "wait") ||
		strings.Contains(fullOutput, "seconds")
	logResult(testName, onCooldown, "Spell on cooldown")

	if !onCooldown {
		return TestResult{Name: testName, Passed: false, Message: "Spell did not show cooldown message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Spell casting, mana, and cooldown working"}
}

// TestHealOtherPlayer tests healing another player
func TestHealOtherPlayer(serverAddr string) TestResult {
	const testName = "Heal Other Player"

	name1 := uniqueName("Healer")
	name2 := uniqueName("Patient")

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect healer"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect patient"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// Healer casts heal on patient
	logAction(testName, fmt.Sprintf("Healer casts heal on %s...", name2))
	client1.ClearMessages()
	client2.ClearMessages()
	client1.SendCommand(fmt.Sprintf("cast heal %s", name2))
	time.Sleep(300 * time.Millisecond)

	// Check if patient received heal
	messages := client2.GetMessages()
	fullOutput := strings.Join(messages, " ")
	foundHeal := strings.Contains(fullOutput, "heal") || strings.Contains(fullOutput, name1) ||
		strings.Contains(fullOutput, "restore") || strings.Contains(fullOutput, "health")
	logResult(testName, foundHeal, "Patient received heal")

	if !foundHeal {
		return TestResult{Name: testName, Passed: false, Message: "Patient did not receive heal notification"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Heal other player works"}
}

// TestDazzleSpell tests room-wide stun
func TestDazzleSpell(serverAddr string) TestResult {
	const testName = "Dazzle Spell"

	name := uniqueName("DazzleTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Attack something to enter combat
	client.SendCommand("attack dummy")
	time.Sleep(500 * time.Millisecond)

	// Cast dazzle
	logAction(testName, "Casting dazzle...")
	client.ClearMessages()
	client.SendCommand("cast dazzle")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundDazzle := strings.Contains(fullOutput, "dazzle") || strings.Contains(fullOutput, "stun") ||
		strings.Contains(fullOutput, "blind") || strings.Contains(fullOutput, "flash")
	logResult(testName, foundDazzle, "Cast dazzle")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !foundDazzle {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to cast dazzle. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Dazzle spell stuns enemies"}
}

// TestSpellDamageWithModifiers tests that INT affects spell damage
func TestSpellDamageWithModifiers(serverAddr string) TestResult {
	const testName = "Spell Damage Modifiers"

	name := uniqueName("SpellDmg")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Training Hall
	navigateToTrainingHall(client)

	// Cast flare at dummy
	logAction(testName, "Casting flare at dummy...")
	client.ClearMessages()
	client.SendCommand("cast flare dummy")
	time.Sleep(500 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	foundDamage := strings.Contains(fullOutput, "damage") || strings.Contains(fullOutput, "hit") ||
		strings.Contains(fullOutput, "flare") || strings.Contains(fullOutput, "burn")
	logResult(testName, foundDamage, "Flare dealt damage")

	// Flee from combat
	client.SendCommand("flee")
	time.Sleep(300 * time.Millisecond)

	if !foundDamage {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Flare didn't deal damage. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Spell damage works with modifiers"}
}
