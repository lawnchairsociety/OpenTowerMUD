package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/mail"
)

// executeMail handles all mail commands.
func executeMail(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		return executeMailList(p)
	}

	subCmd := strings.ToLower(c.Args[0])
	switch subCmd {
	case "read":
		if len(c.Args) < 2 {
			return "Usage: mail read <id>"
		}
		return executeMailRead(c.Args[1], p)
	case "send":
		if len(c.Args) < 2 {
			return "Usage: mail send <player>"
		}
		return executeMailSend(strings.Join(c.Args[1:], " "), p)
	case "collect":
		if len(c.Args) < 2 {
			return "Usage: mail collect <id>"
		}
		return executeMailCollect(c.Args[1], p)
	case "delete":
		if len(c.Args) < 2 {
			return "Usage: mail delete <id>"
		}
		return executeMailDelete(c.Args[1], p)
	case "reply":
		if len(c.Args) < 2 {
			return "Usage: mail reply <id>"
		}
		return executeMailReply(c.Args[1], p)
	default:
		// Maybe they typed a number directly
		if _, err := strconv.ParseInt(subCmd, 10, 64); err == nil {
			return executeMailRead(subCmd, p)
		}
		return "Unknown mail command. Use: mail, mail read <id>, mail send <player>, mail collect <id>, mail delete <id>, mail reply <id>"
	}
}

// executeMailList shows the player's mailbox.
func executeMailList(p PlayerInterface) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	// Get unread count (works anywhere)
	unreadCount, err := db.GetUnreadMailCount(p.GetCharacterID())
	if err != nil {
		logger.Error("Failed to get unread mail count", "error", err, "player", p.GetName())
		return "Failed to check mailbox."
	}

	// If not at a mailbox, just show the count
	if !room.HasFeature("mailbox") {
		if unreadCount == 0 {
			return "You have no unread messages. Visit a mailbox to manage your mail."
		}
		return fmt.Sprintf("You have %d unread message(s). Visit a mailbox to read them.", unreadCount)
	}

	// At a mailbox - show full list
	summaries, err := db.GetMailbox(p.GetCharacterID())
	if err != nil {
		logger.Error("Failed to get mailbox", "error", err, "player", p.GetName())
		return "Failed to retrieve mailbox."
	}

	if len(summaries) == 0 {
		return "Your mailbox is empty."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("\n=== Your Mailbox (%d unread) ===\n", unreadCount))
	result.WriteString(" ID   From            Subject                      Date\n")
	result.WriteString("---  ----            -------                      ----\n")

	for _, s := range summaries {
		// Format the ID with brackets if unread
		idStr := fmt.Sprintf("%3d", s.ID)
		if !s.Read {
			idStr = fmt.Sprintf("[%d]", s.ID)
		}

		// Truncate subject if too long
		subject := s.Subject
		if len(subject) > 25 {
			subject = subject[:22] + "..."
		}

		// Add attachment indicators
		if s.HasGold || s.HasItems {
			indicators := ""
			if s.HasGold {
				indicators += "[GOLD]"
			}
			if s.HasItems {
				indicators += "[ITEMS]"
			}
			maxSubjectLen := 25 - len(indicators) - 1 // -1 for space
			if maxSubjectLen < 4 {
				maxSubjectLen = 4 // Minimum readable length
			}
			if len(subject) > maxSubjectLen {
				subject = subject[:maxSubjectLen-3] + "..."
			}
			subject = subject + " " + indicators
		}

		// Format time
		timeStr := formatMailTime(s.SentAt)

		result.WriteString(fmt.Sprintf("%-4s %-15s %-28s %s\n", idStr, truncate(s.SenderName, 15), subject, timeStr))
	}

	result.WriteString("\nCommands: mail read <id>, mail collect <id>, mail delete <id>, mail send <player>")

	return result.String()
}

// translateMailIndex converts a user-facing mail index (1, 2, 3...) to the actual database mail ID.
// Returns the mail ID, or 0 if the index is invalid.
func translateMailIndex(indexStr string, p PlayerInterface, db *database.Database) (int64, int64, string) {
	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil || index < 1 {
		return 0, 0, "Invalid mail number. Use a number from your mailbox list."
	}

	mailID, err := db.GetMailIDByIndex(p.GetCharacterID(), index)
	if err != nil {
		logger.Error("Failed to translate mail index", "error", err, "player", p.GetName(), "index", index)
		return 0, 0, "Failed to look up mail."
	}
	if mailID == 0 {
		return 0, 0, "Mail not found."
	}

	return mailID, index, ""
}

