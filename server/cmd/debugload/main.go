package main

import (
	"fmt"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/tower"
)

func main() {
	config, err := npc.LoadMultipleNPCFiles("data/test/npcs_test.yaml", "data/test/mobs_test.yaml")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Loaded %d NPCs\n", len(config.NPCs))

	// Check for test_rat
	if rat, ok := config.NPCs["test_rat"]; ok {
		fmt.Printf("Found test_rat: %s at locations: %v\n", rat.Name, rat.Locations)
	} else {
		fmt.Println("test_rat NOT FOUND")
	}

	// List NPCs by location
	byLocation := config.GetNPCsByLocation()
	fmt.Printf("\nNPCs in training_hall:\n")
	for _, n := range byLocation["training_hall"] {
		fmt.Printf("  - %s (attackable: %v)\n", n.Name, n.Attackable)
	}

	// Check city rooms
	fmt.Println("\n--- Checking city rooms ---")
	cityFloor, err := tower.LoadAndCreateCity("data/cities/human_city.yaml")
	if err != nil {
		fmt.Println("Error loading city:", err)
		return
	}

	rooms := cityFloor.GetRooms()
	fmt.Printf("Loaded %d city rooms\n", len(rooms))
	for _, room := range rooms {
		if room.ID == "training_hall" {
			fmt.Printf("Found training_hall room: %s\n", room.Name)
		}
	}
}
