package config

import (
	"reflect"
	"regexp"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	Site                 string            `env:"SITE"`
	AllowedDomains       []string          `env:"ALLOWED_DOMAINS" envSeparator:","`
	UserAgent            string            `env:"USER_AGENT" envDefault:"govukbot"`
	Headers              map[string]string `env:"HEADERS"`
	Concurrency          int               `env:"CONCURRENCY" envDefault:"10"`
	URLFilters           []*regexp.Regexp  `env:"URL_RULES" envSeparator:","`
	DisallowedURLFilters []*regexp.Regexp  `env:"DISALLOWED_URL_RULES" envSeparator:","`
}

func NewConfig() (*Config, error) {
	options := env.Options{FuncMap: map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(regexp.Regexp{}): func(v string) (interface{}, error) {
			return regexp.Compile(v)
		},
	}}

	cfg := Config{}

	err := env.ParseWithOptions(&cfg, options)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
