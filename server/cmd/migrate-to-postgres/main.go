// migrate-to-postgres migrates data from SQLite to PostgreSQL.
//
// Usage:
//
//	go run ./cmd/migrate-to-postgres \
//	    -sqlite data/opentowermud.db \
//	    -pg-host localhost \
//	    -pg-port 5435 \
//	    -pg-user opentower \
//	    -pg-password opentower \
//	    -pg-database opentower
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func main() {
	// Parse command-line flags
	sqlitePath := flag.String("sqlite", "data/opentowermud.db", "Path to SQLite database")
	pgHost := flag.String("pg-host", "localhost", "PostgreSQL host")
	pgPort := flag.Int("pg-port", 5435, "PostgreSQL port")
	pgUser := flag.String("pg-user", "opentower", "PostgreSQL user")
	pgPassword := flag.String("pg-password", "opentower", "PostgreSQL password")
	pgDatabase := flag.String("pg-database", "opentower", "PostgreSQL database name")
	pgSSLMode := flag.String("pg-sslmode", "disable", "PostgreSQL SSL mode")
	dryRun := flag.Bool("dry-run", false, "Show what would be migrated without making changes")
	flag.Parse()

	log.Println("SQLite to PostgreSQL Migration Tool")
	log.Println("====================================")

	// Open SQLite database
	log.Printf("Opening SQLite database: %s", *sqlitePath)
	sqliteDB, err := sql.Open("sqlite", *sqlitePath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	// Verify SQLite connection
	if err := sqliteDB.Ping(); err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	// Build PostgreSQL connection string
	pgConnStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		*pgHost, *pgPort, *pgUser, *pgPassword, *pgDatabase, *pgSSLMode,
	)

	// Open PostgreSQL database
	log.Printf("Opening PostgreSQL database: %s@%s:%d/%s", *pgUser, *pgHost, *pgPort, *pgDatabase)
	pgDB, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Failed to open PostgreSQL database: %v", err)
	}
	defer pgDB.Close()

	// Verify PostgreSQL connection
	if err := pgDB.Ping(); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database: %v", err)
	}

	if *dryRun {
		log.Println("DRY RUN MODE - No changes will be made")
	}

	// Run migrations on PostgreSQL first
	log.Println("Ensuring PostgreSQL schema is ready...")
	if !*dryRun {
		if err := migratePostgres(pgDB); err != nil {
			log.Fatalf("Failed to migrate PostgreSQL schema: %v", err)
		}
	}

	// Migrate each table
	tables := []struct {
		name    string
		migrate func(*sql.DB, *sql.DB, bool) (int64, error)
	}{
		{"accounts", migrateAccounts},
		{"characters", migrateCharacters},
		{"inventory", migrateInventory},
		{"equipment", migrateEquipment},
		{"mail", migrateMail},
		{"mail_items", migrateMailItems},
		{"boss_kills", migrateBossKills},
		{"web_sessions", migrateWebSessions},
	}

	var totalRows int64
	for _, t := range tables {
		log.Printf("Migrating table: %s", t.name)
		count, err := t.migrate(sqliteDB, pgDB, *dryRun)
		if err != nil {
			log.Fatalf("Failed to migrate %s: %v", t.name, err)
		}
		log.Printf("  Migrated %d rows", count)
		totalRows += count
	}

	log.Println("====================================")
	log.Printf("Migration complete! Total rows migrated: %d", totalRows)
	if *dryRun {
		log.Println("(DRY RUN - No actual changes were made)")
	}
}

