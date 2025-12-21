package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// towerRoomIDPattern matches tower room IDs like "elf_f5_r10_7" or "unified_f1_r5_3"
// Captures the tower ID prefix before "_f"
var towerRoomIDPattern = regexp.MustCompile(`^([a-z]+)_f\d+_`)

// getTowerFromRoomID extracts the tower ID from a room ID.
// Tower room IDs follow the pattern: {tower}_f{floor}_r{x}_{y} (e.g., "elf_f5_r10_7")
// Returns the tower ID (e.g., "elf", "human", "unified") or empty string for non-tower rooms.
func getTowerFromRoomID(roomID string) string {
	matches := towerRoomIDPattern.FindStringSubmatch(roomID)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// executeLook handles looking at the room or examining specific objects
func executeLook(c *Command, p PlayerInterface) string {
	// If no arguments, look at the room
	if len(c.Args) == 0 {
		roomIface := p.GetCurrentRoom()
		room, ok := roomIface.(RoomInterface)
		if !ok {
			return "Internal error: invalid room type"
		}

		// Get unique items the player already owns (to filter from display)
		ownedUniqueIDs := p.GetOwnedUniqueItemIDs()

		// Get time-appropriate description
		serverIface := p.GetServer()
		server, ok := serverIface.(ServerInterface)
		if !ok {
			// Fallback to filtered description if server not available
			return room.GetDescriptionForPlayerFiltered(p.GetName(), ownedUniqueIDs)
		}

		// Select description based on time of day
		baseDesc := room.GetBaseDescription()
		if server.IsDay() && room.GetDescriptionDay() != "" {
			baseDesc = room.GetDescriptionDay()
		} else if server.IsNight() && room.GetDescriptionNight() != "" {
			baseDesc = room.GetDescriptionNight()
		}

		// Build full description with time-based variant and item filtering
		desc := room.GetDescriptionForPlayerFilteredWithCustomDesc(p.GetName(), baseDesc, ownedUniqueIDs)

		// Append player stall information
		stallInfo := getPlayersWithStallsInRoom(p, server, room)
		if stallInfo != "" {
			desc += stallInfo
		}

		return desc
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

	// Handle chest/treasure lookups - the "treasure" feature represents an opened chest
	if targetLower == "chest" || targetLower == "treasure" || targetLower == "treasure chest" {
		if room.HasFeature("treasure") {
			return "An ornate treasure chest lies open, its lock broken and lid thrown back. Whatever riches it once held have been scattered across the floor. Look around for items to take."
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

	// Show if they have a stall open
	if target.IsStallOpen() {
		stallItems := target.GetStallInventory()
		sb.WriteString(fmt.Sprintf("\nThey have a stall open with %d item(s) for sale.\n", len(stallItems)))
		sb.WriteString("Use 'browse " + name + "' to see their wares.\n")
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
	var exitMsg string
	switch direction {
	case "enter":
		exitMsg = fmt.Sprintf("%s enters the labyrinth.\n", p.GetName())
	case "leave":
		exitMsg = fmt.Sprintf("%s leaves the labyrinth.\n", p.GetName())
	default:
		exitMsg = fmt.Sprintf("%s leaves %s.\n", p.GetName(), direction)
	}
	server.BroadcastToRoom(currentRoom.GetID(), exitMsg, p)

	// Move the player
	p.MoveTo(nextRoomIface)

	// Record movement in statistics
	p.RecordMove()

	logger.Debug("Player moved",
		"player", p.GetName(),
		"direction", direction,
		"from_room", currentRoom.GetID(),
		"to_room", nextRoom.GetID(),
		"to_floor", nextRoom.GetFloor())

	// Track highest floor reached in towers and tower runs for unkillable achievement
	floor := nextRoom.GetFloor()
	if floor > 0 {
		// Determine tower ID from room ID (not player's home tower)
		towerID := getTowerFromRoomID(nextRoom.GetID())
		if towerID == "" {
			towerID = p.GetHomeTowerString()
		}
		p.RecordFloorReached(towerID, floor)
		// Start or continue tower run tracking for unkillable achievement
		p.StartTowerRun(towerID)
	} else if floor == 0 {
		// End tower run when returning to city
		p.EndTowerRun()
		// Track city visits for achievement (floor 0 is the city)
		roomID := nextRoom.GetID()
		// Determine which city based on room ID prefix
		for _, cityID := range []string{"human", "elf", "dwarf", "gnome", "orc"} {
			if len(roomID) > len(cityID) && roomID[:len(cityID)+1] == cityID+"_" {
				p.RecordCityVisited(cityID)
				break
			}
		}
	}

	// Broadcast enter message to new room
	// Determine opposite direction for enter message
	oppositeDir := getOppositeDirection(direction)
	enterMsg := fmt.Sprintf("%s arrives from the %s.\n", p.GetName(), oppositeDir)
	server.BroadcastToRoom(nextRoom.GetID(), enterMsg, p)

	// Track floor portal discovery - if room has a portal, mark the floor as discovered
	if nextRoom.HasFeature("portal") {
		floorNum := nextRoom.GetFloor()
		// Determine tower ID from room ID (not player's home tower)
		portalTowerID := getTowerFromRoomID(nextRoom.GetID())
		if portalTowerID == "" {
			portalTowerID = p.GetHomeTowerString()
		}
		if !p.HasDiscoveredPortalInTowerByString(portalTowerID, floorNum) {
			p.DiscoverPortalInTowerByString(portalTowerID, floorNum)
			towerDisplayName := getTowerDisplayName(portalTowerID)
			logger.Debug("Portal discovered",
				"player", p.GetName(),
				"tower", portalTowerID,
				"floor", floorNum)
			p.SendMessage(fmt.Sprintf("\n*** You have discovered a portal on %s in %s! ***\n", getFloorDisplayName(floorNum), towerDisplayName))
		}
	}

	// Track labyrinth gate discovery - if entering a labyrinth gate room, discover that city's portal
	if nextRoom.HasFeature("labyrinth_entrance") {
		// Check if this is a labyrinth gate room by checking for the "gate" feature
		if nextRoom.HasFeature("gate") {
			// Get the tower manager to determine which city this gate leads to
			if towerMgr, ok := server.GetTowerManager().(*tower.TowerManager); ok {
				if lab := towerMgr.GetLabyrinth(); lab != nil {
					// Check if this room is a labyrinth gate
					cityID := lab.GetCityIDForGateRoom(nextRoom.GetID())
					if cityID != "" {
						// Track this gate visit for the Wanderer of the Ways title
						if p.VisitLabyrinthGate(cityID) {
							// First time visiting this gate - also track for achievement
							p.RecordCityVisited(cityID)
							visitedCount := len(p.GetVisitedLabyrinthGates())
							p.SendMessage(fmt.Sprintf("\n*** You have discovered the %s Gate! (%d/5 gates found) ***\n", getCityDisplayName(cityID), visitedCount))

							// Check if player has now visited all gates
							if p.HasVisitedAllLabyrinthGates() {
								// Track labyrinth completion for achievement
								p.RecordLabyrinthCompleted()
								title := "Wanderer of the Ways"
								if !p.HasEarnedTitle(title) {
									p.EarnTitle(title)
									p.SendMessage(fmt.Sprintf("\n================================================================================\n                    TITLE EARNED: %s\n\n  You have discovered all five city gates in the Great Labyrinth!\n  The paths between all cities are now known to you.\n================================================================================\n", title))
									// Announce to server
									server.BroadcastToAll(fmt.Sprintf("\n*** %s has earned the title: %s ***\n", p.GetName(), title))
								}
							}
						}

						// Also discover their floor 0 portal if it's a different city
						if cityID != p.GetHomeTowerString() {
							if !p.HasDiscoveredPortalInTowerByString(cityID, 0) {
								p.DiscoverPortalInTowerByString(cityID, 0)
								cityName := getCityDisplayName(cityID)
								logger.Debug("City portal discovered via labyrinth",
									"player", p.GetName(),
									"city", cityID)
								p.SendMessage(fmt.Sprintf("\n*** You can now use portals to travel to %s! ***\n", cityName))
							}
						}
					}
				}
			}
		}
	}

	// Update quest explore progress
	updateQuestExploreProgress(p, server, nextRoom)

	// Generate appropriate movement message based on direction
	var moveMsg string
	switch direction {
	case "enter":
		moveMsg = "You enter the labyrinth."
	case "leave":
		moveMsg = "You leave the labyrinth."
	default:
		moveMsg = fmt.Sprintf("You move %s.", direction)
	}

	return fmt.Sprintf("%s\n\n%s", moveMsg, nextRoom.GetDescriptionForPlayer(p.GetName()))
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
		"enter": "outside",
		"leave": "the labyrinth",
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

	// Determine current tower from room ID (not player's home tower)
	currentRoomID := room.GetID()
	currentFloor := room.GetFloor()
	currentTowerID := getTowerFromRoomID(currentRoomID)
	if currentTowerID == "" {
		// Fall back to home tower for city rooms or unknown formats
		currentTowerID = p.GetHomeTowerString()
	}

	// Check if unified tower is unlocked
	unifiedUnlocked := server.IsUnifiedTowerUnlocked()

	// Get tower manager for multi-tower operations
	towerMgrIface := server.GetTowerManager()
	towerMgr, hasTowerMgr := towerMgrIface.(*tower.TowerManager)

	// Get discovered floor portals for current tower (excluding current floor)
	discoveredPortals := p.GetDiscoveredPortalsInTowerByString(currentTowerID)
	availableFloors := make([]int, 0)
	for _, floor := range discoveredPortals {
		if floor != currentFloor {
			availableFloors = append(availableFloors, floor)
		}
	}

	// If no arguments, show available destinations
	if len(c.Args) == 0 {
		var sb strings.Builder
		sb.WriteString("The portal shimmers with arcane energy. Available destinations:\n\n")

		// Show current tower floors
		if len(availableFloors) > 0 {
			towerDisplayName := getTowerDisplayName(currentTowerID)
			sb.WriteString(fmt.Sprintf("== %s ==\n", towerDisplayName))
			for _, floor := range availableFloors {
				floorName := getFloorDisplayName(floor)
				sb.WriteString(fmt.Sprintf("  - %s (portal %d)\n", floorName, floor))
			}
		} else {
			sb.WriteString("You haven't discovered any other floors in this tower.\n")
		}

		// Show discovered cities from labyrinth exploration
		for _, racialTowerID := range tower.AllRacialTowers {
			towerIDStr := string(racialTowerID)
			if towerIDStr == currentTowerID {
				continue // Skip current tower, already shown above
			}
			// Check if player has discovered this city's floor 0
			if p.HasDiscoveredPortalInTowerByString(towerIDStr, 0) {
				cityName := getCityDisplayName(towerIDStr)
				sb.WriteString(fmt.Sprintf("\n== %s ==\n", cityName))
				discoveredFloors := p.GetDiscoveredPortalsInTowerByString(towerIDStr)
				for _, floor := range discoveredFloors {
					floorName := getFloorDisplayName(floor)
					sb.WriteString(fmt.Sprintf("  - %s (portal %s %d)\n", floorName, towerIDStr, floor))
				}
			}
		}

		// Show unified tower option if unlocked and not already in unified
		if unifiedUnlocked && currentTowerID != string(tower.TowerUnified) {
			sb.WriteString("\n== Infinity Spire (Unlocked!) ==\n")
			unifiedPortals := p.GetDiscoveredPortalsInTowerByString(string(tower.TowerUnified))
			if len(unifiedPortals) == 0 {
				sb.WriteString("  - Ground Floor (portal unified 0)\n")
			} else {
				for _, floor := range unifiedPortals {
					floorName := getUnifiedFloorDisplayName(floor)
					sb.WriteString(fmt.Sprintf("  - %s (portal unified %d)\n", floorName, floor))
				}
			}
		}

		// Show home tower option if in unified tower
		if currentTowerID == string(tower.TowerUnified) {
			homeTowerID := p.GetHomeTowerString()
			homeTowerName := getTowerDisplayName(homeTowerID)
			sb.WriteString(fmt.Sprintf("\n== %s (Home) ==\n", homeTowerName))
			homePortals := p.GetDiscoveredPortalsInTowerByString(homeTowerID)
			for _, floor := range homePortals {
				floorName := getFloorDisplayName(floor)
				sb.WriteString(fmt.Sprintf("  - %s (portal home %d)\n", floorName, floor))
			}
		}

		sb.WriteString("\nUsage: portal <floor number> | portal <race> <floor>")
		if unifiedUnlocked {
			sb.WriteString(" | portal unified <floor> | portal home <floor>")
		}
		return sb.String()
	}

	// Parse destination arguments
	destArg := strings.TrimSpace(strings.ToLower(c.Args[0]))
	destFloor := -1
	destTowerID := currentTowerID

	// Handle special destinations
	switch destArg {
	case "unified", "infinity", "spire":
		if !unifiedUnlocked {
			return "The Infinity Spire remains sealed. Defeat all five tower guardians to unlock it."
		}
		destTowerID = string(tower.TowerUnified)
		if len(c.Args) > 1 {
			_, err := fmt.Sscanf(c.Args[1], "%d", &destFloor)
			if err != nil {
				return "Invalid floor number. Usage: portal unified <floor>"
			}
		} else {
			destFloor = 0 // Default to ground floor
		}

	case "home":
		destTowerID = p.GetHomeTowerString()
		if len(c.Args) > 1 {
			_, err := fmt.Sscanf(c.Args[1], "%d", &destFloor)
			if err != nil {
				return "Invalid floor number. Usage: portal home <floor>"
			}
		} else {
			destFloor = 0 // Default to city
		}

	case "city", "town", "ground":
		destFloor = 0

	case "human", "elf", "dwarf", "gnome", "orc":
		// Cross-city travel via labyrinth discovery
		destTowerID = destArg
		if len(c.Args) > 1 {
			_, err := fmt.Sscanf(c.Args[1], "%d", &destFloor)
			if err != nil {
				return fmt.Sprintf("Invalid floor number. Usage: portal %s <floor>", destArg)
			}
		} else {
			destFloor = 0 // Default to city
		}

	default:
		// Try to parse as number
		_, err := fmt.Sscanf(destArg, "%d", &destFloor)
		if err != nil {
			return fmt.Sprintf("Invalid destination: '%s'. Type 'portal' to see available destinations.", destArg)
		}
	}

	// Check if player has discovered this floor (grant floor 0 access for unified if unlocked)
	if destTowerID == string(tower.TowerUnified) && destFloor == 0 && unifiedUnlocked {
		// Auto-discover unified tower floor 0 when first accessing it
		if !p.HasDiscoveredPortalInTowerByString(destTowerID, 0) {
			p.DiscoverPortalInTowerByString(destTowerID, 0)
		}
	}

	if !p.HasDiscoveredPortalInTowerByString(destTowerID, destFloor) {
		towerName := getTowerDisplayName(destTowerID)
		return fmt.Sprintf("You haven't discovered a portal on %s floor %d. Type 'portal' to see available destinations.", towerName, destFloor)
	}

	// Can't portal to current location
	if destFloor == currentFloor && destTowerID == currentTowerID {
		return "You're already here!"
	}

	// Get the destination room
	var destRoom *world.Room
	if hasTowerMgr && destTowerID != currentTowerID {
		// Cross-tower travel
		destRoom = towerMgr.GetFloorPortalRoom(tower.TowerID(destTowerID), destFloor)
	} else if destTowerID == currentTowerID {
		// Same tower travel
		destRoom = w.GetFloorPortalRoom(destFloor)
	} else {
		return "Unable to access the destination tower."
	}

	if destRoom == nil {
		return fmt.Sprintf("Floor %d doesn't have a portal room.", destFloor)
	}

	destRoomID := destRoom.GetID()

	// Broadcast departure
	server.BroadcastToRoom(currentRoomID, fmt.Sprintf("%s steps through the portal and vanishes!\n", p.GetName()), p)

	// Move the player
	p.MoveTo(destRoom)

	// Track portal usage for achievement
	p.RecordPortalUsed()

	// Broadcast arrival
	server.BroadcastToRoom(destRoomID, fmt.Sprintf("%s emerges from the portal in a flash of light!\n", p.GetName()), p)

	// Generate arrival message
	var arrivalMsg string
	if destTowerID == string(tower.TowerUnified) {
		arrivalMsg = fmt.Sprintf("You step through the shimmering portal into the Infinity Spire...\n\nYou emerge on %s!\n\n%s", getUnifiedFloorDisplayName(destFloor), destRoom.GetDescriptionForPlayer(p.GetName()))
	} else {
		arrivalMsg = fmt.Sprintf("You step through the shimmering portal...\n\nYou emerge on %s!\n\n%s", getFloorDisplayName(destFloor), destRoom.GetDescriptionForPlayer(p.GetName()))
	}

	return arrivalMsg
}

// getFloorDisplayName returns a human-readable floor name
func getFloorDisplayName(floor int) string {
	if floor == 0 {
		return "the City"
	}
	return fmt.Sprintf("Floor %d", floor)
}

// getUnifiedFloorDisplayName returns display name for unified tower floors
func getUnifiedFloorDisplayName(floor int) string {
	if floor == 0 {
		return "the Spire Base"
	}
	if floor == 100 {
		return "The Architect's Domain (Floor 100)"
	}
	if floor == 25 || floor == 50 || floor == 75 {
		return fmt.Sprintf("Sub-Boss Chamber (Floor %d)", floor)
	}
	return fmt.Sprintf("Floor %d", floor)
}

// getTowerDisplayName returns a human-readable tower name
func getTowerDisplayName(towerID string) string {
	switch tower.TowerID(towerID) {
	case tower.TowerHuman:
		return "Aetherspire (Human)"
	case tower.TowerElf:
		return "Sylvan Heights (Elf)"
	case tower.TowerDwarf:
		return "Khazad-Karn Depths (Dwarf)"
	case tower.TowerGnome:
		return "Mechanical Spire (Gnome)"
	case tower.TowerOrc:
		return "Eternal Battlefield (Orc)"
	case tower.TowerUnified:
		return "Infinity Spire"
	default:
		return "Unknown Tower"
	}
}

// getCityDisplayName returns a human-readable city name for portal display
func getCityDisplayName(towerID string) string {
	switch tower.TowerID(towerID) {
	case tower.TowerHuman:
		return "Ironhaven (Human)"
	case tower.TowerElf:
		return "Sylvanthal (Elf)"
	case tower.TowerDwarf:
		return "Khazad-Karn (Dwarf)"
	case tower.TowerGnome:
		return "Cogsworth (Gnome)"
	case tower.TowerOrc:
		return "Skullgar (Orc)"
	default:
		return "Unknown City"
	}
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

// getPlayersWithStallsInRoom returns information about players with open stalls in the room
func getPlayersWithStallsInRoom(currentPlayer PlayerInterface, server ServerInterface, room RoomInterface) string {
	onlinePlayers := server.GetOnlinePlayers()
	var stallOwners []string

	for _, playerName := range onlinePlayers {
		// Skip the current player
		if playerName == currentPlayer.GetName() {
			continue
		}

		// Find the player object
		playerIface := server.FindPlayer(playerName)
		if playerIface == nil {
			continue
		}

		player, ok := playerIface.(PlayerInterface)
		if !ok {
			continue
		}

		// Check if in the same room
		playerRoomIface := player.GetCurrentRoom()
		playerRoom, ok := playerRoomIface.(RoomInterface)
		if !ok {
			continue
		}

		if playerRoom.GetID() != room.GetID() {
			continue
		}

		// Check if stall is open
		if player.IsStallOpen() {
			itemCount := len(player.GetStallInventory())
			stallOwners = append(stallOwners, fmt.Sprintf("%s (%d items)", playerName, itemCount))
		}
	}

	if len(stallOwners) == 0 {
		return ""
	}

	return "\nPlayer stalls: " + strings.Join(stallOwners, ", ") + "\n"
}
