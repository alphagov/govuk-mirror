package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	initLogger()
}

func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "INFO"
	}
	level, err := zerolog.ParseLevel(logLevel)
	checkError(err, "Error parsing log level")
	zerolog.SetGlobalLevel(level)
}

func checkError(err error, message string) {
	if err != nil {
		log.Fatal().Err(err).Msg(message)
	}
}
