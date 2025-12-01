package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Group 2: Communication
// =============================================================================

// TestSayCommand tests the say command broadcasts to room
func TestSayCommand(serverAddr string) TestResult {
	const testName = "Say Command"

	name1 := uniqueName("Speaker")
	name2 := uniqueName("Listener")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", name1, name2))

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect speaker"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect listener"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()

	logAction(testName, "Speaker says: Hello everyone!")
	client1.SendCommand("say Hello everyone!")
	time.Sleep(300 * time.Millisecond)

	foundMessage := client2.WaitForMessage("Hello everyone", 1*time.Second)
	logResult(testName, foundMessage, "Listener received message")

	if !foundMessage {
		return TestResult{Name: testName, Passed: false, Message: "Listener did not receive say message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Say command successfully broadcast to room"}
}

// TestTellCommand tests private messaging
func TestTellCommand(serverAddr string) TestResult {
	const testName = "Tell Command"

	aliceName := uniqueName("Alice")
	bobName := uniqueName("Bob")
	charlieName := uniqueName("Charlie")
	logAction(testName, fmt.Sprintf("Connecting %s, %s, %s...", aliceName, bobName, charlieName))

	alice, err1 := testclient.NewTestClient(aliceName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Alice"}
	}
	defer alice.Close()

	bob, err2 := testclient.NewTestClient(bobName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Bob"}
	}
	defer bob.Close()

	charlie, err3 := testclient.NewTestClient(charlieName, serverAddr)
	if err3 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect Charlie"}
	}
	defer charlie.Close()

	time.Sleep(500 * time.Millisecond)

	// Move Bob to different room
	bob.SendCommand("north")
	time.Sleep(200 * time.Millisecond)

	alice.ClearMessages()
	bob.ClearMessages()
	charlie.ClearMessages()

	logAction(testName, fmt.Sprintf("Alice tells %s: Secret message!", bobName))
	alice.SendCommand(fmt.Sprintf("tell %s Secret message!", bobName))
	time.Sleep(300 * time.Millisecond)

	foundBob := bob.WaitForMessage("Secret message", 1*time.Second)
	logResult(testName, foundBob, "Bob received tell")

	if !foundBob {
		return TestResult{Name: testName, Passed: false, Message: "Bob did not receive tell from Alice"}
	}

	// Charlie should NOT receive the message
	messages := charlie.GetMessages()
	charlieReceivedSecret := false
	for _, msg := range messages {
		if strings.Contains(msg, "Secret message") {
			charlieReceivedSecret = true
			break
		}
	}
	logResult(testName, !charlieReceivedSecret, "Charlie did NOT receive private message")

	if charlieReceivedSecret {
		return TestResult{Name: testName, Passed: false, Message: "Charlie incorrectly received private tell"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Tell command works privately between players"}
}

// TestShoutCommand tests the shout command broadcasts to floor
func TestShoutCommand(serverAddr string) TestResult {
	const testName = "Shout Command"

	name1 := uniqueName("Shouter")
	name2 := uniqueName("FloorListener")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", name1, name2))

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect shouter"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect listener"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// Move listener to a different room on the same floor
	client2.SendCommand("south")
	time.Sleep(200 * time.Millisecond)

	client1.ClearMessages()
	client2.ClearMessages()

	logAction(testName, "Shouter shouts: Can you hear me!")
	client1.SendCommand("shout Can you hear me!")
	time.Sleep(300 * time.Millisecond)

	foundMessage := client2.WaitForMessage("Can you hear me", 1*time.Second) ||
		client2.WaitForMessage("shout", 1*time.Second)
	logResult(testName, foundMessage, "Listener received shout")

	if !foundMessage {
		return TestResult{Name: testName, Passed: false, Message: "Listener did not receive shout message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Shout command successfully broadcast to floor"}
}

// TestEmoteCommand tests the emote command shows custom actions
func TestEmoteCommand(serverAddr string) TestResult {
	const testName = "Emote Command"

	name1 := uniqueName("Emoter")
	name2 := uniqueName("Watcher")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", name1, name2))

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect emoter"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect watcher"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()

	logAction(testName, "Emoter emotes: dances a jig")
	client1.SendCommand("emote dances a jig")
	time.Sleep(300 * time.Millisecond)

	// Watcher should see "<name> dances a jig"
	foundEmote := client2.WaitForMessage("dances a jig", 1*time.Second)
	logResult(testName, foundEmote, "Watcher saw emote")

	if !foundEmote {
		return TestResult{Name: testName, Passed: false, Message: "Watcher did not see emote action"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Emote command shows custom actions to room"}
}

// TestChatFilterReplace tests that banned words are replaced
func TestChatFilterReplace(serverAddr string) TestResult {
	const testName = "Chat Filter Replace"

	name1 := uniqueName("Talker")
	name2 := uniqueName("Hearer")

	client1, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect talker"}
	}
	defer client1.Close()

	client2, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect hearer"}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)
	client1.ClearMessages()
	client2.ClearMessages()

	// "badword" is in chat_filter_test.yaml
	logAction(testName, "Talker says message with banned word 'badword'")
	client1.SendCommand("say This is a badword test")
	time.Sleep(300 * time.Millisecond)

	messages := client2.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Word should be replaced with asterisks
	hasAsterisks := strings.Contains(fullOutput, "***")
	hasBadword := strings.Contains(fullOutput, "badword")

	logResult(testName, hasAsterisks, "Message contains asterisks")
	logResult(testName, !hasBadword, "Banned word filtered out")

	if hasBadword {
		return TestResult{Name: testName, Passed: false, Message: "Banned word was not filtered"}
	}
	if !hasAsterisks {
		return TestResult{Name: testName, Passed: false, Message: "No asterisks in filtered message"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Chat filter replaces banned words with asterisks"}
}
