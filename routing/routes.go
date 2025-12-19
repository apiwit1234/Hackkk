package routing

import (
	"encoding/json"
	"net/http"

	"teletubpax-api/logger"
	"teletubpax-api/services"

	"github.com/gorilla/mux"
)

// CORS middleware
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func SetupRoutes(questionSearchService services.QuestionSearchService, documentDetailsService services.DocumentDetailsService, documentSearchService services.DocumentSearchService, maxQuestionLength int) *mux.Router {
	router := mux.NewRouter()

	// Apply CORS middleware to all routes
	router.Use(CORSMiddleware)

	// Health check endpoint
	router.HandleFunc("/api/teletubpax/healthcheck", HealthCheckHandler).Methods("GET", "OPTIONS")

	questionSearchHandler := NewQuestionSearchHandler(questionSearchService, maxQuestionLength)
	router.HandleFunc("/api/teletubpax/question-search", questionSearchHandler.Handle).Methods("POST", "OPTIONS")

	documentDetailsHandler := NewDocumentDetailsHandler(documentDetailsService)
	router.HandleFunc("/api/teletubpax/last-update-document", documentDetailsHandler.Handle).Methods("GET", "OPTIONS")

	documentSearchHandler := NewDocumentSearchHandler(documentSearchService, maxQuestionLength)
	router.HandleFunc("/api/teletubpax/document-search", documentSearchHandler.Handle).Methods("POST")

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
