package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func New(logLevel string) *zerolog.Logger {
	configureLogger(os.Stdout, logLevel)
	return &Log
}

func configureLogger(output io.Writer, logLevel string) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	Log = zerolog.New(output).With().Timestamp().Logger().Level(level)

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"

	zerolog.DefaultContextLogger = &Log
}
