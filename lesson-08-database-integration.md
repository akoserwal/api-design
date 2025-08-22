# Lesson 8: Database Integration

## Learning Objectives
By the end of this lesson, you will be able to:
- Set up PostgreSQL with Docker Compose for development
- Implement proper database connection management
- Use the repository pattern for data access
- Handle database transactions effectively
- Manage database migrations and schema changes
- Implement connection pooling and performance optimization
- Handle database errors gracefully
- Write testable database code

## Why Database Integration Matters

Moving from in-memory storage to a real database is crucial for production APIs:
- **Persistence**: Data survives server restarts
- **Scalability**: Handle large datasets efficiently
- **Concurrency**: Multiple users can access data safely
- **ACID Compliance**: Ensure data consistency and reliability
- **Advanced Queries**: Complex filtering, sorting, and aggregation

## Development Environment Setup

### Docker Compose Configuration

Create a complete development environment with PostgreSQL, Redis, and database administration tools.

```yaml
# docker-compose.yml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:15-alpine
    container_name: taskapi_postgres
    environment:
      POSTGRES_DB: taskapi
      POSTGRES_USER: taskuser
      POSTGRES_PASSWORD: taskpass
      POSTGRES_HOST_AUTH_METHOD: trust
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/01-init.sql
      - ./scripts/seed.sql:/docker-entrypoint-initdb.d/02-seed.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U taskuser -d taskapi"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - taskapi_network

  # Redis for Caching
  redis:
    image: redis:7-alpine
    container_name: taskapi_redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - taskapi_network

  # pgAdmin for Database Management
  pgadmin:
    image: dpage/pgadmin4:latest
    container_name: taskapi_pgadmin
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@taskapi.com
      PGADMIN_DEFAULT_PASSWORD: admin
      PGADMIN_CONFIG_SERVER_MODE: 'False'
    ports:
      - "8080:80"
    volumes:
      - pgadmin_data:/var/lib/pgadmin
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - taskapi_network

  # Application (for development)
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    container_name: taskapi_app
    environment:
      - DATABASE_URL=postgres://taskuser:taskpass@postgres:5432/taskapi?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - PORT=8080
      - APP_ENV=development
    ports:
      - "8080:8080"
    volumes:
      - .:/app
      - /app/tmp
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - taskapi_network
    command: air # Hot reload for development

volumes:
  postgres_data:
  redis_data:
  pgadmin_data:

networks:
  taskapi_network:
    driver: bridge
```

### Database Initialization Scripts

```sql
-- scripts/init.sql
-- Database schema initialization

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    completed BOOLEAN NOT NULL DEFAULT false,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    due_date TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7), -- Hex color code
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, user_id)
);

-- Task categories junction table (many-to-many)
CREATE TABLE task_categories (
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, category_id)
);

-- API Keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permissions TEXT[], -- Array of permissions
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_completed ON tasks(completed);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

```sql
-- scripts/seed.sql
-- Sample data for development

-- Insert sample users
INSERT INTO users (id, email, password_hash, first_name, last_name, role) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'admin@taskapi.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye', 'Admin', 'User', 'admin'),
    ('550e8400-e29b-41d4-a716-446655440002', 'john@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye', 'John', 'Doe', 'user'),
    ('550e8400-e29b-41d4-a716-446655440003', 'jane@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye', 'Jane', 'Smith', 'user');

