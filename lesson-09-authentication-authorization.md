# Lesson 9: Authentication and Authorization

## Learning Objectives
By the end of this lesson, you will be able to:
- Implement JWT-based authentication
- Create middleware for authentication and authorization
- Handle user registration and login
- Implement role-based access control (RBAC)
- Secure API endpoints appropriately
- Handle API keys for service-to-service communication

## Authentication vs Authorization

### Authentication
**"Who are you?"** - Verifying the identity of a user or service.

Examples:
- Username/password
- JWT tokens
- API keys
- OAuth tokens

### Authorization
**"What can you do?"** - Determining what an authenticated user is allowed to do.

Examples:
- Role-based access (admin, user, guest)
- Permission-based access (read, write, delete)
- Resource ownership (users can only edit their own data)

## JWT (JSON Web Tokens)

### JWT Structure
```
header.payload.signature
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjMiLCJuYW1lIjoiSm9obiIsImV4cCI6MTYwOTQ1OTIwMH0.signature
```

#### Header
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

#### Payload (Claims)
```json
{
  "sub": "123",           // Subject (user ID)
  "name": "John Doe",     // Custom claim
  "role": "user",         // Custom claim
  "exp": 1609459200,      // Expiration time
  "iat": 1609372800       // Issued at time
}
```

### JWT Implementation in Go

#### Install Dependencies
```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
```

#### JWT Service
```go
// internal/auth/jwt.go
package auth

import (
    "errors"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

type JWTService struct {
    secretKey []byte
    issuer    string
}

func NewJWTService(secretKey, issuer string) *JWTService {
    return &JWTService{
        secretKey: []byte(secretKey),
        issuer:    issuer,
    }
}

func (s *JWTService) GenerateToken(userID, email, role string) (string, error) {
    claims := Claims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    s.issuer,
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }
        return s.secretKey, nil
    })
    
    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrExpiredToken
        }
        return nil, ErrInvalidToken
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, ErrInvalidToken
}

func (s *JWTService) RefreshToken(tokenString string) (string, error) {
    claims, err := s.ValidateToken(tokenString)
    if err != nil {
        return "", err
    }
    
    // Generate new token with same claims but new expiration
    return s.GenerateToken(claims.UserID, claims.Email, claims.Role)
}
```

## User Management

