package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// executeAdmin handles all admin subcommands
func executeAdmin(c *Command, p PlayerInterface) string {
	// Non-admins see "Unknown command" to hide existence of admin commands
	if !p.IsAdmin() {
		return fmt.Sprintf("Unknown command: %s. Type 'help' for available commands.", c.Name)
	}

	if len(c.Args) == 0 {
		return executeAdminHelp(c, p)
	}

	subcommand := strings.ToLower(c.Args[0])
	switch subcommand {
	case "help":
		return executeAdminHelp(c, p)
	case "promote":
		return executeAdminPromote(c, p)
	case "demote":
		return executeAdminDemote(c, p)
	case "ban":
		return executeAdminBan(c, p)
	case "unban":
		return executeAdminUnban(c, p)
	case "kick":
		return executeAdminKick(c, p)
	case "announce":
		return executeAdminAnnounce(c, p)
	case "teleport", "tp":
		return executeAdminTeleport(c, p)
	case "goto":
		return executeAdminGoto(c, p)
	case "stats":
		return executeAdminStats(c, p)
	case "players":
		return executeAdminPlayers(c, p)
	default:
		return fmt.Sprintf("Unknown admin command: %s. Type 'admin help' for commands.", subcommand)
	}
}

// executeAdminHelp shows admin command help
func executeAdminHelp(c *Command, p PlayerInterface) string {
	return `
Admin Commands
==============

Player Management:
  admin promote <player>     - Grant admin privileges to a player
  admin demote <player>      - Remove admin privileges from a player
  admin ban <player> [reason] - Ban a player's account
  admin unban <username>     - Unban an account by username
  admin kick <player> [reason] - Disconnect a player

Communication:
  admin announce <message>   - Broadcast to all players

Teleportation:
  admin teleport <player> <room> - Move a player to a room
  admin tp <player> <room>   - Alias for teleport
  admin goto <room>          - Teleport yourself to a room

Information:
  admin stats               - Show server statistics
  admin players             - List all online players with details
  admin help                - Show this help message
`
}

// executeAdminPromote grants admin privileges to a player
func executeAdminPromote(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin promote <player_name>"
	}

	targetName := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Find the target player (online)
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		// Try to find by account username (offline player)
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found (must be online or use account username).", targetName)
		}

		if account.IsAdmin {
			return fmt.Sprintf("Account '%s' is already an admin.", account.Username)
		}

		if err := db.SetAdmin(account.ID, true); err != nil {
			return fmt.Sprintf("Failed to promote account: %v", err)
		}

		// Log admin action
		logger.Always("ADMIN_ACTION",
			"action", "promote",
			"admin", p.GetName(),
			"target_account", account.Username,
			"target_online", false)

		return fmt.Sprintf("Account '%s' has been promoted to admin.", account.Username)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	if target.IsAdmin() {
		return fmt.Sprintf("%s is already an admin.", target.GetName())
	}

	// Promote the account
	if err := db.SetAdmin(target.GetAccountID(), true); err != nil {
		return fmt.Sprintf("Failed to promote: %v", err)
	}

	// Notify the target
	target.SendMessage("\n*** You have been granted admin privileges! ***\n")

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "promote",
		"admin", p.GetName(),
		"target", target.GetName(),
		"target_account_id", target.GetAccountID())

	return fmt.Sprintf("%s has been promoted to admin.", target.GetName())
}

// executeAdminDemote removes admin privileges from a player
func executeAdminDemote(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin demote <player_name>"
	}

	targetName := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Check if this would leave no admins
	admins, err := db.GetAllAdmins()
	if err != nil {
		return fmt.Sprintf("Failed to check admin count: %v", err)
	}

	if len(admins) <= 1 {
		return "Cannot demote: this would leave the server with no admins."
	}

	// Find the target player (online)
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		// Try to find by account username (offline player)
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found (must be online or use account username).", targetName)
		}

		if !account.IsAdmin {
			return fmt.Sprintf("Account '%s' is not an admin.", account.Username)
		}

		if err := db.SetAdmin(account.ID, false); err != nil {
			return fmt.Sprintf("Failed to demote account: %v", err)
		}

		// Log admin action
		logger.Always("ADMIN_ACTION",
			"action", "demote",
			"admin", p.GetName(),
			"target_account", account.Username,
			"target_online", false)

		return fmt.Sprintf("Account '%s' has been demoted from admin.", account.Username)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	if !target.IsAdmin() {
		return fmt.Sprintf("%s is not an admin.", target.GetName())
	}

	// Demote the account
	if err := db.SetAdmin(target.GetAccountID(), false); err != nil {
		return fmt.Sprintf("Failed to demote: %v", err)
	}

	// Notify the target
	target.SendMessage("\n*** Your admin privileges have been revoked. ***\n")

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "demote",
		"admin", p.GetName(),
		"target", target.GetName(),
		"target_account_id", target.GetAccountID())

	return fmt.Sprintf("%s has been demoted from admin.", target.GetName())
}

