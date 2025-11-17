package config

import (
	"github.com/caarlos0/env/v9"
)

type MirrorComparisonConfig struct {
	CompareTopUnsampledCount     int `env:"COMPARE_TOP_UNSAMPLED_COUNT" envDefault:"100"`
	CompareRemainingSampledCount int `env:"COMPARE_REMAINING_SAMPLED_COUNT" envDefault:"100"`
}

func NewMirrorComparisonConfig() (*MirrorComparisonConfig, error) {
	cfg := MirrorComparisonConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
