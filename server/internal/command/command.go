package command

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/crafting"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
)

// StallItem represents an item for sale in a player's stall.
type StallItem struct {
	Item  *items.Item // The item being sold
	Price int         // The asking price in gold
}

// ServerInterface defines the contract for server operations needed by command handlers.
//
// This interface exists to avoid circular dependencies between the command and server
// packages. The concrete implementation is *server.Server.
//
// # Thread Safety
//
// All methods on ServerInterface are thread-safe. The server implementation uses
// appropriate locking (sync.RWMutex) to protect shared state. Command handlers
// can call these methods without additional synchronization.
//
// # Interface{} Parameters
//
// Several methods use interface{} for parameters or return values to avoid import
// cycles. These should be type-asserted to the appropriate types:
//   - FindPlayer() returns a PlayerInterface (or nil if not found)
//   - GetDatabase() returns *database.Database
//   - GetWorld() returns *world.World
//   - GetGameClock() returns *gametime.GameClock
//   - GenerateNextFloor() returns a RoomInterface for the stairs room
//
// # Required vs Optional
//
// All methods are required for full functionality. However, some methods may return
// nil or zero values in test scenarios:
//   - GetChatFilter() may return nil if chat filtering is disabled
//   - GetSpellRegistry() may return nil in tests without spell data
//   - GetRecipeRegistry() may return nil in tests without crafting data
//   - GetQuestRegistry() may return nil in tests without quest data
type ServerInterface interface {
	// === Broadcasting Methods ===
	// These methods send messages to players. The exclude parameter (PlayerInterface)
	// specifies a player who should NOT receive the message (typically the sender).

	// BroadcastToRoom sends a message to all players in the specified room.
	// exclude should be a PlayerInterface or nil.
	BroadcastToRoom(roomID string, message string, exclude interface{})

	// BroadcastToRoomFromPlayer sends a message to all players in a room,
	// respecting ignore lists. Players ignoring senderName won't receive the message.
	BroadcastToRoomFromPlayer(roomID string, message string, exclude interface{}, senderName string)

	// BroadcastToFloor sends a message to all players on the specified tower floor.
	BroadcastToFloor(floor int, message string, exclude interface{})

	// BroadcastToFloorFromPlayer sends a floor-wide message respecting ignore lists.
	BroadcastToFloorFromPlayer(floor int, message string, exclude interface{}, senderName string)

	// BroadcastToAll sends a message to all connected players (server announcements).
	BroadcastToAll(message string)

	// BroadcastToAdmins sends a message only to players with admin privileges.
	BroadcastToAdmins(message string)

	// === Player Lookup Methods ===

	// FindPlayer finds an online player by name (case-insensitive).
	// Returns a PlayerInterface if found, nil otherwise.
	// Supports partial name matching (prefix match).
	FindPlayer(name string) interface{}

	// GetOnlinePlayers returns a list of all online player names.
	GetOnlinePlayers() []string

	// GetOnlinePlayersDetailed returns detailed info about online players (admin only).
	GetOnlinePlayersDetailed() []PlayerInfo

	// === Server State Methods ===

	// GetUptime returns how long the server has been running.
	GetUptime() time.Duration

	// === Game Time Methods ===
	// The game has an accelerated day/night cycle (default: 1 real hour = 1 game day).

	// GetCurrentHour returns the current in-game hour (0-23).
	GetCurrentHour() int

	// GetTimeOfDay returns a human-readable time description (e.g., "morning", "night").
	GetTimeOfDay() string

	// IsDay returns true if it's currently daytime in-game (6:00-17:59).
	IsDay() bool

	// IsNight returns true if it's currently nighttime in-game (18:00-5:59).
	IsNight() bool

	// GetGameClock returns the game clock for advanced time queries.
	// Returns *gametime.GameClock.
	GetGameClock() interface{}

	// === Server Mode Methods ===

	// IsPilgrimMode returns true if the server is in pilgrim (permadeath) mode.
	IsPilgrimMode() bool

	// === Filter Methods ===

	// GetChatFilter returns the chat filter for profanity/spam filtering.
	// May return nil if filtering is disabled.
	GetChatFilter() *chatfilter.ChatFilter

	// === Persistence Methods ===

	// GetDatabase returns the database connection for direct queries.
	// Returns *database.Database.
	GetDatabase() interface{}

	// SavePlayer persists the player's current state to the database.
	// The player parameter should be a PlayerInterface.
	SavePlayer(player interface{}) error

	// === World Methods ===

	// GetWorld returns the game world containing all rooms.
	// Returns *world.World.
	GetWorld() interface{}

	// GetWorldRoomCount returns the total number of rooms across all floors.
	GetWorldRoomCount() int

	// === Admin Methods ===

	// KickPlayer disconnects a player by name with an optional reason.
	// Returns true if the player was found and kicked, false otherwise.
	KickPlayer(playerName string, reason string) bool

	// === Registry Methods ===
	// These return nil if the corresponding system is not initialized.

	// GetSpellRegistry returns the registry of available spells.
	GetSpellRegistry() *spells.SpellRegistry

	// GetRecipeRegistry returns the registry of crafting recipes.
	GetRecipeRegistry() *crafting.RecipeRegistry

	// GetQuestRegistry returns the registry of available quests.
	GetQuestRegistry() *quest.QuestRegistry

	// === Tower Methods ===

	// GenerateNextFloor generates the next tower floor and returns the stairs room.
	// Called when a player climbs stairs to an ungenerated floor.
	// Returns (RoomInterface, nil) on success, (nil, error) on failure.
	GenerateNextFloor(currentFloor int) (nextFloorStairsRoom interface{}, err error)

	// === Item Methods ===

	// GetItemByID returns an item template by its ID, or nil if not found.
	// This returns the template, not a copy - do not modify.
	GetItemByID(id string) *items.Item

	// CreateItem creates a new item instance from a template ID.
	// Returns nil if the template doesn't exist.
	CreateItem(id string) *items.Item
}

