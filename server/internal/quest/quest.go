package quest

// QuestType defines the type of objective
type QuestType string

const (
	QuestTypeKill     QuestType = "kill"     // Defeat enemies
	QuestTypeFetch    QuestType = "fetch"    // Collect items
	QuestTypeDelivery QuestType = "delivery" // Transport items to NPCs
	QuestTypeExplore  QuestType = "explore"  // Visit specific rooms
	QuestTypeCraft    QuestType = "craft"    // Create items
	QuestTypeCast     QuestType = "cast"     // Cast spells
)

// QuestCategory defines the category of quest
type QuestCategory string

const (
	QuestCategoryMain     QuestCategory = "main"     // Main story quests
	QuestCategorySide     QuestCategory = "side"     // Optional side quests
	QuestCategoryClass    QuestCategory = "class"    // Class-specific quests
	QuestCategoryCrafting QuestCategory = "crafting" // Crafting skill quests
)

// QuestObjective represents a single objective within a quest
type QuestObjective struct {
	Type       QuestType // What type of objective
	Target     string    // mob ID, item ID, room ID, spell ID (empty = any)
	TargetName string    // Display name for the target
	Required   int       // How many needed (kills, items, visits, casts)
}

// QuestReward defines what the player receives on completion
type QuestReward struct {
	Gold       int      // Gold awarded
	Experience int      // XP awarded
	Items      []string // Item IDs to grant
	Recipes    []string // Recipe IDs to unlock
	Title      string   // Title to award (empty = none)
}

// Quest represents a quest definition
type Quest struct {
	ID          string           // Unique identifier (e.g., "pest_control", "warrior_level_05")
	Name        string           // Display name
	Description string           // Full description
	Category    QuestCategory    // main, side, class, or crafting
	GiverNPC    string           // NPC ID who gives this quest
	TurnInNPC   string           // NPC ID to turn in (often same as giver)
	Objectives  []QuestObjective // What must be completed
	Rewards     QuestReward      // What player receives
	QuestItems  []string         // Items given on accept (for delivery quests)

	// Requirements
	MinLevel              int      // Minimum player level to accept
	Prereqs               []string // Quest IDs that must be completed first
	RequiredClass         string   // For class quests (e.g., "warrior")
	RequiredClassLevel    int      // Class level needed (e.g., 5, 10, 15)
	RequiredCraftingSkill string   // For crafting quests (e.g., "blacksmithing")
	RequiredCraftingLevel int      // Skill level needed (e.g., 10, 20, 30)

	// Flags
	Repeatable bool // Can be done multiple times
}

// IsClassQuest returns true if this is a class-specific quest
func (q *Quest) IsClassQuest() bool {
	return q.Category == QuestCategoryClass && q.RequiredClass != ""
}

// IsCraftingQuest returns true if this is a crafting skill quest
func (q *Quest) IsCraftingQuest() bool {
	return q.Category == QuestCategoryCrafting && q.RequiredCraftingSkill != ""
}

// HasQuestItems returns true if quest gives items on accept
func (q *Quest) HasQuestItems() bool {
	return len(q.QuestItems) > 0
}

// HasPrereqs returns true if quest has prerequisite quests
func (q *Quest) HasPrereqs() bool {
	return len(q.Prereqs) > 0
}

// GetObjectiveCount returns the number of objectives
func (q *Quest) GetObjectiveCount() int {
	return len(q.Objectives)
}