### User Model
```go
// internal/models/user.go
package models

import (
    "time"
    "golang.org/x/crypto/bcrypt"
    "github.com/google/uuid"
)

type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Password  string    `json:"-"` // Never include in JSON responses
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Role      string    `json:"role"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type RegisterRequest struct {
    Email     string `json:"email"`
    Password  string `json:"password"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
}

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

const (
    RoleAdmin = "admin"
    RoleUser  = "user"
    RoleGuest = "guest"
)

func NewUser(email, password, firstName, lastName string) (*User, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    
    now := time.Now()
    return &User{
        ID:        uuid.New().String(),
        Email:     email,
        Password:  string(hashedPassword),
        FirstName: firstName,
        LastName:  lastName,
        Role:      RoleUser,
        IsActive:  true,
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}

func (u *User) CheckPassword(password string) error {
    return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

func (u *User) HasRole(role string) bool {
    return u.Role == role
}

func (u *User) IsAdmin() bool {
    return u.Role == RoleAdmin
}
```

### User Storage
```go
// internal/storage/user.go
package storage

import (
    "errors"
    "sync"
    "github.com/yourusername/task-api/internal/models"
)

var (
    ErrUserNotFound    = errors.New("user not found")
    ErrUserExists      = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserStorage interface {
    Create(user *models.User) error
    GetByID(id string) (*models.User, error)
    GetByEmail(email string) (*models.User, error)
    Update(user *models.User) error
    Delete(id string) error
    GetAll() []*models.User
}

type MemoryUserStorage struct {
    users      map[string]*models.User
    emailIndex map[string]string // email -> id mapping
    mutex      sync.RWMutex
}

func NewMemoryUserStorage() *MemoryUserStorage {
    return &MemoryUserStorage{
        users:      make(map[string]*models.User),
        emailIndex: make(map[string]string),
    }
}

func (s *MemoryUserStorage) Create(user *models.User) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    // Check if email already exists
    if _, exists := s.emailIndex[user.Email]; exists {
        return ErrUserExists
    }
    
    s.users[user.ID] = user
    s.emailIndex[user.Email] = user.ID
    return nil
}

func (s *MemoryUserStorage) GetByID(id string) (*models.User, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    user, exists := s.users[id]
    if !exists {
        return nil, ErrUserNotFound
    }
    return user, nil
}

func (s *MemoryUserStorage) GetByEmail(email string) (*models.User, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    userID, exists := s.emailIndex[email]
    if !exists {
        return nil, ErrUserNotFound
    }
    
    user := s.users[userID]
    return user, nil
}

func (s *MemoryUserStorage) Update(user *models.User) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if _, exists := s.users[user.ID]; !exists {
        return ErrUserNotFound
    }
    
    s.users[user.ID] = user
    return nil
}

func (s *MemoryUserStorage) Delete(id string) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    user, exists := s.users[id]
    if !exists {
        return ErrUserNotFound
    }
    
    delete(s.users, id)
    delete(s.emailIndex, user.Email)
    return nil
}

func (s *MemoryUserStorage) GetAll() []*models.User {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    users := make([]*models.User, 0, len(s.users))
    for _, user := range s.users {
        users = append(users, user)
    }
    return users
}
```

## Authentication Handlers

### Auth Handlers
```go
// internal/handlers/auth.go
package handlers

import (
    "encoding/json"
    "net/http"
    "strings"
    
    "github.com/yourusername/task-api/internal/auth"
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

type AuthHandler struct {
    userStorage storage.UserStorage
    jwtService  *auth.JWTService
}

func NewAuthHandler(userStorage storage.UserStorage, jwtService *auth.JWTService) *AuthHandler {
    return &AuthHandler{
        userStorage: userStorage,
        jwtService:  jwtService,
    }
}

func (h *AuthHandler) respondWithError(w http.ResponseWriter, code int, message string) {
    response := map[string]string{"error": message}
    h.respondWithJSON(w, code, response)
}

func (h *AuthHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}

// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req models.RegisterRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    // Validation
    if err := h.validateRegisterRequest(req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Check if user already exists
    _, err := h.userStorage.GetByEmail(req.Email)
    if err == nil {
        h.respondWithError(w, http.StatusConflict, "User already exists")
        return
    }
    
    // Create user
    user, err := models.NewUser(req.Email, req.Password, req.FirstName, req.LastName)
    if err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to create user")
        return
    }
    
    if err := h.userStorage.Create(user); err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to save user")
        return
    }
    
    // Generate token
    token, err := h.jwtService.GenerateToken(user.ID, user.Email, user.Role)
    if err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }
    
    response := models.LoginResponse{
        Token: token,
        User:  *user,
    }
    
    h.respondWithJSON(w, http.StatusCreated, response)
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req models.LoginRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    // Validation
    if req.Email == "" || req.Password == "" {
        h.respondWithError(w, http.StatusBadRequest, "Email and password are required")
        return
    }
    
    // Get user by email
    user, err := h.userStorage.GetByEmail(req.Email)
    if err != nil {
        h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
        return
    }
    
    // Check if user is active
    if !user.IsActive {
        h.respondWithError(w, http.StatusUnauthorized, "Account is disabled")
        return
    }
    
    // Verify password
    if err := user.CheckPassword(req.Password); err != nil {
        h.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
        return
    }
    
    // Generate token
    token, err := h.jwtService.GenerateToken(user.ID, user.Email, user.Role)
    if err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }
    
    response := models.LoginResponse{
        Token: token,
        User:  *user,
    }
    
    h.respondWithJSON(w, http.StatusOK, response)
}

// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
    tokenString := h.extractTokenFromHeader(r)
    if tokenString == "" {
        h.respondWithError(w, http.StatusUnauthorized, "Token required")
        return
    }
    
    newToken, err := h.jwtService.RefreshToken(tokenString)
    if err != nil {
        h.respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
        return
    }
    
    response := map[string]string{"token": newToken}
    h.respondWithJSON(w, http.StatusOK, response)
}

// GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
    // Get user from context (set by auth middleware)
    userID := r.Context().Value("user_id").(string)
    
    user, err := h.userStorage.GetByID(userID)
    if err != nil {
        h.respondWithError(w, http.StatusNotFound, "User not found")
        return
    }
    
    h.respondWithJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) extractTokenFromHeader(r *http.Request) string {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return ""
    }
    
    // Bearer token format: "Bearer <token>"
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || parts[0] != "Bearer" {
        return ""
    }
    
    return parts[1]
}

func (h *AuthHandler) validateRegisterRequest(req models.RegisterRequest) error {
    if req.Email == "" {
        return errors.New("email is required")
    }
    if req.Password == "" {
        return errors.New("password is required")
    }
    if len(req.Password) < 6 {
        return errors.New("password must be at least 6 characters")
    }
    if req.FirstName == "" {
        return errors.New("first name is required")
    }
    if req.LastName == "" {
        return errors.New("last name is required")
    }
    return nil
}
```

## Authentication Middleware

### JWT Middleware
```go
// internal/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/yourusername/task-api/internal/auth"
)

type AuthMiddleware struct {
    jwtService *auth.JWTService
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
    return &AuthMiddleware{jwtService: jwtService}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tokenString := m.extractTokenFromHeader(r)
        if tokenString == "" {
            http.Error(w, "Authorization token required", http.StatusUnauthorized)
            return
        }
        
        claims, err := m.jwtService.ValidateToken(tokenString)
        if err != nil {
            http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
            return
        }
        
        // Add user information to request context
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "user_email", claims.Email)
        ctx = context.WithValue(ctx, "user_role", claims.Role)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userRole := r.Context().Value("user_role")
            if userRole == nil || userRole.(string) != role {
                http.Error(w, "Insufficient permissions", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func (m *AuthMiddleware) RequireOwnershipOrAdmin(getResourceOwnerID func(*http.Request) string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Context().Value("user_id").(string)
            userRole := r.Context().Value("user_role").(string)
            
            // Admin can access everything
            if userRole == models.RoleAdmin {
                next.ServeHTTP(w, r)
                return
            }
            
            // Check ownership
            resourceOwnerID := getResourceOwnerID(r)
            if resourceOwnerID != userID {
                http.Error(w, "Access denied", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

func (m *AuthMiddleware) extractTokenFromHeader(r *http.Request) string {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return ""
    }
    
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || parts[0] != "Bearer" {
        return ""
    }
    
    return parts[1]
}
```

## Updating Task Handlers for Authorization

### Add User Context to Tasks
```go
// Update internal/models/task.go
type Task struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Completed   bool      `json:"completed"`
    UserID      string    `json:"user_id"`  // Add this field
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

func NewTask(title, description, userID string) *Task {
    now := time.Now()
    return &Task{
        ID:          uuid.New().String(),
        Title:       title,
        Description: description,
        UserID:      userID,  // Set user ID
        Completed:   false,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
}
```

### Update Task Handlers
```go
// Update internal/handlers/tasks.go
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req models.CreateTaskRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    if req.Title == "" {
        h.respondWithError(w, http.StatusBadRequest, "Title is required")
        return
    }
    
    // Get user ID from context
    userID := r.Context().Value("user_id").(string)
    
    task := models.NewTask(req.Title, req.Description, userID)
    
    if err := h.storage.Create(task); err != nil {
        h.respondWithError(w, http.StatusInternalServerError, "Failed to create task")
        return
    }
    
    h.respondWithJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)
    userRole := r.Context().Value("user_role").(string)
    
    var tasks []*models.Task
    
    // Admin can see all tasks, users can only see their own
    if userRole == models.RoleAdmin {
        tasks = h.storage.GetAll()
    } else {
        tasks = h.storage.GetByUserID(userID)
    }
    
    // Apply completed filter if specified
    completedParam := r.URL.Query().Get("completed")
    if completedParam != "" {
        completed, err := strconv.ParseBool(completedParam)
        if err != nil {
            h.respondWithError(w, http.StatusBadRequest, "Invalid 'completed' parameter")
            return
        }
        tasks = h.filterByCompleted(tasks, completed)
    }
    
    taskList := make([]models.Task, len(tasks))
    for i, task := range tasks {
        taskList[i] = *task
    }
    
    h.respondWithJSON(w, http.StatusOK, TaskListResponse{
        Tasks: taskList,
        Count: len(taskList),
    })
}
```

## API Keys for Service-to-Service Authentication

### API Key Model
```go
// internal/models/apikey.go
package models

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

type APIKey struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Key         string    `json:"key"`
    UserID      string    `json:"user_id"`
    Permissions []string  `json:"permissions"`
    IsActive    bool      `json:"is_active"`
    LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

func GenerateAPIKey(name, userID string, permissions []string) (*APIKey, error) {
    // Generate random 32-byte key
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return nil, err
    }
    
    key := hex.EncodeToString(bytes)
    
    return &APIKey{
        ID:          uuid.New().String(),
        Name:        name,
        Key:         key,
        UserID:      userID,
        Permissions: permissions,
        IsActive:    true,
        CreatedAt:   time.Now(),
    }, nil
}
```

### API Key Middleware
```go
// internal/middleware/apikey.go
package middleware

import (
    "context"
    "net/http"
    
    "github.com/yourusername/task-api/internal/storage"
)

type APIKeyMiddleware struct {
    apiKeyStorage storage.APIKeyStorage
}

func NewAPIKeyMiddleware(apiKeyStorage storage.APIKeyStorage) *APIKeyMiddleware {
    return &APIKeyMiddleware{apiKeyStorage: apiKeyStorage}
}

func (m *APIKeyMiddleware) RequireAPIKey(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            http.Error(w, "API key required", http.StatusUnauthorized)
            return
        }
        
        key, err := m.apiKeyStorage.GetByKey(apiKey)
        if err != nil {
            http.Error(w, "Invalid API key", http.StatusUnauthorized)
            return
        }
        
        if !key.IsActive {
            http.Error(w, "API key is disabled", http.StatusUnauthorized)
            return
        }
        
        // Check expiration
        if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
            http.Error(w, "API key has expired", http.StatusUnauthorized)
            return
        }
        
        // Update last used time
        now := time.Now()
        key.LastUsedAt = &now
        m.apiKeyStorage.Update(key)
        
        // Add API key context
        ctx := context.WithValue(r.Context(), "api_key_id", key.ID)
        ctx = context.WithValue(ctx, "api_key_user_id", key.UserID)
        ctx = context.WithValue(ctx, "api_key_permissions", key.Permissions)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Integrating Authentication in Main Application

### Updated Main Application
```go
// cmd/api/main.go
package main

import (
    "log"
    "net/http"
    "os"
    
    "github.com/gorilla/mux"
    "github.com/yourusername/task-api/internal/auth"
    "github.com/yourusername/task-api/internal/handlers"
    "github.com/yourusername/task-api/internal/middleware"
    "github.com/yourusername/task-api/internal/storage"
)

func main() {
    // Initialize services
    jwtService := auth.NewJWTService(
        os.Getenv("JWT_SECRET"),
        "task-api",
    )
    
    // Initialize storage
    userStorage := storage.NewMemoryUserStorage()
    taskStorage := storage.NewMemoryTaskStorage()
    
    // Initialize handlers
    authHandler := handlers.NewAuthHandler(userStorage, jwtService)
    taskHandler := handlers.NewTaskHandler(taskStorage)
    
    // Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(jwtService)
    
    // Initialize router
    router := mux.NewRouter()
    api := router.PathPrefix("/api").Subrouter()
    
    // Public routes
    api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
    api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
    api.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST")
    
    // Protected routes
    protected := api.PathPrefix("").Subrouter()
    protected.Use(authMiddleware.RequireAuth)
    
    // User routes
    protected.HandleFunc("/auth/me", authHandler.GetCurrentUser).Methods("GET")
    
    // Task routes (require authentication)
    protected.HandleFunc("/tasks", taskHandler.GetTasks).Methods("GET")
    protected.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
    protected.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
    protected.HandleFunc("/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
    protected.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
    
    // Admin-only routes
    adminOnly := protected.PathPrefix("").Subrouter()
    adminOnly.Use(authMiddleware.RequireRole(models.RoleAdmin))
    adminOnly.HandleFunc("/admin/users", userHandler.GetAllUsers).Methods("GET")
    
    // Health check (public)
    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "ok"}`))
    }).Methods("GET")
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, router))
}
```

## Testing Authentication

### Register a User
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "password123"
  }'
```

### Access Protected Endpoint
```bash
curl -X GET http://localhost:8080/api/tasks \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Create Task (Authenticated)
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Authenticated Task",
    "description": "This task requires authentication"
  }'
