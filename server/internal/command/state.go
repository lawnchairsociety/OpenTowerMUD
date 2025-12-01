package command

import (
	"fmt"

	"github.com/lawnchairsociety/opentowermud/server/internal/gametime"
)

// executeTime shows the current game time and server uptime
func executeTime(c *Command, p PlayerInterface) string {
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Internal error: invalid server type"
	}

	// Get game clock
	gameClockIface := server.GetGameClock()
	gameClock, ok := gameClockIface.(*gametime.GameClock)
	if !ok {
		return "Internal error: game clock not available"
	}

	timeDesc := gameClock.GetDescriptiveTime()
	timeOfDay := gameClock.GetTimeOfDay()

	// Determine day/night status message
	var periodMsg string
	if gameClock.IsDay() {
		minutesUntilNight := gameClock.GetMinutesUntilNextPeriod()
		periodMsg = fmt.Sprintf("It is daytime. Night falls in %.1f minutes.", minutesUntilNight)
	} else {
		minutesUntilDay := gameClock.GetMinutesUntilNextPeriod()
		periodMsg = fmt.Sprintf("It is nighttime. Dawn breaks in %.1f minutes.", minutesUntilDay)
	}

	uptime := server.GetUptime()
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	return fmt.Sprintf(
		"%s (%s).\n%s\n\nServer uptime: %d hours, %d minutes, %d seconds",
		timeDesc,
		timeOfDay,
		periodMsg,
		hours, minutes, seconds,
	)
}

// executeSleep puts the player to sleep for maximum regeneration
func executeSleep(c *Command, p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already sleeping
	if currentState == "sleeping" {
		return "You are already sleeping."
	}

	// Can't sleep while fighting
	if currentState == "fighting" {
		return "You can't sleep while fighting!"
	}

	// Change to sleeping state
	err := p.SetState("sleeping")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You lie down and fall asleep."
}

// executeWake wakes the player up from sleeping
func executeWake(c *Command, p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already awake
	if currentState != "sleeping" {
		return "You are already awake."
	}

	// Change to standing state
	err := p.SetState("standing")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You wake up and stand."
}

// executeStand makes the player stand up
func executeStand(c *Command, p PlayerInterface) string {
	currentState := p.GetState()

	// Check if already standing
	if currentState == "standing" {
		return "You are already standing."
	}

	// Can't stand while fighting (you're always standing in combat)
	if currentState == "fighting" {
		return "You are already standing (fighting)."
	}

	// Change to standing state
	err := p.SetState("standing")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return "You stand up."
}