func migratePostgres(db *sql.DB) error {
	// Enable citext extension
	if _, err := db.Exec("CREATE EXTENSION IF NOT EXISTS citext"); err != nil {
		return fmt.Errorf("failed to create citext extension: %w", err)
	}

	migrations := []string{
		// Accounts table
		`CREATE TABLE IF NOT EXISTS accounts (
			id SERIAL PRIMARY KEY,
			username CITEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			last_ip TEXT,
			banned INTEGER NOT NULL DEFAULT 0,
			is_admin INTEGER NOT NULL DEFAULT 0,
			email TEXT,
			email_verified INTEGER DEFAULT 0
		)`,

		// Characters table
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

		// Indexes
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

		// Web sessions table
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
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	return nil
}

func migrateAccounts(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	// Check if email columns exist in SQLite
	hasEmailColumns := false
	var colCount int
	err := sqlite.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name = 'email'`).Scan(&colCount)
	if err == nil && colCount > 0 {
		hasEmailColumns = true
	}

	var rows *sql.Rows
	if hasEmailColumns {
		rows, err = sqlite.Query(`
			SELECT id, username, password_hash, created_at, last_login, last_ip, banned, is_admin,
			       COALESCE(email, ''), COALESCE(email_verified, 0)
			FROM accounts
		`)
	} else {
		rows, err = sqlite.Query(`
			SELECT id, username, password_hash, created_at, last_login, last_ip, banned, is_admin
			FROM accounts
		`)
	}
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id int64
		var username, passwordHash string
		var createdAt string
		var lastLogin, lastIP sql.NullString
		var banned, isAdmin int
		var email string
		var emailVerified int

		if hasEmailColumns {
			if err := rows.Scan(&id, &username, &passwordHash, &createdAt, &lastLogin, &lastIP, &banned, &isAdmin, &email, &emailVerified); err != nil {
				return count, err
			}
		} else {
			if err := rows.Scan(&id, &username, &passwordHash, &createdAt, &lastLogin, &lastIP, &banned, &isAdmin); err != nil {
				return count, err
			}
			email = ""
			emailVerified = 0
		}

		if dryRun {
			count++
			continue
		}

		// Check if account already exists
		var existingID int64
		err := pg.QueryRow(`SELECT id FROM accounts WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			// Account exists, skip
			continue
		}

		// Insert with explicit ID to preserve relationships
		_, err = pg.Exec(`
			INSERT INTO accounts (id, username, password_hash, created_at, last_login, last_ip, banned, is_admin, email, email_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, id, username, passwordHash, parseTime(createdAt), parseNullTime(lastLogin), nullString(lastIP), banned, isAdmin, nullableEmail(email), emailVerified)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	// Reset the sequence to avoid ID conflicts for new records
	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('accounts_id_seq', COALESCE((SELECT MAX(id) FROM accounts), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateCharacters(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`
		SELECT id, account_id, name, room_id, health, max_health, mana, max_mana,
		       level, experience, state, max_carry_weight, learned_spells, discovered_portals,
		       strength, dexterity, constitution, intelligence, wisdom, charisma,
		       gold, key_ring, primary_class, class_levels, active_class,
		       COALESCE(race, 'human'), COALESCE(home_tower, 'human'),
		       COALESCE(crafting_skills, ''), COALESCE(known_recipes, ''),
		       COALESCE(quest_log, '{}'), COALESCE(quest_inventory, ''),
		       COALESCE(trophy_case, ''), COALESCE(earned_titles, ''), COALESCE(active_title, ''),
		       COALESCE(visited_labyrinth_gates, ''), COALESCE(talked_to_lore_npcs, ''),
		       COALESCE(statistics, '{}'),
		       created_at, last_played
		FROM characters
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, accountID int64
		var name, roomID, state, learnedSpells, discoveredPortals, keyRing string
		var primaryClass, classLevels, activeClass, race, homeTower string
		var craftingSkills, knownRecipes, questLog, questInventory string
		var trophyCase, earnedTitles, activeTitle string
		var visitedLabyrinthGates, talkedToLoreNPCs, statistics string
		var createdAt string
		var lastPlayed sql.NullString
		var health, maxHealth, mana, maxMana, level, experience int
		var maxCarryWeight float64
		var strength, dexterity, constitution, intelligence, wisdom, charisma, gold int

		if err := rows.Scan(
			&id, &accountID, &name, &roomID, &health, &maxHealth, &mana, &maxMana,
			&level, &experience, &state, &maxCarryWeight, &learnedSpells, &discoveredPortals,
			&strength, &dexterity, &constitution, &intelligence, &wisdom, &charisma,
			&gold, &keyRing, &primaryClass, &classLevels, &activeClass,
			&race, &homeTower, &craftingSkills, &knownRecipes,
			&questLog, &questInventory, &trophyCase, &earnedTitles, &activeTitle,
			&visitedLabyrinthGates, &talkedToLoreNPCs, &statistics,
			&createdAt, &lastPlayed,
		); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		// Check if character already exists
		var existingID int64
		err := pg.QueryRow(`SELECT id FROM characters WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`
			INSERT INTO characters (
				id, account_id, name, room_id, health, max_health, mana, max_mana,
				level, experience, state, max_carry_weight, learned_spells, discovered_portals,
				strength, dexterity, constitution, intelligence, wisdom, charisma,
				gold, key_ring, primary_class, class_levels, active_class,
				race, home_tower, crafting_skills, known_recipes,
				quest_log, quest_inventory, trophy_case, earned_titles, active_title,
				visited_labyrinth_gates, talked_to_lore_npcs, statistics,
				created_at, last_played
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
				$21, $22, $23, $24, $25, $26, $27, $28, $29,
				$30, $31, $32, $33, $34, $35, $36, $37, $38, $39
			)
		`, id, accountID, name, roomID, health, maxHealth, mana, maxMana,
			level, experience, state, maxCarryWeight, learnedSpells, discoveredPortals,
			strength, dexterity, constitution, intelligence, wisdom, charisma,
			gold, keyRing, primaryClass, classLevels, activeClass,
			race, homeTower, craftingSkills, knownRecipes,
			questLog, questInventory, trophyCase, earnedTitles, activeTitle,
			visitedLabyrinthGates, talkedToLoreNPCs, statistics,
			parseTime(createdAt), parseNullTime(lastPlayed))
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('characters_id_seq', COALESCE((SELECT MAX(id) FROM characters), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateInventory(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`SELECT id, character_id, item_id FROM inventory`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, characterID int64
		var itemID string

		if err := rows.Scan(&id, &characterID, &itemID); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM inventory WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`INSERT INTO inventory (id, character_id, item_id) VALUES ($1, $2, $3)`, id, characterID, itemID)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('inventory_id_seq', COALESCE((SELECT MAX(id) FROM inventory), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateEquipment(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`SELECT id, character_id, slot, item_id FROM equipment`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, characterID int64
		var slot, itemID string

		if err := rows.Scan(&id, &characterID, &slot, &itemID); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM equipment WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`INSERT INTO equipment (id, character_id, slot, item_id) VALUES ($1, $2, $3, $4)`, id, characterID, slot, itemID)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('equipment_id_seq', COALESCE((SELECT MAX(id) FROM equipment), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateMail(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`
		SELECT id, sender_id, sender_name, recipient_id, recipient_name, subject, body,
		       gold_attached, gold_collected, items_collected, read, sent_at
		FROM mail
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, senderID, recipientID int64
		var senderName, recipientName, subject, body string
		var goldAttached, goldCollected, itemsCollected, read int
		var sentAt string

		if err := rows.Scan(&id, &senderID, &senderName, &recipientID, &recipientName, &subject, &body,
			&goldAttached, &goldCollected, &itemsCollected, &read, &sentAt); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM mail WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`
			INSERT INTO mail (id, sender_id, sender_name, recipient_id, recipient_name, subject, body,
			                  gold_attached, gold_collected, items_collected, read, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, id, senderID, senderName, recipientID, recipientName, subject, body,
			goldAttached, goldCollected, itemsCollected, read, parseTime(sentAt))
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('mail_id_seq', COALESCE((SELECT MAX(id) FROM mail), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateMailItems(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`SELECT id, mail_id, item_id, collected FROM mail_items`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, mailID int64
		var itemID string
		var collected int

		if err := rows.Scan(&id, &mailID, &itemID, &collected); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM mail_items WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`INSERT INTO mail_items (id, mail_id, item_id, collected) VALUES ($1, $2, $3, $4)`, id, mailID, itemID, collected)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('mail_items_id_seq', COALESCE((SELECT MAX(id) FROM mail_items), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateBossKills(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`SELECT id, tower_id, player_name, killed_at, is_first_kill FROM boss_kills`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id int64
		var towerID, playerName string
		var killedAt string
		var isFirstKill int

		if err := rows.Scan(&id, &towerID, &playerName, &killedAt, &isFirstKill); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM boss_kills WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`INSERT INTO boss_kills (id, tower_id, player_name, killed_at, is_first_kill) VALUES ($1, $2, $3, $4, $5)`,
			id, towerID, playerName, parseTime(killedAt), isFirstKill)
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('boss_kills_id_seq', COALESCE((SELECT MAX(id) FROM boss_kills), 0) + 1, false)`)
	}

	return count, rows.Err()
}

