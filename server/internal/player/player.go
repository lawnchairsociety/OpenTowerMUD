package player

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/antispam"
	"github.com/lawnchairsociety/opentowermud/server/internal/class"
	"github.com/lawnchairsociety/opentowermud/server/internal/command"
	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/leveling"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/race"
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
	// Tower portal system - discovered floors
	discoveredPortals map[int]bool // Set of floor numbers with discovered portals
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
	// Class system
	classLevels *class.ClassLevels // Levels in each class
	activeClass class.Class        // Which class currently gains XP
	// Race system
	race race.Race // Player's race (e.g., "human", "dwarf")
	// Anti-spam tracking
	spamTracker *antispam.Tracker
	// Ignore list - players whose messages we won't see
	ignoreList map[string]bool
}

func NewPlayer(name string, conn net.Conn, world *world.World, server ServerInterface) *Player {
	// Default to Warrior class
	startingClass := class.Warrior
	classLevels := class.NewClassLevels(startingClass)

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
		Mana:           0,
		MaxMana:        0,
		Level:          1,
		Experience:     0,
		State:          StateStanding, // Default state
		InCombat:       false,
		CombatTarget:   "",
		CurrentRoom:    world.GetStartingRoom(),
		discoveredPortals: map[int]bool{0: true}, // Ground floor always available
		learnedSpells:  make(map[string]bool),
		spellCooldowns: make(map[string]time.Time),
		// Default ability scores (all 10s)
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
		// Class system
		classLevels: classLevels,
		activeClass: startingClass,
		// Race system
		race: race.Human, // Default to Human
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

// GetEquippedWeapon returns the player's equipped weapon, or nil if unarmed
func (p *Player) GetEquippedWeapon() *items.Item {
	return p.Equipment[items.SlotWeapon]
}

// HasRangedWeapon returns true if the player has a ranged weapon equipped
func (p *Player) HasRangedWeapon() bool {
	weapon := p.GetEquippedWeapon()
	return weapon != nil && weapon.IsRanged()
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

// isWearingHeavyArmor returns true if the player has heavy armor equipped on body slot
func (p *Player) isWearingHeavyArmor() bool {
	if bodyArmor, hasBody := p.Equipment[items.SlotBody]; hasBody {
		return bodyArmor.ArmorType == "heavy"
	}
	return false
}

// GetEffectiveArmor returns total armor including class bonuses
func (p *Player) GetEffectiveArmor() int {
	// Base armor from equipment
	totalArmor := 0
	for _, item := range p.Equipment {
		if item != nil {
			totalArmor += item.Armor
		}
	}

	// Warrior: +1 AC when wearing heavy armor (level 10+)
	if p.HasClass(class.Warrior) && p.GetClassLevel(class.Warrior) >= 10 && p.isWearingHeavyArmor() {
		totalArmor += 1
	}

	// Mage: Arcane Shield +2 AC (level 15+)
	if p.HasClass(class.Mage) && p.GetClassLevel(class.Mage) >= 15 {
		totalArmor += 2
	}

	// Cleric: Divine Protection +1 AC (level 10+)
	if p.HasClass(class.Cleric) && p.GetClassLevel(class.Cleric) >= 10 {
		totalArmor += 1
	}

	return totalArmor
}

// TakeDamage applies damage to the player and returns actual damage taken
// Includes class passive effects (evasion, sanctuary, armor bonuses)
func (p *Player) TakeDamage(damage int) int {
	// Rogue: Evasion - 10% chance to completely avoid damage (level 15+)
	if p.HasClass(class.Rogue) && p.GetClassLevel(class.Rogue) >= 15 {
		if stats.D100() <= 10 {
			// Evasion triggered - no damage taken
			return 0
		}
	}

	// Get effective armor including class bonuses
	totalArmor := p.GetEffectiveArmor()

	// Apply armor reduction
	actualDamage := damage - totalArmor
	if actualDamage < 1 {
		actualDamage = 1 // Minimum 1 damage
	}

	// Cleric: Sanctuary - 25% damage reduction when below 25% HP (level 20+)
	if p.HasClass(class.Cleric) && p.GetClassLevel(class.Cleric) >= 20 {
		hpPercent := float64(p.Health) / float64(p.MaxHealth) * 100
		if hpPercent < 25 {
			actualDamage = actualDamage * 75 / 100 // 25% reduction
			if actualDamage < 1 {
				actualDamage = 1
			}
		}
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

// getWeaponAttackMod returns the appropriate attack modifier for the equipped weapon
// Finesse weapons use the higher of STR or DEX
// Ranged weapons always use DEX
// Other weapons use STR
func (p *Player) getWeaponAttackMod() (int, string) {
	strMod := p.GetStrengthMod()
	dexMod := p.GetDexterityMod()

	// Check equipped weapon
	if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon {
		if weapon.IsRanged() {
			return dexMod, "DEX"
		}
		if weapon.IsFinesse() {
			// Finesse: use the higher of STR or DEX
			if dexMod > strMod {
				return dexMod, "DEX"
			}
			return strMod, "STR"
		}
	}

	// Default to STR for melee/unarmed
	return strMod, "STR"
}

// GetAttackDamage returns the damage this player deals in combat
// Uses dice rolling for weapons with damage_dice, falls back to static damage
func (p *Player) GetAttackDamage() int {
	// Get the appropriate modifier based on weapon type
	attackMod, _ := p.getWeaponAttackMod()

	// Check for equipped weapon with dice notation
	if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon {
		if weapon.DamageDice != "" {
			// Roll weapon dice + attack modifier (STR or DEX)
			damage := stats.ParseDiceWithBonus(weapon.DamageDice, attackMod)
			if damage < 1 {
				damage = 1
			}
			return damage
		}
		// Fallback to static damage + attack modifier
		damage := weapon.Damage + attackMod
		if damage < 1 {
			damage = 1
		}
		return damage
	}

	// Unarmed: 1d4 + STR modifier (always STR for unarmed)
	damage := stats.ParseDiceWithBonus("1d4", p.GetStrengthMod())
	if damage < 1 {
		damage = 1
	}
	return damage
}

// RollAttack rolls a d20 + attack modifier for attack
// Uses STR for melee, DEX for ranged, higher of STR/DEX for finesse
// Returns the roll result and the breakdown string for display
func (p *Player) RollAttack() (int, string) {
	d20 := stats.D20()
	attackMod, statName := p.getWeaponAttackMod()
	total := d20 + attackMod

	var breakdown string
	if attackMod >= 0 {
		breakdown = fmt.Sprintf("d20+%d(%s) = %d", attackMod, statName, total)
	} else {
		breakdown = fmt.Sprintf("d20%d(%s) = %d", attackMod, statName, total)
	}

	return total, breakdown
}

// GetAttackDamageAgainst calculates damage against a specific NPC target
// This includes class-based damage bonuses:
// - Warrior: +1 damage per 3 levels with melee weapons
// - Rogue: Sneak attack (+1d6, +1d6 every 5 levels) - applied on first hit (caller handles this)
// - Ranger: +2 damage with ranged weapons, +1 per 3 levels, +25% vs beasts (Favored Enemy)
// - Paladin: +2 damage vs undead and demons
func (p *Player) GetAttackDamageAgainst(target *npc.NPC, isSneakAttack bool) int {
	baseDamage := p.GetAttackDamage()
	bonusDamage := 0

	// Get equipped weapon for weapon type checks
	weapon, hasWeapon := p.Equipment[items.SlotWeapon]
	isRanged := hasWeapon && weapon.IsRanged()
	isMelee := !isRanged

	// Warrior: +1 damage per 3 levels with melee weapons
	if p.HasClass(class.Warrior) && isMelee {
		warriorLevel := p.GetClassLevel(class.Warrior)
		bonusDamage += warriorLevel / 3
	}

	// Ranger: +2 base damage with ranged, +1 per 3 levels
	if p.HasClass(class.Ranger) && isRanged {
		rangerLevel := p.GetClassLevel(class.Ranger)
		bonusDamage += 2                // Base ranged bonus
		bonusDamage += rangerLevel / 3  // Scaling bonus
	}

	// Ranger: Favored Enemy (+25% damage vs beasts)
	if p.HasClass(class.Ranger) && target != nil && target.IsBeast() {
		// Apply 25% bonus (round down)
		favoredBonus := (baseDamage + bonusDamage) / 4
		if favoredBonus < 1 {
			favoredBonus = 1
		}
		bonusDamage += favoredBonus
	}

	// Paladin: +2 damage vs undead and demons
	if p.HasClass(class.Paladin) && target != nil {
		if target.IsUndead() || target.IsDemon() {
			bonusDamage += 2
		}
	}

	// Rogue: Sneak attack (+1d6, +1d6 every 5 levels)
	if p.HasClass(class.Rogue) && isSneakAttack {
		rogueLevel := p.GetClassLevel(class.Rogue)
		sneakDice := 1 + (rogueLevel / 5) // 1d6 at level 1, 2d6 at level 5, etc.
		for i := 0; i < sneakDice; i++ {
			bonusDamage += stats.D6()
		}
	}

	totalDamage := baseDamage + bonusDamage
	if totalDamage < 1 {
		totalDamage = 1
	}

	return totalDamage
}

// IsAlive returns true if the player has health remaining
func (p *Player) IsAlive() bool {
	return p.Health > 0
}

// GetPassiveCombatRegen returns HP regen per combat tick from class passives
// Warrior: Second Wind - 1 HP per tick in combat (level 15+)
func (p *Player) GetPassiveCombatRegen() int {
	regen := 0

	// Warrior: Second Wind - 1 HP per tick in combat (level 15+)
	if p.HasClass(class.Warrior) && p.GetClassLevel(class.Warrior) >= 15 {
		regen += 1
	}

	return regen
}

// GetPassiveOutOfCombatRegen returns HP regen per minute from class passives
// Paladin: Lay on Hands - 5 HP per minute (level 15+)
func (p *Player) GetPassiveOutOfCombatRegen() int {
	regen := 0

	// Paladin: Lay on Hands - 5 HP per minute out of combat (level 15+)
	if p.HasClass(class.Paladin) && p.GetClassLevel(class.Paladin) >= 15 && !p.InCombat {
		regen += 5
	}

	return regen
}

// ApplyPassiveRegen applies class-based passive regeneration
// Returns the amount healed (0 if already at full health)
func (p *Player) ApplyPassiveRegen(inCombat bool) int {
	if p.Health >= p.MaxHealth {
		return 0
	}

	var regen int
	if inCombat {
		regen = p.GetPassiveCombatRegen()
	} else {
		regen = p.GetPassiveOutOfCombatRegen()
	}

	if regen > 0 {
		return p.Heal(regen)
	}
	return 0
}

// CanMultishot returns true if the player can trigger multishot (20% chance)
// Ranger: Multishot - 20% chance to hit twice with ranged weapons (level 20+)
func (p *Player) CanMultishot() bool {
	if !p.HasClass(class.Ranger) || p.GetClassLevel(class.Ranger) < 20 {
		return false
	}

	// Check if using ranged weapon
	if weapon, hasWeapon := p.Equipment[items.SlotWeapon]; hasWeapon {
		if weapon.IsRanged() {
			return stats.D100() <= 20 // 20% chance
		}
	}

	return false
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

	// Calculate HP gain based on active class hit die + CON modifier
	def := p.GetActiveClassDefinition()
	conMod := p.GetConstitutionMod()

	// Roll hit die for HP (average for simplicity: (hitDie/2)+1)
	// e.g., d10 = 6, d8 = 5, d6 = 4
	var hpGain int
	if def != nil {
		hpGain = (def.HitDie / 2) + 1 + conMod
	} else {
		hpGain = leveling.HPPerLevel + conMod
	}
	if hpGain < 1 {
		hpGain = 1 // Minimum 1 HP per level
	}

	// Calculate mana gain based on active class + casting stat modifier
	var manaGain int
	if def != nil {
		manaGain = def.ManaPerLevel
		switch p.activeClass {
		case class.Mage:
			manaGain += p.GetIntelligenceMod()
		case class.Cleric, class.Ranger:
			manaGain += p.GetWisdomMod()
		case class.Paladin:
			manaGain += p.GetCharismaMod()
		case class.Rogue:
			manaGain += p.GetIntelligenceMod()
		}
	} else {
		manaGain = leveling.ManaPerLevel
	}
	if manaGain < 0 {
		manaGain = 0
	}

	// Increase stats
	p.MaxHealth += hpGain
	p.MaxMana += manaGain

	// Also gain a level in the active class
	if p.classLevels != nil {
		p.classLevels.GainLevel(p.activeClass)
	}

	// Warrior: +10% HP bonus at level 20 (one-time bonus)
	// Check if we just hit level 20 as warrior
	if p.activeClass == class.Warrior && p.GetClassLevel(class.Warrior) == 20 {
		hpBonus := p.MaxHealth / 10 // 10% bonus
		p.MaxHealth += hpBonus
		hpGain += hpBonus // Include in level-up report
	}

	// Fully restore on level up
	p.Health = p.MaxHealth
	p.Mana = p.MaxMana

	return leveling.LevelUpInfo{
		NewLevel: p.Level,
		HPGain:   hpGain,
		ManaGain: manaGain,
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
	if p.discoveredPortals == nil {
		p.discoveredPortals = make(map[int]bool)
		p.discoveredPortals[0] = true // Ground floor always available
	}
	p.discoveredPortals[floorNum] = true
}

// HasDiscoveredPortal returns true if the player has discovered the portal on a floor
func (p *Player) HasDiscoveredPortal(floorNum int) bool {
	if p.discoveredPortals == nil {
		return floorNum == 0 // Ground floor always available
	}
	return p.discoveredPortals[floorNum]
}

// GetDiscoveredPortals returns a sorted list of floor numbers with discovered portals
func (p *Player) GetDiscoveredPortals() []int {
	if p.discoveredPortals == nil {
		return []int{0}
	}
	floors := make([]int, 0, len(p.discoveredPortals))
	for floor := range p.discoveredPortals {
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

// GetDiscoveredPortalsString returns discovered portals as a comma-separated string (for persistence)
func (p *Player) GetDiscoveredPortalsString() string {
	floors := p.GetDiscoveredPortals()
	strs := make([]string, len(floors))
	for i, f := range floors {
		strs[i] = fmt.Sprintf("%d", f)
	}
	return strings.Join(strs, ",")
}

// SetDiscoveredPortals sets the discovered portals from a list (used when loading from database)
func (p *Player) SetDiscoveredPortals(floors []int) {
	p.discoveredPortals = make(map[int]bool)
	p.discoveredPortals[0] = true // Ground floor always available
	for _, floor := range floors {
		p.discoveredPortals[floor] = true
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

// GetAllClassLevelsMap returns a map of class name -> level for spell access checking
func (p *Player) GetAllClassLevelsMap() map[string]int {
	result := make(map[string]int)
	if p.classLevels == nil {
		return result
	}
	for _, c := range p.classLevels.GetClasses() {
		result[c.String()] = p.classLevels.GetLevel(c)
	}
	return result
}

// CanCastSpellForClass checks if this player can cast a spell based on their class levels.
// The spell must be allowed for one of the player's classes at their current level.
// allowedClasses: list of class names that can use this spell (empty = all)
// requiredLevel: the level required in that class to use the spell
func (p *Player) CanCastSpellForClass(allowedClasses []string, requiredLevel int) bool {
	// If no class restrictions, check overall player level
	if len(allowedClasses) == 0 {
		return p.Level >= requiredLevel
	}

	// Check if any of the player's classes can use this spell
	for _, allowedClass := range allowedClasses {
		c, err := class.ParseClass(allowedClass)
		if err != nil {
			continue
		}
		if p.HasClass(c) && p.GetClassLevel(c) >= requiredLevel {
			return true
		}
	}
	return false
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

// ==================== CLASS METHODS ====================

// GetPrimaryClass returns the player's primary class
func (p *Player) GetPrimaryClass() class.Class {
	if p.classLevels == nil {
		return class.Warrior
	}
	return p.classLevels.GetPrimaryClass()
}

// GetActiveClass returns the class currently gaining XP
func (p *Player) GetActiveClass() class.Class {
	return p.activeClass
}

// SetActiveClass changes which class gains XP
func (p *Player) SetActiveClass(c class.Class) {
	p.activeClass = c
}

// GetRace returns the player's race
func (p *Player) GetRace() race.Race {
	return p.race
}

// GetRaceName returns the display name of the player's race
func (p *Player) GetRaceName() string {
	return p.race.String()
}

// SetRace sets the player's race
func (p *Player) SetRace(r race.Race) {
	p.race = r
}

// GetClassLevel returns the level in a specific class
func (p *Player) GetClassLevel(c class.Class) int {
	if p.classLevels == nil {
		return 0
	}
	return p.classLevels.GetLevel(c)
}

// GetClassLevels returns the ClassLevels struct
func (p *Player) GetClassLevels() *class.ClassLevels {
	return p.classLevels
}

// SetClassLevels sets the ClassLevels struct (used when loading from DB)
func (p *Player) SetClassLevels(cl *class.ClassLevels) {
	p.classLevels = cl
	if cl != nil {
		p.activeClass = cl.GetPrimaryClass()
	}
}

// HasClass returns true if the player has at least 1 level in a class
func (p *Player) HasClass(c class.Class) bool {
	if p.classLevels == nil {
		return false
	}
	return p.classLevels.HasClass(c)
}

// GetEffectiveLevel returns the highest class level (used for scaling)
func (p *Player) GetEffectiveLevel() int {
	if p.classLevels == nil {
		return p.Level
	}
	return p.classLevels.GetEffectiveLevel()
}

// GetClassDefinition returns the definition for the player's primary class
func (p *Player) GetClassDefinition() *class.Definition {
	return class.GetDefinition(p.GetPrimaryClass())
}

// GetActiveClassDefinition returns the definition for the player's active class
func (p *Player) GetActiveClassDefinition() *class.Definition {
	return class.GetDefinition(p.activeClass)
}

// HasArmorProficiency checks if the player can wear a specific armor type
// Returns true if any of the player's classes has the proficiency
func (p *Player) HasArmorProficiency(armorType class.ArmorType) bool {
	if p.classLevels == nil {
		return false
	}
	for _, c := range p.classLevels.GetClasses() {
		def := class.GetDefinition(c)
		if def != nil && def.HasArmorProficiency(armorType) {
			return true
		}
	}
	return false
}

// HasWeaponProficiency checks if the player can use a specific weapon type
// Returns true if any of the player's classes has the proficiency
func (p *Player) HasWeaponProficiency(weaponType class.WeaponType) bool {
	if p.classLevels == nil {
		return false
	}
	for _, c := range p.classLevels.GetClasses() {
		def := class.GetDefinition(c)
		if def != nil && def.HasWeaponProficiency(weaponType) {
			return true
		}
	}
	return false
}

// InitializeClassStats sets up HP and Mana based on class and stats
// Should be called when creating a new character with a chosen class
func (p *Player) InitializeClassStats(c class.Class) {
	def := class.GetDefinition(c)
	if def == nil {
		return
	}

	// Set up class levels
	p.classLevels = class.NewClassLevels(c)
	p.activeClass = c

	// Calculate starting HP: class base + CON modifier (min 1)
	conMod := p.GetConstitutionMod()
	startHP := def.StartingHP + conMod
	if startHP < 1 {
		startHP = 1
	}
	p.MaxHealth = startHP
	p.Health = startHP

	// Calculate starting Mana based on class
	var startMana int
	switch c {
	case class.Mage:
		startMana = def.StartingMana + p.GetIntelligenceMod()
	case class.Cleric, class.Ranger:
		startMana = def.StartingMana + p.GetWisdomMod()
	case class.Paladin:
		startMana = def.StartingMana + p.GetCharismaMod()
	case class.Rogue:
		startMana = def.StartingMana + p.GetIntelligenceMod()
	default:
		startMana = def.StartingMana
	}
	if startMana < 0 {
		startMana = 0
	}
	p.MaxMana = startMana
	p.Mana = startMana
}

// GetClassLevelsJSON returns the class levels as a JSON string for persistence
func (p *Player) GetClassLevelsJSON() string {
	if p.classLevels == nil {
		return "{}"
	}
	return p.classLevels.ToJSON()
}

// GetPrimaryClassName returns the display name of the primary class
func (p *Player) GetPrimaryClassName() string {
	return p.GetPrimaryClass().String()
}

// CanEquipItem checks if the player can equip an item based on class proficiencies
// Returns (canEquip, reason) where reason explains why if canEquip is false
func (p *Player) CanEquipItem(item *items.Item) (bool, string) {
	// Check class restriction first
	if item.RequiredClass != "" {
		requiredClass, err := class.ParseClass(item.RequiredClass)
		if err == nil && !p.HasClass(requiredClass) {
			return false, fmt.Sprintf("Only %ss can equip %s.", requiredClass.String(), item.Name)
		}
	}

	// Check armor proficiency
	if item.Type == items.Armor && item.ArmorType != "" {
		armorType := class.ArmorType(item.ArmorType)
		if !p.HasArmorProficiency(armorType) {
			return false, fmt.Sprintf("You are not proficient with %s armor.", item.ArmorType)
		}
	}

	// Check weapon proficiency
	if item.Type == items.Weapon && item.WeaponType != "" {
		weaponType := class.WeaponType(item.WeaponType)
		if !p.HasWeaponProficiency(weaponType) {
			return false, fmt.Sprintf("You are not proficient with %s weapons.", item.WeaponType)
		}
	}

	return true, ""
}

// ==================== MULTICLASS METHODS ====================

// GetActiveClassName returns the display name of the active class (the one gaining XP)
func (p *Player) GetActiveClassName() string {
	return p.activeClass.String()
}

// GetClassLevelsSummary returns a formatted string of all class levels
func (p *Player) GetClassLevelsSummary() string {
	if p.classLevels == nil {
		return p.activeClass.String() + " 1"
	}

	classes := p.classLevels.GetClasses()
	if len(classes) == 0 {
		return p.activeClass.String() + " 1"
	}

	parts := make([]string, len(classes))
	for i, c := range classes {
		level := p.classLevels.GetLevel(c)
		parts[i] = fmt.Sprintf("%s %d", c.String(), level)
	}
	return strings.Join(parts, " / ")
}

// CanMulticlass returns true if the player meets the level requirement for multiclassing
func (p *Player) CanMulticlass() bool {
	if p.classLevels == nil {
		return false
	}
	return p.classLevels.CanMulticlass()
}

// CanMulticlassInto checks if the player can multiclass into a specific class
// Returns (canMulticlass, reason) where reason explains why if false
func (p *Player) CanMulticlassInto(className string) (bool, string) {
	// Parse the class name
	targetClass, err := class.ParseClass(className)
	if err != nil {
		return false, fmt.Sprintf("Unknown class: %s", className)
	}

	// Check if already has this class
	if p.HasClass(targetClass) {
		return false, fmt.Sprintf("You already have levels in %s.", targetClass.String())
	}

	// Check if meets level requirement
	if !p.CanMulticlass() {
		primaryLevel := 0
		if p.classLevels != nil {
			primaryLevel = p.classLevels.GetLevel(p.classLevels.GetPrimaryClass())
		}
		return false, fmt.Sprintf("You must reach level %d in your primary class before multiclassing. (Currently level %d)", class.MinLevelForMulticlass, primaryLevel)
	}

	// Check stat requirements
	def := class.GetDefinition(targetClass)
	if def == nil {
		return false, fmt.Sprintf("Class definition not found for %s.", targetClass.String())
	}

	stats := map[string]int{
		"STR": p.Strength,
		"DEX": p.Dexterity,
		"CON": p.Constitution,
		"INT": p.Intelligence,
		"WIS": p.Wisdom,
		"CHA": p.Charisma,
	}

	if !def.CanMulticlassInto(stats) {
		return false, fmt.Sprintf("You don't meet the requirements to become a %s. (Requires: %s)", targetClass.String(), def.GetMulticlassRequirementsString())
	}

	return true, ""
}

// AddNewClass adds a new class at level 1 (for multiclassing)
func (p *Player) AddNewClass(className string) error {
	targetClass, err := class.ParseClass(className)
	if err != nil {
		return fmt.Errorf("unknown class: %s", className)
	}

	// Check if can multiclass into this class
	canMulti, reason := p.CanMulticlassInto(className)
	if !canMulti {
		return fmt.Errorf("%s", reason)
	}

	// Add the class
	if p.classLevels == nil {
		p.classLevels = class.NewClassLevels(p.activeClass)
	}
	p.classLevels.AddClass(targetClass)

	// Switch active class to the new class
	p.activeClass = targetClass

	return nil
}

// SwitchActiveClass switches which class gains XP
func (p *Player) SwitchActiveClass(className string) error {
	targetClass, err := class.ParseClass(className)
	if err != nil {
		return fmt.Errorf("unknown class: %s", className)
	}

	// Check if player has this class
	if !p.HasClass(targetClass) {
		return fmt.Errorf("you don't have any levels in %s", targetClass.String())
	}

	// Check if already active
	if p.activeClass == targetClass {
		return fmt.Errorf("%s is already your active class", targetClass.String())
	}

	// Check if at max level for this class
	if p.classLevels != nil && !p.classLevels.CanGainLevel(targetClass) {
		maxLevel := class.MaxPrimaryLevel
		if targetClass != p.classLevels.GetPrimaryClass() {
			maxLevel = class.MaxSecondaryLevel
		}
		return fmt.Errorf("%s is already at maximum level (%d)", targetClass.String(), maxLevel)
	}

	p.activeClass = targetClass
	return nil
}
