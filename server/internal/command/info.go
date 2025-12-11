package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/help"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
)

// executeHelp shows help for commands
func executeHelp(c *Command, p PlayerInterface) string {
	// Get topic from args
	topic := ""
	if len(c.Args) > 0 {
		topic = strings.ToLower(strings.Join(c.Args, " "))
	}

	// Check if player is admin for showing admin commands
	isAdmin := false
	if p != nil {
		isAdmin = p.IsAdmin()
	}

	return getHelpText(topic, isAdmin)
}

// executeScore shows a comprehensive character summary (replaces stats command)
func executeScore(c *Command, p PlayerInterface) string {
	var result strings.Builder

	// Header with name, race, and class
	result.WriteString(fmt.Sprintf("=== %s ===\n", p.GetName()))
	result.WriteString(fmt.Sprintf("Race: %s\n", p.GetRaceName()))
	result.WriteString(fmt.Sprintf("Class: %s\n", p.GetClassLevelsSummary()))
	result.WriteString(fmt.Sprintf("Active: %s (gaining XP)\n", p.GetActiveClassName()))

	// Level and XP section
	level := p.GetLevel()
	xp := p.GetExperience()
	result.WriteString(fmt.Sprintf("Level: %d", level))
	if level >= leveling.MaxPlayerLevel {
		result.WriteString(" (MAX)\n")
	} else {
		xpNeeded := leveling.XPForLevel(level + 1)
		result.WriteString(fmt.Sprintf("  |  XP: %d / %d\n", xp, xpNeeded))
	}

	// Health and Mana
	result.WriteString(fmt.Sprintf("Health: %d / %d\n", p.GetHealth(), p.GetMaxHealth()))
	result.WriteString(fmt.Sprintf("Mana: %d / %d\n", p.GetMana(), p.GetMaxMana()))

	// Gold
	result.WriteString(fmt.Sprintf("Gold: %d\n", p.GetGold()))

	// Ability Scores section
	result.WriteString("\n--- Ability Scores ---\n")
	abilities := []struct {
		name  string
		short string
		score int
	}{
		{"Strength", "STR", p.GetStrength()},
		{"Dexterity", "DEX", p.GetDexterity()},
		{"Constitution", "CON", p.GetConstitution()},
		{"Intelligence", "INT", p.GetIntelligence()},
		{"Wisdom", "WIS", p.GetWisdom()},
		{"Charisma", "CHA", p.GetCharisma()},
	}

	for _, a := range abilities {
		mod := stats.Modifier(a.score)
		modStr := fmt.Sprintf("%+d", mod)
		result.WriteString(fmt.Sprintf("  %-12s (%s): %2d (%s)\n", a.name, a.short, a.score, modStr))
	}

	// Current state
	result.WriteString(fmt.Sprintf("\nState: %s\n", p.GetState()))

	return result.String()
}

// executeLevel shows detailed level progression information
func executeLevel(c *Command, p PlayerInterface) string {
	level := p.GetLevel()
	xp := p.GetExperience()

	var result strings.Builder
	result.WriteString("=== Level Progress ===\n")
	result.WriteString(fmt.Sprintf("Current Level: %d", level))

	if level >= leveling.MaxPlayerLevel {
		result.WriteString(" (MAX)\n")
		result.WriteString(fmt.Sprintf("Total Experience: %d\n", xp))
		result.WriteString("\nYou have reached the maximum level!")
	} else {
		result.WriteString("\n")

		xpNeeded := leveling.XPForLevel(level + 1)
		xpCurrent := leveling.XPForLevel(level)
		xpProgress := xp - xpCurrent
		xpRequired := xpNeeded - xpCurrent
		xpToGo := xpNeeded - xp

		percent := 0
		if xpRequired > 0 {
			percent = (xpProgress * 100) / xpRequired
		}

		result.WriteString(fmt.Sprintf("Experience: %d / %d\n", xp, xpNeeded))

		// Build progress bar (20 characters wide)
		barWidth := 20
		filled := (percent * barWidth) / 100
		bar := strings.Repeat("#", filled) + strings.Repeat(".", barWidth-filled)
		result.WriteString(fmt.Sprintf("Progress: [%s] %d%%\n", bar, percent))

		result.WriteString(fmt.Sprintf("XP to next level: %d", xpToGo))
	}

	return result.String()
}

