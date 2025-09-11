package main

import (
	"fmt"
	"os"
	"strings"

	"wheres-my-pizza/cmd/kitchenworker"
	"wheres-my-pizza/cmd/notificationsubscriber"
	"wheres-my-pizza/cmd/orderservice"
	"wheres-my-pizza/cmd/trackingservice"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Extract mode from arguments
	var mode string
	var serviceArgs []string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--mode=") {
			mode = strings.TrimPrefix(arg, "--mode=")
		} else if arg == "--mode" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			i++ // skip the next argument
		} else {
			serviceArgs = append(serviceArgs, arg)
		}
	}

	if mode == "" {
		printUsage()
		os.Exit(1)
	}

	// Set the service-specific arguments
	os.Args = append([]string{os.Args[0]}, serviceArgs...)

	switch mode {
	case "order-service":
		orderservice.Main()
	case "kitchen-worker":
		kitchenworker.Main()
	case "tracking-service":
		trackingservice.Main()
	case "notification-subscriber":
		notificationsubscriber.Main()
	default:
		fmt.Printf("Invalid mode: %s\n", mode)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: restaurant-system --mode=<service-mode> [service-specific-flags]")
	fmt.Println("Available modes:")
	fmt.Println("  order-service --port=3000 --max-concurrent=50")
	fmt.Println("  kitchen-worker --worker-name=chef1 --order-types=dine_in,takeout --prefetch=1")
	fmt.Println("  tracking-service --port=3002")
	fmt.Println("  notification-subscriber")
}
