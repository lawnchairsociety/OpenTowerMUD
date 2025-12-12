package server

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lawnchairsociety/opentowermud/server/internal/antispam"
	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/command"
	"github.com/lawnchairsociety/opentowermud/server/internal/config"
	"github.com/lawnchairsociety/opentowermud/server/internal/crafting"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/gametime"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/namefilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/player"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

type Server struct {
	address             string
	listener            net.Listener
	world               *world.World
	clients             map[string]*player.Player
	mu                  sync.RWMutex
	shutdown            chan struct{}
	shutdownOnce        sync.Once
	StartTime           time.Time
	gameClock           *gametime.GameClock
	respawnManager      *RespawnManager
	dynamicSpawnManager *DynamicSpawnManager
	pilgrimMode         bool
	chatFilter          *chatfilter.ChatFilter
	chatFilterConfig    *chatfilter.Config
	nameFilter          *namefilter.NameFilter
	db                  *database.Database
	itemsConfig         *items.ItemsConfig
	spellRegistry       *spells.SpellRegistry
	recipeRegistry      *crafting.RecipeRegistry
	questRegistry       *quest.QuestRegistry
	serverConfig        *config.ServerConfig
	connLimiter         *ConnLimiter
	loginRateLimiter    *LoginRateLimiter
	bossTracker         *tower.BossTracker
}

func NewServer(address string, world *world.World, pilgrimMode bool) *Server {
	return &Server{
		address:        address,
		world:          world,
		clients:        make(map[string]*player.Player),
		shutdown:       make(chan struct{}),
		StartTime:      time.Now(),
		gameClock:      gametime.NewGameClock(),
		respawnManager: NewRespawnManager(),
		pilgrimMode:    pilgrimMode,
	}
}

// SetDatabase sets the database connection for the server
func (s *Server) SetDatabase(db *database.Database) {
	s.db = db
}

// SetItemsConfig sets the items configuration for loading player inventory
func (s *Server) SetItemsConfig(config *items.ItemsConfig) {
	s.itemsConfig = config
}

// SetSpellRegistry sets the spell registry
func (s *Server) SetSpellRegistry(registry *spells.SpellRegistry) {
	s.spellRegistry = registry
}

// GetSpellRegistry returns the spell registry
func (s *Server) GetSpellRegistry() *spells.SpellRegistry {
	return s.spellRegistry
}

// SetRecipeRegistry sets the recipe registry
func (s *Server) SetRecipeRegistry(registry *crafting.RecipeRegistry) {
	s.recipeRegistry = registry
}

// GetRecipeRegistry returns the recipe registry
func (s *Server) GetRecipeRegistry() *crafting.RecipeRegistry {
	return s.recipeRegistry
}

// SetQuestRegistry sets the quest registry
func (s *Server) SetQuestRegistry(registry *quest.QuestRegistry) {
	s.questRegistry = registry
}

// GetQuestRegistry returns the quest registry
func (s *Server) GetQuestRegistry() *quest.QuestRegistry {
	return s.questRegistry
}

// SetServerConfig sets the server configuration
func (s *Server) SetServerConfig(cfg *config.ServerConfig) {
	s.serverConfig = cfg
	// Initialize connection limiter with the new config
	s.connLimiter = NewConnLimiter(cfg.Connections)
	// Initialize login rate limiter
	s.loginRateLimiter = NewLoginRateLimiter(cfg.RateLimit)
}

// InitBossTracker initializes the boss tracker using the database.
// Must be called after SetDatabase.
func (s *Server) InitBossTracker() error {
	if s.db == nil {
		return fmt.Errorf("database not set")
	}
	adapter := NewBossKillAdapter(s.db)
	tracker, err := tower.NewBossTracker(adapter)
	if err != nil {
		return fmt.Errorf("failed to initialize boss tracker: %w", err)
	}
	s.bossTracker = tracker
	return nil
}

// GetBossTracker returns the boss tracker
func (s *Server) GetBossTracker() *tower.BossTracker {
	return s.bossTracker
}

// IsUnifiedTowerUnlocked returns true if all five racial towers have been defeated.
func (s *Server) IsUnifiedTowerUnlocked() bool {
	if s.bossTracker == nil {
		return false
	}
	return s.bossTracker.IsUnifiedUnlocked()
}

// GetTowerManager returns the tower manager for multi-tower operations.
func (s *Server) GetTowerManager() interface{} {
	if s.world == nil {
		return nil
	}
	return s.world.GetTowerManager()
}

// GetServerConfig returns the server configuration
func (s *Server) GetServerConfig() *config.ServerConfig {
	if s.serverConfig == nil {
		return config.DefaultConfig()
	}
	return s.serverConfig
}

// CreateItem creates a new instance of an item by its ID
func (s *Server) CreateItem(id string) *items.Item {
	if s.itemsConfig == nil {
		return nil
	}
	// GetItemByID creates a new item instance via CreateItemFromDefinition
	item, found := s.itemsConfig.GetItemByID(id)
	if !found {
		return nil
	}
	return item
}

// SetupDynamicSpawns initializes and starts the dynamic spawn manager
func (s *Server) SetupDynamicSpawns(t *tower.Tower) {
	if t == nil {
		return
	}
	s.dynamicSpawnManager = NewDynamicSpawnManager(t, s.GetOnlinePlayerCount)
	s.dynamicSpawnManager.Start()
}

// GetDatabase returns the database connection
func (s *Server) GetDatabase() interface{} {
	return s.db
}

// GetItemByID returns an item by its ID from the items config, or nil if not found
func (s *Server) GetItemByID(id string) *items.Item {
	if s.itemsConfig == nil {
		return nil
	}
	item, found := s.itemsConfig.GetItemByID(id)
	if !found {
		return nil
	}
	return item
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	logger.Info("Server listening", "address", s.address)

	// Start the regeneration ticker
	go s.startRegenerationTicker()

	// Start the combat ticker
	go s.startCombatTicker()

	// Start the game clock ticker
	go s.startGameClockTicker()

	// Start the respawn manager
	s.respawnManager.Start(s.respawnNPC)

	// Start the idle timeout checker
	go s.startIdleTimeoutTicker()

	// Start the auto-save ticker
	go s.startAutoSaveTicker()

	for {
		select {
		case <-s.shutdown:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-s.shutdown:
					return nil
				default:
					logger.Error("Error accepting connection", "error", err)
					continue
				}
			}

			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	ip := extractIP(remoteAddr)

	// Check connection limits
	if s.connLimiter != nil && !s.connLimiter.TryAcquire(ip) {
		logger.Warning("Connection rejected - limit exceeded",
			"remote_addr", remoteAddr,
			"ip", ip)
		conn.Write([]byte("Too many connections. Please try again later.\r\n"))
		conn.Close()
		return
	}

	defer func() {
		if s.connLimiter != nil {
			s.connLimiter.Release(ip)
		}
		conn.Close()
	}()

	client := NewTelnetClient(conn)
	s.handleClient(client)
}

