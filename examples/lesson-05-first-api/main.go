package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Task represents a task in our system
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

// TaskListResponse represents a list of tasks with metadata
type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
	Meta  Meta   `json:"meta"`
}

// Meta contains response metadata
type Meta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// In-memory storage
var tasks []Task
var nextID = 1

// Storage interface for future database implementation
type TaskStorage interface {
	GetAll() []Task
	GetByID(id string) (*Task, error)
	GetByCompleted(completed bool) []Task
	Create(task *Task) error
	Update(id string, updates UpdateTaskRequest) (*Task, error)
	Delete(id string) error
}

// MemoryStorage implements TaskStorage using in-memory storage
type MemoryStorage struct{}

func (ms *MemoryStorage) GetAll() []Task {
	return tasks
}

func (ms *MemoryStorage) GetByID(id string) (*Task, error) {
	for _, task := range tasks {
		if task.ID == id {
			return &task, nil
		}
	}
	return nil, fmt.Errorf("task not found")
}

func (ms *MemoryStorage) GetByCompleted(completed bool) []Task {
	var filtered []Task
	for _, task := range tasks {
		if task.Completed == completed {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func (ms *MemoryStorage) Create(task *Task) error {
	tasks = append(tasks, *task)
	return nil
}

func (ms *MemoryStorage) Update(id string, updates UpdateTaskRequest) (*Task, error) {
	for i, task := range tasks {
		if task.ID == id {
			if updates.Title != nil {
				tasks[i].Title = *updates.Title
			}
			if updates.Description != nil {
				tasks[i].Description = *updates.Description
			}
			if updates.Completed != nil {
				tasks[i].Completed = *updates.Completed
			}
			tasks[i].UpdatedAt = time.Now()
			return &tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task not found")
}

func (ms *MemoryStorage) Delete(id string) error {
	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task not found")
}

// TaskHandler handles HTTP requests for tasks
type TaskHandler struct {
	storage TaskStorage
}

func NewTaskHandler(storage TaskStorage) *TaskHandler {
	return &TaskHandler{storage: storage}
}

func (h *TaskHandler) respondWithError(w http.ResponseWriter, code int, message string, requestID string) {
	response := ErrorResponse{
		Error:     http.StatusText(code),
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (h *TaskHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func generateRequestID() string {
	return uuid.New().String()[:8]
}

// GET /api/tasks
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	
	// Check for completed query parameter
	completedParam := r.URL.Query().Get("completed")
	
	var taskList []Task
	
	if completedParam != "" {
		completed, err := strconv.ParseBool(completedParam)
		if err != nil {
			h.respondWithError(w, http.StatusBadRequest, "Invalid 'completed' parameter. Use true or false.", requestID)
			return
		}
		taskList = h.storage.GetByCompleted(completed)
	} else {
		taskList = h.storage.GetAll()
	}
	
	response := TaskListResponse{
		Tasks: taskList,
		Count: len(taskList),
		Meta: Meta{
			RequestID: requestID,
			Timestamp: time.Now(),
		},
	}
	
	h.respondWithJSON(w, http.StatusOK, response)
}

// GET /api/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	task, err := h.storage.GetByID(id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusOK, task)
}

// POST /api/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", requestID)
		return
	}
	
	// Validation
	if strings.TrimSpace(req.Title) == "" {
		h.respondWithError(w, http.StatusBadRequest, "Title is required and cannot be empty", requestID)
		return
	}
	
	if len(req.Title) > 100 {
		h.respondWithError(w, http.StatusBadRequest, "Title cannot exceed 100 characters", requestID)
		return
	}
	
	if len(req.Description) > 500 {
		h.respondWithError(w, http.StatusBadRequest, "Description cannot exceed 500 characters", requestID)
		return
	}
	
	now := time.Now()
	task := Task{
		ID:          uuid.New().String(),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	
	if err := h.storage.Create(&task); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create task", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusCreated, task)
}

// PUT /api/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", requestID)
		return
	}
	
	// For PUT, we require all fields (except completed)
	if req.Title == nil || req.Description == nil {
		h.respondWithError(w, http.StatusBadRequest, "PUT requires title and description fields", requestID)
		return
	}
	
	// Validation
	if strings.TrimSpace(*req.Title) == "" {
		h.respondWithError(w, http.StatusBadRequest, "Title is required and cannot be empty", requestID)
		return
	}
	
	if len(*req.Title) > 100 {
		h.respondWithError(w, http.StatusBadRequest, "Title cannot exceed 100 characters", requestID)
		return
	}
	
	if len(*req.Description) > 500 {
		h.respondWithError(w, http.StatusBadRequest, "Description cannot exceed 500 characters", requestID)
		return
	}
	
	// Trim whitespace
	trimmedTitle := strings.TrimSpace(*req.Title)
	trimmedDesc := strings.TrimSpace(*req.Description)
	req.Title = &trimmedTitle
	req.Description = &trimmedDesc
	
	task, err := h.storage.Update(id, req)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusOK, task)
}

// PATCH /api/tasks/{id}
func (h *TaskHandler) PatchTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload", requestID)
		return
	}
	
	// For PATCH, at least one field must be provided
	if req.Title == nil && req.Description == nil && req.Completed == nil {
		h.respondWithError(w, http.StatusBadRequest, "At least one field (title, description, or completed) must be provided", requestID)
		return
	}
	
	// Validation for provided fields
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			h.respondWithError(w, http.StatusBadRequest, "Title cannot be empty", requestID)
			return
		}
		if len(*req.Title) > 100 {
			h.respondWithError(w, http.StatusBadRequest, "Title cannot exceed 100 characters", requestID)
			return
		}
		trimmed := strings.TrimSpace(*req.Title)
		req.Title = &trimmed
	}
	
	if req.Description != nil {
		if len(*req.Description) > 500 {
			h.respondWithError(w, http.StatusBadRequest, "Description cannot exceed 500 characters", requestID)
			return
		}
		trimmed := strings.TrimSpace(*req.Description)
		req.Description = &trimmed
	}
	
	task, err := h.storage.Update(id, req)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusOK, task)
}