```

## Security Best Practices

### 1. Token Security
- Use strong, random secret keys
- Set appropriate token expiration times
- Implement token refresh mechanism
- Store secrets in environment variables

### 2. Password Security
- Use bcrypt for password hashing
- Enforce minimum password requirements
- Implement account lockout after failed attempts
- Consider password strength validation

### 3. API Security
- Always use HTTPS in production
- Implement rate limiting
- Validate all input data
- Log authentication events
- Use CORS appropriately

### 4. Authorization
- Follow principle of least privilege
- Implement proper role-based access control
- Check ownership for resource-specific operations
- Regularly audit permissions

## Common Pitfalls

1. **Storing passwords in plain text**: Always hash passwords
2. **Weak JWT secrets**: Use strong, random secrets
3. **No token expiration**: Always set expiration times
4. **Missing authorization checks**: Check permissions on every protected endpoint
5. **Logging sensitive data**: Never log passwords or tokens

## Next Steps

In the next lesson, we'll cover error handling and validation strategies to make your API more robust and user-friendly.

## Key Takeaways

- JWT provides stateless authentication
- Middleware handles cross-cutting authentication concerns
- Role-based access control enables fine-grained permissions
- API keys are useful for service-to-service communication
- Security should be implemented at multiple layers

## Practice Exercises

1. Implement password reset functionality
2. Add email verification for new accounts
3. Create an admin dashboard endpoint
4. Implement API key management endpoints
5. Add two-factor authentication support