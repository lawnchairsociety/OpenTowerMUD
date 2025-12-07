package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
)

// executeStall handles the stall command and its subcommands
func executeStall(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		return executeStallHelp()
	}

	subcommand := strings.ToLower(c.Args[0])
	switch subcommand {
	case "open":
		return executeStallOpen(c, p)
	case "close":
		return executeStallClose(c, p)
	case "add":
		return executeStallAdd(c, p)
	case "remove":
		return executeStallRemove(c, p)
	case "list":
		return executeStallList(c, p)
	case "help":
		return executeStallHelp()
	default:
		return fmt.Sprintf("Unknown stall command: %s\n%s", subcommand, executeStallHelp())
	}
}

// executeStallHelp shows the stall command help
func executeStallHelp() string {
	return `=== Player Stall Commands ===
Set up a stall to sell items to other players!

Commands:
  stall open           - Open your stall for business
  stall close          - Close your stall (items return to inventory)
  stall add <item> <price> - Add an item from your inventory to your stall
  stall remove <item>  - Remove an item from your stall
  stall list           - View items in your stall

Other players can use:
  browse <player>      - View a player's stall
  purchase <item> from <player> - Buy from a player's stall

Notes:
  - You can only open a stall in the city (floor 0)
  - Your stall closes automatically if you leave the room or disconnect
  - Items in your stall are not in your inventory until you remove them`
}

// executeStallOpen opens the player's stall for business
func executeStallOpen(c *Command, p PlayerInterface) string {
	// Check if already open
	if p.IsStallOpen() {
		return "Your stall is already open for business."
	}

	// Check if in city (floor 0)
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if room.GetFloor() != 0 {
		return "You can only set up a stall in the city. Return to floor 0 to open your stall."
	}

	// Open the stall
	p.OpenStall()

	// Get server to broadcast
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if ok {
		server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s has opened a stall for business.", p.GetName()), p)
	}

	logger.Debug("Stall opened",
		"player", p.GetName(),
		"room", room.GetID())

	if len(p.GetStallInventory()) == 0 {
		return "You open your stall for business.\nYour stall is empty. Use 'stall add <item> <price>' to add items for sale."
	}
	return "You open your stall for business. Customers can now browse and purchase your wares!"
}

// executeStallClose closes the player's stall
func executeStallClose(c *Command, p PlayerInterface) string {
	if !p.IsStallOpen() {
		return "Your stall is not open."
	}

	// Get items before closing
	stallItems := p.GetStallInventory()
	itemCount := len(stallItems)

	// Clear the stall (returns items to inventory)
	returnedItems := p.ClearStall()
	for _, item := range returnedItems {
		p.AddItem(item)
	}

	// Get server to broadcast
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if ok {
		serverIface := p.GetServer()
		server, serverOk := serverIface.(ServerInterface)
		if serverOk {
			server.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s has closed their stall.", p.GetName()), p)
		}
	}

	logger.Debug("Stall closed",
		"player", p.GetName(),
		"items_returned", itemCount)

	if itemCount > 0 {
		return fmt.Sprintf("You close your stall. %d item(s) have been returned to your inventory.", itemCount)
	}
	return "You close your stall."
}

// executeStallAdd adds an item to the player's stall
func executeStallAdd(c *Command, p PlayerInterface) string {
	// Need at least "add <item> <price>"
	if len(c.Args) < 3 {
		return "Usage: stall add <item> <price>\nExample: stall add sword 50"
	}

	// Check if in city
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if room.GetFloor() != 0 {
		return "You can only manage your stall in the city."
	}

	// Parse the price (last argument)
	priceStr := c.Args[len(c.Args)-1]
	price, err := strconv.Atoi(priceStr)
	if err != nil || price <= 0 {
		return "Invalid price. Please specify a positive number.\nUsage: stall add <item> <price>"
	}

	// Get the item name (everything except "add" and price)
	itemName := strings.Join(c.Args[1:len(c.Args)-1], " ")
	if itemName == "" {
		return "Usage: stall add <item> <price>\nExample: stall add rusty sword 50"
	}

	// Find the item in player's inventory
	item, found := p.FindItem(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your inventory.", itemName)
	}

	// Remove from inventory and add to stall
	removedItem, removed := p.RemoveItem(item.Name)
	if !removed {
		return "Something went wrong trying to add that item to your stall."
	}

	p.AddToStall(removedItem, price)

	logger.Debug("Item added to stall",
		"player", p.GetName(),
		"item", removedItem.Name,
		"price", price)

	result := fmt.Sprintf("You add %s to your stall for %d gold.", removedItem.Name, price)
	if !p.IsStallOpen() {
		result += "\nNote: Your stall is not open. Use 'stall open' to start selling."
	}
	return result
}

// executeStallRemove removes an item from the player's stall
func executeStallRemove(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: stall remove <item>"
	}

	// Check if in city
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if room.GetFloor() != 0 {
		return "You can only manage your stall in the city."
	}

	itemName := strings.Join(c.Args[1:], " ")

	// Find and remove from stall
	stallItem, found := p.RemoveFromStall(itemName)
	if !found {
		return fmt.Sprintf("You don't have '%s' in your stall.", itemName)
	}

	// Add back to inventory
	p.AddItem(stallItem.Item)

	logger.Debug("Item removed from stall",
		"player", p.GetName(),
		"item", stallItem.Item.Name)

	return fmt.Sprintf("You remove %s from your stall and return it to your inventory.", stallItem.Item.Name)
}

