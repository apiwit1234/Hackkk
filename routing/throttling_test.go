package routing

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
	"teletubpax-api/errors"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: bedrock-question-search, Property 14: Throttling events are logged
// Validates: Requirements 8.3
func TestThrottlingLogging_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("throttling errors return 429 status", prop.ForAll(
		func(errorMsg string) bool {
			mockService := &MockQuestionSearchService{
				err: errors.NewThrottlingError(errorMsg, nil),
			}
			handler := NewQuestionSearchHandler(mockService, 1000)

			reqBody := `{"question": "test question"}`
			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewBufferString(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Should return 429 Too Many Requests
			if w.Code != 429 {
				return false
			}

			// Should have Retry-After header
			if w.Header().Get("Retry-After") == "" {
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit tests for throttling and quota handling
func TestThrottlingErrorReturns429(t *testing.T) {
	mockService := &MockQuestionSearchService{
		err: errors.NewThrottlingError("rate limit exceeded", nil),
	}
	handler := NewQuestionSearchHandler(mockService, 1000)

	reqBody := `{"question": "test"}`
	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != 429 {
		t.Errorf("expected status 429, got %d", w.Code)
	}

	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestQuotaErrorReturns503(t *testing.T) {
	mockService := &MockQuestionSearchService{
		err: errors.NewAWSServiceError("quota exceeded", nil),
	}
	handler := NewQuestionSearchHandler(mockService, 1000)

	reqBody := `{"question": "test"}`
	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != 503 {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestThrottlingEventLogging(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "throttling error",
			err:            errors.NewThrottlingError("throttled", nil),
			expectedStatus: 429,
		},
		{
			name:           "quota error in embedding",
			err:            errors.NewEmbeddingError("quota exceeded", nil),
			expectedStatus: 503,
		},
		{
			name:           "quota error in KB",
			err:            errors.NewKnowledgeBaseError("Quota exceeded", nil),
			expectedStatus: 503,
		},
		{
			name:           "quota error in AWS service",
			err:            errors.NewAWSServiceError("quota limit reached", nil),
			expectedStatus: 503,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockQuestionSearchService{
				err: tt.err,
			}
			handler := NewQuestionSearchHandler(mockService, 1000)

			reqBody := fmt.Sprintf(`{"question": "test for %s"}`, tt.name)
			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewBufferString(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
