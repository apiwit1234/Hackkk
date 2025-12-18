package errors

import (
	"errors"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property 10: Error responses contain descriptive messages
// Validates: Requirements 5.3
func TestErrorMessagePresence_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("validation errors contain descriptive messages", prop.ForAll(
		func(message string) bool {
			err := NewValidationError(message)
			errorStr := err.Error()
			return len(errorStr) > 0 && errorStr != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("embedding errors contain descriptive messages", prop.ForAll(
		func(message, causeMsg string) bool {
			cause := errors.New(causeMsg)
			err := NewEmbeddingError(message, cause)
			errorStr := err.Error()
			return len(errorStr) > 0 && errorStr != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("knowledge base errors contain descriptive messages", prop.ForAll(
		func(message, causeMsg string) bool {
			cause := errors.New(causeMsg)
			err := NewKnowledgeBaseError(message, cause)
			errorStr := err.Error()
			return len(errorStr) > 0 && errorStr != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("throttling errors contain descriptive messages", prop.ForAll(
		func(message, causeMsg string) bool {
			cause := errors.New(causeMsg)
			err := NewThrottlingError(message, cause)
			errorStr := err.Error()
			return len(errorStr) > 0 && errorStr != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("AWS service errors contain descriptive messages", prop.ForAll(
		func(message, causeMsg string) bool {
			cause := errors.New(causeMsg)
			err := NewAWSServiceError(message, cause)
			errorStr := err.Error()
			return len(errorStr) > 0 && errorStr != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("error messages include error code", prop.ForAll(
		func(message string) bool {
			err := NewValidationError(message)
			errorStr := err.Error()
			return len(errorStr) > 0 && err.Code == ErrCodeValidation
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
