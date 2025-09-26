package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global SugaredLogger instance.
// Initialized with a no-op logger until Initialize is called.
var Log *zap.SugaredLogger = zap.NewNop().Sugar()

// Initialize sets up the global logger with the given log level.
func Initialize(level string) error {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = logger.Sugar()
	return nil
}
