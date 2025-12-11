package npc

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"gopkg.in/yaml.v3"
)

// LootEntryYAML represents a loot entry in YAML format
type LootEntryYAML struct {
	Item   string  `yaml:"item"`   // Item name/ID
	Chance float64 `yaml:"chance"` // Drop chance percentage (0-100)
}

// ShopItemYAML represents an item for sale in YAML format
type ShopItemYAML struct {
	Item  string `yaml:"item"`  // Item name/ID
	Price int    `yaml:"price"` // Price in gold (0 = use item's base value)
}

// NPCDefinition represents an NPC definition from the YAML file
type NPCDefinition struct {
	Name             string          `yaml:"name"`
	Description      string          `yaml:"description"`
	Level            int             `yaml:"level"`
	Health           int             `yaml:"health"`
	Damage           int             `yaml:"damage"`
	Armor            int             `yaml:"armor"`
	Experience       int             `yaml:"experience"`
	Aggressive       bool            `yaml:"aggressive"`
	Attackable       bool            `yaml:"attackable"`
	GoldMin          int             `yaml:"gold_min"`       // Minimum gold dropped on death
	GoldMax          int             `yaml:"gold_max"`       // Maximum gold dropped on death
	LootTable        []LootEntryYAML `yaml:"loot_table"`     // Percentage-based loot entries
	ShopInventory    []ShopItemYAML  `yaml:"shop_inventory"` // Items this NPC sells
	Dialogue         []string        `yaml:"dialogue"`       // Lines the NPC can say when talked to
	Tier             int             `yaml:"tier"`           // Mob tier (1=easy, 2=medium, 3=hard, 4=elite)
	Boss             bool            `yaml:"boss"`           // Is this a boss mob?
	MobType          string          `yaml:"mob_type"`       // Creature type: beast, undead, humanoid, demon, construct, giant
	TrainerClass     string          `yaml:"trainer_class"`     // Class this NPC trains (warrior, mage, cleric, rogue, ranger, paladin)
	CraftingTrainer  string          `yaml:"crafting_trainer"`  // Crafting skill this NPC teaches (blacksmithing, leatherworking, alchemy, enchanting)
	TeachesRecipes   []string        `yaml:"teaches_recipes"`   // Recipe IDs this NPC can teach
	QuestGiver       bool            `yaml:"quest_giver"`       // Can this NPC give quests?
	GivesQuests      []string        `yaml:"gives_quests"`      // Quest IDs this NPC can give
	TurnInQuests     []string        `yaml:"turn_in_quests"`    // Quest IDs that can be turned in to this NPC
	LoreNPC          bool            `yaml:"lore_npc"`          // Is this a labyrinth lore NPC?
	Locations        []string        `yaml:"locations"`         // Room IDs where this NPC spawns
	RespawnMedian    int             `yaml:"respawn_median"`    // Median respawn time in seconds
	RespawnVariation int             `yaml:"respawn_variation"` // Variation in respawn time (+/- seconds)
	TowerTags        []string        `yaml:"tower_tags"`        // Tower tags for themed spawning (e.g., "shared", "human", "arcane")
}

// NPCsConfig represents the structure of the npcs.yaml file
type NPCsConfig struct {
	NPCs map[string]NPCDefinition `yaml:"npcs"`
}

// LoadNPCsFromYAML loads NPC definitions from a YAML file
func LoadNPCsFromYAML(filename string) (*NPCsConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read NPCs file: %w", err)
	}

	var config NPCsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse NPCs YAML: %w", err)
	}

	// Validate NPC configurations
	for npcID, def := range config.NPCs {
		// Aggressive NPCs must be attackable (can't have auto-attacking un-attackable NPCs)
		if def.Aggressive && !def.Attackable {
			logger.Warning("NPC auto-correction applied",
				"npc_name", def.Name,
				"npc_id", npcID,
				"issue", "aggressive=true but attackable=false",
				"action", "set attackable=true")
			def.Attackable = true
			config.NPCs[npcID] = def // Update the map with corrected value
		}
	}

	return &config, nil
}

