package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)


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

	// Find the NPC - try first arg, then full name for multi-word NPCs
	// This allows "talk aldric tower" to find "aldric" with topic "tower"
	var foundNPC *npc.NPC
	var npcName string

	// First try just the first argument (e.g., "aldric" from "talk aldric tower")
	if len(c.Args) >= 1 {
		npcName = c.Args[0]
		foundNPC = worldRoom.FindNPC(npcName)
	}

	// If not found, try full name for multi-word NPCs (e.g., "old guide")
	if foundNPC == nil {
		npcName = c.GetItemName()
		foundNPC = worldRoom.FindNPC(npcName)
	}

	if foundNPC == nil {
		return fmt.Sprintf("You don't see '%s' here.", npcName)
	}

	// Check if NPC is alive
	if !foundNPC.IsAlive() {
		return fmt.Sprintf("The %s is dead and cannot respond.", foundNPC.GetName())
	}

	// Special handling for the bard - flavor interaction
	if strings.Contains(strings.ToLower(foundNPC.GetName()), "bard") {
		return handleBardInteraction(p, foundNPC)
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

	response := fmt.Sprintf("The %s says, \"%s\"", foundNPC.GetName(), dialogue)

	// Check if NPC is a quest giver with available quests
	if foundNPC.IsQuestGiver() {
		questHint := getQuestGiverHint(p, foundNPC)
		if questHint != "" {
			response += "\n\n" + questHint
		}
	}

	return response
}

// getQuestGiverHint returns a hint about available quests from this NPC
func getQuestGiverHint(p PlayerInterface, npcGiver *npc.NPC) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return ""
	}

	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return ""
	}

	playerState := p.GetQuestState()
	available := questRegistry.GetAvailableQuestsForPlayer(npcGiver.GetName(), playerState)

	if len(available) == 0 {
		return ""
	}

	return fmt.Sprintf("[%s has %d quest(s) available. Type 'quests available' to see them.]", npcGiver.GetName(), len(available))
}

// handleBardInteraction provides flavor dialogue with the bard
func handleBardInteraction(p PlayerInterface, bard *npc.NPC) string {
	// Generate a fun song snippet based on player stats
	songLines := []string{
		fmt.Sprintf("~ Of %s the brave, level %d and bold ~", p.GetName(), p.GetLevel()),
		fmt.Sprintf("~ With %d gold in pocket, adventures untold ~", p.GetGold()),
		fmt.Sprintf("~ Through tower floors they climb so high ~"),
		fmt.Sprintf("~ A hero's tale that will never die! ~"),
	}

	return fmt.Sprintf(`The %s strums his lute and smiles warmly at you.

"Ah, %s! Let me sing of your adventures..."

He clears his throat and begins to play:

%s
%s
%s
%s

The bard bows with a flourish. "May your legend grow ever greater, friend!"`,
		bard.GetName(), p.GetName(),
		songLines[0], songLines[1], songLines[2], songLines[3])
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
	case "quest", "quests", "journal":
		return getGuideQuestTopic(guide)
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
  talk %s save     - How your progress is saved
  talk %s shop     - Buying, selling, and equipment
  talk %s portal   - Fast travel between floors
  talk %s quests   - Finding and completing quests
  talk %s commands - Quick reference of useful commands

He leans on his walking stick. "Just ask about any topic, friend!"`,
		guide.GetName(), p.GetName(),
		strings.ToLower(strings.Split(guide.GetName(), " ")[0]),
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

"That's the Endless Tower - it's why we're all here." He squints upward,
shielding his eyes. "The tower shimmers and shifts... some say it rearranges
itself when no one is watching. No one knows how tall it truly is."

  - Go SOUTH four times from Town Square to reach the TOWER ENTRANCE
  - Type 'up' to begin climbing
  - Each floor has corridors, chambers, and treasure rooms
  - Every 10th floor has a powerful BOSS guarding the way forward!
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

// getGuideSaveTopic explains the auto-save mechanic
func getGuideSaveTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s nods reassuringly.

"Ah, worried about losing your progress? Fear not!"

  Your progress is saved automatically:
  - When you disconnect or quit the game
  - When the server shuts down for maintenance
  - After important events like learning a new class

  You don't need to do anything special - just play and enjoy!

"The magic of this realm remembers all your deeds, friend."`, guide.GetName())
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

