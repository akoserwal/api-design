package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration
var testConfig = Config{
	DatabaseURL: "postgres://taskuser:taskpass@localhost:5432/taskapi_test?sslmode=disable",
	Port:        "8089",
	JWTSecret:   "test-secret-key",
	Environment: "test",
}

// Global test database and handler
var testDB *Database
var testHandler *Handler

func TestMain(m *testing.M) {
	// Setup test database
	db, err := NewDatabase(testConfig.DatabaseURL)
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		fmt.Println("Make sure PostgreSQL is running with test database 'taskapi_test'")
		os.Exit(1)
	}
	testDB = db
	defer testDB.Close()

	// Initialize handler
	jwtService := NewJWTService(testConfig.JWTSecret)
	testHandler = NewHandler(testDB, jwtService)

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestData()
	os.Exit(code)
}

func cleanupTestData() {
	ctx := context.Background()
	testDB.ExecContext(ctx, "DELETE FROM task_categories")
	testDB.ExecContext(ctx, "DELETE FROM tasks")
	testDB.ExecContext(ctx, "DELETE FROM categories")
	testDB.ExecContext(ctx, "DELETE FROM users")
}

func TestDatabaseConnection(t *testing.T) {
	err := testDB.HealthCheck()
	assert.NoError(t, err, "Database should be accessible")

	stats := testDB.Stats()
	assert.Greater(t, stats.MaxOpenConnections, 0, "Connection pool should be configured")
}

