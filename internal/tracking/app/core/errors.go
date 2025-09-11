package core

import "errors"

var (
	ErrParseCmd        = errors.New("cannot parse arguments")
	ErrHelp            = errors.New("")
	ErrModeFlag        = errors.New("mode flag is required")
	ErrUnknownService  = errors.New("unknown service, write --help command to see valid services")
	ErrNoWorkers       = errors.New("There are no workers")
	ErrOrderNotFound   = errors.New("There are no such order")
	ErrWorkersNotFound = errors.New("There are no workers")

	ErrDBConn = errors.New("db connection failure")
)
