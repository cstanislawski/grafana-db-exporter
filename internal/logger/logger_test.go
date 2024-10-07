package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	configureLogger(buf, zerolog.InfoLevel)

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
	levels := []struct {
		level    string
		expected zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
	}

	for _, level := range levels {
		t.Run(level.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			configureLogger(buf, level.expected)

			Log.WithLevel(level.expected).Msg("test message")

			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse log entry: %v", err)
			}

			assertLogField(t, logEntry, "lvl", level.level)
			assertLogField(t, logEntry, "msg", "test message")
			assertTimeField(t, logEntry, "t")
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name          string
		envLogLevel   string
		expectedLevel zerolog.Level
		wantErr       bool
	}{
		{"Default log level", "", zerolog.InfoLevel, false},
		{"Debug log level", "debug", zerolog.DebugLevel, false},
		{"Info log level", "info", zerolog.InfoLevel, false},
		{"Warn log level", "warn", zerolog.WarnLevel, false},
		{"Error log level", "error", zerolog.ErrorLevel, false},
		{"Invalid log level", "invalid", zerolog.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.envLogLevel)
			defer os.Unsetenv("LOG_LEVEL")

			err := Init()
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && Log.GetLevel() != tt.expectedLevel {
				t.Errorf("Init() set log level to %v, want %v", Log.GetLevel(), tt.expectedLevel)
			}
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
