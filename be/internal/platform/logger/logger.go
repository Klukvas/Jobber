package logger

import (
	"go.uber.org/zap"
)

// Logger wraps zap.Logger
type Logger struct {
	*zap.Logger
}

// New creates a new logger instance
func New(level, format string) (*Logger, error) {
	var cfg zap.Config

	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapLogger, err := cfg.Build(
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, err
	}

	return &Logger{Logger: zapLogger}, nil
}

// WithRequestID adds request_id to the logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String("request_id", requestID)),
	}
}

// WithUserID adds user_id to the logger context
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String("user_id", userID)),
	}
}

// WithAction adds action to the logger context
func (l *Logger) WithAction(action string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String("action", action)),
	}
}

// WithError adds error_code to the logger context
func (l *Logger) WithError(errorCode string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String("error_code", errorCode)),
	}
}

// WithDuration adds duration to the logger context
func (l *Logger) WithDuration(duration int64) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Int64("duration_ms", duration)),
	}
}