-- Insert sample categories
INSERT INTO categories (id, name, color, user_id) VALUES
    ('650e8400-e29b-41d4-a716-446655440001', 'Work', '#FF6B6B', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440002', 'Personal', '#4ECDC4', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440003', 'Learning', '#45B7D1', '550e8400-e29b-41d4-a716-446655440002');

-- Insert sample tasks
INSERT INTO tasks (id, title, description, completed, priority, user_id, due_date) VALUES
    ('750e8400-e29b-41d4-a716-446655440001', 'Complete REST API Course', 'Finish all lessons and examples', false, 'high', '550e8400-e29b-41d4-a716-446655440002', '2024-12-31 23:59:59+00'),
    ('750e8400-e29b-41d4-a716-446655440002', 'Review database patterns', 'Study transaction management and connection pooling', false, 'medium', '550e8400-e29b-41d4-a716-446655440002', '2024-12-25 17:00:00+00'),
    ('750e8400-e29b-41d4-a716-446655440003', 'Setup development environment', 'Configure Docker Compose and database', true, 'high', '550e8400-e29b-41d4-a716-446655440002', NULL);

-- Link tasks to categories
INSERT INTO task_categories (task_id, category_id) VALUES
    ('750e8400-e29b-41d4-a716-446655440001', '650e8400-e29b-41d4-a716-446655440003'),
    ('750e8400-e29b-41d4-a716-446655440002', '650e8400-e29b-41d4-a716-446655440003'),
    ('750e8400-e29b-41d4-a716-446655440003', '650e8400-e29b-41d4-a716-446655440001');
```

### Development Dockerfile

```dockerfile
# Dockerfile.dev
FROM golang:1.21-alpine AS development

# Install air for hot reloading
RUN go install github.com/cosmtrek/air@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Use air for hot reloading
CMD ["air"]
```

## Database Connection Management

### Configuration Structure

```go
// internal/config/database.go
package config

import (
    "fmt"
    "time"
)

type DatabaseConfig struct {
    Host            string        `env:"DB_HOST" envDefault:"localhost"`
    Port            int           `env:"DB_PORT" envDefault:"5432"`
    User            string        `env:"DB_USER" envDefault:"taskuser"`
    Password        string        `env:"DB_PASSWORD" envDefault:"taskpass"`
    Name            string        `env:"DB_NAME" envDefault:"taskapi"`
    SSLMode         string        `env:"DB_SSL_MODE" envDefault:"disable"`
    MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
    MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
    ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"1h"`
    ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"30m"`
}

func (c DatabaseConfig) ConnectionString() string {
    return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

func (c DatabaseConfig) Validate() error {
    if c.Host == "" {
        return fmt.Errorf("database host is required")
    }
    if c.User == "" {
        return fmt.Errorf("database user is required")
    }
    if c.Name == "" {
        return fmt.Errorf("database name is required")
    }
    if c.MaxOpenConns <= 0 {
        return fmt.Errorf("max open connections must be positive")
    }
    if c.MaxIdleConns <= 0 {
        return fmt.Errorf("max idle connections must be positive")
    }
    if c.MaxIdleConns > c.MaxOpenConns {
        return fmt.Errorf("max idle connections cannot exceed max open connections")
    }
    return nil
}
```

### Database Connection Setup

```go
// internal/database/connection.go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    _ "github.com/lib/pq" // PostgreSQL driver
    "github.com/yourusername/task-api/internal/config"
)

type DB struct {
    *sql.DB
    config config.DatabaseConfig
}

func NewConnection(cfg config.DatabaseConfig) (*DB, error) {
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid database config: %w", err)
    }

    db, err := sql.Open("postgres", cfg.ConnectionString())
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

    // Test the connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{DB: db, config: cfg}, nil
}

func (db *DB) Close() error {
    return db.DB.Close()
}

func (db *DB) HealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    return db.PingContext(ctx)
}

func (db *DB) Stats() sql.DBStats {
    return db.DB.Stats()
}
```

## Repository Pattern Implementation

### Base Repository Interface

```go
// internal/repository/interfaces.go
package repository

import (
    "context"
    "database/sql"
    "github.com/yourusername/task-api/internal/models"
)

type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id string) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filters ListFilters) ([]*models.User, error)
}

type TaskRepository interface {
    Create(ctx context.Context, task *models.Task) error
    GetByID(ctx context.Context, id string) (*models.Task, error)
    GetByUserID(ctx context.Context, userID string, filters TaskFilters) ([]*models.Task, error)
    Update(ctx context.Context, task *models.Task) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filters TaskFilters) ([]*models.Task, error)
    Count(ctx context.Context, filters TaskFilters) (int64, error)
}

type CategoryRepository interface {
    Create(ctx context.Context, category *models.Category) error
    GetByID(ctx context.Context, id string) (*models.Category, error)
    GetByUserID(ctx context.Context, userID string) ([]*models.Category, error)
    Update(ctx context.Context, category *models.Category) error
    Delete(ctx context.Context, id string) error
}

