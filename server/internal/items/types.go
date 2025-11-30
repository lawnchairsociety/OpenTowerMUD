package items

// ItemType represents the category of an item
type ItemType int

const (
	Misc ItemType = iota
	Weapon
	Armor
	Food
	Drink
	Potion
	Key
	Container
)

// String returns the string representation of an ItemType
func (t ItemType) String() string {
	switch t {
	case Weapon:
		return "weapon"
	case Armor:
		return "armor"
	case Food:
		return "food"
	case Drink:
		return "drink"
	case Potion:
		return "potion"
	case Key:
		return "key"
	case Container:
		return "container"
	case Misc:
		return "misc"
	default:
		return "unknown"
	}
}

// IsEquippable returns true if the item type can be equipped
func (t ItemType) IsEquippable() bool {
	return t == Weapon || t == Armor
}

// IsConsumable returns true if the item type can be consumed
func (t ItemType) IsConsumable() bool {
	return t == Food || t == Drink || t == Potion
}
