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

type DocumentSearchRequest struct {
	Keyword string `json:"keyword"`
}

type DocumentSearchResponse struct {
	RelatedDocuments []string `json:"relatedDocuments"`
}

type DocumentSearchHandler struct {
	service        services.DocumentSearchService
	maxKeywordLength int
}

func NewDocumentSearchHandler(service services.DocumentSearchService, maxKeywordLength int) *DocumentSearchHandler {
	return &DocumentSearchHandler{
		service:        service,
		maxKeywordLength: maxKeywordLength,
	}
}

func (h *DocumentSearchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.WithContext(r.Context())
	
	log.Info("Incoming request", map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.Header.Get("User-Agent"),
	})

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "" {
		log.Warn("Invalid content type", map[string]interface{}{
			"content_type": contentType,
		})
		BadRequestHandler(w, "Content-Type must be application/json")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		BadRequestHandler(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var request DocumentSearchRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Warn("Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		BadRequestHandler(w, "Invalid JSON format")
		return
	}

	if request.Keyword == "" {
		log.Warn("Keyword field is empty")
		BadRequestHandler(w, "Keyword field is required")
		return
	}

	if strings.TrimSpace(request.Keyword) == "" {
		log.Warn("Keyword is whitespace-only")
		BadRequestHandler(w, "Keyword cannot be empty or whitespace-only")
		return
	}

	if len(request.Keyword) > h.maxKeywordLength {
		log.Warn("Keyword exceeds maximum length", map[string]interface{}{
			"length":     len(request.Keyword),
			"max_length": h.maxKeywordLength,
		})
		BadRequestHandler(w, "Keyword exceeds maximum length")
		return
	}

	ctx := r.Context()
	relatedDocuments, err := h.service.SearchDocumentsByKeyword(ctx, request.Keyword)

	if err != nil {
		h.handleError(w, r, err)
		return
	}

	response := DocumentSearchResponse{
		RelatedDocuments: relatedDocuments,
	}

	log.Info("Request completed successfully", map[string]interface{}{
		"document_count": len(relatedDocuments),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *DocumentSearchHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log := logger.WithContext(r.Context())
	
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

	log.Error("Unhandled error", map[string]interface{}{
		"error": err.Error(),
	})
	InternalServerErrorHandler(w, "An error occurred processing your request")
}

func (h *DocumentSearchHandler) handleThrottlingError(w http.ResponseWriter, r *http.Request, message string) {
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

func (h *DocumentSearchHandler) handleQuotaError(w http.ResponseWriter, r *http.Request, message string) {
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
