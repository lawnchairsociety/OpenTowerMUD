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
