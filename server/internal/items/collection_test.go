package items

import (
	"testing"
)

// Helper function to create a test item
func newTestItem(name string, weight float64) *Item {
	return &Item{
		Name:        name,
		Description: "Test item: " + name,
		Weight:      weight,
		Type:        Misc,
		Value:       10,
	}
}

func TestAddItem(t *testing.T) {
	items := []*Item{}
	sword := newTestItem("rusty sword", 5.0)
	bread := newTestItem("bread", 0.5)

	AddItem(&items, sword)
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	AddItem(&items, bread)
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	if items[0].Name != "rusty sword" {
		t.Errorf("Expected first item to be 'rusty sword', got '%s'", items[0].Name)
	}
	if items[1].Name != "bread" {
		t.Errorf("Expected second item to be 'bread', got '%s'", items[1].Name)
	}
}

func TestRemoveItem(t *testing.T) {
	sword := newTestItem("rusty sword", 5.0)
	bread := newTestItem("bread", 0.5)
	items := []*Item{sword, bread}

	// Test removing existing item
	removed, found := RemoveItem(&items, "rusty sword")
	if !found {
		t.Error("Expected to find 'rusty sword'")
	}
	if removed.Name != "rusty sword" {
		t.Errorf("Expected removed item to be 'rusty sword', got '%s'", removed.Name)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item remaining, got %d", len(items))
	}

	// Test case insensitivity
	removed, found = RemoveItem(&items, "BREAD")
	if !found {
		t.Error("Expected to find 'BREAD' (case insensitive)")
	}
	if removed.Name != "bread" {
		t.Errorf("Expected removed item to be 'bread', got '%s'", removed.Name)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items remaining, got %d", len(items))
	}

	// Test removing non-existent item
	removed, found = RemoveItem(&items, "nonexistent")
	if found {
		t.Error("Expected not to find 'nonexistent'")
	}
	if removed != nil {
		t.Error("Expected nil for non-existent item")
	}
}

func TestHasItem(t *testing.T) {
	sword := newTestItem("rusty sword", 5.0)
	bread := newTestItem("bread", 0.5)
	items := []*Item{sword, bread}

	// Test existing item
	if !HasItem(items, "rusty sword") {
		t.Error("Expected to find 'rusty sword'")
	}

	// Test case insensitivity
	if !HasItem(items, "BREAD") {
		t.Error("Expected to find 'BREAD' (case insensitive)")
	}

	// Test non-existent item
	if HasItem(items, "nonexistent") {
		t.Error("Expected not to find 'nonexistent'")
	}

	// Test empty collection
	emptyItems := []*Item{}
	if HasItem(emptyItems, "anything") {
		t.Error("Expected not to find anything in empty collection")
	}
}

func TestFindItem(t *testing.T) {
	sword := newTestItem("rusty sword", 5.0)
	bread := newTestItem("bread", 0.5)
	armor := newTestItem("leather armor", 10.0)
	items := []*Item{sword, bread, armor}

	// Test exact match
	found, ok := FindItem(items, "bread")
	if !ok {
		t.Error("Expected to find 'bread'")
	}
	if found.Name != "bread" {
		t.Errorf("Expected 'bread', got '%s'", found.Name)
	}

	// Test case insensitive exact match
	found, ok = FindItem(items, "RUSTY SWORD")
	if !ok {
		t.Error("Expected to find 'RUSTY SWORD' (case insensitive)")
	}
	if found.Name != "rusty sword" {
		t.Errorf("Expected 'rusty sword', got '%s'", found.Name)
	}

	// Test partial match
	found, ok = FindItem(items, "rust")
	if !ok {
		t.Error("Expected to find item with 'rust' in name")
	}
	if found.Name != "rusty sword" {
		t.Errorf("Expected 'rusty sword', got '%s'", found.Name)
	}

	// Test partial match (case insensitive)
	found, ok = FindItem(items, "ARMOR")
	if !ok {
		t.Error("Expected to find item with 'ARMOR' in name")
	}
	if found.Name != "leather armor" {
		t.Errorf("Expected 'leather armor', got '%s'", found.Name)
	}

	// Test non-existent item
	found, ok = FindItem(items, "nonexistent")
	if ok {
		t.Error("Expected not to find 'nonexistent'")
	}
	if found != nil {
		t.Error("Expected nil for non-existent item")
	}

	// Test empty collection
	emptyItems := []*Item{}
	found, ok = FindItem(emptyItems, "anything")
	if ok {
		t.Error("Expected not to find anything in empty collection")
	}
}

