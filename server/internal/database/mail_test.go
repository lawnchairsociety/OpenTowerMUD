package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMailOperations(t *testing.T) {
	// Create a temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Create two test accounts and characters
	account1, err := db.CreateAccount("sender", "password123")
	if err != nil {
		t.Fatalf("Failed to create account1: %v", err)
	}
	char1, err := db.CreateCharacter(account1.ID, "SenderChar")
	if err != nil {
		t.Fatalf("Failed to create character1: %v", err)
	}

	account2, err := db.CreateAccount("recipient", "password456")
	if err != nil {
		t.Fatalf("Failed to create account2: %v", err)
	}
	char2, err := db.CreateCharacter(account2.ID, "RecipientChar")
	if err != nil {
		t.Fatalf("Failed to create character2: %v", err)
	}

	t.Run("SendAndReceiveMail", func(t *testing.T) {
		// Send mail from char1 to char2
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Test Subject", "Test Body", 100, []string{"rusty_sword", "bandage"})
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		if mailID == 0 {
			t.Fatal("Expected non-zero mail ID")
		}

		// Check unread count
		count, err := db.GetUnreadMailCount(char2.ID)
		if err != nil {
			t.Fatalf("Failed to get unread count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 unread mail, got %d", count)
		}

		// Get mailbox
		summaries, err := db.GetMailbox(char2.ID)
		if err != nil {
			t.Fatalf("Failed to get mailbox: %v", err)
		}
		if len(summaries) != 1 {
			t.Fatalf("Expected 1 mail in mailbox, got %d", len(summaries))
		}

		s := summaries[0]
		if s.SenderName != "SenderChar" {
			t.Errorf("Expected sender 'SenderChar', got '%s'", s.SenderName)
		}
		if s.Subject != "Test Subject" {
			t.Errorf("Expected subject 'Test Subject', got '%s'", s.Subject)
		}
		if !s.HasGold {
			t.Error("Expected HasGold to be true")
		}
		if !s.HasItems {
			t.Error("Expected HasItems to be true")
		}
		if s.Read {
			t.Error("Expected mail to be unread")
		}

		// Read the mail
		mail, err := db.GetMail(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to get mail: %v", err)
		}
		if mail == nil {
			t.Fatal("Expected mail to be found")
		}
		if mail.Body != "Test Body" {
			t.Errorf("Expected body 'Test Body', got '%s'", mail.Body)
		}
		if mail.GoldAttached != 100 {
			t.Errorf("Expected 100 gold, got %d", mail.GoldAttached)
		}
		if len(mail.Items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(mail.Items))
		}
	})

	t.Run("MarkMailRead", func(t *testing.T) {
		// Send another mail
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Another Subject", "Another Body", 0, nil)
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		// Mark as read
		err = db.MarkMailRead(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to mark mail as read: %v", err)
		}

		// Verify it's read
		mail, err := db.GetMail(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to get mail: %v", err)
		}
		if !mail.Read {
			t.Error("Expected mail to be marked as read")
		}
	})

	t.Run("CollectGold", func(t *testing.T) {
		// Send mail with gold
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Gold Mail", "Here's some gold", 50, nil)
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		// Collect gold
		gold, err := db.CollectMailGold(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to collect gold: %v", err)
		}
		if gold != 50 {
			t.Errorf("Expected 50 gold, got %d", gold)
		}

		// Try to collect again (should fail)
		_, err = db.CollectMailGold(mailID, char2.ID)
		if err == nil {
			t.Error("Expected error when collecting gold twice")
		}
	})

	t.Run("CollectItems", func(t *testing.T) {
		// Send mail with items
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Item Mail", "Here are some items", 0, []string{"dagger", "health_potion"})
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		// Collect items
		items, err := db.CollectMailItems(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to collect items: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}

		// Try to collect again (should fail)
		_, err = db.CollectMailItems(mailID, char2.ID)
		if err == nil {
			t.Error("Expected error when collecting items twice")
		}
	})

	t.Run("DeleteMail", func(t *testing.T) {
		// Send mail without attachments
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Delete Me", "This mail will be deleted", 0, nil)
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		// Delete it
		err = db.DeleteMail(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to delete mail: %v", err)
		}

		// Verify it's gone
		mail, err := db.GetMail(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Error checking deleted mail: %v", err)
		}
		if mail != nil {
			t.Error("Expected mail to be deleted")
		}
	})

	t.Run("DeleteMailWithUncollectedAttachments", func(t *testing.T) {
		// Send mail with gold
		mailID, err := db.SendMail(char1.ID, "SenderChar", char2.ID, "RecipientChar",
			"Has Gold", "Can't delete me yet", 25, nil)
		if err != nil {
			t.Fatalf("Failed to send mail: %v", err)
		}

		// Try to delete (should fail)
		err = db.DeleteMail(mailID, char2.ID)
		if err == nil {
			t.Error("Expected error when deleting mail with uncollected attachments")
		}

		// Collect the gold
		_, err = db.CollectMailGold(mailID, char2.ID)
		if err != nil {
			t.Fatalf("Failed to collect gold: %v", err)
		}

		// Now delete should work
		err = db.DeleteMail(mailID, char2.ID)
		if err != nil {
			t.Errorf("Failed to delete mail after collecting attachments: %v", err)
		}
	})

	t.Run("GetCharacterIDByName", func(t *testing.T) {
		// Test case-insensitive lookup
		id, err := db.GetCharacterIDByName("senderchar")
		if err != nil {
			t.Fatalf("Failed to get character ID: %v", err)
		}
		if id != char1.ID {
			t.Errorf("Expected character ID %d, got %d", char1.ID, id)
		}

		// Test non-existent character
		id, err = db.GetCharacterIDByName("nonexistent")
		if err != nil {
			t.Fatalf("Error looking up non-existent character: %v", err)
		}
		if id != 0 {
			t.Errorf("Expected 0 for non-existent character, got %d", id)
		}
	})
}
