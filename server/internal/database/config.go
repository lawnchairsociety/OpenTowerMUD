package database

import "time"

// Config holds database connection configuration.
type Config struct {
	// Driver specifies which database to use: "sqlite" or "postgres"
	Driver string

	// SQLite configuration
	SQLitePath string

	// PostgreSQL configuration
	Postgres PostgresConfig
}

// PostgresConfig holds PostgreSQL-specific configuration.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns a Config with sensible defaults for SQLite.
func DefaultConfig(sqlitePath string) Config {
	return Config{
		Driver:     "sqlite",
		SQLitePath: sqlitePath,
	}
}

// DefaultPostgresConfig returns PostgresConfig with recommended pool settings.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
}
