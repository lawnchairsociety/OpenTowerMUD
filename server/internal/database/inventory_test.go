package database

import (
	"reflect"
	"sort"
	"testing"
)

func TestSaveAndLoadInventory(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save inventory
	items := []string{"rusty_sword", "healing_potion", "bread"}
	err := db.SaveInventory(char.ID, items)
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Load inventory
	loaded, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Sort both slices for comparison
	sort.Strings(items)
	sort.Strings(loaded)

	if !reflect.DeepEqual(items, loaded) {
		t.Errorf("Expected inventory %v, got %v", items, loaded)
	}
}

func TestSaveInventoryReplacesExisting(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save initial inventory
	err := db.SaveInventory(char.ID, []string{"item1", "item2"})
	if err != nil {
		t.Fatalf("Failed to save initial inventory: %v", err)
	}

	// Save new inventory (should replace)
	err = db.SaveInventory(char.ID, []string{"item3"})
	if err != nil {
		t.Fatalf("Failed to save new inventory: %v", err)
	}

	// Load and verify
	loaded, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	if len(loaded) != 1 || loaded[0] != "item3" {
		t.Errorf("Expected [item3], got %v", loaded)
	}
}

func TestSaveEmptyInventory(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save with items
	err := db.SaveInventory(char.ID, []string{"item1"})
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Clear inventory
	err = db.SaveInventory(char.ID, []string{})
	if err != nil {
		t.Fatalf("Failed to clear inventory: %v", err)
	}

	// Verify empty
	loaded, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("Expected empty inventory, got %v", loaded)
	}
}

func TestLoadInventoryEmpty(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Load without saving anything
	loaded, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Failed to load empty inventory: %v", err)
	}

	if loaded != nil && len(loaded) != 0 {
		t.Errorf("Expected empty/nil inventory, got %v", loaded)
	}
}

func TestSaveAndLoadEquipment(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save equipment
	equipment := map[string]string{
		"weapon": "rusty_sword",
		"body":   "leather_armor",
		"head":   "leather_cap",
	}
	err := db.SaveEquipment(char.ID, equipment)
	if err != nil {
		t.Fatalf("Failed to save equipment: %v", err)
	}

	// Load equipment
	loaded, err := db.LoadEquipment(char.ID)
	if err != nil {
		t.Fatalf("Failed to load equipment: %v", err)
	}

	if !reflect.DeepEqual(equipment, loaded) {
		t.Errorf("Expected equipment %v, got %v", equipment, loaded)
	}
}

func TestSaveEquipmentReplacesExisting(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save initial equipment
	err := db.SaveEquipment(char.ID, map[string]string{
		"weapon": "rusty_sword",
		"body":   "leather_armor",
	})
	if err != nil {
		t.Fatalf("Failed to save initial equipment: %v", err)
	}

	// Save new equipment (should replace)
	err = db.SaveEquipment(char.ID, map[string]string{
		"weapon": "iron_dagger",
	})
	if err != nil {
		t.Fatalf("Failed to save new equipment: %v", err)
	}

	// Load and verify
	loaded, err := db.LoadEquipment(char.ID)
	if err != nil {
		t.Fatalf("Failed to load equipment: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("Expected 1 item, got %d", len(loaded))
	}
	if loaded["weapon"] != "iron_dagger" {
		t.Errorf("Expected weapon 'iron_dagger', got '%s'", loaded["weapon"])
	}
}

func TestSaveEmptyEquipment(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save with equipment
	err := db.SaveEquipment(char.ID, map[string]string{"weapon": "sword"})
	if err != nil {
		t.Fatalf("Failed to save equipment: %v", err)
	}

	// Clear equipment
	err = db.SaveEquipment(char.ID, map[string]string{})
	if err != nil {
		t.Fatalf("Failed to clear equipment: %v", err)
	}

	// Verify empty
	loaded, err := db.LoadEquipment(char.ID)
	if err != nil {
		t.Fatalf("Failed to load equipment: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("Expected empty equipment, got %v", loaded)
	}
}

func TestSaveCharacterFull(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Modify character
	char.Health = 75
	char.Level = 3
	char.RoomID = "dark_forest"

	inventory := []string{"healing_potion", "bread"}
	equipment := map[string]string{
		"weapon": "rusty_sword",
		"body":   "leather_armor",
	}

	// Save full character
	err := db.SaveCharacterFull(char, inventory, equipment)
	if err != nil {
		t.Fatalf("Failed to save character full: %v", err)
	}

	// Verify character
	loaded, _ := db.GetCharacterByID(char.ID)
	if loaded.Health != 75 {
		t.Errorf("Expected health 75, got %d", loaded.Health)
	}
	if loaded.Level != 3 {
		t.Errorf("Expected level 3, got %d", loaded.Level)
	}
	if loaded.RoomID != "dark_forest" {
		t.Errorf("Expected room 'dark_forest', got '%s'", loaded.RoomID)
	}

	// Verify inventory
	loadedInv, _ := db.LoadInventory(char.ID)
	sort.Strings(inventory)
	sort.Strings(loadedInv)
	if !reflect.DeepEqual(inventory, loadedInv) {
		t.Errorf("Expected inventory %v, got %v", inventory, loadedInv)
	}

	// Verify equipment
	loadedEquip, _ := db.LoadEquipment(char.ID)
	if !reflect.DeepEqual(equipment, loadedEquip) {
		t.Errorf("Expected equipment %v, got %v", equipment, loadedEquip)
	}
}

func TestDuplicateItems(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save inventory with duplicate items
	items := []string{"healing_potion", "healing_potion", "bread"}
	err := db.SaveInventory(char.ID, items)
	if err != nil {
		t.Fatalf("Failed to save inventory with duplicates: %v", err)
	}

	// Load and verify duplicates preserved
	loaded, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 items (with duplicates), got %d", len(loaded))
	}
}

func TestCharacterDeleteCascades(t *testing.T) {
	db := setupTestDB(t)

	account, _ := db.CreateAccount("testuser", "password123")
	char, _ := db.CreateCharacter(account.ID, "TestHero")

	// Save inventory and equipment
	db.SaveInventory(char.ID, []string{"item1", "item2"})
	db.SaveEquipment(char.ID, map[string]string{"weapon": "sword"})

	// Delete character
	err := db.DeleteCharacter(char.ID)
	if err != nil {
		t.Fatalf("Failed to delete character: %v", err)
	}

	// Verify inventory deleted (should return empty, not error)
	inv, err := db.LoadInventory(char.ID)
	if err != nil {
		t.Fatalf("Error loading inventory after delete: %v", err)
	}
	if len(inv) != 0 {
		t.Errorf("Inventory should be empty after character delete, got %v", inv)
	}

	// Verify equipment deleted
	equip, err := db.LoadEquipment(char.ID)
	if err != nil {
		t.Fatalf("Error loading equipment after delete: %v", err)
	}
	if len(equip) != 0 {
		t.Errorf("Equipment should be empty after character delete, got %v", equip)
	}
}
