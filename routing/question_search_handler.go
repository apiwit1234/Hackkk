package routing

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	bedrockErrors "teletubpax-api/errors"
	"teletubpax-api/logger"
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
	log := logger.WithContext(r.Context())
	
	log.Info("Incoming request", map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.Header.Get("User-Agent"),
	})

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "" {
		log.Warn("Invalid content type", map[string]interface{}{
			"content_type": contentType,
		})
		BadRequestHandler(w, "Content-Type must be application/json")
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		BadRequestHandler(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var request QuestionSearchRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Warn("Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		BadRequestHandler(w, "Invalid JSON format")
		return
	}

	// Validate question field presence
	if request.Question == "" {
		log.Warn("Question field is empty")
		BadRequestHandler(w, "Question field is required")
		return
	}

	// Validate question is not whitespace-only
	if strings.TrimSpace(request.Question) == "" {
		log.Warn("Question is whitespace-only")
		BadRequestHandler(w, "Question cannot be empty or whitespace-only")
		return
	}

	// Validate question length
	if len(request.Question) > h.maxQuestionLength {
		log.Warn("Question exceeds maximum length", map[string]interface{}{
			"length":     len(request.Question),
			"max_length": h.maxQuestionLength,
		})
		BadRequestHandler(w, "Question exceeds maximum length")
		return
	}

	// Call service layer
	ctx := r.Context()
	answer, err := h.service.SearchAnswer(ctx, request.Question)

	if err != nil {
		h.handleError(w, r, err)
		return
	}

	// Format success response
	response := QuestionSearchResponse{
		Answer: answer,
	}

	log.Info("Request completed successfully", map[string]interface{}{
		"answer_length": len(answer),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *QuestionSearchHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log := logger.WithContext(r.Context())
	
	// Check if it's a BedrockError
	if bedrockErr, ok := err.(*bedrockErrors.BedrockError); ok {
		switch bedrockErr.Code {
		case bedrockErrors.ErrCodeValidation:
			log.Warn("Validation error", map[string]interface{}{
				"error": bedrockErr.Message,
			})
			BadRequestHandler(w, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeThrottling:
			h.handleThrottlingError(w, r, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeEmbedding, bedrockErrors.ErrCodeKnowledgeBase:
			// Check if it's a quota error
			if strings.Contains(bedrockErr.Message, "quota") || strings.Contains(bedrockErr.Message, "Quota") {
				h.handleQuotaError(w, r, bedrockErr.Message)
				return
			}
			log.Error("Bedrock service error", map[string]interface{}{
				"error_code": bedrockErr.Code,
				"error":      bedrockErr.Message,
			})
			InternalServerErrorHandler(w, bedrockErr.Message)
			return
		case bedrockErrors.ErrCodeAWSService:
			// Check if it's a quota error
			if strings.Contains(bedrockErr.Message, "quota") || strings.Contains(bedrockErr.Message, "Quota") {
				h.handleQuotaError(w, r, bedrockErr.Message)
				return
			}
			log.Error("AWS service error", map[string]interface{}{
				"error": bedrockErr.Message,
			})
			InternalServerErrorHandler(w, bedrockErr.Message)
			return
		}
	}

	// Default to internal server error
	log.Error("Unhandled error", map[string]interface{}{
		"error": err.Error(),
	})
	InternalServerErrorHandler(w, "An error occurred processing your request")
}

func (h *QuestionSearchHandler) handleThrottlingError(w http.ResponseWriter, r *http.Request, message string) {
	log := logger.WithContext(r.Context())
	log.Warn("Request throttled", map[string]interface{}{
		"error":       message,
		"retry_after": 60,
	})

	errorResponse := ErrorResponse{
		Error:  message,
		Status: 429,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(errorResponse)
}

func (h *QuestionSearchHandler) handleQuotaError(w http.ResponseWriter, r *http.Request, message string) {
	log := logger.WithContext(r.Context())
	log.Error("Quota exceeded", map[string]interface{}{
		"error": message,
	})

	errorResponse := ErrorResponse{
		Error:  message,
		Status: 503,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(errorResponse)
}
