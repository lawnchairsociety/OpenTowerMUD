package gametime

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Time constants
	HoursPerDay        = 24
	RealMinutesPerCycle = 60.0
	RealMinutesPerHour  = RealMinutesPerCycle / HoursPerDay // 2.5 minutes

	// Time periods
	DawnHour = 6
	DuskHour = 18
)

type GameClock struct {
	currentHour int
	startTime   time.Time
	mu          sync.RWMutex
}

func NewGameClock() *GameClock {
	return &GameClock{
		currentHour: 0, // Start at midnight
		startTime:   time.Now(),
	}
}

// GetHour returns the current game hour (0-23)
func (gc *GameClock) GetHour() int {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.currentHour
}

// AdvanceHour increments the game hour, wrapping at 24
func (gc *GameClock) AdvanceHour() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.currentHour = (gc.currentHour + 1) % HoursPerDay
}

// IsDay returns true if current hour is during day period (6:00-17:59)
func (gc *GameClock) IsDay() bool {
	hour := gc.GetHour()
	return hour >= DawnHour && hour < DuskHour
}

// IsNight returns true if current hour is during night period (18:00-5:59)
func (gc *GameClock) IsNight() bool {
	return !gc.IsDay()
}

// GetTimeOfDay returns a string describing the current time period
func (gc *GameClock) GetTimeOfDay() string {
	hour := gc.GetHour()

	switch {
	case hour >= 0 && hour < 6:
		return "night"
	case hour >= 6 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 18:
		return "afternoon"
	case hour >= 18 && hour < 24:
		return "evening"
	default:
		return "day"
	}
}

// GetTimeString returns a formatted time string (e.g., "14:00" or "06:00")
func (gc *GameClock) GetTimeString() string {
	return fmt.Sprintf("%02d:00", gc.GetHour())
}

// GetMinutesUntilNextPeriod returns minutes until next day/night transition
func (gc *GameClock) GetMinutesUntilNextPeriod() float64 {
	hour := gc.GetHour()
	var hoursUntilTransition int

	if gc.IsDay() {
		// Currently day, next transition is dusk (18:00)
		hoursUntilTransition = DuskHour - hour
	} else {
		// Currently night, next transition is dawn (6:00)
		if hour >= DuskHour {
			hoursUntilTransition = (HoursPerDay - hour) + DawnHour
		} else {
			hoursUntilTransition = DawnHour - hour
		}
	}

	return float64(hoursUntilTransition) * RealMinutesPerHour
}

// GetDescriptiveTime returns a natural language time description
func (gc *GameClock) GetDescriptiveTime() string {
	hour := gc.GetHour()
	timeOfDay := gc.GetTimeOfDay()

	switch hour {
	case 0:
		return "It is midnight"
	case 6:
		return "It is dawn"
	case 12:
		return "It is noon"
	case 18:
		return "It is dusk"
	default:
		return fmt.Sprintf("It is %s in the %s", gc.GetTimeString(), timeOfDay)
	}
}
