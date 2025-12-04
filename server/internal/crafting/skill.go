package crafting

import (
	"fmt"
	"strings"
)

// CraftingSkill represents a type of crafting skill
type CraftingSkill string

const (
	Blacksmithing  CraftingSkill = "blacksmithing"
	Leatherworking CraftingSkill = "leatherworking"
	Alchemy        CraftingSkill = "alchemy"
	Enchanting     CraftingSkill = "enchanting"
)

// MaxSkillLevel is the maximum level for any crafting skill
const MaxSkillLevel = 100

// Station types
const (
	StationForge           = "forge"
	StationWorkbench       = "workbench"
	StationAlchemyLab      = "alchemy_lab"
	StationEnchantingTable = "enchanting_table"
)

// AllSkills returns all available crafting skills
func AllSkills() []CraftingSkill {
	return []CraftingSkill{
		Blacksmithing,
		Leatherworking,
		Alchemy,
		Enchanting,
	}
}

// String returns the display name of the skill
func (s CraftingSkill) String() string {
	switch s {
	case Blacksmithing:
		return "Blacksmithing"
	case Leatherworking:
		return "Leatherworking"
	case Alchemy:
		return "Alchemy"
	case Enchanting:
		return "Enchanting"
	default:
		return string(s)
	}
}

// Station returns the required station for this skill
func (s CraftingSkill) Station() string {
	switch s {
	case Blacksmithing:
		return StationForge
	case Leatherworking:
		return StationWorkbench
	case Alchemy:
		return StationAlchemyLab
	case Enchanting:
		return StationEnchantingTable
	default:
		return ""
	}
}

// StationName returns the display name for a station type
func StationName(station string) string {
	switch station {
	case StationForge:
		return "Forge"
	case StationWorkbench:
		return "Workbench"
	case StationAlchemyLab:
		return "Alchemy Lab"
	case StationEnchantingTable:
		return "Enchanting Table"
	default:
		return station
	}
}

// ParseSkill parses a string into a CraftingSkill
func ParseSkill(s string) (CraftingSkill, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "blacksmithing", "smithing", "smith":
		return Blacksmithing, nil
	case "leatherworking", "leather", "leatherwork":
		return Leatherworking, nil
	case "alchemy", "alch":
		return Alchemy, nil
	case "enchanting", "enchant":
		return Enchanting, nil
	default:
		return "", fmt.Errorf("unknown crafting skill: %s", s)
	}
}

// IsValidStation checks if a station type is valid
func IsValidStation(station string) bool {
	switch station {
	case StationForge, StationWorkbench, StationAlchemyLab, StationEnchantingTable:
		return true
	default:
		return false
	}
}

// GetSkillForStation returns the crafting skill associated with a station
func GetSkillForStation(station string) (CraftingSkill, bool) {
	switch station {
	case StationForge:
		return Blacksmithing, true
	case StationWorkbench:
		return Leatherworking, true
	case StationAlchemyLab:
		return Alchemy, true
	case StationEnchantingTable:
		return Enchanting, true
	default:
		return "", false
	}
}
