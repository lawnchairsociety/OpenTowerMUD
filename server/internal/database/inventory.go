package database

import (
	"fmt"
)

// InventoryItem represents an item in a character's inventory.
type InventoryItem struct {
	ID          int64
	CharacterID int64
	ItemID      string // References item_id from items.yaml
}

// EquipmentItem represents an equipped item.
type EquipmentItem struct {
	ID          int64
	CharacterID int64
	Slot        string // head, body, legs, feet, weapon, offhand, held
	ItemID      string // References item_id from items.yaml
}

// SaveInventory replaces all inventory items for a character.
// This is a full replace operation - existing items are deleted first.
func (d *Database) SaveInventory(characterID int64, itemIDs []string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing inventory
	if _, err := tx.Exec("DELETE FROM inventory WHERE character_id = ?", characterID); err != nil {
		return fmt.Errorf("failed to clear inventory: %w", err)
	}

	// Insert new items
	if len(itemIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO inventory (character_id, item_id) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, itemID := range itemIDs {
			if _, err := stmt.Exec(characterID, itemID); err != nil {
				return fmt.Errorf("failed to insert inventory item: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadInventory retrieves all inventory item IDs for a character.
func (d *Database) LoadInventory(characterID int64) ([]string, error) {
	rows, err := d.db.Query(
		"SELECT item_id FROM inventory WHERE character_id = ?",
		characterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory: %w", err)
	}
	defer rows.Close()

	var itemIDs []string
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("failed to scan inventory item: %w", err)
		}
		itemIDs = append(itemIDs, itemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory: %w", err)
	}

	return itemIDs, nil
}

// SaveEquipment replaces all equipped items for a character.
// equipment is a map of slot -> item_id.
func (d *Database) SaveEquipment(characterID int64, equipment map[string]string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing equipment
	if _, err := tx.Exec("DELETE FROM equipment WHERE character_id = ?", characterID); err != nil {
		return fmt.Errorf("failed to clear equipment: %w", err)
	}

	// Insert new equipment
	if len(equipment) > 0 {
		stmt, err := tx.Prepare("INSERT INTO equipment (character_id, slot, item_id) VALUES (?, ?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for slot, itemID := range equipment {
			if _, err := stmt.Exec(characterID, slot, itemID); err != nil {
				return fmt.Errorf("failed to insert equipment: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadEquipment retrieves all equipped items for a character.
// Returns a map of slot -> item_id.
func (d *Database) LoadEquipment(characterID int64) (map[string]string, error) {
	rows, err := d.db.Query(
		"SELECT slot, item_id FROM equipment WHERE character_id = ?",
		characterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query equipment: %w", err)
	}
	defer rows.Close()

	equipment := make(map[string]string)
	for rows.Next() {
		var slot, itemID string
		if err := rows.Scan(&slot, &itemID); err != nil {
			return nil, fmt.Errorf("failed to scan equipment: %w", err)
		}
		equipment[slot] = itemID
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating equipment: %w", err)
	}

	return equipment, nil
}

// SaveCharacterFull saves character stats, inventory, and equipment in a single transaction.
func (d *Database) SaveCharacterFull(c *Character, inventoryIDs []string, equipment map[string]string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save character stats
	_, err = tx.Exec(
		`UPDATE characters SET
			room_id = ?,
			health = ?,
			max_health = ?,
			mana = ?,
			max_mana = ?,
			level = ?,
			experience = ?,
			state = ?,
			max_carry_weight = ?,
			learned_spells = ?,
			discovered_portals = ?,
			strength = ?,
			dexterity = ?,
			constitution = ?,
			intelligence = ?,
			wisdom = ?,
			charisma = ?,
			gold = ?,
			key_ring = ?,
			primary_class = ?,
			class_levels = ?,
			active_class = ?,
			race = ?,
			crafting_skills = ?,
			known_recipes = ?,
			last_played = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		c.RoomID, c.Health, c.MaxHealth, c.Mana, c.MaxMana,
		c.Level, c.Experience, c.State, c.MaxCarryWeight, c.LearnedSpells,
		c.DiscoveredPortals, c.Strength, c.Dexterity, c.Constitution, c.Intelligence, c.Wisdom, c.Charisma,
		c.Gold, c.KeyRing, c.PrimaryClass, c.ClassLevels, c.ActiveClass, c.Race,
		c.CraftingSkills, c.KnownRecipes, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to save character: %w", err)
	}

	// Clear and save inventory
	if _, err := tx.Exec("DELETE FROM inventory WHERE character_id = ?", c.ID); err != nil {
		return fmt.Errorf("failed to clear inventory: %w", err)
	}

	if len(inventoryIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO inventory (character_id, item_id) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare inventory statement: %w", err)
		}
		defer stmt.Close()

		for _, itemID := range inventoryIDs {
			if _, err := stmt.Exec(c.ID, itemID); err != nil {
				return fmt.Errorf("failed to insert inventory item: %w", err)
			}
		}
	}

	// Clear and save equipment
	if _, err := tx.Exec("DELETE FROM equipment WHERE character_id = ?", c.ID); err != nil {
		return fmt.Errorf("failed to clear equipment: %w", err)
	}

	if len(equipment) > 0 {
		stmt, err := tx.Prepare("INSERT INTO equipment (character_id, slot, item_id) VALUES (?, ?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare equipment statement: %w", err)
		}
		defer stmt.Close()

		for slot, itemID := range equipment {
			if _, err := stmt.Exec(c.ID, slot, itemID); err != nil {
				return fmt.Errorf("failed to insert equipment: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
