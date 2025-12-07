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

// TestGiveItem tests giving items to another player
func TestGiveItem(serverAddr string) TestResult {
	const testName = "Give Item"

	giverName := uniqueName("Giver")
	receiverName := uniqueName("Receiver")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", giverName, receiverName))

	giver, err1 := testclient.NewTestClient(giverName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect giver"}
	}
	defer giver.Close()

	receiver, err2 := testclient.NewTestClient(receiverName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect receiver"}
	}
	defer receiver.Close()

	time.Sleep(500 * time.Millisecond)

	// Navigate both to General Store
	logAction(testName, "Both players navigating to General Store...")
	navigateToGeneralStore(giver)
	navigateToGeneralStore(receiver)

	// Giver buys bread
	logAction(testName, "Giver buying bread...")
	giver.SendCommand("buy bread")
	time.Sleep(300 * time.Millisecond)

	giver.ClearMessages()
	receiver.ClearMessages()

	// Give item to receiver
	logAction(testName, fmt.Sprintf("Giver giving bread to %s...", receiverName))
	giver.SendCommand(fmt.Sprintf("give bread %s", receiverName))
	time.Sleep(300 * time.Millisecond)

	// Check giver got confirmation
	giverConfirm := giver.WaitForMessage("give", 1*time.Second)
	logResult(testName, giverConfirm, "Giver received confirmation")

	// Check receiver got the item notification
	receiverNotify := receiver.WaitForMessage("gives you", 1*time.Second)
	logResult(testName, receiverNotify, "Receiver notified of gift")

	if !giverConfirm {
		return TestResult{Name: testName, Passed: false, Message: "Giver did not get confirmation"}
	}
	if !receiverNotify {
		return TestResult{Name: testName, Passed: false, Message: "Receiver was not notified of gift"}
	}

	// Verify receiver has the item
	receiver.ClearMessages()
	receiver.SendCommand("inventory")
	time.Sleep(300 * time.Millisecond)

	messages := receiver.GetMessages()
	fullOutput := strings.Join(messages, " ")
	hasItem := strings.Contains(strings.ToLower(fullOutput), "bread")
	logResult(testName, hasItem, "Receiver has bread in inventory")

	if !hasItem {
		return TestResult{Name: testName, Passed: false, Message: "Receiver does not have bread in inventory"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Give item command works correctly"}
}

// TestGiveGold tests giving gold to another player
func TestGiveGold(serverAddr string) TestResult {
	const testName = "Give Gold"

	giverName := uniqueName("GoldGiver")
	receiverName := uniqueName("GoldReceiver")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", giverName, receiverName))

	giver, err1 := testclient.NewTestClient(giverName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect giver"}
	}
	defer giver.Close()

	receiver, err2 := testclient.NewTestClient(receiverName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect receiver"}
	}
	defer receiver.Close()

	time.Sleep(500 * time.Millisecond)

	// Check receiver's starting gold
	receiver.ClearMessages()
	receiver.SendCommand("gold")
	time.Sleep(300 * time.Millisecond)

	giver.ClearMessages()
	receiver.ClearMessages()

	// Give gold to receiver (players start with 20 gold)
	logAction(testName, fmt.Sprintf("Giver giving 5 gold to %s...", receiverName))
	giver.SendCommand(fmt.Sprintf("give 5 gold %s", receiverName))
	time.Sleep(300 * time.Millisecond)

	// Check giver got confirmation
	giverConfirm := giver.WaitForMessage("give", 1*time.Second) && giver.WaitForMessage("gold", 1*time.Second)
	logResult(testName, giverConfirm, "Giver received confirmation")

	// Check receiver got the gold notification
	receiverNotify := receiver.WaitForMessage("gives you", 1*time.Second) && receiver.WaitForMessage("gold", 1*time.Second)
	logResult(testName, receiverNotify, "Receiver notified of gold")

	if !giverConfirm {
		messages := giver.GetMessages()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Giver did not get confirmation. Got: %v", messages)}
	}
	if !receiverNotify {
		return TestResult{Name: testName, Passed: false, Message: "Receiver was not notified of gold"}
	}

	// Verify receiver has more gold (started with 20, now should have 25)
	receiver.ClearMessages()
	receiver.SendCommand("gold")
	time.Sleep(300 * time.Millisecond)

	messages := receiver.GetMessages()
	fullOutput := strings.Join(messages, " ")
	hasMoreGold := strings.Contains(fullOutput, "25")
	logResult(testName, hasMoreGold, "Receiver has 25 gold")

	if !hasMoreGold {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Receiver gold not updated correctly. Got: %s", fullOutput)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Give gold command works correctly"}
}

// TestGiveRequiresSameRoom tests that give fails when players are in different rooms
func TestGiveRequiresSameRoom(serverAddr string) TestResult {
	const testName = "Give Requires Same Room"

	giverName := uniqueName("RoomGiver")
	receiverName := uniqueName("RoomReceiver")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", giverName, receiverName))

	giver, err1 := testclient.NewTestClient(giverName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect giver"}
	}
	defer giver.Close()

	receiver, err2 := testclient.NewTestClient(receiverName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect receiver"}
	}
	defer receiver.Close()

	time.Sleep(500 * time.Millisecond)

	// Move receiver to a different room
	logAction(testName, "Moving receiver to different room...")
	receiver.SendCommand("south")
	time.Sleep(300 * time.Millisecond)

	giver.ClearMessages()

	// Try to give gold to receiver in different room
	logAction(testName, fmt.Sprintf("Giver trying to give gold to %s in different room...", receiverName))
	giver.SendCommand(fmt.Sprintf("give 5 gold %s", receiverName))
	time.Sleep(300 * time.Millisecond)

	// Should fail with "not here" message
	notHere := giver.WaitForMessage("not here", 1*time.Second)
	logResult(testName, notHere, "Give failed - player not here")

	if !notHere {
		messages := giver.GetMessages()
		// Check if it succeeded (which would be wrong)
		fullOutput := strings.Join(messages, " ")
		if strings.Contains(fullOutput, "give") && strings.Contains(fullOutput, "gold") && !strings.Contains(strings.ToLower(fullOutput), "not") {
			return TestResult{Name: testName, Passed: false, Message: "Give succeeded when players in different rooms"}
		}
	}

	return TestResult{Name: testName, Passed: true, Message: "Give correctly requires players in same room"}
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

// TestAntispamRateLimit tests that sending too many messages triggers rate limiting
func TestAntispamRateLimit(serverAddr string) TestResult {
	const testName = "Antispam Rate Limit"

	name := uniqueName("Spammer")
	logAction(testName, fmt.Sprintf("Connecting %s...", name))

	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect"}
	}
	defer client.Close()

	time.Sleep(500 * time.Millisecond)
	client.ClearMessages()

	// Send messages rapidly (more than 5 in 10 seconds)
	logAction(testName, "Sending 6 messages rapidly...")
	for i := 0; i < 6; i++ {
		client.SendCommand(fmt.Sprintf("say Message number %d", i+1))
		time.Sleep(50 * time.Millisecond) // Very fast
	}

	time.Sleep(300 * time.Millisecond)

	// Should have received a rate limit warning
	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	rateLimited := strings.Contains(strings.ToLower(fullOutput), "too quickly") ||
		strings.Contains(strings.ToLower(fullOutput), "slow down")
	logResult(testName, rateLimited, "Rate limit triggered")

	if !rateLimited {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Rate limit not triggered after 6 rapid messages. Got: %s", fullOutput)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Antispam rate limiting works correctly"}
}

// TestAntispamRepeatMessage tests that repeating the same message is blocked
func TestAntispamRepeatMessage(serverAddr string) TestResult {
	const testName = "Antispam Repeat Message"

	name := uniqueName("Repeater")
	logAction(testName, fmt.Sprintf("Connecting %s...", name))

	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect"}
	}
	defer client.Close()

	time.Sleep(500 * time.Millisecond)
	client.ClearMessages()

	// Send the same message twice
	logAction(testName, "Sending same message twice...")
	client.SendCommand("say Buy gold at mysite dot com")
	time.Sleep(200 * time.Millisecond)
	client.ClearMessages()

	client.SendCommand("say Buy gold at mysite dot com")
	time.Sleep(300 * time.Millisecond)

	// Should have received a repeat warning
	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")
	repeatBlocked := strings.Contains(strings.ToLower(fullOutput), "repeat") ||
		strings.Contains(strings.ToLower(fullOutput), "same message")
	logResult(testName, repeatBlocked, "Repeat message blocked")

	if !repeatBlocked {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Repeat message not blocked. Got: %s", fullOutput)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Antispam repeat detection works correctly"}
}

// TestIgnoreCommand tests that ignoring a player blocks their messages
func TestIgnoreCommand(serverAddr string) TestResult {
	const testName = "Ignore Command"

	senderName := uniqueName("Sender")
	ignorerName := uniqueName("Ignorer")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", senderName, ignorerName))

	sender, err1 := testclient.NewTestClient(senderName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect sender"}
	}
	defer sender.Close()

	ignorer, err2 := testclient.NewTestClient(ignorerName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect ignorer"}
	}
	defer ignorer.Close()

	time.Sleep(500 * time.Millisecond)

	// First verify messages are received before ignoring
	sender.ClearMessages()
	ignorer.ClearMessages()

	logAction(testName, "Sender says hello (before ignore)...")
	sender.SendCommand("say Hello before ignore!")
	time.Sleep(300 * time.Millisecond)

	receivedBefore := ignorer.WaitForMessage("Hello before ignore", 1*time.Second)
	logResult(testName, receivedBefore, "Message received before ignore")

	if !receivedBefore {
		return TestResult{Name: testName, Passed: false, Message: "Message not received before ignore (test setup issue)"}
	}

	// Now ignore the sender
	logAction(testName, fmt.Sprintf("Ignorer ignoring %s...", senderName))
	ignorer.SendCommand(fmt.Sprintf("ignore %s", senderName))
	time.Sleep(300 * time.Millisecond)

	// Verify ignore confirmation
	ignoreConfirm := ignorer.WaitForMessage("ignoring", 1*time.Second)
	logResult(testName, ignoreConfirm, "Ignore confirmed")

	ignorer.ClearMessages()

	// Sender tries to talk again
	logAction(testName, "Sender says hello (after ignore)...")
	sender.SendCommand("say Hello after ignore!")
	time.Sleep(300 * time.Millisecond)

	// Ignorer should NOT see the message
	messages := ignorer.GetMessages()
	fullOutput := strings.Join(messages, " ")
	receivedAfter := strings.Contains(fullOutput, "Hello after ignore")
	logResult(testName, !receivedAfter, "Message blocked after ignore")

	if receivedAfter {
		return TestResult{Name: testName, Passed: false, Message: "Message received after ignore (should be blocked)"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Ignore command blocks messages from ignored players"}
}

// TestIgnoreTell tests that ignore blocks tell messages
func TestIgnoreTell(serverAddr string) TestResult {
	const testName = "Ignore Tell"

	senderName := uniqueName("TellSender")
	ignorerName := uniqueName("TellIgnorer")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", senderName, ignorerName))

	sender, err1 := testclient.NewTestClient(senderName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect sender"}
	}
	defer sender.Close()

	ignorer, err2 := testclient.NewTestClient(ignorerName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect ignorer"}
	}
	defer ignorer.Close()

	time.Sleep(500 * time.Millisecond)

	// Ignore the sender
	logAction(testName, fmt.Sprintf("Ignorer ignoring %s...", senderName))
	ignorer.SendCommand(fmt.Sprintf("ignore %s", senderName))
	time.Sleep(300 * time.Millisecond)

	ignorer.ClearMessages()
	sender.ClearMessages()

	// Sender tries to tell the ignorer
	logAction(testName, fmt.Sprintf("Sender trying to tell %s...", ignorerName))
	sender.SendCommand(fmt.Sprintf("tell %s Secret message!", ignorerName))
	time.Sleep(300 * time.Millisecond)

	// Sender should get confirmation (to not reveal they're ignored)
	senderMessages := sender.GetMessages()
	senderOutput := strings.Join(senderMessages, " ")
	senderGotConfirm := strings.Contains(senderOutput, "You tell")
	logResult(testName, senderGotConfirm, "Sender got fake confirmation")

	// Ignorer should NOT receive the tell
	ignorerMessages := ignorer.GetMessages()
	ignorerOutput := strings.Join(ignorerMessages, " ")
	ignorerReceived := strings.Contains(ignorerOutput, "Secret message")
	logResult(testName, !ignorerReceived, "Tell blocked")

	if ignorerReceived {
		return TestResult{Name: testName, Passed: false, Message: "Tell message received after ignore (should be blocked)"}
	}
	if !senderGotConfirm {
		return TestResult{Name: testName, Passed: false, Message: "Sender did not get fake confirmation (reveals ignore status)"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Ignore correctly blocks tell messages without revealing ignore status"}
}

// TestUnignoreCommand tests removing a player from ignore list
func TestUnignoreCommand(serverAddr string) TestResult {
	const testName = "Unignore Command"

	senderName := uniqueName("UnignoreSender")
	ignorerName := uniqueName("Unignorer")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", senderName, ignorerName))

	sender, err1 := testclient.NewTestClient(senderName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect sender"}
	}
	defer sender.Close()

	ignorer, err2 := testclient.NewTestClient(ignorerName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect ignorer"}
	}
	defer ignorer.Close()

	time.Sleep(500 * time.Millisecond)

	// Ignore then unignore
	logAction(testName, fmt.Sprintf("Ignorer ignoring then unignoring %s...", senderName))
	ignorer.SendCommand(fmt.Sprintf("ignore %s", senderName))
	time.Sleep(200 * time.Millisecond)
	ignorer.SendCommand(fmt.Sprintf("unignore %s", senderName))
	time.Sleep(300 * time.Millisecond)

	// Verify unignore confirmation
	unignoreConfirm := ignorer.WaitForMessage("no longer ignoring", 1*time.Second)
	logResult(testName, unignoreConfirm, "Unignore confirmed")

	ignorer.ClearMessages()

	// Sender talks
	logAction(testName, "Sender says hello (after unignore)...")
	sender.SendCommand("say Hello after unignore!")
	time.Sleep(300 * time.Millisecond)

	// Ignorer should now see the message
	receivedAfter := ignorer.WaitForMessage("Hello after unignore", 1*time.Second)
	logResult(testName, receivedAfter, "Message received after unignore")

	if !receivedAfter {
		return TestResult{Name: testName, Passed: false, Message: "Message not received after unignore"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Unignore command restores message reception"}
}

// TestReportCommand tests that report command works and logs
func TestReportCommand(serverAddr string) TestResult {
	const testName = "Report Command"

	reporterName := uniqueName("Reporter")
	reportedName := uniqueName("Reported")
	logAction(testName, fmt.Sprintf("Connecting %s and %s...", reporterName, reportedName))

	reporter, err1 := testclient.NewTestClient(reporterName, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect reporter"}
	}
	defer reporter.Close()

	reported, err2 := testclient.NewTestClient(reportedName, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect reported player"}
	}
	defer reported.Close()

	time.Sleep(500 * time.Millisecond)
	reporter.ClearMessages()

	// Report the other player
	logAction(testName, fmt.Sprintf("Reporting %s for spam...", reportedName))
	reporter.SendCommand(fmt.Sprintf("report %s Spamming in chat", reportedName))
	time.Sleep(300 * time.Millisecond)

	// Should get confirmation
	messages := reporter.GetMessages()
	fullOutput := strings.Join(messages, " ")
	gotConfirm := strings.Contains(fullOutput, "report") && strings.Contains(fullOutput, "logged")
	logResult(testName, gotConfirm, "Report confirmation received")

	if !gotConfirm {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Did not get report confirmation. Got: %s", fullOutput)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Report command successfully logs report"}
}

// TestIgnoreList tests viewing the ignore list
func TestIgnoreList(serverAddr string) TestResult {
	const testName = "Ignore List"

	name1 := uniqueName("ListViewer")
	name2 := uniqueName("ToIgnoreOne")
	name3 := uniqueName("ToIgnoreTwo")
	logAction(testName, fmt.Sprintf("Connecting %s, %s, %s...", name1, name2, name3))

	viewer, err1 := testclient.NewTestClient(name1, serverAddr)
	if err1 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect viewer"}
	}
	defer viewer.Close()

	time.Sleep(300 * time.Millisecond) // Wait between client connections

	ignored1, err2 := testclient.NewTestClient(name2, serverAddr)
	if err2 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect ignored1"}
	}
	defer ignored1.Close()

	time.Sleep(300 * time.Millisecond) // Wait between client connections

	ignored2, err3 := testclient.NewTestClient(name3, serverAddr)
	if err3 != nil {
		return TestResult{Name: testName, Passed: false, Message: "Failed to connect ignored2"}
	}
	defer ignored2.Close()

	time.Sleep(500 * time.Millisecond)

	// Ignore two players
	logAction(testName, "Ignoring two players...")
	viewer.SendCommand(fmt.Sprintf("ignore %s", name2))
	time.Sleep(200 * time.Millisecond)
	viewer.SendCommand(fmt.Sprintf("ignore %s", name3))
	time.Sleep(200 * time.Millisecond)

	viewer.ClearMessages()

	// View ignore list
	logAction(testName, "Viewing ignore list...")
	viewer.SendCommand("ignore")
	time.Sleep(300 * time.Millisecond)

	messages := viewer.GetMessages()
	fullOutput := strings.Join(messages, " ")
	fullOutputLower := strings.ToLower(fullOutput)

	// Should show both ignored players (case-insensitive check since names may be stored lowercase)
	hasIgnored1 := strings.Contains(fullOutputLower, strings.ToLower(name2))
	hasIgnored2 := strings.Contains(fullOutputLower, strings.ToLower(name3))
	logResult(testName, hasIgnored1, fmt.Sprintf("List shows %s", name2))
	logResult(testName, hasIgnored2, fmt.Sprintf("List shows %s", name3))

	if !hasIgnored1 || !hasIgnored2 {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Ignore list missing players. Got: %s", fullOutput)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Ignore list shows all ignored players"}
}

// TestWhoCommand tests the who command listing online players
func TestWhoCommand(serverAddr string) TestResult {
	const testName = "Who Command"

	name := uniqueName("WhoTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Execute the who command
	logAction(testName, "Checking who command...")
	client.ClearMessages()
	client.SendCommand("who")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should see "Online Players" and our own name in the list
	hasPlayerList := strings.Contains(fullOutput, "Online") || strings.Contains(fullOutput, "Player")
	hasOwnName := strings.Contains(fullOutput, "WhoTest")
	logResult(testName, hasPlayerList, "Player list header shown")
	logResult(testName, hasOwnName, "Own name in list")

	if !hasPlayerList {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Who command failed. Got: %v", messages)}
	}
	if !hasOwnName {
		return TestResult{Name: testName, Passed: false, Message: "Own name not shown in who list"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Who command shows online players including self"}
}

// TestWhoMultiplePlayers tests the who command with multiple connected players
func TestWhoMultiplePlayers(serverAddr string) TestResult {
	const testName = "Who Multiple Players"

	// Create first player
	name1 := uniqueName("WhoMultiOne")
	client1, err := testclient.NewTestClient(name1, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Client 1 connection failed: %v", err)}
	}
	defer client1.Close()

	time.Sleep(500 * time.Millisecond) // Wait between client connections

	// Create second player
	name2 := uniqueName("WhoMultiTwo")
	client2, err := testclient.NewTestClient(name2, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Client 2 connection failed: %v", err)}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// First player checks who
	logAction(testName, "Player 1 checking who list...")
	client1.ClearMessages()
	client1.SendCommand("who")
	time.Sleep(300 * time.Millisecond)

	messages := client1.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Both players should be visible
	hasPlayer1 := strings.Contains(fullOutput, "WhoMultiOne")
	hasPlayer2 := strings.Contains(fullOutput, "WhoMultiTwo")
	logResult(testName, hasPlayer1, "Player 1 in list")
	logResult(testName, hasPlayer2, "Player 2 in list")

	if !hasPlayer1 || !hasPlayer2 {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Who command missing players. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Who command shows multiple online players"}
}
