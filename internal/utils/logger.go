package utils

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func New(level string, noColor bool) *logrus.Logger {
	logger := logrus.New()

	parsedLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = logrus.InfoLevel
	}
	logger.SetLevel(parsedLevel)
	logger.SetOutput(os.Stderr)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		DisableColors:   noColor,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}
