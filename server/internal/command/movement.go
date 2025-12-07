package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// executeLook handles looking at the room or examining specific objects
func executeLook(c *Command, p PlayerInterface) string {
	// If no arguments, look at the room
	if len(c.Args) == 0 {
		roomIface := p.GetCurrentRoom()
		room, ok := roomIface.(RoomInterface)
		if !ok {
			return "Internal error: invalid room type"
		}

		// Get time-appropriate description
		serverIface := p.GetServer()
		server, ok := serverIface.(ServerInterface)
		if !ok {
			// Fallback to default description if server not available
			return room.GetDescriptionForPlayer(p.GetName())
		}

		// Select description based on time of day
		baseDesc := room.GetBaseDescription()
		if server.IsDay() && room.GetDescriptionDay() != "" {
			baseDesc = room.GetDescriptionDay()
		} else if server.IsNight() && room.GetDescriptionNight() != "" {
			baseDesc = room.GetDescriptionNight()
		}

		// Build full description with time-based variant
		return room.GetDescriptionForPlayerWithCustomDesc(p.GetName(), baseDesc)
	}

	// Otherwise, examine a specific object
	targetName := c.GetItemName()

	// Get current room
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// First, check if it's an item in the room
	item, foundInRoom := room.FindItem(targetName)
	if foundInRoom {
		return fmt.Sprintf("%s\n%s", item.Name, item.Description)
	}

	// Next, check if it's an item in player's inventory
	item, foundInInventory := p.FindItem(targetName)
	if foundInInventory {
		return fmt.Sprintf("%s\n%s", item.Name, item.Description)
	}

	// Check if it's a room feature
	targetLower := strings.ToLower(targetName)

	// Handle stairs/stairway lookups - match either stairs_up or stairs_down
	if targetLower == "stairs" || targetLower == "stairway" {
		if room.HasFeature("stairs_up") || room.HasFeature("stairs_down") {
			return "A spiral staircase winds through the tower, its ancient stones worn smooth by countless adventurers. Who knows what awaits beyond?"
		}
	}

	if room.HasFeature(targetLower) {
		switch targetLower {
		case "altar":
			return "A sacred altar carved from white marble. It radiates a gentle warmth and divine energy. You could pray here to seek healing."
		case "portal":
			return "A shimmering portal of swirling blue and silver energy. It offers travel to tower floors you have discovered. Type 'portal' to see available destinations."
		case "workbench":
			return "A sturdy wooden workbench with various tools hanging nearby. Perfect for leatherworking and other crafts. Type 'craft' to see available recipes."
		case "forge":
			return "A blazing forge, hot enough to work metal into weapons and armor. Type 'craft' to see available blacksmithing recipes."
		case "alchemy_lab":
			return "An elaborate alchemy laboratory with bubbling cauldrons, glass vials, and brass tubes. The air smells of exotic herbs and strange chemicals. Type 'craft' to see available alchemy recipes."
		case "enchanting_table":
			return "A mystical table covered in glowing runes and arcane symbols. Magical energy crackles around its surface, ready to imbue objects with power. Type 'craft' to see available enchanting recipes."
		default:
			return fmt.Sprintf("You see a %s here.", targetName)
		}
	}

	// Finally, check if it's another player in the room
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if ok {
		targetPlayerIface := server.FindPlayer(targetName)
		if targetPlayerIface != nil {
			targetPlayer, ok := targetPlayerIface.(PlayerInterface)
			if ok {
				// Check if the target player is in the same room
				targetRoomIface := targetPlayer.GetCurrentRoom()
				targetRoom, ok := targetRoomIface.(RoomInterface)
				if ok && targetRoom.GetID() == room.GetID() {
					return formatPlayerDescription(targetPlayer)
				}
			}
		}
	}

	return fmt.Sprintf("You don't see '%s' here.", targetName)
}

