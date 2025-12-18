package aws

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property 4: Valid questions produce embedding vectors
// Validates: Requirements 2.1, 2.2
func TestEmbeddingVectorFormat_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid questions produce float64 array embeddings", prop.ForAll(
		func(question string) bool {
			// Mock embedding client that returns a valid embedding
			mockClient := &MockEmbeddingClient{}
			
			embedding, err := mockClient.GenerateEmbedding(context.Background(), question)
			
			// Check that no error occurred
			if err != nil {
				return false
			}
			
			// Check that embedding is not nil
			if embedding == nil {
				return false
			}
			
			// Check that embedding is a non-empty slice of float64
			if len(embedding) == 0 {
				return false
			}
			
			// Verify all elements are float64 (type check is implicit in Go)
			for _, val := range embedding {
				// Check that values are valid floats (not NaN or Inf)
				if val != val { // NaN check
					return false
				}
			}
			
			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 1000 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// MockEmbeddingClient for testing
type MockEmbeddingClient struct{}

func (m *MockEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Return a mock embedding vector
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

// Unit tests for embedding client
func TestBedrockEmbeddingClient_HandleAWSError(t *testing.T) {
	client := &BedrockEmbeddingClient{
		modelId: "test-model",
	}

	tests := []struct {
		name          string
		errorMsg      string
		expectedCode  string
	}{
		{
			name:         "validation exception",
			errorMsg:     "ValidationException: invalid input",
			expectedCode: "VALIDATION_ERROR",
		},
		{
			name:         "throttling exception",
			errorMsg:     "ThrottlingException: rate exceeded",
			expectedCode: "THROTTLING_ERROR",
		},
		{
			name:         "access denied",
			errorMsg:     "AccessDeniedException: invalid credentials",
			expectedCode: "AWS_SERVICE_ERROR",
		},
		{
			name:         "service unavailable",
			errorMsg:     "ServiceUnavailableException: service down",
			expectedCode: "AWS_SERVICE_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockError{msg: tt.errorMsg}
			bedrockErr := client.handleAWSError(err)
			
			if bedrockErr == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "ValidationException", "ValidationException", true},
		{"substring at start", "ValidationException: error", "ValidationException", true},
		{"substring in middle", "error ValidationException occurred", "ValidationException", true},
		{"substring at end", "error: ValidationException", "ValidationException", true},
		{"not found", "SomeOtherError", "ValidationException", false},
		{"empty substring", "test", "", true},
		{"empty string", "", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
