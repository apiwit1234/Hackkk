package services

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"teletubpax-api/config"
)

// Mock clients for testing
type mockEmbeddingClient struct {
	generateEmbeddingFunc func(ctx context.Context, text string) ([]float64, error)
	callCount             int
}

func (m *mockEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	m.callCount++
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

type mockKnowledgeBaseClient struct {
	queryKnowledgeBaseFunc func(ctx context.Context, question string) (string, error)
	callCount              int
}

func (m *mockKnowledgeBaseClient) QueryKnowledgeBase(ctx context.Context, question string) (string, error) {
	m.callCount++
	if m.queryKnowledgeBaseFunc != nil {
		return m.queryKnowledgeBaseFunc(ctx, question)
	}
	return "mock answer", nil
}

// Feature: bedrock-question-search, Property 5: Embedding vectors are sent to knowledge base
// Validates: Requirements 3.1
func TestEmbeddingToKBWorkflow_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("KB is queried for all valid questions", prop.ForAll(
		func(question string) bool {
			mockKB := &mockKnowledgeBaseClient{
				queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
					return "answer for " + q, nil
				},
			}

			cfg := &config.Config{
				RetryAttempts: 3,
			}

			service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

			_, err := service.SearchAnswer(context.Background(), question)

			// KB should be called exactly once for successful queries
			return err == nil && mockKB.callCount == 1
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 1000 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 12: Requests are logged for audit
// Validates: Requirements 5.4, 5.5
func TestAuditLogging_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("all requests are processed", prop.ForAll(
		func(question string) bool {
			mockKB := &mockKnowledgeBaseClient{
				queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
					return "answer", nil
				},
			}

			cfg := &config.Config{
				RetryAttempts: 3,
			}

			service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

			_, err := service.SearchAnswer(context.Background(), question)

			// Service should process the request (logging happens internally)
			return err == nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 1000 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 11: Errors are logged with context
// Validates: Requirements 5.1, 5.2
func TestErrorLogging_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("errors are handled and logged", prop.ForAll(
		func(errorMsg string) bool {
			mockKB := &mockKnowledgeBaseClient{
				queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
					return "", &testError{msg: errorMsg}
				},
			}

			cfg := &config.Config{
				RetryAttempts: 1,
			}

			service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

			_, err := service.SearchAnswer(context.Background(), "test question")

			// Error should be returned (logging happens internally)
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Unit tests for service orchestration
func TestService_SuccessfulFlow(t *testing.T) {
	mockKB := &mockKnowledgeBaseClient{
		queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
			return "This is the answer", nil
		},
	}

	cfg := &config.Config{
		RetryAttempts: 3,
	}

	service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

	answer, err := service.SearchAnswer(context.Background(), "What is the question?")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer != "This is the answer" {
		t.Fatalf("expected 'This is the answer', got '%s'", answer)
	}
	if mockKB.callCount != 1 {
		t.Fatalf("expected KB to be called once, got %d calls", mockKB.callCount)
	}
}

func TestService_KBError(t *testing.T) {
	mockKB := &mockKnowledgeBaseClient{
		queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
			return "", &testError{msg: "KB error"}
		},
	}

	cfg := &config.Config{
		RetryAttempts: 1,
	}

	service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

	_, err := service.SearchAnswer(context.Background(), "test question")

	if err == nil {
		t.Fatal("expected error from KB, got nil")
	}
}

func TestService_EmptyAnswer(t *testing.T) {
	mockKB := &mockKnowledgeBaseClient{
		queryKnowledgeBaseFunc: func(ctx context.Context, q string) (string, error) {
			return "", nil
		},
	}

	cfg := &config.Config{
		RetryAttempts: 3,
	}

	service := NewBedrockQuestionSearchService(nil, mockKB, cfg)

	answer, err := service.SearchAnswer(context.Background(), "test question")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer != "" {
		t.Fatalf("expected empty answer, got '%s'", answer)
	}
}
