package errors

import "errors"

var (
	ErrParseCmd       = errors.New("cannot parse arguments")
	ErrHelp           = errors.New("")
	ErrWorkerStopped  = errors.New("worker stopped")
	ErrModeFlag       = errors.New("mode flag is required")
	ErrUnknownService = errors.New("unknown service, write --help command to see valid services")

	ErrDBConn  = errors.New("db connection failure")
	ErrRMQConn = errors.New("rabbitmq connection failure")

	ErrMBConn = errors.New("message broker connection failure")
	ErrMBCh   = errors.New("message broker channel failure")

	ErrFieldIsEmpty         = errors.New("field is empty")
	ErrMaxConcurentExceeded = errors.New("too many order, try again later")
)
