package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
)

// executeHelp shows help for commands
func executeHelp(c *Command, p PlayerInterface) string {
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

	return getHelpText(topic, isAdmin)
}

// executeScore shows a comprehensive character summary (replaces stats command)
func executeScore(c *Command, p PlayerInterface) string {
	var result strings.Builder

	// Header with name
	result.WriteString(fmt.Sprintf("=== %s ===\n", p.GetName()))

	// Level and XP section
	level := p.GetLevel()
	xp := p.GetExperience()
	result.WriteString(fmt.Sprintf("Level: %d", level))
	if level >= leveling.MaxPlayerLevel {
		result.WriteString(" (MAX)\n")
	} else {
		xpNeeded := leveling.XPForLevel(level + 1)
		result.WriteString(fmt.Sprintf("  |  XP: %d / %d\n", xp, xpNeeded))
	}

	// Health and Mana
	result.WriteString(fmt.Sprintf("Health: %d / %d\n", p.GetHealth(), p.GetMaxHealth()))
	result.WriteString(fmt.Sprintf("Mana: %d / %d\n", p.GetMana(), p.GetMaxMana()))

	// Gold
	result.WriteString(fmt.Sprintf("Gold: %d\n", p.GetGold()))

	// Ability Scores section
	result.WriteString("\n--- Ability Scores ---\n")
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

	// Current state
	result.WriteString(fmt.Sprintf("\nState: %s\n", p.GetState()))

	return result.String()
}

// executeLevel shows detailed level progression information
func executeLevel(c *Command, p PlayerInterface) string {
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

// executePassword changes the player's account password
func executePassword(c *Command, p PlayerInterface) string {
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

// getHelpText returns help text for a given topic
func getHelpText(topic string, isAdmin bool) string {

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

	case "shout", "yell":
		return `SHOUT <message>
Shout something to everyone on the same floor.

Usage:
  shout Help! I need assistance!

Everyone on the same tower floor will see your message.

Alias: yell`

	case "emote", "me":
		return `EMOTE <action>
Perform a custom action visible to everyone in the room.

Usage:
  emote laughs heartily     -> "YourName laughs heartily"
  emote scratches head      -> "YourName scratches head"

Everyone in the same room will see your action.

Alias: me`

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
		return `USE <item or feature>
Use a consumable item from your inventory or interact with a room feature.

Usage:
  use healing potion - Restore 30 HP
  use mana potion    - Restore 30 MP
  use workbench      - Use a workbench for crafting
  use forge          - Use a forge for smithing

Consumable items (potions, food) are used for their effects.
Room features like workbenches and forges are used for crafting.

Note: If an item and feature share the same name, the item takes priority.`

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

	case "score", "sc", "stats", "abilities", "attributes":
		return `SCORE
Display a comprehensive summary of your character.

Shows your name, level, experience, health, mana, gold,
all ability scores with modifiers, and current state.

Aliases: sc, stats, abilities, attributes`

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

	case "give":
		return `GIVE <item> <player> or GIVE <amount> gold <player>
Give an item or gold to another player in the same room.

Usage:
  give sword Bob           - Give your sword to Bob
  give rusty sword to Bob  - Give your rusty sword to Bob
  give 50 gold Bob         - Give 50 gold to Bob
  give 100 gold to Bob     - Give 100 gold to Bob

Both players must be in the same room.
Items require the recipient to have enough carrying capacity.`

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
  score (sc)        - Show your full character sheet
  level (lvl)       - Show level progression and XP

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
  use <item>        - Use any consumable item or room feature

Communication:
  say <message>     - Say something to everyone in the room
  shout <message>   - Shout to everyone on the same floor
  emote <action>    - Perform a custom action (emote laughs)
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
  give <item/gold> <player> - Give item or gold to another player

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
