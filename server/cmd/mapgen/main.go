package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Tower YAML structures
type TowerData struct {
	Seed         int64     `yaml:"seed"`
	HighestFloor int       `yaml:"highest_floor"`
	SavedAt      time.Time `yaml:"saved_at"`
	Floors       []Floor   `yaml:"floors"`
}

type Floor struct {
	Number         int       `yaml:"number"`
	StairsUpRoom   string    `yaml:"stairs_up_room"`
	StairsDownRoom string    `yaml:"stairs_down_room"`
	PortalRoom     string    `yaml:"portal_room"`
	GeneratedAt    time.Time `yaml:"generated_at"`
	Rooms          []Room    `yaml:"rooms"`
}

type Room struct {
	ID               string            `yaml:"id"`
	Name             string            `yaml:"name"`
	Description      string            `yaml:"description"`
	DescriptionDay   string            `yaml:"description_day"`
	DescriptionNight string            `yaml:"description_night"`
	Type             string            `yaml:"type"`
	Features         []string          `yaml:"features"`
	Floor            int               `yaml:"floor"`
	Exits            map[string]string `yaml:"exits"`
}

// Grid position for a room
type GridPos struct {
	X, Y int
}

// Room cell for rendering
type RoomCell struct {
	Room      *Room
	Pos       GridPos
	HasNorth  bool
	HasSouth  bool
	HasEast   bool
	HasWest   bool
	HasUp     bool
	HasDown   bool
}

func main() {
	inputFile := flag.String("input", "data/tower.yaml", "Path to tower.yaml file")
	floorNum := flag.Int("floor", -1, "Floor number to display (-1 for all floors)")
	outputFile := flag.String("output", "", "Output file (empty for stdout)")
	showLegend := flag.Bool("legend", true, "Show legend")
	flag.Parse()

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var tower TowerData
	if err := yaml.Unmarshal(data, &tower); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	var output strings.Builder

	output.WriteString(fmt.Sprintf("Tower Map (Seed: %d, Highest Floor: %d)\n", tower.Seed, tower.HighestFloor))
	output.WriteString(fmt.Sprintf("Generated: %s\n", tower.SavedAt.Format("2006-01-02 15:04:05")))
	output.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Sort floors by floor number for consistent display order
	sort.Slice(tower.Floors, func(i, j int) bool {
		return tower.Floors[i].Number < tower.Floors[j].Number
	})

	for _, floor := range tower.Floors {
		if *floorNum >= 0 && floor.Number != *floorNum {
			continue
		}
		renderFloor(&output, &floor)
		output.WriteString("\n")
	}

	if *showLegend {
		output.WriteString(getLegend())
	}

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(output.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Map written to %s\n", *outputFile)
	} else {
		fmt.Print(output.String())
	}
}

func renderFloor(output *strings.Builder, floor *Floor) {
	output.WriteString(fmt.Sprintf("Floor %d", floor.Number))
	if floor.Number == 0 {
		output.WriteString(" (City)")
	}
	output.WriteString("\n")
	output.WriteString(strings.Repeat("-", 40) + "\n")

	if floor.Number == 0 {
		// City floor uses named rooms - render as a graph
		renderCityFloor(output, floor)
	} else {
		// Tower floors use grid-based rooms
		renderGridFloor(output, floor)
	}
}

