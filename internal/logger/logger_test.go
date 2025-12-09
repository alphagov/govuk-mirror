package logger

import (
	"testing"

	"github.com/ctx42/logkit/pkg/logkit"
	"github.com/rs/zerolog"
)

func Test_Zerolog(t *testing.T) {
	testLogStore := logkit.New(t) // Initialize logkit.

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// Configure zerolog with Tester as the writer.
	testLogger := zerolog.New(testLogStore).With().Timestamp().Logger()

	// Generate some logs...
	testLogger.Info().Msg("Test Message Info")
	testLogger.Warn().Msg("Test Message Warn")
	testLogger.Error().Msg("Test Message Error")

	// Test that the log entries are present...
	entriesSlice := testLogStore.Entries().Get()
	entriesSlice[0].AssertLevel("info")
	entriesSlice[0].AssertMsg("Test Message Info")
	entriesSlice[1].AssertLevel("warn")
	entriesSlice[1].AssertMsg("Test Message Warn")
	entriesSlice[2].AssertLevel("error")
	entriesSlice[2].AssertMsg("Test Message Error")

	t.Log(testLogStore.Entries().Summary())
}
