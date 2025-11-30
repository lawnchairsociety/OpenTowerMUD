package database

import (
	"errors"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *Database {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestCreateAccount(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	account, err := db.CreateAccount("testuser", "password123")
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
	if account.PasswordHash == "password123" {
		t.Error("Password should be hashed, not stored in plain text")
	}
}

func TestCreateAccountDuplicate(t *testing.T) {
	db := setupTestDB(t)

	// Create first account
	_, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create first account: %v", err)
	}

	// Try to create duplicate
	_, err = db.CreateAccount("testuser", "differentpass")
	if !errors.Is(err, ErrAccountExists) {
		t.Errorf("Expected ErrAccountExists, got: %v", err)
	}
}

func TestCreateAccountCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)

	// Create first account
	_, err := db.CreateAccount("TestUser", "password123")
	if err != nil {
		t.Fatalf("Failed to create first account: %v", err)
	}

	// Try to create with different case
	_, err = db.CreateAccount("testuser", "password123")
	if !errors.Is(err, ErrAccountExists) {
		t.Errorf("Expected ErrAccountExists for case-insensitive duplicate, got: %v", err)
	}
}

func TestCreateAccountEmptyUsername(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.CreateAccount("", "password123")
	if err == nil {
		t.Error("Expected error for empty username")
	}
}

func TestCreateAccountShortPassword(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.CreateAccount("testuser", "abc")
	if err == nil {
		t.Error("Expected error for short password")
	}
}

func TestValidateLogin(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	_, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Valid login
	account, err := db.ValidateLogin("testuser", "password123", "192.168.1.1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if account.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", account.Username)
	}
}

func TestValidateLoginWrongPassword(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	_, err = db.ValidateLogin("testuser", "wrongpassword", "192.168.1.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidateLoginNonexistentUser(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.ValidateLogin("nonexistent", "password123", "192.168.1.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestValidateLoginCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.CreateAccount("TestUser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Login with different case
	account, err := db.ValidateLogin("testuser", "password123", "192.168.1.1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if account.Username != "TestUser" {
		t.Errorf("Expected username 'TestUser', got '%s'", account.Username)
	}
}

func TestValidateLoginRecordsIP(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Login with IP
	_, err = db.ValidateLogin("testuser", "password123", "10.0.0.42")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Check IP was recorded
	account, err := db.GetAccountByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}
	if account.LastIP != "10.0.0.42" {
		t.Errorf("Expected last IP '10.0.0.42', got '%s'", account.LastIP)
	}
}

func TestValidateLoginBannedAccount(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Ban the account
	err = db.BanAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to ban account: %v", err)
	}

	// Try to login
	_, err = db.ValidateLogin("testuser", "password123", "192.168.1.1")
	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("Expected ErrAccountBanned, got: %v", err)
	}
}

func TestBanAndUnbanAccount(t *testing.T) {
	db := setupTestDB(t)

	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Initially not banned
	banned, err := db.IsAccountBanned(account.ID)
	if err != nil {
		t.Fatalf("Error checking ban status: %v", err)
	}
	if banned {
		t.Error("Account should not be banned initially")
	}

	// Ban account
	err = db.BanAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to ban account: %v", err)
	}

	banned, err = db.IsAccountBanned(account.ID)
	if err != nil {
		t.Fatalf("Error checking ban status: %v", err)
	}
	if !banned {
		t.Error("Account should be banned")
	}

	// Unban account
	err = db.UnbanAccount(account.ID)
	if err != nil {
		t.Fatalf("Failed to unban account: %v", err)
	}

	banned, err = db.IsAccountBanned(account.ID)
	if err != nil {
		t.Fatalf("Error checking ban status: %v", err)
	}
	if banned {
		t.Error("Account should not be banned after unban")
	}

	// Should be able to login after unban
	_, err = db.ValidateLogin("testuser", "password123", "192.168.1.1")
	if err != nil {
		t.Errorf("Should be able to login after unban: %v", err)
	}
}

func TestGetAccountByUsername(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	created, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Get account
	account, err := db.GetAccountByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if account.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, account.ID)
	}
}

func TestGetAccountByUsernameNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetAccountByUsername("nonexistent")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("Expected ErrAccountNotFound, got: %v", err)
	}
}

func TestAccountExists(t *testing.T) {
	db := setupTestDB(t)

	// Check nonexistent
	exists, err := db.AccountExists("testuser")
	if err != nil {
		t.Fatalf("Error checking account: %v", err)
	}
	if exists {
		t.Error("Account should not exist")
	}

	// Create account
	_, err = db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Check exists
	exists, err = db.AccountExists("testuser")
	if err != nil {
		t.Fatalf("Error checking account: %v", err)
	}
	if !exists {
		t.Error("Account should exist")
	}
}

func TestChangePassword(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	account, err := db.CreateAccount("testuser", "oldpassword")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Change password
	err = db.ChangePassword(account.ID, "newpassword")
	if err != nil {
		t.Fatalf("Failed to change password: %v", err)
	}

	// Verify old password no longer works
	_, err = db.ValidateLogin("testuser", "oldpassword", "192.168.1.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Error("Old password should not work")
	}

	// Verify new password works
	_, err = db.ValidateLogin("testuser", "newpassword", "192.168.1.1")
	if err != nil {
		t.Errorf("New password should work: %v", err)
	}
}

