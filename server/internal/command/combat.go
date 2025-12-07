package command

import (
	"fmt"
	"strings"
)

// executeAttack initiates combat with an NPC
func executeAttack(c *Command, p PlayerInterface) string {
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
	room, ok := GetRoom(p)
	if !ok {
		return "Error: You are not in a valid room."
	}

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

// executeFlee attempts to escape from combat
func executeFlee(c *Command, p PlayerInterface) string {
	// Check if in combat
	if !p.IsInCombat() {
		return "You aren't fighting anyone!"
	}

	room, ok := GetRoom(p)
	if !ok {
		return "Error: You are not in a valid room."
	}

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

	newRoom, _ := GetRoom(p)
	if newRoom == nil {
		return fmt.Sprintf("You flee %s!", direction)
	}
	return fmt.Sprintf("You flee %s!\n\n%s", direction, newRoom.GetDescriptionForPlayer(p.GetName()))
}

// executeConsider evaluates an NPC's difficulty
func executeConsider(c *Command, p PlayerInterface) string {
	// Require target name
	if err := c.RequireArgs(1, "Usage: consider <target>"); err != nil {
		return err.Error()
	}

	targetName := c.GetItemName()
	targetLower := strings.ToLower(targetName)

	// Handle "consider self" or "consider me"
	if targetLower == "self" || targetLower == "me" {
		return executeConsiderSelf(c, p)
	}

	room, ok := GetRoom(p)
	if !ok {
		return "Error: You are not in a valid room."
	}

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

// executeConsiderSelf shows the player their own stats (delegates to score command)
func executeConsiderSelf(c *Command, p PlayerInterface) string {
	return executeScore(c, p)
}
