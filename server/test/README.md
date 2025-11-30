# Integration Testing System

Automated multi-client testing framework for OpenTower MUD server.

## Overview

This testing system creates multiple simulated clients that connect to the MUD server and run through various scenarios to verify functionality.

## Components

- **`scripts/run-integration-tests.sh`** - All-in-one test runner script
- **`internal/testclient/`** - Test client package with connection handling
- **`test/scenarios.go`** - Test scenarios and assertions
- **`cmd/testrunner/`** - Main test runner executable

## Running Tests

### Quick Start (Recommended)

Use the all-in-one script that handles server startup, testing, and cleanup:

```bash
cd server
./scripts/run-integration-tests.sh
```

The script will:
1. Build the server and test runner
2. Start the server with test configuration (read-only mode)
3. Wait for the server to be ready
4. Run all integration tests
5. Stop the server and clean up

**Options:**
```bash
# Run on a custom port
SERVER_PORT=4001 ./scripts/run-integration-tests.sh

# Enable verbose output (shows each command and result)
./scripts/run-integration-tests.sh -v

# Combine options
SERVER_PORT=4001 ./scripts/run-integration-tests.sh -v
```

### Manual Testing

If you prefer to run the server and tests separately:

1. Start the MUD server with test configuration files:
```bash
cd server
./opentowermud --readonly --port 4000 --db data/test/players_test.db --world data/test/world_test.yaml --npcs data/test/npcs_test.yaml --mobs data/test/mobs_test.yaml --items data/test/items_test.yaml --chatfilter data/test/chat_filter_test.yaml
```

**Important:**
- The `--readonly` flag prevents the world file from being modified during tests
- The `--port` flag allows running on a custom port (default: 4000)
- The `--db` flag uses a separate test database (cleaned up after tests)
- The test configuration files in `data/test/` are required for all tests to pass:
  - `world_test.yaml` - Test world with specific room IDs
  - `npcs_test.yaml` - Town NPCs (bartender, merchant, etc.)
  - `mobs_test.yaml` - Test mobs with fast respawn times (10 seconds)
  - `items_test.yaml` - Items placed in test world rooms
  - `chat_filter_test.yaml` - Chat filter with test banned words
  - `players_test.db` - Test database (auto-created, auto-deleted)

2. In another terminal, run the tests:
```bash
cd server
go run ./cmd/testrunner
```

### Build and Run

```bash
cd server
go build -o opentowermud ./cmd/mud
go build -o testrunner ./cmd/testrunner
./testrunner
```

### Custom Server Address

```bash
./testrunner -addr localhost:4000
```

### Verbose Output

```bash
./testrunner -v
```

This shows detailed logging for each test action and result.

## Current Tests (33/33 passing)

### Phase 1-2 Tests (Implemented)
1. **Basic Connection** - Verify clients can connect and receive welcome
2. **Movement** - Test north/south movement between rooms
3. **Item Pickup/Drop** - Test get/drop commands with items
4. **Multiple Clients Movement** - Test 3 clients moving to different rooms simultaneously
5. **Say Command** - Test room-wide message broadcasting
6. **Tell Command** - Test private player-to-player messaging

### Phase 3 Tests (Implemented)
7. **Weight/Capacity System** - Test carry weight limits and capacity checks
8. **Inventory Display** - Test inventory command with item details

### Phase 10 Tests (Implemented)
9. **Multi-Target Combat** - Test cooperative group combat with XP split

### Phase 12 Tests (Implemented)
10. **Un-Attackable NPCs** - Test friendly NPC protection system

### Phase 15 Tests (Implemented)
11. **NPC Respawning** - Test NPC respawn system with configurable timers

### Phase 16 Tests (Implemented)
12. **World Expansion** - Test dynamic world expansion when approaching world boundaries

### Phase 19 Tests (Implemented)
13. **Chat Filter (REPLACE mode)** - Test word filtering in say/tell commands

### Phase 20 Tests (Implemented)
14. **Account Registration** - Test new account creation and game entry
15. **Login Flow** - Test login with existing credentials and location persistence
16. **Invalid Login** - Test that invalid credentials are rejected
17. **Banned Account** - Test that banned accounts cannot log in
18. **Inventory Persistence** - Test inventory persists across sessions

### Phase 21 Tests (Implemented)
19. **Admin Commands Hidden** - Test non-admins can't access admin commands
20. **Admin Announce** - Test admin announcement broadcasts to all players

### Phase 22 Tests (Implemented)
21. **Pray at Altar** - Test pray command heals player at temple altar
22. **Pray Without Altar** - Test pray command fails without altar
23. **Consider Self** - Test consider self/me shows player stats
24. **Portal Command** - Test portal shows available destinations
25. **Look at Features** - Test looking at altar and portal features

### Phase 23 Tests (Implemented)
26. **Spell Casting** - Test basic spell mechanics (cast, mana cost, cooldown)
27. **Heal Other Player** - Test healing another player with heal spell
28. **Dazzle Spell** - Test room-wide stun effect on hostile NPCs
29. **Dazzle In Combat** - Test stunned NPCs don't attack during combat

