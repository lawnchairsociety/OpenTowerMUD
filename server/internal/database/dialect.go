package database

// Dialect abstracts database-specific SQL syntax differences between SQLite and PostgreSQL.
type Dialect interface {
	// DriverName returns the driver name for sql.Open().
	// SQLite: "sqlite", PostgreSQL: "postgres"
	DriverName() string

	// Placeholder returns the parameter placeholder for the given position (1-indexed).
	// SQLite: "?" (ignores position), PostgreSQL: "$1", "$2", etc.
	Placeholder(position int) string

	// SupportsLastInsertID returns true if the database supports LastInsertId().
	// SQLite: true, PostgreSQL: false (uses RETURNING clause instead)
	SupportsLastInsertID() bool

	// ReturningClause returns the RETURNING clause for INSERT statements.
	// SQLite: "" (not used), PostgreSQL: "RETURNING id"
	ReturningClause(column string) string

	// InitStatements returns database-specific initialization statements.
	// SQLite: PRAGMA statements, PostgreSQL: extension creation
	InitStatements() []string

	// IsDuplicateKeyError returns true if the error is a unique constraint violation.
	IsDuplicateKeyError(err error) bool

	// CaseInsensitiveCollation returns the collation for case-insensitive text comparison.
	// SQLite: "COLLATE NOCASE", PostgreSQL: "" (uses CITEXT type instead)
	CaseInsensitiveCollation() string
}

// DialectType identifies the database dialect.
type DialectType string

const (
	DialectSQLite   DialectType = "sqlite"
	DialectPostgres DialectType = "postgres"
)

// NewDialect creates a new Dialect for the given type.
func NewDialect(dialectType DialectType) Dialect {
	switch dialectType {
	case DialectPostgres:
		return &PostgresDialect{}
	default:
		return &SQLiteDialect{}
	}
}
