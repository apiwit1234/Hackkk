package utils

import (
	"context"
	"fmt"
	"testing"
	"time"
	"teletubpax-api/errors"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property 13: Retry logic with exponential backoff
// Validates: Requirements 8.4
func TestRetryBackoff_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("retries up to max attempts for retryable errors", prop.ForAll(
		func(maxAttempts int) bool {
			if maxAttempts < 1 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			config := RetryConfig{
				MaxAttempts:       maxAttempts,
				InitialBackoff:    1 * time.Millisecond,
				BackoffMultiplier: 2.0,
				MaxBackoff:        10 * time.Millisecond,
			}

			attemptCount := 0
			operation := func() error {
				attemptCount++
				return errors.NewThrottlingError("throttled", nil)
			}

			ctx := context.Background()
			_ = RetryWithBackoff(ctx, config, operation)

			// Should attempt exactly maxAttempts times
			return attemptCount == maxAttempts
		},
		gen.IntRange(1, 5),
	))

	properties.Property("does not retry non-retryable errors", prop.ForAll(
		func(errorMsg string) bool {
			config := RetryConfig{
				MaxAttempts:       3,
				InitialBackoff:    1 * time.Millisecond,
				BackoffMultiplier: 2.0,
				MaxBackoff:        10 * time.Millisecond,
			}

			attemptCount := 0
			operation := func() error {
				attemptCount++
				return errors.NewValidationError(errorMsg)
			}

			ctx := context.Background()
			_ = RetryWithBackoff(ctx, config, operation)

			// Should only attempt once for validation errors
			return attemptCount == 1
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("stops retrying on success", prop.ForAll(
		func(successAfter int) bool {
			if successAfter < 1 || successAfter > 3 {
				return true // Skip invalid inputs
			}

			config := RetryConfig{
				MaxAttempts:       5,
				InitialBackoff:    1 * time.Millisecond,
				BackoffMultiplier: 2.0,
				MaxBackoff:        10 * time.Millisecond,
			}

			attemptCount := 0
			operation := func() error {
				attemptCount++
				if attemptCount >= successAfter {
					return nil
				}
				return errors.NewThrottlingError("throttled", nil)
			}

			ctx := context.Background()
			err := RetryWithBackoff(ctx, config, operation)

			// Should succeed and attempt exactly successAfter times
			return err == nil && attemptCount == successAfter
		},
		gen.IntRange(1, 3),
	))

	properties.Property("backoff increases exponentially", prop.ForAll(
		func() bool {
			config := RetryConfig{
				MaxAttempts:       3,
				InitialBackoff:    10 * time.Millisecond,
				BackoffMultiplier: 2.0,
				MaxBackoff:        1 * time.Second,
			}

			attemptTimes := []time.Time{}
			operation := func() error {
				attemptTimes = append(attemptTimes, time.Now())
				return errors.NewAWSServiceError("service error", nil)
			}

			ctx := context.Background()
			_ = RetryWithBackoff(ctx, config, operation)

			// Check that delays increase
			if len(attemptTimes) < 2 {
				return false
			}

			// First delay should be approximately initialBackoff
			firstDelay := attemptTimes[1].Sub(attemptTimes[0])
			if firstDelay < config.InitialBackoff {
				return false
			}

			// If there's a third attempt, second delay should be larger
			if len(attemptTimes) >= 3 {
				secondDelay := attemptTimes[2].Sub(attemptTimes[1])
				// Second delay should be at least as long as first delay
				if secondDelay < firstDelay {
					return false
				}
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit tests for retry logic
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "throttling error",
			err:      errors.NewThrottlingError("throttled", nil),
			expected: true,
		},
		{
			name:     "AWS service error",
			err:      errors.NewAWSServiceError("service down", nil),
			expected: true,
		},
		{
			name:     "validation error",
			err:      errors.NewValidationError("invalid input"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      fmt.Errorf("request timeout"),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      fmt.Errorf("ServiceUnavailable"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       5,
		InitialBackoff:    100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.NewThrottlingError("throttled", nil)
	}

	err := RetryWithBackoff(ctx, config, operation)

	// Should return context error
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Should have attempted at least twice
	if attemptCount < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attemptCount)
	}
}