func TestChangePasswordWithVerify(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	account, err := db.CreateAccount("testuser", "oldpassword")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Change password with correct old password
	err = db.ChangePasswordWithVerify(account.ID, "oldpassword", "newpassword")
	if err != nil {
		t.Fatalf("Failed to change password: %v", err)
	}

	// Verify new password works
	_, err = db.ValidateLogin("testuser", "newpassword", "192.168.1.1")
	if err != nil {
		t.Errorf("New password should work: %v", err)
	}
}

func TestChangePasswordWithVerifyWrongOldPassword(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	account, err := db.CreateAccount("testuser", "oldpassword")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Try to change password with wrong old password
	err = db.ChangePasswordWithVerify(account.ID, "wrongpassword", "newpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}

	// Verify original password still works
	_, err = db.ValidateLogin("testuser", "oldpassword", "192.168.1.1")
	if err != nil {
		t.Errorf("Original password should still work: %v", err)
	}
}

// Admin Tests

func TestSetAndCheckAdmin(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	account, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Initially not admin
	isAdmin, err := db.IsAdmin(account.ID)
	if err != nil {
		t.Fatalf("Error checking admin status: %v", err)
	}
	if isAdmin {
		t.Error("Account should not be admin initially")
	}

	// Promote to admin
	err = db.SetAdmin(account.ID, true)
	if err != nil {
		t.Fatalf("Failed to set admin: %v", err)
	}

	isAdmin, err = db.IsAdmin(account.ID)
	if err != nil {
		t.Fatalf("Error checking admin status: %v", err)
	}
	if !isAdmin {
		t.Error("Account should be admin after promotion")
	}

	// Demote from admin
	err = db.SetAdmin(account.ID, false)
	if err != nil {
		t.Fatalf("Failed to remove admin: %v", err)
	}

	isAdmin, err = db.IsAdmin(account.ID)
	if err != nil {
		t.Fatalf("Error checking admin status: %v", err)
	}
	if isAdmin {
		t.Error("Account should not be admin after demotion")
	}
}

func TestGetAccountByUsernameIncludesAdmin(t *testing.T) {
	db := setupTestDB(t)

	// Create account and set as admin
	account, err := db.CreateAccount("adminuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	err = db.SetAdmin(account.ID, true)
	if err != nil {
		t.Fatalf("Failed to set admin: %v", err)
	}

	// Get account and check IsAdmin field
	retrieved, err := db.GetAccountByUsername("adminuser")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if !retrieved.IsAdmin {
		t.Error("Retrieved account should have IsAdmin=true")
	}
}

func TestGetAllAdmins(t *testing.T) {
	db := setupTestDB(t)

	// Create several accounts
	admin1, _ := db.CreateAccount("admin1", "password123")
	admin2, _ := db.CreateAccount("admin2", "password123")
	_, _ = db.CreateAccount("regularuser", "password123")

	// Set some as admin
	db.SetAdmin(admin1.ID, true)
	db.SetAdmin(admin2.ID, true)

	// Get all admins
	admins, err := db.GetAllAdmins()
	if err != nil {
		t.Fatalf("Failed to get admins: %v", err)
	}

	if len(admins) != 2 {
		t.Errorf("Expected 2 admins, got %d", len(admins))
	}

	// Verify both are admins
	foundAdmin1, foundAdmin2 := false, false
	for _, a := range admins {
		if a.Username == "admin1" {
			foundAdmin1 = true
		}
		if a.Username == "admin2" {
			foundAdmin2 = true
		}
	}

	if !foundAdmin1 || !foundAdmin2 {
		t.Error("Not all admins found in GetAllAdmins result")
	}
}

func TestGetAccountByID(t *testing.T) {
	db := setupTestDB(t)

	// Create account
	created, err := db.CreateAccount("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Get by ID
	account, err := db.GetAccountByID(created.ID)
	if err != nil {
		t.Fatalf("Failed to get account by ID: %v", err)
	}

	if account.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", account.Username)
	}
}

func TestGetAccountByIDNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetAccountByID(99999)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("Expected ErrAccountNotFound, got: %v", err)
	}
}

func TestGetTotalAccounts(t *testing.T) {
	db := setupTestDB(t)

	// Initially 0
	count, err := db.GetTotalAccounts()
	if err != nil {
		t.Fatalf("Failed to get account count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 accounts, got %d", count)
	}

	// Create some accounts
	db.CreateAccount("user1", "password123")
	db.CreateAccount("user2", "password123")
	db.CreateAccount("user3", "password123")

	count, err = db.GetTotalAccounts()
	if err != nil {
		t.Fatalf("Failed to get account count: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 accounts, got %d", count)
	}
}

func TestGetTotalCharacters(t *testing.T) {
	db := setupTestDB(t)

	// Initially 0
	count, err := db.GetTotalCharacters()
	if err != nil {
		t.Fatalf("Failed to get character count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 characters, got %d", count)
	}

	// Create account and characters
	account, _ := db.CreateAccount("user1", "password123")
	db.CreateCharacter(account.ID, "char1")
	db.CreateCharacter(account.ID, "char2")

	count, err = db.GetTotalCharacters()
	if err != nil {
		t.Fatalf("Failed to get character count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 characters, got %d", count)
	}
}
