package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/database"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/logger"
	"github.com/lawnchairsociety/opentowermud/server/internal/player"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// loadPlayer creates a Player from database character data
func (s *Server) loadPlayer(conn net.Conn, auth *AuthResult) (*player.Player, error) {
	char := auth.Character

	// Create the player with basic data
	p := player.NewPlayer(char.Name, conn, s.world, s)

	// Set persistence IDs
	p.SetAccountID(auth.Account.ID)
	p.SetCharacterID(char.ID)

	// Set admin status from account
	p.SetAdmin(auth.Account.IsAdmin)

	// Load character stats
	p.Health = char.Health
	p.MaxHealth = char.MaxHealth
	p.Mana = char.Mana
	p.MaxMana = char.MaxMana
	p.Level = char.Level
	p.Experience = char.Experience
	p.MaxCarryWeight = char.MaxCarryWeight

	// Load ability scores
	p.Strength = char.Strength
	p.Dexterity = char.Dexterity
	p.Constitution = char.Constitution
	p.Intelligence = char.Intelligence
	p.Wisdom = char.Wisdom
	p.Charisma = char.Charisma

	// Set state
	if err := p.SetState(char.State); err != nil {
		p.SetState("standing") // Default to standing if invalid
	}

	// Load room - find the room in the world
	room := s.world.GetRoom(char.RoomID)
	if room == nil {
		// Room doesn't exist (maybe world changed), use starting room
		room = s.world.GetStartingRoom()
	}
	if room == nil {
		return nil, fmt.Errorf("no valid room found for player %s (tried %s and starting room)", char.Name, char.RoomID)
	}

	// Remove from NewPlayer's starting room assignment and move to correct room
	startRoom := s.world.GetStartingRoom()
	if startRoom != nil {
		startRoom.RemovePlayer(char.Name)
	}
	p.CurrentRoom = room
	room.AddPlayer(char.Name)

	// Load inventory
	inventoryIDs, err := s.db.LoadInventory(char.ID)
	if err != nil {
		logger.Warning("Failed to load inventory", "character", char.Name, "error", err)
	} else {
		p.Inventory = make([]*items.Item, 0, len(inventoryIDs))
		for _, itemID := range inventoryIDs {
			if s.itemsConfig != nil {
				if item, exists := s.itemsConfig.GetItemByID(itemID); exists {
					p.Inventory = append(p.Inventory, item)
				} else {
					logger.Warning("Unknown item in inventory", "character", char.Name, "item_id", itemID)
				}
			}
		}
	}

	// Load equipment
	equipmentIDs, err := s.db.LoadEquipment(char.ID)
	if err != nil {
		logger.Warning("Failed to load equipment", "character", char.Name, "error", err)
	} else {
		p.Equipment = make(map[items.EquipmentSlot]*items.Item)
		for slotStr, itemID := range equipmentIDs {
			if s.itemsConfig != nil {
				if item, exists := s.itemsConfig.GetItemByID(itemID); exists {
					slot := items.StringToEquipmentSlot(slotStr)
					p.Equipment[slot] = item
				} else {
					logger.Warning("Unknown item in equipment", "character", char.Name, "item_id", itemID)
				}
			}
		}
	}

	// Load learned spells
	if char.LearnedSpells != "" {
		spellIDs := strings.Split(char.LearnedSpells, ",")
		p.SetLearnedSpells(spellIDs)
	} else {
		// Fallback for characters created before magic system
		p.SetLearnedSpells(strings.Split(database.DefaultStarterSpells, ","))
	}

	// Load gold
	p.SetGold(char.Gold)

	// Load key ring
	p.SetKeyRingFromString(char.KeyRing)

	// Load visited portals
	if char.VisitedPortals != "" {
		portalStrs := strings.Split(char.VisitedPortals, ",")
		portals := make([]int, 0, len(portalStrs))
		for _, s := range portalStrs {
			if floor, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				portals = append(portals, floor)
			}
		}
		p.SetVisitedPortals(portals)
	}

	logger.Info("Player loaded",
		"player", char.Name,
		"player_level", char.Level,
		"room", room.GetID(),
		"inventory_count", len(p.Inventory),
		"equipment_count", len(p.Equipment))

	return p, nil
}

// SavePlayer saves a player's current state to the database
// Accepts interface{} to satisfy command.ServerInterface
func (s *Server) SavePlayer(pIface interface{}) error {
	p, ok := pIface.(*player.Player)
	if !ok {
		return fmt.Errorf("invalid player type")
	}
	return s.savePlayerImpl(p)
}

// savePlayerImpl is the internal implementation of SavePlayer
func (s *Server) savePlayerImpl(p *player.Player) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	charID := p.GetCharacterID()
	if charID == 0 {
		return fmt.Errorf("player has no character ID")
	}

	// Build character data
	char := &database.Character{
		ID:             charID,
		AccountID:      p.GetAccountID(),
		Name:           p.GetName(),
		RoomID:         p.GetRoomID(),
		Health:         p.GetHealth(),
		MaxHealth:      p.GetMaxHealth(),
		Mana:           p.GetMana(),
		MaxMana:        p.GetMaxMana(),
		Level:          p.GetLevel(),
		Experience:     p.GetExperience(),
		State:          p.GetState(),
		MaxCarryWeight: p.MaxCarryWeight,
		LearnedSpells:  p.GetLearnedSpellsString(),
		VisitedPortals: p.GetVisitedPortalsString(),
		Strength:       p.GetStrength(),
		Dexterity:      p.GetDexterity(),
		Constitution:   p.GetConstitution(),
		Intelligence:   p.GetIntelligence(),
		Wisdom:         p.GetWisdom(),
		Charisma:       p.GetCharisma(),
		Gold:           p.GetGold(),
		KeyRing:        p.GetKeyRingString(),
	}

	// Get inventory and equipment IDs
	inventoryIDs := p.GetInventoryIDs()
	equipmentIDs := p.GetEquipmentIDs()

	// Save everything in a transaction
	if err := s.db.SaveCharacterFull(char, inventoryIDs, equipmentIDs); err != nil {
		return fmt.Errorf("failed to save character: %w", err)
	}

	logger.Debug("Player saved",
		"player", p.GetName(),
		"room", char.RoomID,
		"health", char.Health,
		"inventory_count", len(inventoryIDs),
		"equipment_count", len(equipmentIDs))

	return nil
}

// handleDisconnect handles player disconnect cleanup
// Note: Progress is NOT saved on disconnect - players must visit the bard to save!
func (s *Server) handleDisconnect(p *player.Player) {
	room := p.GetCurrentRoom().(*world.Room)

	// Check if player was in combat - end the combat state
	if p.IsInCombat() {
		npc := room.FindNPC(p.GetCombatTarget())

		// End combat with NPC
		if npc != nil {
			npc.EndCombat(p.GetName())
		}
		p.EndCombat()
	}

	// Remove from current room
	if p.CurrentRoom != nil {
		p.CurrentRoom.RemovePlayer(p.GetName())
	}

	logger.Info("Player disconnected (progress not saved - must visit bard)",
		"player", p.GetName())
}
