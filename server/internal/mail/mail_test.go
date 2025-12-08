package mail

import "testing"

func TestMailHasAttachments(t *testing.T) {
	tests := []struct {
		name     string
		mail     Mail
		expected bool
	}{
		{
			name:     "no attachments",
			mail:     Mail{},
			expected: false,
		},
		{
			name: "uncollected gold",
			mail: Mail{
				GoldAttached:  100,
				GoldCollected: false,
			},
			expected: true,
		},
		{
			name: "collected gold",
			mail: Mail{
				GoldAttached:  100,
				GoldCollected: true,
			},
			expected: false,
		},
		{
			name: "uncollected items",
			mail: Mail{
				Items: []MailItem{
					{ItemID: "sword", Collected: false},
				},
			},
			expected: true,
		},
		{
			name: "collected items",
			mail: Mail{
				ItemsCollected: true,
				Items: []MailItem{
					{ItemID: "sword", Collected: true},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mail.HasAttachments()
			if result != tt.expected {
				t.Errorf("HasAttachments() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
