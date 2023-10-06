package main

import (
	"mirrorer/internal/config"
	"mirrorer/internal/crawler"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	initLogger()
	cfg := initConfig()

	cr, err := crawler.NewCrawler(cfg)
	checkError(err, "Error creating new crawler")

	cr.Run()
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

func initConfig() *config.Config {
	cfg, err := config.NewConfig()
	checkError(err, "Error parsing configuration")
	return cfg
}

func checkError(err error, message string) {
	if err != nil {
		log.Fatal().Err(err).Msg(message)
	}
}
