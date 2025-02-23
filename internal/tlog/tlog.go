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
func New(level string, noColor bool) *slog.Logger {
	if os.Getenv("LOG_COLORIZE") != "" {
		noColor = false
	}

	opts := &tint.Options{
		Level:      setLogLevel(level),
		TimeFormat: "15:04:05",
		NoColor:    noColor,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if noColor {
				return a
			}

			if a.Key == slog.LevelKey {
				switch a.Value.Any().(slog.Level) {
				case slog.LevelDebug:
					return slog.String(a.Key, "\033[38;5;35mDBG\033[0m")
				case slog.LevelInfo:
					return slog.String(a.Key, "\033[38;5;75mINF\033[0m")
				case slog.LevelWarn:
					return slog.String(a.Key, "\033[38;5;220mWRN\033[0m")
				case slog.LevelError:
					return slog.String(a.Key, "\033[38;5;196mERR\033[0m")
				}
			}
			return a
		},
	}

	return slog.New(
		tint.NewHandler(os.Stderr, opts),
	)
}
