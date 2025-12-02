package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// executeCast handles casting spells
func executeCast(c *Command, p PlayerInterface) string {
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
		return fmt.Sprintf("Unknown spell: '%s'. Type 'spells' to see your available spells.", spellName)
	}

	// Check if player can cast this spell based on class and level
	// With the new class system, spells are automatically available based on class levels
	if !p.CanCastSpellForClass(spell.AllowedClasses, spell.Level) {
		// Check if it's a class restriction or level restriction
		if len(spell.AllowedClasses) > 0 {
			return fmt.Sprintf("You cannot cast '%s'. This spell requires being a %s.", spell.Name, strings.Join(spell.AllowedClasses, " or "))
		}
		return fmt.Sprintf("You cannot cast '%s' yet. (Requires level %d)", spell.Name, spell.Level)
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
		return castRoomSpell(c, p, spell)
	}

	// Determine how to cast based on target and spell capabilities
	if targetName == "" {
		// No target specified - cast on self if possible
		if spell.CanTargetSelf() {
			return castSelfSpell(c, p, spell)
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
					return castAllySpell(c, p, spell, targetPlayer)
				}
			}
		}
	}

	// Then check if target is an NPC
	if spell.CanTargetEnemy() {
		if npc := room.FindNPC(targetName); npc != nil {
			return castEnemySpell(c, p, spell, npc, room)
		}
	}

	// Target not found
	return fmt.Sprintf("You don't see '%s' here.", targetName)
}

// castSelfSpell handles spells that target the caster
func castSelfSpell(c *Command, p PlayerInterface, spell *spells.Spell) string {
	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	logger.Debug("Spell cast (self)",
		"player", p.GetName(),
		"spell", spell.Name,
		"mana_cost", spell.ManaCost,
		"cooldown", spell.Cooldown)

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
func castAllySpell(c *Command, p PlayerInterface, spell *spells.Spell, target PlayerInterface) string {
	// Deduct mana
	if !p.UseMana(spell.ManaCost) {
		return "Not enough mana!"
	}

	// Start cooldown
	if spell.Cooldown > 0 {
		p.StartSpellCooldown(spell.ID, spell.Cooldown)
	}

	logger.Debug("Spell cast (ally)",
		"player", p.GetName(),
		"spell", spell.Name,
		"target", target.GetName(),
		"mana_cost", spell.ManaCost,
		"cooldown", spell.Cooldown)

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
func castEnemySpell(c *Command, p PlayerInterface, spell *spells.Spell, targetNPC *npc.NPC, room *world.Room) string {
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

	logger.Debug("Spell cast (enemy)",
		"player", p.GetName(),
		"spell", spell.Name,
		"target", targetNPC.GetName(),
		"mana_cost", spell.ManaCost,
		"cooldown", spell.Cooldown)

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
func castRoomSpell(c *Command, p PlayerInterface, spell *spells.Spell) string {
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

	logger.Debug("Spell cast (room)",
		"player", p.GetName(),
		"spell", spell.Name,
		"target_count", len(targetNPCs),
		"mana_cost", spell.ManaCost,
		"cooldown", spell.Cooldown)

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

// executeSpells shows the player's available spells based on class levels
func executeSpells(c *Command, p PlayerInterface) string {
	// Get the spell registry from server
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	registry := server.GetSpellRegistry()
	if registry == nil {
		return "Magic is not available."
	}

	// Get spells available based on class levels
	classLevels := p.GetAllClassLevelsMap()
	availableSpells := registry.GetSpellsForClasses(classLevels)

	if len(availableSpells) == 0 {
		return "You don't have access to any spells yet."
	}

	// Sort spells by level, then by name
	for i := 0; i < len(availableSpells)-1; i++ {
		for j := i + 1; j < len(availableSpells); j++ {
			if availableSpells[i].Level > availableSpells[j].Level ||
				(availableSpells[i].Level == availableSpells[j].Level && availableSpells[i].Name > availableSpells[j].Name) {
				availableSpells[i], availableSpells[j] = availableSpells[j], availableSpells[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("=== Your Spells ===\n")

	for _, spell := range availableSpells {
		// Check cooldown status
		onCooldown, remaining := p.IsSpellOnCooldown(spell.ID)
		status := "[Ready]"
		if onCooldown {
			status = fmt.Sprintf("[Cooldown: %ds]", remaining)
		}

		// Show class restriction if applicable
		classReq := ""
		if len(spell.AllowedClasses) > 0 {
			classReq = fmt.Sprintf(" (%s)", spell.AllowedClasses[0])
		}

		sb.WriteString(fmt.Sprintf("  %-15s - %s (%d mana)%s %s\n",
			spell.Name, spell.Description, spell.ManaCost, classReq, status))
	}

	sb.WriteString(fmt.Sprintf("\nMana: %d/%d", p.GetMana(), p.GetMaxMana()))

	return sb.String()
}
