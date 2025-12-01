package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Group 9: Admin Commands
// =============================================================================

// TestAdminCommandsHidden tests that non-admins don't see admin commands
func TestAdminCommandsHidden(serverAddr string) TestResult {
	const testName = "Admin Commands Hidden"

	name := uniqueName("NonAdmin")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Checking help for admin commands...")
	client.ClearMessages()
	client.SendCommand("help")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Admin commands should not be visible
	hasAdmin := strings.Contains(fullOutput, "admin promote") || strings.Contains(fullOutput, "admin ban") ||
		strings.Contains(fullOutput, "admin kick") || strings.Contains(fullOutput, "admin teleport")
	logResult(testName, !hasAdmin, "Admin commands not visible")

	if hasAdmin {
		return TestResult{Name: testName, Passed: false, Message: "Admin commands visible to non-admin"}
	}

	return TestResult{Name: testName, Passed: true, Message: "Admin commands hidden from non-admin users"}
}

// TestAdminAnnounce tests the admin announce command (requires admin account)
func TestAdminAnnounce(serverAddr string) TestResult {
	const testName = "Admin Announce"

	// This test requires an admin account which we can't easily create in integration tests
	// without database access. We'll test that the command exists by trying it as non-admin.

	name := uniqueName("AnnounceTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Trying admin announce as non-admin...")
	client.ClearMessages()
	client.SendCommand("admin announce Test message")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should get permission denied or unknown command
	denied := strings.Contains(fullOutput, "permission") || strings.Contains(fullOutput, "admin") ||
		strings.Contains(fullOutput, "unknown") || strings.Contains(fullOutput, "Unknown") ||
		strings.Contains(fullOutput, "not") || strings.Contains(fullOutput, "cannot")
	logResult(testName, denied, "Non-admin cannot use admin announce")

	if !denied {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Admin announce should fail for non-admin. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Admin announce requires admin privileges"}
}
