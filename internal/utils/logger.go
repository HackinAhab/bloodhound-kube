package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
	With(kv ...any) Logger
	Component(name string) Logger
}

type logWrapper struct {
	entry *logrus.Entry
}

var defaultLogger Logger = New("info", false)

func New(level string, noColor bool) Logger {
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

	return &logWrapper{entry: logrus.NewEntry(logger)}
}

func DefaultLogger() Logger {
	return defaultLogger
}

func SetDefaultLogger(logger Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

func (l *logWrapper) Debug(msg string, kv ...any) {
	l.logWith(logrus.DebugLevel, msg, kv...)
}

func (l *logWrapper) Info(msg string, kv ...any) {
	l.logWith(logrus.InfoLevel, msg, kv...)
}

func (l *logWrapper) Warn(msg string, kv ...any) {
	l.logWith(logrus.WarnLevel, msg, kv...)
}

func (l *logWrapper) Error(msg string, kv ...any) {
	l.logWith(logrus.ErrorLevel, msg, kv...)
}

func (l *logWrapper) With(kv ...any) Logger {
	entry := l.entry
	fields, errField := parseFields(kv...)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	if errField != nil {
		entry = entry.WithError(errField)
	}

	return &logWrapper{entry: entry}
}

func (l *logWrapper) Component(name string) Logger {
	return l.With("component", name)
}

func (l *logWrapper) logWith(level logrus.Level, msg string, kv ...any) {
	entry := l.entry
	fields, errField := parseFields(kv...)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	if errField != nil {
		entry = entry.WithError(errField)
	}
	entry.Log(level, msg)
}

func parseFields(kv ...any) (logrus.Fields, error) {
	fields := logrus.Fields{}
	var errField error

	if len(kv) == 0 {
		return fields, nil
	}

	if len(kv)%2 != 0 {
		fields["kv_error"] = "odd number of key/value pairs"
		fields["kv_extra"] = kv[len(kv)-1]
		kv = kv[:len(kv)-1]
	}

	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok || key == "" {
			key = fmt.Sprintf("key_%d", i)
		}
		value := kv[i+1]
		if key == "error" {
			if err, ok := value.(error); ok {
				errField = err
				continue
			}
		}
		fields[key] = value
	}

	return fields, errField
}
