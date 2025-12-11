# OpenTowerMUD Server

A procedurally generated MUD (Multi-User Dungeon) server featuring multiple racial cities, themed tower dungeons, and a massive labyrinth connecting them all.

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

The server uses YAML configuration files in the `data/` directory. See [data/README.md](data/README.md) for details.

## Game World

### Five Racial Cities

Each race has their own starting city with unique themes:

| City | Race | Tower Theme |
|------|------|-------------|
| Ironhaven | Human | Arcane Spire |
| Sylvanthal | Elf | Diseased World Tree |
| Khazad-Karn | Dwarf | Descending Mines |
| Cogsworth | Gnome | Mechanical Tower |
| Skullgar | Orc | Beast-Skull Tower |

### The Towers

Each city has a 25-floor tower with themed mobs and a final boss:
- **Floors 1-5**: Tier 1 mobs (easy)
- **Floors 6-10**: Tier 2 mobs (medium)
- **Floors 11-20**: Tier 3 mobs (hard)
- **Floors 21-25**: Tier 4 mobs (elite) + Final Boss

### The Infinity Spire

A 100-floor unified tower unlocked after all 5 racial tower bosses are defeated (tracked globally). Features the ultimate challenge: The Blighted One.

### The Great Labyrinth

A 40x40 procedurally generated maze connecting all five cities:
- Navigate between cities through underground passages
- Discover lore NPCs who reveal world history
- Find wandering merchants with rare goods
- Battle labyrinth-dwelling creatures
- Earn titles: "Wanderer of the Ways" and "Keeper of Forgotten Lore"

## Features

### Playable Races
- **Human** - Versatile, bonus ability point
- **Elf** - Agile, magic affinity
- **Dwarf** - Sturdy, poison resistance
- **Gnome** - Clever, tinker bonuses
- **Orc** - Strong, combat bonuses

### Classes
- Warrior, Mage, Cleric, Rogue, Ranger, Paladin
- Multiclassing available through trainers

### Mail System
Send mail with gold and item attachments to other players. Access mailboxes in city centers.

### Portal System
- Discover portals when visiting stairway rooms
- Fast-travel to any discovered floor
- City portals always available

### Other Features
- D20-based combat with ability modifiers
- Crafting system (Blacksmithing, Leatherworking, Alchemy, Enchanting)
- Quest system with NPC quest givers
- Day/night cycle affecting room descriptions
- Title system for achievements

## Commands

Use `help` in-game to see all available commands. Common commands:

| Command | Description |
|---------|-------------|
| `look` | Examine surroundings |
| `north/south/east/west/up/down` | Movement |
| `attack <target>` | Enter combat |
| `cast <spell> [target]` | Cast spells |
| `inventory` | View items |
| `portal` | Fast-travel menu |
| `mail` | Access mailbox |
| `races` | View race information |
| `title` | View/set titles |

## Data Persistence

- Player accounts and characters: SQLite (`opentowermud.db`)
- Tower state: `data/tower.yaml`
- Boss defeat tracking: Persisted in database

## Deployment

For production deployment with TLS/SSL support, see [deploy/README.md](deploy/README.md).

## Building from Source

```bash
cd server
go build -o opentowermud ./cmd/mud
```

## Testing

```bash
# Run integration tests
./scripts/run-integration-tests.sh

# Run unit tests
go test ./...
```

See [test/README.md](test/README.md) for details.

## License

See LICENSE file in the repository.
