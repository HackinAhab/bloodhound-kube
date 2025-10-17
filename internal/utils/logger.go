package utils

import (
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	*slog.Logger
	level slog.Level
}

func New(level string) *Logger {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format for better readability
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
				}
			}
			return a
		},
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		level:  logLevel,
	}
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() slog.Level {
	return l.level
}

// IsDebugEnabled checks if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= slog.LevelDebug
}

// IsInfoEnabled checks if info logging is enabled
func (l *Logger) IsInfoEnabled() bool {
	return l.level <= slog.LevelInfo
}
