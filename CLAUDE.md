# OpenTowerMUD

A procedurally generated MUD (Multi-User Dungeon) server written in Go, featuring a vertical tower dungeon with Wave Function Collapse-based floor generation.

## Project Structure

```
server/
+-- cmd/mud/           # Server entry point
+-- internal/          # All packages (Go convention)
|   +-- command/       # Command handling
|   +-- player/        # Player state, portal memory
|   +-- world/         # Rooms, room types
|   +-- tower/         # Tower structure, floor generation, spawning
|   +-- wfc/           # Wave Function Collapse algorithm
|   +-- npc/           # NPCs and mobs (with tiers)
|   +-- items/         # Item system (with loot tiers)
|   +-- server/        # TCP server, client handling
|   +-- spells/        # Magic system
|   +-- stats/         # Ability scores, dice
|   +-- database/      # SQLite player persistence
|   +-- ...
+-- data/              # YAML configs
|   +-- city_rooms.yaml   # 10-room walled city (floor 0)
|   +-- mobs.yaml         # Mob definitions with tiers
|   +-- items.yaml        # Items with loot tiers
|   +-- npcs.yaml         # City NPCs
|   +-- spells.yaml       # Spell definitions
|   +-- tower.yaml        # Tower save file (generated)
+-- test/              # Integration tests
```

## Build & Run

```bash
cd server
go build -o opentowermud ./cmd/mud
./opentowermud
```

Connect via telnet: `telnet localhost 4000`

## Tower Architecture

The game world is a vertical tower:

- **Floor 0 (City)**: 10-room walled city with NPCs, shops, altar, portal
- **Floor 1+**: Procedurally generated dungeon floors via WFC
- **Every 10 floors**: Boss floor with boss room and boss mob
- **Treasure rooms**: 1-3 per floor with tiered loot

### Floor Generation (WFC)

Each floor is generated on-demand using Wave Function Collapse:
- 20-50 rooms per floor
- Tile types: corridor, room, dead-end, stairs, treasure, boss
- All rooms guaranteed connected via BFS verification
- Deterministic with seed (same seed = same floor)

### Scaling System

Difficulty scales by floor:
- **Mob Tiers**: 1 (floors 1-5), 2 (6-10), 3 (11-20), 4 (21+)
- **Loot Tiers**: 1-5 (common to legendary)
- **Stat Scaling**: HP/Damage/XP scale with floor number
- **Boss Floors**: Every 10th floor (10, 20, 30...)

### Portal System

- Players discover portals when visiting stairway rooms
- `portal` command lists discovered floors
- `portal <floor>` teleports to that floor's portal room
- Floor 0 (city) always available

## Code Philosophy

**Go's philosophy: "A little copying is better than a little dependency"**

- Prefer simplicity over abstraction
- Use interfaces only for cross-package boundaries
- Use concrete types for simple structs
- Only extract utilities when pattern appears 3+ times
- Don't over-engineer for hypothetical futures

## Key Features

- **Tower System**: Infinite vertical dungeon with WFC generation
- **D20 Combat**: Attack rolls vs AC, dice damage, ability modifiers
- **Magic System**: Spells with mana costs, cooldowns, stun effects
- **Tiered Loot**: 5 loot tiers from common to legendary
- **Scaled Mobs**: 4 mob tiers + boss mobs
- **Player Persistence**: SQLite accounts, characters, inventory
- **Portal Travel**: Discovered floor fast-travel system
- **Day/Night Cycle**: Time-based room descriptions

## Testing

```bash
# Run all unit tests
go test ./...

# Run tower tests with benchmarks
go test ./internal/tower/... -bench=.

# Run integration tests (requires running server)
go run ./cmd/testrunner
```
