package main

import (
	"log"
	"time"
	"wheres-my-pizza/internal"
)

func main() {
	conn, err := internal.ConnectRabbitMQ("Tair", "secret", "localhost:5672", "customers")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	time.Sleep(10 * time.Second)
	log.Println(client)
}

// // FlagHandler and config structs (same as provided in your last response)
// type ServiceConfig struct {
// 	Mode string
// 	Port int
// }

// type OrderServiceConfig struct {
// 	ServiceConfig
// 	MaxConcurrent int
// }

// type KitchenWorkerConfig struct {
// 	ServiceConfig
// 	WorkerName        string
// 	OrderTypes        []string
// 	HeartbeatInterval int
// 	Prefetch          int
// }

// type TrackingServiceConfig struct {
// 	ServiceConfig
// }

// type NotificationSubscriberConfig struct {
// 	ServiceConfig
// }

// type FlagHandler struct {
// 	mode              *string
// 	port              *int
// 	maxConcurrent     *int
// 	workerName        *string
// 	orderTypes        *string
// 	heartbeatInterval *int
// 	prefetch          *int
// }

// func NewFlagHandler() *FlagHandler {
// 	fh := &FlagHandler{
// 		mode:              flag.String("mode", "", "Service mode: order-service, kitchen-worker, tracking-service, or notification-subscriber (required)"),
// 		port:              flag.Int("port", 3000, "HTTP port for API services (default 3000 for order-service, 3002 for tracking-service)"),
// 		maxConcurrent:     flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process (for order-service)"),
// 		workerName:        flag.String("worker-name", "", "Unique name for the kitchen worker (required for kitchen-worker)"),
// 		orderTypes:        flag.String("order-types", "", "Comma-separated list of order types the worker can handle (optional for kitchen-worker, e.g., 'dine_in,takeout')"),
// 		heartbeatInterval: flag.Int("heartbeat-interval", 30, "Interval (seconds) between heartbeats (for kitchen-worker)"),
// 		prefetch:          flag.Int("prefetch", 1, "RabbitMQ prefetch count, limiting how many messages the worker receives at once (for kitchen-worker)"),
// 	}
// 	flag.Parse()
// 	return fh
// }

// func (fh *FlagHandler) ValidateAndGetConfig() (interface{}, error) {
// 	if *fh.mode == "" {
// 		return nil, fmt.Errorf("mode flag is required")
// 	}
// 	switch *fh.mode {
// 	case "order-service":
// 		if flag.Lookup("port").Value.String() == "3000" {
// 			*fh.port = 3000
// 		}
// 		return &OrderServiceConfig{
// 			ServiceConfig: ServiceConfig{Mode: *fh.mode, Port: *fh.port},
// 			MaxConcurrent: *fh.maxConcurrent,
// 		}, nil
// 	case "kitchen-worker":
// 		if *fh.workerName == "" {
// 			return nil, fmt.Errorf("worker-name flag is required for kitchen-worker")
// 		}
// 		orderTypesList := []string{}
// 		if *fh.orderTypes != "" {
// 			orderTypesList = strings.Split(*fh.orderTypes, ",")
// 			for i, ot := range orderTypesList {
// 				orderTypesList[i] = strings.TrimSpace(ot)
// 			}
// 		}
// 		return &KitchenWorkerConfig{
// 			ServiceConfig:     ServiceConfig{Mode: *fh.mode},
// 			WorkerName:        *fh.workerName,
// 			OrderTypes:        orderTypesList,
// 			HeartbeatInterval: *fh.heartbeatInterval,
// 			Prefetch:          *fh.prefetch,
// 		}, nil
// 	case "tracking-service":
// 		if flag.Lookup("port").Value.String() == "3000" {
// 			*fh.port = 3002
// 		}
// 		return &TrackingServiceConfig{
// 			ServiceConfig: ServiceConfig{Mode: *fh.mode, Port: *fh.port},
// 		}, nil
// 	case "notification-subscriber":
// 		return &NotificationSubscriberConfig{
// 			ServiceConfig: ServiceConfig{Mode: *fh.mode},
// 		}, nil
// 	default:
// 		return nil, fmt.Errorf("invalid mode: %s", *fh.mode)
// 	}
// }

// func main() {
// 	// Parse flags.
// 	fh := NewFlagHandler()
// 	serviceCfg, err := fh.ValidateAndGetConfig()
// 	if err != nil {
// 		log.Fatalf("Flag validation failed: %v", err)
// 	}

// 	// Load configuration.
// 	appCfg, err := config.LoadConfig("config.yaml")
// 	if err != nil {
// 		log.Fatalf("Failed to load config: %v", err)
// 	}

// 	// Initialize logger.
// 	serviceLogger := logger.NewLogger(*fh.mode, "localhost") // Use os.Hostname() in production.

// 	// Set up context for graceful shutdown.
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	// Handle SIGINT/SIGTERM for graceful shutdown.
// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
// 	go func() {
// 		<-sigChan
// 		serviceLogger.Info("graceful_shutdown", "Received shutdown signal", "shutdown-001")
// 		cancel()
// 	}()

// 	// Dispatch to service-specific logic.
// 	switch cfg := serviceCfg.(type) {
// 	case *OrderServiceConfig:
// 		serviceLogger.Info("service_started", "Starting Order Service", "startup-001", "port", cfg.Port, "max_concurrent", cfg.MaxConcurrent)
// 		if err := orders.Run(ctx, appCfg, cfg, serviceLogger); err != nil {
// 			serviceLogger.Error("service_failed", "Order Service failed", "startup-001", err)
// 			os.Exit(1)
// 		}

// 	case *KitchenWorkerConfig:
// 		serviceLogger.Info("service_started", "Starting Kitchen Worker", "startup-001", "worker_name", cfg.WorkerName, "order_types", strings.Join(cfg.OrderTypes, ","), "heartbeat_interval", cfg.HeartbeatInterval, "prefetch", cfg.Prefetch)
// 		if err := kitchen.Run(ctx, appCfg, cfg, serviceLogger); err != nil {
// 			serviceLogger.Error("service_failed", "Kitchen Worker failed", "startup-001", err)
// 			os.Exit(1)
// 		}

// 	case *TrackingServiceConfig:
// 		serviceLogger.Info("service_started", "Starting Tracking Service", "startup-001", "port", cfg.Port)
// 		if err := tracking.Run(ctx, appCfg, cfg, serviceLogger); err != nil {
// 			serviceLogger.Error("service_failed", "Tracking Service failed", "startup-001", err)
// 			os.Exit(1)
// 		}

// 	case *NotificationSubscriberConfig:
// 		serviceLogger.Info("service_started", "Starting Notification Subscriber", "startup-001")
// 		if err := notification.Run(ctx, appCfg, cfg, serviceLogger); err != nil {
// 			serviceLogger.Error("service_failed", "Notification Subscriber failed", "startup-001", err)
// 			os.Exit(1)
// 		}
// 	}
// }
