# Lesson 5: Building Your First REST API in Go

## Learning Objectives
By the end of this lesson, you will be able to:
- Create a complete CRUD REST API in Go
- Handle HTTP requests and responses properly
- Implement JSON marshaling and unmarshaling
- Use proper HTTP status codes
- Structure a maintainable API codebase
- Test your API endpoints

## Project Overview

We'll build a **Task Management API** with the following features:
- Create, read, update, and delete tasks
- Mark tasks as completed/incomplete
- Filter tasks by status
- Proper error handling and validation

## Setting Up the Project

### Initialize the Project
```bash
mkdir task-api
cd task-api
go mod init github.com/yourusername/task-api
```

### Install Dependencies
```bash
go get github.com/gorilla/mux
go get github.com/google/uuid
```

### Project Structure
```
task-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── models/
│   │   └── task.go
│   ├── handlers/
│   │   └── tasks.go
│   └── storage/
│       └── memory.go
├── go.mod
└── go.sum
```

## Data Models

### Task Model
```go
// internal/models/task.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type Task struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Completed   bool      `json:"completed"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTaskRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

type UpdateTaskRequest struct {
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    Completed   *bool   `json:"completed,omitempty"`
}

func NewTask(title, description string) *Task {
    now := time.Now()
    return &Task{
        ID:          uuid.New().String(),
        Title:       title,
        Description: description,
        Completed:   false,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
}

func (t *Task) Update(req UpdateTaskRequest) {
    if req.Title != nil {
        t.Title = *req.Title
    }
    if req.Description != nil {
        t.Description = *req.Description
    }
    if req.Completed != nil {
        t.Completed = *req.Completed
    }
    t.UpdatedAt = time.Now()
}
```

## In-Memory Storage

### Storage Interface
```go
// internal/storage/memory.go
package storage

import (
    "errors"
    "sync"
    "github.com/yourusername/task-api/internal/models"
)

var (
    ErrTaskNotFound = errors.New("task not found")
    ErrTaskExists   = errors.New("task already exists")
)

type TaskStorage interface {
    GetAll() []*models.Task
    GetByID(id string) (*models.Task, error)
    Create(task *models.Task) error
    Update(id string, req models.UpdateTaskRequest) (*models.Task, error)
    Delete(id string) error
    GetByStatus(completed bool) []*models.Task
}

type MemoryStorage struct {
    tasks map[string]*models.Task
    mutex sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
    return &MemoryStorage{
        tasks: make(map[string]*models.Task),
    }
}

func (s *MemoryStorage) GetAll() []*models.Task {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    tasks := make([]*models.Task, 0, len(s.tasks))
    for _, task := range s.tasks {
        tasks = append(tasks, task)
    }
    return tasks
}

func (s *MemoryStorage) GetByID(id string) (*models.Task, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    task, exists := s.tasks[id]
    if !exists {
        return nil, ErrTaskNotFound
    }
    return task, nil
}

func (s *MemoryStorage) Create(task *models.Task) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if _, exists := s.tasks[task.ID]; exists {
        return ErrTaskExists
    }
    
    s.tasks[task.ID] = task
    return nil
}

func (s *MemoryStorage) Update(id string, req models.UpdateTaskRequest) (*models.Task, error) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    task, exists := s.tasks[id]
    if !exists {
        return nil, ErrTaskNotFound
    }
    
    task.Update(req)
    return task, nil
}

func (s *MemoryStorage) Delete(id string) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if _, exists := s.tasks[id]; !exists {
        return ErrTaskNotFound
    }
    
    delete(s.tasks, id)
    return nil
}

func (s *MemoryStorage) GetByStatus(completed bool) []*models.Task {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    var tasks []*models.Task
    for _, task := range s.tasks {
        if task.Completed == completed {
            tasks = append(tasks, task)
        }
    }
    return tasks
}
```

## HTTP Handlers

### Task Handlers
```go
// internal/handlers/tasks.go
package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"
    
    "github.com/gorilla/mux"
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

type TaskHandler struct {
    storage storage.TaskStorage
}

func NewTaskHandler(storage storage.TaskStorage) *TaskHandler {
    return &TaskHandler{storage: storage}
}

// Error response structure
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}

// Success response structure for lists
type TaskListResponse struct {
    Tasks []models.Task `json:"tasks"`
    Count int           `json:"count"`
}

func (h *TaskHandler) respondWithError(w http.ResponseWriter, code int, message string) {
    h.respondWithJSON(w, code, ErrorResponse{
        Error:   http.StatusText(code),
        Message: message,
    })
}

func (h *TaskHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}

