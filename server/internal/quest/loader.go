package quest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"gopkg.in/yaml.v3"
)

// QuestObjectiveYAML for YAML parsing
type QuestObjectiveYAML struct {
	Type       string `yaml:"type"`        // kill, fetch, delivery, explore, craft, cast
	Target     string `yaml:"target"`      // ID of target (empty = any)
	TargetName string `yaml:"target_name"` // Display name
	Required   int    `yaml:"required"`    // Amount needed
}

// QuestRewardYAML for YAML parsing
type QuestRewardYAML struct {
	Gold       int      `yaml:"gold"`
	Experience int      `yaml:"experience"`
	Items      []string `yaml:"items"`
	Recipes    []string `yaml:"recipes"`
	Title      string   `yaml:"title"`
}

// QuestDefinition for YAML parsing
type QuestDefinition struct {
	Name                  string               `yaml:"name"`
	Description           string               `yaml:"description"`
	Category              string               `yaml:"category"` // main, side, class, crafting
	GiverNPC              string               `yaml:"giver_npc"`
	TurnInNPC             string               `yaml:"turn_in_npc"`
	Objectives            []QuestObjectiveYAML `yaml:"objectives"`
	Rewards               QuestRewardYAML      `yaml:"rewards"`
	QuestItems            []string             `yaml:"quest_items"`
	MinLevel              int                  `yaml:"min_level"`
	Prereqs               []string             `yaml:"prereqs"`
	RequiredClass         string               `yaml:"required_class"`
	RequiredClassLevel    int                  `yaml:"required_class_level"`
	RequiredCraftingSkill string               `yaml:"required_crafting_skill"`
	RequiredCraftingLevel int                  `yaml:"required_crafting_level"`
	Repeatable            bool                 `yaml:"repeatable"`
}

// QuestsConfig represents the quests.yaml structure
type QuestsConfig struct {
	Quests map[string]QuestDefinition `yaml:"quests"`
}

// LoadQuestsFromYAML loads quest definitions from YAML file
func LoadQuestsFromYAML(filename string) (*QuestsConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read quests file: %w", err)
	}

	var config QuestsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse quests YAML: %w", err)
	}

	return &config, nil
}

// GetQuestByID returns a Quest struct from the config
func (config *QuestsConfig) GetQuestByID(id string) (*Quest, bool) {
	def, exists := config.Quests[id]
	if !exists {
		return nil, false
	}

	return createQuestFromDefinition(id, &def), true
}

// GetAllQuests returns all quests from the config
func (config *QuestsConfig) GetAllQuests() []*Quest {
	quests := make([]*Quest, 0, len(config.Quests))
	for id, def := range config.Quests {
		quests = append(quests, createQuestFromDefinition(id, &def))
	}
	return quests
}

// createQuestFromDefinition converts a YAML definition to a Quest struct
func createQuestFromDefinition(id string, def *QuestDefinition) *Quest {
	// Convert objectives
	objectives := make([]QuestObjective, len(def.Objectives))
	for i, objDef := range def.Objectives {
		objectives[i] = QuestObjective{
			Type:       parseQuestType(objDef.Type),
			Target:     objDef.Target,
			TargetName: objDef.TargetName,
			Required:   objDef.Required,
		}
	}

	// Convert rewards
	rewards := QuestReward{
		Gold:       def.Rewards.Gold,
		Experience: def.Rewards.Experience,
		Items:      def.Rewards.Items,
		Recipes:    def.Rewards.Recipes,
		Title:      def.Rewards.Title,
	}

	// Ensure slices are not nil
	if rewards.Items == nil {
		rewards.Items = []string{}
	}
	if rewards.Recipes == nil {
		rewards.Recipes = []string{}
	}

	questItems := def.QuestItems
	if questItems == nil {
		questItems = []string{}
	}

	prereqs := def.Prereqs
	if prereqs == nil {
		prereqs = []string{}
	}

	return &Quest{
		ID:                    id,
		Name:                  def.Name,
		Description:           def.Description,
		Category:              parseQuestCategory(def.Category),
		GiverNPC:              def.GiverNPC,
		TurnInNPC:             def.TurnInNPC,
		Objectives:            objectives,
		Rewards:               rewards,
		QuestItems:            questItems,
		MinLevel:              def.MinLevel,
		Prereqs:               prereqs,
		RequiredClass:         def.RequiredClass,
		RequiredClassLevel:    def.RequiredClassLevel,
		RequiredCraftingSkill: def.RequiredCraftingSkill,
		RequiredCraftingLevel: def.RequiredCraftingLevel,
		Repeatable:            def.Repeatable,
	}
}

// parseQuestType converts string to QuestType
func parseQuestType(s string) QuestType {
	switch s {
	case "kill":
		return QuestTypeKill
	case "fetch":
		return QuestTypeFetch
	case "delivery":
		return QuestTypeDelivery
	case "explore":
		return QuestTypeExplore
	case "craft":
		return QuestTypeCraft
	case "cast":
		return QuestTypeCast
	default:
		return QuestTypeKill // Default fallback
	}
}

// parseQuestCategory converts string to QuestCategory
func parseQuestCategory(s string) QuestCategory {
	switch s {
	case "main":
		return QuestCategoryMain
	case "side":
		return QuestCategorySide
	case "class":
		return QuestCategoryClass
	case "crafting":
		return QuestCategoryCrafting
	default:
		return QuestCategorySide // Default fallback
	}
}

// Merge combines another QuestsConfig into this one
func (config *QuestsConfig) Merge(other *QuestsConfig) {
	if other == nil {
		return
	}
	for id, def := range other.Quests {
		config.Quests[id] = def
	}
}

// LoadQuestsFromDirectory loads and merges all YAML files from a directory
func LoadQuestsFromDirectory(dir string) (*QuestsConfig, error) {
	merged := &QuestsConfig{
		Quests: make(map[string]QuestDefinition),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	fileCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		filePath := filepath.Join(dir, name)
		config, err := LoadQuestsFromYAML(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filePath, err)
		}
		merged.Merge(config)
		fileCount++
		logger.Info("Loaded quest file", "path", filePath, "quests", len(config.Quests))
	}

	logger.Info("Loaded quests from directory", "dir", dir, "files", fileCount, "total_quests", len(merged.Quests))
	return merged, nil
}
