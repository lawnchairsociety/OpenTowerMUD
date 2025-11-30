package command

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/gametime"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// ServerInterface defines the methods we need from the server
// To avoid circular dependencies, this is defined with interface{} parameters
type ServerInterface interface {
	BroadcastToRoom(roomID string, message string, exclude interface{})
	BroadcastToAll(message string)
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
	GetInventory() []*items.Item  // Returns concrete item slice
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

type Command struct {
	Name string
	Args []string
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

func (c *Command) Execute(playerIface interface{}, worldIface interface{}) string {
	// Type assertion to interface type
	p, ok := playerIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Type assertion for world interface
	w, ok := worldIface.(WorldInterface)
	if !ok {
		return "Internal error: invalid world type"
	}

	switch c.Name {
	case "help":
		return c.executeHelpWithPlayer(p)
	case "look", "l", "examine", "ex":
		return c.executeLook(p)
	case "go", "move", "walk":
		return c.executeMove(p)
	case "north", "n":
		return c.executeMoveDirection(p, "north")
	case "south", "s":
		return c.executeMoveDirection(p, "south")
	case "east", "e":
		return c.executeMoveDirection(p, "east")
	case "west", "w":
		return c.executeMoveDirection(p, "west")
	case "up", "u":
		return c.executeMoveDirection(p, "up")
	case "down", "d":
		return c.executeMoveDirection(p, "down")
	case "take", "get", "pickup":
		return c.executeTake(p)
	case "drop":
		return c.executeDrop(p)
	case "inventory", "inv", "i":
		return c.executeInventory(p)
	case "quit", "exit":
		return c.executeQuit(p)
	case "say":
		return c.executeSay(p)
	case "who":
		return c.executeWho(p)
	case "tell":
		return c.executeTell(p)
	case "exits":
		return c.executeExits(p)
	case "equipment", "eq":
		return c.executeEquipment(p)
	case "wield":
		return c.executeWield(p)
	case "wear":
		return c.executeWear(p)
	case "remove":
		return c.executeRemove(p)
	case "hold":
		return c.executeHold(p)
	case "eat":
		return c.executeEat(p)
	case "drink":
		return c.executeDrink(p)
	case "use":
		return c.executeUse(p)
	case "time":
		return c.executeTime(p)
	case "sleep":
		return c.executeSleep(p)
	case "wake":
		return c.executeWake(p)
	case "stand":
		return c.executeStand(p)
	case "attack", "kill", "hit":
		return c.executeAttack(p, w)
	case "flee":
		return c.executeFlee(p, w)
	case "consider", "con":
		return c.executeConsider(p)
	case "save":
		return "To save your progress, visit the bard in the tavern and ask him to write a song about your adventures."
	case "password":
		return c.executePassword(p)
	case "admin":
		return c.executeAdmin(p)
	case "pray":
		return c.executePray(p)
	case "portal":
		return c.executePortal(p)
	case "cast":
		return c.executeCast(p)
	case "spells":
		return c.executeSpells(p)
	case "level", "lvl":
		return c.executeLevel(p)
	case "stats", "abilities", "attributes":
		return c.executeStats(p)
	case "unlock":
		return c.executeUnlock(p)
	case "shop", "list":
		return c.executeShop(p)
	case "buy", "purchase":
		return c.executeBuy(p)
	case "sell":
		return c.executeSell(p)
	case "gold", "money", "wallet":
		return c.executeGold(p)
	case "talk", "speak", "chat":
		return c.executeTalk(p)
	default:
		return fmt.Sprintf("Unknown command: %s. Type 'help' for available commands.", c.Name)
	}
}

func (c *Command) executeHelpWithPlayer(p PlayerInterface) string {
	// Get topic from args
	topic := ""
	if len(c.Args) > 0 {
		topic = strings.ToLower(strings.Join(c.Args, " "))
	}

	// Check if player is admin for showing admin commands
	isAdmin := false
	if p != nil {
		isAdmin = p.IsAdmin()
	}

	return c.getHelpText(topic, isAdmin)
}

func (c *Command) executeHelp() string {
	return c.getHelpText("", false)
}

func (c *Command) getHelpText(topic string, isAdmin bool) string {

	// Topic-specific help
	switch topic {
	case "look", "l", "examine", "ex":
		return `LOOK [target]
Look around your current location, or examine something specific.

Usage:
  look              - Look at the room
  look sword        - Examine an item in the room or your inventory
  look Bob          - Look at another player

Aliases: l, examine, ex`

	case "exits":
		return `EXITS
Display all available exits from your current location.

The exits command shows which directions you can travel and
where they lead.`

	case "move", "movement", "go", "north", "south", "east", "west", "up", "down":
		return `MOVEMENT
Move around the world using directions.

Usage:
  north, south, east, west - horizontal movement
  up, down                 - vertical movement (stairs)
  Shortcuts: n, s, e, w, u, d

You can also use: go <direction>`

	case "inventory", "inv", "i":
		return `INVENTORY
Show all items you are carrying, including their weight and type.

The inventory displays your total carry weight.
Default capacity: 100

Aliases: inv, i`

	case "get", "take", "pickup":
		return `GET <item>
Pick up an item from the current room.

Usage:
  get sword
  take rusty sword  (partial names work!)

Items have weight - you can only carry so much!

Aliases: take, pickup`

	case "drop":
		return `DROP <item>
Drop an item from your inventory into the current room.

Usage:
  drop sword
  drop rusty (partial names work!)`

	case "say":
		return `SAY <message>
Say something to everyone in the current room.

Usage:
  say Hello everyone!

Everyone in the same room will see your message.`

	case "tell":
		return `TELL <player> <message>
Send a private message to another player.

Usage:
  tell Bob Hey, how are you?

Only the target player will see your message.
Works with partial player names.`

	case "who":
		return `WHO
List all players currently online.`

	case "equipment", "eq":
		return `EQUIPMENT
Display all items you currently have equipped.

Shows each equipment slot (head, body, weapon, etc.) and the item
equipped in it, along with its stats (armor, damage).

Alias: eq`

	case "wield":
		return `WIELD <weapon>
Equip a weapon from your inventory to your weapon slot.

Usage:
  wield sword       - Wield a sword as your weapon
  wield dagger      - Wield a dagger

Two-handed weapons require both hands free (no off-hand item).`

	case "wear":
		return `WEAR <armor>
Put on armor from your inventory.

Usage:
  wear leather armor    - Wear leather armor on your body
  wear iron boots       - Wear boots on your feet

The armor will automatically go to the appropriate slot
based on its type (head, body, legs, feet, hands).`

	case "remove":
		return `REMOVE <item>
Remove an equipped item and put it back in your inventory.

Usage:
  remove sword      - Remove your equipped sword
  remove armor      - Remove your equipped armor

You must have enough carrying capacity to remove the item.`

	case "hold":
		return `HOLD <item>
Hold a miscellaneous item in your hand.

Usage:
  hold torch        - Hold a torch in your hand
  hold crystal      - Hold a crystal

Useful for torches, keys, and other items you want ready.`

	case "eat":
		return `EAT <food>
Consume a food item to restore health.

Usage:
  eat bread         - Eat bread to restore 10 HP
  eat apple         - Eat an apple to restore 8 HP
  eat roasted meat  - Eat roasted meat to restore 20 HP

The item will be consumed (removed from inventory).`

	case "drink":
		return `DRINK <drink/potion>
Drink a beverage or potion to restore mana or health.

Usage:
  drink water       - Drink water to restore 10 MP
  drink ale         - Drink ale to restore 5 MP
  drink healing potion - Drink a healing potion to restore 30 HP

Potions can restore health, mana, or both.
The item will be consumed (removed from inventory).`

	case "use":
		return `USE <consumable>
Use any consumable item (food, drink, potion).

Usage:
  use healing potion - Restore 30 HP
  use mana potion    - Restore 30 MP
  use elixir         - Restore 20 HP and 20 MP

Generic command that works with any consumable.
The item will be consumed (removed from inventory).`

	case "time":
		return `TIME
Display the current game time, day/night status, and server uptime.`

	case "sleep":
		return `SLEEP
Lie down and fall asleep for maximum regeneration.

While sleeping, you regenerate 5 HP and 3 MP every 10 seconds.
You cannot move while sleeping - use 'wake' to stand up.`

	case "wake":
		return `WAKE
Wake up from sleeping and stand.

Use this command when you're done resting.`

	case "stand":
		return `STAND
Stand up from sitting, resting, or sleeping.

Standing position regenerates 1 HP per 10 seconds.`

	case "attack", "kill", "hit":
		return `ATTACK <target>
Attack an NPC to initiate combat.

Usage:
  attack goblin     - Start fighting a goblin
  kill rat          - Start fighting a rat
  hit orc           - Start fighting an orc

Once in combat, rounds happen automatically every 3 seconds.
Use 'flee' to escape from combat.

Aliases: kill, hit`

	case "flee":
		return `FLEE
Escape from combat by fleeing to a random exit.

Usage:
  flee              - Run away from your current opponent

This will end combat and move you to an adjacent room.
Use this when you're losing a fight!`

	case "consider", "con":
		return `CONSIDER <target>
Assess the difficulty of fighting an NPC, or view your own stats.

Usage:
  consider goblin   - Check how dangerous the goblin is
  con orc           - Check the orc's level and stats
  consider self     - View your own character stats
  con me            - Alias for consider self

Shows the NPC's level, health, armor, damage, and XP reward.
Also gives a difficulty assessment based on level difference.

Alias: con`

	case "pray":
		return `PRAY
Pray at a sacred altar to restore your health and mana to full.

Usage:
  pray              - Kneel at the altar and pray

Requires an altar to be present in the room (such as in the Temple).
Your health and mana will be fully restored when you pray.`

	case "portal":
		return `PORTAL [floor]
Use a magical portal to fast travel between discovered tower floors.

Usage:
  portal            - List available destinations
  portal 0          - Travel to Ground Floor (City)
  portal 5          - Travel to Floor 5
  portal city       - Travel to Ground Floor (alias)

You must be in a room with a portal to use this command.
Portals are found at stairway landings throughout the tower.
Portals are automatically discovered when you enter a stairway room.`

	case "quit", "exit":
		return `QUIT
Disconnect from the game.

Alias: exit`

	case "cast":
		return `CAST <spell> [target]
Cast a spell from your spellbook.

Usage:
  cast heal           - Cast heal on yourself (restores 5% of your max HP)
  cast heal <player>  - Cast heal on another player in the same room
  cast flare <target> - Cast flare at an enemy NPC
  cast dazzle         - Stun all hostile creatures in the room

Spells cost mana and may have cooldowns.
Use 'spells' to see your available spells and their status.`

	case "spells":
		return `SPELLS
Display your known spells and their status.

Shows:
  - Spell name and description
  - Mana cost
  - Cooldown status (ready or seconds remaining)
  - Your current mana

New characters start with 'heal' and 'flare' spells.`

	case "level", "lvl":
		return `LEVEL
Display detailed level progression information.

Usage: level (or lvl)

Shows:
  - Current level
  - Experience points (current / needed for next level)
  - Visual progress bar
  - XP remaining to reach next level

Leveling increases your maximum Health (+10) and Mana (+5) per level.
Maximum level is 50.`

	case "stats", "abilities", "attributes":
		return `STATS
Display your ability scores and their modifiers.

Usage: stats (or abilities, attributes)

Shows your six core attributes:
  - Strength (STR)     - Physical power, melee damage
  - Dexterity (DEX)    - Agility, reflexes, ranged attacks
  - Constitution (CON) - Endurance, health
  - Intelligence (INT) - Mental acuity, spell power
  - Wisdom (WIS)       - Perception, spell resistance
  - Charisma (CHA)     - Social influence, leadership

Each score has a modifier calculated as: (score - 10) / 2
A score of 10 is average (+0), 15 is good (+2), 8 is below average (-1).`

	case "shop", "list":
		return `SHOP
View items available for purchase at a shop.

Usage:
  shop              - Display shop inventory and prices

You must be in a room with a shop (like the General Store).
Shows your current gold and available items for purchase.

Alias: list`

	case "buy", "purchase":
		return `BUY <item>
Purchase an item from a shop.

Usage:
  buy treasure key  - Buy a treasure key
  buy key           - Partial names work

You must be in a room with a shop and have enough gold.
Keys are automatically added to your key ring.

Alias: purchase`

	case "sell":
		return `SELL <item>
Sell an item from your inventory to a shop.

Usage:
  sell sword        - Sell the sword from your inventory
  sell rusty        - Partial names work

You must be in a room with a shop.
Items sell for 50% of their value (minimum 1 gold).
Items with no value cannot be sold.`

	case "gold", "money", "wallet":
		return `GOLD
Check your current gold balance.

Usage:
  gold              - Display how much gold you have

Aliases: money, wallet`

	case "talk", "speak", "chat":
		return `TALK <npc name>
Talk to an NPC to hear what they have to say.

Usage:
  talk merchant     - Talk to the merchant
  talk priestess    - Talk to a priestess
  speak guard       - Talk to a guard
  talk bard         - SAVE YOUR GAME! (costs 5 gold)

NPCs may give you helpful hints, lore, or just colorful dialogue.
Not all NPCs have something to say - monsters typically don't talk!

IMPORTANT: To save your progress, visit the bard in the tavern!
He will compose a song about your adventures for 5 gold.

Aliases: speak, chat`

	case "save":
		return `SAVE
Your progress is saved by visiting the wandering bard in the tavern.

The bard will compose a ballad about your adventures for 5 gold.
This immortalizes your deeds and saves your progress!

How to save:
  1. Go to The Weary Wanderer Tavern (from Town Square: south, south, east)
  2. Type: talk bard
  3. Pay 5 gold for your song

WARNING: If you disconnect without saving, you will lose all progress
since your last song! Always visit the bard before logging out.`

	case "tutorial", "guide", "aldric", "newplayer", "new":
		return `TUTORIAL / NEW PLAYER GUIDE

If you're new to the game, find Aldric the old guide in the Town Square!

  Type: talk aldric

He will explain everything you need to know:
  - How to explore the tower
  - Combat basics
  - Where to buy and sell items
  - How to save your progress (IMPORTANT!)
  - Portal travel system
  - And much more!

You can talk to Aldric anytime you need a refresher.

Quick Start:
  1. talk aldric     - Get the full tutorial
  2. pray            - Heal up at the Temple (go east first)
  3. Go south x3     - Head to the Tower Entrance
  4. up              - Enter the tower and begin your adventure!
  5. talk bard       - Save at the tavern when done (south, south, east from square)`

	case "":
		// Default help - show all commands
		generalHelp := `
Available Commands:
  help [topic]      - Show this help message or help for a specific command
  look (l)          - Look around the current room or examine something
  examine (ex)      - Alias for look
  exits             - Show available exits
  inventory (inv, i) - Show your inventory
  equipment (eq)    - Show equipped items
  consider self     - Show your character stats
  level (lvl)       - Show level progression and XP
  stats             - Show your ability scores

Movement:
  go <direction>    - Move in a direction
  north (n)         - Move north
  south (s)         - Move south
  east (e)          - Move east
  west (w)          - Move west
  up (u)            - Move up
  down (d)          - Move down

Items:
  get <item>        - Pick up an item from the room (also: take, pickup)
  drop <item>       - Drop an item from your inventory

Equipment:
  wield <weapon>    - Equip a weapon to your weapon slot
  wear <armor>      - Equip armor to the appropriate slot
  remove <item>     - Unequip an item back to inventory
  hold <item>       - Hold a misc item in your hand

Consumables:
  eat <food>        - Eat food to restore health
  drink <drink>     - Drink beverages/potions to restore mana or health
  use <item>        - Use any consumable item

Communication:
  say <message>     - Say something to everyone in the room
  tell <player> <message> - Send a private message to a player
  talk <npc>        - Talk to an NPC (also: speak, chat)
  who               - List all online players

Player State:
  sleep             - Lie down and sleep (best regeneration: 5 HP, 3 MP/tick)
  wake              - Wake up and stand
  stand             - Stand up from sitting/resting/sleeping

Combat:
  attack <npc>      - Attack an NPC to start combat (also: kill, hit)
  consider <npc>    - Assess NPC difficulty before fighting (also: con)
  consider self     - View your own stats and discovered cities
  flee              - Escape from combat to a random exit

Magic:
  cast <spell> [target] - Cast a spell (e.g., cast heal, cast flare goblin)
  spells            - List your known spells and their status

Special Locations:
  pray              - Pray at an altar to restore full health
  portal [floor]    - Fast travel between discovered tower floors
  unlock <dir>      - Unlock a locked door with a key from your key ring

Shop (at General Store):
  shop              - View items for sale
  buy <item>        - Purchase an item
  sell <item>       - Sell an item (50% of item value)
  gold              - Check your gold balance

Saving Progress:
  Visit the bard in the tavern and talk to him to save your progress!
  The bard will write a song about your adventures (for a small fee).

Other:
  time              - Show server uptime
  quit              - Disconnect from the game

Type 'help <command>' for more information about a specific command.
`
		// Add admin section for admins
		if isAdmin {
			return generalHelp + `
Admin Commands:
  admin             - Show admin commands (admin help for details)
`
		}
		return generalHelp

	default:
		return fmt.Sprintf("No help available for '%s'.\nType 'help' for a list of commands.", topic)
	}
}

func (c *Command) executeLook(p PlayerInterface) string {
	// If no arguments, look at the room
	if len(c.Args) == 0 {
		roomIface := p.GetCurrentRoom()
		room, ok := roomIface.(RoomInterface)
		if !ok {
			return "Internal error: invalid room type"
		}

		// Get time-appropriate description
		serverIface := p.GetServer()
		server, ok := serverIface.(ServerInterface)
		if !ok {
			// Fallback to default description if server not available
			return room.GetDescriptionForPlayer(p.GetName())
		}

		// Select description based on time of day
		baseDesc := room.GetBaseDescription()
		if server.IsDay() && room.GetDescriptionDay() != "" {
			baseDesc = room.GetDescriptionDay()
		} else if server.IsNight() && room.GetDescriptionNight() != "" {
			baseDesc = room.GetDescriptionNight()
		}

		// Build full description with time-based variant
		return room.GetDescriptionForPlayerWithCustomDesc(p.GetName(), baseDesc)
	}

	// Otherwise, examine a specific object
	targetName := c.GetItemName()

	// Get current room
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// First, check if it's an item in the room
	item, foundInRoom := room.FindItem(targetName)
	if foundInRoom {
		return fmt.Sprintf("%s\n%s", item.Name, item.Description)
	}

	// Next, check if it's an item in player's inventory
	item, foundInInventory := p.FindItem(targetName)
	if foundInInventory {
		return fmt.Sprintf("%s\n%s", item.Name, item.Description)
	}

	// Check if it's a room feature
	targetLower := strings.ToLower(targetName)
	if room.HasFeature(targetLower) {
		switch targetLower {
		case "altar":
			return "A sacred altar carved from white marble. It radiates a gentle warmth and divine energy. You could pray here to seek healing."
		case "portal":
			return "A shimmering portal of swirling blue and silver energy. It offers travel to tower floors you have discovered. Type 'portal' to see available destinations."
		default:
			return fmt.Sprintf("You see a %s here.", targetName)
		}
	}

	// Finally, check if it's another player in the room
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if ok {
		targetPlayerIface := server.FindPlayer(targetName)
		if targetPlayerIface != nil {
			targetPlayer, ok := targetPlayerIface.(PlayerInterface)
			if ok {
				// Check if the target player is in the same room
				targetRoomIface := targetPlayer.GetCurrentRoom()
				targetRoom, ok := targetRoomIface.(RoomInterface)
				if ok && targetRoom.GetID() == room.GetID() {
					return fmt.Sprintf("%s is standing here.", targetPlayer.GetName())
				}
			}
		}
	}

	return fmt.Sprintf("You don't see '%s' here.", targetName)
}

func (c *Command) executeMove(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Go where? Specify a direction (north, south, east, west, up, down)"); err != nil {
		return err.Error()
	}
	direction := strings.ToLower(c.Args[0])
	return c.executeMoveDirection(p, direction)
}

func (c *Command) executeMoveDirection(p PlayerInterface, direction string) string {
	// Check if player can move (not sleeping)
	currentState := p.GetState()
	if currentState == "sleeping" {
		return "You can't move while sleeping! Wake up first."
	}

	currentRoomIface := p.GetCurrentRoom()
	currentRoom, ok := currentRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasts and floor generation
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Check if exit is locked
	if currentRoom.IsExitLocked(direction) {
		keyID := currentRoom.GetExitKeyRequired(direction)
		return fmt.Sprintf("The way %s is locked. You need a key to unlock it. (Requires: %s)", direction, keyID)
	}

	nextRoomIface := currentRoom.GetExit(direction)

	// Handle stairs - if going up from a stairs room and no exit exists, generate next floor
	if nextRoomIface == nil && direction == "up" && currentRoom.HasFeature("stairs_up") {
		currentFloor := currentRoom.GetFloor()
		nextFloorRoom, err := server.GenerateNextFloor(currentFloor)
		if err != nil {
			return fmt.Sprintf("The stairs seem to lead nowhere... (%v)", err)
		}
		if nextFloorRoom == nil {
			return "The stairs seem to lead nowhere."
		}
		nextRoomIface = nextFloorRoom
	}

	if nextRoomIface == nil {
		return fmt.Sprintf("You can't go %s from here.", direction)
	}

	nextRoom, ok := nextRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Broadcast exit message to current room
	exitMsg := fmt.Sprintf("%s leaves %s.\n", p.GetName(), direction)
	server.BroadcastToRoom(currentRoom.GetID(), exitMsg, p)

	// Move the player
	p.MoveTo(nextRoomIface)

	// Broadcast enter message to new room
	// Determine opposite direction for enter message
	oppositeDir := getOppositeDirection(direction)
	enterMsg := fmt.Sprintf("%s arrives from the %s.\n", p.GetName(), oppositeDir)
	server.BroadcastToRoom(nextRoom.GetID(), enterMsg, p)

	// Track floor portal discovery - if room has a portal, mark the floor as discovered
	if nextRoom.HasFeature("portal") {
		floorNum := nextRoom.GetFloor()
		if !p.HasDiscoveredPortal(floorNum) {
			p.DiscoverPortal(floorNum)
			p.SendMessage(fmt.Sprintf("\n*** You have discovered a portal on %s! ***\n", getFloorDisplayName(floorNum)))
		}
	}

	return fmt.Sprintf("You move %s.\n\n%s", direction, nextRoom.GetDescriptionForPlayer(p.GetName()))
}

// getOppositeDirection returns the opposite direction for enter messages
func getOppositeDirection(direction string) string {
	opposites := map[string]string{
		"north": "south",
		"south": "north",
		"east":  "west",
		"west":  "east",
		"up":    "above",
		"down":  "below",
	}
	if opposite, ok := opposites[direction]; ok {
		return opposite
	}
	return "somewhere"
}

func (c *Command) executeTake(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Take what? Specify an item to pick up."); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Check if there's a shop in this room - can't just take shop merchandise
	if findShopNPC(room) != nil {
		return "You can't just take items from a shop! Use 'buy <item>' to purchase."
	}

	// Try to find the item in the room (supports partial matching)
	foundItem, exists := room.FindItem(itemName)
	if !exists {
		return fmt.Sprintf("You don't see '%s' here.", itemName)
	}

	// Keys go directly to key ring (no weight/capacity check)
	if foundItem.Type == items.Key {
		removedItem, removed := room.RemoveItem(foundItem.Name)
		if removed {
			p.AddKey(removedItem)
			return fmt.Sprintf("You take the %s and add it to your key ring.", foundItem.Name)
		}
		return fmt.Sprintf("You can't take the %s.", foundItem.Name)
	}

	// Check if player can carry the item
	if !p.CanCarry(foundItem) {
		return fmt.Sprintf("You can't carry the %s. It's too heavy! (%.1f)", foundItem.Name, foundItem.Weight)
	}

	// Remove from room and add to inventory
	removedItem, removed := room.RemoveItem(foundItem.Name)
	if removed {
		p.AddItem(removedItem)
		return fmt.Sprintf("You take the %s.", foundItem.Name)
	}

	return fmt.Sprintf("You can't take the %s.", foundItem.Name)
}

func (c *Command) executeDrop(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Drop what? Specify an item to drop."); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Try to find the item in inventory (supports partial matching)
	foundItem, exists := p.FindItem(itemName)
	if !exists {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Remove from inventory and add to room
	removedItem, removed := p.RemoveItem(foundItem.Name)
	if removed {
		roomIface := p.GetCurrentRoom()
		room, ok := roomIface.(RoomInterface)
		if !ok {
			return "Internal error: invalid room type"
		}
		room.AddItem(removedItem)
		return fmt.Sprintf("You drop the %s.", foundItem.Name)
	}

	return fmt.Sprintf("You can't drop the %s.", foundItem.Name)
}

func (c *Command) executeInventory(p PlayerInterface) string {
	inventory := p.GetInventory()
	keyRing := p.GetKeyRing()
	currentWeight := p.GetCurrentWeight()

	var result string

	// Show gold
	result = fmt.Sprintf("Gold: %d\n", p.GetGold())

	// Show inventory
	if len(inventory) == 0 {
		result += "\nYour inventory is empty.\n"
	} else {
		result += "\nYou are carrying:\n"
		for _, item := range inventory {
			result += fmt.Sprintf("  - %s (%.1f, %s)\n", item.Name, item.Weight, item.Type.String())
		}
	}

	result += fmt.Sprintf("\nTotal weight: %.1f\n", currentWeight)

	// Show key ring
	if len(keyRing) > 0 {
		result += "\nKey Ring:\n"
		for _, key := range keyRing {
			result += fmt.Sprintf("  - %s\n", key.Name)
		}
	}

	return result
}

func (c *Command) executeSay(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Say what?"); err != nil {
		return err.Error()
	}

	message := c.GetItemName() // Reusing GetItemName to join all args

	// Get the current room
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasting and chat filter
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Apply chat filter if enabled
	filteredMessage := message
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(message)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "say",
				"room", room.GetID(),
				"original", message,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your message contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered message
			filteredMessage = result.Filtered
		}
	}

	broadcastMsg := fmt.Sprintf("%s says: \"%s\"\n", p.GetName(), filteredMessage)
	server.BroadcastToRoom(room.GetID(), broadcastMsg, p)

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_SAY",
		"player", p.GetName(),
		"room", room.GetID(),
		"message", filteredMessage)

	return fmt.Sprintf("You say: \"%s\"", filteredMessage)
}

