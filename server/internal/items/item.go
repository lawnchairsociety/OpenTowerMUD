package items

import "fmt"

// EquipmentSlot represents where an item can be equipped
type EquipmentSlot int

const (
	SlotNone EquipmentSlot = iota
	SlotHead
	SlotNeck
	SlotBody
	SlotBack
	SlotLegs
	SlotFeet
	SlotHands
	SlotRing
	SlotWeapon
	SlotOffHand
	SlotHeld
)

// String returns the string representation of an EquipmentSlot
func (s EquipmentSlot) String() string {
	switch s {
	case SlotHead:
		return "head"
	case SlotNeck:
		return "neck"
	case SlotBody:
		return "body"
	case SlotBack:
		return "back"
	case SlotLegs:
		return "legs"
	case SlotFeet:
		return "feet"
	case SlotHands:
		return "hands"
	case SlotRing:
		return "ring"
	case SlotWeapon:
		return "weapon"
	case SlotOffHand:
		return "off-hand"
	case SlotHeld:
		return "held"
	default:
		return "none"
	}
}

// Item represents an in-game item with properties
type Item struct {
	ID          string // Unique identifier from YAML key (e.g., "rusty_sword")
	Name        string
	Description string
	Weight      float64
	Type        ItemType
	Value       int // Gold value
	// Equipment stats (optional, only for equippable items)
	Slot       EquipmentSlot
	Armor      int    // Damage reduction for armor
	Damage     int    // Damage value for weapons (legacy, used as fallback)
	DamageDice string // Dice notation for damage (e.g., "1d6", "2d4+1")
	TwoHanded  bool   // Whether weapon requires both hands
	// Proficiency requirements
	ArmorType  string // light, medium, heavy, shield, none (for armor)
	WeaponType string // simple, martial, finesse, ranged (for weapons)
	// Class restrictions (optional - if empty, any class can use)
	RequiredClass string // e.g., "mage", "cleric" - only this class can equip
	// Consumable stats (optional, only for consumable items)
	Consumable bool // Can this item be consumed?
	HealAmount int  // HP restored when consumed
	ManaAmount int  // MP restored when consumed
	// Unique item flag - player can only have one of these
	Unique bool // If true, player can only possess one instance of this item
}

// NewItem creates a new item with the given properties
func NewItem(name, description string, weight float64, itemType ItemType, value int) *Item {
	return &Item{
		Name:        name,
		Description: description,
		Weight:      weight,
		Type:        itemType,
		Value:       value,
		Slot:        SlotNone,
		Armor:       0,
		Damage:      0,
		TwoHanded:   false,
		Consumable:  false,
		HealAmount:  0,
		ManaAmount:  0,
	}
}

// NewWeapon creates a new weapon item
func NewWeapon(name, description string, weight float64, value, damage int, twoHanded bool) *Item {
	return &Item{
		Name:        name,
		Description: description,
		Weight:      weight,
		Type:        Weapon,
		Value:       value,
		Slot:        SlotWeapon,
		Damage:      damage,
		TwoHanded:   twoHanded,
	}
}

// NewArmor creates a new armor item
func NewArmor(name, description string, weight float64, value, armor int, slot EquipmentSlot) *Item {
	return &Item{
		Name:        name,
		Description: description,
		Weight:      weight,
		Type:        Armor,
		Value:       value,
		Slot:        slot,
		Armor:       armor,
	}
}

// NewConsumable creates a new consumable item
func NewConsumable(name, description string, weight float64, itemType ItemType, value, healAmount, manaAmount int) *Item {
	return &Item{
		Name:        name,
		Description: description,
		Weight:      weight,
		Type:        itemType,
		Value:       value,
		Consumable:  true,
		HealAmount:  healAmount,
		ManaAmount:  manaAmount,
	}
}

// NewBossKey creates a boss key item for a specific floor
func NewBossKey(keyID string, floorNum int) *Item {
	return &Item{
		ID:          keyID,
		Name:        fmt.Sprintf("Boss Key (Floor %d)", floorNum),
		Description: fmt.Sprintf("A heavy iron key dropped by the boss of floor %d. It unlocks the sealed door to the next level.", floorNum),
		Weight:      0.0, // Keys have no weight (stored on key ring)
		Type:        Key,
		Value:       0, // Boss keys are not sellable
	}
}

// NewTreasureKey creates a treasure room key item
func NewTreasureKey() *Item {
	return &Item{
		ID:          "treasure_key",
		Name:        "Treasure Key",
		Description: "A golden key that unlocks treasure room doors in the tower. Single use.",
		Weight:      0.0, // Keys have no weight (stored on key ring)
		Type:        Key,
		Value:       50, // Can be purchased at shop
	}
}

// NewLegendaryKey creates a legendary key item (master key)
func NewLegendaryKey() *Item {
	return &Item{
		ID:          "legendary_key",
		Name:        "Legendary Key",
		Description: "A golden key that glows with immense power. It can unlock any door in the tower.",
		Weight:      0.0, // Keys have no weight (stored on key ring)
		Type:        Key,
		Value:       0,    // Not sellable
		Unique:      true, // Can only have one
	}
}

// String returns a formatted string representation of the item
func (i *Item) String() string {
	return fmt.Sprintf("%s (%s, %.1f, %d gold)", i.Name, i.Type.String(), i.Weight, i.Value)
}

// IsFinesse returns true if this weapon can use DEX instead of STR
func (i *Item) IsFinesse() bool {
	return i.WeaponType == "finesse"
}

// IsRanged returns true if this is a ranged weapon
func (i *Item) IsRanged() bool {
	return i.WeaponType == "ranged"
}

// UsesDexterity returns true if this weapon should use DEX for attack/damage
// Finesse weapons can use either STR or DEX (player chooses higher)
// Ranged weapons always use DEX
func (i *Item) UsesDexterity() bool {
	return i.IsRanged() || i.IsFinesse()
}
