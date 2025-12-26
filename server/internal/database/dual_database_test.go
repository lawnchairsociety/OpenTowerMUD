package database

import (
	"fmt"
	"path/filepath"
	"testing"
)

// getDualTestDatabases returns both SQLite and PostgreSQL databases for testing.
// If PostgreSQL is not available, it returns only SQLite.
func getDualTestDatabases(t *testing.T) map[string]*Database {
	dbs := make(map[string]*Database)

	// Always include SQLite
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sqliteDB, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}
	dbs["sqlite"] = sqliteDB

	// Include PostgreSQL if available
	pgConfig := getPostgresTestConfig()
	if pgConfig != nil {
		pgDB, err := OpenWithConfig(*pgConfig)
		if err != nil {
			t.Logf("PostgreSQL not available: %v", err)
		} else {
			// Clean up PostgreSQL tables
			tables := []string{
				"mail_items", "mail", "equipment", "inventory",
				"characters", "boss_kills", "web_sessions", "accounts",
			}
			for _, table := range tables {
				pgDB.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
			}
			dbs["postgres"] = pgDB
		}
	}

	t.Cleanup(func() {
		for name, db := range dbs {
			if name == "postgres" {
				// Clean up PostgreSQL tables before closing
				tables := []string{
					"mail_items", "mail", "equipment", "inventory",
					"characters", "boss_kills", "web_sessions", "accounts",
				}
				for _, table := range tables {
					db.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
				}
			}
			db.Close()
		}
	})

	return dbs
}

// TestDual_CreateAccount tests account creation on both databases
func TestDual_CreateAccount(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("testuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			if account.ID == 0 {
				t.Error("Account ID should not be 0")
			}
			if account.Username != "testuser" {
				t.Errorf("Expected username 'testuser', got '%s'", account.Username)
			}
			if account.PasswordHash == "" {
				t.Error("Password hash should not be empty")
			}
		})
	}
}

// TestDual_CaseInsensitiveUsernames tests case-insensitive username handling
func TestDual_CaseInsensitiveUsernames(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			// Create with lowercase
			_, err := db.CreateAccount("caseuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Try to create with different case - should fail
			_, err = db.CreateAccount("CaseUser", "Password123")
			if err == nil {
				t.Error("Expected error for case-insensitive duplicate")
			}

			// Lookup with different case should succeed
			account, err := db.GetAccountByUsername("CASEUSER")
			if err != nil {
				t.Fatalf("Case-insensitive lookup failed: %v", err)
			}
			if account.Username != "caseuser" {
				t.Errorf("Expected username 'caseuser', got '%s'", account.Username)
			}
		})
	}
}

// TestDual_ValidateLogin tests login validation on both databases
func TestDual_ValidateLogin(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			// Create account
			_, err := db.CreateAccount("loginuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Valid login
			account, err := db.ValidateLogin("loginuser", "Password123", "192.168.1.1")
			if err != nil {
				t.Fatalf("Login failed: %v", err)
			}
			if account.Username != "loginuser" {
				t.Errorf("Expected username 'loginuser', got '%s'", account.Username)
			}

			// Invalid password
			_, err = db.ValidateLogin("loginuser", "WrongPassword", "192.168.1.1")
			if err == nil {
				t.Error("Expected error for wrong password")
			}

			// Case-insensitive login
			account, err = db.ValidateLogin("LOGINUSER", "Password123", "192.168.1.1")
			if err != nil {
				t.Fatalf("Case-insensitive login failed: %v", err)
			}
		})
	}
}

// TestDual_CreateCharacter tests character creation on both databases
func TestDual_CreateCharacter(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			// Create account
			account, err := db.CreateAccount("charowner", "Password123")
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

			// Verify defaults
			if char.Level != 1 {
				t.Errorf("Expected default level 1, got %d", char.Level)
			}
			// Default warrior with 10 CON gets base HP of 10 (d10) + 0 CON modifier = 10
			if char.Health != 10 {
				t.Errorf("Expected default health 10 (warrior base), got %d", char.Health)
			}
		})
	}
}