func (c *Command) executeQuit(p PlayerInterface) string {
	p.Disconnect()
	return "Goodbye!"
}

func (c *Command) executeWho(p PlayerInterface) string {
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	players := server.GetOnlinePlayers()

	if len(players) == 0 {
		return "No players online."
	}

	result := "Online Players:\n"
	for _, playerName := range players {
		result += fmt.Sprintf("  - %s\n", playerName)
	}
	return result
}

func (c *Command) executeTell(p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: tell <player> <message>"); err != nil {
		return err.Error()
	}

	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Try to find player by matching progressively longer portions of args
	// This allows player names with spaces like "Bob Johnson"
	var target PlayerInterface
	var messageStartIndex int

	// Try matching from longest possible name to shortest
	for i := len(c.Args) - 1; i >= 0; i-- {
		candidateName := strings.Join(c.Args[0:i+1], " ")
		targetIface := server.FindPlayer(candidateName)
		if targetIface != nil {
			var ok bool
			target, ok = targetIface.(PlayerInterface)
			if ok {
				messageStartIndex = i + 1
				break
			}
		}
	}

	// If no player found, return error
	if target == nil {
		return fmt.Sprintf("Player '%s' not found.", c.Args[0])
	}

	// Check if there's a message after the player name
	if messageStartIndex >= len(c.Args) {
		return "Usage: tell <player> <message>"
	}

	message := strings.Join(c.Args[messageStartIndex:], " ")

	// Apply chat filter if enabled
	filteredMessage := message
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(message)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "tell",
				"recipient", target.GetName(),
				"original", message,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your message contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered message
			filteredMessage = result.Filtered
		}
	}

	// Send message to target
	target.SendMessage(fmt.Sprintf("%s tells you: \"%s\"\n", p.GetName(), filteredMessage))

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_TELL",
		"sender", p.GetName(),
		"recipient", target.GetName(),
		"message", filteredMessage)

	return fmt.Sprintf("You tell %s: \"%s\"", target.GetName(), filteredMessage)
}

