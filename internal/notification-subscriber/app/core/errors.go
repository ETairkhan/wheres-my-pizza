package core

import "errors"

var (
	ErrParseCmd       = errors.New("cannot parse arguments")
	ErrHelp           = errors.New("")
	ErrModeFlag       = errors.New("mode flag is required")
	ErrUnknownService = errors.New("unknown service, write --help command to see valid services")

	ErrRMQConn = errors.New("rabbitmq connection failure")
)
