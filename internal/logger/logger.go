package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global logger instance
	// Use it directly with zerolog's builder pattern:
	//   logger.Logger.Debug().Str("key", "value").Msg("message")
	// Or use the helper functions below for common cases
	Logger zerolog.Logger
)

// Init initializes the global logger with the specified log level
func Init(level string) {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Parse log level
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Configure output format
	// In development, use console writer for better readability
	// In production, use JSON format
	output := os.Getenv("LOG_FORMAT")
	if output == "json" {
		Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// Console writer for development (colored, human-readable)
		Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}).With().Timestamp().Logger()
	}

	// Set as global logger for compatibility
	log.Logger = Logger
}

// parseLogLevel converts string to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}

// Helper functions for common logging patterns

// Err returns a logger event with error attached (most common case)
// Usage: logger.Err(err).Str("key", "value").Msg("message")
func Err(err error) *zerolog.Event {
	return Logger.Error().Err(err)
}

// Debug returns a debug level event
// Usage: logger.Debug().Str("key", "value").Msg("message")
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info returns an info level event
// Usage: logger.Info().Str("key", "value").Msg("message")
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn returns a warn level event
// Usage: logger.Warn().Str("key", "value").Msg("message")
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error returns an error level event
// Usage: logger.Error().Str("key", "value").Msg("message")
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal logs a fatal message and exits
// Usage: logger.Fatal("message")
func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}