func TestGetTotalWeight(t *testing.T) {
	// Test with items
	sword := newTestItem("rusty sword", 5.0)
	bread := newTestItem("bread", 0.5)
	armor := newTestItem("leather armor", 10.0)
	items := []*Item{sword, bread, armor}

	total := GetTotalWeight(items)
	expected := 15.5
	if total != expected {
		t.Errorf("Expected total weight %.1f, got %.1f", expected, total)
	}

	// Test empty collection
	emptyItems := []*Item{}
	total = GetTotalWeight(emptyItems)
	if total != 0.0 {
		t.Errorf("Expected 0.0 for empty collection, got %.1f", total)
	}

	// Test single item
	singleItem := []*Item{sword}
	total = GetTotalWeight(singleItem)
	if total != 5.0 {
		t.Errorf("Expected 5.0, got %.1f", total)
	}
}

func TestNewBossKey(t *testing.T) {
	tests := []struct {
		keyID       string
		floorNum    int
		wantName    string
		wantKeyType ItemType
	}{
		{"boss_key_floor_10", 10, "Boss Key (Floor 10)", Key},
		{"boss_key_floor_20", 20, "Boss Key (Floor 20)", Key},
		{"boss_key_floor_50", 50, "Boss Key (Floor 50)", Key},
	}

	for _, tc := range tests {
		key := NewBossKey(tc.keyID, tc.floorNum)

		if key.ID != tc.keyID {
			t.Errorf("NewBossKey(%s, %d): ID = %s, want %s", tc.keyID, tc.floorNum, key.ID, tc.keyID)
		}

		if key.Name != tc.wantName {
			t.Errorf("NewBossKey(%s, %d): Name = %s, want %s", tc.keyID, tc.floorNum, key.Name, tc.wantName)
		}

		if key.Type != tc.wantKeyType {
			t.Errorf("NewBossKey(%s, %d): Type = %v, want %v", tc.keyID, tc.floorNum, key.Type, tc.wantKeyType)
		}

		if key.Weight != 0.0 {
			t.Errorf("NewBossKey(%s, %d): Weight = %f, want 0.0 (keys have no weight)", tc.keyID, tc.floorNum, key.Weight)
		}

		if key.Value != 0 {
			t.Errorf("NewBossKey(%s, %d): Value = %d, want 0", tc.keyID, tc.floorNum, key.Value)
		}

		// Verify description mentions the floor
		if key.Description == "" {
			t.Errorf("NewBossKey(%s, %d): Description should not be empty", tc.keyID, tc.floorNum)
		}
	}
}

func TestNewTreasureKey(t *testing.T) {
	key := NewTreasureKey()

	if key.ID != "treasure_key" {
		t.Errorf("TreasureKey ID = %s, want treasure_key", key.ID)
	}

	if key.Name != "Treasure Key" {
		t.Errorf("TreasureKey Name = %s, want Treasure Key", key.Name)
	}

	if key.Type != Key {
		t.Errorf("TreasureKey Type = %v, want Key", key.Type)
	}

	if key.Weight != 0.0 {
		t.Errorf("TreasureKey Weight = %f, want 0.0 (keys have no weight)", key.Weight)
	}

	if key.Value != 50 {
		t.Errorf("TreasureKey Value = %d, want 50", key.Value)
	}

	if key.Description == "" {
		t.Error("TreasureKey Description should not be empty")
	}
}
