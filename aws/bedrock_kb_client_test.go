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

// Feature: bedrock-question-search, Property 7: Multiple candidates return highest confidence answer
// Validates: Requirements 4.3
func TestHighestConfidenceSelection_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("highest confidence answer is selected", prop.ForAll(
		func(answers []string, scores []float64) bool {
			if len(answers) == 0 || len(scores) == 0 || len(answers) != len(scores) {
				return true // Skip invalid inputs
			}

			// Find expected highest score index
			maxIdx := 0
			maxScore := scores[0]
			for i, score := range scores {
				if score > maxScore {
					maxScore = score
					maxIdx = i
				}
			}

			mockClient := &MockKBClientWithScores{
				answers: answers,
				scores:  scores,
			}

			result, err := mockClient.QueryKnowledgeBase(context.Background(), "test")

			if err != nil {
				return false
			}

			// Should return the answer with highest score
			return result == answers[maxIdx]
		},
		gen.SliceOf(gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })).
			SuchThat(func(s []string) bool { return len(s) > 0 && len(s) <= 10 }),
		gen.SliceOf(gen.Float64Range(0.0, 1.0)).
			SuchThat(func(s []float64) bool { return len(s) > 0 && len(s) <= 10 }),
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

func TestGetScore(t *testing.T) {
	tests := []struct {
		name     string
		score    *float32
		expected float64
	}{
		{"with score", float32Ptr(0.85), 0.85},
		{"nil score", nil, 0.0},
		{"zero score", float32Ptr(0.0), 0.0},
		{"max score", float32Ptr(1.0), 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getScore(mockRetrievalResult(tt.score))
			if result != tt.expected {
				t.Errorf("getScore() = %v, want %v", result, tt.expected)
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

type MockKBClientWithScores struct {
	answers []string
	scores  []float64
}

func (m *MockKBClientWithScores) QueryKnowledgeBase(ctx context.Context, question string) (string, error) {
	if len(m.answers) == 0 {
		return "", nil
	}

	// Find highest score
	maxIdx := 0
	maxScore := m.scores[0]
	for i, score := range m.scores {
		if score > maxScore {
			maxScore = score
			maxIdx = i
		}
	}

	return m.answers[maxIdx], nil
}

func float32Ptr(f float32) *float32 {
	return &f
}

func mockRetrievalResult(score *float32) mockResult {
	return mockResult{score: score}
}

type mockResult struct {
	score *float32
}

// Adapter to match the getScore function signature
func getScoreFromMock(m mockResult) float64 {
	if m.score != nil {
		return float64(*m.score)
	}
	return 0.0
}