// Transaction interface for managing database transactions
type Transactor interface {
    WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error
}

// Filter types
type ListFilters struct {
    Limit  int
    Offset int
    Search string
}

type TaskFilters struct {
    ListFilters
    UserID      string
    Completed   *bool
    Priority    string
    CategoryIDs []string
    DueBefore   *time.Time
    DueAfter    *time.Time
}
```

### User Repository Implementation

```go
// internal/repository/user_repository.go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/lib/pq"
    "github.com/yourusername/task-api/internal/models"
)

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
    query := `
        INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, email_verified)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING created_at, updated_at`

    err := r.db.QueryRowContext(ctx, query,
        user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
        user.Role, user.IsActive, user.EmailVerified,
    ).Scan(&user.CreatedAt, &user.UpdatedAt)

    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok {
            switch pqErr.Code {
            case "23505": // unique_violation
                if strings.Contains(pqErr.Detail, "email") {
                    return fmt.Errorf("user with email %s already exists", user.Email)
                }
            }
        }
        return fmt.Errorf("failed to create user: %w", err)
    }

    return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
    user := &models.User{}
    query := `
        SELECT id, email, password_hash, first_name, last_name, role, 
               is_active, email_verified, created_at, updated_at
        FROM users 
        WHERE id = $1`

    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
        &user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user with id %s not found", id)
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    user := &models.User{}
    query := `
        SELECT id, email, password_hash, first_name, last_name, role, 
               is_active, email_verified, created_at, updated_at
        FROM users 
        WHERE email = $1`

    err := r.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
        &user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user with email %s not found", email)
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
    query := `
        UPDATE users 
        SET email = $2, password_hash = $3, first_name = $4, last_name = $5, 
            role = $6, is_active = $7, email_verified = $8, updated_at = CURRENT_TIMESTAMP
        WHERE id = $1
        RETURNING updated_at`

    err := r.db.QueryRowContext(ctx, query,
        user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
        user.Role, user.IsActive, user.EmailVerified,
    ).Scan(&user.UpdatedAt)

    if err != nil {
        if err == sql.ErrNoRows {
            return fmt.Errorf("user with id %s not found", user.ID)
        }
        return fmt.Errorf("failed to update user: %w", err)
    }

    return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM users WHERE id = $1`
    
    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete user: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to check rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return fmt.Errorf("user with id %s not found", id)
    }

    return nil
}

func (r *userRepository) List(ctx context.Context, filters ListFilters) ([]*models.User, error) {
    var conditions []string
    var args []interface{}
    argIndex := 1

    query := `
        SELECT id, email, password_hash, first_name, last_name, role, 
               is_active, email_verified, created_at, updated_at
        FROM users`

    if filters.Search != "" {
        conditions = append(conditions, fmt.Sprintf(
            "(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)",
            argIndex, argIndex+1, argIndex+2))
        searchTerm := "%" + filters.Search + "%"
        args = append(args, searchTerm, searchTerm, searchTerm)
        argIndex += 3
    }

    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }

    query += " ORDER BY created_at DESC"

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
        return nil, fmt.Errorf("failed to list users: %w", err)
    }
    defer rows.Close()

    var users []*models.User
    for rows.Next() {
        user := &models.User{}
        err := rows.Scan(
            &user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
            &user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return users, nil
}
```

### Task Repository Implementation

