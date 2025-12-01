package items

import (
	"fmt"
	"math/rand"
	"os"

	"gopkg.in/yaml.v3"
)

// ItemDefinition represents an item definition from the YAML file
type ItemDefinition struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Weight      float64  `yaml:"weight"`
	Type        string   `yaml:"type"`
	Value       int      `yaml:"value"`
	Tier int `yaml:"tier,omitempty"` // Loot tier (1=common, 2=uncommon, 3=rare, 4=epic, 5=legendary)
	// Equipment fields (optional)
	Slot       string `yaml:"slot,omitempty"`
	Armor      int    `yaml:"armor,omitempty"`
	Damage     int    `yaml:"damage,omitempty"`
	DamageDice string `yaml:"damage_dice,omitempty"` // Dice notation e.g. "1d6", "2d4+1"
	TwoHanded  bool   `yaml:"two_handed,omitempty"`
	// Consumable fields (optional)
	Consumable bool `yaml:"consumable,omitempty"`
	HealAmount int  `yaml:"heal_amount,omitempty"`
	ManaAmount int  `yaml:"mana_amount,omitempty"`
}

// ItemsConfig represents the structure of the items.yaml file
type ItemsConfig struct {
	Items map[string]ItemDefinition `yaml:"items"`
}

// LoadItemsFromYAML loads item definitions from a YAML file
func LoadItemsFromYAML(filename string) (*ItemsConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read items file: %w", err)
	}

	var config ItemsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse items YAML: %w", err)
	}

	return &config, nil
}

// StringToItemType converts a string to an ItemType
func StringToItemType(typeStr string) ItemType {
	switch typeStr {
	case "weapon":
		return Weapon
	case "armor":
		return Armor
	case "food":
		return Food
	case "drink":
		return Drink
	case "potion":
		return Potion
	case "key":
		return Key
	case "container":
		return Container
	case "misc":
		return Misc
	default:
		return Misc
	}
}

// StringToEquipmentSlot converts a string to an EquipmentSlot
func StringToEquipmentSlot(slotStr string) EquipmentSlot {
	switch slotStr {
	case "head":
		return SlotHead
	case "body":
		return SlotBody
	case "legs":
		return SlotLegs
	case "feet":
		return SlotFeet
	case "hands":
		return SlotHands
	case "weapon":
		return SlotWeapon
	case "off-hand":
		return SlotOffHand
	case "held":
		return SlotHeld
	default:
		return SlotNone
	}
}

// CreateItemFromDefinition creates an Item from an ItemDefinition
// The id parameter is the YAML key for this item (e.g., "rusty_sword")
func CreateItemFromDefinition(id string, def ItemDefinition) *Item {
	item := NewItem(
		def.Name,
		def.Description,
		def.Weight,
		StringToItemType(def.Type),
		def.Value,
	)

	// Set the unique identifier
	item.ID = id

	// Set equipment fields if provided
	if def.Slot != "" {
		item.Slot = StringToEquipmentSlot(def.Slot)
	}
	item.Armor = def.Armor
	item.Damage = def.Damage
	item.DamageDice = def.DamageDice
	item.TwoHanded = def.TwoHanded

	// Set consumable fields if provided
	item.Consumable = def.Consumable
	item.HealAmount = def.HealAmount
	item.ManaAmount = def.ManaAmount

	return item
}

// GetItemByID returns an item by its ID
func (config *ItemsConfig) GetItemByID(id string) (*Item, bool) {
	def, exists := config.Items[id]
	if !exists {
		return nil, false
	}
	return CreateItemFromDefinition(id, def), true
}

// getItemIDsByTier returns all item IDs for a given tier
func (config *ItemsConfig) getItemIDsByTier(tier int) []string {
	var ids []string
	for id, def := range config.Items {
		if def.Tier == tier {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetRandomItemForTier returns a random item for the given tier
// If no items exist for the tier, falls back to lower tiers
func (config *ItemsConfig) GetRandomItemForTier(tier int, rng *rand.Rand) *Item {
	for t := tier; t >= 1; t-- {
		ids := config.getItemIDsByTier(t)
		if len(ids) > 0 {
			id := ids[rng.Intn(len(ids))]
			item, _ := config.GetItemByID(id)
			return item
		}
	}
	return nil
}

// GetRandomItemsForTier returns multiple random items for the given tier
func (config *ItemsConfig) GetRandomItemsForTier(tier int, count int, rng *rand.Rand) []*Item {
	var result []*Item
	for i := 0; i < count; i++ {
		item := config.GetRandomItemForTier(tier, rng)
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}