// executePassword changes the player's account password
func executePassword(c *Command, p PlayerInterface) string {
	if err := c.RequireArgs(2, "Usage: password <old_password> <new_password>"); err != nil {
		return err.Error()
	}

	oldPassword := c.Args[0]
	newPassword := c.Args[1]

	// Validate new password length
	if len(newPassword) < 4 {
		return "New password must be at least 4 characters."
	}

	// Get database from server
	server, ok := p.GetServer().(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	dbIface := server.GetDatabase()
	if dbIface == nil {
		return "Password change is not available."
	}

	db, ok := dbIface.(*database.Database)
	if !ok {
		return "Internal error: invalid database type"
	}

	accountID := p.GetAccountID()
	if accountID == 0 {
		return "Account not found."
	}

	// Verify old password and change to new password
	if err := db.ChangePasswordWithVerify(accountID, oldPassword, newPassword); err != nil {
		if errors.Is(err, database.ErrInvalidCredentials) {
			return "Old password is incorrect."
		}
		return fmt.Sprintf("Failed to change password: %v", err)
	}

	return "Password changed successfully."
}

// executeRace shows race information
func executeRace(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		// Show player's own race info
		playerRace, err := race.ParseRace(strings.ToLower(p.GetRaceName()))
		if err != nil {
			return fmt.Sprintf("Your race: %s\n\nUse 'race <name>' to view information about a specific race.\nValid races: Human, Dwarf, Elf, Gnome, Orc", p.GetRaceName())
		}
		return formatRaceInfo(playerRace)
	}

	// Show info about a specific race
	raceName := strings.ToLower(strings.Join(c.Args, "-"))
	r, err := race.ParseRace(raceName)
	if err != nil {
		return fmt.Sprintf("Unknown race: %s\nValid races: Human, Dwarf, Elf, Gnome, Orc", c.Args[0])
	}
	return formatRaceInfo(r)
}

// formatRaceInfo formats detailed race information
func formatRaceInfo(r race.Race) string {
	def := race.GetDefinition(r)
	if def == nil {
		return fmt.Sprintf("Race information not found for %s", r.String())
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== %s ===\n", r.String()))
	sb.WriteString(fmt.Sprintf("Size: %s\n\n", def.Size))
	sb.WriteString(fmt.Sprintf("%s\n\n", def.Description))

	sb.WriteString("Stat Bonuses:\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", def.GetStatBonusesString()))

	sb.WriteString("Racial Abilities:\n")
	for _, ability := range def.Abilities {
		sb.WriteString(fmt.Sprintf("  - %s\n", ability))
	}

	return sb.String()
}

// executeRaces lists all available races
func executeRaces(c *Command, p PlayerInterface) string {
	var sb strings.Builder

	sb.WriteString("=== Available Races ===\n\n")

	for _, r := range race.AllRaces() {
		def := race.GetDefinition(r)
		if def == nil {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s (%s)\n", r.String(), def.Size))
		sb.WriteString(fmt.Sprintf("  Bonuses: %s\n", def.GetStatBonusesString()))
		sb.WriteString(fmt.Sprintf("  %s\n\n", def.Description))
	}

	sb.WriteString("Use 'race <name>' for detailed information about a specific race.")

	return sb.String()
}

// getHelpText returns help text for a given topic using the YAML-based help system.
func getHelpText(topic string, isAdmin bool) string {
	h := help.GetInstance()
	if h == nil {
		// Fallback if help not loaded
		if topic == "" {
			return "Help system not loaded. Type 'help <command>' for specific help."
		}
		return fmt.Sprintf("No help available for '%s'.", topic)
	}
	return h.GetHelpText(topic, isAdmin)
}
