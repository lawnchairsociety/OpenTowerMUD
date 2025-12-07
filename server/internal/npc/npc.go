package npc

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// LootEntry represents an item that can drop with a percentage chance
type LootEntry struct {
	ItemName   string  // Name/ID of the item to drop
	DropChance float64 // Percentage chance to drop (0.0 to 100.0)
}

// ShopItem represents an item for sale by an NPC merchant
type ShopItem struct {
	ItemName string // Name/ID of the item to sell
	Price    int    // Price in gold (0 = use item's base value)
}

// MobType represents the creature type for class bonuses (favored enemy, smite)
type MobType string

const (
	MobTypeUnknown   MobType = ""
	MobTypeBeast     MobType = "beast"
	MobTypeHumanoid  MobType = "humanoid"
	MobTypeUndead    MobType = "undead"
	MobTypeDemon     MobType = "demon"
	MobTypeConstruct MobType = "construct"
	MobTypeGiant     MobType = "giant"
)

// NPC represents a non-player character or monster
type NPC struct {
	Name             string
	Description      string
	Level            int
	Health           int
	MaxHealth        int
	Damage           int
	Armor            int
	Experience       int             // XP awarded on death
	GoldMin          int             // Minimum gold dropped on death
	GoldMax          int             // Maximum gold dropped on death
	Aggressive       bool            // Auto-attack players?
	Attackable       bool            // Can players attack this NPC?
	LootTable        []LootEntry     // Item drops with percentage chances
	ShopInventory    []ShopItem      // Items this NPC sells
	Dialogue         []string        // Lines the NPC can say when talked to
	RoomID           string          // Current location
	InCombat         bool            // Is this NPC currently fighting?
	Targets          map[string]bool // Names of players being fought
	ThreatTable      map[string]int  // Threat per player (for target selection)
	RespawnMedian    int             // Median respawn time in seconds (0 = no respawn)
	RespawnVariation int             // Variation in respawn time (+/- seconds)
	OriginalRoomID   string          // Room where NPC originally spawned
	DeathTime        time.Time       // When this NPC died
	RespawnTime      time.Time       // When this NPC should respawn
	StunEndTime      time.Time       // When stun effect expires
	RootEndTime      time.Time       // When root effect expires (prevents fleeing)
	FleeThreshold    float64         // HP percentage at which mob will flee (0.0-1.0, 0 = never)
	IsBoss           bool            // Is this a boss mob?
	Floor            int             // Tower floor this mob is on (for boss key drops)
	MobType          MobType         // Creature type for class bonuses
	TrainerClass     string          // Class this NPC trains (for multiclassing)
	CraftingTrainer  string          // Crafting skill this NPC teaches (blacksmithing, alchemy, etc.)
	TeachesRecipes   []string        // Recipe IDs this NPC can teach
	QuestGiver       bool            // Can this NPC give quests?
	GivesQuests      []string        // Quest IDs this NPC can give
	TurnInQuests     []string        // Quest IDs that can be turned in to this NPC
	mu               sync.RWMutex
}

// NewNPC creates a new NPC with the given properties
func NewNPC(name, description string, level, health, damage, armor, experience int, aggressive, attackable bool, roomID string, respawnMedian, respawnVariation int) *NPC {
	return &NPC{
		Name:             name,
		Description:      description,
		Level:            level,
		Health:           health,
		MaxHealth:        health,
		Damage:           damage,
		Armor:            armor,
		Experience:       experience,
		Aggressive:       aggressive,
		Attackable:       attackable,
		RoomID:           roomID,
		OriginalRoomID:   roomID, // Track original spawn location
		InCombat:         false,
		Targets:          make(map[string]bool),
		ThreatTable:      make(map[string]int),
		RespawnMedian:    respawnMedian,
		RespawnVariation: respawnVariation,
	}
}

// GetName returns the NPC's name
func (n *NPC) GetName() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Name
}

// GetDescription returns the NPC's description
func (n *NPC) GetDescription() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Description
}

// GetLevel returns the NPC's level
func (n *NPC) GetLevel() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Level
}

// GetHealth returns the NPC's current health
func (n *NPC) GetHealth() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Health
}

