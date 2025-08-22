package main

import (
	"context"
	"database/sql"
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
	_ "github.com/lib/pq"
)

// Configuration
type Config struct {
	DatabaseURL string
	Port        string
	Environment string
}

func loadConfig() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Task represents a task item
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Completed   *bool   `json:"completed"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Database wrapper
type Database struct {
	*sql.DB
}

func NewDatabase(databaseURL string) (*Database, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db}, nil
}

func (db *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id UUID PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		completed BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Create function to update updated_at timestamp
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	-- Create trigger if it doesn't exist
	DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
	CREATE TRIGGER update_tasks_updated_at 
		BEFORE UPDATE ON tasks 
		FOR EACH ROW 
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(schema)
	return err
}

// TaskRepository interface
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetAll(ctx context.Context) ([]*Task, error)
	GetByID(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
}

// PostgreSQL implementation
type postgresTaskRepository struct {
	db *sql.DB
}

func NewPostgresTaskRepository(db *sql.DB) TaskRepository {
	return &postgresTaskRepository{db: db}
}

func (r *postgresTaskRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, title, description, completed)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		task.ID, task.Title, task.Description, task.Completed,
	).Scan(&task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *postgresTaskRepository) GetAll(ctx context.Context) ([]*Task, error) {
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Completed,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tasks, nil
}

func (r *postgresTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	task := &Task{}
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM tasks
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Completed,
		&task.CreatedAt, &task.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (r *postgresTaskRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks 
		SET title = $2, description = $3, completed = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		task.ID, task.Title, task.Description, task.Completed,
	).Scan(&task.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (r *postgresTaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Handler struct
type Handler struct {
	taskRepo TaskRepository
}

func NewHandler(taskRepo TaskRepository) *Handler {
	return &Handler{taskRepo: taskRepo}
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
	})
}

// Task handlers
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.taskRepo.GetAll(r.Context())
	if err != nil {
		log.Printf("Error getting tasks: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate input
	if req.Title == "" {
		h.respondWithError(w, http.StatusBadRequest, "Title is required")
		return
	}

	// Create task
	task := &Task{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
	}

	if err := h.taskRepo.Create(r.Context(), task); err != nil {
		log.Printf("Error creating task: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	task, err := h.taskRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		log.Printf("Error getting task: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve task")
		return
	}

	h.respondWithJSON(w, http.StatusOK, task)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing task
	task, err := h.taskRepo.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		log.Printf("Error getting task: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve task")
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Apply updates
	if req.Title != nil {
		if *req.Title == "" {
			h.respondWithError(w, http.StatusBadRequest, "Title cannot be empty")
			return
		}
		task.Title = *req.Title
	}

	if req.Description != nil {
		task.Description = *req.Description
	}

	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	if err := h.taskRepo.Update(r.Context(), task); err != nil {
		log.Printf("Error updating task: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	h.respondWithJSON(w, http.StatusOK, task)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.taskRepo.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		log.Printf("Error deleting task: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Health check handler
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "task-api-database",
	})
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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

func main() {
	config := loadConfig()

	// Initialize database
	db, err := NewDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.initSchema(); err != nil {
		log.Fatal("Failed to initialize schema:", err)
	}

	// Initialize repository and handler
	taskRepo := NewPostgresTaskRepository(db.DB)
	handler := NewHandler(taskRepo)

	// Setup routes
	router := mux.NewRouter()

	// Apply middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	// Health check
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/tasks", handler.GetTasks).Methods("GET")
	api.HandleFunc("/tasks", handler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", handler.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", handler.UpdateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", handler.DeleteTask).Methods("DELETE")

	// Start server
	log.Printf("🚀 Database-backed Task API")
	log.Printf("Server starting on port %s", config.Port)
	log.Printf("Environment: %s", config.Environment)
	log.Printf("Health check: http://localhost:%s/health", config.Port)

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}