// GET /api/tasks
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
    // Check for completed query parameter
    completedParam := r.URL.Query().Get("completed")
    
    var tasks []*models.Task
    
    if completedParam != "" {
        completed, err := strconv.ParseBool(completedParam)
        if err != nil {
            h.respondWithError(w, http.StatusBadRequest, "Invalid 'completed' parameter")
            return
        }
        tasks = h.storage.GetByStatus(completed)
    } else {
        tasks = h.storage.GetAll()
    }
    
    // Convert pointers to values for JSON response
    taskList := make([]models.Task, len(tasks))
    for i, task := range tasks {
        taskList[i] = *task
    }
    
    h.respondWithJSON(w, http.StatusOK, TaskListResponse{
        Tasks: taskList,
        Count: len(taskList),
    })
}

// GET /api/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    task, err := h.storage.GetByID(id)
    if err != nil {
        if err == storage.ErrTaskNotFound {
            h.respondWithError(w, http.StatusNotFound, "Task not found")
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve task")
        return
    }
    
    h.respondWithJSON(w, http.StatusOK, task)
}

// POST /api/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req models.CreateTaskRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    // Validation
    if req.Title == "" {
        h.respondWithError(w, http.StatusBadRequest, "Title is required")
        return
    }
    
    task := models.NewTask(req.Title, req.Description)
    
    if err := h.storage.Create(task); err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to create task")
        return
    }
    
    h.respondWithJSON(w, http.StatusCreated, task)
}

// PUT /api/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    var req models.UpdateTaskRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    // Validation: at least one field should be provided
    if req.Title == nil && req.Description == nil && req.Completed == nil {
        h.respondWithError(w, http.StatusBadRequest, "At least one field must be provided")
        return
    }
    
    // Validate title if provided
    if req.Title != nil && *req.Title == "" {
        h.respondWithError(w, http.StatusBadRequest, "Title cannot be empty")
        return
    }
    
    task, err := h.storage.Update(id, req)
    if err != nil {
        if err == storage.ErrTaskNotFound {
            h.respondWithError(w, http.StatusNotFound, "Task not found")
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "Failed to update task")
        return
    }
    
    h.respondWithJSON(w, http.StatusOK, task)
}

// DELETE /api/tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    err := h.storage.Delete(id)
    if err != nil {
        if err == storage.ErrTaskNotFound {
            h.respondWithError(w, http.StatusNotFound, "Task not found")
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "Failed to delete task")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// PATCH /api/tasks/{id}/complete
func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    completed := true
    req := models.UpdateTaskRequest{Completed: &completed}
    
    task, err := h.storage.Update(id, req)
    if err != nil {
        if err == storage.ErrTaskNotFound {
            h.respondWithError(w, http.StatusNotFound, "Task not found")
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "Failed to complete task")
        return
    }
    
    h.respondWithJSON(w, http.StatusOK, task)
}

// PATCH /api/tasks/{id}/uncomplete
func (h *TaskHandler) UncompleteTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    completed := false
    req := models.UpdateTaskRequest{Completed: &completed}
    
    task, err := h.storage.Update(id, req)
    if err != nil {
        if err == storage.ErrTaskNotFound {
            h.respondWithError(w, http.StatusNotFound, "Task not found")
            return
        }
        h.respondWithError(w, http.StatusInternalServerError, "Failed to uncomplete task")
        return
    }
    
    h.respondWithJSON(w, http.StatusOK, task)
}
```

## Main Application

### Application Entry Point
```go
// cmd/api/main.go
package main

import (
    "log"
    "net/http"
    "os"
    
    "github.com/gorilla/mux"
    "github.com/yourusername/task-api/internal/handlers"
    "github.com/yourusername/task-api/internal/storage"
)

func main() {
    // Initialize storage
    taskStorage := storage.NewMemoryStorage()
    
    // Initialize handlers
    taskHandler := handlers.NewTaskHandler(taskStorage)
    
    // Initialize router
    router := mux.NewRouter()
    
    // API routes
    api := router.PathPrefix("/api").Subrouter()
    
    // Task routes
    api.HandleFunc("/tasks", taskHandler.GetTasks).Methods("GET")
    api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
    api.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
    api.HandleFunc("/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
    api.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
    api.HandleFunc("/tasks/{id}/complete", taskHandler.CompleteTask).Methods("PATCH")
    api.HandleFunc("/tasks/{id}/uncomplete", taskHandler.UncompleteTask).Methods("PATCH")
    
    // Health check endpoint
    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "ok"}`))
    }).Methods("GET")
    
    // Add CORS middleware for development
    router.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    })
    
    // Get port from environment or use default
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Printf("Health check: http://localhost:%s/health", port)
    log.Printf("API base URL: http://localhost:%s/api", port)
    
    log.Fatal(http.ListenAndServe(":"+port, router))
}
```

## Running and Testing the API

### Start the Server
```bash
go run cmd/api/main.go
```

### Test with curl

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Create a Task
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Learn Go REST APIs",
    "description": "Complete the tutorial and build a task API"
  }'
