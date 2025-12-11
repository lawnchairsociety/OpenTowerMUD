package server

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
	"github.com/lawnchairsociety/opentowermud/server/internal/text"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
)

// isValidCharacterName checks if a character name contains only allowed characters.
// Allowed: letters (any language), hyphens, and apostrophes.
// Names cannot start or end with hyphens/apostrophes, and cannot have consecutive special characters.
func isValidCharacterName(name string) bool {
	if len(name) == 0 {
		return false
	}

	runes := []rune(name)

	// First and last character must be a letter
	if !unicode.IsLetter(runes[0]) || !unicode.IsLetter(runes[len(runes)-1]) {
		return false
	}

	prevWasSpecial := false
	for _, r := range runes {
		if unicode.IsLetter(r) {
			prevWasSpecial = false
			continue
		}
		if r == '-' || r == '\'' {
			// No consecutive special characters
			if prevWasSpecial {
				return false
			}
			prevWasSpecial = true
			continue
		}
		// Any other character is invalid
		return false
	}

	return true
}

// AuthResult contains the result of the authentication flow
type AuthResult struct {
	Account   *database.Account
	Character *database.Character
}

// handleAuth handles the login/registration flow for a new connection.
// Returns the authenticated account and selected character, or an error.
func (s *Server) handleAuth(client Client) (*AuthResult, error) {
	// Check if website registration is configured
	cfg := s.GetServerConfig()
	websiteURL := cfg.Website.URL

	// Welcome screen
	t := text.GetInstance()
	if t != nil {
		client.WriteLine("\n")
		client.WriteLine(t.GetWelcomeBanner())
		client.WriteLine("\n")
	} else {
		// Fallback if text not loaded
		client.WriteLine("\n")
		client.WriteLine("=====================================\n")
		client.WriteLine("    Welcome to Open Tower MUD!\n")
		client.WriteLine("=====================================\n")
		client.WriteLine("\n")
		client.WriteLine("  [L] Login\n")
		if websiteURL == "" {
			client.WriteLine("  [R] Register\n")
		}
		client.WriteLine("\n")
	}

	// Show website registration URL if configured
	if websiteURL != "" {
		client.WriteLine(fmt.Sprintf("New player? Register at: %s/register\n\n", websiteURL))
	}

	client.WriteLine("Enter choice: ")

	choice, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	choice = strings.ToLower(strings.TrimSpace(choice))

	switch choice {
	case "l", "login":
		return s.handleLogin(client)
	case "r", "register":
		// Only allow registration if website URL is not configured
		if websiteURL != "" {
			client.WriteLine(fmt.Sprintf("\nIn-game registration is disabled.\nPlease register at: %s/register\n\n", websiteURL))
			return nil, errors.New("registration disabled - use website")
		}
		return s.handleRegister(client)
	default:
		client.WriteLine("Invalid choice. Disconnecting.\n")
		return nil, errors.New("invalid choice")
	}
}