func (c *Command) executeExits(p PlayerInterface) string {
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	exits := room.GetExits()
	if len(exits) == 0 {
		return "There are no obvious exits."
	}

	result := "Obvious exits:\n"
	for direction, roomName := range exits {
		result += fmt.Sprintf("  %-6s - %s\n", direction, roomName)
	}

	return result
}

func (c *Command) executeEquipment(p PlayerInterface) string {
	equipment := p.GetEquipment()

	if len(equipment) == 0 {
		return "You are not wearing any equipment."
	}

	result := "You are wearing:\n"

	// Display equipment in a logical order
	slots := []items.EquipmentSlot{
		items.SlotHead,
		items.SlotBody,
		items.SlotLegs,
		items.SlotFeet,
		items.SlotHands,
		items.SlotWeapon,
		items.SlotOffHand,
		items.SlotHeld,
	}

	for _, slot := range slots {
		if item, equipped := equipment[slot]; equipped {
			result += fmt.Sprintf("  <%s> %s", slot.String(), item.Name)

			// Add stats if available
			if item.Damage > 0 {
				result += fmt.Sprintf(" (damage: %d)", item.Damage)
			}
			if item.Armor > 0 {
				result += fmt.Sprintf(" (armor: %d)", item.Armor)
			}
			if item.TwoHanded {
				result += " [two-handed]"
			}
			result += "\n"
		}
	}

	return result
}

func (c *Command) executeWield(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: wield <weapon>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the weapon in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Check if it's a weapon
	if item.Type != items.Weapon {
		return fmt.Sprintf("You can't wield %s - it's not a weapon!", item.Name)
	}

	// Try to equip it
	if err := p.EquipItem(item); err != nil {
		return err.Error()
	}

	// Remove from inventory
	p.RemoveItem(item.Name)

	// Success message
	if item.TwoHanded {
		return fmt.Sprintf("You wield %s with both hands.", item.Name)
	}
	return fmt.Sprintf("You wield %s.", item.Name)
}

func (c *Command) executeWear(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: wear <armor>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the armor in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Check if it's armor
	if item.Type != items.Armor {
		return fmt.Sprintf("You can't wear %s - it's not armor!", item.Name)
	}

	// Try to equip it
	if err := p.EquipItem(item); err != nil {
		return err.Error()
	}

	// Remove from inventory
	p.RemoveItem(item.Name)

	// Success message
	return fmt.Sprintf("You wear %s on your %s.", item.Name, item.Slot.String())
}

func (c *Command) executeRemove(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: remove <item>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the item in equipment
	item, slot, found := p.FindEquippedItem(itemName)
	if !found {
		return fmt.Sprintf("You are not wearing '%s'.", itemName)
	}

	// Check if we have space in inventory
	if !p.CanCarry(item) {
		return "You are carrying too much to remove that item."
	}

	// Unequip the item
	_, err := p.UnequipItem(slot)
	if err != nil {
		return err.Error()
	}

	// Add to inventory
	p.AddItem(item)

	// Success message
	return fmt.Sprintf("You remove %s.", item.Name)
}

func (c *Command) executeHold(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: hold <item>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the item in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Set the item to be held (SlotHeld)
	// This allows holding misc items like torches, keys, etc.
	item.Slot = items.SlotHeld

	// Try to equip it
	if err := p.EquipItem(item); err != nil {
		return err.Error()
	}

	// Remove from inventory
	p.RemoveItem(item.Name)

	// Success message
	return fmt.Sprintf("You hold %s in your hand.", item.Name)
}

func (c *Command) executeEat(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: eat <food>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the item in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Check if it's food
	if item.Type != items.Food {
		return fmt.Sprintf("You can't eat %s - it's not food!", item.Name)
	}

	// Check if it's consumable
	if !item.Consumable {
		return fmt.Sprintf("%s is not edible.", item.Name)
	}

	// Consume the item
	result := p.ConsumeItem(item)

	// Remove from inventory (item is consumed)
	p.RemoveItem(item.Name)

	return result
}

func (c *Command) executeDrink(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: drink <drink>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the item in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Check if it's a drink or potion
	if item.Type != items.Drink && item.Type != items.Potion {
		return fmt.Sprintf("You can't drink %s!", item.Name)
	}

	// Check if it's consumable
	if !item.Consumable {
		return fmt.Sprintf("%s is not drinkable.", item.Name)
	}

	// Consume the item
	result := p.ConsumeItem(item)

	// Remove from inventory (item is consumed)
	p.RemoveItem(item.Name)

	return result
}

func (c *Command) executeUse(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: use <item>"); err != nil {
		return err.Error()
	}

	itemName := c.GetItemName()

	// Find the item in inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Check if it's consumable
	if !item.Consumable {
		return fmt.Sprintf("You can't use %s like that.", item.Name)
	}

	// Consume the item
	result := p.ConsumeItem(item)

	// Remove from inventory (item is consumed)
	p.RemoveItem(item.Name)

	return result
}

