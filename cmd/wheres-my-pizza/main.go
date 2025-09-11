package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"wheres-my-pizza/internal/kitchen"
	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/notsub"
	"wheres-my-pizza/internal/order"
	"wheres-my-pizza/internal/order/app/core"
	"wheres-my-pizza/internal/tracking"
)

func main() {
	mylogger, err := logger.New("DEBUG")
	if err != nil {
		log.Fatalf("log error: %v", err)
	}

	mylogger.Action("restaurant_system_started").Info("Successfully started")
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
		mylogger.Action("restaurant_system_failed").Error("Failed to parse flags", err)
		help(fs)
		return
	}

	if *mode == "" {
		mylogger.Action("restaurant_system_failed").Error("Failed to start restaurant system", core.ErrModeFlag)
		help(fs)
		return
	}

	// Remaining args after parsing --mode
	remainingArgs := args[len(modeArgs):]

	ctx := context.Background()
	switch *mode {
	case "order-service", "os":
		l := mylogger.With("service", "order-service")
		l.Action("order_service_started").Info("Successfully started")
		if err := order.Execute(ctx, l, remainingArgs); err != nil {
			l.Action("order_servicle_failed").Error("Error in order-service", err)
			if !errors.Is(err, core.ErrHelp) {
				log.Fatalf("failed to execute order-service: %s", err)
			}
		}
		l.Action("order_service_completed").Info("Successfully completed")

	case "kitchen-worker", "kw":
		l := mylogger.With("service", "kitchen-worker")

		l.Action("kitchen_service_started").Info("Successfully started")
		if err := kitchen.Execute(ctx, l, remainingArgs); err != nil {
			l.Action("kitchen_service_failed").Error("Error in kitchen-service", err)

			if !errors.Is(err, core.ErrHelp) {
				log.Fatalf("failed to execute kitchen-service: %s", err)
			}
		}
		l.Action("kitchen_service_completed").Info("Successfully completed")

	case "notification-subscriber", "ns":
		l := mylogger.With("service", "notification-subscriber-service")

		l.Action("notification_subscriber_service_started").Info("Successfully started")
		if err := notsub.Execute(ctx, l, remainingArgs); err != nil {
			l.Action("notification_subscriber_service_failed").Error("Error in notification-subscriber-service", err)

			if !errors.Is(err, core.ErrHelp) {
				log.Fatalf("failed to execute notification-subscriber-service: %s", err)
			}
		}
		l.Action("notification_subscriber_service_completed").Info("Successfully completed")

	case "tracking-service", "ts":
		l := mylogger.With("service", "tracking-service")

		l.Action("tracking_service_started").Info("Successfully started")
		if err := tracking.Execute(ctx, l, remainingArgs); err != nil {
			l.Action("tracking_service_failed").Error("Error in tracking-service", err)

			if !errors.Is(err, core.ErrHelp) {
				log.Fatalf("failed to execute tracking-service: %s", err)
			}
		}
		l.Action("tracking_service_completed").Info("Successfully completed")

	default:
		mylogger.Action("restaurant_system_failed").Error("Failed to start restaurant system", core.ErrUnknownService)
		help(fs)
	}
}

func help(fs *flag.FlagSet) {
	fmt.Println("\nUsage:")
	fs.PrintDefaults()
	fmt.Println("\nExample:")
	fmt.Println("  ./where-is-my-pizza --mode=order-service --port=3000 --max-concurrent=10")
}
