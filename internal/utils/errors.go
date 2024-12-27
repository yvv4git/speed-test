package utils

import "errors"

var (
	ErrNoLoggerProvided      = errors.New("no logger provided")
	ErrNoApplicationProvided = errors.New("no application provided")
	ErrGracefulShutdown      = errors.New("graceful shutdown")
)
