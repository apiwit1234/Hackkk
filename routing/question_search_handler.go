package routing

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	bedrockErrors "teletubpax-api/errors"
	"teletubpax-api/services"
)

type QuestionSearchRequest struct {
	Question string `json:"question"`
}

type QuestionSearchResponse struct {
	Answer string `json:"answer"`
}

type QuestionSearchHandler struct {
	service           services.QuestionSearchService
	maxQuestionLength int
}

func NewQuestionSearchHandler(service services.QuestionSearchService, maxQuestionLength int) *QuestionSearchHandler {
	return &QuestionSearchHandler{
		service:           service,
		maxQuestionLength: maxQuestionLength,
	}
}

func (h *QuestionSearchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "" {
		BadRequestHandler(w, "Content-Type must be application/json")
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		BadRequestHandler(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var request QuestionSearchRequest
	if err := json.Unmarshal(body, &request); err != nil {
		BadRequestHandler(w, "Invalid JSON format")
		return
	}

	// Validate question field presence
	if request.Question == "" {
		BadRequestHandler(w, "Question field is required")
		return
	}

	// Validate question is not whitespace-only
	if strings.TrimSpace(request.Question) == "" {
		BadRequestHandler(w, "Question cannot be empty or whitespace-only")
		return
	}

	// Validate question length
	if len(request.Question) > h.maxQuestionLength {
		BadRequestHandler(w, "Question exceeds maximum length")
		return
	}

	// Call service layer
	ctx := context.Background()
	answer, err := h.service.SearchAnswer(ctx, request.Question)

	if err != nil {
		h.handleError(w, err)
		return
	}

	// Format success response
	response := QuestionSearchResponse{
		Answer: answer,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *QuestionSearchHandler) handleError(w http.ResponseWriter, err error) {
	// Check if it's a BedrockError
	if bedrockErr, ok := err.(*bedrockErrors.BedrockError); ok {
		switch bedrockErr.Code {
		case bedrockErrors.ErrCodeValidation:
			BadRequestHandler(w, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeThrottling:
			h.handleThrottlingError(w, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeEmbedding, bedrockErrors.ErrCodeKnowledgeBase:
			// Check if it's a quota error
			if strings.Contains(bedrockErr.Message, "quota") || strings.Contains(bedrockErr.Message, "Quota") {
				h.handleQuotaError(w, bedrockErr.Message)
				return
			}
			InternalServerErrorHandler(w, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeAWSService:
			// Check if it's a quota error
			if strings.Contains(bedrockErr.Message, "quota") || strings.Contains(bedrockErr.Message, "Quota") {
				h.handleQuotaError(w, bedrockErr.Message)
				return
			}
			InternalServerErrorHandler(w, bedrockErr.Message)
			return
		}
	}

	// Default to internal server error
	InternalServerErrorHandler(w, "An error occurred processing your request")
}

func (h *QuestionSearchHandler) handleThrottlingError(w http.ResponseWriter, message string) {
	errorResponse := ErrorResponse{
		Error:  message,
		Status: 429,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(errorResponse)
	
	log.Printf("[THROTTLING] Request throttled: %s", message)
}

func (h *QuestionSearchHandler) handleQuotaError(w http.ResponseWriter, message string) {
	errorResponse := ErrorResponse{
		Error:  message,
		Status: 503,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(errorResponse)
	
	log.Printf("[QUOTA] Quota exceeded: %s", message)
}
