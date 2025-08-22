# Lesson 4: Setting up Go Development Environment

## Learning Objectives
By the end of this lesson, you will be able to:
- Install and configure Go on your system
- Understand Go modules and dependency management
- Set up a development environment for API development
- Create your first Go HTTP server
- Use essential Go tools for development

## Installing Go

### Download and Install

Visit https://golang.org/dl/ and download the appropriate version for your operating system.

#### Windows
1. Download the Windows installer (.msi)
2. Run the installer and follow the prompts
3. Add Go to your PATH (usually done automatically)

#### macOS
```bash
# Using Homebrew (recommended)
brew install go

# Or download from official site
```

#### Linux
```bash
# Download and extract
wget https://golang.org/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH in ~/.bashrc or ~/.zshrc
export PATH=$PATH:/usr/local/go/bin
```

### Verify Installation
```bash
go version
# Should output: go version go1.21.5 linux/amd64
```

## Go Workspace and Environment

### Environment Variables

#### GOPATH (Legacy)
- Used in older Go versions
- Still useful for some tools
- Not required for modules

#### GOROOT
- Where Go is installed
- Usually set automatically
- Don't modify unless necessary

#### GOPROXY
- Module proxy for downloading dependencies
- Default: https://proxy.golang.org

### Setting up GOPATH (Optional)
```bash
# Add to ~/.bashrc or ~/.zshrc
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

## Go Modules

Go modules are the standard way to manage dependencies in Go projects.

### Creating a New Module
```bash
# Create project directory
mkdir my-api
cd my-api

# Initialize module
go mod init github.com/yourusername/my-api
```

This creates a `go.mod` file:
```go
module github.com/yourusername/my-api

go 1.21
```

### Adding Dependencies
```bash
# Add a dependency
go get github.com/gorilla/mux

# Add specific version
go get github.com/gorilla/mux@v1.8.0

# Add development dependencies
go get -t github.com/stretchr/testify
```

### go.mod File Structure
```go
module github.com/yourusername/my-api

go 1.21

require (
    github.com/gorilla/mux v1.8.0
    github.com/lib/pq v1.10.9
)

require (
    github.com/gorilla/context v1.1.1 // indirect
)
```

### Common Module Commands
```bash
# Download dependencies
go mod download

# Clean up unused dependencies
go mod tidy

# Verify dependencies
go mod verify

# Show dependency graph
go mod graph

# Update dependencies
go get -u ./...
```

## Essential Go Concepts for Web Development

### Package System
```go
// main.go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    fmt.Println("Starting server...")
}
```

### Imports
```go
import (
    "fmt"          // Standard library
    "net/http"     // Standard library
    
    "github.com/gorilla/mux"  // External dependency
    
    "myapp/internal/handlers" // Local package
)
```

### Basic HTTP Server
```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World!")
    })
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Structs for Data Models
```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Category    string  `json:"category"`
    InStock     bool    `json:"in_stock"`
}
```

### JSON Handling
```go
import (
    "encoding/json"
    "net/http"
)

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{
        ID:    1,
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

## Development Tools

### Go Tools
```bash
# Format code
go fmt ./...

# Run tests
go test ./...

# Build binary
go build

# Run without building
go run main.go

# Install dependencies
go install

# Check for issues
go vet ./...
```

### Useful External Tools

#### Air (Live Reload)
```bash
# Install
go install github.com/cosmtrek/air@latest

# Initialize config
air init

# Run with live reload
air
```

#### golangci-lint (Linting)
```bash
# Install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## IDE Setup

### Visual Studio Code
1. Install Go extension
2. Configure settings:
```json
{
    "go.useLanguageServer": true,
    "go.lintOnSave": "package",
    "go.formatTool": "goimports"
}
```

### GoLand (JetBrains)
- Built-in Go support
- Excellent debugging
- Integrated testing

### Vim/Neovim
- vim-go plugin
- LSP support with gopls

## Project Structure