// GetMaxHealth returns the NPC's maximum health
func (n *NPC) GetMaxHealth() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.MaxHealth
}

// IsAlive returns true if the NPC has health remaining
func (n *NPC) IsAlive() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Health > 0
}

// IsInCombat returns true if the NPC is currently fighting
func (n *NPC) IsInCombat() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.InCombat
}

// GetTargets returns the names of all players this NPC is fighting
func (n *NPC) GetTargets() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	targets := make([]string, 0, len(n.Targets))
	for name := range n.Targets {
		targets = append(targets, name)
	}
	return targets
}

// StartCombat adds a player to this NPC's combat targets
func (n *NPC) StartCombat(targetName string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.InCombat = true
	n.Targets[targetName] = true
}

// EndCombat removes a player from combat, or clears all if targetName is empty
func (n *NPC) EndCombat(targetName string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if targetName == "" {
		// Clear all targets and threat
		n.Targets = make(map[string]bool)
		n.ThreatTable = make(map[string]int)
		n.InCombat = false
	} else {
		// Remove specific target and their threat
		delete(n.Targets, targetName)
		delete(n.ThreatTable, targetName)
		if len(n.Targets) == 0 {
			n.InCombat = false
		}
	}
}

// TakeDamage applies damage to the NPC and returns actual damage taken
func (n *NPC) TakeDamage(damage int) int {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Apply armor reduction
	actualDamage := damage - n.Armor
	if actualDamage < 1 {
		actualDamage = 1 // Minimum 1 damage
	}

	n.Health -= actualDamage
	if n.Health < 0 {
		n.Health = 0
	}

	return actualDamage
}

// TakeMagicDamage applies magic damage to the NPC (bypasses armor)
func (n *NPC) TakeMagicDamage(damage int) int {
	n.mu.Lock()
	defer n.mu.Unlock()

	if damage < 1 {
		damage = 1 // Minimum 1 damage
	}

	n.Health -= damage
	if n.Health < 0 {
		n.Health = 0
	}

	return damage
}

// GetAttackDamage returns the damage this NPC deals in combat
func (n *NPC) GetAttackDamage() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Damage
}

// GetArmorClass returns the NPC's armor class (10 + armor bonus)
// This is the target number for attack rolls
func (n *NPC) GetArmorClass() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return 10 + n.Armor
}

// RollLoot performs percentage-based loot rolls and returns items that dropped
// For bosses, all loot drops (100% chance). For regular mobs, each item is rolled independently.
func (n *NPC) RollLoot() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var dropped []string

	// If this is a boss, drop everything (100% chance)
	if n.IsBoss {
		for _, entry := range n.LootTable {
			dropped = append(dropped, entry.ItemName)
		}
		return dropped
	}

	// For regular mobs, roll each item in the loot table
	for _, entry := range n.LootTable {
		// Roll percentage (0-100)
		roll := rand.Float64() * 100.0
		if roll < entry.DropChance {
			dropped = append(dropped, entry.ItemName)
		}
	}

	return dropped
}

// SetLootTable sets the NPC's loot table
func (n *NPC) SetLootTable(lootTable []LootEntry) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.LootTable = lootTable
}

// GetLootTable returns a copy of the NPC's loot table
func (n *NPC) GetLootTable() []LootEntry {
	n.mu.RLock()
	defer n.mu.RUnlock()
	lootTable := make([]LootEntry, len(n.LootTable))
	copy(lootTable, n.LootTable)
	return lootTable
}

// GetExperience returns the XP awarded for defeating this NPC
func (n *NPC) GetExperience() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Experience
}

// SetGoldDrop sets the gold drop range for this NPC
func (n *NPC) SetGoldDrop(min, max int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.GoldMin = min
	n.GoldMax = max
}

// RollGold returns a random gold amount between GoldMin and GoldMax
// Returns 0 if no gold range is set
func (n *NPC) RollGold() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.GoldMax <= 0 {
		return 0
	}
	if n.GoldMin >= n.GoldMax {
		return n.GoldMin
	}
	return n.GoldMin + rand.Intn(n.GoldMax-n.GoldMin+1)
}

