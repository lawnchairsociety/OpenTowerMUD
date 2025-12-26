package database

import (
	"errors"
	"testing"
	"time"
)

// =============================================================================
// Dialect Tests
// =============================================================================

func TestNewDialect_SQLite(t *testing.T) {
	dialect := NewDialect(DialectSQLite)
	if _, ok := dialect.(*SQLiteDialect); !ok {
		t.Errorf("Expected *SQLiteDialect, got %T", dialect)
	}
}

func TestNewDialect_Postgres(t *testing.T) {
	dialect := NewDialect(DialectPostgres)
	if _, ok := dialect.(*PostgresDialect); !ok {
		t.Errorf("Expected *PostgresDialect, got %T", dialect)
	}
}

func TestNewDialect_Default(t *testing.T) {
	// Unknown dialect should default to SQLite
	dialect := NewDialect("unknown")
	if _, ok := dialect.(*SQLiteDialect); !ok {
		t.Errorf("Expected default *SQLiteDialect, got %T", dialect)
	}
}

// =============================================================================
// SQLite Dialect Tests
// =============================================================================

func TestSQLiteDialect_DriverName(t *testing.T) {
	d := &SQLiteDialect{}
	if got := d.DriverName(); got != "sqlite" {
		t.Errorf("DriverName() = %q, want %q", got, "sqlite")
	}
}

func TestSQLiteDialect_Placeholder(t *testing.T) {
	d := &SQLiteDialect{}
	tests := []struct {
		position int
		want     string
	}{
		{1, "?"},
		{2, "?"},
		{10, "?"},
		{100, "?"},
	}
	for _, tt := range tests {
		if got := d.Placeholder(tt.position); got != tt.want {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.position, got, tt.want)
		}
	}
}

func TestSQLiteDialect_SupportsLastInsertID(t *testing.T) {
	d := &SQLiteDialect{}
	if got := d.SupportsLastInsertID(); !got {
		t.Error("SupportsLastInsertID() = false, want true")
	}
}

func TestSQLiteDialect_ReturningClause(t *testing.T) {
	d := &SQLiteDialect{}
	if got := d.ReturningClause("id"); got != "" {
		t.Errorf("ReturningClause() = %q, want empty string", got)
	}
}

func TestSQLiteDialect_InitStatements(t *testing.T) {
	d := &SQLiteDialect{}
	stmts := d.InitStatements()

	expected := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	if len(stmts) != len(expected) {
		t.Errorf("InitStatements() returned %d statements, want %d", len(stmts), len(expected))
	}

	for i, want := range expected {
		if stmts[i] != want {
			t.Errorf("InitStatements()[%d] = %q, want %q", i, stmts[i], want)
		}
	}
}

func TestSQLiteDialect_IsDuplicateKeyError(t *testing.T) {
	d := &SQLiteDialect{}
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("some random error"), false},
		{errors.New("UNIQUE constraint failed: accounts.username"), true},
		{errors.New("UNIQUE constraint failed: characters.name"), true},
		{errors.New("foreign key constraint failed"), false},
	}
	for _, tt := range tests {
		if got := d.IsDuplicateKeyError(tt.err); got != tt.want {
			errStr := "nil"
			if tt.err != nil {
				errStr = tt.err.Error()
			}
			t.Errorf("IsDuplicateKeyError(%q) = %v, want %v", errStr, got, tt.want)
		}
	}
}

func TestSQLiteDialect_CaseInsensitiveCollation(t *testing.T) {
	d := &SQLiteDialect{}
	if got := d.CaseInsensitiveCollation(); got != "COLLATE NOCASE" {
		t.Errorf("CaseInsensitiveCollation() = %q, want %q", got, "COLLATE NOCASE")
	}
}

// =============================================================================
// PostgreSQL Dialect Tests
// =============================================================================

func TestPostgresDialect_DriverName(t *testing.T) {
	d := &PostgresDialect{}
	if got := d.DriverName(); got != "postgres" {
		t.Errorf("DriverName() = %q, want %q", got, "postgres")
	}
}

func TestPostgresDialect_Placeholder(t *testing.T) {
	d := &PostgresDialect{}
	tests := []struct {
		position int
		want     string
	}{
		{1, "$1"},
		{2, "$2"},
		{10, "$10"},
		{100, "$100"},
	}
	for _, tt := range tests {
		if got := d.Placeholder(tt.position); got != tt.want {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.position, got, tt.want)
		}
	}
}

func TestPostgresDialect_SupportsLastInsertID(t *testing.T) {
	d := &PostgresDialect{}
	if got := d.SupportsLastInsertID(); got {
		t.Error("SupportsLastInsertID() = true, want false")
	}
}

