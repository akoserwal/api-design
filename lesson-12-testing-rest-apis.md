# Lesson 12: Testing REST APIs

## Learning Objectives
By the end of this lesson, you will be able to:
- Write comprehensive unit tests for HTTP handlers
- Implement integration tests for API endpoints
- Use test doubles and mocking effectively
- Set up test databases and clean test environments
- Write table-driven tests for various scenarios
- Implement performance and load testing
- Use testing tools and frameworks effectively

## Testing Pyramid for APIs

### 1. Unit Tests (70%)
- Test individual functions and methods
- Fast execution
- Mock external dependencies
- Test business logic in isolation

### 2. Integration Tests (20%)
- Test API endpoints end-to-end
- Test database interactions
- Test middleware functionality
- Test authentication flows

### 3. End-to-End Tests (10%)
- Test complete user workflows
- Test real deployment scenarios
- Slower but comprehensive

## Setting Up Test Environment

### Test Dependencies
```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go get github.com/stretchr/testify/suite
go get github.com/DATA-DOG/go-sqlmock
```

### Test Structure
```
project/
├── internal/
│   ├── handlers/
│   │   ├── tasks.go
│   │   └── tasks_test.go
│   ├── models/
│   │   ├── task.go
│   │   └── task_test.go
│   └── storage/
│       ├── memory.go
│       └── memory_test.go
├── tests/
│   ├── integration/
│   │   └── api_test.go
│   └── fixtures/
│       └── testdata.json
└── testutil/
    ├── helpers.go
    └── mocks.go
```

## Unit Testing HTTP Handlers

### Basic Handler Test
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
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

// Mock storage implementation
type MockTaskStorage struct {
    mock.Mock
}

func (m *MockTaskStorage) GetAll() []*models.Task {
    args := m.Called()
    return args.Get(0).([]*models.Task)
}

func (m *MockTaskStorage) GetByID(id string) (*models.Task, error) {
    args := m.Called(id)
    return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskStorage) Create(task *models.Task) error {
    args := m.Called(task)
    return args.Error(0)
}

func (m *MockTaskStorage) Update(id string, req models.UpdateTaskRequest) (*models.Task, error) {
    args := m.Called(id, req)
    return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskStorage) Delete(id string) error {
    args := m.Called(id)
    return args.Error(0)
}

func (m *MockTaskStorage) GetByStatus(completed bool) []*models.Task {
    args := m.Called(completed)
    return args.Get(0).([]*models.Task)
}

func TestTaskHandler_GetTasks(t *testing.T) {
    // Setup
    mockStorage := new(MockTaskStorage)
    handler := NewTaskHandler(mockStorage)
    
    tasks := []*models.Task{
        {
            ID:          "1",
            Title:       "Test Task 1",
            Description: "Description 1",
            Completed:   false,
        },
        {
            ID:          "2",
            Title:       "Test Task 2",
            Description: "Description 2",
            Completed:   true,
        },
    }
    
    mockStorage.On("GetAll").Return(tasks)
    
    // Create request
    req, err := http.NewRequest("GET", "/api/tasks", nil)
    assert.NoError(t, err)
    
    // Create response recorder
    rr := httptest.NewRecorder()
    
    // Execute
    handler.GetTasks(rr, req)
    
    // Assert
    assert.Equal(t, http.StatusOK, rr.Code)
    assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
    
    var response TaskListResponse
    err = json.Unmarshal(rr.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, 2, response.Count)
    assert.Len(t, response.Tasks, 2)
    
    mockStorage.AssertExpectations(t)
}

