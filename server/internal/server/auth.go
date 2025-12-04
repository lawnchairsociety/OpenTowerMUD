package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
)

// AuthResult contains the result of the authentication flow
type AuthResult struct {
	Account   *database.Account
	Character *database.Character
}

// handleAuth handles the login/registration flow for a new connection.
// Returns the authenticated account and selected character, or an error.
func (s *Server) handleAuth(conn net.Conn, scanner *bufio.Scanner) (*AuthResult, error) {
	// Welcome screen
	conn.Write([]byte("\n"))
	conn.Write([]byte("=====================================\n"))
	conn.Write([]byte("    Welcome to Open Tower MUD!\n"))
	conn.Write([]byte("=====================================\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte("  [L] Login\n"))
	conn.Write([]byte("  [R] Register\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte("Enter choice: "))

	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

	switch choice {
	case "l", "login":
		return s.handleLogin(conn, scanner)
	case "r", "register":
		return s.handleRegister(conn, scanner)
	default:
		conn.Write([]byte("Invalid choice. Disconnecting.\n"))
		return nil, errors.New("invalid choice")
	}
}

// handleLogin handles the login flow
func (s *Server) handleLogin(conn net.Conn, scanner *bufio.Scanner) (*AuthResult, error) {
	conn.Write([]byte("\n--- Login ---\n"))

	// Get username
	conn.Write([]byte("Username: "))
	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	username := strings.TrimSpace(scanner.Text())
	if username == "" {
		conn.Write([]byte("Username cannot be empty.\n"))
		return nil, errors.New("empty username")
	}

	// Get password
	conn.Write([]byte("Password: "))
	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	password := scanner.Text()

	// Get IP address from connection
	ipAddress := getIPFromConn(conn)

	// Validate credentials
	account, err := s.db.ValidateLogin(username, password, ipAddress)
	if err != nil {
		if errors.Is(err, database.ErrAccountBanned) {
			conn.Write([]byte("\n*** YOUR ACCOUNT HAS BEEN BANNED ***\n"))
			conn.Write([]byte("Contact an administrator if you believe this is an error.\n"))
			return nil, errors.New("account banned")
		}
		if errors.Is(err, database.ErrInvalidCredentials) {
			conn.Write([]byte("Invalid username or password.\n"))
			return nil, errors.New("invalid credentials")
		}
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return nil, err
	}

	conn.Write([]byte(fmt.Sprintf("\nWelcome back, %s!\n", account.Username)))

	// Character selection
	character, err := s.handleCharacterSelection(conn, scanner, account)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Account: account, Character: character}, nil
}

// getIPFromConn extracts the IP address from a connection
func getIPFromConn(conn net.Conn) string {
	addr := conn.RemoteAddr().String()
	// Remove port from address (format is usually "ip:port")
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// handleRegister handles the registration flow
func (s *Server) handleRegister(conn net.Conn, scanner *bufio.Scanner) (*AuthResult, error) {
	conn.Write([]byte("\n--- Register ---\n"))

	// Get username
	conn.Write([]byte("Choose a username: "))
	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	username := strings.TrimSpace(scanner.Text())
	if username == "" {
		conn.Write([]byte("Username cannot be empty.\n"))
		return nil, errors.New("empty username")
	}
	if len(username) < 3 {
		conn.Write([]byte("Username must be at least 3 characters.\n"))
		return nil, errors.New("username too short")
	}
	if len(username) > 20 {
		conn.Write([]byte("Username must be 20 characters or less.\n"))
		return nil, errors.New("username too long")
	}

	// Check if username exists
	exists, err := s.db.AccountExists(username)
	if err != nil {
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return nil, err
	}
	if exists {
		conn.Write([]byte("That username is already taken.\n"))
		return nil, errors.New("username taken")
	}

	// Get password
	conn.Write([]byte("Choose a password (min 4 characters): "))
	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	password := scanner.Text()
	if len(password) < 4 {
		conn.Write([]byte("Password must be at least 4 characters.\n"))
		return nil, errors.New("password too short")
	}

	// Confirm password
	conn.Write([]byte("Confirm password: "))
	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	confirmPassword := scanner.Text()
	if password != confirmPassword {
		conn.Write([]byte("Passwords do not match.\n"))
		return nil, errors.New("password mismatch")
	}

	// Create account
	account, err := s.db.CreateAccount(username, password)
	if err != nil {
		if errors.Is(err, database.ErrAccountExists) {
			conn.Write([]byte("That username is already taken.\n"))
			return nil, errors.New("username taken")
		}
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return nil, err
	}

	conn.Write([]byte(fmt.Sprintf("\nAccount created! Welcome, %s!\n", account.Username)))

	// Go straight to character creation for new accounts
	character, err := s.handleCharacterCreation(conn, scanner, account)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Account: account, Character: character}, nil
}

// handleCharacterSelection handles character selection/creation
func (s *Server) handleCharacterSelection(conn net.Conn, scanner *bufio.Scanner, account *database.Account) (*database.Character, error) {
	for {
		// Get characters for this account
		characters, err := s.db.GetCharactersByAccount(account.ID)
		if err != nil {
			conn.Write([]byte("An error occurred. Please try again.\n"))
			return nil, err
		}

		conn.Write([]byte("\n--- Character Selection ---\n"))

		if len(characters) == 0 {
			conn.Write([]byte("You have no characters.\n"))
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
				conn.Write([]byte(fmt.Sprintf("  [%d] %s - Level %d %s %s\n", i+1, c.Name, c.Level, raceDisplay, classDisplay)))
			}
		}

		conn.Write([]byte("\n  [C] Create new character\n"))
		if len(characters) > 0 {
			conn.Write([]byte("  [D] Delete a character\n"))
		}
		conn.Write([]byte("\nEnter choice: "))

		if !scanner.Scan() {
			return nil, errors.New("connection closed")
		}
		choice := strings.TrimSpace(scanner.Text())

		// Check for create/delete commands
		if strings.ToLower(choice) == "c" {
			character, err := s.handleCharacterCreation(conn, scanner, account)
			if err != nil {
				// Show error but continue the loop
				continue
			}
			return character, nil
		}

		if strings.ToLower(choice) == "d" && len(characters) > 0 {
			if err := s.handleCharacterDeletion(conn, scanner, account); err != nil {
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
						conn.Write([]byte("That character is already logged in.\n"))
						continue
					}

					return selected, nil
				}
			}
		}

		conn.Write([]byte("Invalid choice.\n"))
	}
}