// PlayerInterface defines the contract for player operations needed by command handlers.
//
// The concrete implementation is *player.Player. This interface exists to avoid
// circular dependencies between the command and player packages.
//
// # Thread Safety
//
// PlayerInterface methods are NOT inherently thread-safe. Each player has their own
// goroutine handling input, and player state should only be modified from that
// goroutine. Cross-player interactions (like trading) must go through the server.
//
// The following methods are safe to call from any goroutine:
//   - GetName() - immutable after creation
//   - GetAccountID(), GetCharacterID() - immutable after creation
//   - SendMessage() - internally synchronized
//
// # Player States
//
// Players can be in different states that affect which commands are available:
//   - "normal" - default state, all commands available
//   - "sleeping" - limited commands (wake, look, quit)
//   - "combat" - combat commands only, no movement
//   - "dead" - very limited commands until respawn
//
// Use GetState() to check and SetState() to change. SetState returns an error
// if the transition is invalid.
//
// # Interface{} Parameters
//
// Some methods use interface{} to avoid import cycles:
//   - GetCurrentRoom() returns a RoomInterface
//   - GetServer() returns a ServerInterface
//   - MoveTo() accepts a RoomInterface
type PlayerInterface interface {
	// === Core Identity ===

	// GetName returns the player's character name (immutable, thread-safe).
	GetName() string

	// GetAccountID returns the database account ID (immutable, thread-safe).
	GetAccountID() int64

	// GetCharacterID returns the database character ID (immutable, thread-safe).
	GetCharacterID() int64

	// IsAdmin returns true if the player has admin privileges.
	IsAdmin() bool

	// === Location & Movement ===

	// GetCurrentRoom returns the player's current room.
	// Returns a RoomInterface. Never returns nil for a valid player.
	GetCurrentRoom() interface{}

	// GetRoomID returns the ID of the player's current room.
	GetRoomID() string

	// MoveTo moves the player to a new room.
	// The room parameter must be a RoomInterface.
	// This updates the room's player list and triggers any room enter effects.
	MoveTo(room interface{})

	// === Communication ===

	// SendMessage sends a message to this player's client (thread-safe).
	// Messages are queued and sent asynchronously.
	SendMessage(message string)

	// Disconnect closes the player's connection gracefully.
	// Triggers save and cleanup.
	Disconnect()

	// GetServer returns the server instance for broadcasting.
	// Returns a ServerInterface.
	GetServer() interface{}

	// === Player State ===

	// GetState returns the current player state ("normal", "sleeping", "combat", "dead").
	GetState() string

	// SetState changes the player state.
	// Returns an error if the transition is invalid (e.g., can't sleep while in combat).
	SetState(state string) error

	// === Vital Statistics ===

	// GetHealth returns current health points.
	GetHealth() int

	// GetMaxHealth returns maximum health points (affected by level and CON).
	GetMaxHealth() int

	// GetMana returns current mana points.
	GetMana() int

	// GetMaxMana returns maximum mana points (affected by level and INT/WIS).
	GetMaxMana() int

	// GetLevel returns the player's current level (1-20).
	GetLevel() int

	// GetExperience returns current experience points toward next level.
	GetExperience() int

	// IsAlive returns true if health > 0.
	IsAlive() bool

	// === Inventory Management ===

	// GetInventory returns a copy of the player's inventory.
	// Modifying the returned slice does not affect the player's inventory.
	GetInventory() []*items.Item

	// AddItem adds an item to the player's inventory.
	// Does not check weight limits - use CanCarry() first.
	AddItem(item *items.Item)

	// RemoveItem removes an item by exact name (case-insensitive).
	// Returns the removed item and true, or nil and false if not found.
	RemoveItem(itemName string) (*items.Item, bool)

	// RemoveItemByID removes the first item matching the given template ID.
	// Returns true if an item was removed.
	RemoveItemByID(itemID string) bool

	// HasItem returns true if the player has an item with the given name.
	HasItem(itemName string) bool

	// FindItem finds an item by partial name match (prefix).
	// Returns the item and true if found, or nil and false.
	FindItem(partial string) (*items.Item, bool)

	// CountItemsByID counts items matching the given template ID.
	CountItemsByID(itemID string) int

	// GetCurrentWeight returns the total weight of carried items.
	GetCurrentWeight() float64

	// CanCarry returns true if the player can carry the additional item.
	// Based on strength and current weight.
	CanCarry(item *items.Item) bool

	// === Equipment ===

	// EquipItem equips an item to its appropriate slot.
	// Returns an error if the item can't be equipped (wrong class, slot occupied, etc.).
	EquipItem(item *items.Item) error

	// UnequipItem removes an item from an equipment slot.
	// Returns the unequipped item and nil, or nil and an error if slot is empty.
	UnequipItem(slot items.EquipmentSlot) (*items.Item, error)

	// FindEquippedItem finds an equipped item by partial name.
	// Returns the item, its slot, and true if found.
	FindEquippedItem(partial string) (*items.Item, items.EquipmentSlot, bool)

	// GetEquipment returns a copy of the equipment map.
	GetEquipment() map[items.EquipmentSlot]*items.Item

	// CanEquipItem checks if the player can equip an item based on class/level.
	// Returns (true, "") if allowed, or (false, reason) if not.
	CanEquipItem(item *items.Item) (bool, string)

	// === Consumables ===

	// ConsumeItem uses a consumable item and returns a result message.
	// Applies the item's effects (healing, mana restore, buffs).
	ConsumeItem(item *items.Item) string

	// === Combat ===

	// IsInCombat returns true if the player is currently in combat.
	IsInCombat() bool

	// GetCombatTarget returns the name of the NPC the player is fighting.
	// Returns empty string if not in combat.
	GetCombatTarget() string

	// StartCombat initiates combat with the named NPC.
	StartCombat(npcName string)

	// EndCombat ends the current combat (victory, flee, or death).
	EndCombat()

	// TakeDamage applies damage to the player.
	// Returns the actual damage taken (after any reductions).
	TakeDamage(damage int) int

	// Heal restores health up to max.
	// Returns the actual amount healed.
	Heal(amount int) int

	// HealToFull restores health to maximum.
	// Returns the amount healed.
	HealToFull() int

	// RestoreManaToFull restores mana to maximum.
	// Returns the amount restored.
	RestoreManaToFull() int

	// GetAttackDamage returns the player's attack damage (weapon + STR mod).
	GetAttackDamage() int

	// GainExperience adds XP and handles level ups.
	// Returns a slice of LevelUpInfo for any levels gained.
	GainExperience(xp int) []leveling.LevelUpInfo

	// === Spells & Magic ===

	// HasSpell returns true if the player knows the spell.
	HasSpell(spellID string) bool

	// LearnSpell teaches the player a new spell.
	LearnSpell(spellID string)

	// GetLearnedSpells returns a list of known spell IDs.
	GetLearnedSpells() []string

	// IsSpellOnCooldown returns true and remaining seconds if on cooldown.
	IsSpellOnCooldown(spellID string) (bool, int)

	// StartSpellCooldown puts a spell on cooldown for the given duration.
	StartSpellCooldown(spellID string, seconds int)

	// UseMana consumes mana if available.
	// Returns true if mana was consumed, false if insufficient.
	UseMana(amount int) bool

	// CanCastSpellForClass checks if the player's class/level allows casting.
	CanCastSpellForClass(allowedClasses []string, requiredLevel int) bool

	// === Ability Scores (D&D-style, 3-18 range) ===

	GetStrength() int
	GetDexterity() int
	GetConstitution() int
	GetIntelligence() int
	GetWisdom() int
	GetCharisma() int

	// Ability modifiers: (score - 10) / 2
	GetIntelligenceMod() int
	GetWisdomMod() int

	// === Class & Race ===

	// GetPrimaryClassName returns the display name of the primary class.
	GetPrimaryClassName() string

	// GetActiveClassName returns the class currently gaining XP.
	GetActiveClassName() string

	// GetClassLevelsSummary returns a formatted string like "Warrior 10 / Mage 5".
	GetClassLevelsSummary() string

	// GetAllClassLevelsMap returns class name -> level for all classes.
	GetAllClassLevelsMap() map[string]int

	// CanMulticlass returns true if player can add a new class (primary >= 10).
	CanMulticlass() bool

	// CanMulticlassInto checks if the player can multiclass into a specific class.
	// Returns (true, "") or (false, reason).
	CanMulticlassInto(className string) (bool, string)

	// AddNewClass adds a new class at level 1.
	AddNewClass(className string) error

	// SwitchActiveClass changes which class gains XP.
	SwitchActiveClass(className string) error

	// GetRaceName returns the display name of the player's race.
	GetRaceName() string

	// === Portal System (Tower Travel) ===

	// DiscoverPortal marks a floor's portal as discovered.
	DiscoverPortal(floorNum int)

	// HasDiscoveredPortal returns true if the player has discovered the floor's portal.
	HasDiscoveredPortal(floorNum int) bool

	// GetDiscoveredPortals returns all discovered floor numbers.
	GetDiscoveredPortals() []int

	// === Key Ring (Separate from Inventory) ===

	// AddKey adds a key to the key ring.
	AddKey(key *items.Item)

	// RemoveKey removes a key by name.
	RemoveKey(keyName string) (*items.Item, bool)

	// RemoveKeyByID removes a key by its template ID.
	RemoveKeyByID(keyID string) (*items.Item, bool)

	// HasKey returns true if the player has a key with the given ID.
	HasKey(keyID string) bool

	// FindKey finds a key by partial name match.
	FindKey(partial string) (*items.Item, bool)

	// GetKeyRing returns all keys the player has.
	GetKeyRing() []*items.Item

	// === Currency ===

	// GetGold returns current gold amount.
	GetGold() int

	// AddGold adds gold (no upper limit).
	AddGold(amount int)

	// SpendGold deducts gold if available.
	// Returns true if successful, false if insufficient funds.
	SpendGold(amount int) bool

	// === Social Features ===

	// CheckChatSpam checks if a message should be blocked for spam.
	// Returns (true, "") if allowed, (false, reason) if blocked.
	CheckChatSpam(message string) (allowed bool, reason string)

	// IsIgnoring returns true if this player is ignoring the given player.
	IsIgnoring(playerName string) bool

	// AddIgnore adds a player to the ignore list.
	AddIgnore(playerName string)

	// RemoveIgnore removes a player from the ignore list.
	RemoveIgnore(playerName string)

	// GetIgnoreList returns all ignored player names.
	GetIgnoreList() []string

	// === Crafting System ===

	// GetCraftingSkill returns the skill level for a crafting skill.
	GetCraftingSkill(skill crafting.CraftingSkill) int

	// AddCraftingSkillPoints adds XP to a crafting skill.
	// Returns the new skill level.
	AddCraftingSkillPoints(skill crafting.CraftingSkill, points int) int

	// GetAllCraftingSkills returns all crafting skill levels.
	GetAllCraftingSkills() map[crafting.CraftingSkill]int

	// KnowsRecipe returns true if the player knows the recipe.
	KnowsRecipe(recipeID string) bool

	// LearnRecipe teaches the player a crafting recipe.
	LearnRecipe(recipeID string)

	// GetKnownRecipes returns all known recipe IDs.
	GetKnownRecipes() []string

	// === Quest System ===

	// GetQuestLog returns the player's quest log.
	GetQuestLog() *quest.PlayerQuestLog

	// GetQuestState returns the full quest state for saving.
	GetQuestState() *quest.PlayerQuestState

	// HasActiveQuest returns true if the quest is in progress.
	HasActiveQuest(questID string) bool

	// HasCompletedQuest returns true if the quest has been completed.
	HasCompletedQuest(questID string) bool

	// Quest inventory (separate from regular inventory)
	GetQuestInventory() []*items.Item
	AddQuestItem(item *items.Item)
	RemoveQuestItem(itemID string) (*items.Item, bool)
	HasQuestItem(itemID string) bool
	ClearQuestInventoryForQuest(questItemIDs []string)

	// === Titles (Achievement Display) ===

	// GetEarnedTitles returns all earned title IDs.
	GetEarnedTitles() []string

	// HasEarnedTitle returns true if the player has earned the title.
	HasEarnedTitle(titleID string) bool

	// EarnTitle grants a new title.
	EarnTitle(titleID string)

	// GetActiveTitle returns the currently displayed title ID.
	GetActiveTitle() string

	// SetActiveTitle changes the displayed title.
	// Returns an error if the title hasn't been earned.
	SetActiveTitle(titleID string) error

	// === Player Stalls (Player-to-Player Trading) ===

	// IsStallOpen returns true if the player's stall is open for business.
	IsStallOpen() bool

	// OpenStall opens the player's stall in the current room.
	OpenStall()

	// CloseStall closes the stall and returns items to inventory.
	CloseStall()

	// GetStallInventory returns items currently for sale.
	GetStallInventory() []*StallItem

	// AddToStall adds an item for sale at the given price.
	AddToStall(item *items.Item, price int)

	// RemoveFromStall removes an item from the stall by partial name.
	RemoveFromStall(partial string) (*StallItem, bool)

	// FindInStall finds an item in the stall by partial name.
	FindInStall(partial string) (*StallItem, bool)

	// ClearStall removes all items from the stall and returns them.
	ClearStall() []*items.Item
}