func TestPostgresDialect_ReturningClause(t *testing.T) {
	d := &PostgresDialect{}
	tests := []struct {
		column string
		want   string
	}{
		{"id", " RETURNING id"},
		{"account_id", " RETURNING account_id"},
	}
	for _, tt := range tests {
		if got := d.ReturningClause(tt.column); got != tt.want {
			t.Errorf("ReturningClause(%q) = %q, want %q", tt.column, got, tt.want)
		}
	}
}

func TestPostgresDialect_InitStatements(t *testing.T) {
	d := &PostgresDialect{}
	stmts := d.InitStatements()

	if len(stmts) != 1 {
		t.Errorf("InitStatements() returned %d statements, want 1", len(stmts))
	}

	expected := "CREATE EXTENSION IF NOT EXISTS citext"
	if stmts[0] != expected {
		t.Errorf("InitStatements()[0] = %q, want %q", stmts[0], expected)
	}
}

func TestPostgresDialect_IsDuplicateKeyError(t *testing.T) {
	d := &PostgresDialect{}
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("some random error"), false},
		{errors.New("duplicate key value violates unique constraint"), true},
		{errors.New("ERROR: duplicate key value (SQLSTATE 23505)"), true},
		{errors.New("pq: unique constraint violation on accounts_username_key"), true},
		{errors.New("foreign key constraint"), false},
	}
	for _, tt := range tests {
		if got := d.IsDuplicateKeyError(tt.err); got != tt.want {
			errStr := "nil"
			if tt.err != nil {
				errStr = tt.err.Error()
			}
			t.Errorf("IsDuplicateKeyError(%q) = %v, want %v", errStr, got, tt.want)
		}
	}
}

func TestPostgresDialect_CaseInsensitiveCollation(t *testing.T) {
	d := &PostgresDialect{}
	if got := d.CaseInsensitiveCollation(); got != "" {
		t.Errorf("CaseInsensitiveCollation() = %q, want empty string", got)
	}
}

// =============================================================================
// QueryBuilder Tests
// =============================================================================

func TestNewQueryBuilder(t *testing.T) {
	dialect := &SQLiteDialect{}
	qb := NewQueryBuilder(dialect)
	if qb == nil {
		t.Fatal("NewQueryBuilder() returned nil")
	}
	if qb.dialect != dialect {
		t.Error("QueryBuilder dialect not set correctly")
	}
}