### Phase 24 Tests (Implemented)
30. **Player Level Up** - Test XP gain and level-up from combat

### Phase 25 Tests (Implemented)
31. **Ability Scores** - Test ability score assignment and display

### Phase 26 Tests (Implemented)
32. **Attack Rolls** - Test D20 attack roll mechanics with hit/miss outcomes
33. **Spell Damage with Modifiers** - Test spell damage scales with INT modifier

**All 33 tests passing (100% success rate)**

## Test Client API

The `testclient` package provides a full-featured test client with authentication support:

```go
// Create client and register a new account (handles full auth flow)
client, err := testclient.NewTestClient("PlayerName", "localhost:4000")
defer client.Close()

// Login with existing credentials
creds := testclient.Credentials{
    Username:      "existinguser",
    Password:      "password123",
    CharacterName: "MyCharacter",
}
client, err := testclient.NewTestClientWithLogin(creds, "localhost:4000")
defer client.Close()

// Raw connection (no auth) for testing auth flow itself
client, err := testclient.NewTestClientRaw("localhost:4000")
defer client.Close()

// Send commands
client.SendCommand("north")
client.SendCommand("get stick")

// Check for messages
if client.WaitForMessage("Town Square", 1*time.Second) {
    // Message was received
}

// Wait for any of multiple messages
text, found := client.WaitForAnyMessage([]string{"success", "failure"}, 2*time.Second)

// Get all messages
messages := client.GetMessages()

// Check if message exists
if client.HasMessage("some text") {
    // Found it
}

// Clear message buffer
client.ClearMessages()

// Debug print
client.PrintMessages()
```

## Adding New Tests

1. Add a test function to `test/scenarios.go`:
```go
func TestMyNewFeature(serverAddr string) TestResult {
    client, err := testclient.NewTestClient("TestName", serverAddr)
    if err != nil {
        return TestResult{
            Name: "My Feature",
            Passed: false,
            Message: fmt.Sprintf("Connection failed: %v", err),
        }
    }
    defer client.Close()

    // Your test logic here

    return TestResult{
        Name: "My Feature",
        Passed: true,
        Message: "Feature works correctly",
    }
}
```

2. Add it to `RunAllTests()` in `scenarios.go`:
```go
results = append(results, TestMyNewFeature(serverAddr))
```

## Test Output

```
============================================================
Integration Test Results
============================================================

[[PASS]] Basic Connection
    Connected successfully, received welcome messages

[[PASS]] Movement
    Successfully moved between rooms

[[PASS]] Item Pickup/Drop
    Successfully picked up and dropped items

[[PASS]] Multiple Clients Movement
    3 clients moved to different rooms simultaneously

[[PASS]] Say Command
    Room-wide message broadcasting works

[[PASS]] Tell Command
    Private player-to-player messaging works

[[PASS]] Weight/Capacity System
    Carry weight limits enforced correctly

[[PASS]] Inventory Display
    Inventory shows item details with weight

[[PASS]] Multi-Target Combat
    Cooperative combat with XP split works

[[PASS]] Un-Attackable NPCs
    Friendly NPCs cannot be attacked

[[PASS]] NPC Respawning
    Goblin respawned successfully after 10 seconds

[[PASS]] World Expansion
    World expansion triggered after 1 moves

[[PASS]] Chat Filter (REPLACE mode)
    Chat filter correctly replaces banned words with asterisks in both say and tell

[[PASS]] Account Registration
    Successfully registered account 'RegTest12345' and entered game

[[PASS]] Login Flow
    Successfully logged back in and location persisted

[[PASS]] Invalid Login
    Invalid credentials correctly rejected

[[PASS]] Banned Account
    Banned account correctly rejected with ban message

[[PASS]] Inventory Persistence
    Inventory successfully persisted across sessions

[[PASS]] Spell Casting
    Basic spell mechanics work correctly

[[PASS]] Heal Other Player
    Successfully healed another player

[[PASS]] Dazzle Spell
    Room-wide stun affects all hostile NPCs

[[PASS]] Dazzle In Combat
    Stunned NPCs do not attack during combat

[[PASS]] Player Level Up
    Player successfully leveled up from XP gain

[[PASS]] Ability Scores
    Ability scores assigned and displayed correctly

[[PASS]] Attack Rolls (Dice Combat)
    D20 attack roll mechanics work correctly

[[PASS]] Spell Damage (INT Modifier)
    Spell damage scales with INT modifier

============================================================
Total: 33 tests, 33 passed, 0 failed
============================================================
```

## CI/CD Integration

The test runner exits with code 0 on success, 1 on failure - perfect for CI pipelines:

```bash
./testrunner || exit 1
```

## Future Enhancements

- Add timeout configuration
- Parallel test execution
- Test coverage reporting
- Performance benchmarks
- Stress testing (100+ concurrent clients)