// PlayerInfo contains detailed information about an online player (for admin commands)
type PlayerInfo struct {
	Name      string
	Level     int
	RoomID    string
	IP        string
	LoginTime time.Time
	IsAdmin   bool
}

// RoomInterface defines the contract for room operations needed by command handlers.
//
// The concrete implementation is *world.Room. Rooms are the fundamental unit of
// the game world - they contain descriptions, exits to other rooms, items, NPCs,
// and special features.
//
// # Thread Safety
//
// Room methods are NOT thread-safe. Rooms should only be modified from the server's
// main goroutine or with appropriate synchronization. In practice, room modifications
// happen during:
//   - Player actions (picking up items, unlocking doors)
//   - NPC spawning/despawning
//   - Combat (NPC death removing them from the room)
//
// Reading room state (descriptions, checking exits) is safe from any goroutine.
//
// # Room Types
//
// Rooms have types that affect their behavior and appearance:
//   - RoomTypeCity - City/town areas (floor 0)
//   - RoomTypeRoom - General dungeon rooms
//   - RoomTypeCorridor - Narrow passages
//   - RoomTypeStairs - Connections between floors
//   - RoomTypeTreasure - Loot rooms (often locked)
//   - RoomTypeBoss - Boss encounter rooms
//
// # Features
//
// Rooms can have string features that indicate special properties:
//   - "portal" - Player can use the portal command here
//   - "stairs_up" / "stairs_down" - Floor transitions
//   - "treasure" - Contains treasure chest
//   - "boss" - Boss room
//   - "merchant" - NPC merchant is here
//   - "altar" - Healing altar (city)
//   - "shop" - NPC shop (city)
//   - "locked_door" - Has a locked exit
//
// # Exits
//
// Standard exits are: "north", "south", "east", "west", "up", "down".
// Exits can be locked, requiring a specific key item to unlock.
type RoomInterface interface {
	// === Identity ===

	// GetID returns the unique room identifier (e.g., "floor5_3_7", "city_square").
	GetID() string

	// GetFloor returns the tower floor number.
	// 0 = city (ground floor), 1+ = dungeon floors.
	GetFloor() int

	// === Descriptions ===
	// Rooms can have different descriptions for day/night.

	// GetDescription returns the current description (day or night based on time).
	GetDescription() string

	// GetBaseDescription returns the default description without time variation.
	GetBaseDescription() string

	// GetDescriptionForPlayer returns a full room description for display.
	// Includes the room name, description, exits, items, NPCs, and other players.
	GetDescriptionForPlayer(playerName string) string

	// GetDescriptionForPlayerWithCustomDesc returns a room description with a custom base.
	// Used for special cases like looking through a window.
	GetDescriptionForPlayerWithCustomDesc(playerName string, baseDesc string) string

	// GetDescriptionDay returns the daytime description.
	GetDescriptionDay() string

	// GetDescriptionNight returns the nighttime description.
	GetDescriptionNight() string

	// === Navigation ===

	// GetExit returns the room in the given direction, or nil if no exit.
	// direction should be: "north", "south", "east", "west", "up", "down".
	// Returns a RoomInterface.
	GetExit(direction string) interface{}

	// GetExits returns a map of direction -> room name for display.
	// Only includes accessible exits (not hidden ones).
	GetExits() map[string]string

	// IsExitLocked returns true if the exit in the given direction is locked.
	IsExitLocked(direction string) bool

	// GetExitKeyRequired returns the key item ID needed to unlock the exit.
	// Returns empty string if the exit isn't locked or doesn't require a key.
	GetExitKeyRequired(direction string) string

	// UnlockExit permanently unlocks an exit.
	// The unlock state persists until the floor is regenerated.
	UnlockExit(direction string)

	// === Items ===

	// HasItem returns true if the room contains an item with the given name.
	HasItem(itemName string) bool

	// FindItem finds an item by partial name match.
	// Returns the item and true if found, or nil and false.
	FindItem(partial string) (*items.Item, bool)

	// AddItem adds an item to the room's floor.
	AddItem(item *items.Item)

	// RemoveItem removes an item by exact name.
	// Returns the removed item and true, or nil and false if not found.
	RemoveItem(itemName string) (*items.Item, bool)

	// === NPCs ===

	// GetNPCs returns all NPCs currently in the room.
	// Includes both friendly NPCs and hostile mobs.
	GetNPCs() []*npc.NPC

	// FindNPC finds an NPC by partial name match.
	// Returns the NPC or nil if not found.
	FindNPC(partial string) *npc.NPC

	// AddNPC adds an NPC to the room.
	AddNPC(n *npc.NPC)

	// RemoveNPC removes an NPC from the room.
	RemoveNPC(n *npc.NPC)

	// === Features ===

	// HasFeature returns true if the room has the given feature.
	// Features are string tags indicating special room properties.
	HasFeature(feature string) bool

	// GetFeatures returns all features on this room.
	GetFeatures() []string

	// RemoveFeature removes a feature from the room.
	// Used when a one-time feature is consumed (e.g., treasure looted).
	RemoveFeature(feature string)
}

