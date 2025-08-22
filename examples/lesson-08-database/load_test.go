package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestConfig defines load testing parameters
type LoadTestConfig struct {
	NumUsers         int
	RequestsPerUser  int
	ConcurrentUsers  int
	TestDurationSecs int
}

var defaultLoadConfig = LoadTestConfig{
	NumUsers:         10,
	RequestsPerUser:  50,
	ConcurrentUsers:  5,
	TestDurationSecs: 30,
}

// LoadTestMetrics tracks performance metrics
type LoadTestMetrics struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedRequests   int64
	TotalDuration    time.Duration
	AvgResponseTime  time.Duration
	MinResponseTime  time.Duration
	MaxResponseTime  time.Duration
	RequestsPerSec   float64
	ConnectionErrors int64
	mu               sync.RWMutex
	responseTimes    []time.Duration
}

func (m *LoadTestMetrics) AddRequest(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.responseTimes = append(m.responseTimes, duration)

	if success {
		m.SuccessfulReqs++
	} else {
		m.FailedRequests++
	}

	if m.MinResponseTime == 0 || duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
}

func (m *LoadTestMetrics) AddConnectionError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionErrors++
}

func (m *LoadTestMetrics) Finalize() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.responseTimes) > 0 {
		var total time.Duration
		for _, rt := range m.responseTimes {
			total += rt
		}
		m.AvgResponseTime = total / time.Duration(len(m.responseTimes))
	}

	if m.TotalDuration > 0 {
		m.RequestsPerSec = float64(m.TotalRequests) / m.TotalDuration.Seconds()
	}
}

func (m *LoadTestMetrics) Report() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Total Requests: %d\n", m.TotalRequests)
	fmt.Printf("Successful Requests: %d (%.2f%%)\n", m.SuccessfulReqs, float64(m.SuccessfulReqs)/float64(m.TotalRequests)*100)
	fmt.Printf("Failed Requests: %d (%.2f%%)\n", m.FailedRequests, float64(m.FailedRequests)/float64(m.TotalRequests)*100)
	fmt.Printf("Connection Errors: %d\n", m.ConnectionErrors)
	fmt.Printf("Average Response Time: %v\n", m.AvgResponseTime)
	fmt.Printf("Min Response Time: %v\n", m.MinResponseTime)
	fmt.Printf("Max Response Time: %v\n", m.MaxResponseTime)
	fmt.Printf("Requests per Second: %.2f\n", m.RequestsPerSec)
	fmt.Printf("Test Duration: %v\n", m.TotalDuration)

	// Calculate percentiles
	if len(m.responseTimes) > 0 {
		// Sort response times for percentile calculation
		times := make([]time.Duration, len(m.responseTimes))
		copy(times, m.responseTimes)
		
		// Simple bubble sort (good enough for testing)
		for i := 0; i < len(times); i++ {
			for j := 0; j < len(times)-1-i; j++ {
				if times[j] > times[j+1] {
					times[j], times[j+1] = times[j+1], times[j]
				}
			}
		}

		p50 := times[len(times)*50/100]
		p90 := times[len(times)*90/100]
		p95 := times[len(times)*95/100]
		p99 := times[len(times)*99/100]

		fmt.Printf("50th percentile: %v\n", p50)
		fmt.Printf("90th percentile: %v\n", p90)
		fmt.Printf("95th percentile: %v\n", p95)
		fmt.Printf("99th percentile: %v\n", p99)
	}
}

func TestLoadUserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanupTestData()
	metrics := &LoadTestMetrics{}
	
	startTime := time.Now()
	var wg sync.WaitGroup

	// Simulate concurrent user registrations
	for i := 0; i < defaultLoadConfig.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()
			
			for j := 0; j < defaultLoadConfig.RequestsPerUser; j++ {
				reqStart := time.Now()
				
				regReq := RegisterRequest{
					Email:     fmt.Sprintf("loadtest%d_%d@example.com", userIndex, j),
					Password:  "password123",
					FirstName: fmt.Sprintf("User%d", userIndex),
					LastName:  fmt.Sprintf("Test%d", j),
				}

				body, _ := json.Marshal(regReq)
				req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				testHandler.Register(w, req)
				
				duration := time.Since(reqStart)
				success := w.Code == http.StatusCreated
				metrics.AddRequest(duration, success)

				if !success {
					t.Logf("Registration failed for user %d_%d: %d", userIndex, j, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	metrics.TotalDuration = time.Since(startTime)
	metrics.Finalize()
	metrics.Report()

	// Assertions
	assert.Greater(t, metrics.SuccessfulReqs, int64(0), "Should have successful registrations")
	assert.Less(t, float64(metrics.FailedRequests)/float64(metrics.TotalRequests), 0.05, "Failure rate should be less than 5%")
	assert.Less(t, metrics.AvgResponseTime, 500*time.Millisecond, "Average response time should be under 500ms")
}

func TestLoadTaskOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanupTestData()
	
	// Create test users and get tokens
	tokens := make([]string, defaultLoadConfig.ConcurrentUsers)
	for i := 0; i < defaultLoadConfig.ConcurrentUsers; i++ {
		email := fmt.Sprintf("taskload%d@example.com", i)
		tokens[i] = createTestUserAndGetToken(t, email)
	}

	metrics := &LoadTestMetrics{}
	startTime := time.Now()
	var wg sync.WaitGroup

	// Test concurrent task operations
	for i := 0; i < defaultLoadConfig.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()
			token := tokens[userIndex]
			
			for j := 0; j < defaultLoadConfig.RequestsPerUser; j++ {
				// Mix of operations: 60% create, 30% read, 10% update
				operation := j % 10
				
				switch {
				case operation < 6: // Create task
					performTaskCreate(t, token, userIndex, j, metrics)
				case operation < 9: // Read tasks
					performTaskRead(t, token, metrics)
				default: // Update task (if any exist)
					performTaskUpdate(t, token, metrics)
				}
			}
		}(i)
	}

	wg.Wait()
	metrics.TotalDuration = time.Since(startTime)
	metrics.Finalize()
	metrics.Report()

	// Performance assertions
	assert.Greater(t, metrics.SuccessfulReqs, int64(0), "Should have successful operations")
	assert.Less(t, float64(metrics.FailedRequests)/float64(metrics.TotalRequests), 0.10, "Failure rate should be less than 10%")
	assert.Greater(t, metrics.RequestsPerSec, 10.0, "Should handle at least 10 requests per second")
}

func TestDatabaseConnectionPoolUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Monitor database connection pool during load
	startStats := testDB.Stats()
	
	var wg sync.WaitGroup
	const numConcurrentConnections = 50

	for i := 0; i < numConcurrentConnections; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			// Each goroutine performs multiple database operations
			for j := 0; j < 10; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				
				// Simulate various database operations
				switch j % 3 {
				case 0:
					// Read operation
					var count int
					testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
				case 1:
					// Complex query
					rows, err := testDB.QueryContext(ctx, `
						SELECT u.id, COUNT(t.id) as task_count 
						FROM users u 
						LEFT JOIN tasks t ON u.id = t.user_id 
						GROUP BY u.id`)
					if err == nil {
						for rows.Next() {
							var id string
							var count int
							rows.Scan(&id, &count)
						}
						rows.Close()
					}
				case 2:
					// Health check
					testDB.PingContext(ctx)
				}
				
				cancel()
				time.Sleep(10 * time.Millisecond) // Small delay between operations
			}
		}(i)
	}

	wg.Wait()
	
	endStats := testDB.Stats()
	
	// Verify connection pool behaved correctly
	assert.LessOrEqual(t, endStats.OpenConnections, endStats.MaxOpenConnections, 
		"Should not exceed max connections")
	assert.Greater(t, endStats.InUse, 0, "Should have used connections")
	
	fmt.Printf("\n=== Connection Pool Stats ===\n")
	fmt.Printf("Max Open Connections: %d\n", endStats.MaxOpenConnections)
	fmt.Printf("Open Connections: %d\n", endStats.OpenConnections)
	fmt.Printf("Connections In Use: %d\n", endStats.InUse)
	fmt.Printf("Idle Connections: %d\n", endStats.Idle)
	fmt.Printf("Total Opened: %d\n", endStats.MaxLifetimeClosed)
}

func TestLongRunningTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanupTestData()
	token := createTestUserAndGetToken(t, "longtx@example.com")

	var wg sync.WaitGroup
	metrics := &LoadTestMetrics{}
	
	// Test multiple long-running transactions
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			start := time.Now()
			
			// Create task with multiple categories (complex transaction)
			createReq := CreateTaskRequest{
				Title:         fmt.Sprintf("Long Transaction Task %d", index),
				Description:   "Testing long transaction handling",
				Priority:      "medium",
				CategoryNames: []string{
					fmt.Sprintf("Category-%d-1", index),
					fmt.Sprintf("Category-%d-2", index),
					fmt.Sprintf("Category-%d-3", index),
				},
			}

			body, _ := json.Marshal(createReq)
			req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			testHandler.CreateTask(w, req)
			
			duration := time.Since(start)
			success := w.Code == http.StatusCreated
			metrics.AddRequest(duration, success)
			
			if !success {
				t.Logf("Long transaction failed for index %d: %d", index, w.Code)
			}
		}(i)
	}

	wg.Wait()
	metrics.Finalize()
	
	// All transactions should complete successfully
	assert.Equal(t, int64(5), metrics.SuccessfulReqs, "All long transactions should succeed")
	assert.Less(t, metrics.AvgResponseTime, 2*time.Second, "Long transactions should complete within 2 seconds")
}

// BenchmarkTaskCreation benchmarks task creation performance
func BenchmarkTaskCreation(b *testing.B) {
	cleanupTestData()
	token := createTestUserAndGetToken(&testing.T{}, "bench@example.com")

	createReq := CreateTaskRequest{
		Title:       "Benchmark Task",
		Description: "Performance testing",
		Priority:    "medium",
	}

	body, _ := json.Marshal(createReq)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		testHandler.CreateTask(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected 201, got %d", w.Code)
		}
	}
}

// BenchmarkTaskRetrieval benchmarks task retrieval performance
func BenchmarkTaskRetrieval(b *testing.B) {
	cleanupTestData()
	token := createTestUserAndGetToken(&testing.T{}, "benchget@example.com")

	// Create some test data
	for i := 0; i < 100; i++ {
		createReq := CreateTaskRequest{
			Title:       fmt.Sprintf("Task %d", i),
			Description: "Benchmark data",
			Priority:    "medium",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		testHandler.CreateTask(w, req)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		testHandler.GetTasks(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// Helper functions for load testing
func performTaskCreate(t *testing.T, token string, userIndex, taskIndex int, metrics *LoadTestMetrics) {
	start := time.Now()
	
	createReq := CreateTaskRequest{
		Title:       fmt.Sprintf("Load Test Task %d-%d", userIndex, taskIndex),
		Description: "Created during load testing",
		Priority:    "medium",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.CreateTask(w, req)
	
	duration := time.Since(start)
	success := w.Code == http.StatusCreated
	metrics.AddRequest(duration, success)
}

func performTaskRead(t *testing.T, token string, metrics *LoadTestMetrics) {
	start := time.Now()
	
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.GetTasks(w, req)
	
	duration := time.Since(start)
	success := w.Code == http.StatusOK
	metrics.AddRequest(duration, success)
}

func performTaskUpdate(t *testing.T, token string, metrics *LoadTestMetrics) {
	start := time.Now()
	
	// First, try to get a task to update
	req := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	testHandler.GetTasks(w, req)
	
	if w.Code == http.StatusOK {
		var response TaskListResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if len(response.Tasks) > 0 {
			taskID := response.Tasks[0].ID
			
			updateReq := UpdateTaskRequest{
				Description: stringPtr("Updated during load test"),
			}

			body, _ := json.Marshal(updateReq)
			req2 := httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, bytes.NewReader(body))
			req2.Header.Set("Content-Type", "application/json")
			req2.Header.Set("Authorization", "Bearer "+token)
			w2 := httptest.NewRecorder()

			testHandler.UpdateTask(w2, req2)
			
			duration := time.Since(start)
			success := w2.Code == http.StatusOK
			metrics.AddRequest(duration, success)
			return
		}
	}
	
	// If no task to update, record as failed
	duration := time.Since(start)
	metrics.AddRequest(duration, false)
}

// TestMain for load tests - you can run with: go test -tags=loadtest
func init() {
	// Check if running in load test mode
	if os.Getenv("LOAD_TEST") == "true" {
		log.Println("Running in load test mode")
		
		// Override default config from environment
		if users := os.Getenv("LOAD_USERS"); users != "" {
			fmt.Sscanf(users, "%d", &defaultLoadConfig.NumUsers)
		}
		if requests := os.Getenv("LOAD_REQUESTS_PER_USER"); requests != "" {
			fmt.Sscanf(requests, "%d", &defaultLoadConfig.RequestsPerUser)
		}
		if concurrent := os.Getenv("LOAD_CONCURRENT"); concurrent != "" {
			fmt.Sscanf(concurrent, "%d", &defaultLoadConfig.ConcurrentUsers)
		}
	}
}