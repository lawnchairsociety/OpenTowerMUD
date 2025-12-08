package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/mail"
)

// SendMail creates a new mail message with optional gold and item attachments.
func (d *Database) SendMail(senderID int64, senderName string, recipientID int64, recipientName string,
	subject, body string, goldAmount int, itemIDs []string) (int64, error) {

	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the mail record
	result, err := tx.Exec(`
		INSERT INTO mail (sender_id, sender_name, recipient_id, recipient_name, subject, body, gold_attached)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		senderID, senderName, recipientID, recipientName, subject, body, goldAmount)
	if err != nil {
		return 0, fmt.Errorf("failed to insert mail: %w", err)
	}

	mailID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get mail ID: %w", err)
	}

	// Insert item attachments
	for _, itemID := range itemIDs {
		_, err = tx.Exec(`INSERT INTO mail_items (mail_id, item_id) VALUES (?, ?)`, mailID, itemID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert mail item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return mailID, nil
}

// GetMailbox returns all mail for a recipient, ordered by sent date descending.
func (d *Database) GetMailbox(recipientID int64) ([]mail.MailSummary, error) {
	rows, err := d.db.Query(`
		SELECT m.id, m.sender_name, m.subject, m.gold_attached, m.gold_collected, m.read, m.sent_at,
		       (SELECT COUNT(*) FROM mail_items mi WHERE mi.mail_id = m.id AND mi.collected = 0) as uncollected_items
		FROM mail m
		WHERE m.recipient_id = ?
		ORDER BY m.sent_at DESC`,
		recipientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query mailbox: %w", err)
	}
	defer rows.Close()

	var summaries []mail.MailSummary
	for rows.Next() {
		var s mail.MailSummary
		var goldAttached int
		var goldCollected bool
		var uncollectedItems int
		var sentAt string

		err := rows.Scan(&s.ID, &s.SenderName, &s.Subject, &goldAttached, &goldCollected, &s.Read, &sentAt, &uncollectedItems)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mail summary: %w", err)
		}

		s.HasGold = goldAttached > 0 && !goldCollected
		s.HasItems = uncollectedItems > 0
		s.SentAt, _ = time.Parse("2006-01-02 15:04:05", sentAt)

		summaries = append(summaries, s)
	}

	return summaries, nil
}

// GetMail returns a single mail message with its items.
func (d *Database) GetMail(mailID int64, recipientID int64) (*mail.Mail, error) {
	var m mail.Mail
	var sentAt string

	err := d.db.QueryRow(`
		SELECT id, sender_id, sender_name, recipient_id, recipient_name, subject, body,
		       gold_attached, gold_collected, items_collected, read, sent_at
		FROM mail
		WHERE id = ? AND recipient_id = ?`,
		mailID, recipientID).Scan(
		&m.ID, &m.SenderID, &m.SenderName, &m.RecipientID, &m.RecipientName,
		&m.Subject, &m.Body, &m.GoldAttached, &m.GoldCollected, &m.ItemsCollected,
		&m.Read, &sentAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query mail: %w", err)
	}

	m.SentAt, _ = time.Parse("2006-01-02 15:04:05", sentAt)

	// Get items
	rows, err := d.db.Query(`
		SELECT id, mail_id, item_id, collected
		FROM mail_items
		WHERE mail_id = ?`,
		mailID)
	if err != nil {
		return nil, fmt.Errorf("failed to query mail items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item mail.MailItem
		if err := rows.Scan(&item.ID, &item.MailID, &item.ItemID, &item.Collected); err != nil {
			return nil, fmt.Errorf("failed to scan mail item: %w", err)
		}
		m.Items = append(m.Items, item)
	}

	return &m, nil
}

// MarkMailRead marks a mail message as read.
func (d *Database) MarkMailRead(mailID int64, recipientID int64) error {
	_, err := d.db.Exec(`UPDATE mail SET read = 1 WHERE id = ? AND recipient_id = ?`, mailID, recipientID)
	if err != nil {
		return fmt.Errorf("failed to mark mail as read: %w", err)
	}
	return nil
}

// CollectMailGold marks gold as collected and returns the amount.
func (d *Database) CollectMailGold(mailID int64, recipientID int64) (int, error) {
	var goldAmount int
	var goldCollected bool

	err := d.db.QueryRow(`SELECT gold_attached, gold_collected FROM mail WHERE id = ? AND recipient_id = ?`,
		mailID, recipientID).Scan(&goldAmount, &goldCollected)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("mail not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query mail: %w", err)
	}

	if goldCollected {
		return 0, fmt.Errorf("gold already collected")
	}

	_, err = d.db.Exec(`UPDATE mail SET gold_collected = 1 WHERE id = ? AND recipient_id = ?`, mailID, recipientID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark gold as collected: %w", err)
	}

	return goldAmount, nil
}

// CollectMailItems marks all items as collected and returns their IDs.
func (d *Database) CollectMailItems(mailID int64, recipientID int64) ([]string, error) {
	// Verify ownership
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM mail WHERE id = ? AND recipient_id = ?`, mailID, recipientID).Scan(&count)
	if err != nil || count == 0 {
		return nil, fmt.Errorf("mail not found")
	}

	// Get uncollected items
	rows, err := d.db.Query(`SELECT item_id FROM mail_items WHERE mail_id = ? AND collected = 0`, mailID)
	if err != nil {
		return nil, fmt.Errorf("failed to query mail items: %w", err)
	}
	defer rows.Close()

	var itemIDs []string
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("failed to scan item ID: %w", err)
		}
		itemIDs = append(itemIDs, itemID)
	}

	if len(itemIDs) == 0 {
		return nil, fmt.Errorf("no items to collect")
	}

	// Mark items as collected
	_, err = d.db.Exec(`UPDATE mail_items SET collected = 1 WHERE mail_id = ?`, mailID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark items as collected: %w", err)
	}

	// Update the mail record
	_, err = d.db.Exec(`UPDATE mail SET items_collected = 1 WHERE id = ?`, mailID)
	if err != nil {
		return nil, fmt.Errorf("failed to update mail items_collected: %w", err)
	}

	return itemIDs, nil
}

