package routing

import (
	"encoding/json"
	"net/http"

	"teletubpax-api/logger"
	"teletubpax-api/services"
)

type DocumentDetailsResponse struct {
	Documents []map[string]interface{} `json:"documents"`
	Total     int                      `json:"total"`
}

type DocumentDetailsHandler struct {
	service services.DocumentDetailsService
}

func NewDocumentDetailsHandler(service services.DocumentDetailsService) *DocumentDetailsHandler {
	return &DocumentDetailsHandler{
		service: service,
	}
}

func (h *DocumentDetailsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log := logger.WithContext(r.Context())

	log.Info("Document details request", map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
	})

	// Call service to get last updated documents from OpenSearch
	ctx := r.Context()
	documents, err := h.service.GetLastUpdateDocuments(ctx)

	if err != nil {
		log.Error("Failed to retrieve documents", map[string]interface{}{
			"error": err.Error(),
		})
		InternalServerErrorHandler(w, "Failed to retrieve document details")
		return
	}

	// Format success response
	response := DocumentDetailsResponse{
		Documents: documents,
		Total:     len(documents),
	}

	log.Info("Document details retrieved successfully", map[string]interface{}{
		"document_count": len(documents),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