// handleLogin handles the login flow
func (s *Server) handleLogin(client Client) (*AuthResult, error) {
	client.WriteLine("\n--- Login ---\n")

	// Get IP address for rate limiting
	ipAddress := getIPFromAddr(client.RemoteAddr())

	// Check if IP is rate limited
	if s.loginRateLimiter != nil {
		if locked, remaining := s.loginRateLimiter.IsLocked(ipAddress); locked {
			client.WriteLine(fmt.Sprintf("Too many failed login attempts. Please wait %d seconds.\n",
				int(remaining.Seconds())))
			return nil, errors.New("rate limited")
		}
	}

	// Get username
	client.WriteLine("Username: ")
	username, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		client.WriteLine("Username cannot be empty.\n")
		return nil, errors.New("empty username")
	}

	// Get password
	client.WriteLine("Password: ")
	password, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}

	// Validate credentials
	account, err := s.db.ValidateLogin(username, password, ipAddress)
	if err != nil {
		if errors.Is(err, database.ErrAccountBanned) {
			logger.Info("Login attempt on banned account",
				"username", username,
				"ip", ipAddress,
				"event", "login_banned")
			client.WriteLine("\n*** YOUR ACCOUNT HAS BEEN BANNED ***\n")
			client.WriteLine("Contact an administrator if you believe this is an error.\n")
			return nil, errors.New("account banned")
		}
		if errors.Is(err, database.ErrInvalidCredentials) {
			logger.Info("Failed login attempt",
				"username", username,
				"ip", ipAddress,
				"event", "login_failed")
			// Record failed attempt
			if s.loginRateLimiter != nil {
				if locked, duration := s.loginRateLimiter.RecordFailure(ipAddress); locked {
					logger.Warning("IP rate limited after failed logins",
						"ip", ipAddress,
						"lockout_seconds", int(duration.Seconds()),
						"event", "login_ratelimit")
					client.WriteLine(fmt.Sprintf("Invalid username or password. Too many attempts - locked out for %d seconds.\n",
						int(duration.Seconds())))
					return nil, errors.New("rate limited")
				}
			}
			client.WriteLine("Invalid username or password.\n")
			return nil, errors.New("invalid credentials")
		}
		client.WriteLine("An error occurred. Please try again.\n")
		return nil, err
	}

	// Successful login - clear rate limit
	if s.loginRateLimiter != nil {
		s.loginRateLimiter.RecordSuccess(ipAddress)
	}

	logger.Info("Successful login",
		"username", account.Username,
		"account_id", account.ID,
		"ip", ipAddress,
		"event", "login_success")

	client.WriteLine(fmt.Sprintf("\nWelcome back, %s!\n", account.Username))

	// Character selection
	character, err := s.handleCharacterSelection(client, account)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Account: account, Character: character}, nil
}

// getIPFromAddr extracts the IP address from an address string
func getIPFromAddr(addr string) string {
	// Remove port from address (format is usually "ip:port")
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// handleRegister handles the registration flow
func (s *Server) handleRegister(client Client) (*AuthResult, error) {
	client.WriteLine("\n--- Register ---\n")

	// Get username
	client.WriteLine("Choose a username: ")
	username, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		client.WriteLine("Username cannot be empty.\n")
		return nil, errors.New("empty username")
	}
	if len(username) < 3 {
		client.WriteLine("Username must be at least 3 characters.\n")
		return nil, errors.New("username too short")
	}
	if len(username) > 20 {
		client.WriteLine("Username must be 20 characters or less.\n")
		return nil, errors.New("username too long")
	}

	// Check if username exists
	exists, err := s.db.AccountExists(username)
	if err != nil {
		client.WriteLine("An error occurred. Please try again.\n")
		return nil, err
	}
	if exists {
		client.WriteLine("That username is already taken.\n")
		return nil, errors.New("username taken")
	}

	// Get password with validation
	pwConfig := s.GetServerConfig().Password
	client.WriteLine(fmt.Sprintf("Choose a password (%s): ", pwConfig.GetRequirementsText()))
	password, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	if validationErr := pwConfig.ValidatePassword(password); validationErr != "" {
		client.WriteLine(validationErr + "\n")
		return nil, errors.New("password requirements not met")
	}

	// Confirm password
	client.WriteLine("Confirm password: ")
	confirmPassword, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	if password != confirmPassword {
		client.WriteLine("Passwords do not match.\n")
		return nil, errors.New("password mismatch")
	}

	// Create account
	ipAddress := getIPFromAddr(client.RemoteAddr())
	account, err := s.db.CreateAccount(username, password)
	if err != nil {
		if errors.Is(err, database.ErrAccountExists) {
			client.WriteLine("That username is already taken.\n")
			return nil, errors.New("username taken")
		}
		client.WriteLine("An error occurred. Please try again.\n")
		return nil, err
	}

	logger.Info("Account registered",
		"username", account.Username,
		"account_id", account.ID,
		"ip", ipAddress,
		"event", "account_register")

	client.WriteLine(fmt.Sprintf("\nAccount created! Welcome, %s!\n", account.Username))

	// Go straight to character creation for new accounts
	character, err := s.handleCharacterCreation(client, account)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Account: account, Character: character}, nil
}