// IsAggressive returns true if this NPC auto-attacks players
func (n *NPC) IsAggressive() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Aggressive
}

// IsAttackable returns true if players can attack this NPC
func (n *NPC) IsAttackable() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Attackable
}

// String returns a formatted string representation of the NPC
func (n *NPC) String() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return fmt.Sprintf("%s (Level %d, %d/%d HP)", n.Name, n.Level, n.Health, n.MaxHealth)
}

// GetRespawnMedian returns the median respawn time in seconds
func (n *NPC) GetRespawnMedian() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.RespawnMedian
}

// GetRespawnVariation returns the respawn time variation in seconds
func (n *NPC) GetRespawnVariation() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.RespawnVariation
}

// GetOriginalRoomID returns the room where this NPC originally spawned
func (n *NPC) GetOriginalRoomID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.OriginalRoomID
}

// SetOriginalRoomID sets the NPC's original spawn room ID (thread-safe)
func (n *NPC) SetOriginalRoomID(roomID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.OriginalRoomID = roomID
}

// GetRoomID returns the NPC's current room ID
func (n *NPC) GetRoomID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.RoomID
}

// SetRoomID sets the NPC's current room ID (thread-safe)
func (n *NPC) SetRoomID(roomID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RoomID = roomID
}

// GetRespawnTime returns when this NPC should respawn
func (n *NPC) GetRespawnTime() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.RespawnTime
}

// CalculateRespawnTime sets death time to now and calculates respawn time
// Returns the calculated respawn time
func (n *NPC) CalculateRespawnTime() time.Time {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.DeathTime = time.Now()

	// If respawn is disabled (median = 0), return zero time
	if n.RespawnMedian == 0 {
		n.RespawnTime = time.Time{}
		return n.RespawnTime
	}

	// Calculate random variation: median +/- variation
	variation := 0
	if n.RespawnVariation > 0 {
		// Random value between -variation and +variation
		variation = rand.Intn(2*n.RespawnVariation+1) - n.RespawnVariation
	}

	respawnSeconds := n.RespawnMedian + variation
	if respawnSeconds < 1 {
		respawnSeconds = 1 // Minimum 1 second
	}

	n.RespawnTime = n.DeathTime.Add(time.Duration(respawnSeconds) * time.Second)
	return n.RespawnTime
}

// Reset resets the NPC to full health and clears combat state
func (n *NPC) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Health = n.MaxHealth
	n.InCombat = false
	n.Targets = make(map[string]bool)
	n.ThreatTable = make(map[string]int)
	n.DeathTime = time.Time{}
	n.RespawnTime = time.Time{}
	n.StunEndTime = time.Time{}
	n.RootEndTime = time.Time{}
}

// Stun applies a stun effect to the NPC for the given duration in seconds
func (n *NPC) Stun(durationSeconds int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.StunEndTime = time.Now().Add(time.Duration(durationSeconds) * time.Second)
}

// IsStunned returns true if the NPC is currently stunned
func (n *NPC) IsStunned() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return time.Now().Before(n.StunEndTime)
}

// GetStunRemaining returns the seconds remaining on the stun, or 0 if not stunned
func (n *NPC) GetStunRemaining() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	remaining := time.Until(n.StunEndTime)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// Root applies a root effect to the NPC for the given duration in seconds
func (n *NPC) Root(durationSeconds int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RootEndTime = time.Now().Add(time.Duration(durationSeconds) * time.Second)
}

// IsRooted returns true if the NPC is currently rooted (cannot flee)
func (n *NPC) IsRooted() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return time.Now().Before(n.RootEndTime)
}

// GetRootRemaining returns the seconds remaining on the root, or 0 if not rooted
func (n *NPC) GetRootRemaining() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	remaining := time.Until(n.RootEndTime)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// GetFleeThreshold returns the HP percentage at which this mob will flee
func (n *NPC) GetFleeThreshold() float64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.FleeThreshold
}

// SetFleeThreshold sets the HP percentage at which this mob will flee
func (n *NPC) SetFleeThreshold(threshold float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.FleeThreshold = threshold
}

// FleeChance is the probability (0.0-1.0) that a mob will flee each round when below threshold
const FleeChance = 0.35

