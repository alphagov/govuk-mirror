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
}
