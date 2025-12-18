package logger

import (
	"context"
	"log"
)

// Global logger instance
var globalLogger Logger
var minLogLevel LogLevel = ERROR // Default to ERROR level

// Initialize sets up the global logger
func Initialize(logger Logger) {
	globalLogger = logger
}

// SetLogLevel sets the minimum log level
func SetLogLevel(level LogLevel) {
	minLogLevel = level
}

// shouldLog checks if a message should be logged based on level
func shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
	}
	return levels[level] >= levels[minLogLevel]
}

// GetLogger returns the global logger instance
func GetLogger() Logger {
	if globalLogger == nil {
		// Fallback to standard logger
		return &StandardLogger{}
	}
	return globalLogger
}

// StandardLogger is a fallback logger that uses Go's standard log package
type StandardLogger struct {
	ctx context.Context
}

func (l *StandardLogger) WithContext(ctx context.Context) Logger {
	return &StandardLogger{ctx: ctx}
}

func (l *StandardLogger) Debug(message string, fields ...map[string]interface{}) {
	if !shouldLog(DEBUG) {
		return
	}
	if len(fields) > 0 {
		log.Printf("[DEBUG] %s %v", message, fields)
	} else {
		log.Printf("[DEBUG] %s", message)
	}
}

func (l *StandardLogger) Info(message string, fields ...map[string]interface{}) {
	if !shouldLog(INFO) {
		return
	}
	if len(fields) > 0 {
		log.Printf("[INFO] %s %v", message, fields)
	} else {
		log.Printf("[INFO] %s", message)
	}
}

func (l *StandardLogger) Warn(message string, fields ...map[string]interface{}) {
	if !shouldLog(WARN) {
		return
	}
	if len(fields) > 0 {
		log.Printf("[WARN] %s %v", message, fields)
	} else {
		log.Printf("[WARN] %s", message)
	}
}

func (l *StandardLogger) Error(message string, fields ...map[string]interface{}) {
	if !shouldLog(ERROR) {
		return
	}
	if len(fields) > 0 {
		log.Printf("[ERROR] %s %v", message, fields)
	} else {
		log.Printf("[ERROR] %s", message)
	}
}

// Convenience functions for global logger
func Debug(message string, fields ...map[string]interface{}) {
	GetLogger().Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	GetLogger().Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	GetLogger().Warn(message, fields...)
}

func Error(message string, fields ...map[string]interface{}) {
	GetLogger().Error(message, fields...)
}

func WithContext(ctx context.Context) Logger {
	return GetLogger().WithContext(ctx)
}
