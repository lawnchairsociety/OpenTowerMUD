package database

import (
	"errors"
	"testing"
)

func TestCreateCharacter(t *testing.T) {
	db := setupTestDB(t)

	// Create account first
	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create character
	char, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	if char.ID == 0 {
		t.Error("Character ID should not be 0")
	}
	if char.Name != "TestHero" {
		t.Errorf("Expected name 'TestHero', got '%s'", char.Name)
	}
	if char.AccountID != account.ID {
		t.Errorf("Expected account ID %d, got %d", account.ID, char.AccountID)
	}
	if char.RoomID != "town_square" {
		t.Errorf("Expected room 'town_square', got '%s'", char.RoomID)
	}
	if char.Level != 1 {
		t.Errorf("Expected level 1, got %d", char.Level)
	}
	if char.Health != 100 || char.MaxHealth != 100 {
		t.Errorf("Expected health 100/100, got %d/%d", char.Health, char.MaxHealth)
	}
}

func TestCreateCharacterDuplicate(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create first character
	_, err = db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}

	// Try to create duplicate name
	_, err = db.CreateCharacter(account.ID, "TestHero")
	if !errors.Is(err, ErrCharacterExists) {
		t.Errorf("Expected ErrCharacterExists, got: %v", err)
	}
}

func TestCreateCharacterCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	_, err = db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create first character: %v", err)
	}

	// Try to create with different case
	_, err = db.CreateCharacter(account.ID, "testhero")
	if !errors.Is(err, ErrCharacterExists) {
		t.Errorf("Expected ErrCharacterExists for case-insensitive duplicate, got: %v", err)
	}
}

func TestGetCharactersByAccount(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create multiple characters
	_, err = db.CreateCharacter(account.ID, "Hero1")
	if err != nil {
		t.Fatalf("Failed to create character 1: %v", err)
	}
	_, err = db.CreateCharacter(account.ID, "Hero2")
	if err != nil {
		t.Fatalf("Failed to create character 2: %v", err)
	}

	// Get characters
	chars, err := db.GetCharactersByAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to get characters: %v", err)
	}

	if len(chars) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(chars))
	}
}

func TestGetCharactersByAccountEmpty(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	chars, err := db.GetCharactersByAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to get characters: %v", err)
	}

	if len(chars) != 0 {
		t.Errorf("Expected 0 characters, got %d", len(chars))
	}
}

func TestGetCharacterByName(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	created, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Get by name
	char, err := db.GetCharacterByName("TestHero")
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	if char.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, char.ID)
	}
}

func TestGetCharacterByNameNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetCharacterByName("nonexistent")
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("Expected ErrCharacterNotFound, got: %v", err)
	}
}

func TestGetCharacterByID(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	created, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Get by ID
	char, err := db.GetCharacterByID(created.ID)
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	if char.Name != "TestHero" {
		t.Errorf("Expected name 'TestHero', got '%s'", char.Name)
	}
}

func TestSaveCharacter(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	char, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Modify character
	char.Health = 50
	char.Mana = 75
	char.Level = 5
	char.Experience = 1000
	char.RoomID = "forest_clearing"
	char.State = "resting"

	// Save
	err = db.SaveCharacter(char)
	if err != nil {
		t.Fatalf("Failed to save character: %v", err)
	}

	// Reload and verify
	loaded, err := db.GetCharacterByID(char.ID)
	if err != nil {
		t.Fatalf("Failed to reload character: %v", err)
	}

	if loaded.Health != 50 {
		t.Errorf("Expected health 50, got %d", loaded.Health)
	}
	if loaded.Mana != 75 {
		t.Errorf("Expected mana 75, got %d", loaded.Mana)
	}
	if loaded.Level != 5 {
		t.Errorf("Expected level 5, got %d", loaded.Level)
	}
	if loaded.Experience != 1000 {
		t.Errorf("Expected experience 1000, got %d", loaded.Experience)
	}
	if loaded.RoomID != "forest_clearing" {
		t.Errorf("Expected room 'forest_clearing', got '%s'", loaded.RoomID)
	}
	if loaded.State != "resting" {
		t.Errorf("Expected state 'resting', got '%s'", loaded.State)
	}
}

func TestDeleteCharacter(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	char, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Delete
	err = db.DeleteCharacter(char.ID)
	if err != nil {
		t.Fatalf("Failed to delete character: %v", err)
	}

	// Verify deleted
	_, err = db.GetCharacterByID(char.ID)
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("Expected ErrCharacterNotFound after delete, got: %v", err)
	}
}

func TestDeleteCharacterNotFound(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteCharacter(99999)
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("Expected ErrCharacterNotFound, got: %v", err)
	}
}

func TestCharacterNameExists(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Check nonexistent
	exists, err := db.CharacterNameExists("TestHero")
	if err != nil {
		t.Fatalf("Error checking character: %v", err)
	}
	if exists {
		t.Error("Character should not exist")
	}

	// Create character
	_, err = db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Check exists
	exists, err = db.CharacterNameExists("TestHero")
	if err != nil {
		t.Fatalf("Error checking character: %v", err)
	}
	if !exists {
		t.Error("Character should exist")
	}
}