func (c *Command) executeTime(p PlayerInterface) string {
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get game clock
	gameClockIface := server.GetGameClock()
	gameClock, ok := gameClockIface.(*gametime.GameClock)
	if !ok {
		return "Internal error: game clock not available"
	}

	timeDesc := gameClock.GetDescriptiveTime()
	timeOfDay := gameClock.GetTimeOfDay()

	// Determine day/night status message
	var periodMsg string
	if gameClock.IsDay() {
		minutesUntilNight := gameClock.GetMinutesUntilNextPeriod()
		periodMsg = fmt.Sprintf("It is daytime. Night falls in %.1f minutes.", minutesUntilNight)
	} else {
		minutesUntilDay := gameClock.GetMinutesUntilNextPeriod()
		periodMsg = fmt.Sprintf("It is nighttime. Dawn breaks in %.1f minutes.", minutesUntilDay)
	}

	uptime := server.GetUptime()
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	return fmt.Sprintf(
		"%s (%s).\n%s\n\nServer uptime: %d hours, %d minutes, %d seconds",
		timeDesc,
		timeOfDay,
		periodMsg,
		hours, minutes, seconds,
	)
}

func (c *Command) executeSleep(p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already sleeping
	if currentState == "sleeping" {
		return "You are already sleeping."
	}

	// Can't sleep while fighting
	if currentState == "fighting" {
		return "You can't sleep while fighting!"
	}

	// Change to sleeping state
	err := p.SetState("sleeping")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You lie down and fall asleep."
}

func (c *Command) executeWake(p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already awake
	if currentState != "sleeping" {
		return "You are already awake."
	}

	// Change to standing state
	err := p.SetState("standing")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You wake up and stand."
}

func (c *Command) executeStand(p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already standing
	if currentState == "standing" {
		return "You are already standing."
	}

	// Can't stand while fighting (you're always standing in combat)
	if currentState == "fighting" {
		return "You are already standing (fighting)."
	}

	// Change to standing state
	err := p.SetState("standing")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You stand up."
}


func (c *Command) executeAttack(p PlayerInterface, w WorldInterface) string {
	// Check if server is in pilgrim mode
	server := p.GetServer().(ServerInterface)
	if server.IsPilgrimMode() {
		return "This server is in pilgrim mode - exploration only!"
	}

	// Check if already in combat
	if p.IsInCombat() {
		return "You are already fighting!"
	}

	// Require target name
	if err := c.RequireArgs(1, "Usage: attack <target>"); err != nil {
		return err.Error()
	}

	targetName := c.GetItemName()
	room := p.GetCurrentRoom().(*world.Room)

	// Find the NPC in the room
	npc := room.FindNPC(targetName)
	if npc == nil {
		return fmt.Sprintf("You don't see '%s' here.", targetName)
	}

	// Check if NPC is attackable
	if !npc.IsAttackable() {
		return fmt.Sprintf("You cannot attack %s!", npc.GetName())
	}

	// Check if joining an ongoing fight
	joiningFight := npc.IsInCombat()

	// Start combat for both player and NPC
	p.StartCombat(npc.GetName())
	npc.StartCombat(p.GetName())

	// Broadcast to room
	if joiningFight {
		server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s joins the fight against %s!", p.GetName(), npc.GetName()), p)
		return fmt.Sprintf("You join the fight against %s!\n\nType 'flee' to escape.", npc.GetName())
	} else {
		server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s attacks %s!", p.GetName(), npc.GetName()), p)
		return fmt.Sprintf("You attack %s!\n\nCombat initiated! Type 'flee' to escape.", npc.GetName())
	}
}

func (c *Command) executeFlee(p PlayerInterface, w WorldInterface) string {
	// Check if in combat
	if !p.IsInCombat() {
		return "You aren't fighting anyone!"
	}

	room := p.GetCurrentRoom().(*world.Room)

	// Find the NPC player is fighting
	npc := room.FindNPC(p.GetCombatTarget())
	if npc == nil {
		// NPC not found (dead?), end combat anyway
		p.EndCombat()
		return "Your opponent has vanished!"
	}

	// End combat for player and remove from NPC's target list
	p.EndCombat()
	npc.EndCombat(p.GetName())

	// Try to move to a random exit
	exits := room.GetExits()
	if len(exits) == 0 {
		return "You can't escape - there are no exits!"
	}

	// Get first available exit (simple implementation)
	var direction string
	for dir := range exits {
		direction = dir
		break
	}

	// Move to the exit
	targetRoom := room.GetExit(direction)
	if targetRoom == nil {
		return "Flee failed - exit is blocked!"
	}

	// Broadcast flee message to room (including remaining fighters)
	server := p.GetServer().(ServerInterface)
	server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s flees from combat %s!", p.GetName(), direction), p)

	// Move player
	p.MoveTo(targetRoom)

	return fmt.Sprintf("You flee %s!\n\n%s", direction, p.GetCurrentRoom().(*world.Room).GetDescriptionForPlayer(p.GetName()))
}

func (c *Command) executeConsider(p PlayerInterface) string {
	// Require target name
	if err := c.RequireArgs(1, "Usage: consider <target>"); err != nil {
		return err.Error()
	}

	targetName := c.GetItemName()
	targetLower := strings.ToLower(targetName)

	// Handle "consider self" or "consider me"
	if targetLower == "self" || targetLower == "me" {
		return c.executeConsiderSelf(p)
	}

	room := p.GetCurrentRoom().(*world.Room)

	// Find the NPC in the room
	npc := room.FindNPC(targetName)
	if npc == nil {
		return fmt.Sprintf("You don't see '%s' here.", targetName)
	}

	// Compare levels
	levelDiff := npc.GetLevel() - p.GetLevel()
	var difficulty string

	switch {
	case levelDiff <= -5:
		difficulty = "trivial (no challenge)"
	case levelDiff <= -3:
		difficulty = "easy (minor challenge)"
	case levelDiff <= -1:
		difficulty = "manageable (fair fight)"
	case levelDiff == 0:
		difficulty = "even match (50/50)"
	case levelDiff <= 2:
		difficulty = "challenging (tough fight)"
	case levelDiff <= 4:
		difficulty = "difficult (very dangerous)"
	case levelDiff <= 6:
		difficulty = "deadly (you will likely die)"
	default:
		difficulty = "impossible (certain death)"
	}

	return fmt.Sprintf(`%s (Level %d)
Health: %d/%d
Difficulty: %s
Armor: %d | Damage: %d | XP Reward: %d`,
		npc.GetName(),
		npc.GetLevel(),
		npc.GetHealth(),
		npc.GetMaxHealth(),
		difficulty,
		npc.Armor,
		npc.Damage,
		npc.GetExperience(),
	)
}

// executeConsiderSelf shows the player their own stats
func (c *Command) executeConsiderSelf(p PlayerInterface) string {
	level := p.GetLevel()
	xp := p.GetExperience()

	// Calculate XP progress
	var xpLine string
	if level >= leveling.MaxPlayerLevel {
		xpLine = fmt.Sprintf("Experience: %d (MAX LEVEL)", xp)
	} else {
		xpNeeded := leveling.XPForLevel(level + 1)
		xpCurrent := leveling.XPForLevel(level)
		xpProgress := xp - xpCurrent
		xpRequired := xpNeeded - xpCurrent
		percent := 0
		if xpRequired > 0 {
			percent = (xpProgress * 100) / xpRequired
		}
		xpLine = fmt.Sprintf("Experience: %d / %d (%d%% to level %d)", xp, xpNeeded, percent, level+1)
	}

	return fmt.Sprintf(`=== %s ===
Level:      %d
%s
Health:     %d/%d
Mana:       %d/%d
State:      %s`,
		p.GetName(),
		level,
		xpLine,
		p.GetHealth(),
		p.GetMaxHealth(),
		p.GetMana(),
		p.GetMaxMana(),
		p.GetState(),
	)
}

// executeLevel shows detailed level progression information
func (c *Command) executeLevel(p PlayerInterface) string {
	level := p.GetLevel()
	xp := p.GetExperience()

	var result strings.Builder
	result.WriteString("=== Level Progress ===\n")
	result.WriteString(fmt.Sprintf("Current Level: %d", level))

	if level >= leveling.MaxPlayerLevel {
		result.WriteString(" (MAX)\n")
		result.WriteString(fmt.Sprintf("Total Experience: %d\n", xp))
		result.WriteString("\nYou have reached the maximum level!")
	} else {
		result.WriteString("\n")

		xpNeeded := leveling.XPForLevel(level + 1)
		xpCurrent := leveling.XPForLevel(level)
		xpProgress := xp - xpCurrent
		xpRequired := xpNeeded - xpCurrent
		xpToGo := xpNeeded - xp

		percent := 0
		if xpRequired > 0 {
			percent = (xpProgress * 100) / xpRequired
		}

		result.WriteString(fmt.Sprintf("Experience: %d / %d\n", xp, xpNeeded))

		// Build progress bar (20 characters wide)
		barWidth := 20
		filled := (percent * barWidth) / 100
		bar := strings.Repeat("#", filled) + strings.Repeat(".", barWidth-filled)
		result.WriteString(fmt.Sprintf("Progress: [%s] %d%%\n", bar, percent))

		result.WriteString(fmt.Sprintf("XP to next level: %d", xpToGo))
	}

	return result.String()
}

// executeStats shows the player's ability scores
func (c *Command) executeStats(p PlayerInterface) string {
	var result strings.Builder
	result.WriteString("=== Ability Scores ===\n")

	// Build ability score display
	abilities := []struct {
		name  string
		short string
		score int
	}{
		{"Strength", "STR", p.GetStrength()},
		{"Dexterity", "DEX", p.GetDexterity()},
		{"Constitution", "CON", p.GetConstitution()},
		{"Intelligence", "INT", p.GetIntelligence()},
		{"Wisdom", "WIS", p.GetWisdom()},
		{"Charisma", "CHA", p.GetCharisma()},
	}

	for _, a := range abilities {
		mod := stats.Modifier(a.score)
		modStr := fmt.Sprintf("%+d", mod)
		result.WriteString(fmt.Sprintf("  %-12s (%s): %2d (%s)\n", a.name, a.short, a.score, modStr))
	}

	return result.String()
}

// getCityDisplayName returns a friendly display name for a city room ID
func getCityDisplayName(roomID string) string {
	// Map known city IDs to friendly names
	switch roomID {
	case "city_square":
		return "Town Square (Starting City)"
	default:
		// For generated cities, use a cleaner name
		if strings.HasPrefix(roomID, "gen_") {
			return "Frontier Settlement"
		}
		return roomID
	}
}

// getFloorDisplayName returns a friendly display name for a floor number
func getFloorDisplayName(floor int) string {
	switch floor {
	case 0:
		return "Ground Floor (City)"
	default:
		if floor%10 == 0 {
			return fmt.Sprintf("Floor %d (Boss)", floor)
		}
		return fmt.Sprintf("Floor %d", floor)
	}
}

func (c *Command) executePassword(p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: password <old_password> <new_password>"); err != nil {
		return err.Error()
	}

	oldPassword := c.Args[0]
	newPassword := c.Args[1]

	// Validate new password length
	if len(newPassword) < 4 {
		return "New password must be at least 4 characters."
	}

	// Get database from server
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	dbIface := server.GetDatabase()
	if dbIface == nil {
		return "Password change is not available."
	}

	db, ok := dbIface.(*database.Database)
	if !ok {
		return "Internal error: invalid database type"
	}

	accountID := p.GetAccountID()
	if accountID == 0 {
		return "Account not found."
	}

	// Verify old password and change to new password
	if err := db.ChangePasswordWithVerify(accountID, oldPassword, newPassword); err != nil {
		if errors.Is(err, database.ErrInvalidCredentials) {
			return "Old password is incorrect."
		}
		return fmt.Sprintf("Failed to change password: %v", err)
	}

	return "Password changed successfully."
}

