package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
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

	// Header with name, race, and class
	result.WriteString(fmt.Sprintf("=== %s ===\n", p.GetName()))
	result.WriteString(fmt.Sprintf("Race: %s\n", p.GetRaceName()))
	result.WriteString(fmt.Sprintf("Class: %s\n", p.GetClassLevelsSummary()))
	result.WriteString(fmt.Sprintf("Active: %s (gaining XP)\n", p.GetActiveClassName()))

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

// executeRace shows race information
func executeRace(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		// Show player's own race info
		playerRace, err := race.ParseRace(strings.ToLower(p.GetRaceName()))
		if err != nil {
			return fmt.Sprintf("Your race: %s\n\nUse 'race <name>' to view information about a specific race.\nValid races: Human, Dwarf, Elf, Halfling, Gnome, Half-Elf, Half-Orc", p.GetRaceName())
		}
		return formatRaceInfo(playerRace)
	}

	// Show info about a specific race
	raceName := strings.ToLower(strings.Join(c.Args, "-"))
	r, err := race.ParseRace(raceName)
	if err != nil {
		return fmt.Sprintf("Unknown race: %s\nValid races: Human, Dwarf, Elf, Halfling, Gnome, Half-Elf, Half-Orc", c.Args[0])
	}
	return formatRaceInfo(r)
}

// formatRaceInfo formats detailed race information
func formatRaceInfo(r race.Race) string {
	def := race.GetDefinition(r)
	if def == nil {
		return fmt.Sprintf("Race information not found for %s", r.String())
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== %s ===\n", r.String()))
	sb.WriteString(fmt.Sprintf("Size: %s\n\n", def.Size))
	sb.WriteString(fmt.Sprintf("%s\n\n", def.Description))

	sb.WriteString("Stat Bonuses:\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", def.GetStatBonusesString()))

	sb.WriteString("Racial Abilities:\n")
	for _, ability := range def.Abilities {
		sb.WriteString(fmt.Sprintf("  - %s\n", ability))
	}

	return sb.String()
}

