package kitchen 

import (
	"os/signal"
	"syscall"
	"context"
	"flag"
	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/kitchen/aggregator/worker"
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

	if err := worker.Run(); err != nil{
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

}

// validateParams validates params
func validateParams(params *params) error {

}
