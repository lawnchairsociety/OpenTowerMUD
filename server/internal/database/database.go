// Package database provides persistence for player accounts and characters.
// Supports both SQLite and PostgreSQL backends.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// Database wraps the database connection and provides persistence operations.
type Database struct {
	db      *sql.DB
	dialect Dialect
	qb      *QueryBuilder
}

// Open opens or creates the SQLite database at the given path.
// This is the legacy function for backward compatibility.
// For new code, use OpenWithConfig instead.
func Open(path string) (*Database, error) {
	return OpenWithConfig(DefaultConfig(path))
}

// OpenWithConfig opens a database connection using the provided configuration.
func OpenWithConfig(cfg Config) (*Database, error) {
	var db *sql.DB
	var dialect Dialect
	var err error

	switch cfg.Driver {
	case "postgres":
		dialect = NewDialect(DialectPostgres)
		db, err = openPostgres(cfg.Postgres)
	default:
		dialect = NewDialect(DialectSQLite)
		db, err = openSQLite(cfg.SQLitePath)
	}

	if err != nil {
		return nil, err
	}

	// Run dialect-specific initialization statements
	for _, stmt := range dialect.InitStatements() {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run init statement: %w", err)
		}
	}

	d := &Database{
		db:      db,
		dialect: dialect,
		qb:      NewQueryBuilder(dialect),
	}

	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return d, nil
}

// openSQLite opens a SQLite database at the given path.
func openSQLite(path string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

// openPostgres opens a PostgreSQL database connection.
func openPostgres(cfg PostgresConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply connection pool settings
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25) // Sensible default
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return db, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}

// Dialect returns the database dialect in use.
func (d *Database) Dialect() Dialect {
	return d.dialect
}

// Query converts the query to the appropriate dialect and returns the converted query.
// Use this for building queries that need placeholder conversion.
func (d *Database) Query(query string) string {
	return d.qb.Build(query)
}

// QueryWithReturning converts the query and appends a RETURNING clause if needed.
// Use this for INSERT statements that need to return the inserted ID.
func (d *Database) QueryWithReturning(query string, column string) string {
	return d.qb.BuildWithReturning(query, column)
}

// migrate creates the database schema if it doesn't exist.
func (d *Database) migrate() error {
	switch d.dialect.(type) {
	case *PostgresDialect:
		return d.migratePostgres()
	default:
		return d.migrateSQLite()
	}
}

// migrateSQLite runs SQLite-specific migrations.
func (d *Database) migrateSQLite() error {
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

		// Boss kills tracking table
		`CREATE TABLE IF NOT EXISTS boss_kills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tower_id TEXT NOT NULL,
			player_name TEXT NOT NULL,
			killed_at DATETIME NOT NULL,
			is_first_kill INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_tower ON boss_kills(tower_id)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_player ON boss_kills(player_name)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_first ON boss_kills(tower_id, is_first_kill)`,
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
		// Trophy system
		`ALTER TABLE characters ADD COLUMN trophy_case TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN earned_titles TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN active_title TEXT NOT NULL DEFAULT ''`,
		// Multi-tower system
		`ALTER TABLE characters ADD COLUMN home_tower TEXT NOT NULL DEFAULT 'human'`,
		// Labyrinth tracking
		`ALTER TABLE characters ADD COLUMN visited_labyrinth_gates TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE characters ADD COLUMN talked_to_lore_npcs TEXT NOT NULL DEFAULT ''`,
		// Character statistics for website
		`ALTER TABLE characters ADD COLUMN statistics TEXT NOT NULL DEFAULT '{}'`,
		// Web sessions table for companion website
		`CREATE TABLE IF NOT EXISTS web_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT UNIQUE NOT NULL,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			ip_address TEXT,
			user_agent TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_web_sessions_token ON web_sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_web_sessions_expires ON web_sessions(expires_at)`,
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

// migratePostgres runs PostgreSQL-specific migrations.
func (d *Database) migratePostgres() error {
	migrations := []string{
		// Accounts table (CITEXT for case-insensitive usernames)
		`CREATE TABLE IF NOT EXISTS accounts (
			id SERIAL PRIMARY KEY,
			username CITEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			last_ip TEXT,
			banned INTEGER NOT NULL DEFAULT 0,
			is_admin INTEGER NOT NULL DEFAULT 0
		)`,

		// Characters table (CITEXT for case-insensitive names)
		`CREATE TABLE IF NOT EXISTS characters (
			id SERIAL PRIMARY KEY,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			name CITEXT UNIQUE NOT NULL,
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
			race TEXT NOT NULL DEFAULT 'human',
			home_tower TEXT NOT NULL DEFAULT 'human',
			crafting_skills TEXT NOT NULL DEFAULT '',
			known_recipes TEXT NOT NULL DEFAULT '',
			quest_log TEXT NOT NULL DEFAULT '{}',
			quest_inventory TEXT NOT NULL DEFAULT '',
			trophy_case TEXT NOT NULL DEFAULT '',
			earned_titles TEXT NOT NULL DEFAULT '',
			active_title TEXT NOT NULL DEFAULT '',
			visited_labyrinth_gates TEXT NOT NULL DEFAULT '',
			talked_to_lore_npcs TEXT NOT NULL DEFAULT '',
			statistics TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_played TIMESTAMP
		)`,

		// Inventory table
		`CREATE TABLE IF NOT EXISTS inventory (
			id SERIAL PRIMARY KEY,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			item_id TEXT NOT NULL
		)`,

		// Equipment table
		`CREATE TABLE IF NOT EXISTS equipment (
			id SERIAL PRIMARY KEY,
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
			id SERIAL PRIMARY KEY,
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
			id SERIAL PRIMARY KEY,
			mail_id INTEGER NOT NULL,
			item_id TEXT NOT NULL,
			collected INTEGER DEFAULT 0,
			FOREIGN KEY (mail_id) REFERENCES mail(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_recipient ON mail(recipient_id, read)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_sender ON mail(sender_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_items_mail ON mail_items(mail_id)`,

		// Boss kills tracking table
		`CREATE TABLE IF NOT EXISTS boss_kills (
			id SERIAL PRIMARY KEY,
			tower_id TEXT NOT NULL,
			player_name TEXT NOT NULL,
			killed_at TIMESTAMP NOT NULL,
			is_first_kill INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_tower ON boss_kills(tower_id)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_player ON boss_kills(player_name)`,
		`CREATE INDEX IF NOT EXISTS idx_boss_kills_first ON boss_kills(tower_id, is_first_kill)`,

		// Web sessions table for companion website
		`CREATE TABLE IF NOT EXISTS web_sessions (
			id SERIAL PRIMARY KEY,
			token TEXT UNIQUE NOT NULL,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			ip_address TEXT,
			user_agent TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_web_sessions_token ON web_sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_web_sessions_expires ON web_sessions(expires_at)`,
	}

	for _, m := range migrations {
		if _, err := d.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	return nil
}

// DB returns the underlying sql.DB for advanced operations.
func (d *Database) DB() *sql.DB {
	return d.db
}
