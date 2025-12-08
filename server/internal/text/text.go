// Package text provides loading and lookup for externalized text blocks.
package text

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// TextData represents the structure of the text.yaml file.
type TextData struct {
	Welcome WelcomeText           `yaml:"welcome"`
	Guide   GuideText             `yaml:"guide"`
	Bard    BardText              `yaml:"bard"`
	Classes ClassText             `yaml:"classes"`
}

// WelcomeText contains welcome/login screen text.
type WelcomeText struct {
	Banner string `yaml:"banner"`
}

// GuideText contains tutorial guide (Aldric) dialogue.
type GuideText struct {
	Greeting string `yaml:"greeting"`
	Tower    string `yaml:"tower"`
	Combat   string `yaml:"combat"`
	Save     string `yaml:"save"`
	Shop     string `yaml:"shop"`
	Portal   string `yaml:"portal"`
	Quests   string `yaml:"quests"`
	Commands string `yaml:"commands"`
}

// BardText contains bard interaction templates.
type BardText struct {
	Song string `yaml:"song"`
}

// ClassText contains class-related text.
type ClassText struct {
	Abilities           map[string]string `yaml:"abilities"`
	Welcome             map[string]string `yaml:"welcome"`
	TrainerAccept       map[string]string `yaml:"trainer_accept"`
	TrainerReject       map[string]string `yaml:"trainer_reject"`
	StatRecommendations map[string]string `yaml:"stat_recommendations"`
}

// Text provides text lookup functionality.
type Text struct {
	data *TextData
	mu   sync.RWMutex
}

var (
	instance *Text
	once     sync.Once
)

// Load loads text data from a YAML file.
func Load(path string) (*Text, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	var textData TextData
	if err := yaml.Unmarshal(data, &textData); err != nil {
		return nil, fmt.Errorf("failed to parse text file: %w", err)
	}

	return &Text{data: &textData}, nil
}

// GetInstance returns the singleton text instance.
// Must call Initialize first.
func GetInstance() *Text {
	return instance
}

// Initialize loads the text data and sets the singleton instance.
func Initialize(path string) error {
	var err error
	once.Do(func() {
		instance, err = Load(path)
	})
	return err
}

// GetWelcomeBanner returns the welcome banner text.
func (t *Text) GetWelcomeBanner() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return strings.TrimSpace(t.data.Welcome.Banner)
}

// GetGuideGreeting returns the guide's greeting template.
func (t *Text) GetGuideGreeting() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return strings.TrimSpace(t.data.Guide.Greeting)
}

// GetGuideTopic returns the guide's topic text by name.
func (t *Text) GetGuideTopic(topic string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var text string
	switch strings.ToLower(topic) {
	case "tower":
		text = t.data.Guide.Tower
	case "combat":
		text = t.data.Guide.Combat
	case "save":
		text = t.data.Guide.Save
	case "shop":
		text = t.data.Guide.Shop
	case "portal":
		text = t.data.Guide.Portal
	case "quests":
		text = t.data.Guide.Quests
	case "commands":
		text = t.data.Guide.Commands
	default:
		return ""
	}
	return strings.TrimSpace(text)
}

// GetBardSong returns the bard song template.
func (t *Text) GetBardSong() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return strings.TrimSpace(t.data.Bard.Song)
}

// GetClassAbilities returns the class abilities preview text.
func (t *Text) GetClassAbilities(className string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text, ok := t.data.Classes.Abilities[strings.ToLower(className)]
	if !ok {
		return "  No special abilities defined."
	}
	return strings.TrimSpace(text)
}

// GetClassWelcome returns the class welcome message when learning a new class.
func (t *Text) GetClassWelcome(className string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text, ok := t.data.Classes.Welcome[strings.ToLower(className)]
	if !ok {
		return fmt.Sprintf("You have learned the ways of the %s.", strings.Title(className))
	}
	return strings.TrimSpace(text)
}

// GetTrainerAccept returns the trainer acceptance dialogue.
func (t *Text) GetTrainerAccept(className string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text, ok := t.data.Classes.TrainerAccept[strings.ToLower(className)]
	if !ok {
		return "%s nods approvingly.\n\n\"You have begun your training as a %s, %s.\""
	}
	return strings.TrimSpace(text)
}

// GetTrainerReject returns the trainer rejection dialogue.
func (t *Text) GetTrainerReject(className string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text, ok := t.data.Classes.TrainerReject[strings.ToLower(className)]
	if !ok {
		return "You do not meet the requirements to learn this class."
	}
	return strings.TrimSpace(text)
}

// GetStatRecommendation returns the stat recommendation for character creation.
func (t *Text) GetStatRecommendation(className string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	text, ok := t.data.Classes.StatRecommendations[strings.ToLower(className)]
	if !ok {
		return "  No specific recommendations."
	}
	return "  " + strings.TrimSpace(text)
}
