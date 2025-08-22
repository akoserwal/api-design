# Lesson 5: First REST API Validation Examples

This directory contains a complete working REST API implementation that demonstrates all the concepts learned in the first 5 lessons.

## Examples Included

1. **main.go** - Complete task management API (in-memory)
2. **main_with_database.go** - Database-backed version with PostgreSQL
3. **models.go** - Data models and validation
4. **storage.go** - In-memory storage implementation
5. **handlers.go** - HTTP handlers for all CRUD operations
6. **main_test.go** - Unit tests for the API
7. **docker-compose.yml** - Docker setup with PostgreSQL
8. **Dockerfile** - Container configuration for database version

## Learning Objectives Validation

- ✅ Build complete CRUD REST API
- ✅ Implement proper error handling
- ✅ Use appropriate HTTP methods and status codes
- ✅ Handle JSON marshaling/unmarshaling
- ✅ Implement input validation
- ✅ Write unit tests for API endpoints

## API Features

### Task Management
- Create, read, update, delete tasks
- Mark tasks as completed/incomplete
- Filter tasks by status
- Search tasks by title

### Error Handling
- Structured error responses
- Input validation
- Appropriate HTTP status codes

### Data Validation
- Required field validation
- Data type validation
- Business rule validation

## Running the API

### Option 1: In-Memory Version (Original)

```bash
# Install dependencies
go mod download

# Run the server with in-memory storage
go run main.go

# The API will be available at http://localhost:8087
```

### Option 2: Database Version (Enhanced)

```bash
# Start with Docker Compose (includes PostgreSQL)
docker-compose up -d

# Or run locally with existing PostgreSQL
DATABASE_URL="postgres://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable" go run main_with_database.go

# The API will be available at http://localhost:8080
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/tasks` | Get all tasks |
| GET | `/api/tasks?completed=true` | Get completed tasks |
| GET | `/api/tasks/{id}` | Get specific task |
| POST | `/api/tasks` | Create new task |
| PUT | `/api/tasks/{id}` | Update entire task |
| PATCH | `/api/tasks/{id}` | Partial task update |
| DELETE | `/api/tasks/{id}` | Delete task |
| PATCH | `/api/tasks/{id}/complete` | Mark task as completed |
| PATCH | `/api/tasks/{id}/uncomplete` | Mark task as incomplete |

## Validation Exercises

### Exercise 1: CRUD Operations
Test all CRUD operations and verify proper responses:

```bash
# Create a task
curl -X POST http://localhost:8087/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Learn REST APIs",
    "description": "Complete the REST API course"
  }'

# Get all tasks
curl http://localhost:8087/api/tasks

# Get specific task (use ID from create response)
curl http://localhost:8087/api/tasks/{task-id}

# Update task
curl -X PUT http://localhost:8087/api/tasks/{task-id} \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Master REST APIs",
    "description": "Complete the REST API course and build a project",
    "completed": true
  }'

# Partial update
curl -X PATCH http://localhost:8087/api/tasks/{task-id} \
  -H "Content-Type: application/json" \
  -d '{"completed": false}'

# Delete task
curl -X DELETE http://localhost:8087/api/tasks/{task-id}
```

### Exercise 2: Error Handling
Test various error scenarios:

```bash
# Invalid JSON
curl -X POST http://localhost:8087/api/tasks \
  -H "Content-Type: application/json" \
  -d 'invalid json'

# Missing required field
curl -X POST http://localhost:8087/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"description": "No title provided"}'

# Invalid task ID
curl http://localhost:8087/api/tasks/invalid-id

# Non-existent task
curl http://localhost:8087/api/tasks/99999
```

### Exercise 3: Filtering and Status Codes
Test filtering and verify status codes:

```bash
# Get completed tasks
curl http://localhost:8087/api/tasks?completed=true

# Get incomplete tasks
curl http://localhost:8087/api/tasks?completed=false

# Mark task as completed
curl -X PATCH http://localhost:8087/api/tasks/{task-id}/complete

# Verify status codes
curl -I http://localhost:8087/api/tasks
curl -I -X POST http://localhost:8087/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Task", "description": "Test"}'
```

### Exercise 4: Testing
Run the unit tests:

```bash
# Run all tests
go test -v

# Run specific test
go test -run TestCreateTask -v

# Run tests with coverage
go test -cover
```

## Key Concepts Demonstrated

### HTTP Methods Usage
- **GET**: Retrieve tasks (safe, idempotent)
- **POST**: Create new tasks (not safe, not idempotent)
- **PUT**: Replace entire task (not safe, idempotent)
- **PATCH**: Partial updates (not safe, may be idempotent)
- **DELETE**: Remove tasks (not safe, idempotent)

### Status Codes
- **200 OK**: Successful GET, PUT, PATCH
- **201 Created**: Successful POST
- **204 No Content**: Successful DELETE
- **400 Bad Request**: Invalid input
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Server errors

### Error Handling
- Structured error responses
- Meaningful error messages
- Proper status codes
- Input validation

### JSON Processing
- Request unmarshaling
- Response marshaling
- Struct tags for JSON fields
- Custom JSON field names

## Expected Behaviors

1. **Data Persistence**: Tasks persist in memory during server lifetime
2. **Validation**: Required fields are enforced
3. **Error Responses**: Consistent error format across all endpoints
4. **Status Filtering**: Can filter tasks by completion status
5. **Idempotency**: PUT and DELETE operations are idempotent
6. **Unique IDs**: Each task gets a unique identifier

## Testing Your Understanding

After running the examples, answer these questions:

1. What happens when you try to create a task without a title?
2. How does the API handle updating a non-existent task?
3. What's the difference between PUT and PATCH for task updates?
4. Why does DELETE return 204 instead of 200?
5. How does filtering work with query parameters?

## Database Version Features

The database version (`main_with_database.go`) demonstrates:

- **PostgreSQL Integration**: Real database persistence
- **Repository Pattern**: Clean data access layer separation
- **Connection Pooling**: Optimized database connections
- **Schema Management**: Automatic table creation
- **Error Handling**: Database-specific error handling
- **Docker Support**: Easy development environment setup

### Database Validation Exercises

```bash
# Test persistence across restarts
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Persistent Task", "description": "This will survive restarts"}'

# Restart the container
docker-compose restart app

# Verify data persists
curl http://localhost:8080/api/tasks
```

## Next Steps

- Add more sophisticated validation
- ✅ Implement database persistence (completed)
- Add authentication and authorization
- Create more complex filtering options
- Add pagination for large result sets
- Implement caching strategies
- Add database migrations