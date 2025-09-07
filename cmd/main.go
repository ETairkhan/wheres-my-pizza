package main

import (
	"flag"
	"fmt"
	"os"

	"wheres-my-pizza/cmd/kitchen-worker"
	"wheres-my-pizza/cmd/notification-subscriber"
	"wheres-my-pizza/cmd/order-service"
	"wheres-my-pizza/cmd/tracking-service"
)

func main() {
	mode := flag.String("mode", "", "Service mode: order-service, kitchen-worker, tracking-service, notification-subscriber")
	flag.Parse()

	switch *mode {
	case "order-service":
		order_service.Main()
	case "kitchen-worker":
		kitchen_worker.Main()
	case "tracking-service":
		tracking_service.Main()
	case "notification-subscriber":
		notification_subscriber.Main()
	default:
		fmt.Println("Invalid mode. Use --mode=order-service, --mode=kitchen-worker, --mode=tracking-service, or --mode=notification-subscriber")
		os.Exit(1)
	}
}
