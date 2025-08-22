package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestHandler creates a fresh handler with empty storage for testing
func setupTestHandler() *TaskHandler {
	storage := &MemoryStorage{}
	return NewTaskHandler(storage)
}

// setupTestTask creates a test task
func setupTestTask() Task {
	return Task{
		ID:          "test-id-123",
		Title:       "Test Task",
		Description: "Test Description",
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestTaskHandler_GetTasks_Empty(t *testing.T) {
	handler := setupTestHandler()
	
	req, err := http.NewRequest("GET", "/api/tasks", nil)
	require.NoError(t, err)
	
	rr := httptest.NewRecorder()
	handler.GetTasks(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	
	var response TaskListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 0, response.Count)
	assert.Len(t, response.Tasks, 0)
	assert.NotEmpty(t, response.Meta.RequestID)
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	handler := setupTestHandler()
	
	createReq := CreateTaskRequest{
		Title:       "New Task",
		Description: "New Description",
	}
	
	jsonData, _ := json.Marshal(createReq)
	req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler.CreateTask(rr, req)
	
	assert.Equal(t, http.StatusCreated, rr.Code)
	
	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	
	assert.Equal(t, createReq.Title, task.Title)
	assert.Equal(t, createReq.Description, task.Description)
	assert.False(t, task.Completed)
	assert.NotEmpty(t, task.ID)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskHandler_CreateTask_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		request     CreateTaskRequest
		expectedMsg string
	}{
		{
			name:        "missing title",
			request:     CreateTaskRequest{Description: "Description"},
			expectedMsg: "Title is required",
		},
		{
			name:        "empty title",
			request:     CreateTaskRequest{Title: "   ", Description: "Description"},
			expectedMsg: "Title is required",
		},
		{
			name:        "title too long",
			request:     CreateTaskRequest{Title: string(make([]byte, 101)), Description: "Description"},
			expectedMsg: "Title cannot exceed 100 characters",
		},
		{
			name:        "description too long",
			request:     CreateTaskRequest{Title: "Title", Description: string(make([]byte, 501))},
			expectedMsg: "Description cannot exceed 500 characters",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler()
			
			jsonData, _ := json.Marshal(tt.request)
			req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler.CreateTask(rr, req)
			
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			
			var errorResp ErrorResponse
			err = json.Unmarshal(rr.Body.Bytes(), &errorResp)
			require.NoError(t, err)
			
			assert.Contains(t, errorResp.Message, tt.expectedMsg)
			assert.NotEmpty(t, errorResp.RequestID)
		})
	}
}

func TestTaskHandler_CreateTask_InvalidJSON(t *testing.T) {
	handler := setupTestHandler()
	
	req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler.CreateTask(rr, req)
	
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	
	var errorResp ErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	assert.Contains(t, errorResp.Message, "Invalid JSON")
}

func TestTaskHandler_GetTask_Success(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	req, err := http.NewRequest("GET", "/api/tasks/"+testTask.ID, nil)
	require.NoError(t, err)
	
	// Setup mux vars
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.GetTask(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	
	assert.Equal(t, testTask.ID, task.ID)
	assert.Equal(t, testTask.Title, task.Title)
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	handler := setupTestHandler()
	
	req, err := http.NewRequest("GET", "/api/tasks/nonexistent", nil)
	require.NoError(t, err)
	
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	
	rr := httptest.NewRecorder()
	handler.GetTask(rr, req)
	
	assert.Equal(t, http.StatusNotFound, rr.Code)
	
	var errorResp ErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	assert.Contains(t, errorResp.Message, "Task not found")
}

func TestTaskHandler_UpdateTask_Success(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	updateReq := UpdateTaskRequest{
		Title:       stringPtr("Updated Title"),
		Description: stringPtr("Updated Description"),
		Completed:   boolPtr(true),
	}
	
	jsonData, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PUT", "/api/tasks/"+testTask.ID, bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.UpdateTask(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	
	assert.Equal(t, "Updated Title", task.Title)
	assert.Equal(t, "Updated Description", task.Description)
	assert.True(t, task.Completed)
	assert.True(t, task.UpdatedAt.After(testTask.UpdatedAt))
}

func TestTaskHandler_PatchTask_Success(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	// Test partial update - only title
	updateReq := UpdateTaskRequest{
		Title: stringPtr("Patched Title"),
	}
	
	jsonData, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PATCH", "/api/tasks/"+testTask.ID, bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.PatchTask(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	
	assert.Equal(t, "Patched Title", task.Title)
	assert.Equal(t, testTask.Description, task.Description) // Should remain unchanged
	assert.Equal(t, testTask.Completed, task.Completed)     // Should remain unchanged
}

func TestTaskHandler_PatchTask_NoFields(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	// Empty update request
	updateReq := UpdateTaskRequest{}
	
	jsonData, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PATCH", "/api/tasks/"+testTask.ID, bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.PatchTask(rr, req)
	
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	
	var errorResp ErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	assert.Contains(t, errorResp.Message, "At least one field")
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	req, err := http.NewRequest("DELETE", "/api/tasks/"+testTask.ID, nil)
	require.NoError(t, err)
	
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.DeleteTask(rr, req)
	
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Body.String())
	
	// Verify task is deleted
	_, err = handler.storage.GetByID(testTask.ID)
	assert.Error(t, err)
}

