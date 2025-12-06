package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/test"
)

func main() {
	serverAddr := flag.String("addr", "localhost:4000", "MUD server address")
	verbose := flag.Bool("v", false, "Verbose output - show detailed actions for each test")
	filterFlag := flag.String("filter", "", "Run only tests containing this string (case-insensitive)")
	listFlag := flag.Bool("list", false, "List all available tests and exit")
	flag.Parse()

	// Set verbose mode
	test.Verbose = *verbose

	// List tests and exit if requested
	if *listFlag {
		fmt.Println("Available tests:")
		for _, name := range test.GetTestNames() {
			fmt.Printf("  %s\n", name)
		}
		return
	}

	fmt.Printf("Running integration tests against %s\n", *serverAddr)
	fmt.Println("Make sure the MUD server is running!")
	if *verbose {
		fmt.Println("Verbose mode enabled - showing detailed test actions")
	}
	if *filterFlag != "" {
		fmt.Printf("Filter: running tests matching '%s'\n", *filterFlag)
	}
	fmt.Println()

	var results []test.TestResult
	if *filterFlag != "" {
		results = test.RunFilteredTests(*serverAddr, *filterFlag)
	} else {
		results = test.RunAllTests(*serverAddr)
	}

	// Don't print if no tests matched
	if len(results) == 0 {
		fmt.Println("No tests matched the filter.")
		fmt.Println("Use -list to see available tests.")
		os.Exit(1)
	}

	test.PrintResults(results)

	// Exit with error code if any tests failed
	for _, result := range results {
		if !result.Passed {
			os.Exit(1)
		}
	}
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
