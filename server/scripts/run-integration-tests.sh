#!/bin/bash
#
# Integration Test Runner for OpenTower MUD
#
# This script handles:
# - Building the server and test runner
# - Starting the server with test configuration
# - Running integration tests
# - Cleaning up server process on exit
#
# Logs are written to the scripts directory:
# - server.log: Server output
# - integration-tests.log: Test runner output
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(dirname "$SCRIPT_DIR")"
SERVER_PID=""
SERVER_PORT="${SERVER_PORT:-4000}"
SERVER_ADDR="localhost:${SERVER_PORT}"
VERBOSE="${VERBOSE:-false}"

# Log files
SERVER_LOG="$SCRIPT_DIR/server.log"
TEST_LOG="$SCRIPT_DIR/integration-tests.log"

# Parse command line args
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

cleanup() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo -e "${YELLOW}Stopping server (PID: $SERVER_PID)...${NC}"
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    # Clean up test database
    rm -f "data/test/players_test.db" "data/test/players_test.db-wal" "data/test/players_test.db-shm"
}

trap cleanup EXIT INT TERM

echo "============================================================"
echo "OpenTower MUD - Integration Test Runner"
echo "============================================================"
echo ""

cd "$SERVER_DIR"

# Build server
echo -e "${YELLOW}Building server...${NC}"
go build -o opentowermud ./cmd/mud
echo -e "${GREEN}Server built successfully${NC}"

# Build test runner
echo -e "${YELLOW}Building test runner...${NC}"
go build -o testrunner ./cmd/testrunner
echo -e "${GREEN}Test runner built successfully${NC}"

echo ""

# Check if port is already in use
if nc -z localhost "$SERVER_PORT" 2>/dev/null; then
    echo -e "${RED}Error: Port $SERVER_PORT is already in use${NC}"
    echo "Please stop any existing server or set SERVER_PORT to a different value"
    exit 1
fi

# Clean up any existing test database and stale tower data
TEST_DB="data/test/players_test.db"
rm -f "$TEST_DB" "$TEST_DB-wal" "$TEST_DB-shm"
rm -f "data/tower.yaml"

# Start server with test configuration
echo -e "${YELLOW}Starting server on port $SERVER_PORT with test configuration...${NC}"
echo "Server logs: $SERVER_LOG"
./opentowermud \
    --readonly \
    --port "$SERVER_PORT" \
    --db "$TEST_DB" \
    --seed 42 \
    --config data/test/server_test.yaml \
    --npcs data/test/npcs_test.yaml \
    --mobs data/test/mobs_test.yaml \
    --items data/test/items_test.yaml \
    --recipes data/test/recipes_test.yaml \
    --chatfilter data/test/chat_filter_test.yaml \
    --quests data/quests.yaml \
    > "$SERVER_LOG" 2>&1 &

SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# Wait for server to be ready
echo -n "Waiting for server to start"
MAX_WAIT=10
WAITED=0
while ! nc -z localhost "$SERVER_PORT" 2>/dev/null; do
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo ""
        echo -e "${RED}Error: Server failed to start within ${MAX_WAIT} seconds${NC}"
        exit 1
    fi
    echo -n "."
    sleep 1
    WAITED=$((WAITED + 1))
done
echo ""
echo -e "${GREEN}Server is ready${NC}"

echo ""

# Run integration tests
echo -e "${YELLOW}Running integration tests...${NC}"
echo "Test logs: $TEST_LOG"
echo ""

VERBOSE_FLAG=""
if [ "$VERBOSE" = "true" ]; then
    VERBOSE_FLAG="-v"
fi

# Run tests and tee output to both console and log file
# Use pipefail to capture testrunner's exit code through the pipe
set -o pipefail
if ./testrunner -addr "$SERVER_ADDR" $VERBOSE_FLAG 2>&1 | tee "$TEST_LOG"; then
    echo ""
    echo -e "${GREEN}All integration tests passed!${NC}"
    EXIT_CODE=0
else
    echo ""
    echo -e "${RED}Some integration tests failed${NC}"
    echo "Check $TEST_LOG and $SERVER_LOG for details"
    EXIT_CODE=1
fi
set +o pipefail

echo ""
echo "============================================================"

exit $EXIT_CODE