func TestTaskHandler_CompleteTask_Success(t *testing.T) {
	handler := setupTestHandler()
	testTask := setupTestTask()
	
	// Add task to storage
	handler.storage.Create(&testTask)
	
	req, err := http.NewRequest("PATCH", "/api/tasks/"+testTask.ID+"/complete", nil)
	require.NoError(t, err)
	
	req = mux.SetURLVars(req, map[string]string{"id": testTask.ID})
	
	rr := httptest.NewRecorder()
	handler.CompleteTask(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	require.NoError(t, err)
	
	assert.True(t, task.Completed)
}

func TestTaskHandler_GetTasks_WithFilter(t *testing.T) {
	handler := setupTestHandler()
	
	// Add tasks with different completion status
	completedTask := setupTestTask()
	completedTask.ID = "completed-task"
	completedTask.Completed = true
	handler.storage.Create(&completedTask)
	
	incompleteTask := setupTestTask()
	incompleteTask.ID = "incomplete-task"
	incompleteTask.Completed = false
	handler.storage.Create(&incompleteTask)
	
	// Test filter for completed tasks
	req, err := http.NewRequest("GET", "/api/tasks?completed=true", nil)
	require.NoError(t, err)
	
	rr := httptest.NewRecorder()
	handler.GetTasks(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var response TaskListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 1, response.Count)
	assert.True(t, response.Tasks[0].Completed)
	assert.Equal(t, "completed-task", response.Tasks[0].ID)
}

func TestTaskHandler_GetTasks_InvalidFilter(t *testing.T) {
	handler := setupTestHandler()
	
	req, err := http.NewRequest("GET", "/api/tasks?completed=invalid", nil)
	require.NoError(t, err)
	
	rr := httptest.NewRecorder()
	handler.GetTasks(rr, req)
	
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	
	var errorResp ErrorResponse
	err = json.Unmarshal(rr.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	assert.Contains(t, errorResp.Message, "Invalid 'completed' parameter")
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// Integration test for the complete workflow
func TestTaskWorkflow_Integration(t *testing.T) {
	handler := setupTestHandler()
	
	// 1. Create a task
	createReq := CreateTaskRequest{
		Title:       "Integration Test Task",
		Description: "Testing the complete workflow",
	}
	
	jsonData, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler.CreateTask(rr, req)
	
	require.Equal(t, http.StatusCreated, rr.Code)
	
	var createdTask Task
	json.Unmarshal(rr.Body.Bytes(), &createdTask)
	
	// 2. Get the task
	req, _ = http.NewRequest("GET", "/api/tasks/"+createdTask.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": createdTask.ID})
	
	rr = httptest.NewRecorder()
	handler.GetTask(rr, req)
	
	require.Equal(t, http.StatusOK, rr.Code)
	
	// 3. Update the task
	updateReq := UpdateTaskRequest{
		Title: stringPtr("Updated Integration Test Task"),
	}
	
	jsonData, _ = json.Marshal(updateReq)
	req, _ = http.NewRequest("PATCH", "/api/tasks/"+createdTask.ID, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": createdTask.ID})
	
	rr = httptest.NewRecorder()
	handler.PatchTask(rr, req)
	
	require.Equal(t, http.StatusOK, rr.Code)
	
	// 4. Complete the task
	req, _ = http.NewRequest("PATCH", "/api/tasks/"+createdTask.ID+"/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": createdTask.ID})
	
	rr = httptest.NewRecorder()
	handler.CompleteTask(rr, req)
	
	require.Equal(t, http.StatusOK, rr.Code)
	
	var completedTask Task
	json.Unmarshal(rr.Body.Bytes(), &completedTask)
	assert.True(t, completedTask.Completed)
	
	// 5. Get all tasks and verify
	req, _ = http.NewRequest("GET", "/api/tasks", nil)
	
	rr = httptest.NewRecorder()
	handler.GetTasks(rr, req)
	
	require.Equal(t, http.StatusOK, rr.Code)
	
	var taskList TaskListResponse
	json.Unmarshal(rr.Body.Bytes(), &taskList)
	assert.Equal(t, 1, taskList.Count)
	assert.True(t, taskList.Tasks[0].Completed)
	
	// 6. Delete the task
	req, _ = http.NewRequest("DELETE", "/api/tasks/"+createdTask.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": createdTask.ID})
	
	rr = httptest.NewRecorder()
	handler.DeleteTask(rr, req)
	
	require.Equal(t, http.StatusNoContent, rr.Code)
	
	// 7. Verify task is deleted
	req, _ = http.NewRequest("GET", "/api/tasks", nil)
	
	rr = httptest.NewRecorder()
	handler.GetTasks(rr, req)
	
	require.Equal(t, http.StatusOK, rr.Code)
	
	json.Unmarshal(rr.Body.Bytes(), &taskList)
	assert.Equal(t, 0, taskList.Count)
}