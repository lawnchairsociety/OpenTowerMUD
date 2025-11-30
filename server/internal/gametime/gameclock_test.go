package gametime

import (
	"sync"
	"testing"
)

func TestNewGameClock(t *testing.T) {
	gc := NewGameClock()
	if gc.GetHour() != 0 {
		t.Errorf("Expected initial hour to be 0, got %d", gc.GetHour())
	}
}

func TestAdvanceHour(t *testing.T) {
	gc := NewGameClock()

	// Advance through all hours
	for expected := 1; expected < 24; expected++ {
		gc.AdvanceHour()
		if gc.GetHour() != expected {
			t.Errorf("Expected hour %d, got %d", expected, gc.GetHour())
		}
	}

	// Test wrap-around at midnight
	gc.AdvanceHour()
	if gc.GetHour() != 0 {
		t.Errorf("Expected hour to wrap to 0, got %d", gc.GetHour())
	}
}

func TestIsDay(t *testing.T) {
	tests := []struct {
		hour     int
		expected bool
	}{
		{0, false},   // Midnight - night
		{5, false},   // Pre-dawn - night
		{6, true},    // Dawn - day
		{12, true},   // Noon - day
		{17, true},   // Late afternoon - day
		{18, false},  // Dusk - night
		{23, false},  // Late night - night
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		if gc.IsDay() != tt.expected {
			t.Errorf("Hour %d: expected IsDay()=%v, got %v", tt.hour, tt.expected, gc.IsDay())
		}
	}
}

func TestIsNight(t *testing.T) {
	tests := []struct {
		hour     int
		expected bool
	}{
		{0, true},    // Midnight - night
		{5, true},    // Pre-dawn - night
		{6, false},   // Dawn - day
		{12, false},  // Noon - day
		{17, false},  // Late afternoon - day
		{18, true},   // Dusk - night
		{23, true},   // Late night - night
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		if gc.IsNight() != tt.expected {
			t.Errorf("Hour %d: expected IsNight()=%v, got %v", tt.hour, tt.expected, gc.IsNight())
		}
	}
}

func TestGetTimeOfDay(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{0, "night"},
		{3, "night"},
		{6, "morning"},
		{9, "morning"},
		{12, "afternoon"},
		{15, "afternoon"},
		{18, "evening"},
		{21, "evening"},
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		if gc.GetTimeOfDay() != tt.expected {
			t.Errorf("Hour %d: expected %s, got %s", tt.hour, tt.expected, gc.GetTimeOfDay())
		}
	}
}

func TestGetTimeString(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{0, "00:00"},
		{6, "06:00"},
		{12, "12:00"},
		{18, "18:00"},
		{23, "23:00"},
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		if gc.GetTimeString() != tt.expected {
			t.Errorf("Hour %d: expected %s, got %s", tt.hour, tt.expected, gc.GetTimeString())
		}
	}
}

func TestGetMinutesUntilNextPeriod(t *testing.T) {
	tests := []struct {
		hour     int
		expected float64
	}{
		{6, 12 * 2.5},   // Dawn: 12 hours until dusk = 30 minutes
		{12, 6 * 2.5},   // Noon: 6 hours until dusk = 15 minutes
		{17, 1 * 2.5},   // Late afternoon: 1 hour until dusk = 2.5 minutes
		{18, 12 * 2.5},  // Dusk: 12 hours until dawn = 30 minutes
		{0, 6 * 2.5},    // Midnight: 6 hours until dawn = 15 minutes
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		result := gc.GetMinutesUntilNextPeriod()
		if result != tt.expected {
			t.Errorf("Hour %d: expected %.1f minutes, got %.1f", tt.hour, tt.expected, result)
		}
	}
}

func TestGetDescriptiveTime(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{0, "It is midnight"},
		{6, "It is dawn"},
		{12, "It is noon"},
		{18, "It is dusk"},
		{9, "It is 09:00 in the morning"},
		{15, "It is 15:00 in the afternoon"},
		{21, "It is 21:00 in the evening"},
		{3, "It is 03:00 in the night"},
	}

	for _, tt := range tests {
		gc := &GameClock{currentHour: tt.hour}
		if gc.GetDescriptiveTime() != tt.expected {
			t.Errorf("Hour %d: expected %s, got %s", tt.hour, tt.expected, gc.GetDescriptiveTime())
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	gc := NewGameClock()
	var wg sync.WaitGroup

	// Test concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				gc.GetHour()
				gc.IsDay()
				gc.IsNight()
				gc.GetTimeOfDay()
				gc.GetTimeString()
			}
		}()
	}

	// Concurrent advancement
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				gc.AdvanceHour()
			}
		}()
	}

	wg.Wait()

	// Verify final state is valid (0-23)
	finalHour := gc.GetHour()
	if finalHour < 0 || finalHour >= HoursPerDay {
		t.Errorf("Concurrent access resulted in invalid hour: %d", finalHour)
	}
}
