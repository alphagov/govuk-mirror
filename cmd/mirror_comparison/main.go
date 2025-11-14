package main

import (
	"mirrorer/internal/config"
	"mirrorer/internal/logger"

	"github.com/rs/zerolog/log"
)

func main() {
	err := logger.InitialiseLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing log level")
	}

	_, err = config.NewMirrorComparisonConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing config")
	}

	log.Fatal().Msg("Command not yet implemented")
}
