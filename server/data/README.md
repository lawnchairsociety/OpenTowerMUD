# Data Files

Configuration and content files for the MUD server.

## Directory Structure

```
data/
├── cities/              # Racial city definitions
│   ├── human_city.yaml
│   ├── elf_city.yaml
│   ├── dwarf_city.yaml
│   ├── gnome_city.yaml
│   └── orc_city.yaml
├── towers/              # Pre-generated tower floors
│   ├── human/           # 25 floors
│   ├── elf/             # 25 floors
│   ├── dwarf/           # 25 floors
│   ├── gnome/           # 25 floors
│   ├── orc/             # 25 floors
│   └── unified/         # 100 floors (Infinity Spire)
├── labyrinth/           # The Great Labyrinth
│   └── labyrinth.yaml   # 40x40 maze connecting cities
├── mobs/                # Monster definitions
│   └── mobs.yaml        # All hostile creatures by tier/theme
├── npcs/                # NPC definitions
│   ├── human_npcs.yaml
│   ├── elf_npcs.yaml
│   ├── dwarf_npcs.yaml
│   ├── gnome_npcs.yaml
│   ├── orc_npcs.yaml
│   └── labyrinth_npcs.yaml
├── quests/              # Quest definitions
├── world/               # Shared world templates
└── test/                # Test configuration files
```

## Core Configuration Files

| File | Description |
|------|-------------|
| `server.yaml` | Server configuration (ports, paths, features) |
| `races.yaml` | Playable race definitions and bonuses |
| `items.yaml` | Item definitions with stats and loot tiers |
| `spells.yaml` | Spell definitions (51 spells) |
| `recipes.yaml` | Crafting recipes (29 recipes) |
| `help.yaml` | In-game help system content |
| `text.yaml` | UI text and messages |
| `logging.yaml` | Logging configuration |
| `chat_filter.yaml` | Chat filter rules |
| `name_filter.yaml` | Character name filter rules |

## Cities

Each racial city (`cities/*.yaml`) contains:
- Town square with portal and mailbox
- Shops (general store, armory, alchemist)
- Training facilities
- Temple with altar
- Tavern
- City gates connecting to the labyrinth
- Tower entrance

## Towers

Pre-generated tower floors using Wave Function Collapse algorithm:
- **Racial towers** (25 floors each): Themed mobs and final boss
- **Unified tower** (100 floors): Unlocked after all racial bosses defeated

Floor files contain room layouts, connections, and spawn points.

## Mobs

Monster definitions in `mobs/mobs.yaml` include:
- **Tiers 1-4**: Scaling difficulty
- **Tower tags**: Theme-specific spawning (arcane, nature, mechanical, etc.)
- **Boss mobs**: Tower final bosses
- Stats, loot tables, gold drops

## NPCs

City NPCs (`npcs/*.yaml`) include:
- Shopkeepers and merchants
- Class trainers
- Crafting trainers
- Quest givers
- City guards
- Lore NPCs (labyrinth)

## Runtime Files

Created automatically at runtime:
- `opentowermud.db` - SQLite player database
- `tower.yaml` - Tower state (if dynamic generation enabled)

## Test Configuration

The `test/` directory contains isolated test configurations:
- Separate database
- Fast mob respawn times
- Controlled test scenarios

See `test/README.md` for integration testing details.