func TestCharacterBelongsToAccount(t *testing.T) {
	db := setupTestDB(t)

	account1, _ := db.CreateAccount("user1", "password123")
	account2, _ := db.CreateAccount("user2", "password123")

	char, _ := db.CreateCharacter(account1.ID, "TestHero")

	// Check ownership
	belongs, err := db.CharacterBelongsToAccount(char.ID, account1.ID)
	if err != nil {
		t.Fatalf("Error checking ownership: %v", err)
	}
	if !belongs {
		t.Error("Character should belong to account1")
	}

	belongs, err = db.CharacterBelongsToAccount(char.ID, account2.ID)
	if err != nil {
		t.Fatalf("Error checking ownership: %v", err)
	}
	if belongs {
		t.Error("Character should not belong to account2")
	}
}

func TestCreateCharacterWithStats(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create character with custom ability scores
	char, err := db.CreateCharacterWithStats(account.ID, "Warrior", 15, 14, 13, 12, 10, 8)
	if err != nil {
		t.Fatalf("Failed to create character with stats: %v", err)
	}

	if char.Strength != 15 {
		t.Errorf("Expected Strength 15, got %d", char.Strength)
	}
	if char.Dexterity != 14 {
		t.Errorf("Expected Dexterity 14, got %d", char.Dexterity)
	}
	if char.Constitution != 13 {
		t.Errorf("Expected Constitution 13, got %d", char.Constitution)
	}
	if char.Intelligence != 12 {
		t.Errorf("Expected Intelligence 12, got %d", char.Intelligence)
	}
	if char.Wisdom != 10 {
		t.Errorf("Expected Wisdom 10, got %d", char.Wisdom)
	}
	if char.Charisma != 8 {
		t.Errorf("Expected Charisma 8, got %d", char.Charisma)
	}
}

func TestCreateCharacterDefaultAbilityScores(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create character with default (all 10s)
	char, err := db.CreateCharacter(account.ID, "DefaultHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// All scores should be 10
	if char.Strength != 10 {
		t.Errorf("Expected default Strength 10, got %d", char.Strength)
	}
	if char.Dexterity != 10 {
		t.Errorf("Expected default Dexterity 10, got %d", char.Dexterity)
	}
	if char.Constitution != 10 {
		t.Errorf("Expected default Constitution 10, got %d", char.Constitution)
	}
	if char.Intelligence != 10 {
		t.Errorf("Expected default Intelligence 10, got %d", char.Intelligence)
	}
	if char.Wisdom != 10 {
		t.Errorf("Expected default Wisdom 10, got %d", char.Wisdom)
	}
	if char.Charisma != 10 {
		t.Errorf("Expected default Charisma 10, got %d", char.Charisma)
	}
}

func TestSaveCharacterAbilityScores(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	char, err := db.CreateCharacter(account.ID, "TestHero")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Modify ability scores
	char.Strength = 18
	char.Dexterity = 16
	char.Constitution = 14
	char.Intelligence = 12
	char.Wisdom = 10
	char.Charisma = 8

	// Save
	err = db.SaveCharacter(char)
	if err != nil {
		t.Fatalf("Failed to save character: %v", err)
	}

	// Reload and verify
	loaded, err := db.GetCharacterByID(char.ID)
	if err != nil {
		t.Fatalf("Failed to reload character: %v", err)
	}

	if loaded.Strength != 18 {
		t.Errorf("Expected Strength 18, got %d", loaded.Strength)
	}
	if loaded.Dexterity != 16 {
		t.Errorf("Expected Dexterity 16, got %d", loaded.Dexterity)
	}
	if loaded.Constitution != 14 {
		t.Errorf("Expected Constitution 14, got %d", loaded.Constitution)
	}
	if loaded.Intelligence != 12 {
		t.Errorf("Expected Intelligence 12, got %d", loaded.Intelligence)
	}
	if loaded.Wisdom != 10 {
		t.Errorf("Expected Wisdom 10, got %d", loaded.Wisdom)
	}
	if loaded.Charisma != 8 {
		t.Errorf("Expected Charisma 8, got %d", loaded.Charisma)
	}
}

func TestGetCharactersByAccountIncludesAbilityScores(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create character with custom stats
	_, err = db.CreateCharacterWithStats(account.ID, "Fighter", 15, 14, 13, 12, 10, 8)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Get characters and verify stats are loaded
	chars, err := db.GetCharactersByAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to get characters: %v", err)
	}

	if len(chars) != 1 {
		t.Fatalf("Expected 1 character, got %d", len(chars))
	}

	char := chars[0]
	if char.Strength != 15 {
		t.Errorf("Expected Strength 15, got %d", char.Strength)
	}
	if char.Charisma != 8 {
		t.Errorf("Expected Charisma 8, got %d", char.Charisma)
	}
}
