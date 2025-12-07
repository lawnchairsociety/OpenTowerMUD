package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrCharacterNotFound is returned when a character lookup fails.
var ErrCharacterNotFound = errors.New("character not found")

// ErrCharacterExists is returned when trying to create a duplicate character.
var ErrCharacterExists = errors.New("character name already taken")

// Character represents a player character's persistent data.
type Character struct {
	ID              int64
	AccountID       int64
	Name            string
	RoomID          string
	Health          int
	MaxHealth       int
	Mana            int
	MaxMana         int
	Level           int
	Experience      int
	State           string
	MaxCarryWeight  float64
	LearnedSpells   string // Comma-separated list of spell IDs
	DiscoveredPortals string // Comma-separated list of floor numbers for portal travel
	// Ability scores (Phase 25)
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
	// Economy (Phase 27)
	Gold    int
	KeyRing string // Comma-separated list of key item IDs
	// Class system
	PrimaryClass string // Primary class (e.g., "warrior")
	ClassLevels  string // JSON map of class -> level (e.g., '{"warrior":5}')
	ActiveClass  string // Which class currently gains XP
	// Race system
	Race string // Player's race (e.g., "human", "dwarf")
	// Crafting system
	CraftingSkills string // Comma-separated list of skill:level pairs (e.g., "blacksmithing:10,alchemy:25")
	KnownRecipes   string // Comma-separated list of recipe IDs
	// Quest system
	QuestLog       string // JSON-serialized PlayerQuestLog
	QuestInventory string // Comma-separated list of quest item IDs
	EarnedTitles   string // Comma-separated list of earned title IDs
	ActiveTitle    string // Currently displayed title ID
	CreatedAt      time.Time
	LastPlayed     *time.Time
}

// CreateCharacter creates a new character for an account with default ability scores and warrior class.
func (d *Database) CreateCharacter(accountID int64, name string) (*Character, error) {
	// Create with default ability scores (all 10s), warrior class, and human race
	return d.CreateCharacterWithClassAndRace(accountID, name, "warrior", "human", 10, 10, 10, 10, 10, 10)
}

// CreateCharacterWithStats creates a new character for an account with specified ability scores.
// Deprecated: Use CreateCharacterWithClassAndRace instead.
func (d *Database) CreateCharacterWithStats(accountID int64, name string, str, dex, con, int_, wis, cha int) (*Character, error) {
	return d.CreateCharacterWithClassAndRace(accountID, name, "warrior", "human", str, dex, con, int_, wis, cha)
}

// CreateCharacterWithClass creates a new character with specified class and ability scores.
// Deprecated: Use CreateCharacterWithClassAndRace instead.
func (d *Database) CreateCharacterWithClass(accountID int64, name string, primaryClass string, str, dex, con, int_, wis, cha int) (*Character, error) {
	return d.CreateCharacterWithClassAndRace(accountID, name, primaryClass, "human", str, dex, con, int_, wis, cha)
}

