// Package database provides SQLite-based persistence for player accounts and characters.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Database wraps the SQLite connection and provides persistence operations.
type Database struct {
	db *sql.DB
}

// Open opens or creates the SQLite database at the given path.
func Open(path string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to wait for locks instead of immediately failing
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	d := &Database{db: db}

	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return d, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}

// migrate creates the database schema if it doesn't exist.
func (d *Database) migrate() error {
	migrations := []string{
		// Accounts table
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			last_ip TEXT,
			banned INTEGER NOT NULL DEFAULT 0,
			is_admin INTEGER NOT NULL DEFAULT 0
		)`,

		// Characters table
		`CREATE TABLE IF NOT EXISTS characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			name TEXT UNIQUE NOT NULL COLLATE NOCASE,
			room_id TEXT NOT NULL DEFAULT 'town_square',
			health INTEGER NOT NULL DEFAULT 100,
			max_health INTEGER NOT NULL DEFAULT 100,
			mana INTEGER NOT NULL DEFAULT 0,
			max_mana INTEGER NOT NULL DEFAULT 0,
			level INTEGER NOT NULL DEFAULT 1,
			experience INTEGER NOT NULL DEFAULT 0,
			state TEXT NOT NULL DEFAULT 'standing',
			max_carry_weight REAL NOT NULL DEFAULT 100.0,
			learned_spells TEXT NOT NULL DEFAULT '',
			discovered_portals TEXT NOT NULL DEFAULT '0',
			strength INTEGER NOT NULL DEFAULT 10,
			dexterity INTEGER NOT NULL DEFAULT 10,
			constitution INTEGER NOT NULL DEFAULT 10,
			intelligence INTEGER NOT NULL DEFAULT 10,
			wisdom INTEGER NOT NULL DEFAULT 10,
			charisma INTEGER NOT NULL DEFAULT 10,
			gold INTEGER NOT NULL DEFAULT 20,
			key_ring TEXT NOT NULL DEFAULT '',
			primary_class TEXT NOT NULL DEFAULT 'warrior',
			class_levels TEXT NOT NULL DEFAULT '{"warrior":1}',
			active_class TEXT NOT NULL DEFAULT 'warrior',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_played TIMESTAMP
		)`,

		// Inventory table
		`CREATE TABLE IF NOT EXISTS inventory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			item_id TEXT NOT NULL
		)`,

		// Equipment table
		`CREATE TABLE IF NOT EXISTS equipment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			slot TEXT NOT NULL,
			item_id TEXT NOT NULL,
			UNIQUE(character_id, slot)
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_characters_account_id ON characters(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_inventory_character_id ON inventory(character_id)`,
		`CREATE INDEX IF NOT EXISTS idx_equipment_character_id ON equipment(character_id)`,

		// Mail tables
		`CREATE TABLE IF NOT EXISTS mail (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_id INTEGER NOT NULL,
			sender_name TEXT NOT NULL,
			recipient_id INTEGER NOT NULL,
			recipient_name TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			gold_attached INTEGER DEFAULT 0,
			gold_collected INTEGER DEFAULT 0,
			items_collected INTEGER DEFAULT 0,
			read INTEGER DEFAULT 0,
			sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (sender_id) REFERENCES characters(id),
			FOREIGN KEY (recipient_id) REFERENCES characters(id)
		)`,
		`CREATE TABLE IF NOT EXISTS mail_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mail_id INTEGER NOT NULL,
			item_id TEXT NOT NULL,
			collected INTEGER DEFAULT 0,
			FOREIGN KEY (mail_id) REFERENCES mail(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_recipient ON mail(recipient_id, read)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_sender ON mail(sender_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_items_mail ON mail_items(mail_id)`,
	}

	// Run safe migrations for new columns (ignore errors if columns already exist)
	safeMigrations := []string{
		`ALTER TABLE characters ADD COLUMN gold INTEGER NOT NULL DEFAULT 100`,
		`ALTER TABLE characters ADD COLUMN key_ring TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN race TEXT NOT NULL DEFAULT 'human'`,
		`ALTER TABLE characters ADD COLUMN crafting_skills TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN known_recipes TEXT NOT NULL DEFAULT ''`,
		// Quest system columns
		`ALTER TABLE characters ADD COLUMN quest_log TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE characters ADD COLUMN quest_inventory TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN earned_titles TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN active_title TEXT NOT NULL DEFAULT ''`,
	}

	for _, m := range migrations {
		if _, err := d.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	// Run safe migrations (ignore "duplicate column" errors for existing databases)
	for _, m := range safeMigrations {
		_, _ = d.db.Exec(m) // Ignore errors - column may already exist
	}

	return nil
}

// DB returns the underlying sql.DB for advanced operations.
func (d *Database) DB() *sql.DB {
	return d.db
}
