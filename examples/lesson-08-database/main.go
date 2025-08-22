package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Configuration
type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	Environment string
}

func loadConfig() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable"),
		Port:        getEnv("PORT", "8088"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
		Environment: getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Models
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Role          string    `json:"role"`
	IsActive      bool      `json:"isActive"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Completed   bool       `json:"completed"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"dueDate"`
	UserID      string     `json:"userId"`
	Categories  []Category `json:"categories"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Request/Response Types
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateTaskRequest struct {
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Priority      string     `json:"priority"`
	DueDate       *time.Time `json:"dueDate"`
	CategoryNames []string   `json:"categoryNames"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Completed   *bool      `json:"completed"`
	Priority    *string    `json:"priority"`
	DueDate     *time.Time `json:"dueDate"`
}

type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	Count      int    `json:"count"`
	TotalCount int64  `json:"totalCount"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// Database
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
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db}, nil
}

func (db *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// Repository Interfaces
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	GetByUserID(ctx context.Context, userID string, filters TaskFilters) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context, userID string, filters TaskFilters) (int64, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	GetByUserID(ctx context.Context, userID string) ([]*Category, error)
	GetByName(ctx context.Context, name, userID string) (*Category, error)
}

type TaskFilters struct {
	Completed   *bool
	Priority    string
	Search      string
	DueBefore   *time.Time
	DueAfter    *time.Time
	CategoryIDs []string
	Limit       int
	Offset      int
}

// Repository Implementations
type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.Role, user.IsActive, user.EmailVerified,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("user with email %s already exists", user.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, 
		       is_active, email_verified, created_at, updated_at
		FROM users WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, 
		       is_active, email_verified, created_at, updated_at
		FROM users WHERE email = $1`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users 
		SET email = $2, first_name = $3, last_name = $4, role = $5, 
		    is_active = $6, email_verified = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.FirstName, user.LastName,
		user.Role, user.IsActive, user.EmailVerified,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

type taskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, title, description, completed, priority, due_date, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		task.ID, task.Title, task.Description, task.Completed,
		task.Priority, task.DueDate, task.UserID,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	task := &Task{}
	query := `
		SELECT t.id, t.title, t.description, t.completed, t.priority, 
		       t.due_date, t.user_id, t.created_at, t.updated_at,
		       COALESCE(array_agg(c.id) FILTER (WHERE c.id IS NOT NULL), '{}') as category_ids,
		       COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as category_names,
		       COALESCE(array_agg(c.color) FILTER (WHERE c.color IS NOT NULL), '{}') as category_colors
		FROM tasks t
		LEFT JOIN task_categories tc ON t.id = tc.task_id
		LEFT JOIN categories c ON tc.category_id = c.id
		WHERE t.id = $1
		GROUP BY t.id`

	var categoryIDs, categoryNames, categoryColors pq.StringArray
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Completed, &task.Priority,
		&task.DueDate, &task.UserID, &task.CreatedAt, &task.UpdatedAt,
		&categoryIDs, &categoryNames, &categoryColors,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Convert arrays to categories
	for i, id := range categoryIDs {
		if id != "" && i < len(categoryNames) {
			color := ""
			if i < len(categoryColors) {
				color = categoryColors[i]
			}
			task.Categories = append(task.Categories, Category{
				ID:    id,
				Name:  categoryNames[i],
				Color: color,
			})
		}
	}

	return task, nil
}

func (r *taskRepository) GetByUserID(ctx context.Context, userID string, filters TaskFilters) ([]*Task, error) {
	var conditions []string
	var args []interface{}
	argIndex := 2 // Start from 2 since $1 is userID

	baseQuery := `
		SELECT t.id, t.title, t.description, t.completed, t.priority, 
		       t.due_date, t.user_id, t.created_at, t.updated_at,
		       COALESCE(array_agg(c.id) FILTER (WHERE c.id IS NOT NULL), '{}') as category_ids,
		       COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as category_names,
		       COALESCE(array_agg(c.color) FILTER (WHERE c.color IS NOT NULL), '{}') as category_colors
		FROM tasks t
		LEFT JOIN task_categories tc ON t.id = tc.task_id
		LEFT JOIN categories c ON tc.category_id = c.id
		WHERE t.user_id = $1`

	args = append(args, userID)

	// Apply filters
	if filters.Completed != nil {
		conditions = append(conditions, fmt.Sprintf("t.completed = $%d", argIndex))
		args = append(args, *filters.Completed)
		argIndex++
	}

	if filters.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("t.priority = $%d", argIndex))
		args = append(args, filters.Priority)
		argIndex++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(t.title ILIKE $%d OR t.description ILIKE $%d)", argIndex, argIndex+1))
		searchTerm := "%" + filters.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	query := baseQuery + `
		GROUP BY t.id, t.title, t.description, t.completed, t.priority, 
		         t.due_date, t.user_id, t.created_at, t.updated_at
		ORDER BY t.created_at DESC`

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var categoryIDs, categoryNames, categoryColors pq.StringArray

		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Completed, &task.Priority,
			&task.DueDate, &task.UserID, &task.CreatedAt, &task.UpdatedAt,
			&categoryIDs, &categoryNames, &categoryColors,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		// Convert arrays to categories
		for i, id := range categoryIDs {
			if id != "" && i < len(categoryNames) {
				color := ""
				if i < len(categoryColors) {
					color = categoryColors[i]
				}
				task.Categories = append(task.Categories, Category{
					ID:    id,
					Name:  categoryNames[i],
					Color: color,
				})
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (r *taskRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks 
		SET title = $2, description = $3, completed = $4, priority = $5, 
		    due_date = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		task.ID, task.Title, task.Description, task.Completed,
		task.Priority, task.DueDate,
	).Scan(&task.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
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

func (r *taskRepository) Count(ctx context.Context, userID string, filters TaskFilters) (int64, error) {
	var conditions []string
	var args []interface{}
	argIndex := 2

	query := `SELECT COUNT(*) FROM tasks WHERE user_id = $1`
	args = append(args, userID)

	if filters.Completed != nil {
		conditions = append(conditions, fmt.Sprintf("completed = $%d", argIndex))
		args = append(args, *filters.Completed)
		argIndex++
	}

	if filters.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, filters.Priority)
		argIndex++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1))
		searchTerm := "%" + filters.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

type categoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *Category) error {
	query := `
		INSERT INTO categories (id, name, color, user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		category.ID, category.Name, category.Color, category.UserID,
	).Scan(&category.CreatedAt, &category.UpdatedAt)
}

