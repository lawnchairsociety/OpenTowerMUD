package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
)

// executeReport allows players to report other players for misconduct
func executeReport(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: report <player> <reason>"); err != nil {
		return err.Error()
	}

	targetName := c.Args[0]
	reason := strings.Join(c.Args[1:], " ")

	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Check if target player exists (online)
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' is not online.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	// Can't report yourself
	if strings.EqualFold(target.GetName(), p.GetName()) {
		return "You can't report yourself."
	}

	// Get room info
	roomID := "unknown"
	if roomIface := p.GetCurrentRoom(); roomIface != nil {
		if room, ok := roomIface.(RoomInterface); ok {
			roomID = room.GetID()
		}
	}

	// Create report entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	reportEntry := fmt.Sprintf("[%s] Reporter: %s | Reported: %s | Room: %s | Reason: %s\n",
		timestamp, p.GetName(), target.GetName(), roomID, reason)

	// Log to reports file
	if err := appendToReportsFile(reportEntry); err != nil {
		logger.Warning("Failed to write report to file", "error", err)
	}

	// Log via logger as well
	logger.Always("PLAYER_REPORT",
		"reporter", p.GetName(),
		"reported", target.GetName(),
		"room", roomID,
		"reason", reason)

	// Notify online admins
	adminNotice := fmt.Sprintf("\n[REPORT] %s reported %s: %s\n", p.GetName(), target.GetName(), reason)
	server.BroadcastToAdmins(adminNotice)

	return fmt.Sprintf("Your report against %s has been logged. Thank you for helping keep the game safe.", target.GetName())
}

// appendToReportsFile appends a report entry to the reports log file
func appendToReportsFile(entry string) error {
	f, err := os.OpenFile("data/reports.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

// executeIgnore manages the player's ignore list
func executeIgnore(c *Command, p PlayerInterface) string {
	// No args - show current ignore list
	if len(c.Args) == 0 {
		return showIgnoreList(p)
	}

	targetName := c.Args[0]

	// Check for "ignore list" command
	if strings.EqualFold(targetName, "list") {
		return showIgnoreList(p)
	}

	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Find the target player to get their actual name
	targetIface := server.FindPlayer(targetName)
	if targetIface == nil {
		return fmt.Sprintf("Player '%s' is not online. You can only ignore online players.", targetName)
	}

	target, ok := targetIface.(PlayerInterface)
	if !ok {
		return "Internal error: invalid player type"
	}

	actualName := target.GetName()

	// Can't ignore yourself
	if strings.EqualFold(actualName, p.GetName()) {
		return "You can't ignore yourself."
	}

	// Toggle ignore status
	if p.IsIgnoring(actualName) {
		p.RemoveIgnore(actualName)
		return fmt.Sprintf("You are no longer ignoring %s.", actualName)
	}

	p.AddIgnore(actualName)
	return fmt.Sprintf("You are now ignoring %s. You will no longer see their messages.", actualName)
}

// executeUnignore removes a player from the ignore list
func executeUnignore(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(1, "Usage: unignore <player>"); err != nil {
		return err.Error()
	}

	targetName := c.Args[0]

	if !p.IsIgnoring(targetName) {
		return fmt.Sprintf("You are not ignoring '%s'.", targetName)
	}

	p.RemoveIgnore(targetName)
	return fmt.Sprintf("You are no longer ignoring %s.", targetName)
}

// showIgnoreList displays the player's current ignore list
func showIgnoreList(p PlayerInterface) string {
	list := p.GetIgnoreList()
	if len(list) == 0 {
		return "Your ignore list is empty."
	}

	sort.Strings(list)
	return fmt.Sprintf("Ignored players: %s\n\nUse 'ignore <player>' to toggle, or 'unignore <player>' to remove.", strings.Join(list, ", "))
}