func TestUserRegistrationFlow(t *testing.T) {
	cleanupTestData()

	// Test user registration
	regReq := RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	body, _ := json.Marshal(regReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	testHandler.Register(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token)
	assert.Equal(t, regReq.Email, response.User.Email)
	assert.Equal(t, regReq.FirstName, response.User.FirstName)

	// Test duplicate registration
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	testHandler.Register(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestLoginFlow(t *testing.T) {
	cleanupTestData()

	// Create test user first
	userRepo := NewUserRepository(testDB.DB)
	user := &User{
		ID:           "test-user-id",
		Email:        "login@example.com",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye",
		FirstName:    "Login",
		LastName:     "Test",
		Role:         "user",
		IsActive:     true,
	}
	err := userRepo.Create(context.Background(), user)
	require.NoError(t, err)

	// Test successful login
	loginReq := LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	testHandler.Login(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test invalid credentials
	loginReq.Password = "wrongpassword"
	body, _ = json.Marshal(loginReq)
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	testHandler.Login(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestTaskCRUDOperations(t *testing.T) {
	cleanupTestData()

	// Create test user and get token
	token := createTestUserAndGetToken(t, "crud@example.com")

	// Test create task
	createReq := CreateTaskRequest{
		Title:       "Integration Test Task",
		Description: "Testing CRUD operations",
		Priority:    "high",
		DueDate:     timePtr(time.Now().Add(24 * time.Hour)),
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.CreateTask(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createdTask Task
	err := json.Unmarshal(w.Body.Bytes(), &createdTask)
	require.NoError(t, err)
	assert.Equal(t, createReq.Title, createdTask.Title)
	taskID := createdTask.ID

	// Test get task
	req2 := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()

	testHandler.GetTask(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Test update task
	updateReq := UpdateTaskRequest{
		Title:     stringPtr("Updated Task Title"),
		Completed: boolPtr(true),
	}

	body, _ = json.Marshal(updateReq)
	req3 := httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, bytes.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()

	testHandler.UpdateTask(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	var updatedTask Task
	err = json.Unmarshal(w3.Body.Bytes(), &updatedTask)
	require.NoError(t, err)
	assert.Equal(t, "Updated Task Title", updatedTask.Title)
	assert.True(t, updatedTask.Completed)

	// Test delete task
	req4 := httptest.NewRequest(http.MethodDelete, "/api/tasks/"+taskID, nil)
	req4.Header.Set("Authorization", "Bearer "+token)
	w4 := httptest.NewRecorder()

	testHandler.DeleteTask(w4, req4)
	assert.Equal(t, http.StatusNoContent, w4.Code)

	// Verify task is deleted
	req5 := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	req5.Header.Set("Authorization", "Bearer "+token)
	w5 := httptest.NewRecorder()

	testHandler.GetTask(w5, req5)
	assert.Equal(t, http.StatusNotFound, w5.Code)
}

func TestTaskFiltering(t *testing.T) {
	cleanupTestData()

	token := createTestUserAndGetToken(t, "filter@example.com")

	// Create multiple tasks
	tasks := []CreateTaskRequest{
		{Title: "High Priority Task", Priority: "high", Description: "Important work"},
		{Title: "Medium Priority Task", Priority: "medium", Description: "Regular work"},
		{Title: "Completed Task", Priority: "low", Description: "Done work"},
	}

	var taskIDs []string
	for _, task := range tasks {
		body, _ := json.Marshal(task)
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		testHandler.CreateTask(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var createdTask Task
		json.Unmarshal(w.Body.Bytes(), &createdTask)
		taskIDs = append(taskIDs, createdTask.ID)
	}

	// Mark third task as completed
	updateReq := UpdateTaskRequest{Completed: boolPtr(true)}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskIDs[2], bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	testHandler.UpdateTask(w, req)

	// Test filter by priority
	req2 := httptest.NewRequest(http.MethodGet, "/api/tasks?priority=high", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()

	testHandler.GetTasks(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var response TaskListResponse
	json.Unmarshal(w2.Body.Bytes(), &response)
	assert.Equal(t, 1, response.Count)
	assert.Equal(t, "High Priority Task", response.Tasks[0].Title)

	// Test filter by completion status
	req3 := httptest.NewRequest(http.MethodGet, "/api/tasks?completed=true", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()

	testHandler.GetTasks(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	json.Unmarshal(w3.Body.Bytes(), &response)
	assert.Equal(t, 1, response.Count)
	assert.True(t, response.Tasks[0].Completed)

	// Test search functionality
	req4 := httptest.NewRequest(http.MethodGet, "/api/tasks?search=work", nil)
	req4.Header.Set("Authorization", "Bearer "+token)
	w4 := httptest.NewRecorder()

	testHandler.GetTasks(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	json.Unmarshal(w4.Body.Bytes(), &response)
	assert.Equal(t, 3, response.Count) // All tasks contain "work" in description
}

func TestTransactionIntegrity(t *testing.T) {
	cleanupTestData()

	token := createTestUserAndGetToken(t, "transaction@example.com")

	// Create task with categories (tests transaction)
	createReq := CreateTaskRequest{
		Title:         "Transaction Test",
		Description:   "Testing transaction rollback",
		Priority:      "medium",
		CategoryNames: []string{"Work", "Testing", "Database"},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.CreateTask(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createdTask Task
	json.Unmarshal(w.Body.Bytes(), &createdTask)

	// Verify categories were created
	assert.Len(t, createdTask.Categories, 3)

	// Verify categories exist in database
	req2 := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()

	testHandler.GetCategories(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var categoryResponse map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &categoryResponse)
	categories := categoryResponse["categories"].([]interface{})
	assert.Len(t, categories, 3)
}

func TestDatabaseConstraints(t *testing.T) {
	cleanupTestData()

	// Test foreign key constraint
	taskRepo := NewTaskRepository(testDB.DB)
	task := &Task{
		ID:          "test-task",
		Title:       "Invalid User Task",
		Description: "This should fail",
		UserID:      "non-existent-user",
		Priority:    "medium",
	}

	err := taskRepo.Create(context.Background(), task)
	assert.Error(t, err, "Should fail due to foreign key constraint")
}

func TestConcurrentAccess(t *testing.T) {
	cleanupTestData()

	token := createTestUserAndGetToken(t, "concurrent@example.com")

	// Test concurrent task creation
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			createReq := CreateTaskRequest{
				Title:       fmt.Sprintf("Concurrent Task %d", index),
				Description: "Testing concurrent access",
				Priority:    "medium",
			}

			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			testHandler.CreateTask(w, req)
			if w.Code != http.StatusCreated {
				results <- fmt.Errorf("failed to create task %d: status %d", index, w.Code)
				return
			}
			results <- nil
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	assert.Empty(t, errors, "All concurrent operations should succeed")

	// Verify all tasks were created
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.GetTasks(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var response TaskListResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, numGoroutines, response.Count)
}

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	testHandler.HealthCheck(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var health map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &health)
	assert.Equal(t, "healthy", health["status"])
	assert.NotNil(t, health["database"])
}

// Helper functions
func createTestUserAndGetToken(t *testing.T, email string) string {
	userRepo := NewUserRepository(testDB.DB)
	jwtService := NewJWTService(testConfig.JWTSecret)

	user := &User{
		ID:           fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Email:        email,
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye", // bcrypt hash for "password123"
		FirstName:    "Test",
		LastName:     "User",
		Role:         "user",
		IsActive:     true,
	}

	err := userRepo.Create(context.Background(), user)
	require.NoError(t, err)

	token, err := jwtService.GenerateToken(user)
	require.NoError(t, err)

	return token
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}