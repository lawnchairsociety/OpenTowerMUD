package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
)

// executeSay broadcasts a message to everyone in the current room
func executeSay(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Say what?"); err != nil {
		return err.Error()
	}

	message := c.GetItemName() // Reusing GetItemName to join all args

	// Check for spam
	if allowed, reason := p.CheckChatSpam(message); !allowed {
		return reason
	}

	// Get the current room
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasting and chat filter
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Apply chat filter if enabled
	filteredMessage := message
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(message)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "say",
				"room", room.GetID(),
				"original", message,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your message contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered message
			filteredMessage = result.Filtered
		}
	}

	broadcastMsg := fmt.Sprintf("%s says: \"%s\"\n", p.GetName(), filteredMessage)
	server.BroadcastToRoomFromPlayer(room.GetID(), broadcastMsg, p, p.GetName())

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_SAY",
		"player", p.GetName(),
		"room", room.GetID(),
		"message", filteredMessage)

	return fmt.Sprintf("You say: \"%s\"", filteredMessage)
}

// executeWho lists all online players
func executeWho(c *Command, p PlayerInterface) string {
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	players := server.GetOnlinePlayers()

	if len(players) == 0 {
		return "No players online."
	}

	result := "Online Players:\n"
	for _, playerName := range players {
		result += fmt.Sprintf("  - %s\n", playerName)
	}
	return result
}

// executeTell sends a private message to another player
func executeTell(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: tell <player> <message>"); err != nil {
		return err.Error()
	}

	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Check for spam (using full args as message approximation for rate limiting)
	fullMessage := strings.Join(c.Args[1:], " ")
	if allowed, reason := p.CheckChatSpam(fullMessage); !allowed {
		return reason
	}

	// Try to find player by matching progressively longer portions of args
	// This allows player names with spaces like "Bob Johnson"
	var target PlayerInterface
	var messageStartIndex int

	// Try matching from longest possible name to shortest
	for i := len(c.Args) - 1; i >= 0; i-- {
		candidateName := strings.Join(c.Args[0:i+1], " ")
		targetIface := server.FindPlayer(candidateName)
		if targetIface != nil {
			var ok bool
			target, ok = targetIface.(PlayerInterface)
			if ok {
				messageStartIndex = i + 1
				break
			}
		}
	}

	// If no player found, return error
	if target == nil {
		return fmt.Sprintf("Player '%s' not found.", c.Args[0])
	}

	// Check if there's a message after the player name
	if messageStartIndex >= len(c.Args) {
		return "Usage: tell <player> <message>"
	}

	message := strings.Join(c.Args[messageStartIndex:], " ")

	// Apply chat filter if enabled
	filteredMessage := message
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(message)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "tell",
				"recipient", target.GetName(),
				"original", message,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your message contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered message
			filteredMessage = result.Filtered
		}
	}

	// Check if target is ignoring the sender
	if target.IsIgnoring(p.GetName()) {
		// Still log it but don't tell the sender they're ignored
		logger.Always("CHAT_TELL_IGNORED",
			"sender", p.GetName(),
			"recipient", target.GetName(),
			"message", filteredMessage)
		// Pretend message was sent (don't reveal ignore status)
		return fmt.Sprintf("You tell %s: \"%s\"", target.GetName(), filteredMessage)
	}

	// Send message to target
	target.SendMessage(fmt.Sprintf("%s tells you: \"%s\"\n", p.GetName(), filteredMessage))

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_TELL",
		"sender", p.GetName(),
		"recipient", target.GetName(),
		"message", filteredMessage)

	return fmt.Sprintf("You tell %s: \"%s\"", target.GetName(), filteredMessage)
}

// executeShout broadcasts a message to all players on the same floor
func executeShout(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Shout what?"); err != nil {
		return err.Error()
	}

	message := c.GetItemName()

	// Check for spam
	if allowed, reason := p.CheckChatSpam(message); !allowed {
		return reason
	}

	// Get the current room to determine floor
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasting and chat filter
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Apply chat filter if enabled
	filteredMessage := message
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(message)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "shout",
				"floor", room.GetFloor(),
				"original", message,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your message contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered message
			filteredMessage = result.Filtered
		}
	}

	floor := room.GetFloor()
	broadcastMsg := fmt.Sprintf("%s shouts: \"%s\"\n", p.GetName(), filteredMessage)
	server.BroadcastToFloorFromPlayer(floor, broadcastMsg, p, p.GetName())

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_SHOUT",
		"player", p.GetName(),
		"floor", floor,
		"message", filteredMessage)

	return fmt.Sprintf("You shout: \"%s\"", filteredMessage)
}

// executeEmote performs a custom action visible to everyone in the room
func executeEmote(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Emote what?"); err != nil {
		return err.Error()
	}

	action := c.GetItemName()

	// Check for spam
	if allowed, reason := p.CheckChatSpam(action); !allowed {
		return reason
	}

	// Get the current room
	roomIface := p.GetCurrentRoom()
	room, ok := roomIface.(RoomInterface)
	if !ok {
		return "Internal error: invalid room type"
	}

	// Get server for broadcasting and chat filter
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Apply chat filter if enabled
	filteredAction := action
	if filter := server.GetChatFilter(); filter != nil && filter.IsEnabled() {
		result := filter.Check(action)
		if result.Violated {
			// Log the violation
			logger.Always("CHAT_FILTER",
				"player", p.GetName(),
				"command", "emote",
				"room", room.GetID(),
				"original", action,
				"matched", strings.Join(result.MatchedWords, ", "),
				"mode", string(filter.Mode()))

			if filter.IsBlockMode() {
				return "Your emote contains inappropriate language and was not sent."
			}
			// REPLACE mode - use filtered action
			filteredAction = result.Filtered
		}
	}

	// Format: "PlayerName laughs" (no quotes around action)
	broadcastMsg := fmt.Sprintf("%s %s\n", p.GetName(), filteredAction)
	server.BroadcastToRoomFromPlayer(room.GetID(), broadcastMsg, p, p.GetName())

	// AUDIT LOG - Always logged regardless of log level (security/moderation)
	logger.Always("CHAT_EMOTE",
		"player", p.GetName(),
		"room", room.GetID(),
		"action", filteredAction)

	return fmt.Sprintf("%s %s", p.GetName(), filteredAction)
}

// executeQuit disconnects the player
func executeQuit(c *Command, p PlayerInterface) string {
	p.Disconnect()
	return "Goodbye!"
}