func renderCityFloor(output *strings.Builder, floor *Floor) {
	// City floors have named rooms that don't follow a strict grid
	// Check connectivity using BFS and report any unreachable rooms

	roomMap := make(map[string]*Room)
	for i := range floor.Rooms {
		roomMap[floor.Rooms[i].ID] = &floor.Rooms[i]
	}

	// BFS from town_square to find all reachable rooms
	visited := make(map[string]bool)
	queue := []string{"town_square"}
	visited["town_square"] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		room, ok := roomMap[current]
		if !ok {
			continue
		}

		for _, targetID := range room.Exits {
			if !visited[targetID] {
				visited[targetID] = true
				queue = append(queue, targetID)
			}
		}
	}

	// Report connectivity
	var unreachable []string
	for _, room := range floor.Rooms {
		if !visited[room.ID] {
			unreachable = append(unreachable, room.ID)
		}
	}

	if len(unreachable) > 0 {
		output.WriteString("WARNING: Unreachable rooms detected!\n")
		for _, id := range unreachable {
			output.WriteString(fmt.Sprintf("  - %s (%s)\n", roomMap[id].Name, id))
		}
		output.WriteString("\n")
	} else {
		output.WriteString(fmt.Sprintf("All %d rooms are connected.\n\n", len(floor.Rooms)))
	}

	// Show the city layout diagram from the YAML comments
	output.WriteString(`City Layout:
                                       [North Gate]
                                            |
 [Castle Gate]------------------------[Town Square]--[Temple]
       |                                    |
 [Guard Post]             [Tavern]--[Market Street]--[General Store]
       |                                    |
 [Castle Hall]         [Barracks]--[Training Hall]--[Armory]
    /      \                                |
[Throne] [Library]                   [Tower Entrance]
    |
[Courtyard]

`)

	// Print room list with exits
	output.WriteString("Room Details:\n")

	var roomIDs []string
	for id := range roomMap {
		roomIDs = append(roomIDs, id)
	}
	sort.Strings(roomIDs)

	for _, id := range roomIDs {
		room := roomMap[id]
		if room == nil {
			continue
		}
		symbol := getRoomSymbol(room, floor)

		// Build exits string
		var exitStrs []string
		for dir, target := range room.Exits {
			exitStrs = append(exitStrs, fmt.Sprintf("%s→%s", dir, target))
		}
		sort.Strings(exitStrs)

		// Build features string
		var features []string
		for _, f := range room.Features {
			features = append(features, f)
		}

		details := fmt.Sprintf("  [%s] %-28s", symbol, truncate(room.Name, 28))
		if len(exitStrs) > 0 {
			details += " exits: " + strings.Join(exitStrs, ", ")
		}
		if len(features) > 0 {
			details += " [" + strings.Join(features, ", ") + "]"
		}

		output.WriteString(details + "\n")
	}
}

func positionTaken(positions map[string]GridPos, pos GridPos) bool {
	for _, p := range positions {
		if p.X == pos.X && p.Y == pos.Y {
			return true
		}
	}
	return false
}

func renderGridFloor(output *strings.Builder, floor *Floor) {
	roomMap := make(map[string]*Room)
	positions := make(map[string]GridPos)

	// Parse grid positions from room IDs (e.g., floor1_4_6 -> x=4, y=6)
	gridRegex := regexp.MustCompile(`floor\d+_(\d+)_(\d+)`)

	for i := range floor.Rooms {
		room := &floor.Rooms[i]
		roomMap[room.ID] = room

		matches := gridRegex.FindStringSubmatch(room.ID)
		if len(matches) == 3 {
			x, _ := strconv.Atoi(matches[1])
			y, _ := strconv.Atoi(matches[2])
			positions[room.ID] = GridPos{X: x, Y: y}
		}
	}

	renderPositionedRooms(output, floor, roomMap, positions)
}

