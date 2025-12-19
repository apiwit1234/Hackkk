package routing

import (
	"encoding/json"
	"io"
	"net/http"

	"teletubpax-api/logger"
	"teletubpax-api/services"
)

type DocumentSummaryRequest struct {
	RelatedDocuments []string `json:"relatedDocuments"`
}

type DocumentSummaryResponse struct {
	Documents []services.DocumentSummaryItem `json:"documents"`
	Total     int                            `json:"total"`
}

type DocumentSummaryHandler struct {
	service services.DocumentSummaryService
}

func NewDocumentSummaryHandler(service services.DocumentSummaryService) *DocumentSummaryHandler {
	return &DocumentSummaryHandler{
		service: service,
	}
}

func (h *DocumentSummaryHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.WithContext(r.Context())

	log.Info("Document summary request", map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
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

	var request DocumentSummaryRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Warn("Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		BadRequestHandler(w, "Invalid JSON format")
		return
	}

	// Validate relatedDocuments field
	if len(request.RelatedDocuments) == 0 {
		log.Warn("relatedDocuments field is empty")
		BadRequestHandler(w, "relatedDocuments field is required and must not be empty")
		return
	}

	// Call service to analyze documents
	ctx := r.Context()
	documents, err := h.service.AnalyzeDocuments(ctx, request.RelatedDocuments)

	if err != nil {
		log.Error("Failed to analyze documents", map[string]interface{}{
			"error": err.Error(),
		})
		InternalServerErrorHandler(w, "Failed to analyze documents")
		return
	}

	// Format success response
	response := DocumentSummaryResponse{
		Documents: documents,
		Total:     len(documents),
	}

	log.Info("Document summary completed successfully", map[string]interface{}{
		"document_count": len(documents),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
