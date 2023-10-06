package config

import (
	"github.com/caarlos0/env/v9"
)

type Config struct {
	Site        string            `env:"SITE"`
	UserAgent   string            `env:"USER_AGENT" envDefault:"govukbot"`
	Headers     map[string]string `env:"HEADERS"`
	Concurrency int               `env:"CONCURRENCY" envDefault:"10"`
}

func NewConfig() (*Config, error) {
	cfg := Config{}

	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
