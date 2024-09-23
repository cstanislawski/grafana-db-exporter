package logger

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	configureLogger(buf)

	Log.Info().Msg("test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	assertLogField(t, logEntry, "lvl", "info")
	assertLogField(t, logEntry, "msg", "test message")
	assertTimeField(t, logEntry, "t")
}

func TestLogLevels(t *testing.T) {
	levels := []zerolog.Level{
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			buf := &bytes.Buffer{}
			configureLogger(buf)
			Log.WithLevel(level).Msg("test message")

			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse log entry: %v", err)
			}

			assertLogField(t, logEntry, "lvl", level.String())
			assertLogField(t, logEntry, "msg", "test message")
			assertTimeField(t, logEntry, "t")
		})
	}
}

func assertLogField(t *testing.T, logEntry map[string]interface{}, key, expectedValue string) {
	value, ok := logEntry[key]
	if !ok {
		t.Errorf("Log entry missing '%s' field", key)
		return
	}
	if value != expectedValue {
		t.Errorf("Expected %s to be '%s', got '%s'", key, expectedValue, value)
	}
}

func assertTimeField(t *testing.T, logEntry map[string]interface{}, key string) {
	value, ok := logEntry[key]
	if !ok {
		t.Errorf("Log entry missing '%s' field", key)
		return
	}
	_, err := time.Parse(time.RFC3339, value.(string))
	if err != nil {
		t.Errorf("Invalid time format for '%s': %v", key, err)
	}
}