```go
// internal/repository/task_repository.go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/lib/pq"
    "github.com/yourusername/task-api/internal/models"
)

type taskRepository struct {
    db *sql.DB
}

func NewTaskRepository(db *sql.DB) TaskRepository {
    return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *models.Task) error {
    query := `
        INSERT INTO tasks (id, title, description, completed, priority, due_date, user_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING created_at, updated_at`

    err := r.db.QueryRowContext(ctx, query,
        task.ID, task.Title, task.Description, task.Completed, 
        task.Priority, task.DueDate, task.UserID,
    ).Scan(&task.CreatedAt, &task.UpdatedAt)

    if err != nil {
        return fmt.Errorf("failed to create task: %w", err)
    }

    return nil
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
    task := &models.Task{}
    query := `
        SELECT t.id, t.title, t.description, t.completed, t.priority, 
               t.due_date, t.user_id, t.created_at, t.updated_at,
               COALESCE(array_agg(c.id) FILTER (WHERE c.id IS NOT NULL), '{}') as category_ids,
               COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as category_names
        FROM tasks t
        LEFT JOIN task_categories tc ON t.id = tc.task_id
        LEFT JOIN categories c ON tc.category_id = c.id
        WHERE t.id = $1
        GROUP BY t.id, t.title, t.description, t.completed, t.priority, 
                 t.due_date, t.user_id, t.created_at, t.updated_at`

    var categoryIDs, categoryNames pq.StringArray
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &task.ID, &task.Title, &task.Description, &task.Completed, &task.Priority,
        &task.DueDate, &task.UserID, &task.CreatedAt, &task.UpdatedAt,
        &categoryIDs, &categoryNames,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("task with id %s not found", id)
        }
        return nil, fmt.Errorf("failed to get task: %w", err)
    }

    // Convert category arrays to structs
    for i, id := range categoryIDs {
        if id != "" && i < len(categoryNames) {
            task.Categories = append(task.Categories, models.Category{
                ID:   id,
                Name: categoryNames[i],
            })
        }
    }

    return task, nil
}

func (r *taskRepository) GetByUserID(ctx context.Context, userID string, filters TaskFilters) ([]*models.Task, error) {
    filters.UserID = userID
    return r.List(ctx, filters)
}

func (r *taskRepository) Update(ctx context.Context, task *models.Task) error {
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
            return fmt.Errorf("task with id %s not found", task.ID)
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
        return fmt.Errorf("task with id %s not found", id)
    }

    return nil
}

func (r *taskRepository) List(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
    var conditions []string
    var args []interface{}
    argIndex := 1

    baseQuery := `
        SELECT t.id, t.title, t.description, t.completed, t.priority, 
               t.due_date, t.user_id, t.created_at, t.updated_at,
               COALESCE(array_agg(c.id) FILTER (WHERE c.id IS NOT NULL), '{}') as category_ids,
               COALESCE(array_agg(c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as category_names
        FROM tasks t
        LEFT JOIN task_categories tc ON t.id = tc.task_id
        LEFT JOIN categories c ON tc.category_id = c.id`

    // Apply filters
    if filters.UserID != "" {
        conditions = append(conditions, fmt.Sprintf("t.user_id = $%d", argIndex))
        args = append(args, filters.UserID)
        argIndex++
    }

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

    if filters.DueBefore != nil {
        conditions = append(conditions, fmt.Sprintf("t.due_date <= $%d", argIndex))
        args = append(args, *filters.DueBefore)
        argIndex++
    }

    if filters.DueAfter != nil {
        conditions = append(conditions, fmt.Sprintf("t.due_date >= $%d", argIndex))
        args = append(args, *filters.DueAfter)
        argIndex++
    }

    if len(filters.CategoryIDs) > 0 {
        conditions = append(conditions, fmt.Sprintf("tc.category_id = ANY($%d)", argIndex))
        args = append(args, pq.Array(filters.CategoryIDs))
        argIndex++
    }

    query := baseQuery
    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }

    query += ` GROUP BY t.id, t.title, t.description, t.completed, t.priority, 
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

    var tasks []*models.Task
    for rows.Next() {
        task := &models.Task{}
        var categoryIDs, categoryNames pq.StringArray
        
        err := rows.Scan(
            &task.ID, &task.Title, &task.Description, &task.Completed, &task.Priority,
            &task.DueDate, &task.UserID, &task.CreatedAt, &task.UpdatedAt,
            &categoryIDs, &categoryNames,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }

        // Convert category arrays to structs
        for i, id := range categoryIDs {
            if id != "" && i < len(categoryNames) {
                task.Categories = append(task.Categories, models.Category{
                    ID:   id,
                    Name: categoryNames[i],
                })
            }
        }

        tasks = append(tasks, task)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return tasks, nil
}

func (r *taskRepository) Count(ctx context.Context, filters TaskFilters) (int64, error) {
    var conditions []string
    var args []interface{}
    argIndex := 1

    query := `SELECT COUNT(DISTINCT t.id) FROM tasks t`

    if len(filters.CategoryIDs) > 0 {
        query += " LEFT JOIN task_categories tc ON t.id = tc.task_id"
    }

    // Apply same filters as List method
    if filters.UserID != "" {
        conditions = append(conditions, fmt.Sprintf("t.user_id = $%d", argIndex))
        args = append(args, filters.UserID)
        argIndex++
    }

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

    if len(filters.CategoryIDs) > 0 {
        conditions = append(conditions, fmt.Sprintf("tc.category_id = ANY($%d)", argIndex))
        args = append(args, pq.Array(filters.CategoryIDs))
        argIndex++
    }

    if len(conditions) > 0 {
        query += " WHERE " + strings.Join(conditions, " AND ")
    }

    var count int64
    err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("failed to count tasks: %w", err)
    }

    return count, nil
}
```

## Transaction Management

### Transaction Wrapper

```go
// internal/database/transaction.go
package database

import (
    "context"
    "database/sql"
    "fmt"
)

type TransactionManager struct {
    db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
    return &TransactionManager{db: db}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p) // Re-throw panic after rollback
        }
    }()

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
        }
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}

