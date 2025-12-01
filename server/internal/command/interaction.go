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
func handleGuideInteraction(c *Command, p PlayerInterface, guide *npc.NPC) string {
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
