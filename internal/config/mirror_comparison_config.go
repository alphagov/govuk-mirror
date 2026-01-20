package config

import (
	"errors"
	"net/url"
	"strings"

	"github.com/caarlos0/env/v9"
)

type MirrorComparisonConfig struct {
	Site                         string `env:"SITE"`
	CompareTopUnsampledCount     int    `env:"COMPARE_TOP_UNSAMPLED_COUNT" envDefault:"100"`
	CompareRemainingSampledCount int    `env:"COMPARE_REMAINING_SAMPLED_COUNT" envDefault:"100"`
	SlackWebhook                 string `env:"SLACK_WEBHOOK" envDefault:""`
}

func NewMirrorComparisonConfig() (*MirrorComparisonConfig, error) {
	cfg := MirrorComparisonConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (mcc *MirrorComparisonConfig) Validate() error {
	if strings.TrimSpace(mcc.Site) == "" {
		return errors.New("site is required")
	}

	_, err := url.Parse(mcc.Site)
	if err != nil {
		return err
	}

	if strings.TrimSpace(mcc.SlackWebhook) == "" {
		return errors.New("slack webhook is required")
	}

	_, err = url.Parse(mcc.SlackWebhook)
	if err != nil {
		return errors.New("slack webhook must be a valid URL")
	}

	return nil
}

func (mcc *MirrorComparisonConfig) HasSlackSettings() bool {
	if strings.TrimSpace(mcc.SlackWebhook) == "" {
		return false
	}
	
	_, err := url.Parse(mcc.SlackWebhook)
	return err == nil
}

func (mcc *MirrorComparisonConfig) SlackWebhookURL() url.URL {
	u, _ := url.Parse(mcc.SlackWebhook)
	return *u
}