// formatPlayerDescription returns a description of another player
func formatPlayerDescription(target PlayerInterface) string {
	name := target.GetName()
	level := target.GetLevel()
	raceName := target.GetRaceName()
	className := target.GetPrimaryClassName()
	title := target.GetActiveTitle()

	// Build basic description
	var sb strings.Builder
	if title != "" {
		sb.WriteString(fmt.Sprintf("%s (%s), a level %d %s %s.\n", name, title, level, raceName, className))
	} else {
		sb.WriteString(fmt.Sprintf("%s, a level %d %s %s.\n", name, level, raceName, className))
	}

	// Show equipment summary
	equipment := target.GetEquipment()
	if len(equipment) > 0 {
		sb.WriteString("\nEquipped:\n")
		for slot, item := range equipment {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", slot.String(), item.Name))
		}
	}

	// Show health status (approximate)
	healthPercent := float64(target.GetHealth()) / float64(target.GetMaxHealth()) * 100
	var healthStatus string
	switch {
	case healthPercent >= 100:
		healthStatus = "in perfect health"
	case healthPercent >= 75:
		healthStatus = "slightly wounded"
	case healthPercent >= 50:
		healthStatus = "moderately wounded"
	case healthPercent >= 25:
		healthStatus = "heavily wounded"
	default:
		healthStatus = "near death"
	}
	sb.WriteString(fmt.Sprintf("\nThey appear to be %s.", healthStatus))

	return sb.String()
}

// executeMove handles the generic "go <direction>" command
func executeMove(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Go where? Specify a direction (north, south, east, west, up, down)"); err != nil {
		return err.Error()
	}
	direction := strings.ToLower(c.Args[0])
	return executeMoveDirection(c, p, direction)
}

// executeMoveDirection handles movement in a specific direction
func executeMoveDirection(c *Command, p PlayerInterface, direction string) string {
	// Check if player can move (not sleeping)
	currentState := p.GetState()
	if currentState == "sleeping" {
		return "You can't move while sleeping! Wake up first."
	}

	currentRoomIface := p.GetCurrentRoom()
	currentRoom, ok := currentRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasts and floor generation
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Check if exit is locked
	if currentRoom.IsExitLocked(direction) {
		keyID := currentRoom.GetExitKeyRequired(direction)
		return fmt.Sprintf("The way %s is locked. You need a key to unlock it. (Requires: %s)", direction, keyID)
	}

	nextRoomIface := currentRoom.GetExit(direction)

	// Handle stairs - if going up from a stairs room and no exit exists, generate next floor
	if nextRoomIface == nil && direction == "up" && currentRoom.HasFeature("stairs_up") {
		currentFloor := currentRoom.GetFloor()
		nextFloorRoom, err := server.GenerateNextFloor(currentFloor)
		if err != nil {
			return fmt.Sprintf("The stairs seem to lead nowhere... (%v)", err)
		}
		if nextFloorRoom == nil {
			return "The stairs seem to lead nowhere."
		}
		nextRoomIface = nextFloorRoom
	}

	if nextRoomIface == nil {
		return fmt.Sprintf("You can't go %s from here.", direction)
	}

	nextRoom, ok := nextRoomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Broadcast exit message to current room
	exitMsg := fmt.Sprintf("%s leaves %s.\n", p.GetName(), direction)
	server.BroadcastToRoom(currentRoom.GetID(), exitMsg, p)

	// Move the player
	p.MoveTo(nextRoomIface)

	logger.Debug("Player moved",
		"player", p.GetName(),
		"direction", direction,
		"from_room", currentRoom.GetID(),
		"to_room", nextRoom.GetID(),
		"to_floor", nextRoom.GetFloor())

	// Broadcast enter message to new room
	// Determine opposite direction for enter message
	oppositeDir := getOppositeDirection(direction)
	enterMsg := fmt.Sprintf("%s arrives from the %s.\n", p.GetName(), oppositeDir)
	server.BroadcastToRoom(nextRoom.GetID(), enterMsg, p)

	// Track floor portal discovery - if room has a portal, mark the floor as discovered
	if nextRoom.HasFeature("portal") {
		floorNum := nextRoom.GetFloor()
		if !p.HasDiscoveredPortal(floorNum) {
			p.DiscoverPortal(floorNum)
			logger.Debug("Portal discovered",
				"player", p.GetName(),
				"floor", floorNum)
			p.SendMessage(fmt.Sprintf("\n*** You have discovered a portal on %s! ***\n", getFloorDisplayName(floorNum)))
		}
	}

	// Update quest explore progress
	updateQuestExploreProgress(p, server, nextRoom)

	return fmt.Sprintf("You move %s.\n\n%s", direction, nextRoom.GetDescriptionForPlayer(p.GetName()))
}

