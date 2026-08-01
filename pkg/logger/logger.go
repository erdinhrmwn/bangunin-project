// Package logger provides a process-wide structured logger built on zerolog.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New builds a zerolog.Logger writing JSON to stdout. In "development" env it
// switches to zerolog's human-readable console writer. level accepts any
// zerolog.ParseLevel value (debug, info, warn, error); invalid values fall
// back to info.
func New(env, level, service string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	logger := zerolog.New(os.Stdout)
	if env == "development" {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}

	return logger.Level(lvl).With().
		Timestamp().
		Str("service", service).
		Str("env", env).
		Logger()
}
