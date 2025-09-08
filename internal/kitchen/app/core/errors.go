package core

import "errors"

var (
	ErrHelp          = errors.New("")
	ErrWorkerStopped = errors.New("worker stopped")

	ErrDBConn  = errors.New("db connection failure")
	ErrRMQConn = errors.New("rabbitmq connection failure")

	ErrMBConn = errors.New("message broker connection failure")
	ErrMBCh   = errors.New("message broker channel failure")
)
