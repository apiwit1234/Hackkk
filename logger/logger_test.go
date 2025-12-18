package logger

import (
	"context"
	"testing"
)

func TestStandardLogger(t *testing.T) {
	logger := &StandardLogger{}

	// Test basic logging methods
	logger.Info("Test info message")
	logger.Debug("Test debug message")
	logger.Warn("Test warn message")
	logger.Error("Test error message")

	// Test with fields
	logger.Info("Test with fields", map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})

	// Test with context
	ctx := context.Background()
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("Test with context")
}

func TestGlobalLogger(t *testing.T) {
	// Initialize with standard logger
	Initialize(&StandardLogger{})

	// Test global functions
	Info("Global info message")
	Debug("Global debug message")
	Warn("Global warn message")
	Error("Global error message")

	// Test with context
	ctx := context.Background()
	ctxLogger := WithContext(ctx)
	ctxLogger.Info("Global logger with context")
}

func TestGetLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	// Should return fallback logger
	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger should never return nil")
	}

	// Initialize and test again
	Initialize(&StandardLogger{})
	logger = GetLogger()
	if logger == nil {
		t.Error("GetLogger should return initialized logger")
	}
}
