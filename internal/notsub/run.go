package notsub

import (
	"context"
	"errors"
	"flag"
	"os/signal"
	"syscall"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/notsub/adapter/consumer"
	"where-is-my-pizza/internal/notsub/app/core"
	"where-is-my-pizza/internal/notsub/config"
)

type params struct {
	configPath string
	cfg        *config.Config
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
	mylog.Action("command_parse_completed").Debug("Received params", "config_path", params.configPath)

	if err = validateParams(params); err != nil {
		mylog.Action("command_validation_failed").Error("Invalid command received", err)
		return err
	}
	mylog.Action("command_validation_completed").Info("Successfully validate params")

	notsub := consumer.NewNotification(newCtx, context.Background(), params.cfg, mylog)

	if err := notsub.Run(); err != nil {
		mylog.Action("notsub_run_failed").Error("Notification subscriber service stopped with error", err)
		return err
	}
	<-newCtx.Done()
	return notsub.Stop(ctx)
}

// Helper functions to validate cli params

// parseParams parse params from terminal
func parseParams(args []string) (*params, error) {
	fs := flag.NewFlagSet("notification-subscriber-service", flag.ContinueOnError)
	showHelp := fs.Bool("help", false, "Show help")
	configPath := fs.String("config-path", "config.yaml", "path for config yaml")

	if err := fs.Parse(args); err != nil {
		// User message
		return nil, errors.New("cannot parse arguments")
	}

	if *showHelp {
		fs.Usage()
		return nil, core.ErrHelp
	}

	return &params{
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
	return nil
}