// ==================== ROOM FEATURE COMMANDS ====================

// executePray allows players to heal at an altar
func (c *Command) executePray(p PlayerInterface) string {
	// Check player state - can't pray while fighting or sleeping
	state := p.GetState()
	if state == "Fighting" {
		return "You can't pray while fighting!"
	}
	if state == "Sleeping" {
		return "You are asleep and can't pray. Wake up first."
	}

	// Get current room and check for altar feature
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return "You are nowhere?"
	}

	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if !room.HasFeature("altar") {
		return "There is nothing to pray at here."
	}

	// Check if already at full health and mana
	if p.GetHealth() >= p.GetMaxHealth() && p.GetMana() >= p.GetMaxMana() {
		return "You kneel before the altar and offer a prayer of thanks. You feel at peace."
	}

	// Restore the player to full health and mana
	healAmount := p.HealToFull()
	manaAmount := p.RestoreManaToFull()

	// Get the server to broadcast the prayer action
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Broadcast the prayer action to the room
	server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s kneels before the altar and prays.\n", p.GetName()), p)

	// Build response based on what was restored
	if healAmount > 0 && manaAmount > 0 {
		return fmt.Sprintf("You kneel before the altar and pray. A warm, golden light washes over you, restoring your body and spirit. (+%d HP, +%d Mana)", healAmount, manaAmount)
	} else if healAmount > 0 {
		return fmt.Sprintf("You kneel before the altar and pray. A warm, golden light washes over you, healing your wounds. (+%d HP)", healAmount)
	} else {
		return fmt.Sprintf("You kneel before the altar and pray. A calm energy fills your mind, restoring your spirit. (+%d Mana)", manaAmount)
	}
}

// executePortal allows players to fast travel between discovered tower floors
func (c *Command) executePortal(p PlayerInterface) string {
	// Check player state - can't portal while fighting or sleeping
	state := p.GetState()
	if state == "Fighting" {
		return "You can't use the portal while fighting!"
	}
	if state == "Sleeping" {
		return "You are asleep and can't use the portal. Wake up first."
	}

	// Get current room and check for portal feature
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return "You are nowhere?"
	}

	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if !room.HasFeature("portal") {
		return "There is no portal here."
	}

	// Get the server for world access
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Get discovered floor portals (excluding current floor)
	discoveredPortals := p.GetDiscoveredPortals()
	currentFloor := room.GetFloor()
	currentRoomID := room.GetID()

	// Filter out current floor
	availableFloors := make([]int, 0)
	for _, floor := range discoveredPortals {
		if floor != currentFloor {
			availableFloors = append(availableFloors, floor)
		}
	}

	// If no arguments, show available destinations
	if len(c.Args) == 0 {
		if len(availableFloors) == 0 {
			return "The portal shimmers before you, but you haven't discovered any other floors to travel to.\nClimb the tower and find stairway landings with portals!"
		}

		var sb strings.Builder
		sb.WriteString("The portal shimmers with arcane energy. Available destinations:\n\n")
		for _, floor := range availableFloors {
			floorName := getFloorDisplayName(floor)
			sb.WriteString(fmt.Sprintf("  - %s (portal %d)\n", floorName, floor))
		}
		sb.WriteString("\nUsage: portal <floor number>")
		return sb.String()
	}

	// Parse floor number from argument
	destArg := strings.TrimSpace(c.Args[0])
	destFloor := -1

	// Handle special names
	switch strings.ToLower(destArg) {
	case "city", "town", "ground":
		destFloor = 0
	default:
		// Try to parse as number
		_, err := fmt.Sscanf(destArg, "%d", &destFloor)
		if err != nil {
			return fmt.Sprintf("Invalid floor number: '%s'. Type 'portal' to see available destinations.", destArg)
		}
	}

	// Check if player has discovered this floor
	if !p.HasDiscoveredPortal(destFloor) {
		return fmt.Sprintf("You haven't discovered a portal on floor %d. Type 'portal' to see available destinations.", destFloor)
	}

	// Can't portal to current floor
	if destFloor == currentFloor {
		return "You're already on this floor!"
	}

	// Get the destination room
	destRoom := w.GetFloorPortalRoom(destFloor)
	if destRoom == nil {
		return fmt.Sprintf("Floor %d doesn't have a portal room.", destFloor)
	}

	destRoomID := destRoom.GetID()

	// Broadcast departure
	server.BroadcastToRoom(currentRoomID, fmt.Sprintf("%s steps through the portal and vanishes!\n", p.GetName()), p)

	// Move the player
	p.MoveTo(destRoom)

	// Broadcast arrival
	server.BroadcastToRoom(destRoomID, fmt.Sprintf("%s emerges from the portal in a flash of light!\n", p.GetName()), p)

	return fmt.Sprintf("You step through the shimmering portal...\n\nYou emerge on %s!\n\n%s", getFloorDisplayName(destFloor), destRoom.GetDescriptionForPlayer(p.GetName()))
}

// getPortalCommandName returns a short name for use in portal commands
func getPortalCommandName(roomID string) string {
	switch roomID {
	case "city_square":
		return "town"
	default:
		if strings.HasPrefix(roomID, "gen_") {
			return "frontier"
		}
		return roomID
	}
}

// ==================== SPELL COMMANDS ====================

// executeCast handles casting spells
func (c *Command) executeCast(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: cast <spell> [target]"); err != nil {
		return err.Error()
	}

	// Get the spell registry from server
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	registry := server.GetSpellRegistry()
	if registry == nil {
		return "Magic is not available."
	}

	// Parse spell name (first argument)
	spellName := strings.ToLower(c.Args[0])

	// Look up the spell
	spell, exists := registry.GetSpell(spellName)
	if !exists {
		return fmt.Sprintf("Unknown spell: '%s'. Type 'spells' to see your known spells.", spellName)
	}

	// Check if player knows the spell
	if !p.HasSpell(spell.ID) {
		return fmt.Sprintf("You don't know the spell '%s'.", spell.Name)
	}

	// Check if player has enough mana
	if p.GetMana() < spell.ManaCost {
		return fmt.Sprintf("Not enough mana to cast %s. (Need %d, have %d)", spell.Name, spell.ManaCost, p.GetMana())
	}

	// Check if spell is on cooldown
	onCooldown, remaining := p.IsSpellOnCooldown(spell.ID)
	if onCooldown {
		return fmt.Sprintf("%s is on cooldown. (%ds remaining)", spell.Name, remaining)
	}

	// Get target if provided
	var targetName string
	if len(c.Args) > 1 {
		targetName = strings.Join(c.Args[1:], " ")
	}

	// Check for room-wide spells first (no target needed)
	if spell.CanTargetRoomEnemies() {
		return c.castRoomSpell(p, spell)
	}

	// Determine how to cast based on target and spell capabilities
	if targetName == "" {
		// No target specified - cast on self if possible
		if spell.CanTargetSelf() {
			return c.castSelfSpell(p, spell)
		}
		// Spell requires a target
		return fmt.Sprintf("Cast %s at whom? Usage: cast %s <target>", spell.Name, spell.Name)
	}

	// Target specified - determine if it's a player or NPC
	room := p.GetCurrentRoom().(*world.Room)

	// First check if target is a player in the room
	if spell.CanTargetAlly() {
		if targetPlayerIface := server.FindPlayer(targetName); targetPlayerIface != nil {
			targetPlayer, ok := targetPlayerIface.(PlayerInterface)
			if ok {
				// Verify target is in same room
				targetRoom := targetPlayer.GetCurrentRoom().(*world.Room)
				if targetRoom.GetID() == room.GetID() {
					return c.castAllySpell(p, spell, targetPlayer)
				}
			}
		}
	}

	// Then check if target is an NPC
	if spell.CanTargetEnemy() {
		if npc := room.FindNPC(targetName); npc != nil {
			return c.castEnemySpell(p, spell, npc, room)
		}
	}

	// Target not found
	return fmt.Sprintf("You don't see '%s' here.", targetName)
}

// castSelfSpell handles spells that target the caster
func (c *Command) castSelfSpell(p PlayerInterface, spell *spells.Spell) string {
	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	// Get WIS modifier for healing
	wisMod := p.GetWisdomMod()

	// Apply effects
	var results []string
	for _, effect := range spell.Effects {
		if effect.Target != spells.TargetSelf {
			continue
		}

		switch effect.Type {
		case spells.EffectHeal:
			var healAmount int
			if effect.Dice != "" {
				// Use dice notation with WIS modifier
				healAmount = stats.ParseDiceWithBonus(effect.Dice, wisMod)
				if healAmount < 1 {
					healAmount = 1
				}
			} else {
				// Fallback to flat amount + WIS modifier
				healAmount = effect.Amount + wisMod
				if healAmount < 1 {
					healAmount = 1
				}
			}
			healed := p.Heal(healAmount)
			if healed > 0 {
				results = append(results, fmt.Sprintf("+%d HP", healed))
			}
		case spells.EffectHealPercent:
			healAmount := (p.GetMaxHealth() * effect.Amount) / 100
			healed := p.Heal(healAmount)
			if healed > 0 {
				results = append(results, fmt.Sprintf("+%d HP", healed))
			}
		}
	}

	// Build result message
	if len(results) == 0 {
		return fmt.Sprintf("You cast %s on yourself.", spell.Name)
	}

	effectStr := strings.Join(results, ", ")
	return fmt.Sprintf("You cast %s on yourself.\nYou feel a warm glow as your wounds begin to mend. [%s]", spell.Name, effectStr)
}

// castAllySpell handles spells that target other players
func (c *Command) castAllySpell(p PlayerInterface, spell *spells.Spell, target PlayerInterface) string {
	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	// Get server for broadcasts
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	room := p.GetCurrentRoom().(*world.Room)

	// Get WIS modifier for healing
	wisMod := p.GetWisdomMod()

	// Apply effects
	var results []string
	for _, effect := range spell.Effects {
		if effect.Target != spells.TargetAlly {
			continue
		}

		switch effect.Type {
		case spells.EffectHeal:
			var healAmount int
			if effect.Dice != "" {
				// Use dice notation with WIS modifier
				healAmount = stats.ParseDiceWithBonus(effect.Dice, wisMod)
				if healAmount < 1 {
					healAmount = 1
				}
			} else {
				// Fallback to flat amount + WIS modifier
				healAmount = effect.Amount + wisMod
				if healAmount < 1 {
					healAmount = 1
				}
			}
			healed := target.Heal(healAmount)
			if healed > 0 {
				results = append(results, fmt.Sprintf("+%d HP", healed))
			}
		case spells.EffectHealPercent:
			// Heal based on CASTER's max HP (scales with caster's level)
			healAmount := (p.GetMaxHealth() * effect.Amount) / 100
			healed := target.Heal(healAmount)
			if healed > 0 {
				results = append(results, fmt.Sprintf("+%d HP", healed))
			}
		}
	}

	// Broadcast to room
	server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s casts %s on %s!\n", p.GetName(), spell.Name, target.GetName()), p)

	// Send message to target
	if len(results) > 0 {
		effectStr := strings.Join(results, ", ")
		target.SendMessage(fmt.Sprintf("%s casts %s on you! [%s]\n", p.GetName(), spell.Name, effectStr))
		return fmt.Sprintf("You cast %s on %s. [%s]", spell.Name, target.GetName(), effectStr)
	}

	target.SendMessage(fmt.Sprintf("%s casts %s on you!\n", p.GetName(), spell.Name))
	return fmt.Sprintf("You cast %s on %s.", spell.Name, target.GetName())
}

