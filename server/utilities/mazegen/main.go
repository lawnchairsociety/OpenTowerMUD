package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	size := flag.Int("size", 40, "Maze size (width and height)")
	seed := flag.Int64("seed", 42, "Seed for random generation")
	outDir := flag.String("out", "", "Output directory (default: data/labyrinth/)")
	flag.Parse()

	// Determine output directory
	outputDir := *outDir
	if outputDir == "" {
		outputDir = "data/labyrinth"
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Create generator
	gen := NewMazeGenerator(*size, *size, *seed)

	fmt.Printf("Generating %dx%d labyrinth (seed: %d)\n", *size, *size, *seed)
	fmt.Printf("Output directory: %s\n\n", outputDir)

	// Generate the maze
	fmt.Print("Generating maze structure... ")
	gen.Generate()
	fmt.Println("OK")

	// Place gates at fixed positions for each city
	fmt.Print("Placing city gates... ")
	gen.PlaceGates()
	fmt.Printf("OK (%d gates)\n", len(gen.Gates))

	// Place POIs (treasure vaults, merchants, lore NPCs, shortcuts)
	fmt.Print("Placing points of interest... ")
	gen.PlacePOIs()
	fmt.Printf("OK (%d POIs)\n", gen.POICount)

	// Convert to rooms and write YAML
	fmt.Print("Writing labyrinth.yaml... ")
	if err := gen.WriteYAML(outputDir); err != nil {
		fmt.Printf("FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	// Print summary
	fmt.Printf("\nLabyrinth generated successfully!\n")
	fmt.Printf("  - Total rooms: %d\n", gen.RoomCount())
	fmt.Printf("  - City gates: %d\n", len(gen.Gates))
	fmt.Printf("  - Treasure vaults: %d\n", gen.TreasureCount)
	fmt.Printf("  - Hidden merchants: %d\n", gen.MerchantCount)
	fmt.Printf("  - Lore NPCs: %d\n", gen.LoreNPCCount)
	fmt.Printf("  - Secret shortcuts: %d pairs\n", gen.ShortcutCount)
}
