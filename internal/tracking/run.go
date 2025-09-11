package tracking

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/tracking/api/http"
	"wheres-my-pizza/internal/tracking/app/core"
	"wheres-my-pizza/internal/xpkg/config"
)

type params struct {
	trackingParams *core.TrackingParams
	configPath     string
	cfg            *config.Config
}

// Run starts order service
func Execute(ctx context.Context, mylog logger.Logger, args []string) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	params, err := parseParams(args)
	if err != nil {
		mylog.Action("command_parse_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_parse_completed").Debug("Received params", "tracking_params", params.trackingParams, "config_path", params.configPath)

	if err = validateParams(params); err != nil {
		mylog.Action("command_validation_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_validation_completed").Info("Successfully validate params")

	server := http.NewServer(newCtx, ctx, params.cfg, params.trackingParams, mylog)

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
	fs := flag.NewFlagSet("tracking-service", flag.ContinueOnError)
	showHelp := fs.Bool("help", false, "Show help")
	configPath := fs.String("config-path", "config.yaml", "path for config yaml")

	port := fs.Int("port", 3002, "Port to run the tracking service")

	if err := fs.Parse(args); err != nil {
		// User message
		return nil, errors.New("cannot parse arguments")
	}

	if *showHelp {
		fs.Usage()
		return nil, core.ErrHelp
	}

	return &params{
		trackingParams: &core.TrackingParams{
			Port: *port,
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

	notsubParams := params.trackingParams

	if notsubParams.Port <= 0 || notsubParams.Port >= 65536 {
		return fmt.Errorf("port must be in [0: 65,535]: %d", notsubParams.Port)
	}

	return nil
}
