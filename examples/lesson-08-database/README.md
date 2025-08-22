# Lesson 8: Database Integration - Validation Examples

This directory contains examples to validate your understanding of database integration with PostgreSQL, Docker Compose setup, and production-ready patterns.

## Examples Included

1. **docker-compose.yml** - Complete development environment
2. **main.go** - Database-integrated task API
3. **repository/** - Repository pattern implementation
4. **migrations/** - Database schema migrations
5. **config/** - Database configuration management
6. **tests/** - Database integration tests

## Learning Objectives Validation

- ✅ Set up PostgreSQL with Docker Compose
- ✅ Implement repository pattern for data access
- ✅ Handle database transactions effectively
- ✅ Manage database migrations and schema changes
- ✅ Implement connection pooling and optimization
- ✅ Write testable database code

## Quick Start

### 1. Start the Environment

```bash
# Start all services (PostgreSQL, Redis, pgAdmin)
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f postgres
```

### 2. Run Database Migrations

```bash
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgresql://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable" up

# Check migration status
migrate -path migrations -database "postgresql://taskuser:taskpass@localhost:5432/taskapi?sslmode=disable" version
```

### 3. Run the Application

```bash
# Install dependencies
go mod download

# Run the API server
go run main.go

# Or use hot reload
air
```

### 4. Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| API Server | http://localhost:8088 | - |
| pgAdmin | http://localhost:8080 | admin@taskapi.com / admin |
| PostgreSQL | localhost:5432 | taskuser / taskpass |
| Redis | localhost:6379 | - |

## API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | User login |
| POST | `/api/auth/refresh` | Refresh JWT token |

### Users
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/users/me` | Get current user |
| PUT | `/api/users/me` | Update current user |
| GET | `/api/users` | List users (admin only) |

### Tasks
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/tasks` | Get user's tasks |
| POST | `/api/tasks` | Create new task |
| GET | `/api/tasks/{id}` | Get specific task |
| PUT | `/api/tasks/{id}` | Update task |
| DELETE | `/api/tasks/{id}` | Delete task |
| POST | `/api/tasks/bulk` | Bulk create tasks |

### Categories
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/categories` | Get user's categories |
| POST | `/api/categories` | Create category |
| PUT | `/api/categories/{id}` | Update category |
| DELETE | `/api/categories/{id}` | Delete category |

## Validation Exercises

### Exercise 1: Basic Database Operations

Test CRUD operations with database persistence:

```bash
# 1. Register a user
curl -X POST http://localhost:8088/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "firstName": "Test",
    "lastName": "User"
  }'

# 2. Login to get token
curl -X POST http://localhost:8088/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'

# 3. Create a task (use token from login)
curl -X POST http://localhost:8088/api/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Database Integration Test",
    "description": "Testing PostgreSQL integration",
    "priority": "high",
    "dueDate": "2024-12-31T23:59:59Z"
  }'

# 4. Get all tasks
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8088/api/tasks

# 5. Restart the server and verify data persistence
docker-compose restart app
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8088/api/tasks
```

### Exercise 2: Transaction Management

Test transactional operations:

```bash
# Create task with categories (atomic operation)
curl -X POST http://localhost:8088/api/tasks/with-categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Project Planning",
    "description": "Plan the next quarter",
    "priority": "high",
    "categoryNames": ["Work", "Planning", "Q4"]
  }'

# Verify categories were created
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8088/api/categories
```

### Exercise 3: Advanced Filtering and Search

Test complex database queries:

```bash
# Filter by completion status
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "http://localhost:8088/api/tasks?completed=false"

# Filter by priority
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "http://localhost:8088/api/tasks?priority=high"

# Search by title/description
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "http://localhost:8088/api/tasks?search=database"

# Combined filters with pagination
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "http://localhost:8088/api/tasks?completed=false&priority=high&limit=10&offset=0"

# Filter by due date
curl -H "Authorization: Bearer YOUR_TOKEN" \
     "http://localhost:8088/api/tasks?dueBefore=2024-12-31T23:59:59Z"
```

### Exercise 4: Connection Pool Monitoring

Monitor database connection health:

```bash
# Check database health
curl http://localhost:8088/health

# Get detailed database stats
curl http://localhost:8088/health/database

# Monitor connection pool metrics
curl http://localhost:8088/metrics
```

### Exercise 5: Error Handling

Test database error scenarios:

```bash
# Try to create user with duplicate email
curl -X POST http://localhost:8088/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "firstName": "Test",
    "lastName": "User"
  }'

# Try to access non-existent task
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8088/api/tasks/550e8400-e29b-41d4-a716-446655440999

# Try to update task with invalid data
curl -X PUT http://localhost:8088/api/tasks/TASK_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "",
    "priority": "invalid_priority"
  }'
```

## Database Administration

### Using pgAdmin

1. Open http://localhost:8080
2. Login with admin@taskapi.com / admin
3. Add server:
   - Host: postgres
   - Database: taskapi
   - Username: taskuser
   - Password: taskpass

### Direct PostgreSQL Access

```bash
# Connect to PostgreSQL
docker exec -it taskapi_postgres psql -U taskuser -d taskapi

# View tables
\dt

# Check user data
SELECT * FROM users;

# Check task data with categories
SELECT t.*, 
       array_agg(c.name) as categories
FROM tasks t
LEFT JOIN task_categories tc ON t.id = tc.task_id
LEFT JOIN categories c ON tc.category_id = c.id
GROUP BY t.id;

# Check connection activity
SELECT * FROM pg_stat_activity WHERE datname = 'taskapi';
```

## Testing

### Run Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with database integration
go test -tags=integration ./tests/integration/...

# Run tests with coverage
go test -cover ./...

# Run specific test suite
go test ./repository -v
```

### Run Load Tests

```bash
# Install hey for load testing
go install github.com/rakyll/hey@latest

# Test user registration endpoint
hey -n 100 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d '{"email":"user%d@test.com","password":"pass123","firstName":"Test","lastName":"User"}' \
    http://localhost:8088/api/auth/register

# Test authenticated endpoints (after getting token)
hey -n 1000 -c 20 \
    -H "Authorization: Bearer YOUR_TOKEN" \
    http://localhost:8088/api/tasks
```

## Migration Management

### Create New Migration

```bash
# Create new migration files
migrate create -ext sql -dir migrations -seq add_task_tags

# This creates:
# migrations/000003_add_task_tags.up.sql
# migrations/000003_add_task_tags.down.sql
```

### Migration Commands

```bash
# Check current version
migrate -path migrations -database $DATABASE_URL version

# Migrate up to latest
migrate -path migrations -database $DATABASE_URL up

# Migrate up by N steps
migrate -path migrations -database $DATABASE_URL up 2

# Migrate down by N steps
migrate -path migrations -database $DATABASE_URL down 1

# Force version (if migrations are dirty)
migrate -path migrations -database $DATABASE_URL force 2
```

## Performance Monitoring

### Database Metrics

Monitor these key metrics in production:

```sql
-- Connection pool usage
SELECT count(*) as active_connections FROM pg_stat_activity WHERE datname = 'taskapi';

-- Slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public';

-- Index usage
SELECT 
    indexrelname as index_name,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes 
ORDER BY idx_scan DESC;
```

### Application Metrics

```bash
# Check application metrics
curl http://localhost:8088/metrics

# Sample response includes:
# - HTTP request duration
# - Active database connections
# - Request rates
# - Error rates
```

## Troubleshooting

### Common Issues

#### Database Connection Failed
```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Test connection manually
docker exec -it taskapi_postgres pg_isready -U taskuser
```

#### Migration Errors
```bash
# Check migration status
migrate -path migrations -database $DATABASE_URL version

# Force specific version if dirty
migrate -path migrations -database $DATABASE_URL force 1

# Drop database and recreate (development only)
docker-compose down -v
docker-compose up -d postgres
# Wait for postgres to be ready, then run migrations
```

#### Performance Issues
```bash
# Check connection pool stats
curl http://localhost:8088/health/database

# Monitor slow queries in PostgreSQL
docker exec -it taskapi_postgres psql -U taskuser -d taskapi -c "SELECT * FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 5;"

# Check for lock contention
docker exec -it taskapi_postgres psql -U taskuser -d taskapi -c "SELECT * FROM pg_locks WHERE NOT granted;"
```

## Key Concepts Demonstrated

### 1. Repository Pattern
- Clean separation of data access logic
- Interface-based design for testability
- Consistent error handling patterns

### 2. Transaction Management
- ACID compliance for complex operations
- Proper rollback handling
- Isolation level management

### 3. Connection Pooling
- Optimized connection usage
- Health monitoring
- Performance tuning

### 4. Migration Management
- Version-controlled schema changes
- Forward and backward migrations
- Safe deployment practices

### 5. Error Handling
- Database-specific error interpretation
- Graceful degradation
- Detailed logging for debugging

## Production Readiness Checklist

- [ ] Connection pooling configured appropriately
- [ ] Database indexes on frequently queried columns
- [ ] Transaction boundaries properly defined
- [ ] Migration system in place
- [ ] Database backup strategy implemented
- [ ] Monitoring and alerting configured
- [ ] Connection timeout handling
- [ ] Prepared statements for security
- [ ] Input validation before database operations
- [ ] Database roles and permissions configured

## Next Steps

After mastering database integration:

1. **Caching Layer**: Add Redis caching for improved performance
2. **Read Replicas**: Scale reads with database replicas
3. **Connection Pooling**: Advanced pool configuration and monitoring
4. **Database Sharding**: Handle massive scale scenarios
5. **Backup/Recovery**: Implement automated backup strategies

This lesson bridges the gap between simple in-memory APIs and production-ready, database-backed services that can handle real-world loads and requirements.