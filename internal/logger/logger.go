package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitialiseLogger() error {
	// Parse log level BEFORE creating the logger
	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "INFO"
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	// Set global level first
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// Create logger with timestamp after level is set
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	return nil
}
