package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	tower := flag.String("tower", "human", "Tower ID (e.g., human, elf, dwarf)")
	floors := flag.String("floors", "", "Floor range to generate (e.g., 1-25 or 5)")
	seed := flag.Int64("seed", 42, "Base seed for generation")
	outDir := flag.String("out", "", "Output directory (default: data/towers/{tower}/)")
	flag.Parse()

	if *floors == "" {
		fmt.Fprintln(os.Stderr, "Error: --floors is required (e.g., --floors=1-25 or --floors=5)")
		flag.Usage()
		os.Exit(1)
	}

	// Parse floor range
	startFloor, endFloor, err := parseFloorRange(*floors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid floor range: %v\n", err)
		os.Exit(1)
	}

	// Determine output directory
	outputDir := *outDir
	if outputDir == "" {
		outputDir = fmt.Sprintf("data/towers/%s", *tower)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Create generator
	gen := NewFloorGenerator(*tower, *seed, outputDir)

	// Generate floors
	fmt.Printf("Generating floors %d-%d for tower '%s' (seed: %d)\n", startFloor, endFloor, *tower, *seed)
	fmt.Printf("Output directory: %s\n\n", outputDir)

	for floorNum := startFloor; floorNum <= endFloor; floorNum++ {
		fmt.Printf("Generating floor %d... ", floorNum)
		if err := gen.GenerateFloor(floorNum); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK\n")
	}

	fmt.Printf("\nSuccessfully generated %d floor(s)\n", endFloor-startFloor+1)
}

// parseFloorRange parses a floor range string like "1-25" or "5"
func parseFloorRange(s string) (start, end int, err error) {
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid range format, expected 'start-end'")
		}
		start, err = strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start floor: %w", err)
		}
		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end floor: %w", err)
		}
	} else {
		start, err = strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid floor number: %w", err)
		}
		end = start
	}

	if start < 1 {
		return 0, 0, fmt.Errorf("floor numbers must be >= 1 (floor 0 is the city)")
	}
	if end < start {
		return 0, 0, fmt.Errorf("end floor must be >= start floor")
	}

	return start, end, nil
}