// WithTransactionIsolation allows specifying isolation level
func (tm *TransactionManager) WithTransactionIsolation(ctx context.Context, isolation sql.IsolationLevel, fn func(tx *sql.Tx) error) error {
    tx, err := tm.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: isolation,
    })
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
        }
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

### Service Layer with Transactions

```go
// internal/service/task_service.go
package service

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/google/uuid"
    "github.com/yourusername/task-api/internal/models"
    "github.com/yourusername/task-api/internal/repository"
)

type TaskService struct {
    taskRepo     repository.TaskRepository
    categoryRepo repository.CategoryRepository
    userRepo     repository.UserRepository
    txManager    *TransactionManager
}

func NewTaskService(
    taskRepo repository.TaskRepository,
    categoryRepo repository.CategoryRepository,
    userRepo repository.UserRepository,
    txManager *TransactionManager,
) *TaskService {
    return &TaskService{
        taskRepo:     taskRepo,
        categoryRepo: categoryRepo,
        userRepo:     userRepo,
        txManager:    txManager,
    }
}

func (s *TaskService) CreateTaskWithCategories(ctx context.Context, req CreateTaskWithCategoriesRequest) (*models.Task, error) {
    // Validate user exists
    if _, err := s.userRepo.GetByID(ctx, req.UserID); err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }

    var createdTask *models.Task

    // Use transaction to ensure consistency
    err := s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        // Create task
        task := &models.Task{
            ID:          uuid.New().String(),
            Title:       req.Title,
            Description: req.Description,
            Priority:    req.Priority,
            DueDate:     req.DueDate,
            UserID:      req.UserID,
            Completed:   false,
        }

        if err := s.taskRepo.Create(ctx, task); err != nil {
            return fmt.Errorf("failed to create task: %w", err)
        }

        // Create categories if they don't exist and link them
        if len(req.CategoryNames) > 0 {
            categoryIDs, err := s.ensureCategoriesExist(ctx, tx, req.CategoryNames, req.UserID)
            if err != nil {
                return fmt.Errorf("failed to handle categories: %w", err)
            }

            if err := s.linkTaskToCategories(ctx, tx, task.ID, categoryIDs); err != nil {
                return fmt.Errorf("failed to link categories: %w", err)
            }
        }

        createdTask = task
        return nil
    })

    if err != nil {
        return nil, err
    }

    // Fetch the complete task with categories
    return s.taskRepo.GetByID(ctx, createdTask.ID)
}

func (s *TaskService) ensureCategoriesExist(ctx context.Context, tx *sql.Tx, categoryNames []string, userID string) ([]string, error) {
    var categoryIDs []string

    for _, name := range categoryNames {
        // Try to get existing category
        existingCategories, err := s.categoryRepo.GetByUserID(ctx, userID)
        if err != nil {
            return nil, err
        }

        var categoryID string
        for _, cat := range existingCategories {
            if cat.Name == name {
                categoryID = cat.ID
                break
            }
        }

        // Create category if it doesn't exist
        if categoryID == "" {
            category := &models.Category{
                ID:     uuid.New().String(),
                Name:   name,
                UserID: userID,
            }

            if err := s.categoryRepo.Create(ctx, category); err != nil {
                return nil, err
            }
            categoryID = category.ID
        }

        categoryIDs = append(categoryIDs, categoryID)
    }

    return categoryIDs, nil
}

func (s *TaskService) linkTaskToCategories(ctx context.Context, tx *sql.Tx, taskID string, categoryIDs []string) error {
    for _, categoryID := range categoryIDs {
        query := `INSERT INTO task_categories (task_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
        if _, err := tx.ExecContext(ctx, query, taskID, categoryID); err != nil {
            return fmt.Errorf("failed to link task to category: %w", err)
        }
    }
    return nil
}