// handleClient is the shared client handling logic for both telnet and WebSocket.
func (s *Server) handleClient(client Client) {
	logger.Info("Client connected", "remote_addr", client.RemoteAddr())

	// Handle authentication
	authResult, err := s.handleAuth(client)
	if err != nil {
		logger.Info("Authentication failed", "remote_addr", client.RemoteAddr(), "error", err)
		return
	}

	// Check if this is a new player (never played before)
	isNewPlayer := authResult.Character.LastPlayed == nil

	// Load character data and create player
	p, err := s.loadPlayer(client, authResult)
	if err != nil {
		logger.Error("Failed to load player", "character", authResult.Character.Name, "error", err)
		client.WriteLine("Failed to load character. Please try again.\n")
		return
	}

	name := p.GetName()

	s.mu.Lock()
	s.clients[name] = p
	s.mu.Unlock()

	defer func() {
		// Handle disconnect (save, combat penalty, etc.)
		s.handleDisconnect(p)

		logger.Info("Client disconnected", "player", name)

		s.mu.Lock()
		delete(s.clients, name)
		s.mu.Unlock()
	}()

	// Send special welcome message and starting equipment for new players
	if isNewPlayer {
		s.giveStartingEquipment(p)
		s.sendNewPlayerWelcome(p)
	}

	// Check for unread mail
	s.notifyUnreadMail(p)

	// Handle player session
	p.HandleSession()
}

// StartWebSocket starts the WebSocket server on the given address.
func (s *Server) StartWebSocket(address string) error {
	http.HandleFunc("/ws", s.handleWebSocketUpgrade)

	logger.Info("WebSocket server listening", "address", address)
	return http.ListenAndServe(address, nil)
}

// handleWebSocketUpgrade upgrades an HTTP connection to WebSocket.
func (s *Server) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// Get the real client IP (supports X-Forwarded-For from reverse proxies)
	clientIP := getRealIP(r)

	// Check connection limits before upgrading
	if s.connLimiter != nil && !s.connLimiter.TryAcquire(clientIP) {
		logger.Warning("WebSocket connection rejected - limit exceeded",
			"remote_addr", r.RemoteAddr,
			"client_ip", clientIP)
		http.Error(w, "Too many connections. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Create upgrader with origin check based on server config
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			cfg := s.GetServerConfig()
			allowed := cfg.WebSocket.IsOriginAllowed(origin, r.Host)
			if !allowed {
				logger.Warning("WebSocket connection rejected - origin not allowed",
					"origin", origin,
					"host", r.Host,
					"remote_addr", r.RemoteAddr)
			}
			return allowed
		},
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", "error", err)
		// Release the connection slot since upgrade failed
		if s.connLimiter != nil {
			s.connLimiter.Release(clientIP)
		}
		return
	}

	go s.handleWebSocketConnection(wsConn, clientIP)
}

// handleWebSocketConnection handles a WebSocket client connection.
func (s *Server) handleWebSocketConnection(wsConn *websocket.Conn, clientIP string) {
	defer func() {
		if s.connLimiter != nil {
			s.connLimiter.Release(clientIP)
		}
		wsConn.Close()
	}()

	client := NewWebSocketClient(wsConn)
	s.handleClient(client)
}

// getRealIP extracts the real client IP from an HTTP request.
// It checks X-Forwarded-For header first (for reverse proxy setups),
// then falls back to the direct remote address.
func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by reverse proxies like nginx)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// The first one is the original client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	// Check X-Real-IP header (alternative header used by some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to direct remote address
	return extractIP(r.RemoteAddr)
}

func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdown)
		if s.listener != nil {
			s.listener.Close()
		}

		// Stop the respawn manager
		s.respawnManager.Stop()

		// Stop the dynamic spawn manager
		if s.dynamicSpawnManager != nil {
			s.dynamicSpawnManager.Stop()
		}

		// Stop the login rate limiter
		if s.loginRateLimiter != nil {
			s.loginRateLimiter.Stop()
		}

		// Auto-save all connected players before shutdown
		s.mu.Lock()
		for _, client := range s.clients {
			if err := s.SavePlayer(client); err != nil {
				logger.Error("Failed to auto-save player on shutdown",
					"player", client.GetName(),
					"error", err)
			} else {
				logger.Info("Auto-saved player on shutdown",
					"player", client.GetName())
			}
			client.Disconnect()
		}
		s.mu.Unlock()

		logger.Info("Server shutdown complete, all players saved")
	})
}

func (s *Server) BroadcastMessage(message string, exclude *player.Player) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if exclude == nil || client != exclude {
			client.SendMessage(message)
		}
	}
}

// sendNewPlayerWelcome sends a special welcome message to new players
func (s *Server) sendNewPlayerWelcome(p *player.Player) {
	theme := tower.GetTheme(p.GetHomeTower())

	// Fallback values
	cityName := "the city"
	guideName := "the guide"
	guideKeyword := "guide"

	if theme != nil {
		cityName = theme.CityName
		if theme.GuideName != "" {
			guideName = theme.GuideName
			guideKeyword = theme.GuideKeyword
		}
	}

	welcome := fmt.Sprintf(`
==============================================================================
                      WELCOME TO OPEN TOWER MUD!
==============================================================================

You find yourself in %s, a city that exists in the shadow of an endless tower.
The air crackles with mystery and danger.

%s notices you and beckons warmly.

  "Ah, a new adventurer! Over here, friend!
   TALK TO ME and I'll tell you everything you need to know about this place!"

   Type: talk %s

------------------------------------------------------------------------------
  TIP: Type 'look' to see your surroundings, 'help' for all commands
------------------------------------------------------------------------------
`, cityName, guideName, guideKeyword)
	p.SendMessage(welcome)
}

// BroadcastToAll sends a message to all connected players
func (s *Server) BroadcastToAll(message string) {
	s.BroadcastMessage(message, nil)
}

// BroadcastToAdmins sends a message to all online admin players
func (s *Server) BroadcastToAdmins(message string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.clients {
		if p.IsAdmin() {
			p.SendMessage(message)
		}
	}
}