// handleCharacterCreation handles creating a new character
func (s *Server) handleCharacterCreation(conn net.Conn, scanner *bufio.Scanner, account *database.Account) (*database.Character, error) {
	conn.Write([]byte("\n--- Create Character ---\n"))
	conn.Write([]byte("Enter character name: "))

	if !scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	name := strings.TrimSpace(scanner.Text())

	// Validate name
	if name == "" {
		conn.Write([]byte("Character name cannot be empty.\n"))
		return nil, errors.New("empty name")
	}
	if len(name) < 2 {
		conn.Write([]byte("Character name must be at least 2 characters.\n"))
		return nil, errors.New("name too short")
	}
	if len(name) > 20 {
		conn.Write([]byte("Character name must be 20 characters or less.\n"))
		return nil, errors.New("name too long")
	}

	// Check if name exists
	exists, err := s.db.CharacterNameExists(name)
	if err != nil {
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return nil, err
	}
	if exists {
		conn.Write([]byte("That character name is already taken.\n"))
		return nil, errors.New("name taken")
	}

	// Select class
	selectedClass, err := s.handleClassSelection(conn, scanner)
	if err != nil {
		return nil, err
	}

	// Select race
	selectedRace, err := s.handleRaceSelection(conn, scanner)
	if err != nil {
		return nil, err
	}

	// Assign ability scores using the standard array
	scores, err := s.handleAbilityScoreAssignment(conn, scanner, selectedClass, selectedRace)
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
		bonusStat, err := s.handleHumanBonusSelection(conn, scanner, scores)
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
	conn.Write([]byte("\n--- Final Ability Scores ---\n"))
	conn.Write([]byte(fmt.Sprintf("  STR: %d (%+d)\n", scores.Strength, stats.Modifier(scores.Strength))))
	conn.Write([]byte(fmt.Sprintf("  DEX: %d (%+d)\n", scores.Dexterity, stats.Modifier(scores.Dexterity))))
	conn.Write([]byte(fmt.Sprintf("  CON: %d (%+d)\n", scores.Constitution, stats.Modifier(scores.Constitution))))
	conn.Write([]byte(fmt.Sprintf("  INT: %d (%+d)\n", scores.Intelligence, stats.Modifier(scores.Intelligence))))
	conn.Write([]byte(fmt.Sprintf("  WIS: %d (%+d)\n", scores.Wisdom, stats.Modifier(scores.Wisdom))))
	conn.Write([]byte(fmt.Sprintf("  CHA: %d (%+d)\n", scores.Charisma, stats.Modifier(scores.Charisma))))

	// Create character with assigned ability scores, class, and race
	character, err := s.db.CreateCharacterWithClassAndRace(account.ID, name, selectedClass, selectedRace,
		scores.Strength, scores.Dexterity, scores.Constitution,
		scores.Intelligence, scores.Wisdom, scores.Charisma)
	if err != nil {
		if errors.Is(err, database.ErrCharacterExists) {
			conn.Write([]byte("That character name is already taken.\n"))
			return nil, errors.New("name taken")
		}
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return nil, err
	}

	conn.Write([]byte(fmt.Sprintf("\nCharacter '%s' the %s %s created!\n", character.Name, strings.Title(selectedRace), strings.Title(selectedClass))))
	return character, nil
}

