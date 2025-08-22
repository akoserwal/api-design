package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Details interface{} `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SuccessResponse represents a structured success response
type SuccessResponse struct {
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message"`
	Code      int         `json:"code"`
	Timestamp time.Time   `json:"timestamp"`
}

// Sample resource
type Resource struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

var resources = []Resource{
	{ID: 1, Name: "Resource 1", Status: "active", CreatedAt: time.Now().Add(-time.Hour)},
	{ID: 2, Name: "Resource 2", Status: "inactive", CreatedAt: time.Now().Add(-2*time.Hour)},
}

func respondWithError(w http.ResponseWriter, code int, message string, details interface{}) {
	response := ErrorResponse{
		Error:     http.StatusText(code),
		Message:   message,
		Code:      code,
		Details:   details,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func respondWithSuccess(w http.ResponseWriter, code int, message string, data interface{}) {
	response := SuccessResponse{
		Data:      data,
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

// Status code demonstration endpoints

// 200 OK - Request succeeded
func test200Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[200] OK - Standard success response\n")
	respondWithSuccess(w, http.StatusOK, "Request successful", resources)
}

// 201 Created - Resource successfully created
func test201Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[201] Created - Resource successfully created\n")
	
	newResource := Resource{
		ID:        len(resources) + 1,
		Name:      "New Resource",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	
	w.Header().Set("Location", fmt.Sprintf("/resources/%d", newResource.ID))
	respondWithSuccess(w, http.StatusCreated, "Resource created successfully", newResource)
}

// 202 Accepted - Request accepted for processing
func test202Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[202] Accepted - Request accepted for processing\n")
	
	processingInfo := map[string]interface{}{
		"job_id": "job-12345",
		"status": "processing",
		"estimated_completion": time.Now().Add(5 * time.Minute),
		"status_url": "/jobs/job-12345",
	}
	
	respondWithSuccess(w, http.StatusAccepted, "Request accepted for processing", processingInfo)
}

// 204 No Content - Success but no content to return
func test204Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[204] No Content - Success with no response body\n")
	w.WriteHeader(http.StatusNoContent)
	// No body for 204 responses
}

// 301 Moved Permanently - Resource permanently moved
func test301Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[301] Moved Permanently - Resource moved to new location\n")
	w.Header().Set("Location", "/api/v2/test/200")
	w.WriteHeader(http.StatusMovedPermanently)
	fmt.Fprintf(w, "Resource moved permanently to /api/v2/test/200")
}

// 302 Found - Resource temporarily moved
func test302Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[302] Found - Resource temporarily moved\n")
	w.Header().Set("Location", "/api/test/200")
	w.WriteHeader(http.StatusFound)
	fmt.Fprintf(w, "Resource temporarily moved to /api/test/200")
}

// 304 Not Modified - Resource not modified since last request
func test304Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[304] Not Modified - Resource unchanged\n")
	
	// Check If-None-Match header
	ifNoneMatch := r.Header.Get("If-None-Match")
	currentETag := `"resource-123-unchanged"`
	
	w.Header().Set("ETag", currentETag)
	
	if ifNoneMatch == currentETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	
	// If no matching ETag, return 200 with content
	respondWithSuccess(w, http.StatusOK, "Resource content", map[string]string{
		"data": "Resource content here",
		"etag": currentETag,
	})
}

// 400 Bad Request - Client sent invalid request
func test400Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[400] Bad Request - Invalid client request\n")
	
	validationErrors := []map[string]string{
		{"field": "email", "error": "Invalid email format"},
		{"field": "age", "error": "Age must be a positive number"},
	}
	
	respondWithError(w, http.StatusBadRequest, "Invalid request data", validationErrors)
}

// 401 Unauthorized - Authentication required
func test401Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[401] Unauthorized - Authentication required\n")
	
	w.Header().Set("WWW-Authenticate", "Bearer")
	respondWithError(w, http.StatusUnauthorized, "Authentication required", map[string]string{
		"hint": "Include Authorization header with valid token",
	})
}

// 403 Forbidden - Access denied
func test403Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[403] Forbidden - Access denied\n")
	
	respondWithError(w, http.StatusForbidden, "Access denied", map[string]string{
		"reason": "Insufficient permissions for this operation",
	})
}

