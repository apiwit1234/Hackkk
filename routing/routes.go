package routing

import (
	"encoding/json"
	"net/http"

	"teletubpax-api/logger"
	"teletubpax-api/services"

	"github.com/gorilla/mux"
)

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func SetupRoutes(questionSearchService services.QuestionSearchService, documentDetailsService services.DocumentDetailsService, maxQuestionLength int) *mux.Router {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/api/teletubpax/healthcheck", HealthCheckHandler).Methods("GET")

	// Question search endpoint
	questionSearchHandler := NewQuestionSearchHandler(questionSearchService, maxQuestionLength)
	router.HandleFunc("/api/teletubpax/question-search", questionSearchHandler.Handle).Methods("POST")

	// Document details endpoint
	documentDetailsHandler := NewDocumentDetailsHandler(documentDetailsService)
	router.HandleFunc("/api/teletubpax/last-update-documentdetails", documentDetailsHandler.Handle).Methods("GET")

	// 404 handler
	router.NotFoundHandler = http.HandlerFunc(NotFoundHandler)

	return router
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Message: "I'm OK",
		Status:  200,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.WithContext(r.Context())
	log.Warn("Resource not found", map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
	})

	errorResponse := ErrorResponse{
		Error:  "Resource not found",
		Status: 404,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(errorResponse)
}

func BadRequestHandler(w http.ResponseWriter, message string) {
	errorResponse := ErrorResponse{
		Error:  message,
		Status: 400,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(errorResponse)
}

func InternalServerErrorHandler(w http.ResponseWriter, message string) {
	errorResponse := ErrorResponse{
		Error:  message,
		Status: 500,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(errorResponse)
}
