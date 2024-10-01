package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func New() *zerolog.Logger {
	return &Log
}

func init() {
	configureLogger(os.Stdout)
}

func configureLogger(output io.Writer) {
	Log = zerolog.New(output).With().Timestamp().Logger()

	zerolog.TimeFieldFormat = time.RFC3339

	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"

	zerolog.DefaultContextLogger = &Log
}