// handleClassSelection guides the player through choosing a class
func (s *Server) handleClassSelection(conn net.Conn, scanner *bufio.Scanner) (string, error) {
	conn.Write([]byte("\n--- Choose Your Class ---\n\n"))

	// Display all classes with descriptions
	allClasses := class.AllClasses()
	for i, c := range allClasses {
		def := class.GetDefinition(c)
		if def == nil {
			continue
		}
		conn.Write([]byte(fmt.Sprintf("  [%d] %s\n", i+1, c.String())))
		conn.Write([]byte(fmt.Sprintf("      %s\n", def.Description)))
		conn.Write([]byte(fmt.Sprintf("      Hit Die: d%d | Primary Stat: %s\n\n", def.HitDie, def.PrimaryStat)))
	}

	for {
		conn.Write([]byte("Enter class number (1-6): "))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		input := strings.TrimSpace(scanner.Text())

		// Parse the input
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(allClasses) {
			conn.Write([]byte("Please enter a number from 1 to 6.\n"))
			continue
		}

		selectedClass := allClasses[choice-1]
		def := class.GetDefinition(selectedClass)

		// Confirm selection
		conn.Write([]byte(fmt.Sprintf("\nYou selected: %s\n", selectedClass.String())))
		conn.Write([]byte(fmt.Sprintf("  %s\n", def.Description)))
		conn.Write([]byte("\nIs this correct? (Y/N): "))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if confirm == "y" || confirm == "yes" {
			return string(selectedClass), nil
		}

		conn.Write([]byte("\n"))
	}
}

// handleRaceSelection guides the player through choosing a race
func (s *Server) handleRaceSelection(conn net.Conn, scanner *bufio.Scanner) (string, error) {
	conn.Write([]byte("\n--- Choose Your Race ---\n\n"))

	// Display all races with descriptions
	allRaces := race.AllRaces()
	for i, r := range allRaces {
		def := race.GetDefinition(r)
		if def == nil {
			continue
		}
		conn.Write([]byte(fmt.Sprintf("  [%d] %s (%s)\n", i+1, r.String(), def.Size)))
		conn.Write([]byte(fmt.Sprintf("      %s\n", def.Description)))
		conn.Write([]byte(fmt.Sprintf("      Stat Bonuses: %s\n", def.GetStatBonusesString())))
		conn.Write([]byte(fmt.Sprintf("      Abilities: %s\n\n", def.GetAbilitiesString())))
	}

	for {
		conn.Write([]byte(fmt.Sprintf("Enter race number (1-%d): ", len(allRaces))))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		input := strings.TrimSpace(scanner.Text())

		// Parse the input
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(allRaces) {
			conn.Write([]byte(fmt.Sprintf("Please enter a number from 1 to %d.\n", len(allRaces))))
			continue
		}

		selectedRace := allRaces[choice-1]
		def := race.GetDefinition(selectedRace)

		// Confirm selection
		conn.Write([]byte(fmt.Sprintf("\nYou selected: %s\n", selectedRace.String())))
		conn.Write([]byte(fmt.Sprintf("  %s\n", def.Description)))
		conn.Write([]byte(fmt.Sprintf("  Stat Bonuses: %s\n", def.GetStatBonusesString())))
		conn.Write([]byte("\nIs this correct? (Y/N): "))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if confirm == "y" || confirm == "yes" {
			return string(selectedRace), nil
		}

		conn.Write([]byte("\n"))
	}
}

