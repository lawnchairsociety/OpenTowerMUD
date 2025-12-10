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
	// Parse command-line flags (only operational flags remain)
	port := flag.Int("port", 4000, "Telnet server port")
	wsPort := flag.Int("wsport", 4443, "WebSocket server port")
	serverConfigFile := flag.String("config", "data/server.yaml", "Path to server config YAML file")
	dbFile := flag.String("db", "data/opentowermud.db", "Path to player database file")
	pilgrimMode := flag.Bool("pilgrim", false, "Enable pilgrim mode (peaceful exploration, no combat)")
	readOnly := flag.Bool("readonly", false, "Run in read-only mode (world changes won't be saved to disk)")
	makeAdmin := flag.String("make-admin", "", "Promote an existing account to admin and exit (requires username)")
	flag.Parse()

	// Handle --make-admin flag (promotes account and exits)
	if *makeAdmin != "" {
		handleMakeAdmin(*makeAdmin, *dbFile)
		return
	}

	// Load server config first (paths and game settings are in here)
	serverCfg, err := config.LoadConfig(*serverConfigFile)
	if err != nil {
		log.Printf("Warning: Failed to load server config, using defaults: %v", err)
		serverCfg = config.DefaultConfig()
	}

	// Initialize logger first (before any logging)
	logConfig, _ := logger.LoadConfig(serverCfg.Paths.Logging)
	logger.Initialize(logConfig)

	logger.Info("Starting Open Tower MUD Server")

	// Use provided seed or generate from time
	worldSeed := serverCfg.Game.Seed
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

	// Load mob and item configurations (needed for tower initialization)
	mobConfig, err := npc.LoadNPCsFromDirectory(serverCfg.Paths.MobsDir)
	if err != nil {
		logger.Warning("Failed to load mobs config, mob spawning disabled", "dir", serverCfg.Paths.MobsDir, "error", err)
	}

	itemsConfig, err := items.LoadItemsFromYAML(serverCfg.Paths.Items)
	if err != nil {
		log.Fatalf("Failed to load items config: %v", err)
	}

	// Create tower manager for multi-tower support
	towerManager := tower.NewTowerManager(serverCfg.Paths.DataDir)
	towerManager.SetMobConfig(mobConfig)
	towerManager.SetItemConfig(itemsConfig)
	towerManager.SetWorldDir(serverCfg.Paths.WorldDir)

	// Initialize each enabled tower
	enabledTowers := serverCfg.Game.GetEnabledTowers()
	logger.Info("Initializing towers", "enabled", enabledTowers)

	for _, towerID := range enabledTowers {
		if err := towerManager.InitializeTower(tower.TowerID(towerID), worldSeed); err != nil {
			log.Fatalf("Failed to initialize tower %s: %v", towerID, err)
		}
		// Try to load any saved world state for this tower
		if loaded, err := towerManager.LoadTowerState(tower.TowerID(towerID)); err != nil {
			logger.Warning("Failed to load world state for tower", "tower_id", towerID, "error", err)
		} else if loaded {
			logger.Info("Tower world state loaded", "tower_id", towerID)
		}
		logger.Info("Tower initialized", "tower_id", towerID)
	}

	// Wire tower manager to world
	gameWorld.SetTowerManager(towerManager)

	// Also set the first tower as the default tower for backward compatibility
	if len(enabledTowers) > 0 {
		firstTower := towerManager.GetTower(tower.TowerID(enabledTowers[0]))
		if firstTower != nil {
			gameWorld.SetTower(firstTower)
		}
	}

	// Add all city rooms to world's room map for direct lookup
	for roomID, room := range towerManager.GetAllCityRooms() {
		gameWorld.AddRoom(room)
		_ = roomID // silence unused variable warning
	}

	// Load NPCs from directories and place them in rooms
	gameWorld.InitializeWithDirs(worldSeed, serverCfg.Paths.WorldDir, serverCfg.Paths.NPCsDir, serverCfg.Paths.MobsDir)

	// Load races config
	racesConfig, err := race.LoadRacesFromYAML(serverCfg.Paths.Races)
	if err != nil {
		logger.Warning("Failed to load races config, using defaults", "path", serverCfg.Paths.Races, "error", err)
	} else {
		logger.Info("Races loaded", "count", len(racesConfig.Races))
	}

	// Load spells config
	spellRegistry := spells.NewSpellRegistry()
	if err := spellRegistry.LoadFromYAML(serverCfg.Paths.Spells); err != nil {
		log.Fatalf("Failed to load spells config: %v", err)
	}
	logger.Info("Spells loaded", "count", len(spellRegistry.GetAllSpells()))

	// Load recipes config
	recipeRegistry := crafting.NewRecipeRegistry()
	if err := recipeRegistry.LoadFromYAML(serverCfg.Paths.Recipes); err != nil {
		logger.Warning("Failed to load recipes config, crafting disabled", "path", serverCfg.Paths.Recipes, "error", err)
	} else {
		logger.Info("Recipes loaded", "count", recipeRegistry.Count())
	}

	// Load quests config
	questRegistry := quest.NewQuestRegistry()
	if err := questRegistry.LoadFromDirectory(serverCfg.Paths.QuestsDir); err != nil {
		logger.Warning("Failed to load quests config, quests disabled", "dir", serverCfg.Paths.QuestsDir, "error", err)
	} else {
		logger.Info("Quests loaded", "count", questRegistry.Count())
	}

	// Load help system
	if err := help.Initialize(serverCfg.Paths.Help); err != nil {
		logger.Warning("Failed to load help config, help system disabled", "path", serverCfg.Paths.Help, "error", err)
	} else {
		logger.Info("Help system loaded", "path", serverCfg.Paths.Help)
	}

	// Load text system
	if err := text.Initialize(serverCfg.Paths.Text); err != nil {
		logger.Warning("Failed to load text config, using fallback text", "path", serverCfg.Paths.Text, "error", err)
	} else {
		logger.Info("Text system loaded", "path", serverCfg.Paths.Text)
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

	// Set server config on server (already loaded earlier)
	srv.SetServerConfig(serverCfg)
	if len(serverCfg.WebSocket.AllowedOrigins) == 0 {
		logger.Info("WebSocket CORS policy", "mode", "same-origin")
	} else if len(serverCfg.WebSocket.AllowedOrigins) == 1 && serverCfg.WebSocket.AllowedOrigins[0] == "*" {
		logger.Warning("WebSocket CORS allows all origins (not recommended for production)")
	} else {
		logger.Info("WebSocket CORS policy", "allowed_origins", serverCfg.WebSocket.AllowedOrigins)
	}

	// Set up dynamic spawn scaling based on player count (use first tower for now)
	if len(enabledTowers) > 0 {
		if firstTower := towerManager.GetTower(tower.TowerID(enabledTowers[0])); firstTower != nil {
			srv.SetupDynamicSpawns(firstTower)
		}
	}

	// Load and set chat filter
	filterCfg, err := chatfilter.LoadConfig(serverCfg.Paths.ChatFilter)
	if err != nil {
		logger.Warning("Failed to load chat filter config, chat filter disabled", "path", serverCfg.Paths.ChatFilter, "error", err)
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
	nameCfg, err := namefilter.LoadConfig(serverCfg.Paths.NameFilter)
	if err != nil {
		logger.Warning("Failed to load name filter config, name filter disabled", "path", serverCfg.Paths.NameFilter, "error", err)
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

	// Save world state (unless in read-only mode)
	if !*readOnly {
		if saved, err := towerManager.SaveAllTowers(); err != nil {
			logger.Error("Failed to save world state", "error", err)
		} else if saved > 0 {
			logger.Info("World state saved", "towers_saved", saved)
		}
	}

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