func TestTaskHandler_CreateTask(t *testing.T) {
    mockStorage := new(MockTaskStorage)
    handler := NewTaskHandler(mockStorage)
    
    req := models.CreateTaskRequest{
        Title:       "New Task",
        Description: "New Description",
    }
    
    mockStorage.On("Create", mock.AnythingOfType("*models.Task")).Return(nil)
    
    jsonData, _ := json.Marshal(req)
    httpReq, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    httpReq.Header.Set("Content-Type", "application/json")
    
    rr := httptest.NewRecorder()
    
    handler.CreateTask(rr, httpReq)
    
    assert.Equal(t, http.StatusCreated, rr.Code)
    
    var task models.Task
    err = json.Unmarshal(rr.Body.Bytes(), &task)
    assert.NoError(t, err)
    assert.Equal(t, req.Title, task.Title)
    assert.Equal(t, req.Description, task.Description)
    assert.NotEmpty(t, task.ID)
    assert.False(t, task.Completed)
    
    mockStorage.AssertExpectations(t)
}

func TestTaskHandler_CreateTask_ValidationError(t *testing.T) {
    mockStorage := new(MockTaskStorage)
    handler := NewTaskHandler(mockStorage)
    
    req := models.CreateTaskRequest{
        Title:       "", // Empty title should cause validation error
        Description: "Description",
    }
    
    jsonData, _ := json.Marshal(req)
    httpReq, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
    assert.NoError(t, err)
    httpReq.Header.Set("Content-Type", "application/json")
    
    rr := httptest.NewRecorder()
    
    handler.CreateTask(rr, httpReq)
    
    assert.Equal(t, http.StatusBadRequest, rr.Code)
    
    var response ErrorResponse
    err = json.Unmarshal(rr.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Contains(t, response.Message, "required")
    
    // Storage should not be called
    mockStorage.AssertNotCalled(t, "Create")
}
```

### Table-Driven Tests
```go
func TestTaskHandler_GetTask(t *testing.T) {
    tests := []struct {
        name           string
        taskID         string
        setupMock      func(*MockTaskStorage)
        expectedStatus int
        expectedError  string
    }{
        {
            name:   "successful retrieval",
            taskID: "valid-id",
            setupMock: func(m *MockTaskStorage) {
                task := &models.Task{
                    ID:    "valid-id",
                    Title: "Test Task",
                }
                m.On("GetByID", "valid-id").Return(task, nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:   "task not found",
            taskID: "invalid-id",
            setupMock: func(m *MockTaskStorage) {
                m.On("GetByID", "invalid-id").Return((*models.Task)(nil), storage.ErrTaskNotFound)
            },
            expectedStatus: http.StatusNotFound,
            expectedError:  "Task not found",
        },
        {
            name:   "storage error",
            taskID: "error-id",
            setupMock: func(m *MockTaskStorage) {
                m.On("GetByID", "error-id").Return((*models.Task)(nil), errors.New("storage error"))
            },
            expectedStatus: http.StatusInternalServerError,
            expectedError:  "Failed to retrieve task",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockStorage := new(MockTaskStorage)
            handler := NewTaskHandler(mockStorage)
            
            tt.setupMock(mockStorage)
            
            req, _ := http.NewRequest("GET", "/api/tasks/"+tt.taskID, nil)
            req = mux.SetURLVars(req, map[string]string{"id": tt.taskID})
            
            rr := httptest.NewRecorder()
            
            handler.GetTask(rr, req)
            
            assert.Equal(t, tt.expectedStatus, rr.Code)
            
            if tt.expectedError != "" {
                var response ErrorResponse
                json.Unmarshal(rr.Body.Bytes(), &response)
                assert.Contains(t, response.Message, tt.expectedError)
            }
            
            mockStorage.AssertExpectations(t)
        })
    }
}
```

## Testing with Test Suites

### Test Suite Structure
```go
// internal/handlers/suite_test.go
package handlers

import (
    "testing"
    
    "github.com/stretchr/testify/suite"
    "github.com/yourusername/task-api/internal/storage"
)

type TaskHandlerTestSuite struct {
    suite.Suite
    handler *TaskHandler
    storage *storage.MemoryStorage
}

func (suite *TaskHandlerTestSuite) SetupTest() {
    // Create fresh storage for each test
    suite.storage = storage.NewMemoryStorage()
    suite.handler = NewTaskHandler(suite.storage)
    
    // Add some test data
    suite.setupTestData()
}

func (suite *TaskHandlerTestSuite) setupTestData() {
    tasks := []*models.Task{
        {
            ID:          "task-1",
            Title:       "Test Task 1",
            Description: "Description 1",
            Completed:   false,
        },
        {
            ID:          "task-2",
            Title:       "Test Task 2",
            Description: "Description 2",
            Completed:   true,
        },
    }
    
    for _, task := range tasks {
        suite.storage.Create(task)
    }
}

func (suite *TaskHandlerTestSuite) TestGetAllTasks() {
    req, _ := http.NewRequest("GET", "/api/tasks", nil)
    rr := httptest.NewRecorder()
    
    suite.handler.GetTasks(rr, req)
    
    suite.Equal(http.StatusOK, rr.Code)
    
    var response TaskListResponse
    json.Unmarshal(rr.Body.Bytes(), &response)
    suite.Equal(2, response.Count)
}

func (suite *TaskHandlerTestSuite) TestGetTasksByStatus() {
    req, _ := http.NewRequest("GET", "/api/tasks?completed=true", nil)
    rr := httptest.NewRecorder()
    
    suite.handler.GetTasks(rr, req)
    
    suite.Equal(http.StatusOK, rr.Code)
    
    var response TaskListResponse
    json.Unmarshal(rr.Body.Bytes(), &response)
    suite.Equal(1, response.Count)
    suite.True(response.Tasks[0].Completed)
}

func TestTaskHandlerTestSuite(t *testing.T) {
    suite.Run(t, new(TaskHandlerTestSuite))
}
```

## Integration Testing

### Test Server Setup
```go
// tests/integration/api_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gorilla/mux"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    
    "github.com/yourusername/task-api/internal/auth"
    "github.com/yourusername/task-api/internal/handlers"
    "github.com/yourusername/task-api/internal/middleware"
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

type APITestSuite struct {
    suite.Suite
    server      *httptest.Server
    client      *http.Client
    userStorage *storage.MemoryUserStorage
    taskStorage *storage.MemoryTaskStorage
    jwtService  *auth.JWTService
    authToken   string
}

func (suite *APITestSuite) SetupSuite() {
    // Initialize services
    suite.jwtService = auth.NewJWTService("test-secret", "test-issuer")
    suite.userStorage = storage.NewMemoryUserStorage()
    suite.taskStorage = storage.NewMemoryTaskStorage()
    
    // Initialize handlers
    authHandler := handlers.NewAuthHandler(suite.userStorage, suite.jwtService)
    taskHandler := handlers.NewTaskHandler(suite.taskStorage)
    
    // Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(suite.jwtService)
    
    // Setup router
    router := mux.NewRouter()
    api := router.PathPrefix("/api").Subrouter()
    
    // Auth routes
    api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
    api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
    
    // Protected routes
    protected := api.PathPrefix("").Subrouter()
    protected.Use(authMiddleware.RequireAuth)
    protected.HandleFunc("/tasks", taskHandler.GetTasks).Methods("GET")
    protected.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
    protected.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
    protected.HandleFunc("/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
    protected.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
    
    // Create test server
    suite.server = httptest.NewServer(router)
    suite.client = suite.server.Client()
    
    // Create test user and get auth token
    suite.createTestUser()
}

func (suite *APITestSuite) TearDownSuite() {
    suite.server.Close()
}

func (suite *APITestSuite) SetupTest() {
    // Clear task storage before each test
    suite.taskStorage = storage.NewMemoryTaskStorage()
}

func (suite *APITestSuite) createTestUser() {
    registerReq := models.RegisterRequest{
        Email:     "test@example.com",
        Password:  "password123",
        FirstName: "Test",
        LastName:  "User",
    }
    
    jsonData, _ := json.Marshal(registerReq)
    resp, err := suite.client.Post(
        suite.server.URL+"/api/auth/register",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    suite.NoError(err)
    defer resp.Body.Close()
    
    var loginResp models.LoginResponse
    json.NewDecoder(resp.Body).Decode(&loginResp)
    suite.authToken = loginResp.Token
}

func (suite *APITestSuite) makeAuthenticatedRequest(method, url string, body []byte) (*http.Response, error) {
    req, err := http.NewRequest(method, suite.server.URL+url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+suite.authToken)
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    
    return suite.client.Do(req)
}

func (suite *APITestSuite) TestTaskCRUDWorkflow() {
    // Create task
    createReq := models.CreateTaskRequest{
        Title:       "Integration Test Task",
        Description: "Testing CRUD operations",
    }
    
    jsonData, _ := json.Marshal(createReq)
    resp, err := suite.makeAuthenticatedRequest("POST", "/api/tasks", jsonData)
    suite.NoError(err)
    suite.Equal(http.StatusCreated, resp.StatusCode)
    
    var createdTask models.Task
    json.NewDecoder(resp.Body).Decode(&createdTask)
    resp.Body.Close()
    
    suite.Equal(createReq.Title, createdTask.Title)
    suite.NotEmpty(createdTask.ID)
    
    // Get all tasks
    resp, err = suite.makeAuthenticatedRequest("GET", "/api/tasks", nil)
    suite.NoError(err)
    suite.Equal(http.StatusOK, resp.StatusCode)
    
    var taskList handlers.TaskListResponse
    json.NewDecoder(resp.Body).Decode(&taskList)
    resp.Body.Close()
    
    suite.Equal(1, taskList.Count)
    
    // Get specific task
    resp, err = suite.makeAuthenticatedRequest("GET", "/api/tasks/"+createdTask.ID, nil)
    suite.NoError(err)
    suite.Equal(http.StatusOK, resp.StatusCode)
    resp.Body.Close()
    
    // Update task
    updateReq := models.UpdateTaskRequest{
        Title:     &[]string{"Updated Title"}[0],
        Completed: &[]bool{true}[0],
    }
    
    jsonData, _ = json.Marshal(updateReq)
    resp, err = suite.makeAuthenticatedRequest("PUT", "/api/tasks/"+createdTask.ID, jsonData)
    suite.NoError(err)
    suite.Equal(http.StatusOK, resp.StatusCode)
    
    var updatedTask models.Task
    json.NewDecoder(resp.Body).Decode(&updatedTask)
    resp.Body.Close()
    
    suite.Equal("Updated Title", updatedTask.Title)
    suite.True(updatedTask.Completed)
    
    // Delete task
    resp, err = suite.makeAuthenticatedRequest("DELETE", "/api/tasks/"+createdTask.ID, nil)
    suite.NoError(err)
    suite.Equal(http.StatusNoContent, resp.StatusCode)
    resp.Body.Close()
    
    // Verify deletion
    resp, err = suite.makeAuthenticatedRequest("GET", "/api/tasks/"+createdTask.ID, nil)
    suite.NoError(err)
    suite.Equal(http.StatusNotFound, resp.StatusCode)
    resp.Body.Close()
}

func (suite *APITestSuite) TestAuthenticationFlow() {
    // Test registration
    registerReq := models.RegisterRequest{
        Email:     "newuser@example.com",
        Password:  "password123",
        FirstName: "New",
        LastName:  "User",
    }
    
    jsonData, _ := json.Marshal(registerReq)
    resp, err := suite.client.Post(
        suite.server.URL+"/api/auth/register",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    suite.NoError(err)
    defer resp.Body.Close()
    suite.Equal(http.StatusCreated, resp.StatusCode)
    
    // Test login
    loginReq := models.LoginRequest{
        Email:    registerReq.Email,
        Password: registerReq.Password,
    }
    
    jsonData, _ = json.Marshal(loginReq)
    resp, err = suite.client.Post(
        suite.server.URL+"/api/auth/login",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    suite.NoError(err)
    defer resp.Body.Close()
    suite.Equal(http.StatusOK, resp.StatusCode)
    
    var loginResp models.LoginResponse
    json.NewDecoder(resp.Body).Decode(&loginResp)
    suite.NotEmpty(loginResp.Token)
    suite.Equal(registerReq.Email, loginResp.User.Email)
    
    // Test accessing protected endpoint with token
    req, _ := http.NewRequest("GET", suite.server.URL+"/api/tasks", nil)
    req.Header.Set("Authorization", "Bearer "+loginResp.Token)
    
    resp, err = suite.client.Do(req)
    suite.NoError(err)
    defer resp.Body.Close()
    suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *APITestSuite) TestUnauthorizedAccess() {
    // Test accessing protected endpoint without token
    resp, err := suite.client.Get(suite.server.URL + "/api/tasks")
    suite.NoError(err)
    defer resp.Body.Close()
    suite.Equal(http.StatusUnauthorized, resp.StatusCode)
    
    // Test accessing protected endpoint with invalid token
    req, _ := http.NewRequest("GET", suite.server.URL+"/api/tasks", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")
    
    resp, err = suite.client.Do(req)
    suite.NoError(err)
    defer resp.Body.Close()
    suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func TestAPITestSuite(t *testing.T) {
    suite.Run(t, new(APITestSuite))
}
```

## Testing with Database

### Test Database Setup
```go
// testutil/database.go
package testutil

import (
    "database/sql"
    "testing"
    
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/assert"
)

func SetupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
    db, mock, err := sqlmock.New()
    assert.NoError(t, err)
    return db, mock
}

func TestWithMockDB(t *testing.T) {
    db, mock := SetupMockDB(t)
    defer db.Close()
    
    // Setup expected queries
    mock.ExpectQuery("SELECT id, title, description FROM tasks WHERE id = ?").
        WithArgs("task-1").
        WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description"}).
            AddRow("task-1", "Test Task", "Test Description"))
    
    // Run test
    storage := storage.NewPostgresStorage(db)
    task, err := storage.GetByID("task-1")
    
    assert.NoError(t, err)
    assert.Equal(t, "task-1", task.ID)
    assert.Equal(t, "Test Task", task.Title)
    
    // Verify expectations
    assert.NoError(t, mock.ExpectationsWereMet())
}
```

### Test Container Database
```go
// testutil/testcontainer.go
package testutil

import (
    "context"
    "database/sql"
    "testing"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    _ "github.com/lib/pq"
)

func SetupTestDB(t *testing.T) *sql.DB {
    ctx := context.Background()
    
    req := testcontainers.ContainerRequest{
        Image:        "postgres:13",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "testdb",
            "POSTGRES_USER":     "testuser",
            "POSTGRES_PASSWORD": "testpass",
        },
        WaitingFor: wait.ForListeningPort("5432/tcp"),
    }
    
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatal(err)
    }
    
    t.Cleanup(func() {
        postgres.Terminate(ctx)
    })
    
    host, _ := postgres.Host(ctx)
    port, _ := postgres.MappedPort(ctx, "5432")
    
    dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
        host, port.Port())
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        t.Fatal(err)
    }
    
    // Run migrations
    runMigrations(db)
    
    return db
}
```

## Benchmarking and Performance Testing

### Benchmark Tests
```go
// internal/handlers/bench_test.go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"
    
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/storage"
)

func BenchmarkTaskHandler_CreateTask(b *testing.B) {
    storage := storage.NewMemoryStorage()
    handler := NewTaskHandler(storage)
    
    req := models.CreateTaskRequest{
        Title:       "Benchmark Task",
        Description: "Performance testing",
    }
    
    jsonData, _ := json.Marshal(req)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        httpReq := httptest.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
        httpReq.Header.Set("Content-Type", "application/json")
        
        rr := httptest.NewRecorder()
        handler.CreateTask(rr, httpReq)
    }
}

