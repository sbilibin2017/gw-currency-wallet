package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInitialize_ValidLevels(t *testing.T) {
	// Save original Log and restore after test
	originalLog := Log
	defer func() { Log = originalLog }()

	levels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}

	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			err := Initialize(lvl)
			assert.NoError(t, err, "expected no error for level %s", lvl)
			assert.NotNil(t, Log, "Log should be initialized")
			assert.IsType(t, &zap.SugaredLogger{}, Log, "Log should be a SugaredLogger")

			// Ensure logging works without panic
			assert.NotPanics(t, func() {
				Log.Infow("test log", "level", lvl)
			})
		})
	}
}

func TestInitialize_InvalidLevel(t *testing.T) {
	// Save original Log and restore after test
	originalLog := Log
	defer func() { Log = originalLog }()

	err := Initialize("not-a-level")
	assert.Error(t, err, "expected error for invalid log level")
}

func TestLog_NopBeforeInitialize(t *testing.T) {
	// Save original Log and restore after test
	originalLog := Log
	defer func() { Log = originalLog }()

	// By default, Log is zap.NewNop().Sugar()
	assert.NotNil(t, Log)
	assert.IsType(t, &zap.SugaredLogger{}, Log)

	// Should not panic even if called
	assert.NotPanics(t, func() {
		Log.Infow("nop logger test")
	})
}