// Direction wrapper functions for the command registry
func executeMoveNorth(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "north")
}

func executeMoveSouth(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "south")
}

func executeMoveEast(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "east")
}

func executeMoveWest(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "west")
}

func executeMoveUp(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "up")
}

func executeMoveDown(c *Command, p PlayerInterface) string {
	return executeMoveDirection(c, p, "down")
}

// getOppositeDirection returns the opposite direction for enter messages
func getOppositeDirection(direction string) string {
	opposites := map[string]string{
		"north": "south",
		"south": "north",
		"east":  "west",
		"west":  "east",
		"up":    "above",
		"down":  "below",
	}
	if opposite, ok := opposites[direction]; ok {
		return opposite
	}
	return "somewhere"
}

// executeExits shows available exits from the current room
func executeExits(c *Command, p PlayerInterface) string {
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	exits := room.GetExits()
	if len(exits) == 0 {
		return "There are no obvious exits."
	}

	result := "Obvious exits:\n"
	for direction, roomName := range exits {
		result += fmt.Sprintf("  %-6s - %s\n", direction, roomName)
	}

	return result
}

// executePortal allows players to fast travel between discovered tower floors
func executePortal(c *Command, p PlayerInterface) string {
	// Check player state - can't portal while fighting or sleeping
	state := p.GetState()
	if state == "Fighting" {
		return "You can't use the portal while fighting!"
	}
	if state == "Sleeping" {
		return "You are asleep and can't use the portal. Wake up first."
	}

	// Get current room and check for portal feature
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return "You are nowhere?"
	}

	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	if !room.HasFeature("portal") {
		return "There is no portal here."
	}

	// Get the server for world access
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Get discovered floor portals (excluding current floor)
	discoveredPortals := p.GetDiscoveredPortals()
	currentFloor := room.GetFloor()
	currentRoomID := room.GetID()

	// Filter out current floor
	availableFloors := make([]int, 0)
	for _, floor := range discoveredPortals {
		if floor != currentFloor {
			availableFloors = append(availableFloors, floor)
		}
	}

	// If no arguments, show available destinations
	if len(c.Args) == 0 {
		if len(availableFloors) == 0 {
			return "The portal shimmers before you, but you haven't discovered any other floors to travel to.\nClimb the tower and find stairway landings with portals!"
		}

		var sb strings.Builder
		sb.WriteString("The portal shimmers with arcane energy. Available destinations:\n\n")
		for _, floor := range availableFloors {
			floorName := getFloorDisplayName(floor)
			sb.WriteString(fmt.Sprintf("  - %s (portal %d)\n", floorName, floor))
		}
		sb.WriteString("\nUsage: portal <floor number>")
		return sb.String()
	}

	// Parse floor number from argument
	destArg := strings.TrimSpace(c.Args[0])
	destFloor := -1

	// Handle special names
	switch strings.ToLower(destArg) {
	case "city", "town", "ground":
		destFloor = 0
	default:
		// Try to parse as number
		_, err := fmt.Sscanf(destArg, "%d", &destFloor)
		if err != nil {
			return fmt.Sprintf("Invalid floor number: '%s'. Type 'portal' to see available destinations.", destArg)
		}
	}

	// Check if player has discovered this floor
	if !p.HasDiscoveredPortal(destFloor) {
		return fmt.Sprintf("You haven't discovered a portal on floor %d. Type 'portal' to see available destinations.", destFloor)
	}

	// Can't portal to current floor
	if destFloor == currentFloor {
		return "You're already on this floor!"
	}

	// Get the destination room
	destRoom := w.GetFloorPortalRoom(destFloor)
	if destRoom == nil {
		return fmt.Sprintf("Floor %d doesn't have a portal room.", destFloor)
	}

	destRoomID := destRoom.GetID()

	// Broadcast departure
	server.BroadcastToRoom(currentRoomID, fmt.Sprintf("%s steps through the portal and vanishes!\n", p.GetName()), p)

	// Move the player
	p.MoveTo(destRoom)

	// Broadcast arrival
	server.BroadcastToRoom(destRoomID, fmt.Sprintf("%s emerges from the portal in a flash of light!\n", p.GetName()), p)

	return fmt.Sprintf("You step through the shimmering portal...\n\nYou emerge on %s!\n\n%s", getFloorDisplayName(destFloor), destRoom.GetDescriptionForPlayer(p.GetName()))
}