// castEnemySpell handles spells that target NPCs/enemies
func (c *Command) castEnemySpell(p PlayerInterface, spell *spells.Spell, targetNPC *npc.NPC, room *world.Room) string {
	// Check if NPC is attackable
	if spell.HasDamageEffect() && !targetNPC.IsAttackable() {
		return fmt.Sprintf("You cannot attack %s!", targetNPC.GetName())
	}

	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	// Get server for broadcasts
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Apply effects
	var results []string
	totalDamage := 0

	// Get INT modifier for spell damage
	intMod := p.GetIntelligenceMod()

	for _, effect := range spell.Effects {
		if effect.Target != spells.TargetEnemy {
			continue
		}

		switch effect.Type {
		case spells.EffectDamage:
			var damage int
			if effect.Dice != "" {
				// Use dice notation with INT modifier
				damage = stats.ParseDiceWithBonus(effect.Dice, intMod)
				if damage < 1 {
					damage = 1
				}
			} else {
				// Fallback to flat amount + INT modifier
				damage = effect.Amount + intMod
				if damage < 1 {
					damage = 1
				}
			}
			// Magic damage bypasses armor
			actualDamage := targetNPC.TakeMagicDamage(damage)
			totalDamage += actualDamage
			results = append(results, fmt.Sprintf("%d damage", actualDamage))
		}
	}

	// Build result message
	var result strings.Builder
	result.WriteString(fmt.Sprintf("You cast %s at %s!\n", spell.Name, targetNPC.GetName()))

	if len(results) > 0 {
		effectStr := strings.Join(results, ", ")
		result.WriteString(fmt.Sprintf("A burst of magical energy strikes %s for %s!", targetNPC.GetName(), effectStr))
	}

	// Broadcast to room
	server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s casts %s at %s!\n", p.GetName(), spell.Name, targetNPC.GetName()), p)

	// If we dealt damage, initiate combat (combat ticker will handle NPC death if needed)
	if spell.HasDamageEffect() && totalDamage > 0 && !p.IsInCombat() {
		p.StartCombat(targetNPC.GetName())
		targetNPC.StartCombat(p.GetName())
		result.WriteString("\n\nCombat initiated! Type 'flee' to escape.")
	}

	return result.String()
}

// castRoomSpell handles spells that affect all enemies in the room
func (c *Command) castRoomSpell(p PlayerInterface, spell *spells.Spell) string {
	room := p.GetCurrentRoom().(*world.Room)

	// Get all attackable NPCs in the room
	allNPCs := room.GetNPCs()
	var targetNPCs []*npc.NPC
	for _, n := range allNPCs {
		if n.IsAttackable() && n.IsAlive() {
			targetNPCs = append(targetNPCs, n)
		}
	}

	if len(targetNPCs) == 0 {
		return "There are no hostile creatures here to affect."
	}

	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	// Get server for broadcasts
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get INT modifier for spell damage
	intMod := p.GetIntelligenceMod()

	// Apply effects to all targets
	var affectedNames []string
	for _, targetNPC := range targetNPCs {
		for _, effect := range spell.Effects {
			if effect.Target != spells.TargetRoomEnemy {
				continue
			}

			switch effect.Type {
			case spells.EffectStun:
				targetNPC.Stun(effect.Amount)
				affectedNames = append(affectedNames, targetNPC.GetName())
			case spells.EffectDamage:
				var damage int
				if effect.Dice != "" {
					// Use dice notation with INT modifier
					damage = stats.ParseDiceWithBonus(effect.Dice, intMod)
					if damage < 1 {
						damage = 1
					}
				} else {
					// Fallback to flat amount + INT modifier
					damage = effect.Amount + intMod
					if damage < 1 {
						damage = 1
					}
				}
				targetNPC.TakeMagicDamage(damage)
				affectedNames = append(affectedNames, targetNPC.GetName())
			}
		}
	}

	// Build result message
	var result strings.Builder
	result.WriteString(fmt.Sprintf("You cast %s!\n", spell.Name))

	if spell.HasStunEffect() {
		// Find stun duration from effects
		var stunDuration int
		for _, effect := range spell.Effects {
			if effect.Type == spells.EffectStun {
				stunDuration = effect.Amount
				break
			}
		}
		result.WriteString(fmt.Sprintf("A blinding flash of light erupts from your hands!\n"))
		result.WriteString(fmt.Sprintf("Stunned for %d seconds: %s", stunDuration, strings.Join(affectedNames, ", ")))
	}

	// Broadcast to room
	server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s casts %s! A blinding flash of light fills the room!\n", p.GetName(), spell.Name), p)

	return result.String()
}

// executeSpells shows the player's known spells
func (c *Command) executeSpells(p PlayerInterface) string {
	// Get the spell registry from server
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	registry := server.GetSpellRegistry()
	if registry == nil {
		return "Magic is not available."
	}

	learnedSpellIDs := p.GetLearnedSpells()
	if len(learnedSpellIDs) == 0 {
		return "You don't know any spells."
	}

	var sb strings.Builder
	sb.WriteString("=== Your Spells ===\n")

	for _, spellID := range learnedSpellIDs {
		spell, exists := registry.GetSpell(spellID)
		if !exists {
			continue
		}

		// Check cooldown status
		onCooldown, remaining := p.IsSpellOnCooldown(spellID)
		status := "[Ready]"
		if onCooldown {
			status = fmt.Sprintf("[Cooldown: %ds]", remaining)
		}

		sb.WriteString(fmt.Sprintf("  %-10s - %s (%d mana) %s\n",
			spell.Name, spell.Description, spell.ManaCost, status))
	}

	sb.WriteString(fmt.Sprintf("\nMana: %d/%d", p.GetMana(), p.GetMaxMana()))

	return sb.String()
}

// ==================== ADMIN COMMANDS ====================

// executeAdmin handles all admin subcommands
func (c *Command) executeAdmin(p PlayerInterface) string {
	// Non-admins see "Unknown command" to hide existence of admin commands
	if !p.IsAdmin() {
		return fmt.Sprintf("Unknown command: %s. Type 'help' for available commands.", c.Name)
	}

	if len(c.Args) == 0 {
		return c.executeAdminHelp()
	}

	subcommand := strings.ToLower(c.Args[0])
	switch subcommand {
	case "help":
		return c.executeAdminHelp()
	case "promote":
		return c.executeAdminPromote(p)
	case "demote":
		return c.executeAdminDemote(p)
	case "ban":
		return c.executeAdminBan(p)
	case "unban":
		return c.executeAdminUnban(p)
	case "kick":
		return c.executeAdminKick(p)
	case "announce":
		return c.executeAdminAnnounce(p)
	case "teleport", "tp":
		return c.executeAdminTeleport(p)
	case "goto":
		return c.executeAdminGoto(p)
	case "stats":
		return c.executeAdminStats(p)
	case "players":
		return c.executeAdminPlayers(p)
	default:
		return fmt.Sprintf("Unknown admin command: %s. Type 'admin help' for commands.", subcommand)
	}
}

// executeAdminHelp shows admin command help
func (c *Command) executeAdminHelp() string {
	return `
Admin Commands
==============

Player Management:
  admin promote <player>     - Grant admin privileges to a player
  admin demote <player>      - Remove admin privileges from a player
  admin ban <player> [reason] - Ban a player's account
  admin unban <username>     - Unban an account by username
  admin kick <player> [reason] - Disconnect a player

Communication:
  admin announce <message>   - Broadcast to all players

Teleportation:
  admin teleport <player> <room> - Move a player to a room
  admin tp <player> <room>   - Alias for teleport
  admin goto <room>          - Teleport yourself to a room

Information:
  admin stats               - Show server statistics
  admin players             - List all online players with details
  admin help                - Show this help message
`
}

// executeAdminPromote grants admin privileges to a player
func (c *Command) executeAdminPromote(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin promote <player_name>"
	}

	targetName := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Find the target player (online)
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		// Try to find by account username (offline player)
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found (must be online or use account username).", targetName)
		}

		if account.IsAdmin {
			return fmt.Sprintf("Account '%s' is already an admin.", account.Username)
		}

		if err := db.SetAdmin(account.ID, true); err != nil {
			return fmt.Sprintf("Failed to promote account: %v", err)
		}

		// Log admin action
		logger.Always("ADMIN_ACTION",
			"action", "promote",
			"admin", p.GetName(),
			"target_account", account.Username,
			"target_online", false)

		return fmt.Sprintf("Account '%s' has been promoted to admin.", account.Username)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	if target.IsAdmin() {
		return fmt.Sprintf("%s is already an admin.", target.GetName())
	}

	// Promote the account
	if err := db.SetAdmin(target.GetAccountID(), true); err != nil {
		return fmt.Sprintf("Failed to promote: %v", err)
	}

	// Notify the target
	target.SendMessage("\n*** You have been granted admin privileges! ***\n")

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "promote",
		"admin", p.GetName(),
		"target", target.GetName(),
		"target_account_id", target.GetAccountID())

	return fmt.Sprintf("%s has been promoted to admin.", target.GetName())
}

// executeAdminDemote removes admin privileges from a player
func (c *Command) executeAdminDemote(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin demote <player_name>"
	}

	targetName := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Check if this would leave no admins
	admins, err := db.GetAllAdmins()
	if err != nil {
		return fmt.Sprintf("Failed to check admin count: %v", err)
	}

	if len(admins) <= 1 {
		return "Cannot demote: this would leave the server with no admins."
	}

	// Find the target player (online)
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		// Try to find by account username (offline player)
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found (must be online or use account username).", targetName)
		}

		if !account.IsAdmin {
			return fmt.Sprintf("Account '%s' is not an admin.", account.Username)
		}

		if err := db.SetAdmin(account.ID, false); err != nil {
			return fmt.Sprintf("Failed to demote account: %v", err)
		}

		// Log admin action
		logger.Always("ADMIN_ACTION",
			"action", "demote",
			"admin", p.GetName(),
			"target_account", account.Username,
			"target_online", false)

		return fmt.Sprintf("Account '%s' has been demoted from admin.", account.Username)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	if !target.IsAdmin() {
		return fmt.Sprintf("%s is not an admin.", target.GetName())
	}

	// Demote the account
	if err := db.SetAdmin(target.GetAccountID(), false); err != nil {
		return fmt.Sprintf("Failed to demote: %v", err)
	}

	// Notify the target
	target.SendMessage("\n*** Your admin privileges have been revoked. ***\n")

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "demote",
		"admin", p.GetName(),
		"target", target.GetName(),
		"target_account_id", target.GetAccountID())

	return fmt.Sprintf("%s has been demoted from admin.", target.GetName())
}

