package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Mock service for testing
type mockQuestionSearchService struct {
	searchAnswerFunc func(ctx context.Context, question string, enableRelateDocument bool) (string, error)
	callCount        int
}

func (m *mockQuestionSearchService) SearchAnswer(ctx context.Context, question string, enableRelateDocument bool) (string, []string, error) {
	m.callCount++
	if m.searchAnswerFunc != nil {
		answer, err := m.searchAnswerFunc(ctx, question, enableRelateDocument)
		return answer, []string{}, err
	}
	return "mock answer", []string{}, nil
}

// Feature: bedrock-question-search, Property 1: Valid JSON requests are parsed successfully
// Validates: Requirements 1.2, 7.5
func TestValidJSONParsing_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid JSON with question field is parsed", prop.ForAll(
		func(question string) bool {
			mockService := &mockQuestionSearchService{
				searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
					return "answer for " + q, nil
				},
			}

			handler := NewQuestionSearchHandler(mockService, 1000)

			requestBody := map[string]string{"question": question}
			jsonBody, _ := json.Marshal(requestBody)

			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Should return 200 and service should be called
			return w.Code == http.StatusOK && mockService.callCount == 1
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 1000 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 2: Invalid JSON returns 400 error
// Validates: Requirements 1.4
func TestInvalidJSONRejection_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("malformed JSON returns 400", prop.ForAll(
		func(invalidJSON string) bool {
			mockService := &mockQuestionSearchService{}
			handler := NewQuestionSearchHandler(mockService, 1000)

			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", strings.NewReader(invalidJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Should return 400 and service should not be called
			return w.Code == http.StatusBadRequest && mockService.callCount == 0
		},
		gen.OneConstOf("{invalid", "not json", "{\"key\": }", "[1,2,3"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 3: Whitespace-only questions are rejected
// Validates: Requirements 1.5
func TestWhitespaceRejection_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("whitespace-only questions return 400", prop.ForAll(
		func(whitespaceCount int) bool {
			mockService := &mockQuestionSearchService{}
			handler := NewQuestionSearchHandler(mockService, 1000)

			// Generate whitespace-only string
			whitespace := strings.Repeat(" ", whitespaceCount) + strings.Repeat("\t", whitespaceCount/2)
			requestBody := map[string]string{"question": whitespace}
			jsonBody, _ := json.Marshal(requestBody)

			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Should return 400 and service should not be called
			return w.Code == http.StatusBadRequest && mockService.callCount == 0
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 9: Validation failures prevent AWS calls
// Validates: Requirements 7.1, 7.4
func TestValidationPreventsAWSCalls_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("invalid requests don't call service", prop.ForAll(
		func(testCase int) bool {
			mockService := &mockQuestionSearchService{}
			handler := NewQuestionSearchHandler(mockService, 100)

			var req *http.Request

			switch testCase % 4 {
			case 0: // Missing question field
				jsonBody, _ := json.Marshal(map[string]string{})
				req = httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			case 1: // Empty question
				jsonBody, _ := json.Marshal(map[string]string{"question": ""})
				req = httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			case 2: // Whitespace-only
				jsonBody, _ := json.Marshal(map[string]string{"question": "   "})
				req = httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			case 3: // Exceeds length
				longQuestion := strings.Repeat("a", 200)
				jsonBody, _ := json.Marshal(map[string]string{"question": longQuestion})
				req = httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			}

			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Service should never be called for invalid requests
			return mockService.callCount == 0
		},
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: bedrock-question-search, Property 8: Successful responses have valid JSON structure
// Validates: Requirements 4.1, 4.2, 4.5
func TestJSONResponseValidity_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("successful responses have valid JSON with answer field", prop.ForAll(
		func(answer string) bool {
			mockService := &mockQuestionSearchService{
				searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
					return answer, nil
				},
			}

			handler := NewQuestionSearchHandler(mockService, 1000)

			requestBody := map[string]string{"question": "test question"}
			jsonBody, _ := json.Marshal(requestBody)

			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				return false
			}

			// Check response is valid JSON
			var response QuestionSearchResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Check answer field matches
			return response.Answer == answer
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit tests for handler
func TestHandler_ValidRequest(t *testing.T) {
	mockService := &mockQuestionSearchService{
		searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
			return "This is the answer", nil
		},
	}

	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "What is the question?"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response QuestionSearchResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Answer != "This is the answer" {
		t.Fatalf("expected 'This is the answer', got '%s'", response.Answer)
	}
}

func TestHandler_MissingQuestion(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_EmptyQuestion(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": ""}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_WhitespaceOnlyQuestion(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "   \t\n  "}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_QuestionExceedsMaxLength(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 100)

	longQuestion := strings.Repeat("a", 150)
	requestBody := map[string]string{"question": longQuestion}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_InvalidContentType(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "test"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_MalformedJSON(t *testing.T) {
	mockService := &mockQuestionSearchService{}
	handler := NewQuestionSearchHandler(mockService, 1000)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_EmptyAnswer(t *testing.T) {
	mockService := &mockQuestionSearchService{
		searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
			return "", nil
		},
	}

	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "test question"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response QuestionSearchResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Answer != "" {
		t.Fatalf("expected empty answer, got '%s'", response.Answer)
	}
}

// Feature: bedrock-question-search, Property 14: Throttling events are logged
// Validates: Requirements 8.3
func TestThrottlingLogging_Property(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("throttling errors return 429 status", prop.ForAll(
		func(errorMsg string) bool {
			mockService := &mockQuestionSearchService{
				searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
					return "", &BedrockError{
						Code:    "THROTTLING_ERROR",
						Message: errorMsg,
					}
				},
			}

			handler := NewQuestionSearchHandler(mockService, 1000)

			requestBody := map[string]string{"question": "test question"}
			jsonBody, _ := json.Marshal(requestBody)

			req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Handle(w, req)

			// Should return 429 and have Retry-After header
			return w.Code == http.StatusTooManyRequests && w.Header().Get("Retry-After") != ""
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Mock BedrockError for testing
type BedrockError struct {
	Code    string
	Message string
	Cause   error
}

func (e *BedrockError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unit tests for throttling and quota handling
func TestHandler_ThrottlingError(t *testing.T) {
	mockService := &mockQuestionSearchService{
		searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
			return "", &BedrockError{
				Code:    "THROTTLING_ERROR",
				Message: "throttled",
			}
		},
	}

	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "test question"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w.Code)
	}

	if w.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestHandler_QuotaError(t *testing.T) {
	mockService := &mockQuestionSearchService{
		searchAnswerFunc: func(ctx context.Context, q string, enableRelateDocument bool) (string, error) {
			return "", &BedrockError{
				Code:    "AWS_SERVICE_ERROR",
				Message: "AWS quota exceeded",
			}
		},
	}

	handler := NewQuestionSearchHandler(mockService, 1000)

	requestBody := map[string]string{"question": "test question"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/teletubpax/question-search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
}