// executeAdminBan bans a player's account
func executeAdminBan(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin ban <player_name> [reason]"
	}

	targetName := c.Args[1]
	reason := ""
	if len(c.Args) > 2 {
		reason = strings.Join(c.Args[2:], " ")
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// First try to find online player
	var accountID int64
	var accountUsername string

	targetIface := server.FindPlayer(targetName)
	if targetIface != nil {
		target, ok := targetIface.(PlayerInterface)
		if !ok {
			return "Internal error: invalid player type"
		}

		// Don't allow banning admins
		if target.IsAdmin() {
			return "Cannot ban an admin account."
		}

		accountID = target.GetAccountID()
		accountUsername = target.GetName()

		// Kick the player
		kickMsg := "\n*** YOU HAVE BEEN BANNED"
		if reason != "" {
			kickMsg += ": " + reason
		}
		kickMsg += " ***\n"
		target.SendMessage(kickMsg)
		target.Disconnect()
	} else {
		// Try to find by account username
		account, err := db.GetAccountByUsername(targetName)
		if err != nil {
			return fmt.Sprintf("Player '%s' not found.", targetName)
		}

		if account.IsAdmin {
			return "Cannot ban an admin account."
		}

		if account.Banned {
			return fmt.Sprintf("Account '%s' is already banned.", account.Username)
		}

		accountID = account.ID
		accountUsername = account.Username
	}

	// Ban the account
	if err := db.BanAccount(accountID); err != nil {
		return fmt.Sprintf("Failed to ban account: %v", err)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "ban",
		"admin", p.GetName(),
		"target_account", accountUsername,
		"reason", reason)

	if reason != "" {
		return fmt.Sprintf("Account '%s' has been banned. Reason: %s", accountUsername, reason)
	}
	return fmt.Sprintf("Account '%s' has been banned.", accountUsername)
}

// executeAdminUnban unbans an account
func executeAdminUnban(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin unban <username>"
	}

	username := c.Args[1]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Get account by username
	account, err := db.GetAccountByUsername(username)
	if err != nil {
		return fmt.Sprintf("Account '%s' not found.", username)
	}

	if !account.Banned {
		return fmt.Sprintf("Account '%s' is not banned.", username)
	}

	// Unban the account
	if err := db.UnbanAccount(account.ID); err != nil {
		return fmt.Sprintf("Failed to unban account: %v", err)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "unban",
		"admin", p.GetName(),
		"target_account", username)

	return fmt.Sprintf("Account '%s' has been unbanned.", username)
}

// executeAdminKick disconnects a player
func executeAdminKick(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin kick <player_name> [reason]"
	}

	targetName := c.Args[1]
	reason := ""
	if len(c.Args) > 2 {
		reason = strings.Join(c.Args[2:], " ")
	}

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	if !server.KickPlayer(targetName, reason) {
		return fmt.Sprintf("Player '%s' not found or not online.", targetName)
	}

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "kick",
		"admin", p.GetName(),
		"target", targetName,
		"reason", reason)

	if reason != "" {
		return fmt.Sprintf("%s has been kicked. Reason: %s", targetName, reason)
	}
	return fmt.Sprintf("%s has been kicked.", targetName)
}

// executeAdminAnnounce broadcasts a server-wide message
func executeAdminAnnounce(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin announce <message>"
	}

	message := strings.Join(c.Args[1:], " ")

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Broadcast announcement
	announcement := fmt.Sprintf("\n[ANNOUNCEMENT from %s] %s\n", p.GetName(), message)
	server.BroadcastToAll(announcement)

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "announce",
		"admin", p.GetName(),
		"message", message)

	return "Announcement sent."
}