// executeAdminBan bans a player's account
func (c *Command) executeAdminBan(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin ban <player_name> [reason]"
	}

	targetName := c.Args[1]
	reason := ""
	if len(c.Args) > 2 {
		reason = strings.Join(c.Args[2:], " ")
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// First try to find online player
	var accountID int64
	var accountUsername string

	targetIface := server.FindPlayer(targetName)
	if targetIface != nil {
		target, ok := targetIface.(PlayerInterface)
		if !ok {
			return "Internal error: invalid player type"
		}

		// Don't allow banning admins
		if target.IsAdmin() {
			return "Cannot ban an admin account."
		}

		accountID = target.GetAccountID()
		accountUsername = target.GetName()

		// Kick the player
		kickMsg := "\n*** YOU HAVE BEEN BANNED"
		if reason != "" {
			kickMsg += ": " + reason
		}
		kickMsg += " ***\n"
		target.SendMessage(kickMsg)
		target.Disconnect()
	} else {
		// Try to find by account username
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found.", targetName)
		}

		if account.IsAdmin {
			return "Cannot ban an admin account."
		}

		if account.Banned {
			return fmt.Sprintf("Account '%s' is already banned.", account.Username)
		}

		accountID = account.ID
		accountUsername = account.Username
	}

	// Ban the account
	if err := db.BanAccount(accountID); err != nil {
		return fmt.Sprintf("Failed to ban account: %v", err)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "ban",
		"admin", p.GetName(),
		"target_account", accountUsername,
		"reason", reason)

	if reason != "" {
		return fmt.Sprintf("Account '%s' has been banned. Reason: %s", accountUsername, reason)
	}
	return fmt.Sprintf("Account '%s' has been banned.", accountUsername)
}

// executeAdminUnban unbans an account
func (c *Command) executeAdminUnban(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin unban <username>"
	}

	username := c.Args[1]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Get account by username
	account, err := db.GetAccountByUsername(username)
	if err != nil {
		return fmt.Sprintf("Account '%s' not found.", username)
	}

	if !account.Banned {
		return fmt.Sprintf("Account '%s' is not banned.", username)
	}

	// Unban the account
	if err := db.UnbanAccount(account.ID); err != nil {
		return fmt.Sprintf("Failed to unban account: %v", err)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "unban",
		"admin", p.GetName(),
		"target_account", username)

	return fmt.Sprintf("Account '%s' has been unbanned.", username)
}

// executeAdminKick disconnects a player
func (c *Command) executeAdminKick(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin kick <player_name> [reason]"
	}

	targetName := c.Args[1]
	reason := ""
	if len(c.Args) > 2 {
		reason = strings.Join(c.Args[2:], " ")
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	if !server.KickPlayer(targetName, reason) {
		return fmt.Sprintf("Player '%s' not found or not online.", targetName)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "kick",
		"admin", p.GetName(),
		"target", targetName,
		"reason", reason)

	if reason != "" {
		return fmt.Sprintf("%s has been kicked. Reason: %s", targetName, reason)
	}
	return fmt.Sprintf("%s has been kicked.", targetName)
}

// executeAdminAnnounce broadcasts a server-wide message
func (c *Command) executeAdminAnnounce(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin announce <message>"
	}

	message := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Broadcast announcement
	announcement := fmt.Sprintf("\n[ANNOUNCEMENT from %s] %s\n", p.GetName(), message)
	server.BroadcastToAll(announcement)

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "announce",
		"admin", p.GetName(),
		"message", message)

	return "Announcement sent."
}

// executeAdminTeleport moves a player to a specific room
func (c *Command) executeAdminTeleport(p PlayerInterface) string {
	if len(c.Args) < 3 {
		return "Usage: admin teleport <player_name> <room_id>"
	}

	targetName := c.Args[1]
	roomID := c.Args[2]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get the world
	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Find the target player
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' not found or not online.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Find the target room
	room := w.GetRoom(roomID)
	if room == nil {
		return fmt.Sprintf("Room '%s' not found.", roomID)
	}

	// Broadcast exit message from current room
	currentRoom := target.GetCurrentRoom()
	if currentRoom != nil {
		if r, ok := currentRoom.(RoomInterface); ok {
			server.BroadcastToRoom(r.GetID(), fmt.Sprintf("%s vanishes in a flash of light!\n", target.GetName()), target)
		}
	}

	// Move the player
	target.MoveTo(room)

	// Broadcast enter message to new room
	server.BroadcastToRoom(roomID, fmt.Sprintf("%s appears in a flash of light!\n", target.GetName()), target)

	// Send room description to teleported player
	target.SendMessage(fmt.Sprintf("\n*** You have been teleported by %s ***\n\n%s", p.GetName(), room.GetDescriptionForPlayer(target.GetName())))

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "teleport",
		"admin", p.GetName(),
		"target", target.GetName(),
		"destination", roomID)

	return fmt.Sprintf("%s has been teleported to %s.", target.GetName(), roomID)
}

// executeAdminGoto teleports the admin to a specific room
func (c *Command) executeAdminGoto(p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin goto <room_id>"
	}

	roomID := c.Args[1]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get the world
	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Find the target room
	room := w.GetRoom(roomID)
	if room == nil {
		return fmt.Sprintf("Room '%s' not found.", roomID)
	}

	// Broadcast exit message from current room
	currentRoom := p.GetCurrentRoom()
	if currentRoom != nil {
		if r, ok := currentRoom.(RoomInterface); ok {
			server.BroadcastToRoom(r.GetID(), fmt.Sprintf("%s vanishes in a flash of light!\n", p.GetName()), p)
		}
	}

	// Move the player
	p.MoveTo(room)

	// Broadcast enter message to new room
	server.BroadcastToRoom(roomID, fmt.Sprintf("%s appears in a flash of light!\n", p.GetName()), p)

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "goto",
		"admin", p.GetName(),
		"destination", roomID)

	return fmt.Sprintf("Teleported to %s.\n\n%s", roomID, room.GetDescriptionForPlayer(p.GetName()))
}

// executeAdminStats shows server statistics
func (c *Command) executeAdminStats(p PlayerInterface) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Get stats
	uptime := server.GetUptime()
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	playersOnline := len(server.GetOnlinePlayers())
	roomCount := server.GetWorldRoomCount()

	totalAccounts, _ := db.GetTotalAccounts()
	totalCharacters, _ := db.GetTotalCharacters()

	return fmt.Sprintf(`
Server Statistics
=================
Uptime:           %d hours, %d minutes, %d seconds
Players Online:   %d
World Rooms:      %d
Total Accounts:   %d
Total Characters: %d
`,
		hours, minutes, seconds,
		playersOnline,
		roomCount,
		totalAccounts,
		totalCharacters)
}

// executeAdminPlayers lists all online players with details
func (c *Command) executeAdminPlayers(p PlayerInterface) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	players := server.GetOnlinePlayersDetailed()

	if len(players) == 0 {
		return "No players online."
	}

	result := "\nOnline Players\n==============\n"
	for _, pi := range players {
		adminTag := ""
		if pi.IsAdmin {
			adminTag = " [ADMIN]"
		}
		result += fmt.Sprintf("  %s (Lvl %d) - Room: %s - IP: %s%s\n",
			pi.Name, pi.Level, pi.RoomID, pi.IP, adminTag)
	}
	result += fmt.Sprintf("\nTotal: %d player(s)\n", len(players))

	return result
}

// executeUnlock handles the unlock command for locked doors
func (c *Command) executeUnlock(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Unlock what direction? Usage: unlock <direction>"); err != nil {
		return err.Error()
	}

	direction := strings.ToLower(c.Args[0])

	// Normalize direction
	switch direction {
	case "n":
		direction = "north"
	case "s":
		direction = "south"
	case "e":
		direction = "east"
	case "w":
		direction = "west"
	case "u":
		direction = "up"
	case "d":
		direction = "down"
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Check if exit exists
	if room.GetExit(direction) == nil && direction != "up" {
		return fmt.Sprintf("There is no exit %s.", direction)
	}

	// Check if exit is locked
	if !room.IsExitLocked(direction) {
		return fmt.Sprintf("The way %s is not locked.", direction)
	}

	// Get required key ID
	keyID := room.GetExitKeyRequired(direction)

	// Check if player has the key on their key ring
	if !p.HasKey(keyID) {
		return fmt.Sprintf("You don't have the key to unlock this door.")
	}

	// Check if this is a boss key (boss keys are reusable)
	isBossKey := strings.HasPrefix(keyID, "boss_key_")

	var keyName string
	if isBossKey {
		// Boss keys are NOT consumed - find the key to get its name
		key, _ := p.FindKey(keyID)
		if key != nil {
			keyName = key.Name
		} else {
			keyName = "Boss Key"
		}
	} else {
		// Non-boss keys (like treasure keys) are consumed on use
		matchingKey, _ := p.RemoveKeyByID(keyID)
		if matchingKey != nil {
			keyName = matchingKey.Name
		} else {
			keyName = "key"
		}
	}

	// Unlock the exit
	room.UnlockExit(direction)

	// Remove the locked_door feature if present
	room.RemoveFeature("locked_door")

	if isBossKey {
		return fmt.Sprintf("You use the %s to unlock the way %s. The door swings open!\n(Boss keys are permanent and remain on your key ring.)", keyName, direction)
	}
	return fmt.Sprintf("You use the %s to unlock the way %s. The door swings open!\n(The key crumbles to dust after use.)", keyName, direction)
}

// findShopNPC finds an NPC with shop inventory in the given room
// Returns the NPC if found, nil otherwise
func findShopNPC(room RoomInterface) *npc.NPC {
	npcs := room.GetNPCs()
	for _, n := range npcs {
		if n.HasShopInventory() {
			return n
		}
	}
	return nil
}

// executeShop shows the shop inventory
func (c *Command) executeShop(p PlayerInterface) string {
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findShopNPC(room)
	if shopNPC == nil {
		return "There is no shop here."
	}

	// Get the server to look up items
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get the NPC's shop inventory
	shopInventory := shopNPC.GetShopInventory()
	isMerchant := room.HasFeature("merchant")

	var result string
	npcName := shopNPC.GetName()
	// Capitalize first letter of NPC name for display
	if len(npcName) > 0 {
		npcName = strings.ToUpper(string(npcName[0])) + npcName[1:]
	}
	if isMerchant {
		result = fmt.Sprintf("\n=== %s ===\n", npcName)
		result += "\"Ye look like ye could use some supplies. I ain't cheap, but I'm all ye got up here.\"\n\n"
		result += fmt.Sprintf("Your gold: %d\n\n", p.GetGold())
		result += "Items for sale:\n"
	} else {
		result = fmt.Sprintf("\n=== %s's Shop ===\n", npcName)
		result += fmt.Sprintf("Your gold: %d\n\n", p.GetGold())
		result += "Items for sale:\n"
	}

	// Display each item from the shop inventory
	for _, shopItem := range shopInventory {
		itemDef := server.GetItemByID(shopItem.ItemName)
		if itemDef != nil {
			price := shopItem.Price
			if price == 0 {
				price = itemDef.Value // Use base value if no custom price
			}
			result += fmt.Sprintf("  %-20s %5d gold - %s\n", itemDef.Name, price, itemDef.Description)
		}
	}

	result += "\nCommands:\n"
	result += "  buy <item>   - Purchase an item\n"
	result += "  sell <item>  - Sell an item from your inventory\n"

	return result
}