// CreateNPCFromDefinition creates an NPC from an NPCDefinition
func CreateNPCFromDefinition(def NPCDefinition, roomID string) *NPC {
	npc := NewNPC(
		def.Name,
		def.Description,
		def.Level,
		def.Health,
		def.Damage,
		def.Armor,
		def.Experience,
		def.Aggressive,
		def.Attackable,
		roomID,
		def.RespawnMedian,
		def.RespawnVariation,
	)
	if def.GoldMin > 0 || def.GoldMax > 0 {
		npc.SetGoldDrop(def.GoldMin, def.GoldMax)
	}
	if len(def.Dialogue) > 0 {
		npc.SetDialogue(def.Dialogue)
	}
	// Set mob type for class bonuses (favored enemy, smite, etc.)
	// and set flee threshold based on mob type
	mobType := StringToMobType(def.MobType)
	if def.MobType != "" {
		npc.SetMobType(mobType)
	}
	// Set flee threshold based on mob type (bosses handled in ShouldFlee)
	npc.SetFleeThreshold(GetDefaultFleeThreshold(mobType))
	// Set trainer class for multiclassing NPCs
	if def.TrainerClass != "" {
		npc.SetTrainerClass(def.TrainerClass)
	}
	// Set crafting trainer info
	if def.CraftingTrainer != "" {
		npc.SetCraftingTrainer(def.CraftingTrainer)
	}
	if len(def.TeachesRecipes) > 0 {
		npc.SetTeachesRecipes(def.TeachesRecipes)
	}
	// Set quest giver info
	if def.QuestGiver || len(def.GivesQuests) > 0 {
		npc.SetQuestGiver(true)
	}
	if len(def.GivesQuests) > 0 {
		npc.SetGivesQuests(def.GivesQuests)
	}
	if len(def.TurnInQuests) > 0 {
		npc.SetTurnInQuests(def.TurnInQuests)
	}
	// Convert YAML loot table to NPC loot table
	if len(def.LootTable) > 0 {
		lootTable := make([]LootEntry, len(def.LootTable))
		for i, entry := range def.LootTable {
			lootTable[i] = LootEntry{
				ItemName:   entry.Item,
				DropChance: entry.Chance,
			}
		}
		npc.SetLootTable(lootTable)
	}
	// Convert YAML shop inventory to NPC shop inventory
	if len(def.ShopInventory) > 0 {
		shopInventory := make([]ShopItem, len(def.ShopInventory))
		for i, entry := range def.ShopInventory {
			shopInventory[i] = ShopItem{
				ItemName: entry.Item,
				Price:    entry.Price,
			}
		}
		npc.SetShopInventory(shopInventory)
	}
	// Set lore NPC flag for labyrinth lore NPCs
	if def.LoreNPC {
		npc.SetLoreNPC(true)
	}
	return npc
}

// CreateNPCFromDefinitionWithID creates an NPC from an NPCDefinition and stores the definition ID
func CreateNPCFromDefinitionWithID(npcID string, def NPCDefinition, roomID string) *NPC {
	npc := CreateNPCFromDefinition(def, roomID)
	npc.SetNPCID(npcID)
	return npc
}

// StringToMobType converts a string to a MobType
func StringToMobType(s string) MobType {
	switch s {
	case "beast":
		return MobTypeBeast
	case "humanoid":
		return MobTypeHumanoid
	case "undead":
		return MobTypeUndead
	case "demon":
		return MobTypeDemon
	case "construct":
		return MobTypeConstruct
	case "giant":
		return MobTypeGiant
	default:
		return MobTypeUnknown
	}
}

// GetNPCsByLocation returns a map of room IDs to NPCs that should spawn there
func (config *NPCsConfig) GetNPCsByLocation() map[string][]*NPC {
	npcsByLocation := make(map[string][]*NPC)

	for npcID, def := range config.NPCs {
		for _, location := range def.Locations {
			npc := CreateNPCFromDefinitionWithID(npcID, def, location)
			npcsByLocation[location] = append(npcsByLocation[location], npc)
		}
	}

	return npcsByLocation
}

// Merge combines another NPCsConfig into this one
func (config *NPCsConfig) Merge(other *NPCsConfig) {
	if other == nil {
		return
	}
	for id, def := range other.NPCs {
		config.NPCs[id] = def
	}
}

// LoadMultipleNPCFiles loads and merges NPC definitions from multiple YAML files
func LoadMultipleNPCFiles(filenames ...string) (*NPCsConfig, error) {
	merged := &NPCsConfig{
		NPCs: make(map[string]NPCDefinition),
	}

	for _, filename := range filenames {
		config, err := LoadNPCsFromYAML(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filename, err)
		}
		merged.Merge(config)
	}

	return merged, nil
}

// LoadNPCsFromDirectory loads and merges all YAML files from a directory
func LoadNPCsFromDirectory(dir string) (*NPCsConfig, error) {
	merged := &NPCsConfig{
		NPCs: make(map[string]NPCDefinition),
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
		config, err := LoadNPCsFromYAML(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filePath, err)
		}
		merged.Merge(config)
		fileCount++
		logger.Info("Loaded NPC file", "path", filePath, "npcs", len(config.NPCs))
	}

	logger.Info("Loaded NPCs from directory", "dir", dir, "files", fileCount, "total_npcs", len(merged.NPCs))
	return merged, nil
}

