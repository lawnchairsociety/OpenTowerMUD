package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/antispam"
	"github.com/lawnchairsociety/opentowermud/server/internal/chatfilter"
	"github.com/lawnchairsociety/opentowermud/server/internal/command"
	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/gametime"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/player"
	"github.com/lawnchairsociety/opentowermud/server/internal/spells"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

type Server struct {
	address        string
	listener       net.Listener
	world          *world.World
	clients        map[string]*player.Player
	mu             sync.RWMutex
	shutdown       chan struct{}
	StartTime      time.Time
	gameClock      *gametime.GameClock
	respawnManager *RespawnManager
	pilgrimMode      bool
	chatFilter       *chatfilter.ChatFilter
	chatFilterConfig *chatfilter.Config
	db               *database.Database
	itemsConfig    *items.ItemsConfig
	spellRegistry  *spells.SpellRegistry
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

	// Note: Auto-save is disabled. Players save by talking to the bard in the tavern.

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
	defer conn.Close()

	logger.Info("Client connected", "remote_addr", conn.RemoteAddr().String())

	scanner := bufio.NewScanner(conn)

	// Handle authentication
	authResult, err := s.handleAuth(conn, scanner)
	if err != nil {
		logger.Info("Authentication failed", "remote_addr", conn.RemoteAddr().String(), "error", err)
		return
	}

	// Check if this is a new player (never played before)
	isNewPlayer := authResult.Character.LastPlayed == nil

	// Load character data and create player
	p, err := s.loadPlayer(conn, authResult)
	if err != nil {
		logger.Error("Failed to load player", "character", authResult.Character.Name, "error", err)
		conn.Write([]byte("Failed to load character. Please try again.\n"))
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

	// Handle player session
	p.HandleSession()
}

func (s *Server) Shutdown() {
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}

	// Stop the respawn manager
	s.respawnManager.Stop()

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
	welcome := `
==============================================================================
                      WELCOME TO OPEN TOWER MUD!
==============================================================================

You find yourself standing in the Town Square, the heart of a walled city
that exists in the shadow of an endless tower. The air crackles with mystery
and danger.

An old man with kind eyes and a weathered cloak notices you and waves warmly.

  "Ah, a new adventurer! Over here, friend! I am Aldric, the old guide.
   TALK TO ME and I'll tell you everything you need to know about this place!"

   Type: talk aldric

------------------------------------------------------------------------------
  TIP: Type 'look' to see your surroundings, 'help' for all commands
------------------------------------------------------------------------------
`
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

		// Check if client is in the specified room
		currentRoomIface := client.GetCurrentRoom()
		if currentRoomIface == nil {
			continue
		}

		// Type assert to access the room's GetID method
		type RoomWithID interface {
			GetID() string
		}
		currentRoom, ok := currentRoomIface.(RoomWithID)
		if !ok {
			continue
		}

		if currentRoom.GetID() == roomID {
			client.SendMessage(message)
		}
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
	room := p.GetCurrentRoom().(*world.Room)

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
					targetPlayer := targetPlayerInterface.(*player.Player)
					targetPlayer.SendMessage(fmt.Sprintf("\n%s %s %s and misses!\n",
						p.GetName(), attackVerbThirdPerson, npc.GetName()))
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
				targetPlayer := targetPlayerInterface.(*player.Player)
				targetPlayer.SendMessage(fmt.Sprintf("\n%s hits %s for %d damage! (%d/%d HP)\n",
					p.GetName(), npc.GetName(), npcDamageTaken, npc.GetHealth(), npc.GetMaxHealth()))
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
			targetPlayer := targetPlayerInterface.(*player.Player)
			if !targetPlayer.IsAlive() {
				// Target is dead, remove from targets
				npc.EndCombat(targetName)
				continue
			}

			// Check if target is still in the same room as the NPC
			targetRoom := targetPlayer.GetCurrentRoom().(*world.Room)
			if targetRoom.GetID() != room.GetID() {
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
					fighter := fighterInterface.(*player.Player)
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
			attacker := attackerInterface.(*player.Player)

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
				attacker := attackerInterface.(*player.Player)
				attacker.AddGold(goldPerPlayer)
				if len(attackers) == 1 {
					attacker.SendMessage(fmt.Sprintf("You loot %d gold.\n", goldPerPlayer))
				} else {
					attacker.SendMessage(fmt.Sprintf("You loot %d gold (split %d ways).\n", goldPerPlayer, len(attackers)))
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
					attacker := attackerInterface.(*player.Player)
					attacker.SendMessage(lootMsg)
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
				attacker := attackerInterface.(*player.Player)
				attacker.SendMessage(fmt.Sprintf("\n*** %s dropped a %s! ***\n", npc.GetName(), bossKey.Name))
			}
		}

		logger.Info("Boss key dropped",
			"npc", npc.GetName(),
			"floor", floorNum,
			"key_id", keyID,
			"room", room.GetID())
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
	npc.RoomID = originalRoomID

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

	// End combat for player and remove from NPC's target list
	p.EndCombat()
	npc.EndCombat(p.GetName())

	// Respawn at starting room (town square)
	respawnRoom := s.world.GetStartingRoom()

	// Send death message
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
	room := p.GetCurrentRoom().(*world.Room)

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
		ip := ""
		if conn := p.GetConnection(); conn != nil {
			ip = conn.RemoteAddr().String()
		}
		players = append(players, command.PlayerInfo{
			Name:    p.GetName(),
			Level:   p.GetLevel(),
			RoomID:  p.GetRoomID(),
			IP:      ip,
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
