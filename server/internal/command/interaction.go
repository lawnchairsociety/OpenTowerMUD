package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// BardSaveCost is the cost in gold to save progress at the bard
const BardSaveCost = 5

// executeTalk initiates conversation with an NPC
func executeTalk(c *Command, p PlayerInterface) string {
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
		return handleBardInteraction(c, p, foundNPC)
	}

	// Special handling for Aldric the old guide - tutorial
	// Check for "old guide" specifically to avoid matching "King Aldric the Wise"
	if strings.Contains(strings.ToLower(foundNPC.GetName()), "old guide") {
		return handleGuideInteraction(c, p, foundNPC)
	}

	// Get a dialogue line
	dialogue := foundNPC.GetDialogue()
	if dialogue == "" {
		return fmt.Sprintf("The %s doesn't seem interested in conversation.", foundNPC.GetName())
	}

	return fmt.Sprintf("The %s says, \"%s\"", foundNPC.GetName(), dialogue)
}

// handleBardInteraction handles the special save-game interaction with the bard
func handleBardInteraction(c *Command, p PlayerInterface, bard *npc.NPC) string {
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
// Supports topic-based conversation: talk aldric, talk aldric tower, etc.
func handleGuideInteraction(c *Command, p PlayerInterface, guide *npc.NPC) string {
	// Check if a topic was specified (args after "aldric" or "guide")
	topic := ""
	args := c.Args
	for i, arg := range args {
		lower := strings.ToLower(arg)
		if lower == "aldric" || lower == "guide" || lower == "old" {
			// Topic is everything after the NPC name
			if i+1 < len(args) {
				topic = strings.ToLower(strings.Join(args[i+1:], " "))
			}
			break
		}
	}
	// If no NPC name found in args, check if there's a second word
	if topic == "" && len(args) >= 2 {
		topic = strings.ToLower(args[len(args)-1])
	}

	switch topic {
	case "tower", "dungeon", "floors":
		return getGuideTowerTopic(guide)
	case "combat", "fighting", "fight", "attack":
		return getGuideCombatTopic(guide)
	case "save", "saving", "bard":
		return getGuideSaveTopic(guide)
	case "shop", "gold", "equipment", "gear", "items":
		return getGuideShopTopic(guide, p)
	case "portal", "portals", "travel":
		return getGuidePortalTopic(guide)
	case "commands", "help":
		return getGuideCommandsTopic(guide)
	default:
		return getGuideGreeting(guide, p)
	}
}

// getGuideGreeting returns Aldric's initial greeting with topic list
func getGuideGreeting(guide *npc.NPC, p PlayerInterface) string {
	return fmt.Sprintf(`%s smiles warmly as you approach.

"Ah, %s! Welcome to our fair city. I'm here to help newcomers survive
the Endless Tower. What would you like to know about?"

  talk %s tower    - The Endless Tower and what awaits you
  talk %s combat   - How to fight and stay alive
  talk %s save     - How to SAVE your progress (IMPORTANT!)
  talk %s shop     - Buying, selling, and equipment
  talk %s portal   - Fast travel between floors
  talk %s commands - Quick reference of useful commands

He leans on his walking stick. "Just ask about any topic, friend!"`,
		guide.GetName(), p.GetName(),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]))
}

// getGuideTowerTopic explains the tower
func getGuideTowerTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s points toward the massive tower looming to the south.

"That's the Endless Tower - it's why we're all here. It stretches infinitely
upward, filled with monsters, treasures, and mysteries."

  - Go SOUTH three times from Town Square to reach the TOWER ENTRANCE
  - Type 'up' to begin climbing
  - Each floor has corridors, chambers, and treasure rooms
  - Every 10th floor (10, 20, 30...) has a powerful BOSS!
  - The higher you climb, the stronger the monsters... and better the loot!

"Start on floor 1, get some experience, then work your way up!"`, guide.GetName())
}

// getGuideCombatTopic explains combat and survival
func getGuideCombatTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s's expression grows serious.

"The tower is dangerous. Here's how to not die... too often:"

  BEFORE FIGHTING:
  - Type 'consider <monster>' to assess if you can handle it
  - Visit the TEMPLE (east from here) and type 'pray' to fully heal

  DURING COMBAT:
  - Type 'attack <monster>' to start fighting
  - Combat continues automatically every few seconds
  - Type 'flee' to escape if you're losing!

  RECOVERY:
  - Type 'sleep' to regenerate health faster (5 HP/tick)
  - Type 'wake' to stand back up
  - Or return to the temple and 'pray' for instant full heal

"Always check your health before going deeper!"`, guide.GetName())
}