```

#### Get All Tasks
```bash
curl http://localhost:8080/api/tasks
```

#### Get a Specific Task
```bash
curl http://localhost:8080/api/tasks/{task-id}
```

#### Update a Task
```bash
curl -X PUT http://localhost:8080/api/tasks/{task-id} \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Task Title",
    "completed": true
  }'
```

#### Complete a Task
```bash
curl -X PATCH http://localhost:8080/api/tasks/{task-id}/complete
```

#### Filter Completed Tasks
```bash
curl http://localhost:8080/api/tasks?completed=true
```

#### Delete a Task
```bash
curl -X DELETE http://localhost:8080/api/tasks/{task-id}
```

## Adding Middleware

### Logging Middleware
```go
// Add to main.go before the CORS middleware
router.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
})
```

### Content-Type Validation Middleware
```go
func requireJSON(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" || r.Method == "PUT" {
            if r.Header.Get("Content-Type") != "application/json" {
                http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to API routes
api.Use(requireJSON)
```

## Testing the API

### Basic Unit Tests
```go
// internal/handlers/tasks_test.go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gorilla/mux"
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

func setupTestHandler() *TaskHandler {
    return NewTaskHandler(storage.NewMemoryStorage())
}

func TestCreateTask(t *testing.T) {
    handler := setupTestHandler()
    
    payload := models.CreateTaskRequest{
        Title:       "Test Task",
        Description: "Test Description",
    }
    
    jsonPayload, _ := json.Marshal(payload)
    
    req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonPayload))
    if err != nil {
        t.Fatal(err)
    }
    req.Header.Set("Content-Type", "application/json")
    
    rr := httptest.NewRecorder()
    
    handler.CreateTask(rr, req)
    
    if status := rr.Code; status != http.StatusCreated {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusCreated)
    }
    
    var task models.Task
    if err := json.Unmarshal(rr.Body.Bytes(), &task); err != nil {
        t.Fatal("Could not unmarshal response")
    }
    
    if task.Title != payload.Title {
        t.Errorf("Expected title %v, got %v", payload.Title, task.Title)
    }
}

func TestGetTaskNotFound(t *testing.T) {
    handler := setupTestHandler()
    
    req, err := http.NewRequest("GET", "/api/tasks/nonexistent", nil)
    if err != nil {
        t.Fatal(err)
    }
    
    rr := httptest.NewRecorder()
    
    // We need to set up the mux vars for this test
    req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
    
    handler.GetTask(rr, req)
    
    if status := rr.Code; status != http.StatusNotFound {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusNotFound)
    }
}
```

### Run Tests
```bash
go test ./...
```

## Common Patterns and Best Practices

### 1. Consistent Error Handling
- Use structured error responses
- Include appropriate HTTP status codes
- Log errors for debugging

### 2. Input Validation
- Validate required fields
- Check data types and formats
- Provide clear error messages

### 3. Response Structure
- Use consistent JSON structures
- Include metadata (count, pagination info)
- Follow naming conventions

### 4. HTTP Method Usage
- GET for reading data
- POST for creating resources
- PUT for replacing resources
- PATCH for partial updates
- DELETE for removing resources

### 5. Status Code Usage
- 200 OK for successful GET/PUT/PATCH
- 201 Created for successful POST
- 204 No Content for successful DELETE
- 400 Bad Request for validation errors
- 404 Not Found for missing resources
- 500 Internal Server Error for server issues

## Performance Considerations

### Memory Usage
- Our in-memory storage grows indefinitely
- Consider implementing cleanup or limits
- Use database storage for production

### Concurrent Access
- Our MemoryStorage uses mutex for thread safety
- Consider read-write locks for better performance
- Database solutions handle concurrency better

## Next Steps

In the next lesson, we'll enhance this API with:
- Advanced routing patterns
- Middleware implementation
- Request/response processing
- Better error handling strategies

## Key Takeaways

- REST APIs follow HTTP conventions
- Proper error handling improves user experience
- Consistent structure makes APIs predictable
- Testing ensures reliability
- Middleware adds cross-cutting concerns

## Practice Exercises

1. Add a `GET /api/tasks/stats` endpoint that returns task statistics
2. Implement soft delete (mark as deleted instead of removing)
3. Add validation for maximum title length
4. Create a search endpoint `GET /api/tasks/search?q=query`
5. Add created_at and updated_at filtering