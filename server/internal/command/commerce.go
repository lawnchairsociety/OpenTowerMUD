package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// executeShop shows the shop inventory
func executeShop(c *Command, p PlayerInterface) string {
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findMerchantNPC(room)
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
func executeBuy(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Buy what? Usage: buy <item name>"); err != nil {
		return err.Error()
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findMerchantNPC(room)
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
		logger.Debug("Item purchased (key)",
			"player", p.GetName(),
			"item", foundItem.Name,
			"price", price,
			"seller", sellerName,
			"gold_remaining", p.GetGold())
		return fmt.Sprintf("You purchase a %s for %d gold and add it to your key ring.\nGold remaining: %d", foundItem.Name, price, p.GetGold())
	}

	p.AddItem(foundItem)
	logger.Debug("Item purchased",
		"player", p.GetName(),
		"item", foundItem.Name,
		"price", price,
		"seller", sellerName,
		"gold_remaining", p.GetGold())
	return fmt.Sprintf("You purchase a %s for %d gold.\nGold remaining: %d", foundItem.Name, price, p.GetGold())
}

// executeSell handles selling items to a shop or merchant
func executeSell(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Sell what? Usage: sell <item name>"); err != nil {
		return err.Error()
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Find an NPC with shop inventory in this room
	shopNPC := findMerchantNPC(room)
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

	logger.Debug("Item sold",
		"player", p.GetName(),
		"item", removedItem.Name,
		"price", sellPrice,
		"buyer", shopNPC.GetName(),
		"is_merchant", isMerchant,
		"gold_total", p.GetGold())

	if isMerchant {
		return fmt.Sprintf("The old merchant grumbles as he hands you %d gold for your %s.\n\"Don't expect charity from me, adventurer.\"\nGold: %d", sellPrice, removedItem.Name, p.GetGold())
	}
	return fmt.Sprintf("You sell your %s for %d gold.\nGold: %d", removedItem.Name, sellPrice, p.GetGold())
}

// executeGold shows the player's gold amount
func executeGold(c *Command, p PlayerInterface) string {
	return fmt.Sprintf("You have %d gold.", p.GetGold())
}

// executeGive transfers items or gold to another player
func executeGive(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: give <item> <player> or give <amount> gold <player>"); err != nil {
		return err.Error()
	}

	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	args := c.Args

	// Check if giving gold: "give <amount> gold <player>" or "give <amount> gold to <player>"
	if len(args) >= 3 && strings.ToLower(args[1]) == "gold" {
		return executeGiveGold(args, p, server, room)
	}

	// Otherwise giving an item: "give <item> <player>" or "give <item> to <player>"
	return executeGiveItem(args, p, server, room)
}

// executeGiveGold handles giving gold to another player
func executeGiveGold(args []string, p PlayerInterface, server ServerInterface, room RoomInterface) string {
	// Parse amount
	amount := 0
	_, err := fmt.Sscanf(args[0], "%d", &amount)
	if err != nil || amount <= 0 {
		return "Invalid amount. Usage: give <amount> gold <player>"
	}

	// Check if player has enough gold
	if p.GetGold() < amount {
		return fmt.Sprintf("You only have %d gold.", p.GetGold())
	}

	// Find target player name (skip "to" if present)
	targetArgs := args[2:] // Skip amount and "gold"
	if len(targetArgs) > 0 && strings.ToLower(targetArgs[0]) == "to" {
		targetArgs = targetArgs[1:]
	}
	if len(targetArgs) == 0 {
		return "Give gold to whom? Usage: give <amount> gold <player>"
	}

	targetName := strings.Join(targetArgs, " ")
	target := findPlayerInRoom(targetName, server, room, p)
	if target == nil {
		return fmt.Sprintf("%s is not here.", targetName)
	}

	// Transfer the gold
	p.SpendGold(amount)
	target.AddGold(amount)

	// Notify target
	target.SendMessage(fmt.Sprintf("%s gives you %d gold.\n", p.GetName(), amount))

	logger.Debug("Gold given",
		"giver", p.GetName(),
		"receiver", target.GetName(),
		"amount", amount)

	return fmt.Sprintf("You give %s %d gold.", target.GetName(), amount)
}

// executeGiveItem handles giving an item to another player
func executeGiveItem(args []string, p PlayerInterface, server ServerInterface, room RoomInterface) string {
	// Find where "to" appears or try to find a player match
	// Formats: "give sword bob" or "give rusty sword to bob"
	var itemParts []string
	var targetParts []string
	foundTo := false

	for i, arg := range args {
		if strings.ToLower(arg) == "to" && i > 0 {
			itemParts = args[:i]
			targetParts = args[i+1:]
			foundTo = true
			break
		}
	}

	// If no "to" found, assume last arg(s) is player name
	// Try progressively shorter player names from the end
	if !foundTo {
		for i := len(args) - 1; i >= 1; i-- {
			candidateTarget := strings.Join(args[i:], " ")
			target := findPlayerInRoom(candidateTarget, server, room, p)
			if target != nil {
				itemParts = args[:i]
				targetParts = args[i:]
				break
			}
		}
	}

	if len(itemParts) == 0 || len(targetParts) == 0 {
		return "Usage: give <item> <player> or give <item> to <player>"
	}

	itemName := strings.Join(itemParts, " ")
	targetName := strings.Join(targetParts, " ")

	// Find the item in player's inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s'.", itemName)
	}

	// Find target player in same room
	target := findPlayerInRoom(targetName, server, room, p)
	if target == nil {
		return fmt.Sprintf("%s is not here.", targetName)
	}

	// Check if target can carry the item
	if !target.CanCarry(item) {
		return fmt.Sprintf("%s can't carry any more weight.", target.GetName())
	}

	// Transfer the item
	removedItem, removed := p.RemoveItem(item.Name)
	if !removed {
		return "Something went wrong trying to give that item."
	}

	target.AddItem(removedItem)

	// Notify target
	target.SendMessage(fmt.Sprintf("%s gives you %s.\n", p.GetName(), removedItem.Name))

	logger.Debug("Item given",
		"giver", p.GetName(),
		"receiver", target.GetName(),
		"item", removedItem.Name)

	return fmt.Sprintf("You give %s to %s.", removedItem.Name, target.GetName())
}

// findPlayerInRoom finds a player by name who is in the same room
func findPlayerInRoom(name string, server ServerInterface, room RoomInterface, excludePlayer PlayerInterface) PlayerInterface {
	targetIface := server.FindPlayer(name)
	if targetIface == nil {
		return nil
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return nil
	}

	// Can't give to yourself
	if target.GetName() == excludePlayer.GetName() {
		return nil
	}

	// Check if target is in the same room
	targetRoomIface := target.GetCurrentRoom()
	targetRoom, ok := targetRoomIface.(RoomInterface)
	if !ok {
		return nil
	}

	if targetRoom.GetID() != room.GetID() {
		return nil
	}

	return target
}

// findMerchantNPC finds an NPC with shop inventory in the given room
func findMerchantNPC(room RoomInterface) *npc.NPC {
	npcs := room.GetNPCs()
	for _, n := range npcs {
		if n.HasShopInventory() {
			return n
		}
	}
	return nil
}
