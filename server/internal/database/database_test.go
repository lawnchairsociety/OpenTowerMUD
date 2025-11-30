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