// GetRoom safely extracts the room from a player's GetCurrentRoom() result.
// Returns the RoomInterface and true if successful, or nil and false if the
// room is nil or not a valid room type.
func GetRoom(p PlayerInterface) (RoomInterface, bool) {
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return nil, false
	}
	room, ok := roomIface.(RoomInterface)
	return room, ok
}

// MustGetRoom returns the room or panics with a descriptive error.
// Use this only in contexts where a nil room indicates a programming error.
func MustGetRoom(p PlayerInterface) RoomInterface {
	room, ok := GetRoom(p)
	if !ok {
		logger.Error("Player has invalid room", "player", p.GetName())
		panic("player has nil or invalid room - this should never happen")
	}
	return room
}

// WorldInterface defines the methods we need from a world object
// Note: We don't actually use this interface, kept for future use
type WorldInterface interface {
	// GetRoom would be here if we needed it
}

// Command represents a parsed player command
type Command struct {
	Name string
	Args []string
}

// CommandHandler is the function signature for command handlers
type CommandHandler func(c *Command, p PlayerInterface) string

// commandRegistry maps command names (and aliases) to their handlers
var commandRegistry = map[string]CommandHandler{
	// Info commands
	"help": executeHelp,

	// Movement commands
	"look":    executeLook,
	"l":       executeLook,
	"examine": executeLook,
	"ex":      executeLook,
	"go":      executeMove,
	"move":    executeMove,
	"walk":    executeMove,
	"north":   func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "north") },
	"n":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "north") },
	"south":   func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "south") },
	"s":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "south") },
	"east":    func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "east") },
	"e":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "east") },
	"west":    func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "west") },
	"w":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "west") },
	"up":      func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "up") },
	"u":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "up") },
	"down":    func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "down") },
	"d":       func(c *Command, p PlayerInterface) string { return executeMoveDirection(c, p, "down") },
	"exits":   executeExits,
	"portal":  executePortal,

	// Item commands
	"take":      executeTake,
	"get":       executeTake,
	"pickup":    executeTake,
	"drop":      executeDrop,
	"inventory": executeInventory,
	"inv":       executeInventory,
	"i":         executeInventory,
	"equipment": executeEquipment,
	"eq":        executeEquipment,
	"wield":     executeWield,
	"wear":      executeWear,
	"remove":    executeRemove,
	"hold":      executeHold,
	"eat":       executeEat,
	"drink":     executeDrink,
	"use":       executeUse,

	// Social commands
	"say":   executeSay,
	"who":      executeWho,
	"tell":     executeTell,
	"shout":    executeShout,
	"yell":     executeShout,
	"emote":    executeEmote,
	"me":       executeEmote,
	"report":   executeReport,
	"ignore":   executeIgnore,
	"unignore": executeUnignore,
	"quit":  executeQuit,
	"exit":  executeQuit,

	// State commands
	"time":  executeTime,
	"sleep": executeSleep,
	"wake":  executeWake,
	"stand": executeStand,

	// Combat commands
	"attack":   executeAttack,
	"kill":     executeAttack,
	"hit":      executeAttack,
	"flee":     executeFlee,
	"consider": executeConsider,
	"con":      executeConsider,

	// Magic commands
	"cast":   executeCast,
	"spells": executeSpells,

	// Commerce commands
	"shop":     executeShop,
	"list":     executeShop,
	"buy":      executeBuy,
	"sell":     executeSell,
	"gold":     executeGold,
	"money":    executeGold,
	"wallet":   executeGold,
	"give":     executeGive,

	// Player stall commands
	"stall":    executeStall,
	"browse":   executeBrowse,
	"purchase": executePurchase,

	// Interaction commands
	"talk":   executeTalk,
	"speak":  executeTalk,
	"chat":   executeTalk,
	"unlock": executeUnlock,
	"pray":   executePray,
	"train":  executeTrain,

	// Character info commands
	"level":      executeLevel,
	"lvl":        executeLevel,
	"score":      executeScore,
	"sc":         executeScore,
	"stats":      executeScore, // Alias for score
	"abilities":  executeScore, // Alias for score
	"attributes": executeScore, // Alias for score
	"password":   executePassword,
	"class":      executeClass,
	"classes":    executeClass, // Alias for class
	"race":       executeRace,
	"races":      executeRaces,

	// Crafting commands
	"craft":  executeCraft,
	"make":   executeCraft, // Alias
	"learn":  executeLearn,
	"skills": executeSkills,

	// Quest commands
	"quest":    executeQuest,
	"quests":   executeQuest,
	"journal":  executeQuest,
	"accept":   executeAccept,
	"complete": executeComplete,
	"turnin":   executeComplete,
	"title":    executeTitle,

	// Mail commands
	"mail": executeMail,

	// Admin commands
	"admin": executeAdmin,

	// Special static responses
	"save": func(c *Command, p PlayerInterface) string {
		return "Your progress is saved automatically when you disconnect or quit. Just play and enjoy!"
	},
}

