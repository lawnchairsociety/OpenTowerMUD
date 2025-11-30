# OpenTowerdMUD

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
- Player persistence (SQLite)
- Day/night cycle
- Multiplayer (say, tell, who)

## Commands

Type `help` in-game for command list.

## Development

See [CLAUDE.md](CLAUDE.md) for architecture and code philosophy.

### Tests

```bash
# Unit tests
go test ./...

# Integration tests
./scripts/run-integration-tests.sh
```
