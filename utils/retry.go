package utils

import (
	"context"
	"log"
	"time"
	"teletubpax-api/errors"
)

type RetryConfig struct {
	MaxAttempts     int
	InitialBackoff  time.Duration
	BackoffMultiplier float64
	MaxBackoff      time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        2 * time.Second,
	}
}

func RetryWithBackoff(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		lastErr = operation()

		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if !isRetryable(lastErr) {
			return lastErr
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Log retry attempt
		log.Printf("Retry attempt %d/%d after error: %v. Waiting %v before retry", 
			attempt, config.MaxAttempts, lastErr, backoff)

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Calculate next backoff duration
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	log.Printf("All %d retry attempts exhausted. Last error: %v", config.MaxAttempts, lastErr)
	return lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a BedrockError
	if bedrockErr, ok := err.(*errors.BedrockError); ok {
		switch bedrockErr.Code {
		case errors.ErrCodeThrottling:
			return true
		case errors.ErrCodeAWSService:
			// Retry AWS service errors (timeouts, 5xx errors)
			return true
		case errors.ErrCodeValidation:
			// Don't retry validation errors
			return false
		case errors.ErrCodeEmbedding, errors.ErrCodeKnowledgeBase:
			// Retry if the underlying cause is retryable
			if bedrockErr.Cause != nil {
				return isRetryable(bedrockErr.Cause)
			}
			return false
		}
	}

	// Check error message for common retryable patterns
	errMsg := err.Error()
	retryablePatterns := []string{
		"timeout",
		"Timeout",
		"ServiceUnavailable",
		"InternalServer",
		"TooManyRequests",
		"Throttling",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