// 404 Not Found - Resource not found
func test404Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[404] Not Found - Resource does not exist\n")
	
	respondWithError(w, http.StatusNotFound, "Resource not found", map[string]string{
		"requested_resource": "/api/nonexistent",
	})
}

// 405 Method Not Allowed - HTTP method not allowed
func test405Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[405] Method Not Allowed - HTTP method not supported\n")
	
	w.Header().Set("Allow", "GET, POST")
	respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", map[string]interface{}{
		"method": r.Method,
		"allowed_methods": []string{"GET", "POST"},
	})
}

// 409 Conflict - Request conflicts with current state
func test409Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[409] Conflict - Request conflicts with current state\n")
	
	respondWithError(w, http.StatusConflict, "Resource already exists", map[string]string{
		"conflict": "A resource with this identifier already exists",
		"existing_resource": "/api/resources/123",
	})
}

// 422 Unprocessable Entity - Semantically invalid request
func test422Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[422] Unprocessable Entity - Semantically invalid\n")
	
	semanticErrors := []map[string]string{
		{"field": "start_date", "error": "Start date cannot be in the future"},
		{"field": "end_date", "error": "End date must be after start date"},
	}
	
	respondWithError(w, http.StatusUnprocessableEntity, "Semantic validation failed", semanticErrors)
}

// 429 Too Many Requests - Rate limit exceeded
func test429Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[429] Too Many Requests - Rate limit exceeded\n")
	
	w.Header().Set("Retry-After", "60")
	w.Header().Set("X-RateLimit-Limit", "100")
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
	
	respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded", map[string]interface{}{
		"retry_after_seconds": 60,
		"limit": 100,
		"window": "1 minute",
	})
}

// 500 Internal Server Error - Server error
func test500Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[500] Internal Server Error - Server encountered an error\n")
	
	respondWithError(w, http.StatusInternalServerError, "Internal server error", map[string]string{
		"error_id": "err-12345",
		"message": "An unexpected error occurred. Please try again later.",
	})
}

// 502 Bad Gateway - Invalid response from upstream
func test502Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[502] Bad Gateway - Invalid upstream response\n")
	
	respondWithError(w, http.StatusBadGateway, "Bad gateway", map[string]string{
		"upstream": "payment-service",
		"error": "Invalid response from upstream service",
	})
}

// 503 Service Unavailable - Service temporarily unavailable
func test503Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[503] Service Unavailable - Service temporarily unavailable\n")
	
	w.Header().Set("Retry-After", "300")
	respondWithError(w, http.StatusServiceUnavailable, "Service unavailable", map[string]interface{}{
		"reason": "Scheduled maintenance",
		"retry_after_seconds": 300,
		"estimated_recovery": time.Now().Add(5 * time.Minute),
	})
}

// Real-world example: CRUD operations with proper status codes
func getResourceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid resource ID", nil)
		return
	}

	for _, resource := range resources {
		if resource.ID == id {
			respondWithSuccess(w, http.StatusOK, "Resource found", resource)
			return
		}
	}

	respondWithError(w, http.StatusNotFound, "Resource not found", nil)
}

func createResourceHandler(w http.ResponseWriter, r *http.Request) {
	var resource Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", nil)
		return
	}

	// Validation
	if resource.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name is required", nil)
		return
	}

	// Check for duplicates (409 Conflict)
	for _, existing := range resources {
		if existing.Name == resource.Name {
			respondWithError(w, http.StatusConflict, "Resource with this name already exists", nil)
			return
		}
	}

	resource.ID = len(resources) + 1
	resource.CreatedAt = time.Now()
	resources = append(resources, resource)

	w.Header().Set("Location", fmt.Sprintf("/resources/%d", resource.ID))
	respondWithSuccess(w, http.StatusCreated, "Resource created successfully", resource)
}