// GetOnlinePlayers returns a list of all online player names
func (s *Server) GetOnlinePlayers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	players := make([]string, 0, len(s.clients))
	for name := range s.clients {
		players = append(players, name)
	}
	return players
}

// GetOnlinePlayerCount returns the number of online players
func (s *Server) GetOnlinePlayerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// FindPlayer finds a player by name (case-insensitive partial matching)
func (s *Server) FindPlayer(name string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Exact match first
	if p, ok := s.clients[name]; ok {
		return p
	}

	// Case-insensitive search
	lowercaseName := strings.ToLower(name)
	for playerName, p := range s.clients {
		if strings.ToLower(playerName) == lowercaseName {
			return p
		}
	}

	// Partial match (starts with)
	for playerName, p := range s.clients {
		if strings.HasPrefix(strings.ToLower(playerName), lowercaseName) {
			return p
		}
	}

	return nil
}

// GetUptime returns the server uptime as a duration
func (s *Server) GetUptime() time.Duration {
	return time.Since(s.StartTime)
}

// GetCurrentHour returns the current game hour (0-23)
func (s *Server) GetCurrentHour() int {
	return s.gameClock.GetHour()
}

// GetTimeOfDay returns the current time of day as a string
func (s *Server) GetTimeOfDay() string {
	return s.gameClock.GetTimeOfDay()
}

// IsDay returns true if it's currently daytime
func (s *Server) IsDay() bool {
	return s.gameClock.IsDay()
}

// IsNight returns true if it's currently nighttime
func (s *Server) IsNight() bool {
	return s.gameClock.IsNight()
}

// IsPilgrimMode returns true if the server is in pilgrim mode (no combat)
func (s *Server) IsPilgrimMode() bool {
	return s.pilgrimMode
}

// SetChatFilter sets the chat filter for the server
func (s *Server) SetChatFilter(cf *chatfilter.ChatFilter) {
	s.chatFilter = cf
}

// SetChatFilterConfig stores the chat filter config (for antispam settings)
func (s *Server) SetChatFilterConfig(cfg *chatfilter.Config) {
	s.chatFilterConfig = cfg
}

// GetChatFilterConfig returns the chat filter config
func (s *Server) GetChatFilterConfig() *chatfilter.Config {
	return s.chatFilterConfig
}

// GetAntispamConfig returns the antispam config from the chat filter config
func (s *Server) GetAntispamConfig() *antispam.Config {
	if s.chatFilterConfig == nil || s.chatFilterConfig.Antispam == nil {
		cfg := antispam.DefaultConfig()
		return &cfg
	}
	as := s.chatFilterConfig.Antispam
	cfg := antispam.ConfigFromYAML(as.Enabled, as.MaxMessages, as.TimeWindowSeconds, as.RepeatCooldownSeconds)
	return &cfg
}

// GetChatFilter returns the chat filter
func (s *Server) GetChatFilter() *chatfilter.ChatFilter {
	return s.chatFilter
}

// SetNameFilter sets the name filter for the server
func (s *Server) SetNameFilter(nf *namefilter.NameFilter) {
	s.nameFilter = nf
}

// GetNameFilter returns the name filter
func (s *Server) GetNameFilter() *namefilter.NameFilter {
	return s.nameFilter
}

// GetGameClock returns the game clock
func (s *Server) GetGameClock() interface{} {
	return s.gameClock
}

// BroadcastToRoom sends a message to all players in a specific room
func (s *Server) BroadcastToRoom(roomID string, message string, exclude interface{}) {
	s.BroadcastToRoomFromPlayer(roomID, message, exclude, "")
}

// BroadcastToRoomFromPlayer sends a message to all players in a specific room,
// respecting ignore lists if senderName is provided
func (s *Server) BroadcastToRoomFromPlayer(roomID string, message string, exclude interface{}, senderName string) {
	// Get the room to access its player list
	room := s.world.GetRoom(roomID)
	if room == nil {
		return
	}

	// Get player names in the room (O(1) lookup vs O(n) iteration)
	playerNames := room.GetPlayers()
	if len(playerNames) == 0 {
		return
	}

	// Type assert exclude to *player.Player if provided
	var excludeName string
	if exclude != nil {
		if excludePlayer, ok := exclude.(*player.Player); ok {
			excludeName = excludePlayer.GetName()
		}
	}

	// Look up each player by name and send message
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, playerName := range playerNames {
		// Skip excluded player
		if excludeName != "" && playerName == excludeName {
			continue
		}

		// Look up the player
		client, exists := s.clients[playerName]
		if !exists {
			continue
		}

		// Check ignore list if sender is specified
		if senderName != "" && client.IsIgnoring(senderName) {
			continue
		}

		client.SendMessage(message)
	}
}

// BroadcastToFloor sends a message to all players on a specific tower floor
func (s *Server) BroadcastToFloor(floor int, message string, exclude interface{}) {
	s.BroadcastToFloorFromPlayer(floor, message, exclude, "")
}

// BroadcastToFloorFromPlayer sends a message to all players on a specific tower floor,
// respecting ignore lists if senderName is provided
func (s *Server) BroadcastToFloorFromPlayer(floor int, message string, exclude interface{}, senderName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Type assert exclude to *player.Player if provided
	var excludePlayer *player.Player
	if exclude != nil {
		excludePlayer, _ = exclude.(*player.Player)
	}

	for _, client := range s.clients {
		// Skip excluded player
		if excludePlayer != nil && client == excludePlayer {
			continue
		}

		// Check ignore list if sender is specified
		if senderName != "" && client.IsIgnoring(senderName) {
			continue
		}

		// Check if client is on the specified floor
		currentRoomIface := client.GetCurrentRoom()
		if currentRoomIface == nil {
			continue
		}

		// Type assert to access the room's GetFloor method
		type RoomWithFloor interface {
			GetFloor() int
		}
		currentRoom, ok := currentRoomIface.(RoomWithFloor)
		if !ok {
			continue
		}

		if currentRoom.GetFloor() == floor {
			client.SendMessage(message)
		}
	}
}

// startIdleTimeoutTicker runs a background ticker that disconnects idle players
func (s *Server) startIdleTimeoutTicker() {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			s.checkIdlePlayers()
		}
	}
}

// startAutoSaveTicker runs a background ticker that periodically saves all players
func (s *Server) startAutoSaveTicker() {
	// Get interval from config (0 means disabled)
	intervalMinutes := 0
	if s.serverConfig != nil {
		intervalMinutes = s.serverConfig.Session.AutoSaveIntervalMinutes
	}
	if intervalMinutes <= 0 {
		logger.Info("Auto-save disabled")
		return
	}

	interval := time.Duration(intervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Auto-save enabled", "interval_minutes", intervalMinutes)

	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			s.autoSaveAllPlayers()
		}
	}
}

