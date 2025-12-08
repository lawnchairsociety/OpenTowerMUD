// Package mail provides player-to-player mail functionality with gold and item attachments.
package mail

import "time"

// Mail represents a single mail message with optional attachments.
type Mail struct {
	ID             int64
	SenderID       int64
	SenderName     string
	RecipientID    int64
	RecipientName  string
	Subject        string
	Body           string
	GoldAttached   int
	GoldCollected  bool
	ItemsCollected bool
	Read           bool
	SentAt         time.Time
	Items          []MailItem
}

// MailItem represents an item attached to a mail message.
type MailItem struct {
	ID        int64
	MailID    int64
	ItemID    string // References items.yaml
	Collected bool
}

// MailSummary is a lightweight view of mail for listing.
type MailSummary struct {
	ID         int64
	SenderName string
	Subject    string
	HasGold    bool
	HasItems   bool
	Read       bool
	SentAt     time.Time
}

// Mail system constants.
const (
	MaxItemsPerMail = 5    // Maximum items that can be attached
	MaxMailboxSize  = 50   // Maximum messages per player
	MaxSubjectLen   = 100  // Maximum subject length
	MaxBodyLen      = 2000 // Maximum message body length
)

// HasAttachments returns true if the mail has any uncollected attachments.
func (m *Mail) HasAttachments() bool {
	return (m.GoldAttached > 0 && !m.GoldCollected) || (len(m.Items) > 0 && !m.ItemsCollected)
}

// HasUncollectedGold returns true if there's gold that hasn't been collected.
func (m *Mail) HasUncollectedGold() bool {
	return m.GoldAttached > 0 && !m.GoldCollected
}

// HasUncollectedItems returns true if there are items that haven't been collected.
func (m *Mail) HasUncollectedItems() bool {
	if m.ItemsCollected {
		return false
	}
	for _, item := range m.Items {
		if !item.Collected {
			return true
		}
	}
	return false
}
