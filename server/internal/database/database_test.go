package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify tables exist by running a simple query
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query accounts table: %v", err)
	}

	err = db.db.QueryRow("SELECT COUNT(*) FROM characters").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query characters table: %v", err)
	}

	err = db.db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query inventory table: %v", err)
	}

	err = db.db.QueryRow("SELECT COUNT(*) FROM equipment").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query equipment table: %v", err)
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

	db, err := Open(nestedPath)
	if err != nil {
		t.Fatalf("Failed to open database with nested path: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("Database file was not created in nested directory")
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Verify database is closed by trying to query
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	if err == nil {
		t.Error("Expected error querying closed database")
	}
}

// TestMigration_AccountsTableSchema verifies the accounts table has correct schema
func TestMigration_AccountsTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check that all expected columns exist
	columns := []string{"id", "username", "password_hash", "created_at", "last_login", "last_ip", "banned", "is_admin"}
	for _, col := range columns {
		var exists int
		err := db.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name = ?", col).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", col, err)
		}
		if exists == 0 {
			t.Errorf("Column %s not found in accounts table", col)
		}
	}
}

// TestMigration_CharactersTableSchema verifies the characters table has correct schema
func TestMigration_CharactersTableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check core columns
	columns := []string{
		"id", "account_id", "name", "room_id", "health", "max_health",
		"mana", "max_mana", "level", "experience", "state",
		"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma",
		"gold", "key_ring", "primary_class", "class_levels", "active_class",
		"race", "crafting_skills", "known_recipes",
		"quest_log", "quest_inventory", "earned_titles", "active_title",
	}

	for _, col := range columns {
		var exists int
		err := db.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('characters') WHERE name = ?", col).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", col, err)
		}
		if exists == 0 {
			t.Errorf("Column %s not found in characters table", col)
		}
	}
}

// TestMigration_IndexesExist verifies that performance indexes are created
func TestMigration_IndexesExist(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	indexes := []string{
		"idx_characters_account_id",
		"idx_inventory_character_id",
		"idx_equipment_character_id",
	}

	for _, idx := range indexes {
		var exists int
		err := db.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check index %s: %v", idx, err)
		}
		if exists == 0 {
			t.Errorf("Index %s not found", idx)
		}
	}
}

// TestMigration_ForeignKeysEnabled verifies foreign keys are enforced
func TestMigration_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check if foreign keys are enabled
	var fkEnabled int
	err = db.db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("Foreign keys are not enabled")
	}
}

// TestMigration_WALModeEnabled verifies WAL journal mode is set
func TestMigration_WALModeEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check journal mode
	var journalMode string
	err = db.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to check journal_mode pragma: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected WAL mode, got %s", journalMode)
	}
}

// TestMigration_Idempotent verifies migrations can be run multiple times safely
func TestMigration_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database first time
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database first time: %v", err)
	}

	// Insert some data
	_, err = db1.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('testuser', 'hash')")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	db1.Close()

	// Open database second time (should re-run migrations without error)
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database second time: %v", err)
	}
	defer db2.Close()

	// Verify data is preserved
	var username string
	err = db2.db.QueryRow("SELECT username FROM accounts WHERE username = 'testuser'").Scan(&username)
	if err != nil {
		t.Errorf("Failed to query inserted data: %v", err)
	}
	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
}

// TestMigration_DefaultValues verifies default values are set correctly
func TestMigration_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create an account
	result, err := db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('testuser', 'hash')")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	accountID, _ := result.LastInsertId()

	// Create a character with minimal fields
	_, err = db.db.Exec("INSERT INTO characters (account_id, name) VALUES (?, 'TestChar')", accountID)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Verify defaults
	var health, maxHealth, level, gold int
	var roomID, state, primaryClass string
	err = db.db.QueryRow(`
		SELECT health, max_health, level, gold, room_id, state, primary_class
		FROM characters WHERE name = 'TestChar'
	`).Scan(&health, &maxHealth, &level, &gold, &roomID, &state, &primaryClass)
	if err != nil {
		t.Fatalf("Failed to query character: %v", err)
	}

	if health != 100 {
		t.Errorf("Expected default health 100, got %d", health)
	}
	if maxHealth != 100 {
		t.Errorf("Expected default max_health 100, got %d", maxHealth)
	}
	if level != 1 {
		t.Errorf("Expected default level 1, got %d", level)
	}
	if gold != 20 {
		t.Errorf("Expected default gold 20, got %d", gold)
	}
	if roomID != "town_square" {
		t.Errorf("Expected default room_id 'town_square', got '%s'", roomID)
	}
	if state != "standing" {
		t.Errorf("Expected default state 'standing', got '%s'", state)
	}
	if primaryClass != "warrior" {
		t.Errorf("Expected default primary_class 'warrior', got '%s'", primaryClass)
	}
}

// TestMigration_ForeignKeyConstraint verifies foreign key constraints work
func TestMigration_ForeignKeyConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Try to insert character with non-existent account_id
	_, err = db.db.Exec("INSERT INTO characters (account_id, name) VALUES (99999, 'OrphanChar')")
	if err == nil {
		t.Error("Expected foreign key constraint error, but insert succeeded")
	}
}

// TestMigration_UniqueConstraints verifies unique constraints work
func TestMigration_UniqueConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create an account
	_, err = db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('uniqueuser', 'hash')")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Try to create duplicate account
	_, err = db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('uniqueuser', 'hash2')")
	if err == nil {
		t.Error("Expected unique constraint error for duplicate username, but insert succeeded")
	}
}

// TestMigration_CaseInsensitiveCollation verifies NOCASE collation works
func TestMigration_CaseInsensitiveCollation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create account with lowercase
	_, err = db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('casetest', 'hash')")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Try to create with different case - should fail due to NOCASE collation
	_, err = db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('CaseTest', 'hash2')")
	if err == nil {
		t.Error("Expected unique constraint error for case-insensitive duplicate, but insert succeeded")
	}
}

// TestMigration_CascadeDelete verifies ON DELETE CASCADE works
func TestMigration_CascadeDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create account and character
	result, err := db.db.Exec("INSERT INTO accounts (username, password_hash) VALUES ('cascadetest', 'hash')")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	accountID, _ := result.LastInsertId()

	charResult, err := db.db.Exec("INSERT INTO characters (account_id, name) VALUES (?, 'CascadeChar')", accountID)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	charID, _ := charResult.LastInsertId()

	// Add inventory item
	_, err = db.db.Exec("INSERT INTO inventory (character_id, item_id) VALUES (?, 'test_item')", charID)
	if err != nil {
		t.Fatalf("Failed to add inventory item: %v", err)
	}

	// Delete account
	_, err = db.db.Exec("DELETE FROM accounts WHERE id = ?", accountID)
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify character was cascade deleted
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM characters WHERE id = ?", charID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check character: %v", err)
	}
	if count != 0 {
		t.Error("Character should have been cascade deleted")
	}

	// Verify inventory was cascade deleted
	err = db.db.QueryRow("SELECT COUNT(*) FROM inventory WHERE character_id = ?", charID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check inventory: %v", err)
	}
	if count != 0 {
		t.Error("Inventory should have been cascade deleted")
	}
}