// DeleteMail deletes a mail message if all attachments are collected.
func (d *Database) DeleteMail(mailID int64, recipientID int64) error {
	// Check if there are uncollected attachments
	var goldAttached int
	var goldCollected bool
	var uncollectedItems int

	err := d.db.QueryRow(`
		SELECT m.gold_attached, m.gold_collected,
		       (SELECT COUNT(*) FROM mail_items mi WHERE mi.mail_id = m.id AND mi.collected = 0)
		FROM mail m
		WHERE m.id = ? AND m.recipient_id = ?`,
		mailID, recipientID).Scan(&goldAttached, &goldCollected, &uncollectedItems)
	if err == sql.ErrNoRows {
		return fmt.Errorf("mail not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query mail: %w", err)
	}

	if (goldAttached > 0 && !goldCollected) || uncollectedItems > 0 {
		return fmt.Errorf("collect all attachments before deleting")
	}

	_, err = d.db.Exec(`DELETE FROM mail WHERE id = ? AND recipient_id = ?`, mailID, recipientID)
	if err != nil {
		return fmt.Errorf("failed to delete mail: %w", err)
	}

	return nil
}

// GetUnreadMailCount returns the number of unread messages for a player.
func (d *Database) GetUnreadMailCount(recipientID int64) (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM mail WHERE recipient_id = ? AND read = 0`, recipientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread mail: %w", err)
	}
	return count, nil
}

// GetMailCount returns the total number of messages for a player.
func (d *Database) GetMailCount(recipientID int64) (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM mail WHERE recipient_id = ?`, recipientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count mail: %w", err)
	}
	return count, nil
}

// GetCharacterIDByName returns the character ID for a given name.
func (d *Database) GetCharacterIDByName(name string) (int64, error) {
	var id int64
	err := d.db.QueryRow(`SELECT id FROM characters WHERE name = ? COLLATE NOCASE`, name).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query character: %w", err)
	}
	return id, nil
}
