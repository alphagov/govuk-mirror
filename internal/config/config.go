package config

import (
	"reflect"
	"regexp"
	"time"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	Site                  string            `env:"SITE"`
	AllowedDomains        []string          `env:"ALLOWED_DOMAINS" envSeparator:","`
	UserAgent             string            `env:"USER_AGENT" envDefault:"govuk-mirror-bot"`
	Headers               map[string]string `env:"HEADERS"`
	Concurrency           int               `env:"CONCURRENCY" envDefault:"10"`
	URLFilters            []*regexp.Regexp  `env:"URL_RULES" envSeparator:","`
	DisallowedURLFilters  []*regexp.Regexp  `env:"DISALLOWED_URL_RULES" envSeparator:","`
	SkipValidation        bool              `env:"SKIP_VALIDATION" envDefault:"false"`
	MetricRefreshInterval time.Duration     `env:"METRIC_REFRESH_INTERVAL" envDefault:"10s"`
	Async                 bool              `env:"ASYNC" envDefault:"true"`
	MirrorFreshnessUrl    string            `env:"MIRROR_FRESHNESS_URL"`
	MirrorAvailabilityUrl string            `env:"MIRROR_AVAILABILITY_URL"`
	Backends              []string          `env:"BACKENDS" envSeparator:","`
	RefreshInterval       time.Duration     `env:"REFRESH_INTERVAL" envDefault:"4h"`
	PushGatewayUrl        string            `env:"PUSHGATEWAY_URL" envDefault:"http://localhost:9091"`
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