// executeAdminTeleport moves a player to a specific room
func executeAdminTeleport(c *Command, p PlayerInterface) string {
	if len(c.Args) < 3 {
		return "Usage: admin teleport <player_name> <room_id>"
	}

	targetName := c.Args[1]
	roomID := c.Args[2]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get the world
	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Find the target player
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' not found or not online.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Find the target room
	room := w.GetRoom(roomID)
	if room == nil {
		return fmt.Sprintf("Room '%s' not found.", roomID)
	}

	// Broadcast exit message from current room
	currentRoom := target.GetCurrentRoom()
	if currentRoom != nil {
		if r, ok := currentRoom.(RoomInterface); ok {
			server.BroadcastToRoom(r.GetID(), fmt.Sprintf("%s vanishes in a flash of light!\n", target.GetName()), target)
		}
	}

	// Move the player
	target.MoveTo(room)

	// Broadcast enter message to new room
	server.BroadcastToRoom(roomID, fmt.Sprintf("%s appears in a flash of light!\n", target.GetName()), target)

	// Send room description to teleported player
	target.SendMessage(fmt.Sprintf("\n*** You have been teleported by %s ***\n\n%s", p.GetName(), room.GetDescriptionForPlayer(target.GetName())))

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "teleport",
		"admin", p.GetName(),
		"target", target.GetName(),
		"destination", roomID)

	return fmt.Sprintf("%s has been teleported to %s.", target.GetName(), roomID)
}

// executeAdminGoto teleports the admin to a specific room
func executeAdminGoto(c *Command, p PlayerInterface) string {
	if len(c.Args) < 2 {
		return "Usage: admin goto <room_id>"
	}

	roomID := c.Args[1]

	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get the world
	worldIface := server.GetWorld()
	w, ok := worldIface.(*world.World)
	if !ok {
		return "Internal error: invalid world type"
	}

	// Find the target room
	room := w.GetRoom(roomID)
	if room == nil {
		return fmt.Sprintf("Room '%s' not found.", roomID)
	}

	// Broadcast exit message from current room
	currentRoom := p.GetCurrentRoom()
	if currentRoom != nil {
		if r, ok := currentRoom.(RoomInterface); ok {
			server.BroadcastToRoom(r.GetID(), fmt.Sprintf("%s vanishes in a flash of light!\n", p.GetName()), p)
		}
	}

	// Move the player
	p.MoveTo(room)

	// Broadcast enter message to new room
	server.BroadcastToRoom(roomID, fmt.Sprintf("%s appears in a flash of light!\n", p.GetName()), p)

	// Log admin action
	logger.Always("ADMIN_ACTION",
		"action", "goto",
		"admin", p.GetName(),
		"destination", roomID)

	return fmt.Sprintf("Teleported to %s.\n\n%s", roomID, room.GetDescriptionForPlayer(p.GetName()))
}

// executeAdminStats shows server statistics
func executeAdminStats(c *Command, p PlayerInterface) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	db, ok := server.GetDatabase().(*database.Database)
	if !ok {
		return "Internal error: database not available"
	}

	// Get stats
	uptime := server.GetUptime()
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	playersOnline := len(server.GetOnlinePlayers())
	roomCount := server.GetWorldRoomCount()

	totalAccounts, _ := db.GetTotalAccounts()
	totalCharacters, _ := db.GetTotalCharacters()

	return fmt.Sprintf(`
Server Statistics
=================
Uptime:           %d hours, %d minutes, %d seconds
Players Online:   %d
World Rooms:      %d
Total Accounts:   %d
Total Characters: %d
`,
		hours, minutes, seconds,
		playersOnline,
		roomCount,
		totalAccounts,
		totalCharacters)
}

// executeAdminPlayers lists all online players with details
func executeAdminPlayers(c *Command, p PlayerInterface) string {
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	players := server.GetOnlinePlayersDetailed()

	if len(players) == 0 {
		return "No players online."
	}

	result := "\nOnline Players\n==============\n"
	for _, pi := range players {
		adminTag := ""
		if pi.IsAdmin {
			adminTag = " [ADMIN]"
		}
		result += fmt.Sprintf("  %s (Lvl %d) - Room: %s - IP: %s%s\n",
			pi.Name, pi.Level, pi.RoomID, pi.IP, adminTag)
	}
	result += fmt.Sprintf("\nTotal: %d player(s)\n", len(players))

	return result
}