// Status codes overview
func statusCodesInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"title": "HTTP Status Codes Demonstration",
		"categories": map[string]interface{}{
			"1xx": "Informational responses (rarely used in REST APIs)",
			"2xx": "Success responses",
			"3xx": "Redirection messages",
			"4xx": "Client error responses",
			"5xx": "Server error responses",
		},
		"status_codes": map[string]interface{}{
			"200": "OK - Standard success response",
			"201": "Created - Resource successfully created",
			"202": "Accepted - Request accepted for processing",
			"204": "No Content - Success with no response body",
			"301": "Moved Permanently - Resource permanently moved",
			"302": "Found - Resource temporarily moved",
			"304": "Not Modified - Resource unchanged",
			"400": "Bad Request - Invalid client request",
			"401": "Unauthorized - Authentication required",
			"403": "Forbidden - Access denied",
			"404": "Not Found - Resource not found",
			"405": "Method Not Allowed - HTTP method not supported",
			"409": "Conflict - Request conflicts with current state",
			"422": "Unprocessable Entity - Semantically invalid",
			"429": "Too Many Requests - Rate limit exceeded",
			"500": "Internal Server Error - Server error",
			"502": "Bad Gateway - Invalid upstream response",
			"503": "Service Unavailable - Service temporarily unavailable",
		},
		"test_endpoints": map[string]string{
			"/api/test/200": "200 OK",
			"/api/test/201": "201 Created",
			"/api/test/400": "400 Bad Request",
			"/api/test/404": "404 Not Found",
			"/api/test/500": "500 Internal Server Error",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	router := mux.NewRouter()

	// Information endpoint
	router.HandleFunc("/", statusCodesInfoHandler).Methods("GET")

	// Status code test endpoints
	api := router.PathPrefix("/api/test").Subrouter()
	
	// 2xx Success
	api.HandleFunc("/200", test200Handler).Methods("GET")
	api.HandleFunc("/201", test201Handler).Methods("POST")
	api.HandleFunc("/202", test202Handler).Methods("POST")
	api.HandleFunc("/204", test204Handler).Methods("DELETE")
	
	// 3xx Redirection
	api.HandleFunc("/301", test301Handler).Methods("GET")
	api.HandleFunc("/302", test302Handler).Methods("GET")
	api.HandleFunc("/304", test304Handler).Methods("GET")
	
	// 4xx Client Errors
	api.HandleFunc("/400", test400Handler).Methods("GET")
	api.HandleFunc("/401", test401Handler).Methods("GET")
	api.HandleFunc("/403", test403Handler).Methods("GET")
	api.HandleFunc("/404", test404Handler).Methods("GET")
	api.HandleFunc("/405", test405Handler).Methods("PUT", "DELETE") // Only allow PUT, DELETE to demo 405
	api.HandleFunc("/409", test409Handler).Methods("POST")
	api.HandleFunc("/422", test422Handler).Methods("POST")
	api.HandleFunc("/429", test429Handler).Methods("GET")
	
	// 5xx Server Errors
	api.HandleFunc("/500", test500Handler).Methods("GET")
	api.HandleFunc("/502", test502Handler).Methods("GET")
	api.HandleFunc("/503", test503Handler).Methods("GET")

	// Real-world examples
	resources := router.PathPrefix("/resources").Subrouter()
	resources.HandleFunc("", createResourceHandler).Methods("POST")
	resources.HandleFunc("/{id}", getResourceHandler).Methods("GET")

	fmt.Println("HTTP Status Codes Demonstration Server")
	fmt.Println("=====================================")
	fmt.Println("Server starting on :8084")
	fmt.Println("\nStatus Code Categories:")
	fmt.Println("2xx - Success responses")
	fmt.Println("3xx - Redirection messages")
	fmt.Println("4xx - Client error responses")
	fmt.Println("5xx - Server error responses")
	fmt.Println("\nTest commands:")
	fmt.Println("curl http://localhost:8084/api/test/200")
	fmt.Println("curl -X POST http://localhost:8084/api/test/201")
	fmt.Println("curl http://localhost:8084/api/test/400")
	fmt.Println("curl http://localhost:8084/api/test/404")
	fmt.Println("curl http://localhost:8084/api/test/500")
	fmt.Println("curl -H \"If-None-Match: \\\"resource-123-unchanged\\\"\" http://localhost:8084/api/test/304")
	fmt.Println("\nVisit http://localhost:8084/ for complete information")

	log.Fatal(http.ListenAndServe(":8084", router))
}