// handleCharacterSelection handles character selection/creation
func (s *Server) handleCharacterSelection(client Client, account *database.Account) (*database.Character, error) {
	for {
		// Get characters for this account
		characters, err := s.db.GetCharactersByAccount(account.ID)
		if err != nil {
			client.WriteLine("An error occurred. Please try again.\n")
			return nil, err
		}

		client.WriteLine("\n--- Character Selection ---\n")

		if len(characters) == 0 {
			client.WriteLine("You have no characters.\n")
		} else {
			for i, c := range characters {
				classDisplay := c.PrimaryClass
				if classDisplay == "" {
					classDisplay = "Warrior"
				} else {
					classDisplay = strings.Title(classDisplay)
				}
				raceDisplay := c.Race
				if raceDisplay == "" {
					raceDisplay = "Human"
				} else {
					raceDisplay = strings.Title(raceDisplay)
				}
				client.WriteLine(fmt.Sprintf("  [%d] %s - Level %d %s %s\n", i+1, c.Name, c.Level, raceDisplay, classDisplay))
			}
		}

		client.WriteLine("\n  [C] Create new character\n")
		if len(characters) > 0 {
			client.WriteLine("  [D] Delete a character\n")
		}
		client.WriteLine("\nEnter choice: ")

		choice, err := client.ReadLine()
		if err != nil {
			return nil, errors.New("connection closed")
		}
		choice = strings.TrimSpace(choice)

		// Check for create/delete commands
		if strings.ToLower(choice) == "c" {
			character, err := s.handleCharacterCreation(client, account)
			if err != nil {
				// Show error but continue the loop
				continue
			}
			return character, nil
		}

		if strings.ToLower(choice) == "d" && len(characters) > 0 {
			if err := s.handleCharacterDeletion(client, account); err != nil {
				// Show error but continue the loop
			}
			continue
		}

		// Try to parse as character number
		if len(characters) > 0 {
			var charIndex int
			if _, err := fmt.Sscanf(choice, "%d", &charIndex); err == nil {
				if charIndex >= 1 && charIndex <= len(characters) {
					selected := characters[charIndex-1]

					// Check if character is already online
					if s.IsCharacterOnline(selected.Name) {
						client.WriteLine("That character is already logged in.\n")
						continue
					}

					return selected, nil
				}
			}
		}

		client.WriteLine("Invalid choice.\n")
	}
}

