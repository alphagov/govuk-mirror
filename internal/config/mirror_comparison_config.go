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
	SlackApiToken                string `env:"SLACK_API_TOKEN" envDefault:""`
	SlackChannelId               string `env:"SLACK_CHANNEL_ID" envDefault:""`
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
	_, err := url.Parse(mcc.Site)
	if err != nil {
		return err
	}

	if (strings.TrimSpace(mcc.SlackApiToken) != "" && strings.TrimSpace(mcc.SlackChannelId) == "") ||
		(strings.TrimSpace(mcc.SlackApiToken) == "" && strings.TrimSpace(mcc.SlackChannelId) != "") {
		return errors.New("the Slack API token and Slack channel id must be provided together, or not at all")
	}

	return nil
}

func (mcc *MirrorComparisonConfig) HasSlackCredentials() bool {
	return strings.TrimSpace(mcc.SlackApiToken) != "" && strings.TrimSpace(mcc.SlackChannelId) != ""
}
