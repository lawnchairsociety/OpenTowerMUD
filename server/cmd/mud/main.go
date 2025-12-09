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
	"github.com/lawnchairsociety/opentowermud/server/internal/config"
	"github.com/lawnchairsociety/opentowermud/server/internal/crafting"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/help"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/text"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/namefilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
	"github.com/lawnchairsociety/opentowermud/server/internal/server"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 4000, "Telnet server port")
	wsPort := flag.Int("wsport", 4443, "WebSocket server port")
	seed := flag.Int64("seed", 0, "World generation seed (default: random based on current time)")
	towerFile := flag.String("tower", "data/tower.yaml", "Path to tower save file")
	cityFile := flag.String("city", "data/cities/human_city.yaml", "Path to city rooms YAML file")
	npcsFile := flag.String("npcs", "data/npcs.yaml", "Path to NPCs YAML file")
	mobsFile := flag.String("mobs", "data/mobs.yaml", "Path to mobs YAML file")
	itemsFile := flag.String("items", "data/items.yaml", "Path to items YAML file")
	racesFile := flag.String("races", "data/races.yaml", "Path to races YAML file")
	spellsFile := flag.String("spells", "data/spells.yaml", "Path to spells YAML file")
	recipesFile := flag.String("recipes", "data/recipes.yaml", "Path to crafting recipes YAML file")
	questsFile := flag.String("quests", "data/quests.yaml", "Path to quests YAML file")
	helpFile := flag.String("help", "data/help.yaml", "Path to help YAML file")
	textFile := flag.String("text", "data/text.yaml", "Path to text YAML file")
	loggingConfig := flag.String("logging", "data/logging.yaml", "Path to logging config YAML file")
	chatFilterConfig := flag.String("chatfilter", "data/chat_filter.yaml", "Path to chat filter config YAML file")
	nameFilterConfig := flag.String("namefilter", "data/name_filter.yaml", "Path to name filter config YAML file")
	serverConfigFile := flag.String("config", "data/server.yaml", "Path to server config YAML file")
	pilgrimMode := flag.Bool("pilgrim", false, "Enable pilgrim mode (peaceful exploration, no combat)")
	readOnly := flag.Bool("readonly", false, "Run in read-only mode (world changes won't be saved to disk)")
	dbFile := flag.String("db", "data/opentowermud.db", "Path to player database file")
	makeAdmin := flag.String("make-admin", "", "Promote an existing account to admin and exit (requires username)")
	towerID := flag.String("tower-id", "human", "Tower identifier (e.g., human, elf, dwarf)")
	dataDir := flag.String("data-dir", "data", "Path to data directory")
	useStaticFloors := flag.Bool("static-floors", false, "Load floors from YAML files instead of generating via WFC")
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

	// Configure tower settings
	gameTower.SetTowerID(*towerID)
	gameTower.SetDataDir(*dataDir)
	gameTower.SetUseStaticFloors(*useStaticFloors)

	if *useStaticFloors {
		logger.Info("Tower using static floors", "tower_id", *towerID, "data_dir", *dataDir)
		// Preload all static floors at startup
		floorsLoaded, err := gameTower.PreloadStaticFloors(100) // Try up to 100 floors
		if err != nil {
			log.Fatalf("Failed to preload static floors: %v", err)
		}
		logger.Info("Static floors preloaded", "count", floorsLoaded)
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

	gameWorld.InitializeWithPaths(worldSeed, *towerFile, *npcsFile, *mobsFile)

	// Load items config (needed for player inventory loading)
	itemsConfig, err := items.LoadItemsFromYAML(*itemsFile)
	if err != nil {
		log.Fatalf("Failed to load items config: %v", err)
	}

	// Load races config
	racesConfig, err := race.LoadRacesFromYAML(*racesFile)
	if err != nil {
		logger.Warning("Failed to load races config, using defaults", "path", *racesFile, "error", err)
	} else {
		logger.Info("Races loaded", "count", len(racesConfig.Races))
	}

	// Load spells config
	spellRegistry := spells.NewSpellRegistry()
	if err := spellRegistry.LoadFromYAML(*spellsFile); err != nil {
		log.Fatalf("Failed to load spells config: %v", err)
	}
	logger.Info("Spells loaded", "count", len(spellRegistry.GetAllSpells()))

	// Load recipes config
	recipeRegistry := crafting.NewRecipeRegistry()
	if err := recipeRegistry.LoadFromYAML(*recipesFile); err != nil {
		logger.Warning("Failed to load recipes config, crafting disabled", "path", *recipesFile, "error", err)
	} else {
		logger.Info("Recipes loaded", "count", recipeRegistry.Count())
	}

	// Load quests config
	questRegistry := quest.NewQuestRegistry()
	if err := questRegistry.LoadFromYAML(*questsFile); err != nil {
		logger.Warning("Failed to load quests config, quests disabled", "path", *questsFile, "error", err)
	} else {
		logger.Info("Quests loaded", "count", questRegistry.Count())
	}

	// Load help system
	if err := help.Initialize(*helpFile); err != nil {
		logger.Warning("Failed to load help config, help system disabled", "path", *helpFile, "error", err)
	} else {
		logger.Info("Help system loaded", "path", *helpFile)
	}

	// Load text system
	if err := text.Initialize(*textFile); err != nil {
		logger.Warning("Failed to load text config, using fallback text", "path", *textFile, "error", err)
	} else {
		logger.Info("Text system loaded", "path", *textFile)
	}

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

	// Set database, items config, spell registry, recipe registry, and quest registry on server
	srv.SetDatabase(db)
	srv.SetItemsConfig(itemsConfig)
	srv.SetSpellRegistry(spellRegistry)
	srv.SetRecipeRegistry(recipeRegistry)
	srv.SetQuestRegistry(questRegistry)

	// Load and set server config (security settings, etc.)
	serverCfg, err := config.LoadConfig(*serverConfigFile)
	if err != nil {
		logger.Warning("Failed to load server config, using defaults", "path", *serverConfigFile, "error", err)
		serverCfg = config.DefaultConfig()
	}
	srv.SetServerConfig(serverCfg)
	if len(serverCfg.WebSocket.AllowedOrigins) == 0 {
		logger.Info("WebSocket CORS policy", "mode", "same-origin")
	} else if len(serverCfg.WebSocket.AllowedOrigins) == 1 && serverCfg.WebSocket.AllowedOrigins[0] == "*" {
		logger.Warning("WebSocket CORS allows all origins (not recommended for production)")
	} else {
		logger.Info("WebSocket CORS policy", "allowed_origins", serverCfg.WebSocket.AllowedOrigins)
	}

	// Set up dynamic spawn scaling based on player count
	srv.SetupDynamicSpawns(gameTower)

	// Load and set chat filter
	filterCfg, err := chatfilter.LoadConfig(*chatFilterConfig)
	if err != nil {
		logger.Warning("Failed to load chat filter config, chat filter disabled", "path", *chatFilterConfig, "error", err)
	} else {
		cf := chatfilter.New(filterCfg)
		srv.SetChatFilter(cf)
		srv.SetChatFilterConfig(filterCfg)
		if filterCfg.Enabled {
			logger.Info("Chat filter enabled", "mode", filterCfg.Mode, "words", len(filterCfg.BannedWords))
		}
		if filterCfg.Antispam != nil && filterCfg.Antispam.Enabled {
			logger.Info("Anti-spam enabled", "max_messages", filterCfg.Antispam.MaxMessages, "time_window", filterCfg.Antispam.TimeWindowSeconds)
		}
	}

	// Load and set name filter
	nameCfg, err := namefilter.LoadConfig(*nameFilterConfig)
	if err != nil {
		logger.Warning("Failed to load name filter config, name filter disabled", "path", *nameFilterConfig, "error", err)
	} else {
		nf := namefilter.New(nameCfg)
		srv.SetNameFilter(nf)
		if nameCfg.Enabled {
			logger.Info("Name filter enabled", "banned_words", len(nameCfg.BannedWords), "banned_names", len(nameCfg.BannedNames))
		}
	}

	if *pilgrimMode {
		logger.Info("Server running in PILGRIM MODE - combat disabled")
	}

	// Start telnet server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Telnet server error: %v", err)
		}
	}()

	// Start WebSocket server in a goroutine
	wsAddr := fmt.Sprintf(":%d", *wsPort)
	go func() {
		if err := srv.StartWebSocket(wsAddr); err != nil {
			log.Fatalf("WebSocket server error: %v", err)
		}
	}()

	logger.Info("MUD Server running", "telnet_port", *port, "websocket_port", *wsPort)
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
