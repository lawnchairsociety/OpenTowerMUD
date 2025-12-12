package tower

// TowerTitles holds the first-clear and shared titles for a tower.
type TowerTitles struct {
	FirstClearTitle  string // Unique title for first player to clear
	SharedClearTitle string // Title for subsequent clears
}

// towerTitles maps tower IDs to their title definitions.
var towerTitles = map[TowerID]TowerTitles{
	TowerHuman: {
		FirstClearTitle:  "Archivist's End",
		SharedClearTitle: "Spire Conqueror",
	},
	TowerElf: {
		FirstClearTitle:  "Purifier of the Grove",
		SharedClearTitle: "Blight Cleanser",
	},
	TowerDwarf: {
		FirstClearTitle:  "Delver of the Depths",
		SharedClearTitle: "Deep Guardian Slayer",
	},
	TowerGnome: {
		FirstClearTitle:  "The Great Debugger",
		SharedClearTitle: "Machine Breaker",
	},
	TowerOrc: {
		FirstClearTitle:  "Kingslayer",
		SharedClearTitle: "Ancestor's Champion",
	},
	TowerUnified: {
		FirstClearTitle:  "Savior of the Realm",
		SharedClearTitle: "Blight Slayer",
	},
}

// GetFirstClearTitle returns the unique title for the first player to clear a tower.
func GetFirstClearTitle(towerID TowerID) string {
	if titles, ok := towerTitles[towerID]; ok {
		return titles.FirstClearTitle
	}
	return ""
}

// GetSharedClearTitle returns the title for subsequent clears of a tower.
func GetSharedClearTitle(towerID TowerID) string {
	if titles, ok := towerTitles[towerID]; ok {
		return titles.SharedClearTitle
	}
	return ""
}

// GetTowerTitles returns both titles for a tower.
func GetTowerTitles(towerID TowerID) (firstClear, sharedClear string) {
	if titles, ok := towerTitles[towerID]; ok {
		return titles.FirstClearTitle, titles.SharedClearTitle
	}
	return "", ""
}