func BenchmarkTaskHandler_GetTasks(b *testing.B) {
    storage := storage.NewMemoryStorage()
    handler := NewTaskHandler(storage)
    
    // Setup test data
    for i := 0; i < 1000; i++ {
        task := &models.Task{
            ID:          fmt.Sprintf("task-%d", i),
            Title:       fmt.Sprintf("Task %d", i),
            Description: fmt.Sprintf("Description %d", i),
        }
        storage.Create(task)
    }
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        httpReq := httptest.NewRequest("GET", "/api/tasks", nil)
        rr := httptest.NewRecorder()
        handler.GetTasks(rr, httpReq)
    }
}
```

### Load Testing with Go
```go
// tests/load/load_test.go
package load

import (
    "bytes"
    "encoding/json"
    "net/http"
    "sync"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
)

func TestLoadCreateTasks(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    const (
        concurrency = 50
        requests    = 1000
        baseURL     = "http://localhost:8080"
    )
    
    var wg sync.WaitGroup
    results := make(chan result, requests)
    
    type result struct {
        statusCode int
        duration   time.Duration
        err        error
    }
    
    // Create worker function
    worker := func(taskChan <-chan int) {
        defer wg.Done()
        
        client := &http.Client{Timeout: 10 * time.Second}
        
        for taskNum := range taskChan {
            start := time.Now()
            
            reqBody := map[string]string{
                "title":       fmt.Sprintf("Load Test Task %d", taskNum),
                "description": "Generated by load test",
            }
            
            jsonData, _ := json.Marshal(reqBody)
            
            req, _ := http.NewRequest("POST", baseURL+"/api/tasks", bytes.NewBuffer(jsonData))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", "Bearer "+getTestToken())
            
            resp, err := client.Do(req)
            duration := time.Since(start)
            
            res := result{
                duration: duration,
                err:      err,
            }
            
            if resp != nil {
                res.statusCode = resp.StatusCode
                resp.Body.Close()
            }
            
            results <- res
        }
    }
    
    // Create task channel
    taskChan := make(chan int, requests)
    
    // Start workers
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go worker(taskChan)
    }
    
    // Send tasks
    start := time.Now()
    for i := 0; i < requests; i++ {
        taskChan <- i
    }
    close(taskChan)
    
    // Wait for completion
    wg.Wait()
    close(results)
    totalDuration := time.Since(start)
    
    // Analyze results
    var (
        successCount int
        totalLatency time.Duration
        maxLatency   time.Duration
        minLatency   = time.Hour
    )
    
    for res := range results {
        if res.err == nil && res.statusCode == http.StatusCreated {
            successCount++
        }
        
        totalLatency += res.duration
        if res.duration > maxLatency {
            maxLatency = res.duration
        }
        if res.duration < minLatency {
            minLatency = res.duration
        }
    }
    
    avgLatency := totalLatency / time.Duration(requests)
    throughput := float64(successCount) / totalDuration.Seconds()
    
    t.Logf("Load test results:")
    t.Logf("Total requests: %d", requests)
    t.Logf("Successful requests: %d", successCount)
    t.Logf("Success rate: %.2f%%", float64(successCount)/float64(requests)*100)
    t.Logf("Total duration: %v", totalDuration)
    t.Logf("Throughput: %.2f requests/second", throughput)
    t.Logf("Average latency: %v", avgLatency)
    t.Logf("Min latency: %v", minLatency)
    t.Logf("Max latency: %v", maxLatency)
    
    // Assertions
    assert.True(t, float64(successCount)/float64(requests) > 0.95, "Success rate should be > 95%")
    assert.True(t, avgLatency < 100*time.Millisecond, "Average latency should be < 100ms")
}
```

## Test Utilities and Helpers

### Test Helpers
```go
// testutil/helpers.go
package testutil

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func AssertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
    assert.Equal(t, expectedStatus, rr.Code)
    assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
    
    if expectedBody != nil {
        expectedJSON, _ := json.Marshal(expectedBody)
        assert.JSONEq(t, string(expectedJSON), rr.Body.String())
    }
}