// TestDual_CaseInsensitiveCharacterNames tests character name case handling
func TestDual_CaseInsensitiveCharacterNames(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("charowner2", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Create with specific case
			_, err = db.CreateCharacter(account.ID, "MyHero")
			if err != nil {
				t.Fatalf("Failed to create character: %v", err)
			}

			// Try to create with different case - should fail
			_, err = db.CreateCharacter(account.ID, "MYHERO")
			if err == nil {
				t.Error("Expected error for case-insensitive duplicate character name")
			}

			// Lookup with different case should work
			char, err := db.GetCharacterByName("myhero")
			if err != nil {
				t.Fatalf("Case-insensitive character lookup failed: %v", err)
			}
			if char.Name != "MyHero" {
				t.Errorf("Expected name 'MyHero', got '%s'", char.Name)
			}
		})
	}
}

// TestDual_SaveAndLoadCharacter tests character persistence on both databases
func TestDual_SaveAndLoadCharacter(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("saveowner", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			char, err := db.CreateCharacter(account.ID, "SaveHero")
			if err != nil {
				t.Fatalf("Failed to create character: %v", err)
			}

			// Modify character
			char.Level = 10
			char.Health = 250
			char.MaxHealth = 300
			char.Gold = 1000
			char.Experience = 5000
			char.RoomID = "dungeon_entrance"

			err = db.SaveCharacter(char)
			if err != nil {
				t.Fatalf("Failed to save character: %v", err)
			}

			// Load and verify
			loaded, err := db.GetCharacterByID(char.ID)
			if err != nil {
				t.Fatalf("Failed to load character: %v", err)
			}

			if loaded.Level != 10 {
				t.Errorf("Expected level 10, got %d", loaded.Level)
			}
			if loaded.Health != 250 {
				t.Errorf("Expected health 250, got %d", loaded.Health)
			}
			if loaded.MaxHealth != 300 {
				t.Errorf("Expected max health 300, got %d", loaded.MaxHealth)
			}
			if loaded.Gold != 1000 {
				t.Errorf("Expected gold 1000, got %d", loaded.Gold)
			}
			if loaded.RoomID != "dungeon_entrance" {
				t.Errorf("Expected room 'dungeon_entrance', got '%s'", loaded.RoomID)
			}
		})
	}
}

// TestDual_Inventory tests inventory operations on both databases
func TestDual_Inventory(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("invowner", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			char, err := db.CreateCharacter(account.ID, "InvHero")
			if err != nil {
				t.Fatalf("Failed to create character: %v", err)
			}

			// Add items using SaveInventory
			err = db.SaveInventory(char.ID, []string{"sword", "potion", "potion"})
			if err != nil {
				t.Fatalf("Failed to save inventory: %v", err)
			}

			// Get inventory
			items, err := db.LoadInventory(char.ID)
			if err != nil {
				t.Fatalf("Failed to load inventory: %v", err)
			}

			if len(items) != 3 {
				t.Errorf("Expected 3 items, got %d", len(items))
			}

			// Count potions
			potionCount := 0
			for _, item := range items {
				if item == "potion" {
					potionCount++
				}
			}
			if potionCount != 2 {
				t.Errorf("Expected 2 potions, got %d", potionCount)
			}

			// Update inventory
			err = db.SaveInventory(char.ID, []string{"sword", "potion", "potion", "potion", "shield"})
			if err != nil {
				t.Fatalf("Failed to update inventory: %v", err)
			}

			// Verify update
			items, _ = db.LoadInventory(char.ID)
			if len(items) != 5 {
				t.Errorf("Expected 5 items after update, got %d", len(items))
			}
		})
	}
}