// handleCharacterCreation handles creating a new character
func (s *Server) handleCharacterCreation(client Client, account *database.Account) (*database.Character, error) {
	client.WriteLine("\n--- Create Character ---\n")
	client.WriteLine("Enter character name: ")

	name, err := client.ReadLine()
	if err != nil {
		return nil, errors.New("connection closed")
	}
	name = strings.TrimSpace(name)

	// Validate name
	if name == "" {
		client.WriteLine("Character name cannot be empty.\n")
		return nil, errors.New("empty name")
	}
	if len(name) < 2 {
		client.WriteLine("Character name must be at least 2 characters.\n")
		return nil, errors.New("name too short")
	}
	if len(name) > 20 {
		client.WriteLine("Character name must be 20 characters or less.\n")
		return nil, errors.New("name too long")
	}
	if !isValidCharacterName(name) {
		client.WriteLine("Character name can only contain letters, hyphens, and apostrophes.\n")
		return nil, errors.New("invalid characters in name")
	}

	// Check name filter for banned words/names
	if nf := s.GetNameFilter(); nf != nil {
		result := nf.Check(name)
		if !result.Allowed {
			client.WriteLine(result.Reason + "\n")
			return nil, errors.New("name not allowed")
		}
	}

	// Check if name exists
	exists, err := s.db.CharacterNameExists(name)
	if err != nil {
		client.WriteLine("An error occurred. Please try again.\n")
		return nil, err
	}
	if exists {
		client.WriteLine("That character name is already taken.\n")
		return nil, errors.New("name taken")
	}

	// Select class
	selectedClass, err := s.handleClassSelection(client)
	if err != nil {
		return nil, err
	}

	// Select race
	selectedRace, err := s.handleRaceSelection(client)
	if err != nil {
		return nil, err
	}

	// Select home city/tower
	selectedTower, err := s.handleCitySelection(client)
	if err != nil {
		return nil, err
	}

	// Assign ability scores using the standard array
	scores, err := s.handleAbilityScoreAssignment(client, selectedClass, selectedRace)
	if err != nil {
		return nil, err
	}

	// Apply racial stat bonuses
	raceDef := race.GetDefinition(race.Race(selectedRace))
	if raceDef != nil && raceDef.HasStatBonus() {
		scores.Strength, scores.Dexterity, scores.Constitution,
			scores.Intelligence, scores.Wisdom, scores.Charisma =
			raceDef.ApplyStatBonuses(scores.Strength, scores.Dexterity, scores.Constitution,
				scores.Intelligence, scores.Wisdom, scores.Charisma)
	}

	// Handle Human's +1 to any stat
	if selectedRace == "human" {
		bonusStat, err := s.handleHumanBonusSelection(client, scores)
		if err != nil {
			return nil, err
		}
		switch bonusStat {
		case "STR":
			scores.Strength++
		case "DEX":
			scores.Dexterity++
		case "CON":
			scores.Constitution++
		case "INT":
			scores.Intelligence++
		case "WIS":
			scores.Wisdom++
		case "CHA":
			scores.Charisma++
		}
	}

	// Show final stats
	client.WriteLine("\n--- Final Ability Scores ---\n")
	client.WriteLine(fmt.Sprintf("  STR: %d (%+d)\n", scores.Strength, stats.Modifier(scores.Strength)))
	client.WriteLine(fmt.Sprintf("  DEX: %d (%+d)\n", scores.Dexterity, stats.Modifier(scores.Dexterity)))
	client.WriteLine(fmt.Sprintf("  CON: %d (%+d)\n", scores.Constitution, stats.Modifier(scores.Constitution)))
	client.WriteLine(fmt.Sprintf("  INT: %d (%+d)\n", scores.Intelligence, stats.Modifier(scores.Intelligence)))
	client.WriteLine(fmt.Sprintf("  WIS: %d (%+d)\n", scores.Wisdom, stats.Modifier(scores.Wisdom)))
	client.WriteLine(fmt.Sprintf("  CHA: %d (%+d)\n", scores.Charisma, stats.Modifier(scores.Charisma)))

	// Create character with assigned ability scores, class, race, and home tower
	character, err := s.db.CreateCharacterFull(account.ID, name, selectedClass, selectedRace, selectedTower,
		scores.Strength, scores.Dexterity, scores.Constitution,
		scores.Intelligence, scores.Wisdom, scores.Charisma)
	if err != nil {
		if errors.Is(err, database.ErrCharacterExists) {
			client.WriteLine("That character name is already taken.\n")
			return nil, errors.New("name taken")
		}
		client.WriteLine("An error occurred. Please try again.\n")
		return nil, err
	}

	// Get theme for display name
	theme := tower.GetTheme(tower.TowerID(selectedTower))
	cityName := "Ironhaven"
	if theme != nil {
		cityName = theme.CityName
	}

	logger.Info("Character created",
		"character", character.Name,
		"character_id", character.ID,
		"account_id", account.ID,
		"class", selectedClass,
		"race", selectedRace,
		"home_tower", selectedTower,
		"event", "character_create")

	client.WriteLine(fmt.Sprintf("\nCharacter '%s' the %s %s created in %s!\n", character.Name, strings.Title(selectedRace), strings.Title(selectedClass), cityName))
	return character, nil
}

