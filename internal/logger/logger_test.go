package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger_ValidLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			l, err := New(lvl)
			assert.NoError(t, err, "expected no error for valid level")
			assert.NotNil(t, l, "logger should not be nil")
		})
	}
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	l, err := New("invalid-level")
	assert.Error(t, err, "expected error for invalid level")
	assert.Nil(t, l, "logger should be nil on error")
}