// TestDual_BanAccount tests account banning on both databases
func TestDual_BanAccount(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("banuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Initially not banned
			banned, err := db.IsAccountBanned(account.ID)
			if err != nil {
				t.Fatalf("Error checking ban: %v", err)
			}
			if banned {
				t.Error("Account should not be banned initially")
			}

			// Ban account
			err = db.BanAccount(account.ID)
			if err != nil {
				t.Fatalf("Failed to ban account: %v", err)
			}

			// Verify banned
			banned, _ = db.IsAccountBanned(account.ID)
			if !banned {
				t.Error("Account should be banned")
			}

			// Login should fail
			_, err = db.ValidateLogin("banuser", "Password123", "192.168.1.1")
			if err == nil {
				t.Error("Banned account should not be able to login")
			}

			// Unban
			err = db.UnbanAccount(account.ID)
			if err != nil {
				t.Fatalf("Failed to unban account: %v", err)
			}

			// Login should work again
			_, err = db.ValidateLogin("banuser", "Password123", "192.168.1.1")
			if err != nil {
				t.Errorf("Unbanned account should be able to login: %v", err)
			}
		})
	}
}

// TestDual_AdminOperations tests admin operations on both databases
func TestDual_AdminOperations(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("adminuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Initially not admin
			isAdmin, err := db.IsAdmin(account.ID)
			if err != nil {
				t.Fatalf("Error checking admin: %v", err)
			}
			if isAdmin {
				t.Error("Account should not be admin initially")
			}

			// Promote to admin
			err = db.SetAdmin(account.ID, true)
			if err != nil {
				t.Fatalf("Failed to set admin: %v", err)
			}

			isAdmin, _ = db.IsAdmin(account.ID)
			if !isAdmin {
				t.Error("Account should be admin after promotion")
			}

			// Demote
			err = db.SetAdmin(account.ID, false)
			if err != nil {
				t.Fatalf("Failed to remove admin: %v", err)
			}

			isAdmin, _ = db.IsAdmin(account.ID)
			if isAdmin {
				t.Error("Account should not be admin after demotion")
			}
		})
	}
}

// TestDual_CascadeDelete tests cascade delete on both databases
func TestDual_CascadeDelete(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount("cascadeuser", "Password123")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			char, err := db.CreateCharacter(account.ID, "CascadeHero")
			if err != nil {
				t.Fatalf("Failed to create character: %v", err)
			}

			// Add inventory
			err = db.SaveInventory(char.ID, []string{"test_item"})
			if err != nil {
				t.Fatalf("Failed to add item: %v", err)
			}

			// Delete account
			query := db.qb.Build("DELETE FROM accounts WHERE id = ?")
			_, err = db.db.Exec(query, account.ID)
			if err != nil {
				t.Fatalf("Failed to delete account: %v", err)
			}

			// Character should be deleted
			_, err = db.GetCharacterByID(char.ID)
			if err == nil {
				t.Error("Character should have been cascade deleted")
			}

			// Inventory should be deleted
			items, _ := db.LoadInventory(char.ID)
			if len(items) != 0 {
				t.Error("Inventory should have been cascade deleted")
			}
		})
	}
}

// TestDual_PasswordHashing tests that passwords are hashed consistently
func TestDual_PasswordHashing(t *testing.T) {
	dbs := getDualTestDatabases(t)

	for name, db := range dbs {
		t.Run(name, func(t *testing.T) {
			_, err := db.CreateAccount("hashuser", "MyPassword123!")
			if err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}

			// Verify login works
			_, err = db.ValidateLogin("hashuser", "MyPassword123!", "192.168.1.1")
			if err != nil {
				t.Fatalf("Login failed: %v", err)
			}

			// Verify wrong password fails
			_, err = db.ValidateLogin("hashuser", "WrongPassword", "192.168.1.1")
			if err == nil {
				t.Error("Wrong password should fail")
			}

			// Change password and verify
			account, _ := db.GetAccountByUsername("hashuser")
			err = db.ChangePassword(account.ID, "NewPassword456!")
			if err != nil {
				t.Fatalf("Failed to change password: %v", err)
			}

			// Old password should fail
			_, err = db.ValidateLogin("hashuser", "MyPassword123!", "192.168.1.1")
			if err == nil {
				t.Error("Old password should fail after change")
			}

			// New password should work
			_, err = db.ValidateLogin("hashuser", "NewPassword456!", "192.168.1.1")
			if err != nil {
				t.Fatalf("New password should work: %v", err)
			}
		})
	}
}