// handleClassSelection guides the player through choosing a class
func (s *Server) handleClassSelection(client Client) (string, error) {
	client.WriteLine("\n--- Choose Your Class ---\n\n")

	// Display all classes with descriptions
	allClasses := class.AllClasses()
	for i, c := range allClasses {
		def := class.GetDefinition(c)
		if def == nil {
			continue
		}
		client.WriteLine(fmt.Sprintf("  [%d] %s\n", i+1, c.String()))
		client.WriteLine(fmt.Sprintf("      %s\n", def.Description))
		client.WriteLine(fmt.Sprintf("      Hit Die: d%d | Primary Stat: %s\n\n", def.HitDie, def.PrimaryStat))
	}

	for {
		client.WriteLine("Enter class number (1-6): ")

		input, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		input = strings.TrimSpace(input)

		// Parse the input
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(allClasses) {
			client.WriteLine("Please enter a number from 1 to 6.\n")
			continue
		}

		selectedClass := allClasses[choice-1]
		def := class.GetDefinition(selectedClass)

		// Confirm selection
		client.WriteLine(fmt.Sprintf("\nYou selected: %s\n", selectedClass.String()))
		client.WriteLine(fmt.Sprintf("  %s\n", def.Description))
		client.WriteLine("\nIs this correct? (Y/N): ")

		confirm, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		confirm = strings.ToLower(strings.TrimSpace(confirm))

		if confirm == "y" || confirm == "yes" {
			return string(selectedClass), nil
		}

		client.WriteLine("\n")
	}
}

// handleRaceSelection guides the player through choosing a race
func (s *Server) handleRaceSelection(client Client) (string, error) {
	client.WriteLine("\n--- Choose Your Race ---\n\n")

	// Display all races with descriptions
	allRaces := race.AllRaces()
	for i, r := range allRaces {
		def := race.GetDefinition(r)
		if def == nil {
			continue
		}
		client.WriteLine(fmt.Sprintf("  [%d] %s (%s)\n", i+1, r.String(), def.Size))
		client.WriteLine(fmt.Sprintf("      %s\n", def.Description))
		client.WriteLine(fmt.Sprintf("      Stat Bonuses: %s\n", def.GetStatBonusesString()))
		client.WriteLine(fmt.Sprintf("      Abilities: %s\n\n", def.GetAbilitiesString()))
	}

	for {
		client.WriteLine(fmt.Sprintf("Enter race number (1-%d): ", len(allRaces)))

		input, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		input = strings.TrimSpace(input)

		// Parse the input
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(allRaces) {
			client.WriteLine(fmt.Sprintf("Please enter a number from 1 to %d.\n", len(allRaces)))
			continue
		}

		selectedRace := allRaces[choice-1]
		def := race.GetDefinition(selectedRace)

		// Confirm selection
		client.WriteLine(fmt.Sprintf("\nYou selected: %s\n", selectedRace.String()))
		client.WriteLine(fmt.Sprintf("  %s\n", def.Description))
		client.WriteLine(fmt.Sprintf("  Stat Bonuses: %s\n", def.GetStatBonusesString()))
		client.WriteLine("\nIs this correct? (Y/N): ")

		confirm, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		confirm = strings.ToLower(strings.TrimSpace(confirm))

		if confirm == "y" || confirm == "yes" {
			return string(selectedRace), nil
		}

		client.WriteLine("\n")
	}
}