// ShouldFlee returns true if the mob should attempt to flee (below threshold, not rooted, and passes chance roll)
func (n *NPC) ShouldFlee() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Bosses never flee
	if n.IsBoss {
		return false
	}

	// No flee threshold set
	if n.FleeThreshold <= 0 {
		return false
	}

	// Check if rooted
	if time.Now().Before(n.RootEndTime) {
		return false
	}

	// Check HP percentage
	if n.MaxHealth <= 0 {
		return false
	}
	hpPercent := float64(n.Health) / float64(n.MaxHealth)
	if hpPercent > n.FleeThreshold {
		return false
	}

	// Roll flee chance - mobs don't always flee when they could
	return rand.Float64() < FleeChance
}

// GetDefaultFleeThreshold returns the default flee threshold for a mob type
func GetDefaultFleeThreshold(mobType MobType) float64 {
	switch mobType {
	case MobTypeUndead:
		return 0 // Undead never flee
	case MobTypeConstruct:
		return 0 // Constructs never flee
	case MobTypeDemon:
		return 0.05 // Demons are brave, flee at 5%
	case MobTypeGiant:
		return 0.10 // Giants are stubborn, flee at 10%
	case MobTypeBeast:
		return 0.15 // Beasts have survival instincts, flee at 15%
	case MobTypeHumanoid:
		return 0.12 // Humanoids flee at 12%
	default:
		return 0.12 // Default to 12%
	}
}

// SetBoss marks this NPC as a boss for the given floor
func (n *NPC) SetBoss(floor int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.IsBoss = true
	n.Floor = floor
}

// GetIsBoss returns whether this NPC is a boss
func (n *NPC) GetIsBoss() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.IsBoss
}

// GetFloor returns the floor this NPC is on
func (n *NPC) GetFloor() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Floor
}

// GetDialogue returns a random dialogue line, or empty string if no dialogue
func (n *NPC) GetDialogue() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.Dialogue) == 0 {
		return ""
	}
	return n.Dialogue[rand.Intn(len(n.Dialogue))]
}

// SetDialogue sets the NPC's dialogue lines
func (n *NPC) SetDialogue(dialogue []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Dialogue = dialogue
}

// SetShopInventory sets the NPC's shop inventory
func (n *NPC) SetShopInventory(inventory []ShopItem) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ShopInventory = inventory
}

// GetShopInventory returns a copy of the NPC's shop inventory
func (n *NPC) GetShopInventory() []ShopItem {
	n.mu.RLock()
	defer n.mu.RUnlock()
	inventory := make([]ShopItem, len(n.ShopInventory))
	copy(inventory, n.ShopInventory)
	return inventory
}

// HasShopInventory returns true if this NPC has items for sale
func (n *NPC) HasShopInventory() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.ShopInventory) > 0
}

// GetMobType returns the NPC's mob type (beast, undead, humanoid, etc.)
func (n *NPC) GetMobType() MobType {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.MobType
}

// SetMobType sets the NPC's mob type
func (n *NPC) SetMobType(mobType MobType) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.MobType = mobType
}

// IsBeast returns true if this NPC is of type beast (for ranger's favored enemy)
func (n *NPC) IsBeast() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.MobType == MobTypeBeast
}

// IsUndead returns true if this NPC is of type undead (for paladin's bonus)
func (n *NPC) IsUndead() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.MobType == MobTypeUndead
}

// IsDemon returns true if this NPC is of type demon (for paladin's bonus)
func (n *NPC) IsDemon() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.MobType == MobTypeDemon
}

// AddThreat adds threat from a player (damage dealt = threat)
func (n *NPC) AddThreat(playerName string, amount int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.ThreatTable == nil {
		n.ThreatTable = make(map[string]int)
	}
	n.ThreatTable[playerName] += amount
}

// GetThreat returns the current threat value for a player
func (n *NPC) GetThreat(playerName string) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.ThreatTable[playerName]
}

