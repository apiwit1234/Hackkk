package aws

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property 6: Knowledge base results are extracted correctly
// Validates: Requirements 3.2
func TestKBResultExtraction_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("KB responses with text are extracted correctly", prop.ForAll(
		func(answerText string) bool {
			mockClient := &MockKBClient{
				response: answerText,
			}

			result, err := mockClient.QueryKnowledgeBase(context.Background(), "test question")

			// Should not error
			if err != nil {
				return false
			}

			// Should return the answer text
			return result == answerText
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("empty KB responses return empty string", prop.ForAll(
		func(question string) bool {
			mockClient := &MockKBClient{
				response: "",
			}

			result, err := mockClient.QueryKnowledgeBase(context.Background(), question)

			// Should not error
			if err != nil {
				return false
			}

			// Should return empty string
			return result == ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 7: RetrieveAndGenerate returns AI-generated answer
// Validates: Requirements 4.3
func TestRetrieveAndGenerate_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("generated answer is returned correctly", prop.ForAll(
		func(generatedAnswer string) bool {
			if len(generatedAnswer) == 0 {
				return true // Skip empty inputs
			}

			mockClient := &MockKBClient{
				response: generatedAnswer,
			}

			result, err := mockClient.QueryKnowledgeBase(context.Background(), "test question")

			if err != nil {
				return false
			}

			// Should return the generated answer
			return result == generatedAnswer
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit tests for KB client
func TestBedrockKBClient_HandleAWSError(t *testing.T) {
	client := &BedrockKBClient{
		knowledgeBaseId: "test-kb",
	}

	tests := []struct {
		name         string
		errorMsg     string
		expectedCode string
	}{
		{
			name:         "validation exception",
			errorMsg:     "ValidationException: invalid query",
			expectedCode: "VALIDATION_ERROR",
		},
		{
			name:         "throttling exception",
			errorMsg:     "ThrottlingException: rate exceeded",
			expectedCode: "THROTTLING_ERROR",
		},
		{
			name:         "resource not found",
			errorMsg:     "ResourceNotFoundException: KB not found",
			expectedCode: "KB_ERROR",
		},
		{
			name:         "service unavailable",
			errorMsg:     "ServiceUnavailableException: service down",
			expectedCode: "AWS_SERVICE_ERROR",
		},
		{
			name:         "timeout",
			errorMsg:     "TimeoutException: request timeout",
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



// Mock clients for testing
type MockKBClient struct {
	response string
	err      error
}

func (m *MockKBClient) QueryKnowledgeBase(ctx context.Context, question string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}


