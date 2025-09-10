package kitchen

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/xpkg/logger"
)

type params struct {
	workerParams *core.WorkerParams
	configPath   string
	cfg          *config.Config
}

// Execute starts kitchen service
func Execute(ctx context.Context, mylog logger.Logger, args []string) error {
	newCtx, cancel := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	params, err := parseParams(args)
	if err != nil {
		mylog.Action("comman_parse_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_parse_completed").Debug("Received params", "worker_params", params.workerParams, "config_path", params.configPath)

	if err := validateParams(params); err != nil {
		mylog.Action("command_validation_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_validation_completed").Info("Successfully validate params")

	worker := workers.NewWorker(newCtx, cancel, context.Background(), params.cfg, params.workerParams, mylog)

	if err := worker.Run(); err != nil {
		mylog.Action("consumer_failed").Error("Error running consumer", err)
		return err
	}
	<-newCtx.Done()

	return worker.Stop()
}

// Helper functions to validate cli params

// parseParams parse params from terminal
func parseParams(args []string) (*params, error) {
	fs := flag.NewFlagSet("kitchen-service", flag.ContinueOnError)
	showHelp := fs.Bool("help", false, "Show help")
	configPath := fs.String("config-path", "config.yaml", "path for config yaml")

	workerName := fs.String("worker-name", "", "Max concurrent requests")
	orderTypes := fs.String("order-types", "", "Optional. Comma-separated list of order types the worker can handle (e.g., dine_in, takeout). If omitted, handles all.")
	heartbeatInterval := fs.Int("heartbeat-interval", 30, "Interval (seconds) between heartbeats")
	prefetch := fs.Int("prefetch", 1, "RabbitMQ prefetch count, limiting how many messages the worker receives at once")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("cannot parse: %w", err)
	}

	if *showHelp {
		fs.Usage()
		return nil, core.ErrHelp
	}

	processedOrderTypes := strings.Split(*orderTypes, ",")

	if len(processedOrderTypes) == 1 && processedOrderTypes[0] == "" {
		processedOrderTypes = []string{}
		for orderType := range core.AllowedOrderTypes {
			processedOrderTypes = append(processedOrderTypes, orderType)
		}
		sort.Strings(processedOrderTypes)
	}

	return &params{
		workerParams: &core.WorkerParams{
			WorkerName:        *workerName,
			OrderTypes:        processedOrderTypes,
			HeartbeatInterval: *heartbeatInterval,
			Prefetch:          *prefetch,
		},
		configPath: *configPath,
	}, nil
}

// validateParams validates params
func validateParams(params *params) error {
	cfg, err := config.LoadConfig(params.configPath)
	if err != nil {
		return err
	}
	params.cfg = cfg

	workerParams := params.workerParams

	if workerParams.WorkerName == "" {
		return fmt.Errorf("worker name is required: %s", workerParams.WorkerName)
	}

	if len(workerParams.OrderTypes) > 0 {
		validateOrderTypes := make([]string, 0)
		for _, orderType := range workerParams.OrderTypes {
			orderType = strings.TrimSpace(orderType)
			if !core.AllowedOrderTypes[orderType]{
				return fmt.Errorf("unknown order type: %s", orderType)
			}
			validateOrderTypes = append(validateOrderTypes, orderType)
		}
		workerParams.OrderTypes = validateOrderTypes
	}

	if workerParams.HeartbeatInterval <= {
		return fmt.Errorf("heartbeat-interval cannot be less or equal zero: %d", workerParams.HeartbeatInterval)
	}

	if workerParams.Prefetch <= {
		return fmt.Errorf("prefetch cannot be less or equal zero: %d", workerParams.Prefetch)
	}

	return nil
}