// getGuideSaveTopic explains the critical save mechanic
func getGuideSaveTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s grabs your arm urgently.

"Listen carefully - this is the MOST important thing I'll tell you!"

  *** YOUR PROGRESS IS ONLY SAVED AT THE BARD! ***

  - Go SOUTH, SOUTH, then EAST to find 'The Weary Wanderer Tavern'
  - Type 'talk bard' - he'll write a song about you for 5 gold
  - This SAVES your character!

  WARNING: If you disconnect WITHOUT visiting the bard, you LOSE
  everything since your last save - experience, items, gold, ALL OF IT!

"Always visit the bard before you log off. ALWAYS!"`, guide.GetName())
}

// getGuideShopTopic explains commerce and equipment
func getGuideShopTopic(guide *npc.NPC, p PlayerInterface) string {
	return fmt.Sprintf(`%s jingles a few coins in his pocket.

"Gold makes the world go round, friend! You've got %d gold to start."

  SHOPPING (General Store - south then east from here):
  - Type 'shop' to see items for sale
  - Type 'buy <item>' to purchase
  - Type 'sell <item>' to sell loot (50%% of value)

  EQUIPMENT:
  - Type 'wield <weapon>' to equip a weapon
  - Type 'wear <armor>' to put on armor
  - Type 'inventory' to see what you're carrying
  - Type 'equipment' to see what you have equipped

  LOOT:
  - Monsters drop items when defeated
  - Type 'get <item>' to pick them up

"Buy a weapon before heading into the tower!"`, guide.GetName(), p.GetGold())
}

// getGuidePortalTopic explains the portal system
func getGuidePortalTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s gestures to a shimmering spot nearby.

"Once you've explored the tower, travel becomes much easier!"

  - Each floor's STAIRWAY has a magical portal
  - When you find a stairway, you 'discover' that floor's portal
  - Type 'portal' to see floors you've discovered
  - Type 'portal <floor>' to instantly travel there!

  - Town Square (floor 0) is always available
  - Great for quick trips back to heal, shop, and save!

"Discover portals as you climb - they're lifesavers!"`, guide.GetName())
}

// getGuideCommandsTopic provides a quick reference
func getGuideCommandsTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s counts off on his fingers.

"Here are the commands you'll use most:"

  MOVEMENT:     north, south, east, west, up, down (or n,s,e,w,u,d)
  LOOKING:      look, exits, inventory, equipment
  COMBAT:       attack <target>, flee, consider <target>
  ITEMS:        get <item>, drop <item>, wield <weapon>, wear <armor>
  RECOVERY:     pray (at altar), sleep, wake
  SOCIAL:       say <msg>, tell <player> <msg>, who
  TRAVEL:       portal, portal <floor>
  COMMERCE:     shop, buy <item>, sell <item>, gold
  OTHER:        help, talk <npc>, time

"Type 'help' for the full list, or 'help <command>' for details!"`, guide.GetName())
}

// executeUnlock handles the unlock command for locked doors
func executeUnlock(c *Command, p PlayerInterface) string {
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

// executePray handles praying at an altar
func executePray(c *Command, p PlayerInterface) string {
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

	logger.Debug("Player prayed at altar",
		"player", p.GetName(),
		"room", room.GetID(),
		"healed", healAmount,
		"mana_restored", manaAmount)

	// Build response based on what was restored
	if healAmount > 0 && manaAmount > 0 {
		return fmt.Sprintf("You kneel before the altar and pray. A warm, golden light washes over you, restoring your body and spirit. (+%d HP, +%d Mana)", healAmount, manaAmount)
	} else if healAmount > 0 {
		return fmt.Sprintf("You kneel before the altar and pray. A warm, golden light washes over you, healing your wounds. (+%d HP)", healAmount)
	} else {
		return fmt.Sprintf("You kneel before the altar and pray. A calm energy fills your mind, restoring your spirit. (+%d Mana)", manaAmount)
	}
}
