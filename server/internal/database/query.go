package database

import (
	"strings"
)

// QueryBuilder converts SQL queries with ? placeholders to dialect-specific format.
type QueryBuilder struct {
	dialect Dialect
}

// NewQueryBuilder creates a new QueryBuilder for the given dialect.
func NewQueryBuilder(dialect Dialect) *QueryBuilder {
	return &QueryBuilder{dialect: dialect}
}

// Build converts a query with ? placeholders to dialect-specific placeholders.
// For SQLite, returns the query unchanged.
// For PostgreSQL, converts ? to $1, $2, etc.
//
// Example:
//
//	input:  "SELECT * FROM users WHERE id = ? AND name = ?"
//	SQLite: "SELECT * FROM users WHERE id = ? AND name = ?"
//	Postgres: "SELECT * FROM users WHERE id = $1 AND name = $2"
func (qb *QueryBuilder) Build(query string) string {
	// SQLite uses ? placeholders, so no conversion needed
	if _, ok := qb.dialect.(*SQLiteDialect); ok {
		return query
	}

	// For PostgreSQL, convert ? to $1, $2, etc.
	var result strings.Builder
	position := 1

	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			result.WriteString(qb.dialect.Placeholder(position))
			position++
		} else {
			result.WriteByte(query[i])
		}
	}

	return result.String()
}

// BuildWithReturning appends a RETURNING clause if the dialect requires it.
// Used for INSERT statements that need the inserted ID.
//
// Example:
//
//	input:  "INSERT INTO users (name) VALUES (?)", "id"
//	SQLite: "INSERT INTO users (name) VALUES (?)"
//	Postgres: "INSERT INTO users (name) VALUES ($1) RETURNING id"
func (qb *QueryBuilder) BuildWithReturning(query string, column string) string {
	converted := qb.Build(query)
	if !qb.dialect.SupportsLastInsertID() {
		converted += qb.dialect.ReturningClause(column)
	}
	return converted
}
