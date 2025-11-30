package world

import "testing"

func TestRoomTypeString(t *testing.T) {
	tests := []struct {
		rt   RoomType
		want string
	}{
		{RoomTypeCity, "city"},
		{RoomTypeCorridor, "corridor"},
		{RoomTypeRoom, "room"},
		{RoomTypeStairs, "stairs"},
		{RoomTypeTreasure, "treasure"},
		{RoomTypeBoss, "boss"},
	}

	for _, tt := range tests {
		if got := tt.rt.String(); got != tt.want {
			t.Errorf("RoomType(%d).String() = %q, want %q", tt.rt, got, tt.want)
		}
	}
}

func TestRoomTypeIsSafe(t *testing.T) {
	if !RoomTypeCity.IsSafe() {
		t.Error("City should be safe")
	}
	if RoomTypeBoss.IsSafe() {
		t.Error("Boss should not be safe")
	}
}

func TestParseRoomType(t *testing.T) {
	tests := []struct {
		input string
		want  RoomType
		ok    bool
	}{
		{"city", RoomTypeCity, true},
		{"corridor", RoomTypeCorridor, true},
		{"room", RoomTypeRoom, true},
		{"stairs", RoomTypeStairs, true},
		{"treasure", RoomTypeTreasure, true},
		{"boss", RoomTypeBoss, true},
		{"invalid", RoomTypeCity, false},
	}

	for _, tt := range tests {
		got, ok := ParseRoomType(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseRoomType(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestRoomTypeDangerLevel(t *testing.T) {
	if RoomTypeCity.GetDangerLevel() != 0 {
		t.Error("City danger level should be 0")
	}
	if RoomTypeBoss.GetDangerLevel() != 5 {
		t.Error("Boss danger level should be 5")
	}
}