// autoSaveAllPlayers saves all connected players' progress
func (s *Server) autoSaveAllPlayers() {
	s.mu.RLock()
	players := make([]*player.Player, 0, len(s.clients))
	for _, p := range s.clients {
		players = append(players, p)
	}
	s.mu.RUnlock()

	if len(players) == 0 {
		return
	}

	savedCount := 0
	errorCount := 0

	for _, p := range players {
		if err := s.savePlayerImpl(p); err != nil {
			logger.Warning("Auto-save failed for player",
				"player", p.GetName(),
				"error", err)
			errorCount++
		} else {
			savedCount++
		}
	}

	logger.Debug("Auto-save completed",
		"saved", savedCount,
		"errors", errorCount)
}

// checkIdlePlayers disconnects players who have been idle too long
func (s *Server) checkIdlePlayers() {
	// Get timeout from config (0 means disabled)
	timeoutMinutes := 0
	if s.serverConfig != nil {
		timeoutMinutes = s.serverConfig.Session.IdleTimeoutMinutes
	}
	if timeoutMinutes <= 0 {
		return // Idle timeout disabled
	}

	timeout := time.Duration(timeoutMinutes) * time.Minute

	s.mu.RLock()
	// Build list of idle players (excluding those with stalls open and items for sale)
	var idlePlayers []*player.Player
	for _, p := range s.clients {
		// Skip players with open stalls that have items - they're intentionally AFK selling
		// Empty stalls don't count as a valid reason to stay connected
		if p.IsStallOpen() && len(p.GetStallInventory()) > 0 {
			continue
		}
		if p.IsIdle(timeout) {
			idlePlayers = append(idlePlayers, p)
		}
	}
	s.mu.RUnlock()

	// Disconnect idle players
	for _, p := range idlePlayers {
		logger.Info("Disconnecting idle player",
			"player", p.GetName(),
			"idle_minutes", timeoutMinutes)
		p.SendMessage("\n\n*** You have been disconnected due to inactivity. ***\n")
		p.Disconnect()
	}
}

// startRegenerationTicker runs a background ticker that regenerates health/mana for all players
func (s *Server) startRegenerationTicker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			// Server is shutting down, stop the ticker
			return
		case <-ticker.C:
			// Apply regeneration to all connected players
			s.mu.RLock()
			for _, client := range s.clients {
				client.Regenerate()
			}
			s.mu.RUnlock()
		}
	}
}

// startCombatTicker runs a background ticker that processes combat rounds
func (s *Server) startCombatTicker() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			// Server is shutting down, stop the ticker
			return
		case <-ticker.C:
			// Process combat for all connected players
			s.mu.RLock()
			players := make([]*player.Player, 0, len(s.clients))
			for _, client := range s.clients {
				players = append(players, client)
			}
			s.mu.RUnlock()

			// Process all player attacks first
			for _, p := range players {
				s.processPlayerAttack(p)
			}

			// Process all NPC attacks (one attack per NPC)
			s.processNPCAttacks()

			// Check for aggressive NPCs attacking players
			for _, p := range players {
				s.checkAggressiveNPCs(p)
			}
		}
	}
}

// processPlayerAttack handles a player's attack during combat
func (s *Server) processPlayerAttack(p *player.Player) {
	// Skip if player not in combat
	if !p.IsInCombat() {
		return
	}

	// Get the room
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return
	}
	room, ok := roomIface.(*world.Room)
	if !ok {
		return
	}

	// Find the NPC
	npc := room.FindNPC(p.GetCombatTarget())
	if npc == nil {
		// NPC is gone (dead or disappeared)
		logger.Debug("Combat target vanished",
			"player", p.GetName(),
			"target", p.GetCombatTarget())
		p.EndCombat()
		p.SendMessage("\nYour opponent has vanished!\n")
		return
	}

	// Roll attack (d20 + STR mod vs AC)
	attackRoll, attackBreakdown := p.RollAttack()
	npcAC := npc.GetArmorClass()

	logger.Debug("Player attack roll",
		"player", p.GetName(),
		"target", npc.GetName(),
		"roll", attackRoll,
		"breakdown", attackBreakdown,
		"target_ac", npcAC,
		"hit", attackRoll >= npcAC)

	// Determine attack verb based on weapon type
	attackVerb := "swing at"
	attackVerbThirdPerson := "swings at"
	if p.HasRangedWeapon() {
		attackVerb = "shoot at"
		attackVerbThirdPerson = "shoots at"
	}

	if attackRoll < npcAC {
		// Miss!
		p.SendMessage(fmt.Sprintf("\nYou %s %s... (%s vs AC %d) Miss!\n",
			attackVerb, npc.GetName(), attackBreakdown, npcAC))

		// Notify other fighters
		targets := npc.GetTargets()
		for _, targetName := range targets {
			if targetName != p.GetName() {
				if targetPlayerInterface := s.FindPlayer(targetName); targetPlayerInterface != nil {
					if targetPlayer, ok := targetPlayerInterface.(*player.Player); ok {
						targetPlayer.SendMessage(fmt.Sprintf("\n%s %s %s and misses!\n",
							p.GetName(), attackVerbThirdPerson, npc.GetName()))
					}
				}
			}
		}
		return
	}

	// Hit! Roll damage with class bonuses
	// Check if this is the player's first successful hit (for rogue sneak attack)
	// Sneak attack applies when the target hasn't been hit by this player yet
	isSneakAttack := npc.GetThreat(p.GetName()) == 0

	playerDamage := p.GetAttackDamageAgainst(npc, isSneakAttack)
	npcDamageTaken := npc.TakeDamage(playerDamage)

	// Record damage dealt in player statistics
	p.RecordDamageDealt(npcDamageTaken)

	// Add threat based on damage dealt
	npc.AddThreat(p.GetName(), playerDamage)

	logger.Debug("Player damage dealt",
		"player", p.GetName(),
		"target", npc.GetName(),
		"damage_dealt", npcDamageTaken,
		"sneak_attack", isSneakAttack,
		"target_hp", npc.GetHealth(),
		"target_max_hp", npc.GetMaxHealth())

	// Send message to attacker with dice details
	p.SendMessage(fmt.Sprintf("\nYou %s %s... (%s vs AC %d) Hit!\nYou deal %d damage! (%d/%d HP)\n",
		attackVerb, npc.GetName(), attackBreakdown, npcAC, npcDamageTaken, npc.GetHealth(), npc.GetMaxHealth()))

	// Send message to all other players fighting this NPC
	targets := npc.GetTargets()
	for _, targetName := range targets {
		if targetName != p.GetName() {
			if targetPlayerInterface := s.FindPlayer(targetName); targetPlayerInterface != nil {
				if targetPlayer, ok := targetPlayerInterface.(*player.Player); ok {
					targetPlayer.SendMessage(fmt.Sprintf("\n%s hits %s for %d damage! (%d/%d HP)\n",
						p.GetName(), npc.GetName(), npcDamageTaken, npc.GetHealth(), npc.GetMaxHealth()))
				}
			}
		}
	}

	// Check if NPC died
	if !npc.IsAlive() {
		s.handleNPCDeath(npc, room)
		return
	}
}

