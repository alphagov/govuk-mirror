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

	t.Run("config is invalid if SlackWebhook is not provided", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackWebhook:                 "",
		}

		assert.Error(t, cfg.Validate())
	})

	t.Run("config is invalid if SlackWebhook is provided but is not a valid URL", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackWebhook:                 "$$invalid%url",
		}

		assert.Error(t, cfg.Validate())
	})
}

func TestMirrorComparisonConfig_HasSlackSettings(t *testing.T) {
	t.Run("true if SlackWebhook is set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackWebhook:                 "webhook",
		}

		assert.True(t, cfg.HasSlackSettings())
	})

	t.Run("false if SlackWebhook is not set", func(t *testing.T) {
		cfg := config.MirrorComparisonConfig{
			Site:                         "https://gov.uk",
			CompareTopUnsampledCount:     0,
			CompareRemainingSampledCount: 0,
			SlackWebhook:                 "",
		}

		assert.False(t, cfg.HasSlackSettings())
	})
}