// handleCitySelection guides the player through choosing their home city
func (s *Server) handleCitySelection(client Client) (string, error) {
	client.WriteLine("\n--- Choose Your Home City ---\n\n")

	// Define available cities with their tower IDs
	type cityOption struct {
		ID   tower.TowerID
		Name string
		Desc string
	}

	cities := []cityOption{
		{tower.TowerHuman, "Ironhaven (Human)", "A grand walled city beneath the magical Arcane Spire."},
		{tower.TowerElf, "Sylvanthal (Elf)", "A forest sanctuary around the ancient, diseased World Tree."},
		{tower.TowerDwarf, "Khazad-Karn (Dwarf)", "A mountain stronghold above the endless descending mines."},
		{tower.TowerGnome, "Cogsworth (Gnome)", "A city of gears and steam beneath the Mechanical Tower."},
		{tower.TowerOrc, "Skullgar (Orc)", "A brutal war camp around the fearsome Beast-Skull Tower."},
	}

	for i, city := range cities {
		client.WriteLine(fmt.Sprintf("  [%d] %s\n", i+1, city.Name))
		client.WriteLine(fmt.Sprintf("      %s\n\n", city.Desc))
	}

	client.WriteLine("This choice determines where you start and respawn.\n")
	client.WriteLine("You can travel to other cities later in the game.\n\n")

	for {
		client.WriteLine(fmt.Sprintf("Enter city number (1-%d): ", len(cities)))

		input, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		input = strings.TrimSpace(input)

		// Parse the input
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(cities) {
			client.WriteLine(fmt.Sprintf("Please enter a number from 1 to %d.\n", len(cities)))
			continue
		}

		selectedCity := cities[choice-1]

		// Confirm selection
		client.WriteLine(fmt.Sprintf("\nYou selected: %s\n", selectedCity.Name))
		client.WriteLine(fmt.Sprintf("  %s\n", selectedCity.Desc))
		client.WriteLine("\nIs this correct? (Y/N): ")

		confirm, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		confirm = strings.ToLower(strings.TrimSpace(confirm))

		if confirm == "y" || confirm == "yes" {
			return string(selectedCity.ID), nil
		}

		client.WriteLine("\n")
	}
}

// handleHumanBonusSelection lets human players choose which stat to increase by +1
func (s *Server) handleHumanBonusSelection(client Client, scores *stats.AbilityScores) (string, error) {
	client.WriteLine("\n--- Human Versatility ---\n")
	client.WriteLine("As a Human, you may increase one ability score by 1.\n")
	client.WriteLine("Current scores:\n")
	client.WriteLine(fmt.Sprintf("  [1] STR: %d\n", scores.Strength))
	client.WriteLine(fmt.Sprintf("  [2] DEX: %d\n", scores.Dexterity))
	client.WriteLine(fmt.Sprintf("  [3] CON: %d\n", scores.Constitution))
	client.WriteLine(fmt.Sprintf("  [4] INT: %d\n", scores.Intelligence))
	client.WriteLine(fmt.Sprintf("  [5] WIS: %d\n", scores.Wisdom))
	client.WriteLine(fmt.Sprintf("  [6] CHA: %d\n", scores.Charisma))

	statNames := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

	for {
		client.WriteLine("\nWhich stat would you like to increase? (1-6): ")

		input, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > 6 {
			client.WriteLine("Please enter a number from 1 to 6.\n")
			continue
		}

		selectedStat := statNames[choice-1]
		client.WriteLine(fmt.Sprintf("\nYou selected +1 %s. Is this correct? (Y/N): ", selectedStat))

		confirm, err := client.ReadLine()
		if err != nil {
			return "", errors.New("connection closed")
		}
		confirm = strings.ToLower(strings.TrimSpace(confirm))

		if confirm == "y" || confirm == "yes" {
			return selectedStat, nil
		}
	}
}

