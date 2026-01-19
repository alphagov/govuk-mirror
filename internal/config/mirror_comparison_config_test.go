package config_test

import (
	"mirrorer/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMirrorComparisonConfig_Validate(t *testing.T) {
	t.Run("config is invalid if 'Site' is not a valid URL", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "%%invalid$url",
			CompareTopUnsampledCount:     1,
			CompareRemainingSampledCount: 1,
		}

		assert.Error(t, cfg.Validate())
	})

	t.Run("config is invalid if SlackApiToken is provided but SlackChannelId is not", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackApiToken:                "token",
			SlackChannelId:               "",
		}

		assert.Error(t, cfg.Validate())
	})
}

func TestMirrorComparisonConfig_HasSlackCredentials(t *testing.T) {
	t.Run("true if API token and channel id are set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackApiToken:                "token",
			SlackChannelId:               "channel",
		}

		assert.True(t, cfg.HasSlackCredentials())
	})

	t.Run("false if API token is not set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackApiToken:                "",
			SlackChannelId:               "channel",
		}

		assert.False(t, cfg.HasSlackCredentials())
	})

	t.Run("false if SlackChannelId is not set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackApiToken:                "token",
			SlackChannelId:               "",
		}

		assert.False(t, cfg.HasSlackCredentials())
	})

	t.Run("false if neither token nor channel id are set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackApiToken:                "",
			SlackChannelId:               "",
		}

		assert.False(t, cfg.HasSlackCredentials())
	})
}