func CreateTestRequest(t *testing.T, method, url string, body interface{}) *http.Request {
    var reqBody []byte
    if body != nil {
        var err error
        reqBody, err = json.Marshal(body)
        assert.NoError(t, err)
    }
    
    req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
    assert.NoError(t, err)
    
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    
    return req
}

func MustMarshalJSON(t *testing.T, v interface{}) []byte {
    data, err := json.Marshal(v)
    assert.NoError(t, err)
    return data
}

func MustUnmarshalJSON(t *testing.T, data []byte, v interface{}) {
    err := json.Unmarshal(data, v)
    assert.NoError(t, err)
}
```

## Test Configuration

### Test Configuration File
```yaml
# config/test.yaml
database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  name: "testdb"
  user: "testuser"
  password: "testpass"
  ssl_mode: "disable"

jwt:
  secret: "test-secret-key"
  issuer: "test-api"
  expiration: "1h"

server:
  port: 8080
  timeout: "30s"
```

### Test Environment Setup
```go
// testutil/env.go
package testutil

import (
    "os"
    "testing"
)

func SetupTestEnv(t *testing.T) {
    os.Setenv("APP_ENV", "test")
    os.Setenv("JWT_SECRET", "test-secret-key")
    os.Setenv("DB_HOST", "localhost")
    os.Setenv("DB_NAME", "testdb")
    
    t.Cleanup(func() {
        os.Unsetenv("APP_ENV")
        os.Unsetenv("JWT_SECRET")
        os.Unsetenv("DB_HOST")
        os.Unsetenv("DB_NAME")
    })
}
```

## Running Tests

### Test Commands
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run only unit tests
go test -short ./...

# Run specific test
go test -run TestTaskHandler_CreateTask ./internal/handlers

# Run benchmarks
go test -bench=. ./...

# Run tests with race detection
go test -race ./...

# Verbose output
go test -v ./...
```