// handleAbilityScoreAssignment guides the player through assigning ability scores
func (s *Server) handleAbilityScoreAssignment(client Client, selectedClass string, selectedRace string) (*stats.AbilityScores, error) {
	client.WriteLine("\n--- Assign Ability Scores ---\n")
	client.WriteLine("You have these values to assign: 15, 14, 13, 12, 10, 8\n")
	client.WriteLine("Each value can only be used once.\n\n")

	// Show racial bonuses
	raceDef := race.GetDefinition(race.Race(selectedRace))
	if raceDef != nil {
		client.WriteLine(fmt.Sprintf("Racial bonuses (%s): %s\n", raceDef.Name.String(), raceDef.GetStatBonusesString()))
	}
	client.WriteLine("\n")

	// Show class-specific recommendations
	client.WriteLine(fmt.Sprintf("Recommended stats for %s:\n", strings.Title(selectedClass)))
	client.WriteLine(getStatRecommendation(selectedClass))
	client.WriteLine("\n")

	// Copy the standard array so we can track which values are still available
	available := make([]int, len(stats.StandardArray))
	copy(available, stats.StandardArray)

	// Store the assigned scores
	assigned := make([]int, 6)

	// Go through each ability in order
	for i, abilityName := range stats.AbilityNames {
		for {
			// Show available values
			client.WriteLine(fmt.Sprintf("Available: %v\n", available))
			client.WriteLine(fmt.Sprintf("%s: ", abilityName))

			input, err := client.ReadLine()
			if err != nil {
				return nil, errors.New("connection closed")
			}
			input = strings.TrimSpace(input)

			// Parse the input
			value, err := strconv.Atoi(input)
			if err != nil {
				client.WriteLine("Please enter a number from the available values.\n")
				continue
			}

			// Check if the value is available
			found := -1
			for j, v := range available {
				if v == value {
					found = j
					break
				}
			}

			if found == -1 {
				client.WriteLine("That value is not available. Choose from the remaining values.\n")
				continue
			}

			// Assign the value and remove from available
			assigned[i] = value
			available = append(available[:found], available[found+1:]...)
			break
		}
	}

	// Show the final assignment
	client.WriteLine("\n--- Your Ability Scores ---\n")
	for i, name := range stats.AbilityNames {
		mod := stats.Modifier(assigned[i])
		modStr := fmt.Sprintf("%+d", mod)
		client.WriteLine(fmt.Sprintf("  %s: %d (%s)\n", name, assigned[i], modStr))
	}

	return stats.NewScores(assigned[0], assigned[1], assigned[2], assigned[3], assigned[4], assigned[5]), nil
}

// handleCharacterDeletion handles deleting a character
func (s *Server) handleCharacterDeletion(client Client, account *database.Account) error {
	characters, err := s.db.GetCharactersByAccount(account.ID)
	if err != nil {
		client.WriteLine("An error occurred. Please try again.\n")
		return err
	}

	client.WriteLine("\nWhich character do you want to delete?\n")
	for i, c := range characters {
		client.WriteLine(fmt.Sprintf("  [%d] %s (Level %d)\n", i+1, c.Name, c.Level))
	}
	client.WriteLine("Enter number (or 0 to cancel): ")

	choice, err := client.ReadLine()
	if err != nil {
		return errors.New("connection closed")
	}
	choice = strings.TrimSpace(choice)

	var charIndex int
	if _, err := fmt.Sscanf(choice, "%d", &charIndex); err != nil || charIndex < 0 || charIndex > len(characters) {
		client.WriteLine("Invalid choice.\n")
		return errors.New("invalid choice")
	}

	if charIndex == 0 {
		client.WriteLine("Deletion cancelled.\n")
		return nil
	}

	selected := characters[charIndex-1]

	// Confirm deletion
	client.WriteLine(fmt.Sprintf("\nAre you sure you want to delete '%s'? This cannot be undone.\n", selected.Name))
	client.WriteLine("Type the character name to confirm: ")

	confirm, err := client.ReadLine()
	if err != nil {
		return errors.New("connection closed")
	}
	confirm = strings.TrimSpace(confirm)

	if !strings.EqualFold(confirm, selected.Name) {
		client.WriteLine("Name does not match. Deletion cancelled.\n")
		return errors.New("confirmation failed")
	}

	// Delete character
	if err := s.db.DeleteCharacter(selected.ID); err != nil {
		client.WriteLine("An error occurred. Please try again.\n")
		return err
	}

	logger.Info("Character deleted",
		"character", selected.Name,
		"character_id", selected.ID,
		"account_id", account.ID,
		"level", selected.Level,
		"event", "character_delete")

	client.WriteLine(fmt.Sprintf("Character '%s' has been deleted.\n", selected.Name))
	return nil
}

// IsCharacterOnline checks if a character is currently logged in
func (s *Server) IsCharacterOnline(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for playerName := range s.clients {
		if strings.EqualFold(playerName, name) {
			return true
		}
	}
	return false
}

// getStatRecommendation returns stat recommendations for a given class
func getStatRecommendation(className string) string {
	t := text.GetInstance()
	if t != nil {
		return t.GetStatRecommendation(className)
	}
	return "  No specific recommendations."
}