// executeStallList shows the player's stall inventory
func executeStallList(c *Command, p PlayerInterface) string {
	stallItems := p.GetStallInventory()

	var status string
	if p.IsStallOpen() {
		status = "OPEN"
	} else {
		status = "CLOSED"
	}

	result := fmt.Sprintf("\n=== Your Stall [%s] ===\n", status)

	if len(stallItems) == 0 {
		result += "Your stall is empty.\n"
		result += "\nUse 'stall add <item> <price>' to add items for sale."
		return result
	}

	result += "\nItems for sale:\n"
	for _, stallItem := range stallItems {
		result += fmt.Sprintf("  %-25s %5d gold\n", stallItem.Item.Name, stallItem.Price)
	}

	result += fmt.Sprintf("\nTotal items: %d\n", len(stallItems))

	if !p.IsStallOpen() {
		result += "\nYour stall is closed. Use 'stall open' to start selling."
	}

	return result
}

// executeBrowse shows another player's stall
func executeBrowse(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Browse whose stall? Usage: browse <player>"); err != nil {
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

	targetName := c.GetTargetName()

	// Find the target player
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' is not online.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Check if target is in the same room
	targetRoomIface := target.GetCurrentRoom()
	targetRoom, ok := targetRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if targetRoom.GetID() != room.GetID() {
		return fmt.Sprintf("%s is not here.", target.GetName())
	}

	// Check if stall is open
	if !target.IsStallOpen() {
		return fmt.Sprintf("%s does not have a stall open.", target.GetName())
	}

	stallItems := target.GetStallInventory()
	if len(stallItems) == 0 {
		return fmt.Sprintf("%s's stall is empty.", target.GetName())
	}

	result := fmt.Sprintf("\n=== %s's Stall ===\n", target.GetName())
	result += fmt.Sprintf("Your gold: %d\n\n", p.GetGold())
	result += "Items for sale:\n"

	for _, stallItem := range stallItems {
		result += fmt.Sprintf("  %-25s %5d gold - %s\n", stallItem.Item.Name, stallItem.Price, stallItem.Item.Description)
	}

	result += fmt.Sprintf("\nTo purchase: purchase <item> from %s", target.GetName())

	return result
}

// executePurchase buys an item from another player's stall
func executePurchase(c *Command, p PlayerInterface) string {
	// Parse: purchase <item> from <player>
	if len(c.Args) < 3 {
		return "Usage: purchase <item> from <player>\nExample: purchase sword from Bob"
	}

	// Find "from" in the args
	fromIdx := -1
	for i, arg := range c.Args {
		if strings.ToLower(arg) == "from" {
			fromIdx = i
			break
		}
	}

	if fromIdx == -1 || fromIdx == 0 || fromIdx == len(c.Args)-1 {
		return "Usage: purchase <item> from <player>\nExample: purchase sword from Bob"
	}

	itemName := strings.Join(c.Args[:fromIdx], " ")
	targetName := strings.Join(c.Args[fromIdx+1:], " ")

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

	// Find the target player
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' is not online.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Can't buy from yourself
	if target.GetName() == p.GetName() {
		return "You can't buy from your own stall. Use 'stall remove' to take items back."
	}

	// Check if target is in the same room
	targetRoomIface := target.GetCurrentRoom()
	targetRoom, ok := targetRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if targetRoom.GetID() != room.GetID() {
		return fmt.Sprintf("%s is not here.", target.GetName())
	}

	// Check if stall is open
	if !target.IsStallOpen() {
		return fmt.Sprintf("%s does not have a stall open.", target.GetName())
	}

	// Find the item in the target's stall
	stallItem, found := target.FindInStall(itemName)
	if !found {
		return fmt.Sprintf("%s doesn't have '%s' in their stall.", target.GetName(), itemName)
	}

	// Check if buyer has enough gold
	if p.GetGold() < stallItem.Price {
		return fmt.Sprintf("You don't have enough gold. The %s costs %d gold, but you only have %d.",
			stallItem.Item.Name, stallItem.Price, p.GetGold())
	}

	// Check if buyer can carry the item
	if !p.CanCarry(stallItem.Item) {
		return "You can't carry any more weight."
	}

	// Execute the transaction
	// Remove from seller's stall
	removedStallItem, removed := target.RemoveFromStall(stallItem.Item.Name)
	if !removed {
		return "Something went wrong with the purchase."
	}

	// Transfer gold
	p.SpendGold(removedStallItem.Price)
	target.AddGold(removedStallItem.Price)

	// Transfer item
	p.AddItem(removedStallItem.Item)

	// Notify seller
	target.SendMessage(fmt.Sprintf("%s has purchased %s from your stall for %d gold.\n",
		p.GetName(), removedStallItem.Item.Name, removedStallItem.Price))

	logger.Debug("Stall purchase",
		"buyer", p.GetName(),
		"seller", target.GetName(),
		"item", removedStallItem.Item.Name,
		"price", removedStallItem.Price)

	return fmt.Sprintf("You purchase %s from %s for %d gold.\nGold remaining: %d",
		removedStallItem.Item.Name, target.GetName(), removedStallItem.Price, p.GetGold())
}
