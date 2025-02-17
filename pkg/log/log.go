package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

var (
	logger  Logger
	zlogger zerolog.Logger
)

// Logger is the interface that wraps the basic logging methods
type Logger interface {
	Info(msg string)
	Debug(msg string)
	Error(msg string)
}

// Log is the struct that implements the Logger interface
type Log struct {
	log zerolog.Logger
}

// SetDefaultLogConfig sets the default logger configuration
func SetDefaultLogConfig() {
	SetLogConfig(zerolog.InfoLevel, os.Stdout)
}

// SetLogConfig sets the logger configuration
// level is the logging level
// output is the output writer
func SetLogConfig(level zerolog.Level, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	zlogger = zerolog.New(output).Level(level).With().Timestamp().Logger()
	logger = &Log{log: zlogger}
}

// Info logs message with INFO level
func (l *Log) Info(msg string) {
	l.log.Info().Msg(msg)
}

// Debug logs message with DEBUG level
func (l *Log) Debug(msg string) {
	l.log.Debug().Msg(msg)
}

// Error logs message with ERROR level
func (l *Log) Error(msg string) {
	l.log.Error().Msg(msg)
}

// Info logs an info message
// msg is the message to log
func Info(msg string) {
	if logger != nil {
		logger.Info(msg)
	}
}

// Debug logs a debug message
// msg is the message to log
func Debug(msg string) {
	if logger != nil {
		logger.Debug(msg)
	}
}

// Error logs an error message
// msg is the message to log
func Error(msg string) {
	if logger != nil {
		logger.Error(msg)
	}
}
