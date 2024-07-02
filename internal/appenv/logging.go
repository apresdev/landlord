package appenv

import (
	"go.uber.org/zap"
)

// ConfigureLogging configures Zap logging.
func ConfigureLogging() (func(), error) {
	factory := zap.NewProduction
	if Debug {
		factory = zap.NewDevelopment
	}
	logger, err := factory()
	if err != nil {
		return nil, err
	}
	return zap.ReplaceGlobals(logger), nil
}

// MustConfigureLogging configures Zap logging. It will panic on error.
// The function to undo the logging configuration is returned.
func MustConfigureLogging() func() {
	undo, err := ConfigureLogging()
	if err != nil {
		panic(err)
	}
	return undo
}

func init() {
	MustConfigureLogging()
}