// GetHighestThreatTarget returns the player with the highest threat
// Falls back to random target if threat table is empty
func (n *NPC) GetHighestThreatTarget() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.Targets) == 0 {
		return ""
	}

	// Find the target with highest threat
	var highestTarget string
	highestThreat := -1

	for name := range n.Targets {
		threat := n.ThreatTable[name]
		if threat > highestThreat {
			highestThreat = threat
			highestTarget = name
		}
	}

	// If we found a target with threat, return it
	if highestTarget != "" {
		return highestTarget
	}

	// Fall back to random target (for cases where threat table is empty)
	targets := make([]string, 0, len(n.Targets))
	for name := range n.Targets {
		targets = append(targets, name)
	}
	return targets[rand.Intn(len(targets))]
}

// ClearThreat removes all threat for a specific player
func (n *NPC) ClearThreat(playerName string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.ThreatTable, playerName)
}

// ModifyThreat multiplies a player's threat by a factor (for threat reduction abilities)
func (n *NPC) ModifyThreat(playerName string, factor float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if threat, exists := n.ThreatTable[playerName]; exists {
		n.ThreatTable[playerName] = int(float64(threat) * factor)
	}
}

// GetTrainerClass returns the class this NPC trains (empty string if not a trainer)
func (n *NPC) GetTrainerClass() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.TrainerClass
}

// SetTrainerClass sets the class this NPC can train
func (n *NPC) SetTrainerClass(className string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.TrainerClass = className
}

// IsTrainer returns true if this NPC can train players in a class
func (n *NPC) IsTrainer() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.TrainerClass != ""
}

// GetCraftingTrainer returns the crafting skill this NPC teaches (empty string if not a crafting trainer)
func (n *NPC) GetCraftingTrainer() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.CraftingTrainer
}

// SetCraftingTrainer sets the crafting skill this NPC teaches
func (n *NPC) SetCraftingTrainer(skill string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.CraftingTrainer = skill
}

// GetTeachesRecipes returns the recipe IDs this NPC can teach
func (n *NPC) GetTeachesRecipes() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	recipes := make([]string, len(n.TeachesRecipes))
	copy(recipes, n.TeachesRecipes)
	return recipes
}

// SetTeachesRecipes sets the recipes this NPC can teach
func (n *NPC) SetTeachesRecipes(recipes []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.TeachesRecipes = recipes
}

// IsCraftingTrainer returns true if this NPC teaches crafting recipes
func (n *NPC) IsCraftingTrainer() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.CraftingTrainer != "" && len(n.TeachesRecipes) > 0
}

// IsQuestGiver returns true if this NPC can give quests
func (n *NPC) IsQuestGiver() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.QuestGiver && len(n.GivesQuests) > 0
}

// SetQuestGiver sets whether this NPC can give quests
func (n *NPC) SetQuestGiver(isQuestGiver bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.QuestGiver = isQuestGiver
}

// GetGivesQuests returns the list of quest IDs this NPC can give
func (n *NPC) GetGivesQuests() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	quests := make([]string, len(n.GivesQuests))
	copy(quests, n.GivesQuests)
	return quests
}

// SetGivesQuests sets the quest IDs this NPC can give
func (n *NPC) SetGivesQuests(questIDs []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.GivesQuests = questIDs
	if len(questIDs) > 0 {
		n.QuestGiver = true
	}
}

// CanGiveQuest returns true if this NPC can give a specific quest
func (n *NPC) CanGiveQuest(questID string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, id := range n.GivesQuests {
		if id == questID {
			return true
		}
	}
	return false
}

// GetTurnInQuests returns the list of quest IDs that can be turned in to this NPC
func (n *NPC) GetTurnInQuests() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	quests := make([]string, len(n.TurnInQuests))
	copy(quests, n.TurnInQuests)
	return quests
}

// SetTurnInQuests sets the quest IDs that can be turned in to this NPC
func (n *NPC) SetTurnInQuests(questIDs []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.TurnInQuests = questIDs
}

// CanTurnInQuest returns true if a specific quest can be turned in to this NPC
func (n *NPC) CanTurnInQuest(questID string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, id := range n.TurnInQuests {
		if id == questID {
			return true
		}
	}
	return false
}

// HasQuestInteraction returns true if this NPC has any quest-related interactions
func (n *NPC) HasQuestInteraction() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.GivesQuests) > 0 || len(n.TurnInQuests) > 0
}
