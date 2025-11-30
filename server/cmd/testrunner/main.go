package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lawnchairsociety/opentowermud/server/test"
)

func main() {
	serverAddr := flag.String("addr", "localhost:4000", "MUD server address")
	verbose := flag.Bool("v", false, "Verbose output - show detailed actions for each test")
	flag.Parse()

	// Set verbose mode
	test.Verbose = *verbose

	fmt.Printf("Running integration tests against %s\n", *serverAddr)
	fmt.Println("Make sure the MUD server is running!")
	if *verbose {
		fmt.Println("Verbose mode enabled - showing detailed test actions")
	}
	fmt.Println()

	results := test.RunAllTests(*serverAddr)
	test.PrintResults(results)

	// Exit with error code if any tests failed
	for _, result := range results {
		if !result.Passed {
			os.Exit(1)
		}
	}
}