// processNPCAttacks handles all NPC attacks (one attack per NPC)
func (s *Server) processNPCAttacks() {
	// Get all rooms with NPCs in combat
	rooms := s.world.GetAllRooms()

	for _, room := range rooms {
		npcs := room.GetNPCs()
		for _, npc := range npcs {
			// Skip if NPC not in combat
			if !npc.IsInCombat() {
				continue
			}

			// Skip if NPC is stunned
			if npc.IsStunned() {
				continue
			}

			// Check if NPC should flee
			if npc.ShouldFlee() {
				s.handleNPCFlee(npc, room)
				continue
			}

			// Pick the highest threat target (falls back to random if no threat data)
			targetName := npc.GetHighestThreatTarget()
			if targetName == "" {
				// No valid targets
				npc.EndCombat("")
				continue
			}

			// Find the target player
			targetPlayerInterface := s.FindPlayer(targetName)
			if targetPlayerInterface == nil {
				// Target is gone, remove from targets
				npc.EndCombat(targetName)
				continue
			}
			targetPlayer, ok := targetPlayerInterface.(*player.Player)
			if !ok {
				npc.EndCombat(targetName)
				continue
			}
			if !targetPlayer.IsAlive() {
				// Target is dead, remove from targets
				npc.EndCombat(targetName)
				continue
			}

			// Check if target is still in the same room as the NPC
			targetRoomIface := targetPlayer.GetCurrentRoom()
			if targetRoomIface == nil {
				npc.EndCombat(targetName)
				targetPlayer.EndCombat()
				continue
			}
			targetRoom, ok := targetRoomIface.(*world.Room)
			if !ok || targetRoom.GetID() != room.GetID() {
				// Target has left the room, remove from combat
				logger.Debug("Combat target left room",
					"npc", npc.GetName(),
					"target", targetName,
					"npc_room", room.GetID(),
					"target_room", targetRoom.GetID())
				npc.EndCombat(targetName)
				targetPlayer.EndCombat()
				continue
			}

			// NPC attacks the random target
			npcDamage := npc.GetAttackDamage()
			playerDamageTaken := targetPlayer.TakeDamage(npcDamage)

			// Record damage taken in player statistics
			targetPlayer.RecordDamageTaken(playerDamageTaken)

			logger.Debug("NPC attack",
				"npc", npc.GetName(),
				"target", targetName,
				"damage_dealt", playerDamageTaken,
				"target_hp", targetPlayer.GetHealth(),
				"target_max_hp", targetPlayer.GetMaxHealth())

			// Send message to all players fighting this NPC
			targets := npc.GetTargets()
			for _, fighterName := range targets {
				if fighterInterface := s.FindPlayer(fighterName); fighterInterface != nil {
					if fighter, ok := fighterInterface.(*player.Player); ok {
						if fighterName == targetName {
							// Message for the target
							fighter.SendMessage(fmt.Sprintf("%s hits you for %d damage! (%d/%d HP)\n",
								npc.GetName(), playerDamageTaken, targetPlayer.GetHealth(), targetPlayer.GetMaxHealth()))
						} else {
							// Message for other fighters
							fighter.SendMessage(fmt.Sprintf("%s hits %s for %d damage!\n",
								npc.GetName(), targetName, playerDamageTaken))
						}
					}
				}
			}

			// Check if player died
			if !targetPlayer.IsAlive() {
				s.handlePlayerDeath(targetPlayer, npc, room)
			}
		}
	}
}

