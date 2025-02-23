/*
Package tlog is a custom log package which uses github.com/lmittmann/tint.
*/
package tlog

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
)

func setLogLevel(level string) slog.Level {
	if level == "" {
		level = strings.ToLower(os.Getenv("LOG_LEVEL"))
	}

	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New instantiates custom logger.
func New(level string, colorize bool) *slog.Logger {
	if os.Getenv("LOG_COLORIZE") != "" {
		colorize = true
	}

	opts := &tint.Options{
		Level:      setLogLevel(level),
		TimeFormat: "15:04:05",
		NoColor:    !colorize,
	}

	return slog.New(
		tint.NewHandler(os.Stderr, opts),
	)
}