// CreateCharacterWithClassAndRace creates a new character with specified class, race, and ability scores.
func (d *Database) CreateCharacterWithClassAndRace(accountID int64, name string, primaryClass string, race string, str, dex, con, int_, wis, cha int) (*Character, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("character name cannot be empty")
	}

	// Validate class
	if primaryClass == "" {
		primaryClass = "warrior"
	}

	// Validate race
	if race == "" {
		race = "human"
	}

	// Build initial class levels JSON using proper marshaling to avoid injection
	classLevelsMap := map[string]int{primaryClass: 1}
	classLevelsBytes, _ := json.Marshal(classLevelsMap)
	classLevels := string(classLevelsBytes)

	// Calculate starting HP/Mana based on class
	startingHP, startingMana := calculateStartingStats(primaryClass, str, dex, con, int_, wis, cha)

	result, err := d.db.Exec(
		`INSERT INTO characters (account_id, name, health, max_health, mana, max_mana,
		                         strength, dexterity, constitution, intelligence, wisdom, charisma,
		                         primary_class, class_levels, active_class, race, learned_spells)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, name, startingHP, startingHP, startingMana, startingMana,
		str, dex, con, int_, wis, cha,
		primaryClass, classLevels, primaryClass, race, "",
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrCharacterExists
		}
		return nil, fmt.Errorf("failed to create character: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get character ID: %w", err)
	}

	return &Character{
		ID:             id,
		AccountID:      accountID,
		Name:           name,
		RoomID:         "town_square",
		Health:         startingHP,
		MaxHealth:      startingHP,
		Mana:           startingMana,
		MaxMana:        startingMana,
		Level:          1,
		Experience:     0,
		State:          "standing",
		MaxCarryWeight: 100.0,
		LearnedSpells:  "",
		Strength:       str,
		Dexterity:      dex,
		Constitution:   con,
		Intelligence:   int_,
		Wisdom:         wis,
		Charisma:       cha,
		Gold:           20, // Starting gold (matches DB default)
		PrimaryClass:   primaryClass,
		ClassLevels:    classLevels,
		ActiveClass:    primaryClass,
		Race:           race,
		CreatedAt:      time.Now(),
	}, nil
}

// calculateStartingStats calculates starting HP and Mana based on class and ability scores.
func calculateStartingStats(primaryClass string, str, dex, con, int_, wis, cha int) (int, int) {
	// Calculate CON modifier
	conMod := (con - 10) / 2

	// Class-specific starting values
	var baseHP, baseMana int
	var castingStatMod int

	switch primaryClass {
	case "warrior":
		baseHP = 10 // d10
		baseMana = 0
		castingStatMod = 0
	case "mage":
		baseHP = 6 // d6
		baseMana = 20
		castingStatMod = (int_ - 10) / 2 // INT modifier
	case "cleric":
		baseHP = 8 // d8
		baseMana = 15
		castingStatMod = (wis - 10) / 2 // WIS modifier
	case "rogue":
		baseHP = 8 // d8
		baseMana = 10
		castingStatMod = (int_ - 10) / 2 // INT modifier
	case "ranger":
		baseHP = 10 // d10
		baseMana = 10
		castingStatMod = (wis - 10) / 2 // WIS modifier
	case "paladin":
		baseHP = 10 // d10
		baseMana = 10
		castingStatMod = (cha - 10) / 2 // CHA modifier
	default:
		baseHP = 10
		baseMana = 0
		castingStatMod = 0
	}

	// Calculate final values
	startingHP := baseHP + conMod
	if startingHP < 1 {
		startingHP = 1
	}

	startingMana := baseMana + castingStatMod
	if startingMana < 0 {
		startingMana = 0
	}

	return startingHP, startingMana
}

// GetCharactersByAccount returns all characters for an account.
func (d *Database) GetCharactersByAccount(accountID int64) ([]*Character, error) {
	rows, err := d.db.Query(
		`SELECT id, account_id, name, room_id, health, max_health, mana, max_mana,
		        level, experience, state, max_carry_weight, learned_spells,
		        discovered_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, primary_class, class_levels, active_class, race,
		        COALESCE(crafting_skills, ''), COALESCE(known_recipes, ''),
		        COALESCE(quest_log, '{}'), COALESCE(quest_inventory, ''),
		        COALESCE(earned_titles, ''), COALESCE(active_title, ''),
		        created_at, last_played
		 FROM characters WHERE account_id = ? ORDER BY last_played DESC NULLS LAST, name`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var characters []*Character
	for rows.Next() {
		c, err := scanCharacter(rows)
		if err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating characters: %w", err)
	}

	return characters, nil
}

// GetCharacterByName retrieves a character by name (case-insensitive).
func (d *Database) GetCharacterByName(name string) (*Character, error) {
	row := d.db.QueryRow(
		`SELECT id, account_id, name, room_id, health, max_health, mana, max_mana,
		        level, experience, state, max_carry_weight, learned_spells,
		        discovered_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, primary_class, class_levels, active_class, race,
		        COALESCE(crafting_skills, ''), COALESCE(known_recipes, ''),
		        COALESCE(quest_log, '{}'), COALESCE(quest_inventory, ''),
		        COALESCE(earned_titles, ''), COALESCE(active_title, ''),
		        created_at, last_played
		 FROM characters WHERE name = ?`,
		name,
	)

	c, err := scanCharacterRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCharacterNotFound
		}
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	return c, nil
}

// GetCharacterByID retrieves a character by ID.
func (d *Database) GetCharacterByID(id int64) (*Character, error) {
	row := d.db.QueryRow(
		`SELECT id, account_id, name, room_id, health, max_health, mana, max_mana,
		        level, experience, state, max_carry_weight, learned_spells,
		        discovered_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, primary_class, class_levels, active_class, race,
		        COALESCE(crafting_skills, ''), COALESCE(known_recipes, ''),
		        COALESCE(quest_log, '{}'), COALESCE(quest_inventory, ''),
		        COALESCE(earned_titles, ''), COALESCE(active_title, ''),
		        created_at, last_played
		 FROM characters WHERE id = ?`,
		id,
	)

	c, err := scanCharacterRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCharacterNotFound
		}
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	return c, nil
}

// SaveCharacter saves all character state to the database.
func (d *Database) SaveCharacter(c *Character) error {
	_, err := d.db.Exec(
		`UPDATE characters SET
			room_id = ?,
			health = ?,
			max_health = ?,
			mana = ?,
			max_mana = ?,
			level = ?,
			experience = ?,
			state = ?,
			max_carry_weight = ?,
			learned_spells = ?,
			discovered_portals = ?,
			strength = ?,
			dexterity = ?,
			constitution = ?,
			intelligence = ?,
			wisdom = ?,
			charisma = ?,
			gold = ?,
			key_ring = ?,
			primary_class = ?,
			class_levels = ?,
			active_class = ?,
			race = ?,
			crafting_skills = ?,
			known_recipes = ?,
			quest_log = ?,
			quest_inventory = ?,
			earned_titles = ?,
			active_title = ?,
			last_played = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		c.RoomID, c.Health, c.MaxHealth, c.Mana, c.MaxMana,
		c.Level, c.Experience, c.State, c.MaxCarryWeight, c.LearnedSpells,
		c.DiscoveredPortals, c.Strength, c.Dexterity, c.Constitution, c.Intelligence, c.Wisdom, c.Charisma,
		c.Gold, c.KeyRing, c.PrimaryClass, c.ClassLevels, c.ActiveClass, c.Race,
		c.CraftingSkills, c.KnownRecipes,
		c.QuestLog, c.QuestInventory, c.EarnedTitles, c.ActiveTitle,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to save character: %w", err)
	}
	return nil
}