func (r *categoryRepository) GetByUserID(ctx context.Context, userID string) ([]*Category, error) {
	query := `
		SELECT id, name, color, user_id, created_at, updated_at
		FROM categories WHERE user_id = $1 ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		category := &Category{}
		err := rows.Scan(
			&category.ID, &category.Name, &category.Color,
			&category.UserID, &category.CreatedAt, &category.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

func (r *categoryRepository) GetByName(ctx context.Context, name, userID string) (*Category, error) {
	category := &Category{}
	query := `
		SELECT id, name, color, user_id, created_at, updated_at
		FROM categories WHERE name = $1 AND user_id = $2`

	err := r.db.QueryRowContext(ctx, query, name, userID).Scan(
		&category.ID, &category.Name, &category.Color,
		&category.UserID, &category.CreatedAt, &category.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, err
	}

	return category, nil
}

// JWT Service
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

func (j *JWTService) GenerateToken(user *User) (string, error) {
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Transaction Manager
func WithTransaction(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	databaseConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(databaseConnectionsActive)
}

// Service Layer
type TaskService struct {
	taskRepo     TaskRepository
	categoryRepo CategoryRepository
	db           *sql.DB
}

func NewTaskService(taskRepo TaskRepository, categoryRepo CategoryRepository, db *sql.DB) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		categoryRepo: categoryRepo,
		db:           db,
	}
}

func (s *TaskService) CreateTaskWithCategories(ctx context.Context, req CreateTaskRequest, userID string) (*Task, error) {
	var task *Task

	err := WithTransaction(s.db, func(tx *sql.Tx) error {
		// Create task
		task = &Task{
			ID:          uuid.New().String(),
			Title:       req.Title,
			Description: req.Description,
			Priority:    req.Priority,
			DueDate:     req.DueDate,
			UserID:      userID,
			Completed:   false,
		}

		if err := s.taskRepo.Create(ctx, task); err != nil {
			return err
		}

		// Handle categories
		for _, categoryName := range req.CategoryNames {
			// Try to get existing category
			category, err := s.categoryRepo.GetByName(ctx, categoryName, userID)
			if err != nil {
				// Create new category
				category = &Category{
					ID:     uuid.New().String(),
					Name:   categoryName,
					UserID: userID,
					Color:  "#3B82F6", // Default blue color
				}
				if err := s.categoryRepo.Create(ctx, category); err != nil {
					return err
				}
			}

			// Link task to category
			_, err = tx.ExecContext(ctx,
				"INSERT INTO task_categories (task_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				task.ID, category.ID)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return task with categories
	return s.taskRepo.GetByID(ctx, task.ID)
}

// Handlers
type Handler struct {
	userRepo     UserRepository
	taskRepo     TaskRepository
	categoryRepo CategoryRepository
	taskService  *TaskService
	jwtService   *JWTService
	db           *Database
}

func NewHandler(db *Database, jwtService *JWTService) *Handler {
	userRepo := NewUserRepository(db.DB)
	taskRepo := NewTaskRepository(db.DB)
	categoryRepo := NewCategoryRepository(db.DB)
	taskService := NewTaskService(taskRepo, categoryRepo, db.DB)

	return &Handler{
		userRepo:     userRepo,
		taskRepo:     taskRepo,
		categoryRepo: categoryRepo,
		taskService:  taskService,
		jwtService:   jwtService,
		db:           db,
	}
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, ErrorResponse{
		Error:     http.StatusText(code),
		Message:   message,
		RequestID: uuid.New().String()[:8],
	})
}

// Auth Handlers
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		h.respondWithError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	user := &User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Role:          "user",
		IsActive:      true,
		EmailVerified: false,
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.respondWithError(w, http.StatusConflict, "User with this email already exists")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, LoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Get user by email
	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check if user is active
	if !user.IsActive {
		h.respondWithError(w, http.StatusUnauthorized, "Account is disabled")
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	h.respondWithJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  *user,
	})
}

// Task Handlers
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	// Parse query parameters
	query := r.URL.Query()
	filters := TaskFilters{
		Search: query.Get("search"),
		Limit:  10,
		Offset: 0,
	}

	if completed := query.Get("completed"); completed != "" {
		if c, err := strconv.ParseBool(completed); err == nil {
			filters.Completed = &c
		}
	}

	if priority := query.Get("priority"); priority != "" {
		filters.Priority = priority
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filters.Limit = l
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	// Get tasks and count
	tasks, err := h.taskRepo.GetByUserID(r.Context(), userID, filters)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get tasks")
		return
	}

	totalCount, err := h.taskRepo.Count(r.Context(), userID, filters)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to count tasks")
		return
	}

	// Convert to response format
	taskList := make([]Task, len(tasks))
	for i, task := range tasks {
		taskList[i] = *task
	}

	response := TaskListResponse{
		Tasks:      taskList,
		Count:      len(taskList),
		TotalCount: totalCount,
		Page:       filters.Offset/filters.Limit + 1,
		Limit:      filters.Limit,
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate
	if req.Title == "" {
		h.respondWithError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if req.Priority == "" {
		req.Priority = "medium"
	}

	// Create task with categories
	task, err := h.taskService.CreateTaskWithCategories(r.Context(), req, userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	h.respondWithJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.taskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get task")
		return
	}

	// Check ownership
	if task.UserID != userID {
		h.respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	h.respondWithJSON(w, http.StatusOK, task)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Get existing task
	task, err := h.taskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get task")
		return
	}

	// Check ownership
	if task.UserID != userID {
		h.respondWithError(w, http.StatusForbidden, "Access denied")
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

	if req.Priority != nil {
		task.Priority = *req.Priority
	}

	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	// Update task
	if err := h.taskRepo.Update(r.Context(), task); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	// Return updated task with categories
	updatedTask, err := h.taskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get updated task")
		return
	}

	h.respondWithJSON(w, http.StatusOK, updatedTask)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Get task to check ownership
	task, err := h.taskRepo.GetByID(r.Context(), taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get task")
		return
	}

	// Check ownership
	if task.UserID != userID {
		h.respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete task
	if err := h.taskRepo.Delete(r.Context(), taskID); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Category Handlers
func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	categories, err := h.categoryRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get categories")
		return
	}

	// Convert to response format
	categoryList := make([]Category, len(categories))
	for i, category := range categories {
		categoryList[i] = *category
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"categories": categoryList,
		"count":      len(categoryList),
	})
}

// Health Check Handler
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "task-api",
		"version":   "1.0.0",
	}

	// Check database health
	if err := h.db.HealthCheck(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		h.respondWithJSON(w, http.StatusServiceUnavailable, health)
		return
	}

	health["database"] = map[string]interface{}{
		"status": "healthy",
		"stats":  h.db.Stats(),
	}

	h.respondWithJSON(w, http.StatusOK, health)
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(ww.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func authMiddleware(jwtService *JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Bearer token required", http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_email", claims.Email)
			ctx = context.WithValue(ctx, "user_role", claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func updateDatabaseMetrics(db *Database) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := db.Stats()
			databaseConnectionsActive.Set(float64(stats.OpenConnections))
		}
	}()
}

func main() {
	config := loadConfig()

	// Initialize database
	db, err := NewDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize JWT service
	jwtService := NewJWTService(config.JWTSecret)

	// Initialize handler
	handler := NewHandler(db, jwtService)

	// Start metrics updater
	updateDatabaseMetrics(db)

	// Setup routes
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)
	router.Use(metricsMiddleware)

	// Health check
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Auth routes (public)
	api.HandleFunc("/auth/register", handler.Register).Methods("POST")
	api.HandleFunc("/auth/login", handler.Login).Methods("POST")

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware(jwtService))

	// Task routes
	protected.HandleFunc("/tasks", handler.GetTasks).Methods("GET")
	protected.HandleFunc("/tasks", handler.CreateTask).Methods("POST")
	protected.HandleFunc("/tasks/{id}", handler.GetTask).Methods("GET")
	protected.HandleFunc("/tasks/{id}", handler.UpdateTask).Methods("PUT")
	protected.HandleFunc("/tasks/{id}", handler.DeleteTask).Methods("DELETE")

	// Category routes
	protected.HandleFunc("/categories", handler.GetCategories).Methods("GET")

	// Create server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("🚀 Database-Integrated Task API")
		log.Printf("Server starting on port %s", config.Port)
		log.Printf("Environment: %s", config.Environment)
		log.Printf("Health check: http://localhost:%s/health", config.Port)
		log.Printf("Metrics: http://localhost:%s/metrics", config.Port)
		log.Printf("API docs: http://localhost:%s/api", config.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server shutdown complete")
}