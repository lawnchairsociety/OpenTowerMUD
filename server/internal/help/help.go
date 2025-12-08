// Package help provides help text loading and lookup from YAML files.
package help

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Topic represents a single help topic with aliases and text.
type Topic struct {
	Aliases []string `yaml:"aliases"`
	Text    string   `yaml:"text"`
}

// HelpData represents the structure of the help.yaml file.
type HelpData struct {
	Topics      map[string]Topic `yaml:"topics"`
	GeneralHelp string           `yaml:"general_help"`
	AdminHelp   string           `yaml:"admin_help"`
}

// Help provides help text lookup.
type Help struct {
	data        *HelpData
	aliasLookup map[string]string // maps alias -> topic name
	mu          sync.RWMutex
}

var (
	instance *Help
	once     sync.Once
)

// Load loads help data from a YAML file.
func Load(path string) (*Help, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read help file: %w", err)
	}

	var helpData HelpData
	if err := yaml.Unmarshal(data, &helpData); err != nil {
		return nil, fmt.Errorf("failed to parse help file: %w", err)
	}

	h := &Help{
		data:        &helpData,
		aliasLookup: make(map[string]string),
	}

	// Build alias lookup map
	for topicName, topic := range helpData.Topics {
		for _, alias := range topic.Aliases {
			h.aliasLookup[strings.ToLower(alias)] = topicName
		}
	}

	return h, nil
}

// GetInstance returns the singleton help instance.
// Must call Initialize first.
func GetInstance() *Help {
	return instance
}

// Initialize loads the help data and sets the singleton instance.
func Initialize(path string) error {
	var err error
	once.Do(func() {
		instance, err = Load(path)
	})
	return err
}

// GetTopic returns help text for a given topic/alias.
// Returns empty string if topic not found.
func (h *Help) GetTopic(topic string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	topic = strings.ToLower(topic)

	// Look up by alias
	topicName, ok := h.aliasLookup[topic]
	if !ok {
		return ""
	}

	t, ok := h.data.Topics[topicName]
	if !ok {
		return ""
	}

	return strings.TrimSpace(t.Text)
}

// GetGeneralHelp returns the general help text.
func (h *Help) GetGeneralHelp() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return strings.TrimSpace(h.data.GeneralHelp)
}

// GetAdminHelp returns the admin help section.
func (h *Help) GetAdminHelp() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return strings.TrimSpace(h.data.AdminHelp)
}

// GetHelpText returns help for a topic, or general help if topic is empty.
// If isAdmin is true, admin commands are appended to general help.
func (h *Help) GetHelpText(topic string, isAdmin bool) string {
	if topic == "" {
		help := h.GetGeneralHelp()
		if isAdmin {
			help += "\n" + h.GetAdminHelp()
		}
		return help
	}

	text := h.GetTopic(topic)
	if text == "" {
		return fmt.Sprintf("No help available for '%s'.\nType 'help' for a list of commands.", topic)
	}
	return text
}
