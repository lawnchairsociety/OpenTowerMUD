# OpenTowerMUD

A procedurally generated MUD (Multi-User Dungeon) server written in Go.

## Quick Start

```bash
cd server
go build -o opentowermud ./cmd/mud
./opentowermud
```

Connect: `telnet localhost 4000`

## Features

- Procedural world generation (200+ rooms, auto-expands)
- D20 combat system with ability scores
- Magic system (spells, mana, cooldowns)
- Player persistence (SQLite or PostgreSQL)
- Day/night cycle
- Multiplayer (say, tell, who)

## Commands

Type `help` in-game for command list.

## Database Configuration

OpenTowerMUD supports both SQLite (default) and PostgreSQL for player data.

### SQLite (Default)

No configuration needed. Data is stored in `data/opentowermud.db`.

### PostgreSQL

1. Start PostgreSQL using Docker Compose:
   ```bash
   docker-compose up -d
   ```

2. Copy and configure `server/data/server.yaml.example`:
   ```bash
   cp server/data/server.yaml.example server/data/server.yaml
   ```

3. Edit `server/data/server.yaml`:
   ```yaml
   database:
     driver: postgres
     postgres:
       host: localhost
       port: 5435
       user: opentower
       password: opentower
       database: opentower
       sslmode: disable
   ```

4. Run the server - schema is created automatically.

### Migrating from SQLite to PostgreSQL

Use the migration tool to transfer existing data:

```bash
cd server
go build -o migrate-to-postgres ./cmd/migrate-to-postgres

# Dry run (preview only)
./migrate-to-postgres --dry-run

# Run migration
./migrate-to-postgres \
  --sqlite data/opentowermud.db \
  --host localhost \
  --port 5435 \
  --user opentower \
  --password opentower \
  --database opentower
```

## Development

See [CLAUDE.md](CLAUDE.md) for architecture and code philosophy.

### Tests

```bash
# Unit tests
go test ./...

# Integration tests
./scripts/run-integration-tests.sh

# PostgreSQL integration tests (requires running PostgreSQL)
OTM_TEST_POSTGRES=1 go test ./internal/database/... -v -run "TestPostgres"
```
