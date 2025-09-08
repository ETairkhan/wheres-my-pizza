package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"wheres-my-pizza/internal/xpkg/errors"
	"wheres-my-pizza/internal/xpkg/logger"
)

func main() {
	logger, err := logger.New("DEBUG")
	if err != nil {
		log.Fatalf("logger error: %v", err)
	}

	logger.Action("restaurant_system_started").Info("Successfully started")
	// Global flags for selecting the service mode
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	mode := fs.String("mode", "", "service to run: kitchen-worker | notification-subscriber-service | order-service | tracking-service")

	// Only parse the first few args for `--mode`, the rest go to the service
	args := os.Args[1:]
	modeArgs := []string{}
	for i, arg := range args {
		if strings.HasPrefix(arg, "--mode") || strings.HasPrefix(arg, "-mode") {
			modeArgs = args[:i+1]
			break
		}
	}
	// parse mode
	if err := fs.Parse(modeArgs); err != nil {
		logger.Action("restaurant_system_failed").Error("Failed to parse flags", err)
		help(fs)
		return
	}

	if *mode == "" {
		logger.Action("restaurant_system_failed").Error("Failed to start restaurant system", errors.ErrModeFlag)
		help(fs)
		return
	}
}

func help(fs *flag.FlagSet) {
	fmt.Println("\nUsage:")
	fs.PrintDefaults()
	fmt.Println("\nExample:")
	fmt.Println("  ./where-is-my-pizza --mode=order-service --port=3000 --max-concurrent=10")
}