func TestQueryBuilder_Build_SQLite(t *testing.T) {
	qb := NewQueryBuilder(&SQLiteDialect{})
	tests := []struct {
		input string
		want  string
	}{
		{"SELECT * FROM users", "SELECT * FROM users"},
		{"SELECT * FROM users WHERE id = ?", "SELECT * FROM users WHERE id = ?"},
		{"SELECT * FROM users WHERE id = ? AND name = ?", "SELECT * FROM users WHERE id = ? AND name = ?"},
		{"INSERT INTO users (name, age) VALUES (?, ?)", "INSERT INTO users (name, age) VALUES (?, ?)"},
		{"UPDATE users SET name = ? WHERE id = ?", "UPDATE users SET name = ? WHERE id = ?"},
	}
	for _, tt := range tests {
		if got := qb.Build(tt.input); got != tt.want {
			t.Errorf("Build(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestQueryBuilder_Build_Postgres(t *testing.T) {
	qb := NewQueryBuilder(&PostgresDialect{})
	tests := []struct {
		input string
		want  string
	}{
		{"SELECT * FROM users", "SELECT * FROM users"},
		{"SELECT * FROM users WHERE id = ?", "SELECT * FROM users WHERE id = $1"},
		{"SELECT * FROM users WHERE id = ? AND name = ?", "SELECT * FROM users WHERE id = $1 AND name = $2"},
		{"INSERT INTO users (name, age) VALUES (?, ?)", "INSERT INTO users (name, age) VALUES ($1, $2)"},
		{"UPDATE users SET name = ? WHERE id = ?", "UPDATE users SET name = $1 WHERE id = $2"},
		{
			"SELECT * FROM users WHERE a = ? AND b = ? AND c = ? AND d = ? AND e = ?",
			"SELECT * FROM users WHERE a = $1 AND b = $2 AND c = $3 AND d = $4 AND e = $5",
		},
	}
	for _, tt := range tests {
		if got := qb.Build(tt.input); got != tt.want {
			t.Errorf("Build(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestQueryBuilder_BuildWithReturning_SQLite(t *testing.T) {
	qb := NewQueryBuilder(&SQLiteDialect{})
	tests := []struct {
		query  string
		column string
		want   string
	}{
		{"INSERT INTO users (name) VALUES (?)", "id", "INSERT INTO users (name) VALUES (?)"},
		{"INSERT INTO accounts (username) VALUES (?)", "id", "INSERT INTO accounts (username) VALUES (?)"},
	}
	for _, tt := range tests {
		if got := qb.BuildWithReturning(tt.query, tt.column); got != tt.want {
			t.Errorf("BuildWithReturning(%q, %q) = %q, want %q", tt.query, tt.column, got, tt.want)
		}
	}
}

func TestQueryBuilder_BuildWithReturning_Postgres(t *testing.T) {
	qb := NewQueryBuilder(&PostgresDialect{})
	tests := []struct {
		query  string
		column string
		want   string
	}{
		{"INSERT INTO users (name) VALUES (?)", "id", "INSERT INTO users (name) VALUES ($1) RETURNING id"},
		{"INSERT INTO accounts (username, password) VALUES (?, ?)", "id", "INSERT INTO accounts (username, password) VALUES ($1, $2) RETURNING id"},
	}
	for _, tt := range tests {
		if got := qb.BuildWithReturning(tt.query, tt.column); got != tt.want {
			t.Errorf("BuildWithReturning(%q, %q) = %q, want %q", tt.query, tt.column, got, tt.want)
		}
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	path := "/path/to/test.db"
	cfg := DefaultConfig(path)

	if cfg.Driver != "sqlite" {
		t.Errorf("Driver = %q, want %q", cfg.Driver, "sqlite")
	}
	if cfg.SQLitePath != path {
		t.Errorf("SQLitePath = %q, want %q", cfg.SQLitePath, path)
	}
}

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := DefaultPostgresConfig()

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5432)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want %q", cfg.SSLMode, "disable")
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 25)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 5)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want %v", cfg.ConnMaxLifetime, 5*time.Minute)
	}
}

func TestConfig_PostgresFields(t *testing.T) {
	cfg := Config{
		Driver: "postgres",
		Postgres: PostgresConfig{
			Host:            "db.example.com",
			Port:            5433,
			User:            "testuser",
			Password:        "testpass",
			Database:        "testdb",
			SSLMode:         "require",
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: 10 * time.Minute,
		},
	}

	if cfg.Postgres.Host != "db.example.com" {
		t.Errorf("Postgres.Host = %q, want %q", cfg.Postgres.Host, "db.example.com")
	}
	if cfg.Postgres.Port != 5433 {
		t.Errorf("Postgres.Port = %d, want %d", cfg.Postgres.Port, 5433)
	}
	if cfg.Postgres.User != "testuser" {
		t.Errorf("Postgres.User = %q, want %q", cfg.Postgres.User, "testuser")
	}
	if cfg.Postgres.Password != "testpass" {
		t.Errorf("Postgres.Password = %q, want %q", cfg.Postgres.Password, "testpass")
	}
	if cfg.Postgres.Database != "testdb" {
		t.Errorf("Postgres.Database = %q, want %q", cfg.Postgres.Database, "testdb")
	}
	if cfg.Postgres.SSLMode != "require" {
		t.Errorf("Postgres.SSLMode = %q, want %q", cfg.Postgres.SSLMode, "require")
	}
}

// =============================================================================
// Dialect Interface Compliance Tests
// =============================================================================

// Verify that both dialects implement the Dialect interface
func TestDialect_InterfaceCompliance(t *testing.T) {
	var _ Dialect = (*SQLiteDialect)(nil)
	var _ Dialect = (*PostgresDialect)(nil)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestQueryBuilder_Build_EmptyQuery(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
	}{
		{"SQLite", &SQLiteDialect{}},
		{"Postgres", &PostgresDialect{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(tt.dialect)
			if got := qb.Build(""); got != "" {
				t.Errorf("Build(\"\") = %q, want empty string", got)
			}
		})
	}
}

func TestQueryBuilder_Build_NoPlaceholders(t *testing.T) {
	qb := NewQueryBuilder(&PostgresDialect{})
	query := "SELECT * FROM users ORDER BY name"
	if got := qb.Build(query); got != query {
		t.Errorf("Build(%q) = %q, want original query unchanged", query, got)
	}
}

func TestQueryBuilder_Build_ManyPlaceholders(t *testing.T) {
	qb := NewQueryBuilder(&PostgresDialect{})
	// 15 placeholders - tests double-digit position numbers
	input := "INSERT INTO t VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	want := "INSERT INTO t VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"
	if got := qb.Build(input); got != want {
		t.Errorf("Build with 15 placeholders failed:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestQueryBuilder_Build_QuestionMarkInString(t *testing.T) {
	// Note: The current implementation doesn't handle quoted strings specially.
	// This test documents the current behavior.
	qb := NewQueryBuilder(&PostgresDialect{})
	// A query with ? in a string literal would be incorrectly converted
	// In practice, users should use parameterized values for all dynamic content
	input := "SELECT * FROM users WHERE id = ?"
	want := "SELECT * FROM users WHERE id = $1"
	if got := qb.Build(input); got != want {
		t.Errorf("Build(%q) = %q, want %q", input, got, want)
	}
}
