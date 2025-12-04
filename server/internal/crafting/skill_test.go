package crafting

import "testing"

func TestAllSkills(t *testing.T) {
	skills := AllSkills()
	if len(skills) != 4 {
		t.Errorf("Expected 4 skills, got %d", len(skills))
	}

	expected := map[CraftingSkill]bool{
		Blacksmithing:  true,
		Leatherworking: true,
		Alchemy:        true,
		Enchanting:     true,
	}

	for _, skill := range skills {
		if !expected[skill] {
			t.Errorf("Unexpected skill: %s", skill)
		}
	}
}

func TestParseSkill(t *testing.T) {
	tests := []struct {
		input    string
		expected CraftingSkill
		hasError bool
	}{
		{"blacksmithing", Blacksmithing, false},
		{"Blacksmithing", Blacksmithing, false},
		{"BLACKSMITHING", Blacksmithing, false},
		{"leatherworking", Leatherworking, false},
		{"alchemy", Alchemy, false},
		{"enchanting", Enchanting, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tc := range tests {
		skill, err := ParseSkill(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("ParseSkill(%q) expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseSkill(%q) unexpected error: %v", tc.input, err)
			}
			if skill != tc.expected {
				t.Errorf("ParseSkill(%q) = %v, want %v", tc.input, skill, tc.expected)
			}
		}
	}
}

func TestSkillStation(t *testing.T) {
	tests := []struct {
		skill    CraftingSkill
		expected string
	}{
		{Blacksmithing, StationForge},
		{Leatherworking, StationWorkbench},
		{Alchemy, StationAlchemyLab},
		{Enchanting, StationEnchantingTable},
	}

	for _, tc := range tests {
		station := tc.skill.Station()
		if station != tc.expected {
			t.Errorf("%s.Station() = %q, want %q", tc.skill, station, tc.expected)
		}
	}
}

func TestStationName(t *testing.T) {
	tests := []struct {
		station  string
		expected string
	}{
		{StationForge, "Forge"},
		{StationWorkbench, "Workbench"},
		{StationAlchemyLab, "Alchemy Lab"},
		{StationEnchantingTable, "Enchanting Table"},
		{"unknown", "unknown"},
	}

	for _, tc := range tests {
		name := StationName(tc.station)
		if name != tc.expected {
			t.Errorf("StationName(%q) = %q, want %q", tc.station, name, tc.expected)
		}
	}
}

func TestGetSkillForStation(t *testing.T) {
	tests := []struct {
		station  string
		expected CraftingSkill
		ok       bool
	}{
		{StationForge, Blacksmithing, true},
		{StationWorkbench, Leatherworking, true},
		{StationAlchemyLab, Alchemy, true},
		{StationEnchantingTable, Enchanting, true},
		{"unknown", "", false},
	}

	for _, tc := range tests {
		skill, ok := GetSkillForStation(tc.station)
		if ok != tc.ok {
			t.Errorf("GetSkillForStation(%q) ok = %v, want %v", tc.station, ok, tc.ok)
		}
		if skill != tc.expected {
			t.Errorf("GetSkillForStation(%q) = %v, want %v", tc.station, skill, tc.expected)
		}
	}
}

func TestCraftingSkillString(t *testing.T) {
	tests := []struct {
		skill    CraftingSkill
		expected string
	}{
		{Blacksmithing, "Blacksmithing"},
		{Leatherworking, "Leatherworking"},
		{Alchemy, "Alchemy"},
		{Enchanting, "Enchanting"},
		{CraftingSkill("unknown"), "unknown"},
	}

	for _, tc := range tests {
		result := tc.skill.String()
		if result != tc.expected {
			t.Errorf("%s.String() = %q, want %q", tc.skill, result, tc.expected)
		}
	}
}