func renderPositionedRooms(output *strings.Builder, floor *Floor, roomMap map[string]*Room, positions map[string]GridPos) {
	if len(positions) == 0 {
		output.WriteString("  (No rooms to display)\n")
		return
	}

	// Find bounds
	minX, maxX, minY, maxY := 999, -999, 999, -999
	for _, pos := range positions {
		if pos.X < minX {
			minX = pos.X
		}
		if pos.X > maxX {
			maxX = pos.X
		}
		if pos.Y < minY {
			minY = pos.Y
		}
		if pos.Y > maxY {
			maxY = pos.Y
		}
	}

	// Create reverse lookup: position -> room ID
	posToRoom := make(map[GridPos]string)
	for id, pos := range positions {
		posToRoom[pos] = id
	}

	// Render the grid
	// Each cell is 5 chars wide, 3 chars tall
	// Format:
	//   |     (north connection)
	// --[R]-- (west-room-east)
	//   |     (south connection)

	for y := minY; y <= maxY; y++ {
		// Top row (north connections)
		for x := minX; x <= maxX; x++ {
			pos := GridPos{X: x, Y: y}
			roomID, hasRoom := posToRoom[pos]
			if hasRoom {
				room := roomMap[roomID]
				if hasExit(room, "north", positions) {
					output.WriteString("  |  ")
				} else {
					output.WriteString("     ")
				}
			} else {
				output.WriteString("     ")
			}
		}
		output.WriteString("\n")

		// Middle row (west-room-east)
		for x := minX; x <= maxX; x++ {
			pos := GridPos{X: x, Y: y}
			roomID, hasRoom := posToRoom[pos]
			if hasRoom {
				room := roomMap[roomID]
				// West connection
				if hasExit(room, "west", positions) {
					output.WriteString("-")
				} else {
					output.WriteString(" ")
				}
				// Room symbol
				output.WriteString("[")
				output.WriteString(getRoomSymbol(room, floor))
				output.WriteString("]")
				// East connection
				if hasExit(room, "east", positions) {
					output.WriteString("-")
				} else {
					output.WriteString(" ")
				}
			} else {
				output.WriteString("     ")
			}
		}
		output.WriteString("\n")

		// Bottom row (south connections)
		for x := minX; x <= maxX; x++ {
			pos := GridPos{X: x, Y: y}
			roomID, hasRoom := posToRoom[pos]
			if hasRoom {
				room := roomMap[roomID]
				if hasExit(room, "south", positions) {
					output.WriteString("  |  ")
				} else {
					output.WriteString("     ")
				}
			} else {
				output.WriteString("     ")
			}
		}
		output.WriteString("\n")
	}

	// Print room list with details
	output.WriteString("\nRoom Details:\n")

	// Sort rooms by ID for consistent output
	var roomIDs []string
	for id := range positions {
		roomIDs = append(roomIDs, id)
	}
	sort.Strings(roomIDs)

	for _, id := range roomIDs {
		room := roomMap[id]
		if room == nil {
			continue
		}
		pos := positions[id]
		symbol := getRoomSymbol(room, floor)

		details := fmt.Sprintf("  [%s] %-25s (%d,%d)", symbol, truncate(room.Name, 25), pos.X, pos.Y)

		// Add special markers
		var markers []string
		if room.ID == floor.StairsUpRoom {
			markers = append(markers, "stairs-up")
		}
		if room.ID == floor.StairsDownRoom {
			markers = append(markers, "stairs-down")
		}
		if room.ID == floor.PortalRoom {
			markers = append(markers, "portal")
		}
		for _, f := range room.Features {
			if f == "treasure" || f == "boss" || f == "altar" || f == "shop" {
				markers = append(markers, f)
			}
		}

		if len(markers) > 0 {
			details += " [" + strings.Join(markers, ", ") + "]"
		}

		output.WriteString(details + "\n")
	}
}

func hasExit(room *Room, direction string, positions map[string]GridPos) bool {
	if room == nil || room.Exits == nil {
		return false
	}
	targetID, ok := room.Exits[direction]
	if !ok {
		return false
	}
	// Check if target exists in our position map
	_, exists := positions[targetID]
	return exists
}

func getRoomSymbol(room *Room, floor *Floor) string {
	// Check for special rooms first
	if room.ID == floor.StairsUpRoom {
		return "^" // Stairs going up
	}
	if room.ID == floor.StairsDownRoom {
		return "v" // Stairs coming down (with portal)
	}
	if room.ID == floor.PortalRoom {
		return "P"
	}

	// Check features
	for _, f := range room.Features {
		switch f {
		case "treasure":
			return "$"
		case "boss":
			return "B"
		case "altar":
			return "A"
		case "portal":
			return "P"
		case "stairs_up":
			return "^"
		case "stairs_down":
			return "v"
		case "shop":
			return "M"
		}
	}

	// Check room type
	switch room.Type {
	case "city":
		return "C"
	case "corridor":
		return "."
	case "room":
		return "#"
	case "stairs":
		return "S"
	case "treasure":
		return "$"
	case "boss":
		return "B"
	case "dead-end":
		return "x"
	default:
		return "?"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getLegend() string {
	return `
Legend:
  [^] Stairs up (to next floor)
  [v] Stairs down (from previous floor) + Portal
  [P] Portal room
  [$] Treasure room
  [B] Boss room
  [A] Altar (respawn point)
  [M] Merchant/Shop
  [C] City room
  [#] Chamber/Room
  [.] Corridor
  [x] Dead-end

  Connections:
  -   Horizontal passage (east-west)
  |   Vertical passage (north-south)
`
}
