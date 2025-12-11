package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/text"
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
		return fmt.Sprintf("The %s is dead and can't respond.", foundNPC.GetName())
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

	// Track lore NPC conversations for the Keeper of Forgotten Lore title
	if foundNPC.IsLoreNPC() {
		npcID := foundNPC.GetNPCID()
		if npcID != "" {
			if p.TalkToLoreNPC(npcID) {
				// First time talking to this lore NPC
				talkedCount := len(p.GetTalkedToLoreNPCs())
				response += fmt.Sprintf("\n\n*** You have spoken with this lore keeper. (%d/5 scholars found) ***", talkedCount)

				// Check if player has now talked to all lore NPCs
				if p.HasTalkedToAllLoreNPCs() {
					title := "Keeper of Forgotten Lore"
					if !p.HasEarnedTitle(title) {
						p.EarnTitle(title)
						response += fmt.Sprintf("\n\n================================================================================\n                    TITLE EARNED: %s\n\n  You have spoken with all five lore keepers of the Great Labyrinth!\n  The ancient secrets of the maze are now yours to keep.\n================================================================================", title)
						// Announce to server
						if server, ok := p.GetServer().(ServerInterface); ok {
							server.BroadcastToAll(fmt.Sprintf("\n*** %s has earned the title: %s ***\n", p.GetName(), title))
						}
					}
				}
			}
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
		"~ Through tower floors they climb so high ~",
		"~ A hero's tale that will never die! ~",
	}

	t := text.GetInstance()
	if t != nil {
		return fmt.Sprintf(t.GetBardSong(),
			bard.GetName(), p.GetName(),
			songLines[0], songLines[1], songLines[2], songLines[3])
	}

	// Fallback if text not loaded
	return fmt.Sprintf("The %s strums his lute and nods at you.", bard.GetName())
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

	t := text.GetInstance()
	if t == nil {
		return fmt.Sprintf("%s smiles warmly but seems distracted.", guide.GetName())
	}

	switch topic {
	case "tower", "dungeon", "floors":
		return fmt.Sprintf(t.GetGuideTopic("tower"), guide.GetName())
	case "combat", "fighting", "fight", "attack":
		return fmt.Sprintf(t.GetGuideTopic("combat"), guide.GetName())
	case "save", "saving", "bard":
		return fmt.Sprintf(t.GetGuideTopic("save"), guide.GetName())
	case "shop", "gold", "equipment", "gear", "items":
		return fmt.Sprintf(t.GetGuideTopic("shop"), guide.GetName(), p.GetGold())
	case "portal", "portals", "travel":
		return fmt.Sprintf(t.GetGuideTopic("portal"), guide.GetName())
	case "quest", "quests", "journal":
		return fmt.Sprintf(t.GetGuideTopic("quests"), guide.GetName())
	case "commands", "help":
		return fmt.Sprintf(t.GetGuideTopic("commands"), guide.GetName())
	default:
		return getGuideGreeting(guide, p)
	}
}

// getGuideGreeting returns Aldric's initial greeting with topic list
func getGuideGreeting(guide *npc.NPC, p PlayerInterface) string {
	t := text.GetInstance()
	if t == nil {
		return fmt.Sprintf("%s smiles warmly but seems distracted.", guide.GetName())
	}

	guideName := strings.ToLower(strings.Split(guide.GetName(), " ")[0])
	return fmt.Sprintf(t.GetGuideGreeting(),
		guide.GetName(), p.GetName(),
		guideName, guideName, guideName, guideName, guideName, guideName, guideName)
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
		return "You don't have the key to unlock this door."
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
	t := text.GetInstance()
	if t != nil {
		return t.GetTrainerReject(className)
	}
	return "You do not meet the requirements to learn this class."
}

// getTrainerAcceptanceDialogue returns flavor text when a trainer accepts a player
func getTrainerAcceptanceDialogue(trainerName, className, playerName string) string {
	t := text.GetInstance()
	if t != nil {
		return fmt.Sprintf(t.GetTrainerAccept(className), trainerName, playerName)
	}
	return fmt.Sprintf("%s nods approvingly.\n\n\"You have begun your training as a %s, %s.\"", trainerName, strings.Title(className), playerName)
}

// getClassWelcomeMessage returns information about what the new class provides
func getClassWelcomeMessage(className string) string {
	t := text.GetInstance()
	if t != nil {
		return t.GetClassWelcome(className)
	}
	return fmt.Sprintf("You have learned the ways of the %s.", strings.Title(className))
}
