package player

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/antispam"
	"github.com/lawnchairsociety/opentowermud/server/internal/command"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/stats"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// ServerInterface defines methods needed from the server
type ServerInterface interface {
	GetOnlinePlayers() []string
	FindPlayer(name string) interface{} // Returns a PlayerInterface
	BroadcastToRoom(roomID string, message string, exclude interface{})
	IsPilgrimMode() bool
	GetAntispamConfig() *antispam.Config // Returns antispam config from chat filter
}

// PlayerState represents the current state of a player
type PlayerState int

const (
	StateStanding PlayerState = iota
	StateSitting
	StateResting
	StateSleeping
	StateFighting // For future combat system
)

// String returns the string representation of a PlayerState
func (s PlayerState) String() string {
	switch s {
	case StateStanding:
		return "standing"
	case StateSitting:
		return "sitting"
	case StateResting:
		return "resting"
	case StateSleeping:
		return "sleeping"
	case StateFighting:
		return "fighting"
	default:
		return "unknown"
	}
}

type Player struct {
	Name           string
	conn           net.Conn
	writer         *bufio.Writer
	CurrentRoom    *world.Room
	world          *world.World
	server         ServerInterface
	Inventory      []*items.Item
	Equipment      map[items.EquipmentSlot]*items.Item
	KeyRing        []*items.Item // Keys stored separately (don't count against weight/inventory)
	Gold           int           // Currency for shops
	MaxCarryWeight float64
	Health         int
	MaxHealth      int
	Mana           int
	MaxMana        int
	Level          int
	Experience     int
	State          PlayerState
	InCombat       bool   // Is this player currently fighting?
	CombatTarget   string // Name of NPC being fought
	disconnected   bool
	// Persistence fields
	AccountID   int64 // Database account ID
	CharacterID int64 // Database character ID
	// Admin fields
	isAdmin bool // Cached admin status (set on login)
	// Tower portal system - floors visited
	visitedPortals map[int]bool // Set of floor numbers with discovered portals
	// Magic system
	learnedSpells  map[string]bool      // spell_id -> learned
	spellCooldowns map[string]time.Time // spell_id -> cooldown expires at
	// Ability scores (Phase 25)
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
	// Anti-spam tracking
	spamTracker *antispam.Tracker
	// Ignore list - players whose messages we won't see
	ignoreList map[string]bool
}

func NewPlayer(name string, conn net.Conn, world *world.World, server ServerInterface) *Player {
	p := &Player{
		Name:           name,
		conn:           conn,
		writer:         bufio.NewWriter(conn),
		world:          world,
		server:         server,
		Inventory:      make([]*items.Item, 0),
		Equipment:      make(map[items.EquipmentSlot]*items.Item),
		KeyRing:        make([]*items.Item, 0),
		Gold:           20, // Starting gold
		MaxCarryWeight: 100.0, // Default carry capacity
		Health:         100,
		MaxHealth:      100,
		Mana:           100,
		MaxMana:        100,
		Level:          1,
		Experience:     0,
		State:          StateStanding, // Default state
		InCombat:       false,
		CombatTarget:   "",
		CurrentRoom:    world.GetStartingRoom(),
		visitedPortals: map[int]bool{0: true}, // Ground floor always available
		learnedSpells:  make(map[string]bool),
		spellCooldowns: make(map[string]time.Time),
		// Default ability scores (all 10s)
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
	}

	// Initialize anti-spam tracker with config from server
	if server != nil {
		if cfg := server.GetAntispamConfig(); cfg != nil {
			p.spamTracker = antispam.NewTracker(*cfg)
		}
	}
	if p.spamTracker == nil {
		p.spamTracker = antispam.NewTracker(antispam.DefaultConfig())
	}

	// Add player to starting room
	p.CurrentRoom.AddPlayer(name)

	return p
}

