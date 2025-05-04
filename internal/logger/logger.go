package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init() error {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := parseLogLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	configureLogger(os.Stdout, level)
	return nil
}

func parseLogLevel(logLevel string) (zerolog.Level, error) {
	switch strings.ToLower(logLevel) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", logLevel)
	}
}

func configureLogger(output io.Writer, level zerolog.Level) {
	Log = zerolog.New(output).With().Timestamp().Logger().Level(level)

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "m"

	zerolog.DefaultContextLogger = &Log
}
