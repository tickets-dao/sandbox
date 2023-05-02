package logging

import (
	"github.com/op/go-logging"
)

func NewHTTPLogger(module string) *logging.Logger {
	lg := &logging.Logger{Module: module}

	lg.SetBackend(NewHTTPBackend())

	return lg
}
