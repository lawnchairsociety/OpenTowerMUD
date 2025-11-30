package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/server"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 4000, "Server port")
	seed := flag.Int64("seed", 0, "World generation seed (default: random based on current time)")
	towerFile := flag.String("tower", "data/tower.yaml", "Path to tower save file")
	cityFile := flag.String("city", "data/city_rooms.yaml", "Path to city rooms YAML file")
	npcsFile := flag.String("npcs", "data/npcs.yaml", "Path to NPCs YAML file")
	mobsFile := flag.String("mobs", "data/mobs.yaml", "Path to mobs YAML file")
	itemsFile := flag.String("items", "data/items.yaml", "Path to items YAML file")
	spellsFile := flag.String("spells", "data/spells.yaml", "Path to spells YAML file")
	loggingConfig := flag.String("logging", "data/logging.yaml", "Path to logging config YAML file")
	chatFilterConfig := flag.String("chatfilter", "data/chat_filter.yaml", "Path to chat filter config YAML file")
	pilgrimMode := flag.Bool("pilgrim", false, "Enable pilgrim mode (peaceful exploration, no combat)")
	readOnly := flag.Bool("readonly", false, "Run in read-only mode (world changes won't be saved to disk)")
	dbFile := flag.String("db", "data/opentowermud.db", "Path to player database file")
	makeAdmin := flag.String("make-admin", "", "Promote an existing account to admin and exit (requires username)")
	flag.Parse()

	// Handle --make-admin flag (promotes account and exits)
	if *makeAdmin != "" {
		handleMakeAdmin(*makeAdmin, *dbFile)
		return
	}

	// Initialize logger first (before any logging)
	logConfig, _ := logger.LoadConfig(*loggingConfig)
	logger.Initialize(logConfig)

	logger.Info("Starting Open Tower MUD Server")

	// Use provided seed or generate from time
	worldSeed := *seed
	if worldSeed == 0 {
		worldSeed = time.Now().UnixNano()
		logger.Info("World seed selected", "seed", worldSeed, "random", true)
	} else {
		logger.Info("World seed selected", "seed", worldSeed, "random", false)
	}

	// Initialize the world
	gameWorld := world.NewWorld()

	// Set read-only mode BEFORE initialization (so initial generation won't save)
	if *readOnly {
		gameWorld.SetReadOnly(true)
		logger.Info("Server running in READ-ONLY MODE - world changes won't be saved")
	}

	// Initialize the tower (load from file or create new)
	gameTower, err := initializeTower(worldSeed, *towerFile, *cityFile, *mobsFile, *itemsFile)
	if err != nil {
		log.Fatalf("Failed to initialize tower: %v", err)
	}

	// Configure auto-save for tower (unless read-only)
	if !*readOnly {
		gameTower.SetSavePath(*towerFile)
	}

	// Wire tower to world
	gameWorld.SetTower(gameTower)

	// Add city rooms to world's room map for direct lookup
	cityFloor := gameTower.GetFloorIfExists(0)
	if cityFloor != nil {
		for _, room := range cityFloor.GetRooms() {
			gameWorld.AddRoom(room)
		}
	}

	gameWorld.InitializeWithPaths(worldSeed, *towerFile, *npcsFile, *mobsFile, *itemsFile)

	// Load items config (needed for player inventory loading)
	itemsConfig, err := items.LoadItemsFromYAML(*itemsFile)
	if err != nil {
		log.Fatalf("Failed to load items config: %v", err)
	}

	// Load spells config
	spellRegistry := spells.NewSpellRegistry()
	if err := spellRegistry.LoadFromYAML(*spellsFile); err != nil {
		log.Fatalf("Failed to load spells config: %v", err)
	}
	logger.Info("Spells loaded", "count", len(spellRegistry.GetAllSpells()))

	// Initialize player database
	db, err := database.Open(*dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	logger.Info("Player database initialized", "path", *dbFile)

	// Create and start the server
	addr := fmt.Sprintf(":%d", *port)
	srv := server.NewServer(addr, gameWorld, *pilgrimMode)

	// Set database, items config, and spell registry on server
	srv.SetDatabase(db)
	srv.SetItemsConfig(itemsConfig)
	srv.SetSpellRegistry(spellRegistry)

	// Load and set chat filter
	filterCfg, err := chatfilter.LoadConfig(*chatFilterConfig)
	if err != nil {
		logger.Warning("Failed to load chat filter config, chat filter disabled", "path", *chatFilterConfig, "error", err)
	} else {
		cf := chatfilter.New(filterCfg)
		srv.SetChatFilter(cf)
		if filterCfg.Enabled {
			logger.Info("Chat filter enabled", "mode", filterCfg.Mode, "words", len(filterCfg.BannedWords))
		}
	}

	if *pilgrimMode {
		logger.Info("Server running in PILGRIM MODE - combat disabled")
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	logger.Info("MUD Server running", "port", *port)
	logger.Info("Press Ctrl+C to shutdown")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server")
	srv.Shutdown()
	logger.Info("Server stopped")
}

// handleMakeAdmin promotes an account to admin and exits
func handleMakeAdmin(username, dbFile string) {
	// Open database
	db, err := database.Open(dbFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get account by username
	account, err := db.GetAccountByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Account '%s' not found\n", username)
		os.Exit(1)
	}

	// Check if already admin
	if account.IsAdmin {
		fmt.Printf("Account '%s' is already an admin.\n", username)
		os.Exit(0)
	}

	// Promote to admin
	if err := db.SetAdmin(account.ID, true); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to promote account: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Account '%s' has been promoted to admin.\n", username)
}

// initializeTower loads an existing tower from file or creates a new one with the city
func initializeTower(seed int64, towerFile, cityFile, mobsFile, itemsFile string) (*tower.Tower, error) {
	// Load mob configuration for spawning
	mobConfig, err := npc.LoadNPCsFromYAML(mobsFile)
	if err != nil {
		logger.Warning("Failed to load mobs config, mob spawning disabled", "error", err)
	}

	// Load items configuration for loot spawning
	itemConfig, err := items.LoadItemsFromYAML(itemsFile)
	if err != nil {
		logger.Warning("Failed to load items config, loot spawning disabled", "error", err)
	}

	// Try to load existing tower
	if tower.TowerFileExists(towerFile) {
		logger.Info("Loading tower from file", "path", towerFile)
		t, err := tower.LoadTower(towerFile)
		if err != nil {
			logger.Warning("Failed to load tower, creating new one", "error", err)
		} else {
			// Set configs on loaded tower
			if mobConfig != nil {
				t.SetMobConfig(mobConfig)
			}
			if itemConfig != nil {
				t.SetItemConfig(itemConfig)
			}
			logger.Info("Tower loaded", "floors", t.FloorCount(), "highest", t.GetHighestFloor())
			return t, nil
		}
	}

	// Create new tower
	logger.Info("Creating new tower", "seed", seed)
	t := tower.NewTower(seed)

	// Set configs for spawning during floor generation
	if mobConfig != nil {
		t.SetMobConfig(mobConfig)
	}
	if itemConfig != nil {
		t.SetItemConfig(itemConfig)
	}

	// Load and create city floor (floor 0)
	cityFloor, err := tower.LoadAndCreateCity(cityFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load city: %w", err)
	}

	// Validate city floor
	if err := tower.ValidateCityFloor(cityFloor); err != nil {
		return nil, fmt.Errorf("city validation failed: %w", err)
	}

	// Add city floor to tower
	t.SetFloor(0, cityFloor)
	logger.Info("City floor created", "rooms", len(cityFloor.GetRooms()))

	// Save the initial tower state
	if err := tower.SaveTower(t, towerFile); err != nil {
		logger.Warning("Failed to save initial tower state", "error", err)
	}

	return t, nil
}