// executeRaces lists all available races
func executeRaces(c *Command, p PlayerInterface) string {
	var sb strings.Builder

	sb.WriteString("=== Available Races ===\n\n")

	for _, r := range race.AllRaces() {
		def := race.GetDefinition(r)
		if def == nil {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s (%s)\n", r.String(), def.Size))
		sb.WriteString(fmt.Sprintf("  Bonuses: %s\n", def.GetStatBonusesString()))
		sb.WriteString(fmt.Sprintf("  %s\n\n", def.Description))
	}

	sb.WriteString("Use 'race <name>' for detailed information about a specific race.")

	return sb.String()
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
Display your available spells based on your class and level.

Shows:
  - Spell name and description
  - Mana cost
  - Class restriction (if any)
  - Cooldown status (ready or seconds remaining)
  - Your current mana

Different classes have access to different spells:
  - Mage: Damage spells (fireball, ice storm, etc.)
  - Cleric: Healing and support spells
  - Rogue: Tricks (vanish, poison, assassinate)
  - Ranger: Nature spells and buffs
  - Paladin: Holy spells and smites
  - Warrior: No spells (pure martial class)`

	case "craft", "crafting", "make":
		return `CRAFT [recipe | info <recipe>]
Create items at crafting stations using materials and skill.

Usage:
  craft              - Show recipes available at current station
  craft <recipe>     - Attempt to craft a recipe
  craft info <recipe> - Show detailed recipe information

Crafting Stations (found in the city):
  Forge (Armory)         - Blacksmithing: weapons, metal armor
  Workbench (Artisan's)  - Leatherworking: leather gear, bags
  Alchemy Lab (Alchemist's) - Alchemy: potions, salves
  Enchanting Table (Mage Tower) - Enchanting: magical items

Crafting requires:
  - The correct crafting station
  - Knowing the recipe (learn from trainers)
  - Required materials in your inventory
  - Meeting skill and level requirements

Success is based on: d20 + (Skill/5) + (INT mod/2) vs Difficulty
On failure, materials are returned to you.

See also: help learn, help skills`

	case "learn":
		return `LEARN [recipe]
Learn crafting recipes from crafting trainer NPCs.

Usage:
  learn              - Show recipes available from the trainer
  learn <recipe>     - Learn a specific recipe

Crafting Trainers (found in the city):
  Forge Master Tormund (Armory)       - Blacksmithing recipes
  Tanner Helga (Artisan's Market)     - Leatherworking recipes
  Alchemist Zara (Alchemist's Shop)   - Alchemy recipes
  Enchantress Lyrel (Mage Tower)      - Enchanting recipes

To learn a recipe you need:
  - To be in the same room as the trainer
  - Meet the recipe's player level requirement
  - Meet the recipe's skill level requirement

Skill requirements: Higher-tier recipes need more skill.
Build skill by crafting easier recipes first!

See also: help craft, help skills`

	case "skills":
		return `SKILLS
Display your crafting skill levels and progress.

Shows all four crafting skills:
  Blacksmithing (0-100)  - Forge weapons and metal armor
  Leatherworking (0-100) - Craft leather gear and bags
  Alchemy (0-100)        - Brew potions and salves
  Enchanting (0-100)     - Create magical items

Skills increase by successfully crafting items.
Higher difficulty recipes give more skill points.
Higher skill = better success chance + access to advanced recipes.

See also: help craft, help learn`

	case "quest", "quests", "journal":
		return `QUEST [list | available [name] | <quest name>]
View your quest journal and track your progress.

Usage:
  quest                    - Show quest journal summary
  quest list               - Show all active quests with progress
  quest available          - Show quests offered by nearby NPCs
  quest available <name>   - Preview a quest before accepting
  quest <name>             - Show detailed info about an active quest

Your quest journal shows:
  - Total active and completed quests
  - Progress on each objective
  - Rewards you'll receive on completion

See also: help accept, help complete, help title`

	case "accept":
		return `ACCEPT <quest name>
Accept quests from quest-giving NPCs.

Usage:
  accept <quest>      - Accept a specific quest

Requirements:
  - You must be in the same room as the quest giver
  - Some quests have level requirements
  - Some quests require completing prerequisite quests

Use 'quests available' to see quests offered by nearby NPCs.

See also: help quest, help complete`

	case "complete", "turnin":
		return `COMPLETE [quest name]
Turn in completed quests to receive rewards.

Usage:
  complete            - Turn in a completed quest
  complete <name>     - Turn in a specific quest

Requirements:
  - You must have completed all quest objectives
  - You must be in the same room as the turn-in NPC

Rewards may include:
  - Gold and Experience
  - Items
  - Crafting recipes
  - Titles

Aliases: turnin

See also: help quest, help accept`

	case "title":
		return `TITLE [title name | none]
View and set your displayed title.

Usage:
  title               - Show your earned titles
  title <name>        - Set your active title
  title none          - Clear your active title

Titles are earned by completing quests. Your active title
is displayed with your name when other players look at you.

See also: help quest, help complete`

	case "class", "classes":
		return `CLASS [subcommand]
View and manage your character classes.

Usage:
  class              - Show your current class information
  class list         - View all available classes and requirements
  class info <class> - View detailed information about a class
  class switch <class> - Change which class gains XP

Multiclassing:
  Once you reach level 10 in your primary class, you can learn
  additional classes from class trainers found in the city.

  Each class has stat requirements for multiclassing:
  - Warrior: STR 13+
  - Mage: INT 13+
  - Cleric: WIS 13+
  - Rogue: DEX 13+
  - Ranger: DEX 13+, WIS 13+
  - Paladin: STR 13+, CHA 13+

  Primary class can reach level 50, secondary classes cap at 25.

See also: help train, help multiclass`

	case "multiclass", "multiclassing":
		return `MULTICLASSING
Learn additional classes to expand your abilities.

Requirements:
  - Reach level 10 in your primary class
  - Meet the stat requirements for the new class
  - Have 500 gold for training

How to Multiclass:
  1. Check your stats with 'score' and class requirements with 'class list'
  2. Visit a class trainer in the city (see 'help train' for locations)
  3. Type 'train' to learn the new class

Benefits:
  - Access to spells and abilities from multiple classes
  - Combine class strengths (e.g., Warrior/Cleric for a tanky healer)
  - Equipment proficiencies are cumulative

Limitations:
  - Primary class can reach level 50
  - Secondary classes cap at level 25
  - XP only goes to your active class (use 'class switch' to change)

See also: help class, help train`

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

	case "buy":
		return `BUY <item>
Purchase an item from an NPC shop.

Usage:
  buy treasure key  - Buy a treasure key
  buy key           - Partial names work

You must be in a room with a shop NPC and have enough gold.
Keys are automatically added to your key ring.

Note: To buy from other players, see 'help purchase'.`

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

	case "stall":
		return `STALL [command]
Set up a stall to sell items to other players.

Commands:
  stall open           - Open your stall for business
  stall close          - Close your stall (items return to inventory)
  stall add <item> <price> - Add an item to your stall
  stall remove <item>  - Remove an item from your stall
  stall list           - View items in your stall
  stall help           - Show this help

Notes:
  - You can only open a stall in the city (floor 0)
  - Your stall closes if you leave the room or disconnect
  - Items in your stall are not in your inventory

See also: help browse, help purchase`

	case "browse":
		return `BROWSE <player>
View items for sale in another player's stall.

Usage:
  browse Bob           - See what Bob has for sale

The player must be in the same room and have an open stall.
Use 'purchase <item> from <player>' to buy items.

See also: help stall, help purchase`

	case "purchase":
		return `PURCHASE <item> from <player>
Buy an item from another player's stall.

Usage:
  purchase sword from Bob     - Buy a sword from Bob's stall
  purchase rusty sword from Bob - Partial names work

The player must be in the same room with an open stall.
You need enough gold and carrying capacity.

See also: help stall, help browse`

	case "talk", "speak", "chat":
		return `TALK <npc name>
Talk to an NPC to hear what they have to say.

Usage:
  talk merchant     - Talk to the merchant
  talk priestess    - Talk to a priestess
  speak guard       - Talk to a guard
  talk bard         - Hear a song about your adventures
  talk aldric       - Get help and tutorials from the guide

NPCs may give you helpful hints, lore, or just colorful dialogue.
Not all NPCs have something to say - monsters typically don't talk!

Your progress is saved automatically when you disconnect or quit.

Aliases: speak, chat`

	case "train":
		return `TRAIN
Learn a new class from a class trainer NPC.

Requirements:
  - You must be level 10+ in your primary class to multiclass
  - You must meet the stat requirements for the new class
  - Training costs 500 gold

Usage:
  train             - Learn the class from the trainer in your room

Class Trainers:
  Warrior  - Battlemaster Korg (Training Hall)
  Mage     - Archmage Thessaly (Royal Library)
  Cleric   - Father Aldous (Temple)
  Rogue    - Shadow (Tavern)
  Ranger   - Warden Ashara (North Gate)
  Paladin  - Sir Gareth the Radiant (Castle Hall)

After learning a new class:
  - Your new class starts at level 1
  - XP earned goes to your new (active) class
  - Use 'class switch <class>' to change which class gains XP
  - Use 'class' to view all your classes

See also: class, help class`

	case "save":
		return `SAVE
Your progress is saved automatically!

  - When you disconnect or quit the game
  - When the server shuts down for maintenance
  - After important events like learning a new class

You don't need to do anything special - just play and enjoy!`

	case "tutorial", "guide", "aldric", "newplayer", "new":
		return `TUTORIAL / NEW PLAYER GUIDE

If you're new to the game, find Aldric the old guide in the Town Square!

  Type: talk aldric

He will explain everything you need to know:
  - How to explore the tower
  - Combat basics
  - Where to buy and sell items
  - Portal travel system
  - And much more!

You can talk to Aldric anytime you need a refresher.

Quick Start:
  1. talk aldric     - Get the full tutorial
  2. pray            - Heal up at the Temple (go east first)
  3. Go south x3     - Head to the Tower Entrance
  4. up              - Enter the tower and begin your adventure!

Your progress is saved automatically when you disconnect or quit.`

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
  class             - View and manage your classes

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
  train             - Learn a new class from a class trainer (multiclass)

Shop (at General Store):
  shop              - View items for sale
  buy <item>        - Purchase an item
  sell <item>       - Sell an item (50% of item value)
  gold              - Check your gold balance
  give <item/gold> <player> - Give item or gold to another player

Player Stalls (sell items to other players):
  stall             - Manage your player stall (open, close, add, remove, list)
  browse <player>   - View another player's stall
  purchase <item> from <player> - Buy from a player's stall

Crafting (at crafting stations):
  craft             - Show recipes available at current station
  craft <recipe>    - Attempt to craft a recipe
  craft info <recipe> - Show recipe details
  learn [recipe]    - Learn recipes from a crafting trainer
  skills            - Show your crafting skill levels

Quests:
  quest             - View your quest journal
  quest list        - Show all active quests with progress
  quest available   - See available quests from nearby NPCs
  quest available <name> - Preview a quest before accepting
  accept <quest>    - Accept a specific quest
  complete          - Turn in a completed quest
  title             - View and set your title

Saving Progress:
  Your progress is saved automatically when you disconnect or quit.

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