// DeleteCharacter removes a character and all associated data.
func (d *Database) DeleteCharacter(characterID int64) error {
	result, err := d.db.Exec("DELETE FROM characters WHERE id = ?", characterID)
	if err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrCharacterNotFound
	}

	return nil
}

// CharacterNameExists checks if a character name is already taken.
func (d *Database) CharacterNameExists(name string) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM characters WHERE name = ?",
		name,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check character existence: %w", err)
	}
	return count > 0, nil
}

// CharacterBelongsToAccount checks if a character belongs to an account.
func (d *Database) CharacterBelongsToAccount(characterID, accountID int64) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM characters WHERE id = ? AND account_id = ?",
		characterID, accountID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check character ownership: %w", err)
	}
	return count > 0, nil
}

// IsCharacterOnline checks if a character name is currently in use.
// This is tracked in-memory by the server, not in the database.
// Placeholder for server-side implementation.
type OnlineChecker interface {
	IsCharacterOnline(name string) bool
}

// scanCharacter scans a character from a *sql.Rows.
func scanCharacter(rows *sql.Rows) (*Character, error) {
	var c Character
	var lastPlayed sql.NullTime

	err := rows.Scan(
		&c.ID, &c.AccountID, &c.Name, &c.RoomID,
		&c.Health, &c.MaxHealth, &c.Mana, &c.MaxMana,
		&c.Level, &c.Experience, &c.State, &c.MaxCarryWeight,
		&c.LearnedSpells,
		&c.DiscoveredPortals, &c.Strength, &c.Dexterity, &c.Constitution, &c.Intelligence, &c.Wisdom, &c.Charisma,
		&c.Gold, &c.KeyRing, &c.PrimaryClass, &c.ClassLevels, &c.ActiveClass, &c.Race,
		&c.CraftingSkills, &c.KnownRecipes,
		&c.QuestLog, &c.QuestInventory, &c.EarnedTitles, &c.ActiveTitle,
		&c.CreatedAt, &lastPlayed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan character: %w", err)
	}

	if lastPlayed.Valid {
		c.LastPlayed = &lastPlayed.Time
	}

	return &c, nil
}

// scanCharacterRow scans a character from a *sql.Row.
func scanCharacterRow(row *sql.Row) (*Character, error) {
	var c Character
	var lastPlayed sql.NullTime

	err := row.Scan(
		&c.ID, &c.AccountID, &c.Name, &c.RoomID,
		&c.Health, &c.MaxHealth, &c.Mana, &c.MaxMana,
		&c.Level, &c.Experience, &c.State, &c.MaxCarryWeight,
		&c.LearnedSpells,
		&c.DiscoveredPortals, &c.Strength, &c.Dexterity, &c.Constitution, &c.Intelligence, &c.Wisdom, &c.Charisma,
		&c.Gold, &c.KeyRing, &c.PrimaryClass, &c.ClassLevels, &c.ActiveClass, &c.Race,
		&c.CraftingSkills, &c.KnownRecipes,
		&c.QuestLog, &c.QuestInventory, &c.EarnedTitles, &c.ActiveTitle,
		&c.CreatedAt, &lastPlayed,
	)
	if err != nil {
		return nil, err
	}

	if lastPlayed.Valid {
		c.LastPlayed = &lastPlayed.Time
	}

	return &c, nil
}

