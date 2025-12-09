package tower

// TowerID identifies a specific tower in the game world.
type TowerID string

const (
	TowerHuman   TowerID = "human"
	TowerElf     TowerID = "elf"
	TowerDwarf   TowerID = "dwarf"
	TowerGnome   TowerID = "gnome"
	TowerOrc     TowerID = "orc"
	TowerUnified TowerID = "unified"
)

// AllRacialTowers contains all racial tower IDs (excludes unified).
var AllRacialTowers = []TowerID{TowerHuman, TowerElf, TowerDwarf, TowerGnome, TowerOrc}

// AllTowers contains all tower IDs including unified.
var AllTowers = []TowerID{TowerHuman, TowerElf, TowerDwarf, TowerGnome, TowerOrc, TowerUnified}

// String returns the string representation of the tower ID.
func (id TowerID) String() string {
	return string(id)
}

// IsValid returns true if the tower ID is a recognized value.
func (id TowerID) IsValid() bool {
	switch id {
	case TowerHuman, TowerElf, TowerDwarf, TowerGnome, TowerOrc, TowerUnified:
		return true
	}
	return false
}

// IsRacial returns true if this is a racial tower (not unified).
func (id TowerID) IsRacial() bool {
	switch id {
	case TowerHuman, TowerElf, TowerDwarf, TowerGnome, TowerOrc:
		return true
	}
	return false
}

// ParseTowerID converts a string to a TowerID, returning false if invalid.
func ParseTowerID(s string) (TowerID, bool) {
	id := TowerID(s)
	return id, id.IsValid()
}