// handleNPCDeath handles what happens when an NPC is defeated
func (s *Server) handleNPCDeath(npc *npc.NPC, room *world.Room) {
	// Get all players who were fighting this NPC
	attackers := npc.GetTargets()

	// Calculate split XP (NOTE: May revisit XP distribution in future)
	totalXP := npc.GetExperience()

	// Log NPC death
	logger.Info("NPC defeated",
		"npc", npc.GetName(),
		"room", room.GetID(),
		"attackers", strings.Join(attackers, ", "),
		"attacker_count", len(attackers),
		"xp_awarded", totalXP)
	xpPerPlayer := totalXP
	if len(attackers) > 1 {
		xpPerPlayer = totalXP / len(attackers)
	}

	// Build attacker names list for broadcast
	attackerNames := make([]string, 0, len(attackers))

	// Award XP and send messages to all attackers
	for _, attackerName := range attackers {
		if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
			attacker, ok := attackerInterface.(*player.Player)
			if !ok {
				continue
			}

			// End combat
			attacker.EndCombat()

			// Award experience and check for level-ups
			levelUps := attacker.GainExperience(xpPerPlayer)

			// Send victory messages
			if len(attackers) == 1 {
				attacker.SendMessage(fmt.Sprintf("\nYou have slain %s!\n", npc.GetName()))
				attacker.SendMessage(fmt.Sprintf("You gain %d experience points.\n", xpPerPlayer))
			} else {
				attacker.SendMessage(fmt.Sprintf("\nYour group has slain %s!\n", npc.GetName()))
				attacker.SendMessage(fmt.Sprintf("You gain %d experience points (split %d ways).\n", xpPerPlayer, len(attackers)))
			}

			// Send level-up notifications
			for _, lu := range levelUps {
				attacker.SendMessage(fmt.Sprintf("\n*** LEVEL UP! ***\n"))
				attacker.SendMessage(fmt.Sprintf("You are now level %d!\n", lu.NewLevel))
				attacker.SendMessage(fmt.Sprintf("Max Health increased by %d (now %d)\n", lu.HPGain, attacker.GetMaxHealth()))
				attacker.SendMessage(fmt.Sprintf("Max Mana increased by %d (now %d)\n", lu.ManaGain, attacker.GetMaxMana()))
				attacker.SendMessage("You feel completely refreshed!\n")
			}

			attackerNames = append(attackerNames, attackerName)

			// Record kill in player statistics
			mobID := strings.ToLower(strings.ReplaceAll(npc.GetName(), " ", "_"))
			attacker.RecordKill(mobID)

			// Update quest kill progress for this attacker
			if s.questRegistry != nil {
				questLog := attacker.GetQuestLog()
				if questLog != nil {
					// Check all active quests for kill objectives matching this mob
					for _, questID := range questLog.GetActiveQuests() {
						questDef, exists := s.questRegistry.GetQuest(questID)
						if exists {
							if questLog.UpdateKillProgressForQuest(questID, questDef, mobID) {
								// Notify player of quest progress
								progress, _ := questLog.GetQuestProgress(questID)
								for i, obj := range questDef.Objectives {
									if obj.Type == quest.QuestTypeKill && strings.ToLower(obj.Target) == mobID {
										current := progress.Objectives[i].Current
										targetName := obj.TargetName
										if targetName == "" {
											targetName = obj.Target
										}
										attacker.SendMessage(fmt.Sprintf("Quest progress: %s - %d/%d\n", targetName, current, obj.Required))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Roll for gold drop and award to attackers
	goldDrop := npc.RollGold()
	if goldDrop > 0 {
		goldPerPlayer := goldDrop
		if len(attackers) > 1 {
			goldPerPlayer = goldDrop / len(attackers)
		}
		for _, attackerName := range attackers {
			if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
				if attacker, ok := attackerInterface.(*player.Player); ok {
					attacker.AddGold(goldPerPlayer)
					if len(attackers) == 1 {
						attacker.SendMessage(fmt.Sprintf("You loot %d gold.\n", goldPerPlayer))
					} else {
						attacker.SendMessage(fmt.Sprintf("You loot %d gold (split %d ways).\n", goldPerPlayer, len(attackers)))
					}
				}
			}
		}
	}

	// Roll for loot drops and add items to the room
	droppedLoot := npc.RollLoot()
	if len(droppedLoot) > 0 && s.itemsConfig != nil {
		var droppedItemNames []string
		for _, itemID := range droppedLoot {
			if item, exists := s.itemsConfig.GetItemByID(itemID); exists {
				room.AddItem(item)
				droppedItemNames = append(droppedItemNames, item.Name)
			} else {
				logger.Warning("Unknown item in loot drop", "item_id", itemID, "npc", npc.GetName())
			}
		}
		// Notify all attackers about dropped loot
		if len(droppedItemNames) > 0 {
			lootMsg := fmt.Sprintf("%s dropped: %s\n", npc.GetName(), strings.Join(droppedItemNames, ", "))
			for _, attackerName := range attackers {
				if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
					if attacker, ok := attackerInterface.(*player.Player); ok {
						attacker.SendMessage(lootMsg)
					}
				}
			}
		}
	}

	// If this was a boss, drop the boss key
	if npc.GetIsBoss() {
		floorNum := npc.GetFloor()
		keyID := tower.GetBossKeyID(floorNum)
		bossKey := items.NewBossKey(keyID, floorNum)
		room.AddItem(bossKey)

		// Notify all attackers about the key drop
		for _, attackerName := range attackers {
			if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
				if attacker, ok := attackerInterface.(*player.Player); ok {
					attacker.SendMessage(fmt.Sprintf("\n*** %s dropped a %s! ***\n", npc.GetName(), bossKey.Name))
				}
			}
		}

		logger.Info("Boss key dropped",
			"npc", npc.GetName(),
			"floor", floorNum,
			"key_id", keyID,
			"room", room.GetID())

		// Check if this is a tower final boss (final floor boss)
		if s.bossTracker != nil && len(attackerNames) > 0 {
			_, towerID := s.world.FindRoomWithTowerID(room.GetID())
			if towerID != "" {
				maxFloors := s.world.GetMaxFloorsForTower(towerID)
				if floorNum == maxFloors {
					// This is the tower's final boss!
					s.handleTowerBossDefeat(tower.TowerID(towerID), attackerNames)
				}
			}
		}
	}

	// End combat for NPC
	npc.EndCombat("")

	// Broadcast to room
	if len(attackerNames) == 1 {
		s.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s has slain %s!", attackerNames[0], npc.GetName()), nil)
	} else if len(attackerNames) > 1 {
		s.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s have slain %s!", strings.Join(attackerNames, ", "), npc.GetName()), nil)
	}

	// Remove NPC from room
	room.RemoveNPC(npc)

	// Schedule respawn (if enabled)
	s.respawnManager.AddDeadNPC(npc)
}

// handleTowerBossDefeat handles when a tower's final boss is defeated
func (s *Server) handleTowerBossDefeat(towerID tower.TowerID, attackerNames []string) {
	if len(attackerNames) == 0 {
		return
	}

	theme := tower.GetTheme(towerID)
	if theme == nil {
		return
	}

	// Record the kill for the first attacker (party leader)
	primaryAttacker := attackerNames[0]
	isFirstKill, err := s.bossTracker.RecordKill(towerID, primaryAttacker)
	if err != nil {
		logger.Error("Failed to record boss kill", "tower", towerID, "player", primaryAttacker, "error", err)
		return
	}

	// Get the appropriate title
	var title string
	if isFirstKill {
		title = tower.GetFirstClearTitle(towerID)
	} else {
		title = tower.GetSharedClearTitle(towerID)
	}

	// Award titles to all attackers
	for _, attackerName := range attackerNames {
		if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
			if attacker, ok := attackerInterface.(*player.Player); ok {
				// Check if they already have this title
				if !attacker.HasEarnedTitle(title) {
					attacker.EarnTitle(title)
					attacker.SendMessage(fmt.Sprintf("\n*** You have earned the title: %s ***\n", title))
				}
			}
		}
	}

	// Special handling for The Architect (unified tower boss)
	if towerID == tower.TowerUnified {
		s.handleArchitectVictory(primaryAttacker, attackerNames, isFirstKill, title)
		return
	}

	// Broadcast announcement for racial tower bosses
	if isFirstKill {
		announcement := fmt.Sprintf(
			"\n================================================================================\n"+
				"                    %s HAS BEEN CONQUERED!\n\n"+
				"  %s has become the FIRST to defeat the guardian of %s!\n"+
				"  They have earned the unique title: %s\n"+
				"================================================================================\n",
			strings.ToUpper(theme.Name), primaryAttacker, theme.Name, title)
		s.BroadcastToAll(announcement)
	} else {
		s.BroadcastToAll(fmt.Sprintf(
			"\n*** %s has defeated the guardian of %s! ***\n",
			primaryAttacker, theme.Name))
	}

	logger.Info("Tower boss defeated",
		"tower", towerID,
		"tower_name", theme.Name,
		"player", primaryAttacker,
		"first_kill", isFirstKill,
		"title_awarded", title)

	// Check if this unlocks the unified tower
	if isFirstKill && s.bossTracker.IsUnifiedUnlocked() {
		s.handleUnifiedTowerUnlock()
	}
}

// handleArchitectVictory handles the epic event when The Architect is defeated
func (s *Server) handleArchitectVictory(primaryAttacker string, attackerNames []string, isFirstKill bool, title string) {
	if isFirstKill {
		// First ever defeat of The Architect - the ultimate achievement
		announcement := `
================================================================================
================================================================================

     █████╗ ██████╗  ██████╗██╗  ██╗██╗████████╗███████╗ ██████╗████████╗
    ██╔══██╗██╔══██╗██╔════╝██║  ██║██║╚══██╔══╝██╔════╝██╔════╝╚══██╔══╝
    ███████║██████╔╝██║     ███████║██║   ██║   █████╗  ██║        ██║
    ██╔══██║██╔══██╗██║     ██╔══██║██║   ██║   ██╔══╝  ██║        ██║
    ██║  ██║██║  ██║╚██████╗██║  ██║██║   ██║   ███████╗╚██████╗   ██║
    ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝   ╚═╝   ╚══════╝ ╚═════╝   ╚═╝

                            HAS BEEN OVERCOME!

================================================================================

    The Architect has acknowledged defeat for the first time in eternity!

    ` + primaryAttacker + ` has become the SAVIOR OF THE REALM!

    The Infinity Spire's trials are complete. The mysterious creator has
    found what it sought. Songs will be sung of this day for generations.

    Heroes who participated in this legendary victory:`

		for _, name := range attackerNames {
			announcement += "\n      - " + name
		}

		announcement += `

    The title "Savior of the Realm" is now yours to bear with pride.

================================================================================
================================================================================
`
		s.BroadcastToAll(announcement)

		// Send special message to all participants
		for _, attackerName := range attackerNames {
			if attackerInterface := s.FindPlayer(attackerName); attackerInterface != nil {
				if attacker, ok := attackerInterface.(*player.Player); ok {
					attacker.SendMessage(`
================================================================================
                        CONGRATULATIONS, HERO!

  You have accomplished the ultimate goal. The Architect's trials are complete.

  Your name will be etched in the annals of history as one of the heroes
  who conquered the Infinity Spire and proved worthy of the Architect's design.

  The tower acknowledges you. The realm is saved. You are a legend.

================================================================================
`)
				}
			}
		}
	} else {
		// Subsequent defeats
		announcement := fmt.Sprintf(`
================================================================================
                    THE ARCHITECT HAS FALLEN AGAIN!

  %s has once again overcome The Architect!

  The Infinity Spire's master acknowledges another worthy champion.

  They have earned the title: %s
================================================================================
`, primaryAttacker, title)
		s.BroadcastToAll(announcement)
	}

	logger.Info("THE ARCHITECT DEFEATED",
		"event", "architect_victory",
		"player", primaryAttacker,
		"first_kill", isFirstKill,
		"attackers", strings.Join(attackerNames, ", "))
}

// handleUnifiedTowerUnlock handles the epic event when the Infinity Spire is unlocked
func (s *Server) handleUnifiedTowerUnlock() {
	announcement := `
================================================================================
                    THE INFINITY SPIRE HAS AWAKENED

  The guardians of all five towers have fallen. The seals are broken.

  A new portal has appeared in every city, leading to the Infinity Spire -
  where The Architect awaits at the apex, testing all who dare ascend.

  Only the bravest heroes dare enter. Only the strongest will survive.
================================================================================
`
	s.BroadcastToAll(announcement)

	logger.Info("UNIFIED TOWER UNLOCKED", "event", "unified_unlock")
}

// respawnNPC handles respawning an NPC at its original location
func (s *Server) respawnNPC(npc *npc.NPC) {
	// Reset NPC to full health and clear combat state
	npc.Reset()

	// Get the original room
	originalRoomID := npc.GetOriginalRoomID()
	originalRoom := s.world.GetRoom(originalRoomID)

	if originalRoom == nil {
		logger.Warning("Cannot respawn NPC - room not found",
			"npc", npc.GetName(),
			"room_id", originalRoomID)
		return
	}

	// Update NPC's current room ID
	npc.SetRoomID(originalRoomID)

	// Add NPC back to the room
	originalRoom.AddNPC(npc)

	// Log successful respawn
	logger.Info("NPC respawned",
		"npc", npc.GetName(),
		"room", originalRoomID)

	// Broadcast respawn message to room (if any players are there)
	s.BroadcastToRoom(originalRoomID, fmt.Sprintf("%s appears in the area.", npc.GetName()), nil)
}

// handlePlayerDeath handles what happens when a player dies
func (s *Server) handlePlayerDeath(p *player.Player, npc *npc.NPC, room *world.Room) {
	// Log player death
	logger.Info("Player died",
		"player", p.GetName(),
		"killed_by", npc.GetName(),
		"room", room.GetID())

	// Record death in player statistics
	p.RecordDeath()

	// End combat for player and remove from NPC's target list
	p.EndCombat()
	npc.EndCombat(p.GetName())

	// Respawn at starting room (town square)
	respawnRoom := s.world.GetStartingRoom()

	// Send death message
	// Note: No gold/XP penalty - respawning at town is the only penalty
	p.SendMessage("\n\n*** YOU HAVE DIED ***\n")
	p.SendMessage(fmt.Sprintf("You will respawn at %s.\n\n", respawnRoom.Name))

	// Broadcast to room
	s.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s has been slain by %s!", p.GetName(), npc.GetName()), p)

	// Respawn player
	p.Health = p.GetMaxHealth()
	p.Mana = p.GetMaxMana()

	// Move to respawn room
	p.MoveTo(respawnRoom)

	p.SendMessage(respawnRoom.GetDescriptionForPlayer(p.GetName()) + "\n")
}

// handleNPCFlee handles an NPC fleeing from combat
func (s *Server) handleNPCFlee(n *npc.NPC, room *world.Room) {
	// Get available exits (excluding up/down to avoid floor changes)
	exits := room.GetExits()
	availableDirections := make([]string, 0, len(exits))
	for direction := range exits {
		// Skip vertical exits - mobs shouldn't flee between floors
		if direction != "up" && direction != "down" {
			availableDirections = append(availableDirections, direction)
		}
	}

	// If no exits available, NPC is cornered and can't flee
	if len(availableDirections) == 0 {
		return
	}

	// Pick a random direction
	fleeDirection := availableDirections[rand.Intn(len(availableDirections))]
	destRoomIface := room.GetExit(fleeDirection)
	if destRoomIface == nil {
		return
	}
	destRoom, ok := destRoomIface.(*world.Room)
	if !ok {
		return
	}

	// Get all players fighting this NPC before ending combat
	targets := n.GetTargets()

	// Notify all players in the room that the mob fled
	fleeMessage := fmt.Sprintf("\n%s panics and flees %s!\n", n.GetName(), fleeDirection)
	for _, targetName := range targets {
		if targetPlayerInterface := s.FindPlayer(targetName); targetPlayerInterface != nil {
			if targetPlayer, ok := targetPlayerInterface.(*player.Player); ok {
				targetPlayer.SendMessage(fleeMessage)
				targetPlayer.EndCombat()
			}
		}
	}

	// Also broadcast to any non-combatant players in the room
	s.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s panics and flees %s!", n.GetName(), fleeDirection), nil)

	// End combat for the NPC
	n.EndCombat("")

	// Move the NPC to the new room
	room.RemoveNPC(n)
	destRoom.AddNPC(n)
	n.SetRoomID(destRoom.GetID())

	logger.Debug("NPC fled",
		"npc", n.GetName(),
		"from_room", room.GetID(),
		"to_room", destRoom.GetID(),
		"direction", fleeDirection,
		"hp_percent", float64(n.GetHealth())/float64(n.GetMaxHealth()))
}

// checkAggressiveNPCs checks if any aggressive NPCs should attack a player
func (s *Server) checkAggressiveNPCs(p *player.Player) {
	// Skip if server is in pilgrim mode
	if s.pilgrimMode {
		return
	}

	// Skip if player is already in combat
	if p.IsInCombat() {
		return
	}

	// Skip if player is dead
	if !p.IsAlive() {
		return
	}

	// Get the room
	roomIface := p.GetCurrentRoom()
	if roomIface == nil {
		return
	}
	room, ok := roomIface.(*world.Room)
	if !ok {
		return
	}

	// Check all NPCs in the room
	npcs := room.GetNPCs()
	for _, n := range npcs {
		// Skip if NPC is not aggressive
		if !n.IsAggressive() {
			continue
		}

		// Skip if NPC is already in combat
		if n.IsInCombat() {
			continue
		}

		// Skip if NPC is dead
		if !n.IsAlive() {
			continue
		}

		// Aggressive NPC attacks player!
		p.StartCombat(n.GetName())
		n.StartCombat(p.GetName())

		// Send messages
		p.SendMessage(fmt.Sprintf("\n%s attacks you!\n", n.GetName()))
		s.BroadcastToRoom(room.GetID(), fmt.Sprintf("%s attacks %s!", n.GetName(), p.GetName()), p)

		// Only allow one NPC to attack per tick
		break
	}
}

// startGameClockTicker advances game time every 2.5 real minutes (150 seconds)
func (s *Server) startGameClockTicker() {
	ticker := time.NewTicker(150 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			oldHour := s.gameClock.GetHour()
			s.gameClock.AdvanceHour()
			newHour := s.gameClock.GetHour()

			// Broadcast dawn transition (5->6)
			if oldHour == 5 && newHour == 6 {
				s.BroadcastToAll("\n=== The sun rises over the horizon. Day has begun. ===\n")
			}

			// Broadcast dusk transition (17->18)
			if oldHour == 17 && newHour == 18 {
				s.BroadcastToAll("\n=== The sun sets, and darkness falls. Night has begun. ===\n")
			}
		}
	}
}


// GetWorld returns the game world
func (s *Server) GetWorld() interface{} {
	return s.world
}

// GetWorldRoomCount returns the number of rooms in the world
func (s *Server) GetWorldRoomCount() int {
	return s.world.GetRoomCount()
}

// GetOnlinePlayersDetailed returns detailed information about all online players
func (s *Server) GetOnlinePlayersDetailed() []command.PlayerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	players := make([]command.PlayerInfo, 0, len(s.clients))
	for _, p := range s.clients {
		players = append(players, command.PlayerInfo{
			Name:    p.GetName(),
			Level:   p.GetLevel(),
			RoomID:  p.GetRoomID(),
			IP:      p.GetRemoteAddr(),
			IsAdmin: p.IsAdmin(),
		})
	}
	return players
}

// KickPlayer disconnects a player by name with an optional reason message
func (s *Server) KickPlayer(playerName string, reason string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the player
	var targetPlayer *player.Player
	for name, p := range s.clients {
		if strings.EqualFold(name, playerName) {
			targetPlayer = p
			break
		}
	}

	if targetPlayer == nil {
		return false
	}

	// Send kick message
	kickMsg := "\n*** You have been kicked from the server"
	if reason != "" {
		kickMsg += ": " + reason
	}
	kickMsg += " ***\n"
	targetPlayer.SendMessage(kickMsg)

	// Disconnect the player
	targetPlayer.Disconnect()

	return true
}

// GenerateNextFloor generates the next floor of the tower and returns the stairs room
// This is called when a player tries to climb stairs to a floor that doesn't exist yet
func (s *Server) GenerateNextFloor(currentFloor int) (interface{}, error) {
	nextFloor := currentFloor + 1

	// Get or generate the portal room for the next floor
	nextRoom, err := s.world.GetOrGenerateFloorPortalRoom(nextFloor)
	if err != nil {
		return nil, fmt.Errorf("failed to generate floor %d: %w", nextFloor, err)
	}
	if nextRoom == nil {
		return nil, fmt.Errorf("floor %d has no portal room", nextFloor)
	}

	// Connect the stairs between the current and next floor
	// On floor 0 (city), the stairs room is separate from the portal room
	// On generated floors, portal and stairs are the same room
	currentRoom := s.world.GetFloorStairsRoom(currentFloor)
	if currentRoom != nil {
		// Connect bidirectionally
		currentRoom.AddExit("up", nextRoom)
		nextRoom.AddExit("down", currentRoom)
	}

	logger.Info("Generated tower floor",
		"floor", nextFloor,
		"connected_from", currentFloor)

	return nextRoom, nil
}
