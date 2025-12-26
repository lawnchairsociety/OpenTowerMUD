package database

import (
	"strings"
)

// SQLiteDialect implements Dialect for SQLite databases.
type SQLiteDialect struct{}

// DriverName returns "sqlite" for the modernc.org/sqlite driver.
func (d *SQLiteDialect) DriverName() string {
	return "sqlite"
}

// Placeholder returns "?" for all positions (SQLite uses positional ? placeholders).
func (d *SQLiteDialect) Placeholder(position int) string {
	return "?"
}

// SupportsLastInsertID returns true because SQLite supports LastInsertId().
func (d *SQLiteDialect) SupportsLastInsertID() bool {
	return true
}

// ReturningClause returns an empty string because SQLite uses LastInsertId() instead.
func (d *SQLiteDialect) ReturningClause(column string) string {
	return ""
}

// InitStatements returns SQLite PRAGMA statements for optimal operation.
func (d *SQLiteDialect) InitStatements() []string {
	return []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
}

// IsDuplicateKeyError returns true if the error is a SQLite UNIQUE constraint violation.
func (d *SQLiteDialect) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// CaseInsensitiveCollation returns "COLLATE NOCASE" for case-insensitive comparison.
func (d *SQLiteDialect) CaseInsensitiveCollation() string {
	return "COLLATE NOCASE"
}
