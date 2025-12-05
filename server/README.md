# OpenTowerMUD Server

A procedurally generated MUD (Multi-User Dungeon) server featuring a vertical tower dungeon with Wave Function Collapse-based floor generation.

## Quick Start

1. Run the server binary:
   ```bash
   ./opentowermud        # Linux/macOS
   opentowermud.exe      # Windows
   ```

2. Connect via telnet or WebSocket:
   ```bash
   # Telnet (traditional MUD clients)
   telnet localhost 4000

   # WebSocket (web clients)
   ws://localhost:4443/ws
   ```

3. Create an account and start playing!

## Configuration

The server uses YAML configuration files in the `data/` directory:

| File | Description |
|------|-------------|
| `city_rooms.yaml` | The 10-room walled city (floor 0) |
| `mobs.yaml` | Monster definitions with tiers |
| `items.yaml` | Item definitions with loot tiers |
| `npcs.yaml` | City NPC definitions |
| `spells.yaml` | Spell definitions |
| `logging.yaml` | Logging configuration |
| `chat_filter.yaml` | Chat filter rules |
| `tower.yaml` | Tower save file (auto-generated) |

## Game Overview

### The Tower

- **Floor 0 (City)**: Safe zone with shops, altar, and portal
- **Floor 1+**: Procedurally generated dungeon floors
- **Every 10 floors**: Boss floor with increased rewards

### Commands

Once connected, use `help` to see available commands. Common commands:

- `look` - Examine your surroundings
- `north/south/east/west/up/down` - Move between rooms
- `attack <target>` - Combat
- `cast <spell> <target>` - Cast spells
- `inventory` - View your items
- `portal` - Fast-travel to discovered floors

### Difficulty Scaling

Difficulty increases as you climb:
- **Floors 1-5**: Tier 1 mobs
- **Floors 6-10**: Tier 2 mobs
- **Floors 11-20**: Tier 3 mobs
- **Floors 21+**: Tier 4 mobs

## Data Persistence

- Player accounts and characters are stored in SQLite (`opentowermud.db`)
- Tower state is saved to `data/tower.yaml`

## Deployment

For production deployment with TLS/SSL support, see [deploy/README.md](deploy/README.md).

The recommended setup uses a reverse proxy (nginx or Caddy) for:
- Secure WebSocket connections (`wss://`)
- Automatic certificate management
- HTTP to HTTPS redirect

## Building from Source

```bash
cd server
go build -o opentowermud ./cmd/mud
```

## License

See LICENSE file in the repository.