func (p *Player) HandleSession() {
	p.SendMessage(fmt.Sprintf("\nWelcome, %s!\n", p.Name))
	if p.server.IsPilgrimMode() {
		p.SendMessage("*** PILGRIM MODE - Peaceful exploration, combat disabled ***\n")
	}
	p.SendMessage(p.CurrentRoom.GetDescription())
	p.SendMessage("\nType 'help' for a list of commands.\n\n")

	scanner := bufio.NewScanner(p.conn)
	for scanner.Scan() {
		if p.disconnected {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Parse and execute command
		cmd := command.ParseCommand(input)
		result := cmd.Execute(p, p.world)
		p.SendMessage(result + "\n")

		// Show status prompt
		p.SendMessage(p.GetStatusPrompt())
	}
}

func (p *Player) SendMessage(message string) {
	if p.disconnected {
		return
	}

	p.writer.WriteString(message)
	p.writer.Flush()
}

func (p *Player) Disconnect() {
	p.disconnected = true
	// Remove player from current room
	if p.CurrentRoom != nil {
		p.CurrentRoom.RemovePlayer(p.Name)
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Player) MoveTo(roomIface interface{}) {
	room, ok := roomIface.(*world.Room)
	if !ok {
		return // Silent fail if invalid room type
	}

	// Remove from old room, add to new room
	if p.CurrentRoom != nil {
		p.CurrentRoom.RemovePlayer(p.Name)
	}
	p.CurrentRoom = room
	if p.CurrentRoom != nil {
		p.CurrentRoom.AddPlayer(p.Name)
	}
}

func (p *Player) AddItem(item *items.Item) {
	items.AddItem(&p.Inventory, item)
}

func (p *Player) RemoveItem(itemName string) (*items.Item, bool) {
	return items.RemoveItem(&p.Inventory, itemName)
}

func (p *Player) HasItem(itemName string) bool {
	return items.HasItem(p.Inventory, itemName)
}

func (p *Player) FindItem(partial string) (*items.Item, bool) {
	return items.FindItem(p.Inventory, partial)
}

// GetCurrentWeight returns the total weight of items in inventory
func (p *Player) GetCurrentWeight() float64 {
	return items.GetTotalWeight(p.Inventory)
}

// CanCarry checks if the player can carry an additional item
func (p *Player) CanCarry(item *items.Item) bool {
	return p.GetCurrentWeight()+item.Weight <= p.MaxCarryWeight
}

// GetCurrentRoom returns the player's current room
// Note: Returns as interface to satisfy command.RoomInterface
func (p *Player) GetCurrentRoom() interface{} {
	return p.CurrentRoom
}

// GetInventory returns the player's inventory
func (p *Player) GetInventory() []*items.Item {
	return p.Inventory
}

// AddKey adds a key to the player's key ring
func (p *Player) AddKey(key *items.Item) {
	items.AddItem(&p.KeyRing, key)
}

// RemoveKey removes a key from the player's key ring by name
func (p *Player) RemoveKey(keyName string) (*items.Item, bool) {
	return items.RemoveItem(&p.KeyRing, keyName)
}

// RemoveKeyByID removes a key from the player's key ring by ID
func (p *Player) RemoveKeyByID(keyID string) (*items.Item, bool) {
	for i, key := range p.KeyRing {
		if key.ID == keyID {
			removed := p.KeyRing[i]
			p.KeyRing = append(p.KeyRing[:i], p.KeyRing[i+1:]...)
			return removed, true
		}
	}
	return nil, false
}

// HasKey checks if the player has a key with the given ID
func (p *Player) HasKey(keyID string) bool {
	for _, key := range p.KeyRing {
		if key.ID == keyID {
			return true
		}
	}
	return false
}

// FindKey finds a key by partial name match
func (p *Player) FindKey(partial string) (*items.Item, bool) {
	return items.FindItem(p.KeyRing, partial)
}

// GetKeyRing returns the player's key ring
func (p *Player) GetKeyRing() []*items.Item {
	return p.KeyRing
}

// GetGold returns the player's gold amount
func (p *Player) GetGold() int {
	return p.Gold
}

// AddGold adds gold to the player's wallet
func (p *Player) AddGold(amount int) {
	p.Gold += amount
}

// SpendGold attempts to spend gold, returns true if successful
func (p *Player) SpendGold(amount int) bool {
	if p.Gold >= amount {
		p.Gold -= amount
		return true
	}
	return false
}

// SetGold sets the player's gold amount (used for persistence)
func (p *Player) SetGold(amount int) {
	p.Gold = amount
}

// GetKeyRingString returns the key ring as a comma-separated string of key IDs (for persistence)
func (p *Player) GetKeyRingString() string {
	if len(p.KeyRing) == 0 {
		return ""
	}
	ids := make([]string, len(p.KeyRing))
	for i, key := range p.KeyRing {
		ids[i] = key.ID
	}
	return strings.Join(ids, ",")
}

// SetKeyRingFromString loads key ring from a comma-separated string of key IDs (from persistence)
func (p *Player) SetKeyRingFromString(keyRingStr string) {
	p.KeyRing = make([]*items.Item, 0)
	if keyRingStr == "" {
		return
	}
	keyIDs := strings.Split(keyRingStr, ",")
	for _, keyID := range keyIDs {
		keyID = strings.TrimSpace(keyID)
		if keyID == "" {
			continue
		}
		// Create the appropriate key based on ID
		if strings.HasPrefix(keyID, "boss_key_floor_") {
			// Parse floor number from boss_key_floor_N
			var floorNum int
			if _, err := fmt.Sscanf(keyID, "boss_key_floor_%d", &floorNum); err == nil {
				p.KeyRing = append(p.KeyRing, items.NewBossKey(keyID, floorNum))
			}
		} else if keyID == "treasure_key" {
			p.KeyRing = append(p.KeyRing, items.NewTreasureKey())
		}
	}
}

// GetName returns the player's name
func (p *Player) GetName() string {
	return p.Name
}

// GetServer returns the player's server interface
func (p *Player) GetServer() interface{} {
	return p.server
}

// GetState returns the player's current state as a string
func (p *Player) GetState() string {
	return p.State.String()
}

// SetState changes the player's state
func (p *Player) SetState(state string) error {
	switch state {
	case "standing":
		p.State = StateStanding
	case "sitting":
		p.State = StateSitting
	case "resting":
		p.State = StateResting
	case "sleeping":
		p.State = StateSleeping
	case "fighting":
		p.State = StateFighting
	default:
		return fmt.Errorf("invalid state: %s", state)
	}
	return nil
}

// GetHealth returns the player's current health
func (p *Player) GetHealth() int {
	return p.Health
}

// GetMaxHealth returns the player's maximum health
func (p *Player) GetMaxHealth() int {
	return p.MaxHealth
}

// GetMana returns the player's current mana
func (p *Player) GetMana() int {
	return p.Mana
}

// GetMaxMana returns the player's maximum mana
func (p *Player) GetMaxMana() int {
	return p.MaxMana
}

// GetLevel returns the player's level
func (p *Player) GetLevel() int {
	return p.Level
}

// GetExperience returns the player's experience points
func (p *Player) GetExperience() int {
	return p.Experience
}

// GetStatusPrompt returns a status line showing HP, MP, and current room
func (p *Player) GetStatusPrompt() string {
	roomName := "Unknown"
	if p.CurrentRoom != nil {
		roomName = p.CurrentRoom.Name
	}
	return fmt.Sprintf("\n[HP: %d/%d | MP: %d/%d | %s]", p.Health, p.MaxHealth, p.Mana, p.MaxMana, roomName)
}

// Regenerate applies health and mana regeneration based on the player's state
func (p *Player) Regenerate() {
	if p.disconnected {
		return
	}

	// Calculate regen amounts based on state
	var healthRegen, manaRegen int

	switch p.State {
	case StateStanding:
		healthRegen = 1
		manaRegen = 0
	case StateSitting:
		healthRegen = 2
		manaRegen = 1
	case StateResting:
		healthRegen = 3
		manaRegen = 2
	case StateSleeping:
		healthRegen = 5
		manaRegen = 3
	case StateFighting:
		// No regeneration during combat
		healthRegen = 0
		manaRegen = 0
	default:
		healthRegen = 1
		manaRegen = 0
	}

	// Apply health regeneration (don't exceed max)
	if p.Health < p.MaxHealth {
		p.Health += healthRegen
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
	}

	// Apply mana regeneration (don't exceed max)
	if p.Mana < p.MaxMana {
		p.Mana += manaRegen
		if p.Mana > p.MaxMana {
			p.Mana = p.MaxMana
		}
	}
}

// EquipItem equips an item to the appropriate slot
// Returns error if slot is occupied, item type mismatch, or two-handed weapon conflict
func (p *Player) EquipItem(item *items.Item) error {
	if item.Slot == items.SlotNone {
		return fmt.Errorf("you can't equip that")
	}

	// Check if slot is already occupied
	if existing, occupied := p.Equipment[item.Slot]; occupied {
		return fmt.Errorf("you are already wearing %s on your %s", existing.Name, item.Slot.String())
	}

	// Special handling for two-handed weapons
	if item.TwoHanded && item.Slot == items.SlotWeapon {
		if offhand, hasOffhand := p.Equipment[items.SlotOffHand]; hasOffhand {
			return fmt.Errorf("you need both hands free to wield %s (remove %s first)", item.Name, offhand.Name)
		}
		if held, hasHeld := p.Equipment[items.SlotHeld]; hasHeld {
			return fmt.Errorf("you need both hands free to wield %s (remove %s first)", item.Name, held.Name)
		}
	}

	// Special handling for off-hand when wielding two-handed weapon
	if item.Slot == items.SlotOffHand {
		if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon && weapon.TwoHanded {
			return fmt.Errorf("you can't use your off-hand while wielding %s with both hands", weapon.Name)
		}
	}

	// Special handling for held slot when wielding two-handed weapon
	if item.Slot == items.SlotHeld {
		if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon && weapon.TwoHanded {
			return fmt.Errorf("you can't hold anything while wielding %s with both hands", weapon.Name)
		}
	}

	// Equip the item
	p.Equipment[item.Slot] = item
	return nil
}

// UnequipItem removes an item from an equipment slot
func (p *Player) UnequipItem(slot items.EquipmentSlot) (*items.Item, error) {
	item, equipped := p.Equipment[slot]
	if !equipped {
		return nil, fmt.Errorf("you don't have anything equipped in that slot")
	}

	delete(p.Equipment, slot)
	return item, nil
}

// FindEquippedItem finds an equipped item by partial name
func (p *Player) FindEquippedItem(partial string) (*items.Item, items.EquipmentSlot, bool) {
	partial = strings.ToLower(partial)
	for slot, item := range p.Equipment {
		if strings.Contains(strings.ToLower(item.Name), partial) {
			return item, slot, true
		}
	}
	return nil, items.SlotNone, false
}

// GetEquipment returns the player's equipment map
func (p *Player) GetEquipment() map[items.EquipmentSlot]*items.Item {
	return p.Equipment
}

// ConsumeItem consumes an item and applies its effects
// Returns a message describing what happened
func (p *Player) ConsumeItem(item *items.Item) string {
	if !item.Consumable {
		return fmt.Sprintf("You can't consume %s!", item.Name)
	}

	var effects []string

	// Apply healing
	if item.HealAmount > 0 {
		oldHealth := p.Health
		p.Health += item.HealAmount
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
		healed := p.Health - oldHealth
		if healed > 0 {
			effects = append(effects, fmt.Sprintf("restored %d HP", healed))
		}
	}

	// Apply mana restoration
	if item.ManaAmount > 0 {
		oldMana := p.Mana
		p.Mana += item.ManaAmount
		if p.Mana > p.MaxMana {
			p.Mana = p.MaxMana
		}
		restored := p.Mana - oldMana
		if restored > 0 {
			effects = append(effects, fmt.Sprintf("restored %d MP", restored))
		}
	}

	// Build result message
	if len(effects) == 0 {
		return fmt.Sprintf("You consume %s, but nothing happens.", item.Name)
	}

	effectsStr := strings.Join(effects, " and ")
	return fmt.Sprintf("You consume %s and %s.", item.Name, effectsStr)
}

// StartCombat sets the player into combat with an NPC
func (p *Player) StartCombat(npcName string) {
	p.InCombat = true
	p.CombatTarget = npcName
	// Automatically set state to fighting
	if p.State != StateFighting {
		p.State = StateFighting
	}
}

// EndCombat removes the player from combat
func (p *Player) EndCombat() {
	p.InCombat = false
	p.CombatTarget = ""
	// Return to standing state after combat
	if p.State == StateFighting {
		p.State = StateStanding
	}
}

// IsInCombat returns true if the player is currently fighting
func (p *Player) IsInCombat() bool {
	return p.InCombat
}

// GetCombatTarget returns the name of the NPC this player is fighting
func (p *Player) GetCombatTarget() string {
	return p.CombatTarget
}

// TakeDamage applies damage to the player and returns actual damage taken
func (p *Player) TakeDamage(damage int) int {
	// Calculate total armor from equipped items
	totalArmor := 0
	for _, item := range p.Equipment {
		if item != nil {
			totalArmor += item.Armor
		}
	}

	// Apply armor reduction
	actualDamage := damage - totalArmor
	if actualDamage < 1 {
		actualDamage = 1 // Minimum 1 damage
	}

	p.Health -= actualDamage
	if p.Health < 0 {
		p.Health = 0
	}

	return actualDamage
}

// Heal restores health to the player, capped at MaxHealth
func (p *Player) Heal(amount int) int {
	oldHealth := p.Health
	p.Health += amount
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
	return p.Health - oldHealth
}

// HealToFull restores the player to full health, returns amount healed
func (p *Player) HealToFull() int {
	return p.Heal(p.MaxHealth - p.Health)
}

// RestoreManaToFull restores the player to full mana, returns amount restored
func (p *Player) RestoreManaToFull() int {
	restored := p.MaxMana - p.Mana
	p.Mana = p.MaxMana
	return restored
}

// GetAttackDamage returns the damage this player deals in combat
// Uses dice rolling for weapons with damage_dice, falls back to static damage
func (p *Player) GetAttackDamage() int {
	strMod := p.GetStrengthMod()

	// Check for equipped weapon with dice notation
	if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon {
		if weapon.DamageDice != "" {
			// Roll weapon dice + STR modifier
			damage := stats.ParseDiceWithBonus(weapon.DamageDice, strMod)
			if damage < 1 {
				damage = 1
			}
			return damage
		}
		// Fallback to static damage + STR modifier
		damage := weapon.Damage + strMod
		if damage < 1 {
			damage = 1
		}
		return damage
	}

	// Unarmed: 1d4 + STR modifier
	damage := stats.ParseDiceWithBonus("1d4", strMod)
	if damage < 1 {
		damage = 1
	}
	return damage
}

// RollAttack rolls a d20 + STR modifier for melee attack
// Returns the roll result and the breakdown string for display
func (p *Player) RollAttack() (int, string) {
	d20 := stats.D20()
	strMod := p.GetStrengthMod()
	total := d20 + strMod

	var breakdown string
	if strMod >= 0 {
		breakdown = fmt.Sprintf("d20+%d = %d", strMod, total)
	} else {
		breakdown = fmt.Sprintf("d20%d = %d", strMod, total)
	}

	return total, breakdown
}

// IsAlive returns true if the player has health remaining
func (p *Player) IsAlive() bool {
	return p.Health > 0
}

// GainExperience adds experience points to the player and returns level-up info if leveled
func (p *Player) GainExperience(xp int) []leveling.LevelUpInfo {
	p.Experience += xp

	var levelUps []leveling.LevelUpInfo

	// Check for level up (can level multiple times from one XP gain)
	for p.Level < leveling.MaxPlayerLevel && p.Experience >= leveling.XPForLevel(p.Level+1) {
		levelUp := p.levelUp()
		levelUps = append(levelUps, levelUp)
	}

	return levelUps
}

// levelUp advances the player one level and returns the level-up info
func (p *Player) levelUp() leveling.LevelUpInfo {
	p.Level++

	// Increase stats
	p.MaxHealth += leveling.HPPerLevel
	p.MaxMana += leveling.ManaPerLevel

	// Fully restore on level up
	p.Health = p.MaxHealth
	p.Mana = p.MaxMana

	return leveling.LevelUpInfo{
		NewLevel: p.Level,
		HPGain:   leveling.HPPerLevel,
		ManaGain: leveling.ManaPerLevel,
	}
}

// GetInventoryIDs returns the item IDs in the player's inventory (for persistence)
func (p *Player) GetInventoryIDs() []string {
	ids := make([]string, len(p.Inventory))
	for i, item := range p.Inventory {
		ids[i] = item.ID
	}
	return ids
}

// GetEquipmentIDs returns a map of slot -> item ID (for persistence)
func (p *Player) GetEquipmentIDs() map[string]string {
	equipment := make(map[string]string)
	for slot, item := range p.Equipment {
		if item != nil {
			equipment[slot.String()] = item.ID
		}
	}
	return equipment
}

// GetAccountID returns the player's account ID
func (p *Player) GetAccountID() int64 {
	return p.AccountID
}

// GetCharacterID returns the player's character ID
func (p *Player) GetCharacterID() int64 {
	return p.CharacterID
}

// SetAccountID sets the player's account ID
func (p *Player) SetAccountID(id int64) {
	p.AccountID = id
}

// SetCharacterID sets the player's character ID
func (p *Player) SetCharacterID(id int64) {
	p.CharacterID = id
}

// GetRoomID returns the ID of the player's current room
func (p *Player) GetRoomID() string {
	if p.CurrentRoom != nil {
		return p.CurrentRoom.GetID()
	}
	return "town_square"
}

// IsAdmin returns whether this player has admin privileges
func (p *Player) IsAdmin() bool {
	return p.isAdmin
}

// SetAdmin sets the player's admin status (cached from database on login)
func (p *Player) SetAdmin(isAdmin bool) {
	p.isAdmin = isAdmin
}

// Ability score getters

// GetStrength returns the player's strength score
func (p *Player) GetStrength() int {
	return p.Strength
}

// GetDexterity returns the player's dexterity score
func (p *Player) GetDexterity() int {
	return p.Dexterity
}

// GetConstitution returns the player's constitution score
func (p *Player) GetConstitution() int {
	return p.Constitution
}

// GetIntelligence returns the player's intelligence score
func (p *Player) GetIntelligence() int {
	return p.Intelligence
}

// GetWisdom returns the player's wisdom score
func (p *Player) GetWisdom() int {
	return p.Wisdom
}

// GetCharisma returns the player's charisma score
func (p *Player) GetCharisma() int {
	return p.Charisma
}

// Ability score modifier getters (using stats.Modifier)

// GetStrengthMod returns the player's strength modifier
func (p *Player) GetStrengthMod() int {
	return stats.Modifier(p.Strength)
}

// GetDexterityMod returns the player's dexterity modifier
func (p *Player) GetDexterityMod() int {
	return stats.Modifier(p.Dexterity)
}

// GetConstitutionMod returns the player's constitution modifier
func (p *Player) GetConstitutionMod() int {
	return stats.Modifier(p.Constitution)
}

// GetIntelligenceMod returns the player's intelligence modifier
func (p *Player) GetIntelligenceMod() int {
	return stats.Modifier(p.Intelligence)
}

// GetWisdomMod returns the player's wisdom modifier
func (p *Player) GetWisdomMod() int {
	return stats.Modifier(p.Wisdom)
}

// GetCharismaMod returns the player's charisma modifier
func (p *Player) GetCharismaMod() int {
	return stats.Modifier(p.Charisma)
}

// Ability score setters

// SetStrength sets the player's strength score
func (p *Player) SetStrength(val int) {
	p.Strength = val
}

// SetDexterity sets the player's dexterity score
func (p *Player) SetDexterity(val int) {
	p.Dexterity = val
}

// SetConstitution sets the player's constitution score
func (p *Player) SetConstitution(val int) {
	p.Constitution = val
}

// SetIntelligence sets the player's intelligence score
func (p *Player) SetIntelligence(val int) {
	p.Intelligence = val
}

// SetWisdom sets the player's wisdom score
func (p *Player) SetWisdom(val int) {
	p.Wisdom = val
}

// SetCharisma sets the player's charisma score
func (p *Player) SetCharisma(val int) {
	p.Charisma = val
}

// GetConnection returns the player's network connection (for admin commands)
func (p *Player) GetConnection() net.Conn {
	return p.conn
}

// ==================== PORTAL METHODS ====================

// DiscoverPortal marks a floor's portal as discovered
func (p *Player) DiscoverPortal(floorNum int) {
	if p.visitedPortals == nil {
		p.visitedPortals = make(map[int]bool)
		p.visitedPortals[0] = true // Ground floor always available
	}
	p.visitedPortals[floorNum] = true
}

// HasDiscoveredPortal returns true if the player has discovered the portal on a floor
func (p *Player) HasDiscoveredPortal(floorNum int) bool {
	if p.visitedPortals == nil {
		return floorNum == 0 // Ground floor always available
	}
	return p.visitedPortals[floorNum]
}

// GetDiscoveredPortals returns a sorted list of floor numbers with discovered portals
func (p *Player) GetDiscoveredPortals() []int {
	if p.visitedPortals == nil {
		return []int{0}
	}
	floors := make([]int, 0, len(p.visitedPortals))
	for floor := range p.visitedPortals {
		floors = append(floors, floor)
	}
	// Sort floors
	for i := 0; i < len(floors)-1; i++ {
		for j := i + 1; j < len(floors); j++ {
			if floors[i] > floors[j] {
				floors[i], floors[j] = floors[j], floors[i]
			}
		}
	}
	return floors
}

// GetVisitedPortalsString returns discovered portals as a comma-separated string (for persistence)
func (p *Player) GetVisitedPortalsString() string {
	floors := p.GetDiscoveredPortals()
	strs := make([]string, len(floors))
	for i, f := range floors {
		strs[i] = fmt.Sprintf("%d", f)
	}
	return strings.Join(strs, ",")
}

// SetVisitedPortals sets the discovered portals from a list (used when loading from database)
func (p *Player) SetVisitedPortals(floors []int) {
	p.visitedPortals = make(map[int]bool)
	p.visitedPortals[0] = true // Ground floor always available
	for _, floor := range floors {
		p.visitedPortals[floor] = true
	}
}

// ==================== SPELL METHODS ====================

// HasSpell returns true if the player has learned the specified spell.
func (p *Player) HasSpell(spellID string) bool {
	if p.learnedSpells == nil {
		return false
	}
	return p.learnedSpells[spellID]
}

// LearnSpell adds a spell to the player's known spells.
func (p *Player) LearnSpell(spellID string) {
	if p.learnedSpells == nil {
		p.learnedSpells = make(map[string]bool)
	}
	p.learnedSpells[spellID] = true
}

// IsSpellOnCooldown returns true if the spell is on cooldown, and the seconds remaining.
func (p *Player) IsSpellOnCooldown(spellID string) (bool, int) {
	if p.spellCooldowns == nil {
		return false, 0
	}
	expiresAt, exists := p.spellCooldowns[spellID]
	if !exists {
		return false, 0
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		// Cooldown expired, clean up
		delete(p.spellCooldowns, spellID)
		return false, 0
	}
	return true, int(remaining.Seconds()) + 1 // Round up
}

// StartSpellCooldown puts a spell on cooldown for the specified duration.
func (p *Player) StartSpellCooldown(spellID string, seconds int) {
	if p.spellCooldowns == nil {
		p.spellCooldowns = make(map[string]time.Time)
	}
	p.spellCooldowns[spellID] = time.Now().Add(time.Duration(seconds) * time.Second)
}

// GetLearnedSpells returns a list of spell IDs the player has learned.
func (p *Player) GetLearnedSpells() []string {
	if p.learnedSpells == nil {
		return nil
	}
	spells := make([]string, 0, len(p.learnedSpells))
	for spellID := range p.learnedSpells {
		spells = append(spells, spellID)
	}
	return spells
}

// GetLearnedSpellsString returns learned spells as a comma-separated string (for persistence).
func (p *Player) GetLearnedSpellsString() string {
	spells := p.GetLearnedSpells()
	return strings.Join(spells, ",")
}

// SetLearnedSpells sets the learned spells from a list (used when loading from database).
func (p *Player) SetLearnedSpells(spellIDs []string) {
	p.learnedSpells = make(map[string]bool)
	for _, spellID := range spellIDs {
		if spellID != "" {
			p.learnedSpells[spellID] = true
		}
	}
}

// UseMana deducts mana from the player. Returns false if not enough mana.
func (p *Player) UseMana(amount int) bool {
	if p.Mana < amount {
		return false
	}
	p.Mana -= amount
	return true
}

// TakeMagicDamage applies damage to the player without armor reduction (for spell damage).
func (p *Player) TakeMagicDamage(damage int) int {
	if damage < 1 {
		damage = 1
	}
	p.Health -= damage
	if p.Health < 0 {
		p.Health = 0
	}
	return damage
}

// CheckChatSpam checks if a chat message should be allowed based on anti-spam rules.
// Returns (allowed, reason) - if not allowed, reason explains why.
func (p *Player) CheckChatSpam(message string) (bool, string) {
	if p.spamTracker == nil {
		// Initialize tracker with config from server if available
		if p.server != nil {
			if cfg := p.server.GetAntispamConfig(); cfg != nil {
				p.spamTracker = antispam.NewTracker(*cfg)
			}
		}
		if p.spamTracker == nil {
			p.spamTracker = antispam.NewTracker(antispam.DefaultConfig())
		}
	}
	result := p.spamTracker.Check(message)
	return result.Allowed, result.Reason
}

// ==================== IGNORE LIST METHODS ====================

// IsIgnoring returns true if this player is ignoring the given player name
func (p *Player) IsIgnoring(playerName string) bool {
	if p.ignoreList == nil {
		return false
	}
	return p.ignoreList[strings.ToLower(playerName)]
}

// AddIgnore adds a player to the ignore list
func (p *Player) AddIgnore(playerName string) {
	if p.ignoreList == nil {
		p.ignoreList = make(map[string]bool)
	}
	p.ignoreList[strings.ToLower(playerName)] = true
}

// RemoveIgnore removes a player from the ignore list
func (p *Player) RemoveIgnore(playerName string) {
	if p.ignoreList != nil {
		delete(p.ignoreList, strings.ToLower(playerName))
	}
}

// GetIgnoreList returns the list of ignored player names
func (p *Player) GetIgnoreList() []string {
	if p.ignoreList == nil {
		return nil
	}
	list := make([]string, 0, len(p.ignoreList))
	for name := range p.ignoreList {
		list = append(list, name)
	}
	return list
}