// RequireArgs checks if the command has at least the minimum number of arguments
// Returns an error with the usage message if not enough arguments are provided
func (c *Command) RequireArgs(min int, usage string) error {
	if len(c.Args) < min {
		return errors.New(usage)
	}
	return nil
}

// GetItemName joins all arguments into a single item name (for multi-word items)
func (c *Command) GetItemName() string {
	return strings.Join(c.Args, " ")
}

// GetTargetName is an alias for GetItemName for clarity when targeting players/NPCs
func (c *Command) GetTargetName() string {
	return c.GetItemName()
}

// ParseCommand parses a raw input string into a Command struct
func ParseCommand(input string) *Command {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return &Command{Name: "", Args: []string{}}
	}

	return &Command{
		Name: strings.ToLower(parts[0]),
		Args: parts[1:],
	}
}

// Execute runs the command for the given player
func (c *Command) Execute(playerIface interface{}, worldIface interface{}) string {
	// Type assertion to interface type
	p, ok := playerIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Log command execution
	logger.Debug("Command executed",
		"player", p.GetName(),
		"command", c.Name,
		"args", strings.Join(c.Args, " "))

	// Look up the handler in the registry
	handler, exists := commandRegistry[c.Name]
	if !exists {
		return fmt.Sprintf("Unknown command: %s. Type 'help' for available commands.", c.Name)
	}

	return handler(c, p)
}
