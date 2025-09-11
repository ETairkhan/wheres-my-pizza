package order

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/order/api/http"
	"where-is-my-pizza/internal/order/app/core"
	"where-is-my-pizza/internal/order/config"
)

type params struct {
	orderParams *core.OrderParams
	configPath  string
	cfg         *config.Config
}

// Execute starts order service
func Execute(ctx context.Context, mylog mylogger.Logger, args []string) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	params, err := parseParams(args)
	if err != nil {
		mylog.Action("command_parse_failed").Error("Invalid command received", err)
		return err
	}
	if err = validateParams(params); err != nil {
		mylog.Action("command_validation_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_validation_completed").Info("Successfully validate params")

	server := http.NewServer(newCtx, context.Background(), params.cfg, params.orderParams, mylog)

	// Run server in goroutine
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- server.Run()
	}()

	// Wait for signal or server crash
	select {
	case <-newCtx.Done():
		mylog.Action("shutdown_signal_received").Info("Shutdown signal received")
		return server.Stop(context.Background())
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			mylog.Action("order_service_failed").Error("Server failed unexpectedly", err)
			return err
		}
		mylog.Action("server_stopped").Info("Server exited normally")
		return nil
	}
}

// Helper functions to validate cli params

// parseParams parse params from terminal
func parseParams(args []string) (*params, error) {
	fs := flag.NewFlagSet("order-service", flag.ContinueOnError)
	showHelp := fs.Bool("help", false, "Show help")
	configPath := fs.String("config-path", "config.yaml", "path for config yaml")

	port := fs.Int("port", 3000, "Port to run the order service")
	maxConcurrent := fs.Int("max-concurrent", 50, "Max concurrent requests")

	if err := fs.Parse(args); err != nil {
		// User message
		return nil, errors.New("cannot parse arguments")
	}

	if *showHelp {
		fs.Usage()
		return nil, core.ErrHelp
	}

	return &params{
		orderParams: &core.OrderParams{
			Port:          *port,
			MaxConcurrent: *maxConcurrent,
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

	orderParams := params.orderParams
	if orderParams.Port <= 0 || orderParams.Port >= 65536 {
		return fmt.Errorf("port must be in [0: 65,535]: %d", orderParams.Port)
	}

	if orderParams.MaxConcurrent <= 0 {
		return fmt.Errorf("max number of concurrent tasks cannot be negative: %d", orderParams.MaxConcurrent)
	}

	return nil
}
