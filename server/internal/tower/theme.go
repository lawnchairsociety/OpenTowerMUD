package tower

// TowerTheme defines the configuration for a specific tower.
type TowerTheme struct {
	ID            TowerID  // Tower identifier
	Name          string   // Display name: "The Arcane Spire"
	CityName      string   // "Ironhaven"
	CityFile      string   // "data/cities/human_city.yaml"
	NPCFile       string   // "data/cities/human_npcs.yaml"
	FloorsDir     string   // "data/towers/human"
	MaxFloors     int      // 25 for racial, 100 for unified
	Descending    bool     // true for dwarf mines (flavor only)
	MobTags       []string // ["arcane", "shared"] - mobs with these tags spawn here
	SpawnRoom     string   // "town_square" - where new characters start
	TowerEntrance string   // "tower_entrance" - room leading into tower
	PortalRoom    string   // "town_square" - room with city portal
}

// themes holds all tower theme definitions.
var themes = map[TowerID]*TowerTheme{
	TowerHuman: {
		ID:            TowerHuman,
		Name:          "The Arcane Spire",
		CityName:      "Ironhaven",
		CityFile:      "data/cities/human_city.yaml",
		NPCFile:       "data/npcs.yaml",
		FloorsDir:     "data/towers/human",
		MaxFloors:     25,
		Descending:    false,
		MobTags:       []string{"shared", "human", "arcane"},
		SpawnRoom:     "town_square",
		TowerEntrance: "tower_entrance",
		PortalRoom:    "town_square",
	},
	TowerElf: {
		ID:            TowerElf,
		Name:          "The Diseased World Tree",
		CityName:      "Sylvanthal",
		CityFile:      "data/cities/elf_city.yaml",
		NPCFile:       "data/cities/elf_npcs.yaml",
		FloorsDir:     "data/towers/elf",
		MaxFloors:     25,
		Descending:    false,
		MobTags:       []string{"shared", "elf", "nature", "corrupted"},
		SpawnRoom:     "elf_town_square",
		TowerEntrance: "elf_tower_entrance",
		PortalRoom:    "elf_town_square",
	},
	TowerDwarf: {
		ID:            TowerDwarf,
		Name:          "The Descending Mines",
		CityName:      "Khazad-Karn",
		CityFile:      "data/cities/dwarf_city.yaml",
		NPCFile:       "data/cities/dwarf_npcs.yaml",
		FloorsDir:     "data/towers/dwarf",
		MaxFloors:     25,
		Descending:    true,
		MobTags:       []string{"shared", "dwarf", "underground", "construct"},
		SpawnRoom:     "dwarf_town_square",
		TowerEntrance: "dwarf_tower_entrance",
		PortalRoom:    "dwarf_town_square",
	},
	TowerGnome: {
		ID:            TowerGnome,
		Name:          "The Mechanical Tower",
		CityName:      "Cogsworth",
		CityFile:      "data/cities/gnome_city.yaml",
		NPCFile:       "data/cities/gnome_npcs.yaml",
		FloorsDir:     "data/towers/gnome",
		MaxFloors:     25,
		Descending:    false,
		MobTags:       []string{"shared", "gnome", "construct", "mechanical"},
		SpawnRoom:     "gnome_town_square",
		TowerEntrance: "gnome_tower_entrance",
		PortalRoom:    "gnome_town_square",
	},
	TowerOrc: {
		ID:            TowerOrc,
		Name:          "The Beast-Skull Tower",
		CityName:      "Skullgar",
		CityFile:      "data/cities/orc_city.yaml",
		NPCFile:       "data/cities/orc_npcs.yaml",
		FloorsDir:     "data/towers/orc",
		MaxFloors:     25,
		Descending:    false,
		MobTags:       []string{"shared", "orc", "beast", "tribal"},
		SpawnRoom:     "orc_town_square",
		TowerEntrance: "orc_tower_entrance",
		PortalRoom:    "orc_town_square",
	},
	TowerUnified: {
		ID:            TowerUnified,
		Name:          "The Infinity Spire",
		CityName:      "The Crossroads",
		CityFile:      "data/cities/unified_city.yaml",
		NPCFile:       "data/cities/unified_npcs.yaml",
		FloorsDir:     "data/towers/unified",
		MaxFloors:     100,
		Descending:    false,
		MobTags:       []string{"shared", "unified", "elite"},
		SpawnRoom:     "unified_town_square",
		TowerEntrance: "unified_tower_entrance",
		PortalRoom:    "unified_town_square",
	},
}

// GetTheme returns the theme for a given tower ID.
func GetTheme(id TowerID) *TowerTheme {
	return themes[id]
}

// GetAllThemes returns all tower themes.
func GetAllThemes() []*TowerTheme {
	result := make([]*TowerTheme, 0, len(themes))
	for _, theme := range themes {
		result = append(result, theme)
	}
	return result
}

// GetRacialThemes returns all racial tower themes (excludes unified).
func GetRacialThemes() []*TowerTheme {
	result := make([]*TowerTheme, 0, len(AllRacialTowers))
	for _, id := range AllRacialTowers {
		if theme := themes[id]; theme != nil {
			result = append(result, theme)
		}
	}
	return result
}

// GetThemeByCity returns the theme for a given city name (case-insensitive match).
func GetThemeByCity(cityName string) *TowerTheme {
	for _, theme := range themes {
		if theme.CityName == cityName {
			return theme
		}
	}
	return nil
}