// handleHumanBonusSelection lets human players choose which stat to increase by +1
func (s *Server) handleHumanBonusSelection(conn net.Conn, scanner *bufio.Scanner, scores *stats.AbilityScores) (string, error) {
	conn.Write([]byte("\n--- Human Versatility ---\n"))
	conn.Write([]byte("As a Human, you may increase one ability score by 1.\n"))
	conn.Write([]byte("Current scores:\n"))
	conn.Write([]byte(fmt.Sprintf("  [1] STR: %d\n", scores.Strength)))
	conn.Write([]byte(fmt.Sprintf("  [2] DEX: %d\n", scores.Dexterity)))
	conn.Write([]byte(fmt.Sprintf("  [3] CON: %d\n", scores.Constitution)))
	conn.Write([]byte(fmt.Sprintf("  [4] INT: %d\n", scores.Intelligence)))
	conn.Write([]byte(fmt.Sprintf("  [5] WIS: %d\n", scores.Wisdom)))
	conn.Write([]byte(fmt.Sprintf("  [6] CHA: %d\n", scores.Charisma)))

	statNames := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

	for {
		conn.Write([]byte("\nWhich stat would you like to increase? (1-6): "))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		input := strings.TrimSpace(scanner.Text())

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > 6 {
			conn.Write([]byte("Please enter a number from 1 to 6.\n"))
			continue
		}

		selectedStat := statNames[choice-1]
		conn.Write([]byte(fmt.Sprintf("\nYou selected +1 %s. Is this correct? (Y/N): ", selectedStat)))

		if !scanner.Scan() {
			return "", errors.New("connection closed")
		}
		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if confirm == "y" || confirm == "yes" {
			return selectedStat, nil
		}
	}
}

// handleAbilityScoreAssignment guides the player through assigning ability scores
func (s *Server) handleAbilityScoreAssignment(conn net.Conn, scanner *bufio.Scanner, selectedClass string, selectedRace string) (*stats.AbilityScores, error) {
	conn.Write([]byte("\n--- Assign Ability Scores ---\n"))
	conn.Write([]byte("You have these values to assign: 15, 14, 13, 12, 10, 8\n"))
	conn.Write([]byte("Each value can only be used once.\n\n"))

	// Show racial bonuses
	raceDef := race.GetDefinition(race.Race(selectedRace))
	if raceDef != nil {
		conn.Write([]byte(fmt.Sprintf("Racial bonuses (%s): %s\n", raceDef.Name.String(), raceDef.GetStatBonusesString())))
	}
	conn.Write([]byte("\n"))

	// Show class-specific recommendations
	conn.Write([]byte(fmt.Sprintf("Recommended stats for %s:\n", strings.Title(selectedClass))))
	conn.Write([]byte(getStatRecommendation(selectedClass)))
	conn.Write([]byte("\n"))

	// Copy the standard array so we can track which values are still available
	available := make([]int, len(stats.StandardArray))
	copy(available, stats.StandardArray)

	// Store the assigned scores
	assigned := make([]int, 6)

	// Go through each ability in order
	for i, abilityName := range stats.AbilityNames {
		for {
			// Show available values
			conn.Write([]byte(fmt.Sprintf("Available: %v\n", available)))
			conn.Write([]byte(fmt.Sprintf("%s: ", abilityName)))

			if !scanner.Scan() {
				return nil, errors.New("connection closed")
			}
			input := strings.TrimSpace(scanner.Text())

			// Parse the input
			value, err := strconv.Atoi(input)
			if err != nil {
				conn.Write([]byte("Please enter a number from the available values.\n"))
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
				conn.Write([]byte("That value is not available. Choose from the remaining values.\n"))
				continue
			}

			// Assign the value and remove from available
			assigned[i] = value
			available = append(available[:found], available[found+1:]...)
			break
		}
	}

	// Show the final assignment
	conn.Write([]byte("\n--- Your Ability Scores ---\n"))
	for i, name := range stats.AbilityNames {
		mod := stats.Modifier(assigned[i])
		modStr := fmt.Sprintf("%+d", mod)
		conn.Write([]byte(fmt.Sprintf("  %s: %d (%s)\n", name, assigned[i], modStr)))
	}

	return stats.NewScores(assigned[0], assigned[1], assigned[2], assigned[3], assigned[4], assigned[5]), nil
}