// executeMailRead reads a specific mail message.
func executeMailRead(idStr string, p PlayerInterface) string {
	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	if !room.HasFeature("mailbox") {
		return "You need to be at a mailbox to read your mail."
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Translate user-facing index to actual mail ID
	mailID, mailIndex, errMsg := translateMailIndex(idStr, p, db)
	if errMsg != "" {
		return errMsg
	}

	m, err := db.GetMail(mailID, p.GetCharacterID())
	if err != nil {
		logger.Error("Failed to get mail", "error", err, "player", p.GetName(), "mailID", mailID)
		return "Failed to retrieve mail."
	}

	if m == nil {
		return "Mail not found."
	}

	// Mark as read
	if !m.Read {
		if err := db.MarkMailRead(mailID, p.GetCharacterID()); err != nil {
			logger.Error("Failed to mark mail as read", "error", err, "player", p.GetName(), "mailID", mailID)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("\n=== Message from %s ===\n", m.SenderName))
	result.WriteString(fmt.Sprintf("Subject: %s\n", m.Subject))
	result.WriteString(fmt.Sprintf("Date: %s\n\n", formatMailTime(m.SentAt)))
	result.WriteString(m.Body)
	result.WriteString("\n")

	// Show attachments
	hasAttachments := false
	if m.GoldAttached > 0 || len(m.Items) > 0 {
		result.WriteString("\nAttachments:\n")
		hasAttachments = true

		if m.GoldAttached > 0 {
			if m.GoldCollected {
				result.WriteString(fmt.Sprintf("  - Gold: %d (collected)\n", m.GoldAttached))
			} else {
				result.WriteString(fmt.Sprintf("  - Gold: %d\n", m.GoldAttached))
			}
		}

		if len(m.Items) > 0 {
			for _, item := range m.Items {
				itemDef := server.GetItemByID(item.ItemID)
				itemName := item.ItemID
				if itemDef != nil {
					itemName = itemDef.Name
				}
				if item.Collected {
					result.WriteString(fmt.Sprintf("  - %s (collected)\n", itemName))
				} else {
					result.WriteString(fmt.Sprintf("  - %s\n", itemName))
				}
			}
		}
	}

	result.WriteString("\n")
	if hasAttachments && m.HasAttachments() {
		result.WriteString(fmt.Sprintf("Type 'mail collect %d' to collect attachments.\n", mailIndex))
	}
	result.WriteString(fmt.Sprintf("Type 'mail reply %d' to reply.\n", mailIndex))
	if !m.HasAttachments() {
		result.WriteString(fmt.Sprintf("Type 'mail delete %d' to delete.\n", mailIndex))
	}

	return result.String()
}

// executeMailSend handles sending mail to another player.
// If the input contains a pipe separator, it sends the mail directly.
// Otherwise, it shows usage instructions.
func executeMailSend(args string, p PlayerInterface) string {
	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	if !room.HasFeature("mailbox") {
		return "You need to be at a mailbox to send mail."
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Check if this is a full send command (contains pipe separator)
	if strings.Contains(args, "|") {
		return executeMailSendFull(args, p, server, db)
	}

	// Just a recipient name - validate and show usage
	recipientName := args

	recipientID, err := db.GetCharacterIDByName(recipientName)
	if err != nil {
		logger.Error("Failed to look up recipient", "error", err, "recipient", recipientName)
		return "Failed to look up recipient."
	}
	if recipientID == 0 {
		return fmt.Sprintf("No player named '%s' exists.", recipientName)
	}

	if recipientID == p.GetCharacterID() {
		return "You cannot send mail to yourself."
	}

	mailCount, err := db.GetMailCount(recipientID)
	if err != nil {
		logger.Error("Failed to check recipient mailbox", "error", err, "recipient", recipientName)
		return "Failed to check recipient's mailbox."
	}

	if mailCount >= mail.MaxMailboxSize {
		return fmt.Sprintf("%s's mailbox is full. They need to delete some messages first.", recipientName)
	}

	return fmt.Sprintf("To send mail to %s, use:\n"+
		"mail send %s <subject> | <message>\n\n"+
		"Example: mail send %s Hello! | This is the body of my message.\n\n"+
		"To attach gold, add 'gold:<amount>' at the end.\n"+
		"Example: mail send %s Payment | Here's the gold I owe you. gold:100",
		recipientName, recipientName, recipientName, recipientName)
}

// executeMailSendFull handles the full mail send with subject and body.
func executeMailSendFull(args string, p PlayerInterface, server ServerInterface, db *database.Database) string {
	// Parse: <recipient> <subject> | <body> [gold:<amount>]
	parts := strings.SplitN(args, "|", 2)
	if len(parts) != 2 {
		return "Usage: mail send <player> <subject> | <message> [gold:<amount>]"
	}

	// Parse recipient and subject from first part
	headerParts := strings.Fields(strings.TrimSpace(parts[0]))
	if len(headerParts) < 2 {
		return "Usage: mail send <player> <subject> | <message> [gold:<amount>]"
	}

	recipientName := headerParts[0]
	subject := strings.Join(headerParts[1:], " ")
	body := strings.TrimSpace(parts[1])

	// Check for gold attachment
	goldAmount := 0
	if idx := strings.LastIndex(body, "gold:"); idx != -1 {
		goldStr := strings.TrimSpace(body[idx+5:])
		body = strings.TrimSpace(body[:idx])
		if amount, err := strconv.Atoi(goldStr); err == nil && amount > 0 {
			goldAmount = amount
		}
	}

	// Look up recipient
	recipientID, err := db.GetCharacterIDByName(recipientName)
	if err != nil {
		logger.Error("Failed to look up recipient", "error", err, "recipient", recipientName)
		return "Failed to look up recipient."
	}
	if recipientID == 0 {
		return fmt.Sprintf("No player named '%s' exists.", recipientName)
	}

	if recipientID == p.GetCharacterID() {
		return "You cannot send mail to yourself."
	}

	// Check if recipient's mailbox is full
	mailCount, err := db.GetMailCount(recipientID)
	if err != nil {
		logger.Error("Failed to check recipient mailbox", "error", err, "recipient", recipientName)
		return "Failed to check recipient's mailbox."
	}
	if mailCount >= mail.MaxMailboxSize {
		return fmt.Sprintf("%s's mailbox is full. They need to delete some messages first.", recipientName)
	}

	// Validate input lengths
	if len(subject) > mail.MaxSubjectLen {
		return fmt.Sprintf("Subject too long. Maximum %d characters.", mail.MaxSubjectLen)
	}
	if len(body) > mail.MaxBodyLen {
		return fmt.Sprintf("Message too long. Maximum %d characters.", mail.MaxBodyLen)
	}

	// Check if player has enough gold to attach
	if goldAmount > 0 && p.GetGold() < goldAmount {
		return fmt.Sprintf("You don't have %d gold to attach. You have %d gold.", goldAmount, p.GetGold())
	}

	// Deduct attached gold from sender
	if goldAmount > 0 {
		p.SpendGold(goldAmount)
	}

	// Send the mail
	mailID, err := db.SendMail(p.GetCharacterID(), p.GetName(), recipientID, recipientName,
		subject, body, goldAmount, nil)
	if err != nil {
		// Refund on failure
		if goldAmount > 0 {
			p.AddGold(goldAmount)
		}
		logger.Error("Failed to send mail", "error", err, "sender", p.GetName(), "recipient", recipientName)
		return "Failed to send mail. Your gold has been refunded."
	}

	logger.Info("Mail sent",
		"mailID", mailID,
		"sender", p.GetName(),
		"recipient", recipientName,
		"gold", goldAmount)

	// Notify recipient if online
	if targetIface := server.FindPlayer(recipientName); targetIface != nil {
		if target, ok := targetIface.(PlayerInterface); ok {
			target.SendMessage(fmt.Sprintf("\nA faint shimmer of magic surrounds you as a letter from %s materializes in your mailbox.\n", p.GetName()))
		}
	}

	result := fmt.Sprintf("Mail sent to %s!", recipientName)
	if goldAmount > 0 {
		result += fmt.Sprintf(" Gold attached: %d. Remaining gold: %d.", goldAmount, p.GetGold())
	}

	return result
}

// executeMailCollect collects attachments from a mail message.
func executeMailCollect(idStr string, p PlayerInterface) string {
	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	if !room.HasFeature("mailbox") {
		return "You need to be at a mailbox to collect mail attachments."
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Translate user-facing index to actual mail ID
	mailID, _, errMsg := translateMailIndex(idStr, p, db)
	if errMsg != "" {
		return errMsg
	}

	m, err := db.GetMail(mailID, p.GetCharacterID())
	if err != nil {
		logger.Error("Failed to get mail", "error", err, "player", p.GetName(), "mailID", mailID)
		return "Failed to retrieve mail."
	}

	if m == nil {
		return "Mail not found."
	}

	if !m.HasAttachments() {
		return "This mail has no uncollected attachments."
	}

	var result strings.Builder
	result.WriteString("Collected:\n")

	// Collect gold
	if m.HasUncollectedGold() {
		goldAmount, err := db.CollectMailGold(mailID, p.GetCharacterID())
		if err != nil {
			logger.Error("Failed to collect mail gold", "error", err, "player", p.GetName(), "mailID", mailID)
			return "Failed to collect gold attachment."
		}
		p.AddGold(goldAmount)
		result.WriteString(fmt.Sprintf("  - %d gold\n", goldAmount))
		logger.Info("Mail gold collected",
			"player", p.GetName(),
			"mailID", mailID,
			"gold", goldAmount)
	}

	// Collect items
	if m.HasUncollectedItems() {
		itemIDs, err := db.CollectMailItems(mailID, p.GetCharacterID())
		if err != nil {
			logger.Error("Failed to collect mail items", "error", err, "player", p.GetName(), "mailID", mailID)
			return result.String() + "\nFailed to collect item attachments."
		}

		for _, itemID := range itemIDs {
			item := server.CreateItem(itemID)
			if item != nil {
				p.AddItem(item)
				result.WriteString(fmt.Sprintf("  - %s\n", item.Name))
				logger.Info("Mail item collected",
					"player", p.GetName(),
					"mailID", mailID,
					"item", item.Name)
			}
		}
	}

	result.WriteString(fmt.Sprintf("\nYour gold: %d", p.GetGold()))

	return result.String()
}

// executeMailDelete deletes a mail message.
func executeMailDelete(idStr string, p PlayerInterface) string {
	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	if !room.HasFeature("mailbox") {
		return "You need to be at a mailbox to delete mail."
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Translate user-facing index to actual mail ID
	mailID, _, errMsg := translateMailIndex(idStr, p, db)
	if errMsg != "" {
		return errMsg
	}

	err := db.DeleteMail(mailID, p.GetCharacterID())
	if err != nil {
		if strings.Contains(err.Error(), "collect all attachments") {
			return "You must collect all attachments before deleting this mail."
		}
		if strings.Contains(err.Error(), "not found") {
			return "Mail not found."
		}
		logger.Error("Failed to delete mail", "error", err, "player", p.GetName(), "mailID", mailID)
		return "Failed to delete mail."
	}

	logger.Info("Mail deleted", "player", p.GetName(), "mailID", mailID)
	return "Mail deleted."
}

// executeMailReply starts a reply to a mail message.
func executeMailReply(idStr string, p PlayerInterface) string {
	room, ok := GetRoom(p)
	if !ok {
		return "Internal error: invalid room"
	}

	if !room.HasFeature("mailbox") {
		return "You need to be at a mailbox to reply to mail."
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Translate user-facing index to actual mail ID
	mailID, _, errMsg := translateMailIndex(idStr, p, db)
	if errMsg != "" {
		return errMsg
	}

	m, err := db.GetMail(mailID, p.GetCharacterID())
	if err != nil {
		logger.Error("Failed to get mail", "error", err, "player", p.GetName(), "mailID", mailID)
		return "Failed to retrieve mail."
	}

	if m == nil {
		return "Mail not found."
	}

	// Generate reply subject
	subject := m.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	return fmt.Sprintf("To reply to %s, use:\n"+
		"mail send %s %s | <your message>\n\n"+
		"Example:\nmail send %s %s | Thanks for your message!",
		m.SenderName, m.SenderName, subject, m.SenderName, subject)
}

// Helper functions

func formatMailTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	return t.Format("Jan 2")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