### Makefile for Testing
```makefile
# Makefile
.PHONY: test test-unit test-integration test-load test-coverage

test:
	go test ./...

test-unit:
	go test -short ./...

test-integration:
	go test -tags=integration ./tests/integration/...

test-load:
	go test -tags=load ./tests/load/...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-race:
	go test -race ./...

benchmark:
	go test -bench=. -benchmem ./...

test-all: test-unit test-integration test-coverage
```

## Best Practices

### 1. Test Organization
- Separate unit, integration, and load tests
- Use build tags for different test types
- Keep tests close to the code they test

### 2. Test Data Management
- Use fixtures for consistent test data
- Clean up after each test
- Use factories for complex object creation

### 3. Mocking Strategy
- Mock external dependencies
- Use interfaces for mockable components
- Don't mock what you don't own

### 4. Assertion Strategy
- Use descriptive test names
- Test one thing per test
- Use table-driven tests for multiple scenarios

### 5. Performance Considerations
- Run benchmarks regularly
- Set performance budgets
- Test under realistic conditions

## Common Testing Pitfalls

1. **Testing implementation details**: Test behavior, not implementation
2. **Brittle tests**: Tests that break with minor changes
3. **Slow tests**: Integration tests that take too long
4. **Flaky tests**: Tests that pass/fail randomly
5. **Poor test coverage**: Missing edge cases and error paths

## Next Steps

In the next lesson, we'll explore performance optimization and caching strategies to make your API faster and more scalable.

## Key Takeaways

- Testing is crucial for API reliability
- Use different testing strategies for different purposes
- Mock external dependencies in unit tests
- Integration tests validate end-to-end functionality
- Performance testing ensures scalability
- Good test coverage provides confidence in changes

## Practice Exercises

1. Write comprehensive tests for authentication handlers
2. Create integration tests for a complete user workflow
3. Implement benchmark tests for critical endpoints
4. Set up automated testing in CI/CD pipeline
5. Add load testing for your API endpoints