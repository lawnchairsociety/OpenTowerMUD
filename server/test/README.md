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
VERBOSE=true ./scripts/run-integration-tests.sh

# Filter to specific tests
./testrunner -filter "Mail"
```

### Manual Testing

If you prefer to run the server and tests separately:

1. Start the MUD server with test configuration:
```bash
cd server
./opentowermud --readonly --port 4000 --db data/test/players_test.db --config data/test/server_test.yaml
```

2. In another terminal, run the tests:
```bash
./testrunner -addr localhost:4000
```

### Test Runner Options

```bash
./testrunner -h
  -addr string     Server address (default "localhost:4000")
  -filter string   Run only tests containing this string
  -list            List all available tests
  -v               Verbose output
```

## Test Groups (119 tests)

### Group 1: Connection & Account System
- Basic connection and welcome
- Account registration
- Login flow and persistence
- Invalid/banned account handling

### Group 2: Communication & Social
- Say command (room broadcast)
- Tell command (private messages)
- Who command
- Ignore/unignore system
- Anti-spam and chat filter

### Group 3: Character Info & Stats
- Score/stats display
- Class and race commands
- Ability scores
- Title system
- Skills display

### Group 4: Inventory & Shopping
- Item pickup/drop
- Buy/sell from NPCs
- Equipment wear/remove
- Consumables (eat/drink)
- Weight/capacity limits

### Group 5: Combat System
- Attack rolls (D20 mechanics)
- Combat threat and targeting
- Flee command
- Mob kills and XP
- Consider command

### Group 6: Magic System
- Spell casting
- Heal other player
- Bless and buff spells
- Spell damage with INT modifier

### Group 7: Room Features & Tower
- Portal command
- Look at features
- Tower climbing
- Pray at altar

### Group 8: NPCs & Training
- Unattackable NPC protection
- Mob respawning
- Train command
- Learn from trainers

### Group 9: Crafting System
- Crafting stations (forge, workbench, etc.)
- Recipe learning
- Material requirements
- Craft command

### Group 10: Quest System
- Quest listing
- Quest acceptance
- Progress tracking
- Quest completion

### Group 11: Admin Commands
- Admin command access control
- Announce broadcasts

### Group 12: Player Stalls
- Stall creation
- Item listing
- Buying from stalls
- Stall management

### Group 13: Multi-City Expansion
- Mail send/receive
- Per-player mail indexing
- Race selection (5 races)
- Portal cross-city travel
- Labyrinth gate access
- Labyrinth navigation
- Lore NPC interaction
- Title system
- Multiple cities verification

## Test Client API

```go
// Create client and register a new account
client, err := testclient.NewTestClient("PlayerName", "localhost:4000")
defer client.Close()

// Login with existing credentials
creds := testclient.Credentials{
    Username:      "existinguser",
    Password:      "password123",
    CharacterName: "MyCharacter",
}
client, err := testclient.NewTestClientWithLogin(creds, "localhost:4000")

// Send commands
client.SendCommand("north")
client.SendCommand("mail read 1")

// Check for messages
if client.WaitForMessage("Town Square", 1*time.Second) {
    // Message received
}

// Check if message exists
if client.HasMessage("Mail sent") {
    // Found it
}

// Get all messages
messages := client.GetMessages()

// Clear message buffer
client.ClearMessages()
```

## Adding New Tests

1. Add a test function to the appropriate file in `test/`:
```go
func TestMyFeature(serverAddr string) TestResult {
    client, err := testclient.NewTestClient(uniqueName("mytest"), serverAddr)
    if err != nil {
        return TestResult{
            Name: "My Feature",
            Passed: false,
            Message: fmt.Sprintf("Connection failed: %v", err),
        }
    }
    defer client.Close()

    // Test logic here
    client.SendCommand("mycommand")
    time.Sleep(300 * time.Millisecond)

    if !client.HasMessage("expected output") {
        return TestResult{Name: "My Feature", Passed: false, Message: "Expected output not found"}
    }

    return TestResult{Name: "My Feature", Passed: true, Message: "Feature works"}
}
```

2. Add to `RunAllTests()` and `getAllTests()` in `scenarios.go`

## CI/CD Integration

The test runner exits with code 0 on success, 1 on failure:

```bash
./scripts/run-integration-tests.sh || exit 1
```

## Test Configuration

Test data files in `data/test/`:
- `server_test.yaml` - Test server configuration
- `players_test.db` - Test database (auto-created/deleted)
- `npcs/npcs_test.yaml` - Test NPCs
- `mobs/mobs_test.yaml` - Test mobs (fast respawn)
