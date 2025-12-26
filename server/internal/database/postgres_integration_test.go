package database

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// getPostgresTestConfig returns PostgreSQL config if available, nil otherwise.
// Set these environment variables to run PostgreSQL tests:
//
//	OTM_TEST_POSTGRES_HOST (default: localhost)
//	OTM_TEST_POSTGRES_PORT (default: 5435)
//	OTM_TEST_POSTGRES_USER (default: opentower)
//	OTM_TEST_POSTGRES_PASSWORD (default: opentower)
//	OTM_TEST_POSTGRES_DATABASE (default: opentower_test)
func getPostgresTestConfig() *Config {
	// Check if PostgreSQL testing is explicitly enabled
	if os.Getenv("OTM_TEST_POSTGRES") == "" {
		return nil
	}

	host := os.Getenv("OTM_TEST_POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 5435
	if portStr := os.Getenv("OTM_TEST_POSTGRES_PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	user := os.Getenv("OTM_TEST_POSTGRES_USER")
	if user == "" {
		user = "opentower"
	}

	password := os.Getenv("OTM_TEST_POSTGRES_PASSWORD")
	if password == "" {
		password = "opentower"
	}

	database := os.Getenv("OTM_TEST_POSTGRES_DATABASE")
	if database == "" {
		database = "opentower_test"
	}

	return &Config{
		Driver: "postgres",
		Postgres: PostgresConfig{
			Host:            host,
			Port:            port,
			User:            user,
			Password:        password,
			Database:        database,
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Minute,
		},
	}
}

// skipIfNoPostgres skips the test if PostgreSQL is not available
func skipIfNoPostgres(t *testing.T) *Config {
	cfg := getPostgresTestConfig()
	if cfg == nil {
		t.Skip("Skipping PostgreSQL test: OTM_TEST_POSTGRES not set")
	}
	return cfg
}

// setupPostgresTestDB opens a PostgreSQL connection for testing and clears test data
func setupPostgresTestDB(t *testing.T, cfg *Config) *Database {
	db, err := OpenWithConfig(*cfg)
	if err != nil {
		t.Fatalf("Failed to open PostgreSQL database: %v", err)
	}

	// Clean up test data (in reverse dependency order)
	tables := []string{
		"mail_items", "mail", "equipment", "inventory",
		"characters", "boss_kills", "web_sessions", "accounts",
	}
	for _, table := range tables {
		_, err := db.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			// Table might not exist yet, that's OK
			t.Logf("Note: Could not clean table %s: %v", table, err)
		}
	}

	t.Cleanup(func() {
		// Clean up after test
		for _, table := range tables {
			db.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		}
		db.Close()
	})

	return db
}

// TestPostgres_OpenWithConfig tests opening a PostgreSQL database
func TestPostgres_OpenWithConfig(t *testing.T) {
	cfg := skipIfNoPostgres(t)

	db, err := OpenWithConfig(*cfg)
	if err != nil {
		t.Fatalf("Failed to open PostgreSQL database: %v", err)
	}
	defer db.Close()

	// Verify connection works
	var result int
	err = db.db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to query PostgreSQL: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

// TestPostgres_ConnectionPoolSettings verifies connection pool is configured correctly
func TestPostgres_ConnectionPoolSettings(t *testing.T) {
	cfg := skipIfNoPostgres(t)

	db, err := OpenWithConfig(*cfg)
	if err != nil {
		t.Fatalf("Failed to open PostgreSQL database: %v", err)
	}
	defer db.Close()

	stats := db.db.Stats()

	// Verify pool settings were applied
	if stats.MaxOpenConnections != cfg.Postgres.MaxOpenConns {
		t.Errorf("Expected MaxOpenConns %d, got %d",
			cfg.Postgres.MaxOpenConns, stats.MaxOpenConnections)
	}
}

// TestPostgres_CITEXTExtension verifies the citext extension is working
func TestPostgres_CITEXTExtension(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create account with lowercase
	_, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Try to get account with different case
	account, err := db.GetAccountByUsername("TESTUSER")
	if err != nil {
		t.Fatalf("Case-insensitive lookup failed: %v", err)
	}
	if account.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", account.Username)
	}

	// Try to create duplicate with different case
	_, err = db.CreateAccount("TestUser", "Password123")
	if err == nil {
		t.Error("Expected error for case-insensitive duplicate, but insert succeeded")
	}
}

// TestPostgres_ConcurrentWrites tests concurrent database writes
func TestPostgres_ConcurrentWrites(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	const numGoroutines = 10
	const writesPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*writesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				username := fmt.Sprintf("user_%d_%d", workerID, j)
				_, err := db.CreateAccount(username, "Password123")
				if err != nil {
					errors <- fmt.Errorf("worker %d: failed to create account %s: %v",
						workerID, username, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify all accounts were created
	count, err := db.GetTotalAccounts()
	if err != nil {
		t.Fatalf("Failed to get account count: %v", err)
	}

	expected := numGoroutines * writesPerGoroutine
	if count != expected {
		t.Errorf("Expected %d accounts, got %d", expected, count)
	}
}

// TestPostgres_ConcurrentReads tests concurrent database reads
func TestPostgres_ConcurrentReads(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create test account
	_, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	const numGoroutines = 20
	const readsPerGoroutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*readsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < readsPerGoroutine; j++ {
				_, err := db.GetAccountByUsername("testuser")
				if err != nil {
					errors <- fmt.Errorf("worker %d read %d: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestPostgres_TransactionIsolation tests transaction isolation
func TestPostgres_TransactionIsolation(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create initial account
	account, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Start transaction 1
	tx1, err := db.db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin tx1: %v", err)
	}

	// Start transaction 2
	tx2, err := db.db.Begin()
	if err != nil {
		tx1.Rollback()
		t.Fatalf("Failed to begin tx2: %v", err)
	}

	// Update in tx1
	query := db.qb.Build("UPDATE accounts SET last_ip = ? WHERE id = ?")
	_, err = tx1.Exec(query, "10.0.0.1", account.ID)
	if err != nil {
		tx1.Rollback()
		tx2.Rollback()
		t.Fatalf("Failed to update in tx1: %v", err)
	}

	// Read in tx2 (should see old value due to isolation)
	var lastIP sql.NullString
	query = db.qb.Build("SELECT last_ip FROM accounts WHERE id = ?")
	err = tx2.QueryRow(query, account.ID).Scan(&lastIP)
	if err != nil {
		tx1.Rollback()
		tx2.Rollback()
		t.Fatalf("Failed to read in tx2: %v", err)
	}

	// Should still see old value (NULL) due to isolation
	if lastIP.Valid && lastIP.String == "10.0.0.1" {
		t.Log("Note: Transaction isolation shows committed data (READ COMMITTED default)")
	}

	// Commit tx1
	if err := tx1.Commit(); err != nil {
		tx2.Rollback()
		t.Fatalf("Failed to commit tx1: %v", err)
	}

	// Now tx2 should see the change on a new read (READ COMMITTED)
	err = tx2.QueryRow(db.qb.Build("SELECT last_ip FROM accounts WHERE id = ?"), account.ID).Scan(&lastIP)
	if err != nil {
		tx2.Rollback()
		t.Fatalf("Failed to read in tx2 after tx1 commit: %v", err)
	}

	tx2.Commit()

	if !lastIP.Valid || lastIP.String != "10.0.0.1" {
		t.Errorf("Expected last_ip '10.0.0.1' after tx1 commit, got '%v'", lastIP)
	}
}

// TestPostgres_RETURNING tests that RETURNING clause works for inserts
func TestPostgres_RETURNING(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create account and verify ID is returned
	account, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	if account.ID == 0 {
		t.Error("Expected non-zero ID from RETURNING clause")
	}

	// Create character and verify ID
	char, err := db.CreateCharacter(account.ID, "TestChar")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	if char.ID == 0 {
		t.Error("Expected non-zero character ID from RETURNING clause")
	}
}

// TestPostgres_ForeignKeyConstraints tests foreign key enforcement
func TestPostgres_ForeignKeyConstraints(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Try to create character with non-existent account
	query := db.qb.Build(`INSERT INTO characters (account_id, name) VALUES (?, ?)`)
	_, err := db.db.Exec(query, 99999, "OrphanChar")

	if err == nil {
		t.Error("Expected foreign key constraint error, but insert succeeded")
	}
}

// TestPostgres_CascadeDelete tests ON DELETE CASCADE behavior
func TestPostgres_CascadeDelete(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create account, character, and inventory item
	account, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	char, err := db.CreateCharacter(account.ID, "TestChar")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Add inventory item using SaveInventory
	err = db.SaveInventory(char.ID, []string{"test_item"})
	if err != nil {
		t.Fatalf("Failed to add inventory item: %v", err)
	}

	// Delete account
	query := db.qb.Build("DELETE FROM accounts WHERE id = ?")
	_, err = db.db.Exec(query, account.ID)
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify character was cascade deleted
	var count int
	query = db.qb.Build("SELECT COUNT(*) FROM characters WHERE id = ?")
	err = db.db.QueryRow(query, char.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check character: %v", err)
	}
	if count != 0 {
		t.Error("Character should have been cascade deleted")
	}

	// Verify inventory was cascade deleted
	items, _ := db.LoadInventory(char.ID)
	if len(items) != 0 {
		t.Error("Inventory should have been cascade deleted")
	}
}

// TestPostgres_MailSystem tests the complete mail system with PostgreSQL
func TestPostgres_MailSystem(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create sender and recipient
	sender, err := db.CreateAccount("sender", "Password123")
	if err != nil {
		t.Fatalf("Failed to create sender account: %v", err)
	}

	recipient, err := db.CreateAccount("recipient", "Password123")
	if err != nil {
		t.Fatalf("Failed to create recipient account: %v", err)
	}

	senderChar, err := db.CreateCharacter(sender.ID, "SenderChar")
	if err != nil {
		t.Fatalf("Failed to create sender character: %v", err)
	}

	recipientChar, err := db.CreateCharacter(recipient.ID, "RecipientChar")
	if err != nil {
		t.Fatalf("Failed to create recipient character: %v", err)
	}

	// Send mail with correct signature
	mailID, err := db.SendMail(senderChar.ID, senderChar.Name, recipientChar.ID, recipientChar.Name,
		"Test Subject", "Test body", 100, nil)
	if err != nil {
		t.Fatalf("Failed to send mail: %v", err)
	}

	if mailID == 0 {
		t.Error("Expected non-zero mail ID")
	}

	// Check mailbox
	mailbox, err := db.GetMailbox(recipientChar.ID)
	if err != nil {
		t.Fatalf("Failed to get mailbox: %v", err)
	}

	if len(mailbox) != 1 {
		t.Errorf("Expected 1 mail in mailbox, got %d", len(mailbox))
	}

	// Verify mail content
	if len(mailbox) > 0 {
		m := mailbox[0]
		if m.Subject != "Test Subject" {
			t.Errorf("Expected subject 'Test Subject', got '%s'", m.Subject)
		}
		if !m.HasGold {
			t.Error("Expected HasGold to be true")
		}
	}
}

// TestPostgres_ConnectionPoolStress tests connection pool under stress
func TestPostgres_ConnectionPoolStress(t *testing.T) {
	cfg := skipIfNoPostgres(t)

	// Use a smaller pool for stress testing
	cfg.Postgres.MaxOpenConns = 5
	cfg.Postgres.MaxIdleConns = 2

	db := setupPostgresTestDB(t, cfg)

	const numGoroutines = 50
	const queriesPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*queriesPerGoroutine)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < queriesPerGoroutine; j++ {
				var result int
				err := db.db.QueryRow("SELECT 1").Scan(&result)
				if err != nil {
					errors <- fmt.Errorf("worker %d query %d: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalQueries := numGoroutines * queriesPerGoroutine

	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	t.Logf("Completed %d queries in %v (%d errors)", totalQueries, duration, errorCount)

	// Check pool stats
	stats := db.db.Stats()
	t.Logf("Pool stats: MaxOpen=%d, Open=%d, InUse=%d, Idle=%d, WaitCount=%d, WaitDuration=%v",
		stats.MaxOpenConnections, stats.OpenConnections, stats.InUse, stats.Idle,
		stats.WaitCount, stats.WaitDuration)
}

// TestPostgres_CharacterWithAllFields tests creating and retrieving a character with all fields
func TestPostgres_CharacterWithAllFields(t *testing.T) {
	cfg := skipIfNoPostgres(t)
	db := setupPostgresTestDB(t, cfg)

	// Create account
	account, err := db.CreateAccount("testuser", "Password123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Create character
	char, err := db.CreateCharacter(account.ID, "TestChar")
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update character with various fields
	char.Health = 150
	char.MaxHealth = 200
	char.Mana = 75
	char.MaxMana = 100
	char.Level = 5
	char.Experience = 1000
	char.Gold = 500
	char.RoomID = "custom_room"
	char.State = "resting"
	char.Strength = 16
	char.Dexterity = 14
	char.Constitution = 15
	char.Intelligence = 12
	char.Wisdom = 10
	char.Charisma = 8

	err = db.SaveCharacter(char)
	if err != nil {
		t.Fatalf("Failed to save character: %v", err)
	}

	// Retrieve and verify
	retrieved, err := db.GetCharacterByID(char.ID)
	if err != nil {
		t.Fatalf("Failed to get character: %v", err)
	}

	if retrieved.Health != 150 {
		t.Errorf("Expected Health 150, got %d", retrieved.Health)
	}
	if retrieved.MaxHealth != 200 {
		t.Errorf("Expected MaxHealth 200, got %d", retrieved.MaxHealth)
	}
	if retrieved.Mana != 75 {
		t.Errorf("Expected Mana 75, got %d", retrieved.Mana)
	}
	if retrieved.Level != 5 {
		t.Errorf("Expected Level 5, got %d", retrieved.Level)
	}
	if retrieved.Gold != 500 {
		t.Errorf("Expected Gold 500, got %d", retrieved.Gold)
	}
	if retrieved.RoomID != "custom_room" {
		t.Errorf("Expected RoomID 'custom_room', got '%s'", retrieved.RoomID)
	}
	if retrieved.Strength != 16 {
		t.Errorf("Expected Strength 16, got %d", retrieved.Strength)
	}
}
