package log

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetLogConfig(t *testing.T) {
	var buf bytes.Buffer
	SetLogConfig(zerolog.DebugLevel, &buf)
	assert.Equal(t, zerolog.DebugLevel, zlogger.GetLevel(), "Log level should be set to Debug")

	SetLogConfig(zerolog.InfoLevel, &buf)
	assert.Equal(t, zerolog.InfoLevel, zlogger.GetLevel(), "Log level should be set to Info")

	SetLogConfig(zerolog.ErrorLevel, nil)
	assert.Equal(t, zerolog.ErrorLevel, zlogger.GetLevel(), "Log level should be set to Error")

	SetLogConfig(zerolog.FatalLevel, nil)
	assert.Equal(t, zerolog.FatalLevel, zlogger.GetLevel(), "Log level should be set to Fatal")
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	SetLogConfig(zerolog.DebugLevel, &buf)

	Info("Info message")
	assert.Contains(t, buf.String(), "Info message", "Buffer should contain 'Info message'")
	buf.Reset()

	Debug("Debug message")
	assert.Contains(t, buf.String(), "Debug message", "Buffer should contain 'Debug message'")
	buf.Reset()

	Error("Error message")
	assert.Contains(t, buf.String(), "Error message", "Buffer should contain 'Error message'")
	buf.Reset()
}

func TestInitLogLevel(t *testing.T) {
	var buf bytes.Buffer
	SetLogConfig(zerolog.ErrorLevel, &buf)
	assert.Equal(t, zerolog.ErrorLevel, zlogger.GetLevel(), "Initial log level should be Error")
}