// getGuideQuestTopic explains the quest system
func getGuideQuestTopic(guide *npc.NPC) string {
	return fmt.Sprintf(`%s strokes his beard thoughtfully.

"Quests! Yes, many folk in the city need help with tasks. Complete their
quests and you'll be rewarded handsomely!"

  FINDING QUESTS:
  - Look for NPCs with tasks - they'll hint at having quests when you talk
  - Type 'quests available' to see what quests nearby NPCs offer

  ACCEPTING QUESTS:
  - Type 'accept <quest name>' to take on a quest
  - Your quest journal tracks all your active quests

  TRACKING PROGRESS:
  - Type 'quest' to see your journal summary
  - Type 'quest list' to see all active quests with progress
  - Type 'quest <name>' for details on a specific quest

  COMPLETING QUESTS:
  - Fulfill the objectives (kill monsters, collect items, explore places)
  - Return to the quest giver and type 'complete' to turn it in
  - Receive gold, experience, items, or even titles as rewards!

  TITLES:
  - Some quests reward titles you can display with your name
  - Type 'title' to see earned titles, 'title <name>' to set one

"I have a few quests myself, if you're interested!"`, guide.GetName())
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
  QUESTS:       quest, accept <quest>, complete, title
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

// TrainingCost is the base cost in gold to learn a new class
const TrainingCost = 500

// executeTrain handles learning a new class from a trainer NPC
func executeTrain(c *Command, p PlayerInterface) string {
	// Check player state
	state := p.GetState()
	if state == "Fighting" {
		return "You can't train while fighting!"
	}
	if state == "Sleeping" {
		return "You are asleep and can't train. Wake up first."
	}

	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find a trainer NPC in the room
	var trainer *npc.NPC
	for _, n := range worldRoom.GetNPCs() {
		if n.IsTrainer() && n.IsAlive() {
			trainer = n
			break
		}
	}

	if trainer == nil {
		return "There is no class trainer here.\n\nClass trainers can be found throughout the city:\n  Warrior - Training Hall\n  Mage - Royal Library\n  Cleric - Temple\n  Rogue - Tavern\n  Ranger - North Gate\n  Paladin - Castle Hall"
	}

	trainerClassName := trainer.GetTrainerClass()

	// Check if player can multiclass at all
	if !p.CanMulticlass() {
		return fmt.Sprintf("%s looks at you appraisingly.\n\n\"%s, you must reach level %d in your primary class before I can teach you a new path. Return when you have proven yourself.\"\n\n(Multiclassing unlocks at primary class level %d)",
			trainer.GetName(), p.GetName(), class.MinLevelForMulticlass, class.MinLevelForMulticlass)
	}

	// Check if player already has this class
	classLevels := p.GetAllClassLevelsMap()
	if level, has := classLevels[trainerClassName]; has && level > 0 {
		return fmt.Sprintf("%s nods approvingly.\n\n\"You already walk the path of the %s, %s. Use 'class switch %s' if you wish to focus your training on this discipline.\"",
			trainer.GetName(), strings.Title(trainerClassName), p.GetName(), trainerClassName)
	}

	// Check if player meets requirements for this class
	canMulti, reason := p.CanMulticlassInto(trainerClassName)
	if !canMulti {
		return fmt.Sprintf("%s examines you carefully.\n\n\"%s\"\n\n%s",
			trainer.GetName(), getTrainerRejectionDialogue(trainerClassName), reason)
	}

	// Check gold
	if p.GetGold() < TrainingCost {
		return fmt.Sprintf("%s strokes their chin thoughtfully.\n\n\"Training in the ways of the %s requires dedication and... resources. %d gold, to be precise.\"\n\nYou have only %d gold.",
			trainer.GetName(), strings.Title(trainerClassName), TrainingCost, p.GetGold())
	}

	// Deduct gold and add the class
	p.SpendGold(TrainingCost)
	err := p.AddNewClass(trainerClassName)
	if err != nil {
		// Refund on error
		p.AddGold(TrainingCost)
		return fmt.Sprintf("Something went wrong: %v", err)
	}

	// Save the player after learning a new class
	server, ok := p.GetServer().(ServerInterface)
	if ok {
		if saveErr := server.SavePlayer(p); saveErr != nil {
			logger.Warning("Failed to save player after learning class",
				"player", p.GetName(),
				"class", trainerClassName,
				"error", saveErr)
		}
	}

	return fmt.Sprintf(`%s

%s

You are now a %s!

Your new class starts at level 1. XP is now being earned for %s.
Use 'class' to view your classes, or 'class switch <class>' to change which class gains XP.

(-%d gold, remaining: %d)`,
		getTrainerAcceptanceDialogue(trainer.GetName(), trainerClassName, p.GetName()),
		getClassWelcomeMessage(trainerClassName),
		strings.Title(trainerClassName),
		strings.Title(trainerClassName),
		TrainingCost, p.GetGold())
}

// getTrainerRejectionDialogue returns flavor text when a trainer rejects a player
func getTrainerRejectionDialogue(className string) string {
	switch className {
	case "warrior":
		return "Your body lacks the strength required for a warrior's training. Build your muscles first."
	case "mage":
		return "I sense your mind is... underdeveloped for arcane study. Sharpen your intellect."
	case "cleric":
		return "The divine requires wisdom to channel. Your spirit is not yet ready."
	case "rogue":
		return "You move like a stone golem. Work on your agility before seeking my teachings."
	case "ranger":
		return "A ranger needs both quick reflexes and keen perception. You lack these qualities."
	case "paladin":
		return "A paladin requires both strength of arm and force of personality. You fall short."
	default:
		return "You do not meet the requirements to learn this class."
	}
}

// getTrainerAcceptanceDialogue returns flavor text when a trainer accepts a player
func getTrainerAcceptanceDialogue(trainerName, className, playerName string) string {
	switch className {
	case "warrior":
		return fmt.Sprintf("%s grips your forearm firmly.\n\n\"Welcome to the path of iron and blood, %s. Your muscles will ache, your bones will bruise, but you will emerge unbreakable.\"", trainerName, playerName)
	case "mage":
		return fmt.Sprintf("%s's eyes glow with arcane power.\n\n\"The weave of magic opens to you now, %s. Feel the energy of the world flowing through your mind. Use it wisely... or not. I find destruction quite educational.\"", trainerName, playerName)
	case "cleric":
		return fmt.Sprintf("%s places a gentle hand on your head.\n\n\"The divine light shines upon you, %s. You are now a vessel for powers beyond mortal understanding. Go forth and heal the world... or smite the wicked.\"", trainerName, playerName)
	case "rogue":
		return fmt.Sprintf("%s seems to appear from nowhere.\n\n\"Clever choice, %s. The shadows welcome you. Remember: the best fight is the one your enemy never sees coming.\"", trainerName, playerName)
	case "ranger":
		return fmt.Sprintf("%s whistles, and the wolf at their feet howls approval.\n\n\"The wilds accept you, %s. Every beast, every track, every rustle in the undergrowth will speak to you now. Listen well.\"", trainerName, playerName)
	case "paladin":
		return fmt.Sprintf("%s draws their blade and touches it to your shoulder.\n\n\"Rise, %s, champion of light. The oath is sworn. Your blade shall be the bane of darkness, your shield the hope of the innocent.\"", trainerName, playerName)
	default:
		return fmt.Sprintf("%s nods approvingly.\n\n\"You have begun your training as a %s, %s.\"", trainerName, strings.Title(className), playerName)
	}
}

// getClassWelcomeMessage returns information about what the new class provides
func getClassWelcomeMessage(className string) string {
	switch className {
	case "warrior":
		return `As a Warrior, you gain:
  - Proficiency with all weapons and heavy armor
  - High hit die (d10) for maximum HP
  - Melee damage bonuses as you level
  - Second Wind ability at higher levels`
	case "mage":
		return `As a Mage, you gain:
  - Access to powerful damage spells (fireball, ice storm, meteor)
  - Intelligence-based spellcasting
  - High mana pool growth
  - Arcane Shield at higher levels`
	case "cleric":
		return `As a Cleric, you gain:
  - Access to healing spells (heal, cure wounds, resurrection)
  - Wisdom-based spellcasting
  - Medium armor proficiency
  - Divine protection abilities at higher levels`
	case "rogue":
		return `As a Rogue, you gain:
  - Sneak Attack bonus damage
  - Finesse weapon proficiency (DEX for attack/damage)
  - Light armor proficiency
  - Evasion and assassination abilities at higher levels`
	case "ranger":
		return `As a Ranger, you gain:
  - Ranged weapon proficiency and damage bonuses
  - Favored Enemy bonus against beasts
  - Nature spells (hunter's mark, spike growth)
  - Medium armor proficiency`
	case "paladin":
		return `As a Paladin, you gain:
  - Smite ability for extra radiant damage
  - Bonus damage against undead and demons
  - Access to healing spells
  - Heavy armor and martial weapon proficiency`
	default:
		return fmt.Sprintf("You have learned the ways of the %s.", strings.Title(className))
	}
}