// handleCharacterDeletion handles deleting a character
func (s *Server) handleCharacterDeletion(conn net.Conn, scanner *bufio.Scanner, account *database.Account) error {
	characters, err := s.db.GetCharactersByAccount(account.ID)
	if err != nil {
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return err
	}

	conn.Write([]byte("\nWhich character do you want to delete?\n"))
	for i, c := range characters {
		conn.Write([]byte(fmt.Sprintf("  [%d] %s (Level %d)\n", i+1, c.Name, c.Level)))
	}
	conn.Write([]byte("Enter number (or 0 to cancel): "))

	if !scanner.Scan() {
		return errors.New("connection closed")
	}
	choice := strings.TrimSpace(scanner.Text())

	var charIndex int
	if _, err := fmt.Sscanf(choice, "%d", &charIndex); err != nil || charIndex < 0 || charIndex > len(characters) {
		conn.Write([]byte("Invalid choice.\n"))
		return errors.New("invalid choice")
	}

	if charIndex == 0 {
		conn.Write([]byte("Deletion cancelled.\n"))
		return nil
	}

	selected := characters[charIndex-1]

	// Confirm deletion
	conn.Write([]byte(fmt.Sprintf("\nAre you sure you want to delete '%s'? This cannot be undone.\n", selected.Name)))
	conn.Write([]byte("Type the character name to confirm: "))

	if !scanner.Scan() {
		return errors.New("connection closed")
	}
	confirm := strings.TrimSpace(scanner.Text())

	if !strings.EqualFold(confirm, selected.Name) {
		conn.Write([]byte("Name does not match. Deletion cancelled.\n"))
		return errors.New("confirmation failed")
	}

	// Delete character
	if err := s.db.DeleteCharacter(selected.ID); err != nil {
		conn.Write([]byte("An error occurred. Please try again.\n"))
		return err
	}

	conn.Write([]byte(fmt.Sprintf("Character '%s' has been deleted.\n", selected.Name)))
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
	switch className {
	case "warrior":
		return "  Primary: STR (attack/damage) | Secondary: CON (HP)\n  Suggested: STR 15, CON 14, DEX 13"
	case "mage":
		return "  Primary: INT (spellcasting) | Secondary: CON (HP)\n  Suggested: INT 15, CON 14, DEX 13"
	case "cleric":
		return "  Primary: WIS (spellcasting) | Secondary: CON (HP)\n  Suggested: WIS 15, CON 14, STR 13"
	case "rogue":
		return "  Primary: DEX (attack/damage) | Secondary: CON (HP)\n  Suggested: DEX 15, CON 14, INT 13"
	case "ranger":
		return "  Primary: DEX (attack) | Secondary: WIS (spells), CON (HP)\n  Suggested: DEX 15, WIS 14, CON 13"
	case "paladin":
		return "  Primary: STR (attack) | Secondary: CHA (spells), CON (HP)\n  Suggested: STR 15, CHA 14, CON 13"
	default:
		return "  No specific recommendations."
	}
}
