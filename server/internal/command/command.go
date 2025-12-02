package command

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
)

// ServerInterface defines the methods we need from the server
// To avoid circular dependencies, this is defined with interface{} parameters
type ServerInterface interface {
	BroadcastToRoom(roomID string, message string, exclude interface{})
	BroadcastToRoomFromPlayer(roomID string, message string, exclude interface{}, senderName string)
	BroadcastToFloor(floor int, message string, exclude interface{})
	BroadcastToFloorFromPlayer(floor int, message string, exclude interface{}, senderName string)
	BroadcastToAll(message string)
	BroadcastToAdmins(message string)
	FindPlayer(name string) interface{} // Returns a PlayerInterface
	GetOnlinePlayers() []string
	GetOnlinePlayersDetailed() []PlayerInfo // For admin players command
	GetUptime() time.Duration
	// Game time methods
	GetCurrentHour() int
	GetTimeOfDay() string
	IsDay() bool
	IsNight() bool
	GetGameClock() interface{} // Returns *gametime.GameClock
	// Server mode methods
	IsPilgrimMode() bool
	// Chat filter methods
	GetChatFilter() *chatfilter.ChatFilter
	// Persistence methods
	GetDatabase() interface{} // Returns *database.Database
	SavePlayer(player interface{}) error
	// World methods
	GetWorld() interface{} // Returns *world.World
	GetWorldRoomCount() int
	// Admin methods
	KickPlayer(playerName string, reason string) bool
	// Spell methods
	GetSpellRegistry() *spells.SpellRegistry
	// Tower methods (for floor generation when climbing stairs)
	GenerateNextFloor(currentFloor int) (nextFloorStairsRoom interface{}, err error)
	// Item methods
	GetItemByID(id string) *items.Item
}

// PlayerInterface defines the methods we need from a player object
// These are satisfied by *player.Player
type PlayerInterface interface {
	GetCurrentRoom() interface{} // Returns a RoomInterface
	GetInventory() []*items.Item // Returns concrete item slice
	GetName() string
	MoveTo(room interface{}) // Accepts a RoomInterface
	AddItem(item *items.Item)
	RemoveItem(itemName string) (*items.Item, bool)
	HasItem(itemName string) bool
	FindItem(partial string) (*items.Item, bool)
	GetCurrentWeight() float64
	CanCarry(item *items.Item) bool
	Disconnect()
	GetServer() interface{} // Returns a ServerInterface (avoid circular dependency)
	SendMessage(message string)
	GetState() string
	SetState(state string) error
	GetHealth() int
	GetMaxHealth() int
	GetMana() int
	GetMaxMana() int
	GetLevel() int
	GetExperience() int
	// Equipment methods
	EquipItem(item *items.Item) error
	UnequipItem(slot items.EquipmentSlot) (*items.Item, error)
	FindEquippedItem(partial string) (*items.Item, items.EquipmentSlot, bool)
	GetEquipment() map[items.EquipmentSlot]*items.Item
	// Consumable methods
	ConsumeItem(item *items.Item) string
	// Combat methods
	IsInCombat() bool
	GetCombatTarget() string
	StartCombat(npcName string)
	EndCombat()
	TakeDamage(damage int) int
	Heal(amount int) int
	HealToFull() int
	RestoreManaToFull() int
	GetAttackDamage() int
	IsAlive() bool
	GainExperience(xp int) []leveling.LevelUpInfo
	// Persistence methods
	GetAccountID() int64
	GetCharacterID() int64
	// Admin methods
	IsAdmin() bool
	GetRoomID() string
	// Spell methods
	HasSpell(spellID string) bool
	LearnSpell(spellID string)
	IsSpellOnCooldown(spellID string) (bool, int)
	StartSpellCooldown(spellID string, seconds int)
	GetLearnedSpells() []string
	UseMana(amount int) bool
	// Ability score methods
	GetStrength() int
	GetDexterity() int
	GetConstitution() int
	GetIntelligence() int
	GetWisdom() int
	GetCharisma() int
	// Ability modifier methods
	GetIntelligenceMod() int
	GetWisdomMod() int
	// Portal discovery methods (tower system)
	DiscoverPortal(floorNum int)
	HasDiscoveredPortal(floorNum int) bool
	GetDiscoveredPortals() []int
	// Key ring methods
	AddKey(key *items.Item)
	RemoveKey(keyName string) (*items.Item, bool)
	RemoveKeyByID(keyID string) (*items.Item, bool)
	HasKey(keyID string) bool
	FindKey(partial string) (*items.Item, bool)
	GetKeyRing() []*items.Item
	// Gold/currency methods
	GetGold() int
	AddGold(amount int)
	SpendGold(amount int) bool
	// Anti-spam methods
	CheckChatSpam(message string) (allowed bool, reason string)
	// Ignore list methods
	IsIgnoring(playerName string) bool
	AddIgnore(playerName string)
	RemoveIgnore(playerName string)
	GetIgnoreList() []string
	// Class/proficiency methods
	CanEquipItem(item *items.Item) (bool, string) // Returns (canEquip, reason)
	GetPrimaryClassName() string                  // Returns the primary class display name
	// Class-based spell access
	CanCastSpellForClass(allowedClasses []string, requiredLevel int) bool
	GetAllClassLevelsMap() map[string]int
	// Multiclass methods
	GetActiveClassName() string     // Returns the active class display name
	GetClassLevelsSummary() string  // Returns a formatted string of all class levels
	CanMulticlass() bool            // Returns true if player can multiclass (primary >= 10)
	CanMulticlassInto(className string) (bool, string) // Returns (canMulticlass, reason)
	AddNewClass(className string) error               // Add a new class at level 1
	SwitchActiveClass(className string) error         // Switch which class gains XP
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

// RoomInterface defines the methods we need from a room object
// These are satisfied by *world.Room
type RoomInterface interface {
	GetDescription() string
	GetBaseDescription() string
	GetDescriptionForPlayer(playerName string) string
	GetDescriptionForPlayerWithCustomDesc(playerName string, baseDesc string) string
	GetDescriptionDay() string
	GetDescriptionNight() string
	GetExit(direction string) interface{} // Returns a RoomInterface or nil
	GetExits() map[string]string          // Returns map of direction -> room name
	GetID() string
	GetFloor() int // Returns the tower floor number (0 = city)
	HasItem(itemName string) bool
	RemoveItem(itemName string) (*items.Item, bool)
	AddItem(item *items.Item)
	FindItem(partial string) (*items.Item, bool)
	HasFeature(feature string) bool
	RemoveFeature(feature string)
	IsExitLocked(direction string) bool
	GetExitKeyRequired(direction string) string
	UnlockExit(direction string)
	GetNPCs() []*npc.NPC
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
	"purchase": executeBuy,
	"sell":     executeSell,
	"gold":     executeGold,
	"money":    executeGold,
	"wallet":   executeGold,
	"give":     executeGive,

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

	// Admin commands
	"admin": executeAdmin,

	// Special static responses
	"save": func(c *Command, p PlayerInterface) string {
		return "To save your progress, visit the bard in the tavern and ask him to write a song about your adventures."
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
