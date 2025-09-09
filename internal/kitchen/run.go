package kitchen 


type params struct {
	workerParams *core.WorkerParams
	configPath   string
	cfg          *config.Config
}

// Execute starts kitchen service
func Execute(ctx context.Context, mylog logger.Logger, args []string) error {
	newCtx, cancle := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defere cancel()

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
	
}

// validateParams validates params
func validateParams(params *params) error {

}