// DELETE /api/tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	err := h.storage.Delete(id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// PATCH /api/tasks/{id}/complete
func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	completed := true
	req := UpdateTaskRequest{Completed: &completed}
	
	task, err := h.storage.Update(id, req)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusOK, task)
}

// PATCH /api/tasks/{id}/uncomplete
func (h *TaskHandler) UncompleteTask(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	vars := mux.Vars(r)
	id := vars["id"]
	
	completed := false
	req := UpdateTaskRequest{Completed: &completed}
	
	task, err := h.storage.Update(id, req)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Task not found", requestID)
		return
	}
	
	h.respondWithJSON(w, http.StatusOK, task)
}

// Health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "task-api",
		"version":   "1.0.0",
		"uptime":    time.Since(startTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// API information endpoint
func apiInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "Task Management API",
		"version":     "1.0.0",
		"description": "A simple REST API for managing tasks",
		"endpoints": map[string]interface{}{
			"GET /health":                        "Health check",
			"GET /api/tasks":                     "Get all tasks",
			"GET /api/tasks?completed=true":      "Get completed tasks",
			"GET /api/tasks/{id}":                "Get specific task",
			"POST /api/tasks":                    "Create new task",
			"PUT /api/tasks/{id}":                "Update entire task",
			"PATCH /api/tasks/{id}":              "Partial task update",
			"DELETE /api/tasks/{id}":             "Delete task",
			"PATCH /api/tasks/{id}/complete":     "Mark task as completed",
			"PATCH /api/tasks/{id}/uncomplete":   "Mark task as incomplete",
		},
		"example_usage": []string{
			`curl -X POST http://localhost:8087/api/tasks -H "Content-Type: application/json" -d '{"title":"Learn REST","description":"Complete the course"}'`,
			`curl http://localhost:8087/api/tasks`,
			`curl http://localhost:8087/api/tasks?completed=false`,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		next.ServeHTTP(w, r)
		
		log.Printf("[%s] %s %s - %v",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			time.Since(start))
	})
}

var startTime = time.Now()

func main() {
	// Initialize storage
	storage := &MemoryStorage{}
	
	// Create some sample data
	sampleTasks := []Task{
		{
			ID:          uuid.New().String(),
			Title:       "Learn Go",
			Description: "Complete Go programming tutorial",
			Completed:   true,
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			Title:       "Build REST API",
			Description: "Create a task management API",
			Completed:   false,
			CreatedAt:   time.Now().Add(-12 * time.Hour),
			UpdatedAt:   time.Now().Add(-12 * time.Hour),
		},
	}
	
	for _, task := range sampleTasks {
		storage.Create(&task)
	}
	
	// Initialize handler
	taskHandler := NewTaskHandler(storage)
	
	// Setup routes
	router := mux.NewRouter()
	
	// Apply middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)
	
	// Health and info endpoints
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
	router.HandleFunc("/", apiInfoHandler).Methods("GET")
	
	// API routes
	api := router.PathPrefix("/api").Subrouter()
	
	// Task routes
	api.HandleFunc("/tasks", taskHandler.GetTasks).Methods("GET")
	api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", taskHandler.PatchTask).Methods("PATCH")
	api.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/complete", taskHandler.CompleteTask).Methods("PATCH")
	api.HandleFunc("/tasks/{id}/uncomplete", taskHandler.UncompleteTask).Methods("PATCH")
	
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}
	
	fmt.Println("🚀 Task Management API")
	fmt.Println("=====================")
	fmt.Printf("Server starting on port %s\n", port)
	fmt.Printf("Health check: http://localhost:%s/health\n", port)
	fmt.Printf("API info: http://localhost:%s/\n", port)
	fmt.Printf("API base URL: http://localhost:%s/api\n", port)
	fmt.Println("\nSample requests:")
	fmt.Printf("curl http://localhost:%s/api/tasks\n", port)
	fmt.Printf(`curl -X POST http://localhost:%s/api/tasks -H "Content-Type: application/json" -d '{"title":"New Task","description":"Task description"}'`+"\n", port)
	
	log.Fatal(http.ListenAndServe(":"+port, router))
}