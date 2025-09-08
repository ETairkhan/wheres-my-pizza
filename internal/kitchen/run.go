package kitchen 


type params struct {
	workerParams *core.WorkerParams
	configPath   string
	cfg          *config.Config
}

// Execute starts kitchen service
func Execute(ctx context.Context, mylog mylogger.Logger, args []string) error {

}

// Helper functions to validate cli params

// parseParams parse params from terminal
func parseParams(args []string) (*params, error) {
	
}

// validateParams validates params
func validateParams(params *params) error {

}
