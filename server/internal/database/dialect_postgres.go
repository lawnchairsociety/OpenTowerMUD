package database

import (
	"fmt"
	"strings"
)

// PostgresDialect implements Dialect for PostgreSQL databases.
type PostgresDialect struct{}

// DriverName returns "postgres" for the lib/pq or pgx driver.
func (d *PostgresDialect) DriverName() string {
	return "postgres"
}

// Placeholder returns "$N" for the given position (PostgreSQL uses numbered placeholders).
func (d *PostgresDialect) Placeholder(position int) string {
	return fmt.Sprintf("$%d", position)
}

// SupportsLastInsertID returns false because PostgreSQL requires RETURNING clause.
func (d *PostgresDialect) SupportsLastInsertID() bool {
	return false
}

// ReturningClause returns "RETURNING <column>" for INSERT statements.
func (d *PostgresDialect) ReturningClause(column string) string {
	return fmt.Sprintf(" RETURNING %s", column)
}

// InitStatements returns PostgreSQL initialization statements.
// Foreign keys are always enabled in PostgreSQL, so no PRAGMA needed.
func (d *PostgresDialect) InitStatements() []string {
	return []string{
		// Enable the citext extension for case-insensitive text columns
		"CREATE EXTENSION IF NOT EXISTS citext",
	}
}

// IsDuplicateKeyError returns true if the error is a PostgreSQL unique violation.
func (d *PostgresDialect) IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL error code 23505 is unique_violation
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "unique constraint")
}

// CaseInsensitiveCollation returns an empty string because PostgreSQL uses CITEXT type.
func (d *PostgresDialect) CaseInsensitiveCollation() string {
	return ""
}
