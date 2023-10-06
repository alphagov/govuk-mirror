package config

import (
	"github.com/caarlos0/env/v9"
)

type Config struct {
	Site      string `env:"SITE"`
	UserAgent string `env:"USER_AGENT" envDefault:"govukbot"`
}

func NewConfig() (*Config, error) {
	cfg := Config{}

	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