### Standard Go Project Layout
```
my-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── handlers/
│   │   ├── users.go
│   │   └── products.go
│   ├── models/
│   │   ├── user.go
│   │   └── product.go
│   └── database/
│       └── connection.go
├── pkg/
│   └── utils/
│       └── helpers.go
├── api/
│   └── openapi.yaml
├── scripts/
├── tests/
├── docs/
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

### Directory Explanations
- **cmd/**: Main applications
- **internal/**: Private application code
- **pkg/**: Public library code
- **api/**: API definitions (OpenAPI, protobuf)
- **scripts/**: Build and deployment scripts

## Environment Configuration

### Environment Variables
```go
package main

import (
    "os"
    "strconv"
)

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func main() {
    port := getEnvInt("PORT", 8080)
    dbURL := os.Getenv("DATABASE_URL")
    
    // Use configuration...
}
```

### Using godotenv
```bash
go get github.com/joho/godotenv
```

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
)

func init() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
}

func main() {
    dbURL := os.Getenv("DATABASE_URL")
    // Use configuration...
}
```

## Creating Your First API Project

### Initialize Project
```bash
mkdir task-api
cd task-api
go mod init github.com/yourusername/task-api
```

### Create Basic Structure
```bash
mkdir -p cmd/api
mkdir -p internal/{handlers,models}
mkdir -p pkg/utils
```

### Basic main.go
```go
// cmd/api/main.go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    
    "github.com/gorilla/mux"
)

type Task struct {
    ID          int    `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Completed   bool   `json:"completed"`
}

var tasks []Task
var nextID = 1

func getTasks(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
    var task Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    task.ID = nextID
    nextID++
    tasks = append(tasks, task)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(task)
}

func main() {
    router := mux.NewRouter()
    
    router.HandleFunc("/api/tasks", getTasks).Methods("GET")
    router.HandleFunc("/api/tasks", createTask).Methods("POST")
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, router))
}
```

### Install Dependencies
```bash
go get github.com/gorilla/mux
go mod tidy
```

### Run the Server
```bash
go run cmd/api/main.go
```

### Test the API
```bash
# Get all tasks
curl http://localhost:8080/api/tasks

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go", "description": "Build REST API"}'
```

## Debugging in Go

### Using Delve Debugger
```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug cmd/api/main.go
```

### VS Code Debugging
```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch API",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/api/main.go"
        }
    ]
}
```

## Performance Profiling

### Basic Profiling
```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your API code...
}
```

Visit http://localhost:6060/debug/pprof/ for profiling data.

## Testing Setup

### Basic Test Structure
```go
// internal/handlers/users_test.go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetUsers(t *testing.T) {
    req, err := http.NewRequest("GET", "/api/users", nil)
    if err != nil {
        t.Fatal(err)
    }
    
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(getUsersHandler)
    
    handler.ServeHTTP(rr, req)
    
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
}
```

## Version Control Setup

### .gitignore
```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env

# IDE files
.vscode/
.idea/
*.swp
*.swo
```

## Best Practices

### Code Organization
- Keep main.go minimal
- Use internal/ for private code
- Group related functionality
- Follow Go naming conventions

### Error Handling
```go
func handler(w http.ResponseWriter, r *http.Request) {
    data, err := fetchData()
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        log.Printf("Error fetching data: %v", err)
        return
    }
    
    // Continue with success case...
}
```

### Configuration Management
- Use environment variables
- Provide sensible defaults
- Validate configuration at startup

## Next Steps

In the next lesson, we'll build upon this foundation to create a complete REST API with proper routing, middleware, and error handling.

## Key Takeaways

- Go modules are the standard for dependency management
- Project structure follows community conventions
- The net/http package provides powerful HTTP capabilities
- Development tools enhance productivity
- Testing is built into the Go ecosystem

## Practice Exercise

1. Set up a new Go project for a blog API
2. Create basic handlers for posts and authors
3. Implement JSON marshaling/unmarshaling
4. Add basic error handling
5. Write unit tests for your handlers