func migrateWebSessions(sqlite, pg *sql.DB, dryRun bool) (int64, error) {
	rows, err := sqlite.Query(`SELECT id, token, account_id, created_at, expires_at, ip_address, user_agent FROM web_sessions`)
	if err != nil {
		// Table might not exist in older databases
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var id, accountID int64
		var token, createdAt, expiresAt string
		var ipAddress, userAgent sql.NullString

		if err := rows.Scan(&id, &token, &accountID, &createdAt, &expiresAt, &ipAddress, &userAgent); err != nil {
			return count, err
		}

		if dryRun {
			count++
			continue
		}

		var existingID int64
		err := pg.QueryRow(`SELECT id FROM web_sessions WHERE id = $1`, id).Scan(&existingID)
		if err == nil {
			continue
		}

		_, err = pg.Exec(`
			INSERT INTO web_sessions (id, token, account_id, created_at, expires_at, ip_address, user_agent)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, token, accountID, parseTime(createdAt), parseTime(expiresAt), nullString(ipAddress), nullString(userAgent))
		if err != nil {
			if !strings.Contains(err.Error(), "duplicate key") {
				return count, err
			}
		} else {
			count++
		}
	}

	if !dryRun {
		_, _ = pg.Exec(`SELECT setval('web_sessions_id_seq', COALESCE((SELECT MAX(id) FROM web_sessions), 0) + 1, false)`)
	}

	return count, rows.Err()
}

// Helper functions

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	// Try various formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	log.Printf("Warning: Could not parse time: %s", s)
	return nil
}

func parseNullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	return parseTime(ns.String)
}

func nullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullableEmail(email string) *string {
	if email == "" {
		return nil
	}
	return &email
}

func init() {
	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Migrates data from SQLite to PostgreSQL.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -sqlite data/opentowermud.db -pg-host localhost -pg-user opentower -pg-password opentower -pg-database opentower\n", os.Args[0])
	}
}