// LoadNPCsFromDirectories loads and merges all YAML files from multiple directories
func LoadNPCsFromDirectories(dirs ...string) (*NPCsConfig, error) {
	merged := &NPCsConfig{
		NPCs: make(map[string]NPCDefinition),
	}

	for _, dir := range dirs {
		config, err := LoadNPCsFromDirectory(dir)
		if err != nil {
			return nil, err
		}
		merged.Merge(config)
	}

	return merged, nil
}

// GetMobsByTier returns all non-boss mob definitions for a given tier
func (config *NPCsConfig) GetMobsByTier(tier int) []NPCDefinition {
	var mobs []NPCDefinition
	for _, def := range config.NPCs {
		if def.Tier == tier && !def.Boss && def.Attackable {
			mobs = append(mobs, def)
		}
	}
	return mobs
}

// GetBossesByTier returns all boss mob definitions for a given tier
func (config *NPCsConfig) GetBossesByTier(tier int) []NPCDefinition {
	var bosses []NPCDefinition
	for _, def := range config.NPCs {
		if def.Tier == tier && def.Boss {
			bosses = append(bosses, def)
		}
	}
	return bosses
}

// GetRandomMobForTier returns a random non-boss mob definition for the given tier
// If no mobs exist for the tier, returns a mob from the closest lower tier
func (config *NPCsConfig) GetRandomMobForTier(tier int, rng *rand.Rand) *NPCDefinition {
	for t := tier; t >= 1; t-- {
		mobs := config.GetMobsByTier(t)
		if len(mobs) > 0 {
			mob := mobs[rng.Intn(len(mobs))]
			return &mob
		}
	}
	return nil
}

// GetRandomBossForTier returns a random boss mob definition for the given tier
// If no bosses exist for the tier, returns a boss from the closest lower tier
func (config *NPCsConfig) GetRandomBossForTier(tier int, rng *rand.Rand) *NPCDefinition {
	for t := tier; t >= 1; t-- {
		bosses := config.GetBossesByTier(t)
		if len(bosses) > 0 {
			boss := bosses[rng.Intn(len(bosses))]
			return &boss
		}
	}
	return nil
}

// mobMatchesTags returns true if a mob definition matches any of the given tags.
// Mobs with empty TowerTags spawn everywhere (backward compatible).
func mobMatchesTags(def NPCDefinition, tags []string) bool {
	// Empty tower_tags = spawns everywhere (backward compatible)
	if len(def.TowerTags) == 0 {
		return true
	}
	// Empty filter tags = match nothing specific
	if len(tags) == 0 {
		return false
	}
	for _, mobTag := range def.TowerTags {
		for _, filterTag := range tags {
			if mobTag == filterTag {
				return true
			}
		}
	}
	return false
}

// GetMobsByTierAndTags returns all non-boss mob definitions for a given tier
// that match at least one of the provided tags.
func (config *NPCsConfig) GetMobsByTierAndTags(tier int, tags []string) []NPCDefinition {
	var mobs []NPCDefinition
	for _, def := range config.NPCs {
		if def.Tier == tier && !def.Boss && def.Attackable && mobMatchesTags(def, tags) {
			mobs = append(mobs, def)
		}
	}
	return mobs
}

// GetBossesByTierAndTags returns all boss mob definitions for a given tier
// that match at least one of the provided tags.
func (config *NPCsConfig) GetBossesByTierAndTags(tier int, tags []string) []NPCDefinition {
	var bosses []NPCDefinition
	for _, def := range config.NPCs {
		if def.Tier == tier && def.Boss && mobMatchesTags(def, tags) {
			bosses = append(bosses, def)
		}
	}
	return bosses
}

// GetRandomMobForTierAndTags returns a random non-boss mob definition for the given tier
// that matches at least one of the provided tags.
// If no mobs exist for the tier, returns a mob from the closest lower tier.
func (config *NPCsConfig) GetRandomMobForTierAndTags(tier int, tags []string, rng *rand.Rand) *NPCDefinition {
	for t := tier; t >= 1; t-- {
		mobs := config.GetMobsByTierAndTags(t, tags)
		if len(mobs) > 0 {
			mob := mobs[rng.Intn(len(mobs))]
			return &mob
		}
	}
	return nil
}

// GetRandomBossForTierAndTags returns a random boss mob definition for the given tier
// that matches at least one of the provided tags.
// If no bosses exist for the tier, returns a boss from the closest lower tier.
func (config *NPCsConfig) GetRandomBossForTierAndTags(tier int, tags []string, rng *rand.Rand) *NPCDefinition {
	for t := tier; t >= 1; t-- {
		bosses := config.GetBossesByTierAndTags(t, tags)
		if len(bosses) > 0 {
			boss := bosses[rng.Intn(len(bosses))]
			return &boss
		}
	}
	return nil
}
