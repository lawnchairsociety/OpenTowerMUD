package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// executeTake picks up an item from the current room
func executeTake(c *Command, p PlayerInterface) string {
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
			logger.Debug("Item taken (key)",
				"player", p.GetName(),
				"item", foundItem.Name,
				"room", room.GetID())
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
		logger.Debug("Item taken",
			"player", p.GetName(),
			"item", foundItem.Name,
			"room", room.GetID())
		return fmt.Sprintf("You take the %s.", foundItem.Name)
	}

	return fmt.Sprintf("You can't take the %s.", foundItem.Name)
}

// executeDrop drops an item from inventory into the current room
func executeDrop(c *Command, p PlayerInterface) string {
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
		logger.Debug("Item dropped",
			"player", p.GetName(),
			"item", foundItem.Name,
			"room", room.GetID())
		return fmt.Sprintf("You drop the %s.", foundItem.Name)
	}

	return fmt.Sprintf("You can't drop the %s.", foundItem.Name)
}

// executeInventory shows the player's inventory
func executeInventory(c *Command, p PlayerInterface) string {
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

// executeEquipment shows the player's equipped items
func executeEquipment(c *Command, p PlayerInterface) string {
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

// executeWield equips a weapon from inventory
func executeWield(c *Command, p PlayerInterface) string {
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

// executeWear equips armor from inventory
func executeWear(c *Command, p PlayerInterface) string {
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

// executeRemove unequips an item and puts it in inventory
func executeRemove(c *Command, p PlayerInterface) string {
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

// executeHold holds an item in hand
func executeHold(c *Command, p PlayerInterface) string {
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

// executeEat consumes a food item
func executeEat(c *Command, p PlayerInterface) string {
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

// executeDrink consumes a drink or potion
func executeDrink(c *Command, p PlayerInterface) string {
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

// executeUse uses a consumable item or room feature
func executeUse(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: use <item or feature>"); err != nil {
		return err.Error()
	}

	targetName := c.GetItemName()
	targetLower := strings.ToLower(targetName)

	// Get current room for feature checking
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Priority 1: Check inventory for consumable items
	item, foundItem := p.FindItem(targetName)
	if foundItem && item.Consumable {
		// Consume the item
		result := p.ConsumeItem(item)
		p.RemoveItem(item.Name)
		return result
	}

	// Priority 2: Check for usable room feature
	if room.HasFeature(targetLower) {
		handler := GetFeatureHandler(targetLower)
		if handler != nil {
			// Check player state - can't use features while fighting or sleeping
			state := p.GetState()
			if state == "Fighting" {
				return "You can't do that while fighting!"
			}
			if state == "Sleeping" {
				return "You are asleep. Wake up first."
			}
			return handler(c, p, room)
		}
		// Feature exists but has no use handler
		return fmt.Sprintf("You can't use the %s directly. Try examining it for hints on what you can do.", targetName)
	}

	// Neither consumable item nor feature found
	if foundItem {
		// We had an item but it wasn't consumable
		return fmt.Sprintf("You can't use %s like that.", item.Name)
	}

	return fmt.Sprintf("You don't have '%s' and there's no %s here to use.", targetName, targetName)
}

// findShopNPC finds a merchant NPC in the room
func findShopNPC(room RoomInterface) *npc.NPC {
	npcs := room.GetNPCs()
	for _, n := range npcs {
		if n.HasShopInventory() {
			return n
		}
	}
	return nil
}