type CreateTaskWithCategoriesRequest struct {
    Title         string
    Description   string
    Priority      string
    DueDate       *time.Time
    UserID        string
    CategoryNames []string
}
```

## Database Migrations

### Migration Framework

```go
// internal/database/migrate.go
package database

import (
    "database/sql"
    "fmt"
    "log"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB, migrationsPath string) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("could not create migration driver: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        fmt.Sprintf("file://%s", migrationsPath),
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("could not create migrate instance: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("could not run migrations: %w", err)
    }

    log.Println("Database migrations completed successfully")
    return nil
}

func RollbackMigrations(db *sql.DB, migrationsPath string, steps int) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("could not create migration driver: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        fmt.Sprintf("file://%s", migrationsPath),
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("could not create migrate instance: %w", err)
    }

    if err := m.Steps(-steps); err != nil {
        return fmt.Errorf("could not rollback migrations: %w", err)
    }

    log.Printf("Rolled back %d migration steps", steps)
    return nil
}
```

### Migration Files

```sql
-- migrations/000001_create_users_table.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

```sql
-- migrations/000001_create_users_table.down.sql
DROP TABLE IF EXISTS users;
```

## Quick Start Guide

### 1. Start Development Environment

```bash
# Clone the repository
git clone <your-repo>
cd task-api

# Start all services
docker-compose up -d

# Check service health
docker-compose ps
```

### 2. Access Services

- **API**: http://localhost:8080
- **pgAdmin**: http://localhost:8080 (admin@taskapi.com / admin)
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### 3. Run Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgresql://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable" up
```

### 4. Test Database Connection

```bash
# Test API endpoints
curl http://localhost:8080/health

# Create a user
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","firstName":"Test","lastName":"User"}'

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"title":"Test Task","description":"Testing database integration","priority":"high"}'
```

## Best Practices Summary

### 1. Connection Management
- ✅ Use connection pooling with appropriate limits
- ✅ Set connection timeouts and max lifetime
- ✅ Monitor connection pool metrics
- ✅ Handle connection failures gracefully

### 2. Query Performance
- ✅ Use proper indexes on frequently queried columns
- ✅ Avoid N+1 query problems with JOIN statements
- ✅ Use LIMIT and OFFSET for pagination
- ✅ Profile slow queries and optimize

### 3. Transaction Management
- ✅ Keep transactions short and focused
- ✅ Handle rollbacks properly
- ✅ Use appropriate isolation levels
- ✅ Avoid nested transactions

### 4. Error Handling
- ✅ Check for specific database errors (unique violations, etc.)
- ✅ Provide meaningful error messages to clients
- ✅ Log detailed errors for debugging
- ✅ Handle connection timeouts and retries

### 5. Security
- ✅ Use prepared statements to prevent SQL injection
- ✅ Validate input data before database operations
- ✅ Store sensitive data securely (passwords, API keys)
- ✅ Use database roles and permissions appropriately

## Next Steps

In the next lesson, we'll enhance this database integration with advanced features like caching, performance monitoring, and database optimization techniques.

## Key Takeaways

- Database integration transforms your API from prototype to production-ready
- Repository pattern provides clean separation between business logic and data access
- Transaction management ensures data consistency for complex operations
- Connection pooling and proper configuration are crucial for performance
- Docker Compose simplifies development environment setup
- Proper error handling and logging are essential for production systems