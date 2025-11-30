package items

import "strings"

// AddItem adds an item to a collection
func AddItem(items *[]*Item, item *Item) {
	*items = append(*items, item)
}

// RemoveItem removes an item from a collection by name (case-insensitive)
// Returns the removed item and true if found, nil and false otherwise
func RemoveItem(items *[]*Item, itemName string) (*Item, bool) {
	for i, item := range *items {
		if strings.EqualFold(item.Name, itemName) {
			removed := item
			*items = append((*items)[:i], (*items)[i+1:]...)
			return removed, true
		}
	}
	return nil, false
}

// HasItem checks if an item exists in a collection (case-insensitive)
func HasItem(items []*Item, itemName string) bool {
	for _, item := range items {
		if strings.EqualFold(item.Name, itemName) {
			return true
		}
	}
	return false
}

// FindItem searches for an item using partial matching (case-insensitive)
// Returns the item pointer and true if found, nil and false otherwise
// If multiple items match, returns the first match
func FindItem(items []*Item, partial string) (*Item, bool) {
	partial = strings.ToLower(partial)

	// First, try exact match
	for _, item := range items {
		if strings.EqualFold(item.Name, partial) {
			return item, true
		}
	}

	// Then try partial match (item name contains the search term)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), partial) {
			return item, true
		}
	}

	return nil, false
}

// GetTotalWeight calculates the total weight of all items in a collection
func GetTotalWeight(items []*Item) float64 {
	total := 0.0
	for _, item := range items {
		total += item.Weight
	}
	return total
}
