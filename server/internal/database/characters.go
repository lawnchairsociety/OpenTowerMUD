package database

import (
	"database/sql"
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
	VisitedPortals  string // Comma-separated list of floor numbers for portal travel
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
	CreatedAt    time.Time
	LastPlayed   *time.Time
}

// DefaultStarterSpells is the list of spells new characters start with.
const DefaultStarterSpells = "heal,flare,dazzle"

// CreateCharacter creates a new character for an account with default ability scores.
func (d *Database) CreateCharacter(accountID int64, name string) (*Character, error) {
	// Create with default ability scores (all 10s)
	return d.CreateCharacterWithStats(accountID, name, 10, 10, 10, 10, 10, 10)
}

// CreateCharacterWithStats creates a new character for an account with specified ability scores.
func (d *Database) CreateCharacterWithStats(accountID int64, name string, str, dex, con, int_, wis, cha int) (*Character, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("character name cannot be empty")
	}

	result, err := d.db.Exec(
		`INSERT INTO characters (account_id, name, learned_spells, strength, dexterity, constitution, intelligence, wisdom, charisma)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, name, DefaultStarterSpells, str, dex, con, int_, wis, cha,
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
		Health:         100,
		MaxHealth:      100,
		Mana:           100,
		MaxMana:        100,
		Level:          1,
		Experience:     0,
		State:          "standing",
		MaxCarryWeight: 100.0,
		LearnedSpells:  DefaultStarterSpells,
		Strength:       str,
		Dexterity:      dex,
		Constitution:   con,
		Intelligence:   int_,
		Wisdom:         wis,
		Charisma:       cha,
		Gold:           20, // Starting gold (matches DB default)
		CreatedAt:      time.Now(),
	}, nil
}

// GetCharactersByAccount returns all characters for an account.
func (d *Database) GetCharactersByAccount(accountID int64) ([]*Character, error) {
	rows, err := d.db.Query(
		`SELECT id, account_id, name, room_id, health, max_health, mana, max_mana,
		        level, experience, state, max_carry_weight, learned_spells,
		        visited_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, created_at, last_played
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
		        visited_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, created_at, last_played
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
		        visited_portals, strength, dexterity, constitution, intelligence, wisdom, charisma,
		        gold, key_ring, created_at, last_played
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
			visited_portals = ?,
			strength = ?,
			dexterity = ?,
			constitution = ?,
			intelligence = ?,
			wisdom = ?,
			charisma = ?,
			gold = ?,
			key_ring = ?,
			last_played = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		c.RoomID, c.Health, c.MaxHealth, c.Mana, c.MaxMana,
		c.Level, c.Experience, c.State, c.MaxCarryWeight, c.LearnedSpells,
		c.VisitedPortals, c.Strength, c.Dexterity, c.Constitution, c.Intelligence, c.Wisdom, c.Charisma,
		c.Gold, c.KeyRing, c.ID,
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
		&c.VisitedPortals, &c.Strength, &c.Dexterity, &c.Constitution, &c.Intelligence, &c.Wisdom, &c.Charisma,
		&c.Gold, &c.KeyRing, &c.CreatedAt, &lastPlayed,
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
		&c.VisitedPortals, &c.Strength, &c.Dexterity, &c.Constitution, &c.Intelligence, &c.Wisdom, &c.Charisma,
		&c.Gold, &c.KeyRing, &c.CreatedAt, &lastPlayed,
	)
	if err != nil {
		return nil, err
	}

	if lastPlayed.Valid {
		c.LastPlayed = &lastPlayed.Time
	}

	return &c, nil
}