// executeBuy handles purchasing items from a shop or merchant
func (c *Command) executeBuy(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Buy what? Usage: buy <item name>"); err != nil {
		return err.Error()
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findShopNPC(room)
	if shopNPC == nil {
		return "There is no shop here."
	}

	// Get the server to look up items
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	itemName := strings.ToLower(c.GetItemName())
	sellerName := shopNPC.GetName()

	// Get the NPC's shop inventory
	shopInventory := shopNPC.GetShopInventory()

	// Find the item in shop inventory
	var foundShopItem *npc.ShopItem
	var foundItem *items.Item
	var price int

	for i := range shopInventory {
		itemDef := server.GetItemByID(shopInventory[i].ItemName)
		if itemDef == nil {
			continue
		}
		// Match by item name (case-insensitive, partial match)
		if strings.ToLower(itemDef.Name) == itemName ||
			strings.Contains(strings.ToLower(itemDef.Name), itemName) {
			foundShopItem = &shopInventory[i]
			foundItem = itemDef
			price = foundShopItem.Price
			if price == 0 {
				price = foundItem.Value
			}
			break
		}
	}

	if foundItem == nil {
		return fmt.Sprintf("%s doesn't sell '%s'. Type 'shop' to see available items.", sellerName, c.GetItemName())
	}

	// Check if player has enough gold
	if p.GetGold() < price {
		return fmt.Sprintf("You don't have enough gold. The %s costs %d gold, but you only have %d.", foundItem.Name, price, p.GetGold())
	}

	// Spend the gold
	p.SpendGold(price)

	// Keys go to key ring, other items to inventory
	if foundItem.Type == items.Key {
		p.AddKey(foundItem)
		return fmt.Sprintf("You purchase a %s for %d gold and add it to your key ring.\nGold remaining: %d", foundItem.Name, price, p.GetGold())
	}

	p.AddItem(foundItem)
	return fmt.Sprintf("You purchase a %s for %d gold.\nGold remaining: %d", foundItem.Name, price, p.GetGold())
}

// executeSell handles selling items to a shop or merchant
func (c *Command) executeSell(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Sell what? Usage: sell <item name>"); err != nil {
		return err.Error()
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findShopNPC(room)
	if shopNPC == nil {
		return "There is no shop here."
	}

	// Check if this is a tower merchant (worse buy prices)
	isMerchant := room.HasFeature("merchant")

	itemName := c.GetItemName()

	// Find the item in player's inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Calculate sell price based on shop type
	// Shop: 50% of item value
	// Merchant: 25% of item value (rounded up)
	var sellPrice int
	if isMerchant {
		// 25% rounded up: (value + 3) / 4 is equivalent to ceil(value/4)
		sellPrice = (item.Value + 3) / 4
	} else {
		sellPrice = item.Value / 2
	}

	// Minimum 1 gold for items with value
	if sellPrice < 1 && item.Value > 0 {
		sellPrice = 1
	}

	// Items with no value can't be sold
	if item.Value == 0 {
		if isMerchant {
			return fmt.Sprintf("The old merchant squints at your %s and shakes his head. \"Ain't worth nothin' to me.\"", item.Name)
		}
		return fmt.Sprintf("%s isn't interested in your %s.", shopNPC.GetName(), item.Name)
	}

	// Remove the item from inventory
	removedItem, removed := p.RemoveItem(item.Name)
	if !removed {
		return "Something went wrong trying to sell that item."
	}

	// Add gold to player
	p.AddGold(sellPrice)

	if isMerchant {
		return fmt.Sprintf("The old merchant grumbles as he hands you %d gold for your %s.\n\"Don't expect charity from me, adventurer.\"\nGold: %d", sellPrice, removedItem.Name, p.GetGold())
	}
	return fmt.Sprintf("You sell your %s for %d gold.\nGold: %d", removedItem.Name, sellPrice, p.GetGold())
}

// executeGold shows the player's gold amount
func (c *Command) executeGold(p PlayerInterface) string {
	return fmt.Sprintf("You have %d gold.", p.GetGold())
}

// BardSaveCost is the gold cost for the bard to write a song (save the game)
const BardSaveCost = 5

// executeTalk allows the player to talk to NPCs
func (c *Command) executeTalk(p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: talk <npc name>"); err != nil {
		return err.Error()
	}

	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find the NPC
	npcName := c.GetItemName()
	foundNPC := worldRoom.FindNPC(npcName)
	if foundNPC == nil {
		return fmt.Sprintf("You don't see '%s' here.", npcName)
	}

	// Check if NPC is alive
	if !foundNPC.IsAlive() {
		return fmt.Sprintf("The %s is dead and cannot respond.", foundNPC.GetName())
	}

	// Special handling for the bard - save game functionality
	if strings.Contains(strings.ToLower(foundNPC.GetName()), "bard") {
		return c.handleBardInteraction(p, foundNPC)
	}

	// Special handling for Aldric the old guide - tutorial
	// Check for "old guide" specifically to avoid matching "King Aldric the Wise"
	if strings.Contains(strings.ToLower(foundNPC.GetName()), "old guide") {
		return c.handleGuideInteraction(p, foundNPC)
	}

	// Get a dialogue line
	dialogue := foundNPC.GetDialogue()
	if dialogue == "" {
		return fmt.Sprintf("The %s doesn't seem interested in conversation.", foundNPC.GetName())
	}

	return fmt.Sprintf("The %s says, \"%s\"", foundNPC.GetName(), dialogue)
}

// handleBardInteraction handles the special save-game interaction with the bard
func (c *Command) handleBardInteraction(p PlayerInterface, bard *npc.NPC) string {
	// Check if player has enough gold
	if p.GetGold() < BardSaveCost {
		return fmt.Sprintf(`The %s strums his lute and looks at you expectantly.

"%s says, "Ah, you wish me to immortalize your deeds in song? A mere %d gold for a ballad that will echo through the ages!"

He notices your empty coin purse and sighs. "Alas, it seems you cannot afford my services. Return when you have %d gold, and I shall compose a masterpiece!"

(You need %d gold to save your progress. You have %d gold.)`,
			bard.GetName(), bard.GetName(), BardSaveCost, BardSaveCost, BardSaveCost, p.GetGold())
	}

	// Deduct the gold
	p.SpendGold(BardSaveCost)

	// Save the player
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		// Refund on error
		p.AddGold(BardSaveCost)
		return "Internal error: unable to save."
	}

	if err := server.SavePlayer(p); err != nil {
		// Refund on error
		p.AddGold(BardSaveCost)
		return fmt.Sprintf("The bard tries to compose, but something goes wrong: %v", err)
	}

	// Generate a fun song snippet based on player stats
	songLines := []string{
		fmt.Sprintf("~ Of %s the brave, level %d and bold ~", p.GetName(), p.GetLevel()),
		fmt.Sprintf("~ With %d gold in pocket, adventures untold ~", p.GetGold()),
		fmt.Sprintf("~ Through tower floors they climb so high ~"),
		fmt.Sprintf("~ A hero's tale that will never die! ~"),
	}

	return fmt.Sprintf(`The %s's eyes light up as you hand over %d gold.

"Ah, a patron of the arts! Let me compose a verse worthy of your adventures..."

He clears his throat and begins to sing:

%s
%s
%s
%s

The bard bows with a flourish. "Your legend is now preserved for all time!"

(Progress saved! Gold remaining: %d)`,
		bard.GetName(), BardSaveCost,
		songLines[0], songLines[1], songLines[2], songLines[3],
		p.GetGold())
}

// handleGuideInteraction provides the new player tutorial from Aldric
func (c *Command) handleGuideInteraction(p PlayerInterface, guide *npc.NPC) string {
	return fmt.Sprintf(`%s smiles warmly and gestures for you to sit beside him on a weathered bench.

"Ah, %s! Welcome to our fair city. I can see you're new here, so let me tell
you everything you need to know to survive... and perhaps even thrive!"

He leans in conspiratorially and begins:

===============================================================================
                            THE ENDLESS TOWER
===============================================================================

"You see that massive tower to the south? That's why we're all here. It stretches
endlessly upward, filled with monsters, treasures, and mysteries. The higher you
climb, the stronger the creatures... but the greater the rewards!"

  - Go SOUTH from here, then SOUTH again, then SOUTH once more to reach the
    TOWER ENTRANCE. Type 'up' to begin your ascent.
  - Each floor is different - corridors, chambers, treasure rooms, and boss lairs.
  - Every 10th floor (10, 20, 30...) has a powerful BOSS you must defeat!

===============================================================================
                              STAYING ALIVE
===============================================================================

"The tower is dangerous! Here's how to not die... too often:"

  - TEMPLE OF LIGHT (east from here): Visit High Priestess Sera and type 'pray'
    at the altar to fully restore your health and mana. Do this before every
    expedition!

  - COMBAT: Type 'attack <monster>' to fight. Combat happens automatically
    every few seconds. Type 'flee' if you're losing!

  - CONSIDER: Type 'consider <monster>' before fighting to see if you can
    handle it. Type 'consider self' to see your own stats.

  - REST: Type 'sleep' to regenerate health faster. Type 'wake' to get up.

===============================================================================
                            GOLD & EQUIPMENT
===============================================================================

"Gold makes the world go round, friend!"

  - GENERAL STORE (south, then east): Type 'shop' to see items for sale.
    Type 'buy <item>' to purchase. Type 'sell <item>' to sell your loot!

  - EQUIPMENT: Type 'wield <weapon>' or 'wear <armor>' to equip items.
    Type 'inventory' to see what you're carrying.

  - LOOT: Monsters drop items when defeated. Type 'get <item>' to pick them up!

  - You start with %d gold. Spend it wisely!

===============================================================================
                       *** SAVING YOUR PROGRESS ***
===============================================================================

"This is VERY important, so listen carefully!"

  - Your progress is ONLY saved when you visit THE BARD in the tavern!
  - Go SOUTH, SOUTH, then EAST to find 'The Weary Wanderer Tavern'.
  - Type 'talk bard' - he'll write a song about your adventures for 5 gold.
  - If you disconnect WITHOUT visiting the bard, you LOSE all progress since
    your last save!

===============================================================================
                              PORTAL TRAVEL
===============================================================================

"Once you've explored a bit, travel becomes easier:"

  - Each floor's STAIRWAY has a magical PORTAL that you can use.
  - Type 'portal' to see floors you've discovered.
  - Type 'portal <floor>' to instantly travel there!
  - There's a portal right here in Town Square too (floor 0).

===============================================================================
                              QUICK REFERENCE
===============================================================================

  look          - See your surroundings       help          - Full command list
  north/s/e/w   - Move around                 inventory     - Your items
  attack <npc>  - Start combat                flee          - Escape combat
  pray          - Heal at altar               talk bard     - SAVE GAME!
  shop/buy/sell - Trading                     portal        - Fast travel

===============================================================================

%s chuckles and pats you on the shoulder.

"That's the basics! Now go forth, brave adventurer. Climb that tower, slay
those monsters, and most importantly - DON'T FORGET TO VISIT THE BARD!"

"Oh, and one more thing... if you ever forget all this, just 'talk' to me again!"

He winks and settles back onto his bench.`, guide.GetName(), p.GetName(), p.GetGold(), guide.GetName())
}