// getFloorDisplayName returns a human-readable floor name
func getFloorDisplayName(floor int) string {
	if floor == 0 {
		return "the City"
	}
	return fmt.Sprintf("Floor %d", floor)
}

// getPortalCommandName returns a short name for use in portal commands
func getPortalCommandName(roomID string) string {
	switch roomID {
	case "city_square":
		return "town"
	default:
		if strings.HasPrefix(roomID, "gen_") {
			return "frontier"
		}
		return roomID
	}
}

// updateQuestExploreProgress updates explore quest progress when a player enters a room
func updateQuestExploreProgress(p PlayerInterface, server ServerInterface, room RoomInterface) {
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return
	}

	questLog := p.GetQuestLog()
	if questLog == nil {
		return
	}

	roomID := room.GetID()

	// Check all active quests for explore objectives matching this room
	for _, questID := range questLog.GetActiveQuests() {
		questDef, exists := questRegistry.GetQuest(questID)
		if !exists {
			continue
		}

		// Check each explore objective to see if this room matches
		for i, obj := range questDef.Objectives {
			if obj.Type != quest.QuestTypeExplore {
				continue
			}

			// Check if room matches the objective target (exact match or symbolic target)
			if !roomMatchesExploreTarget(room, obj.Target) {
				continue
			}

			if questLog.UpdateExploreProgressForQuest(questID, questDef, obj.Target) {
				// Notify player of quest progress
				progress, _ := questLog.GetQuestProgress(questID)
				current := progress.Objectives[i].Current
				targetName := obj.TargetName
				if targetName == "" {
					targetName = obj.Target
				}
				p.SendMessage(fmt.Sprintf("\nQuest progress: Explored %s - %d/%d\n", targetName, current, obj.Required))
			}
		}
	}

	// Also check for exact room ID matches (for city rooms like "temple")
	for _, questID := range questLog.GetActiveQuests() {
		questDef, exists := questRegistry.GetQuest(questID)
		if !exists {
			continue
		}

		if questLog.UpdateExploreProgressForQuest(questID, questDef, roomID) {
			// Notify player of quest progress
			progress, _ := questLog.GetQuestProgress(questID)
			for i, obj := range questDef.Objectives {
				if obj.Type == quest.QuestTypeExplore && strings.EqualFold(obj.Target, roomID) {
					current := progress.Objectives[i].Current
					targetName := obj.TargetName
					if targetName == "" {
						targetName = obj.Target
					}
					p.SendMessage(fmt.Sprintf("\nQuest progress: Explored %s - %d/%d\n", targetName, current, obj.Required))
				}
			}
		}
	}
}

// roomMatchesExploreTarget checks if a room matches a symbolic explore target
// Symbolic targets:
//   - gen_fN_portal: portal room on floor N (e.g., gen_f1_portal, gen_f5_portal)
//   - gen_fN_boss: boss room on floor N (e.g., gen_f10_boss)
func roomMatchesExploreTarget(room RoomInterface, target string) bool {
	// Check for generated floor portal targets (gen_f1_portal, gen_f5_portal, etc.)
	if strings.HasPrefix(target, "gen_f") && strings.HasSuffix(target, "_portal") {
		// Extract floor number from target
		floorStr := strings.TrimPrefix(target, "gen_f")
		floorStr = strings.TrimSuffix(floorStr, "_portal")
		var targetFloor int
		if _, err := fmt.Sscanf(floorStr, "%d", &targetFloor); err != nil {
			return false
		}
		// Check if room is on the target floor and has portal feature
		return room.GetFloor() == targetFloor && room.HasFeature("portal")
	}

	// Check for generated floor boss targets (gen_f10_boss, etc.)
	if strings.HasPrefix(target, "gen_f") && strings.HasSuffix(target, "_boss") {
		// Extract floor number from target
		floorStr := strings.TrimPrefix(target, "gen_f")
		floorStr = strings.TrimSuffix(floorStr, "_boss")
		var targetFloor int
		if _, err := fmt.Sscanf(floorStr, "%d", &targetFloor); err != nil {
			return false
		}
		// Check if room is on the target floor and has boss feature
		return room.GetFloor() == targetFloor && room.HasFeature("boss")
	}

	return false
}
