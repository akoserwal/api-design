# Lesson 16: Deployment and Monitoring

## Learning Objectives
By the end of this lesson, you will be able to:
- Containerize Go APIs with Docker
- Deploy APIs to cloud platforms
- Implement health checks and graceful shutdown
- Set up logging and monitoring
- Configure CI/CD pipelines
- Implement observability with metrics and tracing

## Containerization with Docker

### Dockerfile for Go API
```dockerfile
# Multi-stage build for optimized image size
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# Start fresh from a smaller image
FROM alpine:3.18

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
```

### Optimized Dockerfile
```dockerfile
# Use distroless for even smaller size and better security
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main cmd/api/main.go

# Use distroless image
FROM gcr.io/distroless/static-debian11

WORKDIR /

# Copy binary
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Non-root user
USER 1000

# Run
ENTRYPOINT ["./main"]
```

### Docker Compose for Development
```yaml
# docker-compose.yml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=development
      - DB_HOST=postgres
      - DB_NAME=taskdb
      - DB_USER=taskuser
      - DB_PASSWORD=taskpass
      - JWT_SECRET=dev-secret-key
    depends_on:
      - postgres
      - redis
    networks:
      - app-network

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=taskdb
      - POSTGRES_USER=taskuser
      - POSTGRES_PASSWORD=taskpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - app-network

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    networks:
      - app-network

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - api
    networks:
      - app-network

volumes:
  postgres_data:

networks:
  app-network:
    driver: bridge
```

## Application Configuration

### Environment-based Configuration
```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Redis    RedisConfig
    Logging  LoggingConfig
}

type ServerConfig struct {
    Port           string
    Host           string
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    IdleTimeout    time.Duration
    MaxHeaderBytes int
}

type DatabaseConfig struct {
    Host     string
    Port     string
    Name     string
    User     string
    Password string
    SSLMode  string
    MaxConns int
    MaxIdle  int
}

type JWTConfig struct {
    Secret     string
    Issuer     string
    Expiration time.Duration
}

type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
}

type LoggingConfig struct {
    Level  string
    Format string
}

func Load() *Config {
    return &Config{
        Server: ServerConfig{
            Port:           getEnv("PORT", "8080"),
            Host:           getEnv("HOST", "0.0.0.0"),
            ReadTimeout:    getEnvDuration("READ_TIMEOUT", 30*time.Second),
            WriteTimeout:   getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
            IdleTimeout:    getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
            MaxHeaderBytes: getEnvInt("MAX_HEADER_BYTES", 1<<20),
        },
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", "5432"),
            Name:     getEnv("DB_NAME", "taskdb"),
            User:     getEnv("DB_USER", "taskuser"),
            Password: getEnv("DB_PASSWORD", ""),
            SSLMode:  getEnv("DB_SSL_MODE", "disable"),
            MaxConns: getEnvInt("DB_MAX_CONNS", 25),
            MaxIdle:  getEnvInt("DB_MAX_IDLE", 10),
        },
        JWT: JWTConfig{
            Secret:     getEnv("JWT_SECRET", ""),
            Issuer:     getEnv("JWT_ISSUER", "task-api"),
            Expiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
        },
        Redis: RedisConfig{
            Host:     getEnv("REDIS_HOST", "localhost"),
            Port:     getEnv("REDIS_PORT", "6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        },
        Logging: LoggingConfig{
            Level:  getEnv("LOG_LEVEL", "info"),
            Format: getEnv("LOG_FORMAT", "json"),
        },
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}
```

## Health Checks and Graceful Shutdown

### Health Check Implementation
```go
// internal/health/health.go
package health

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/go-redis/redis/v8"
)

type HealthChecker struct {
    db    *sql.DB
    redis *redis.Client
}

type HealthStatus struct {
    Status   string            `json:"status"`
    Version  string            `json:"version"`
    Uptime   string            `json:"uptime"`
    Checks   map[string]Check  `json:"checks"`
}

type Check struct {
    Status  string        `json:"status"`
    Message string        `json:"message,omitempty"`
    Latency time.Duration `json:"latency,omitempty"`
}

var startTime = time.Now()

func NewHealthChecker(db *sql.DB, redis *redis.Client) *HealthChecker {
    return &HealthChecker{
        db:    db,
        redis: redis,
    }
}

func (h *HealthChecker) HealthHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    status := h.checkHealth(ctx)
    
    w.Header().Set("Content-Type", "application/json")
    
    if status.Status == "healthy" {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(status)
}

func (h *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    checks := make(map[string]Check)
    
    // Check database
    if h.db != nil {
        checks["database"] = h.checkDatabase(ctx)
    }
    
    // Check Redis
    if h.redis != nil {
        checks["redis"] = h.checkRedis(ctx)
    }
    
    allHealthy := true
    for _, check := range checks {
        if check.Status != "healthy" {
            allHealthy = false
            break
        }
    }
    
    status := "ready"
    if !allHealthy {
        status = "not_ready"
    }
    
    response := map[string]interface{}{
        "status": status,
        "checks": checks,
    }
    
    w.Header().Set("Content-Type", "application/json")
    if allHealthy {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(response)
}

func (h *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "alive",
    })
}

func (h *HealthChecker) checkHealth(ctx context.Context) HealthStatus {
    checks := make(map[string]Check)
    
    // Check database
    if h.db != nil {
        checks["database"] = h.checkDatabase(ctx)
    }
    
    // Check Redis
    if h.redis != nil {
        checks["redis"] = h.checkRedis(ctx)
    }
    
    // Determine overall status
    overall := "healthy"
    for _, check := range checks {
        if check.Status != "healthy" {
            overall = "unhealthy"
            break
        }
    }
    
    return HealthStatus{
        Status:  overall,
        Version: getVersion(),
        Uptime:  time.Since(startTime).String(),
        Checks:  checks,
    }
}

func (h *HealthChecker) checkDatabase(ctx context.Context) Check {
    start := time.Now()
    
    err := h.db.PingContext(ctx)
    latency := time.Since(start)
    
    if err != nil {
        return Check{
            Status:  "unhealthy",
            Message: err.Error(),
            Latency: latency,
        }
    }
    
    return Check{
        Status:  "healthy",
        Latency: latency,
    }
}

func (h *HealthChecker) checkRedis(ctx context.Context) Check {
    start := time.Now()
    
    _, err := h.redis.Ping(ctx).Result()
    latency := time.Since(start)
    
    if err != nil {
        return Check{
            Status:  "unhealthy",
            Message: err.Error(),
            Latency: latency,
        }
    }
    
    return Check{
        Status:  "healthy",
        Latency: latency,
    }
}

func getVersion() string {
    // This could be set at build time with ldflags
    return os.Getenv("APP_VERSION")
}
```

### Graceful Shutdown
```go
// internal/server/server.go
package server

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

type Server struct {
    httpServer *http.Server
    config     *config.Config
}

func NewServer(handler http.Handler, config *config.Config) *Server {
    srv := &http.Server{
        Addr:           config.Server.Host + ":" + config.Server.Port,
        Handler:        handler,
        ReadTimeout:    config.Server.ReadTimeout,
        WriteTimeout:   config.Server.WriteTimeout,
        IdleTimeout:    config.Server.IdleTimeout,
        MaxHeaderBytes: config.Server.MaxHeaderBytes,
    }
    
    return &Server{
        httpServer: srv,
        config:     config,
    }
}

func (s *Server) Start() error {
    // Start server in goroutine
    go func() {
        log.Printf("Server starting on %s", s.httpServer.Addr)
        if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Server shutting down...")
    
    // Create context with timeout for shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Attempt graceful shutdown
    if err := s.httpServer.Shutdown(ctx); err != nil {
        log.Printf("Server forced to shutdown: %v", err)
        return err
    }
    
    log.Println("Server shutdown complete")
    return nil
}

func (s *Server) Stop() error {
    return s.httpServer.Close()
}
```

## Logging and Observability

### Structured Logging
```go
// internal/logging/logger.go
package logging

import (
    "os"
    
    "github.com/sirupsen/logrus"
)

type Logger struct {
    *logrus.Logger
}

func NewLogger(level, format string) *Logger {
    logger := logrus.New()
    
    // Set log level
    switch level {
    case "debug":
        logger.SetLevel(logrus.DebugLevel)
    case "info":
        logger.SetLevel(logrus.InfoLevel)
    case "warn":
        logger.SetLevel(logrus.WarnLevel)
    case "error":
        logger.SetLevel(logrus.ErrorLevel)
    default:
        logger.SetLevel(logrus.InfoLevel)
    }
    
    // Set log format
    if format == "json" {
        logger.SetFormatter(&logrus.JSONFormatter{
            TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
        })
    } else {
        logger.SetFormatter(&logrus.TextFormatter{
            FullTimestamp: true,
        })
    }
    
    logger.SetOutput(os.Stdout)
    
    return &Logger{logger}
}

// Middleware for request logging
func (l *Logger) RequestLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create a wrapper to capture status code
        ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        // Process request
        next.ServeHTTP(ww, r)
        
        // Log request details
        l.WithFields(logrus.Fields{
            "method":      r.Method,
            "url":         r.URL.String(),
            "status_code": ww.statusCode,
            "duration":    time.Since(start),
            "user_agent":  r.UserAgent(),
            "remote_addr": r.RemoteAddr,
        }).Info("HTTP request")
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
```

### Metrics with Prometheus
```go
// internal/metrics/metrics.go
package metrics

import (
    "net/http"
    "strconv"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

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
            Name:    "http_request_duration_seconds",
            Help:    "Duration of HTTP requests in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_connections",
            Help: "Number of active connections",
        },
    )
    
    databaseConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "database_connections",
            Help: "Number of database connections",
        },
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(activeConnections)
    prometheus.MustRegister(databaseConnections)
}

func PrometheusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(ww, r)
        
        duration := time.Since(start)
        statusCode := strconv.Itoa(ww.statusCode)
        
        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(duration.Seconds())
    })
}

func MetricsHandler() http.Handler {
    return promhttp.Handler()
}

func UpdateDatabaseConnections(count int) {
    databaseConnections.Set(float64(count))
}
```

## Cloud Deployment

### Kubernetes Deployment
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: task-api
  labels:
    app: task-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: task-api
  template:
    metadata:
      labels:
        app: task-api
    spec:
      containers:
      - name: task-api
        image: your-registry/task-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: DB_HOST
          value: "postgres-service"
        - name: DB_NAME
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: database
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: jwt-secret
              key: secret
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"

---
apiVersion: v1
kind: Service
metadata:
  name: task-api-service
spec:
  selector:
    app: task-api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: task-api-ingress
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - api.yourdomain.com
    secretName: task-api-tls
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: task-api-service
            port:
              number: 80
```

### AWS ECS Deployment
```json
{
  "family": "task-api",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "task-api",
      "image": "your-account.dkr.ecr.region.amazonaws.com/task-api:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        }
      ],
      "secrets": [
        {
          "name": "DB_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-password"
        },
        {
          "name": "JWT_SECRET",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:jwt-secret"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/task-api",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "curl -f http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

## CI/CD Pipeline

### GitHub Actions Workflow
```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: 1.21
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      env:
        DB_HOST: localhost
        DB_USER: postgres
        DB_PASSWORD: postgres
        DB_NAME: testdb
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
    
    - name: Run linter
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    
    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
    
    - name: Log in to Container Registry
      uses: docker/login-action@v2
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v4
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=sha,prefix=sha-
          type=raw,value=latest,enable={{is_default_branch}}
    
    - name: Build and push Docker image
      uses: docker/build-push-action@v4
      with:
        context: .
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}

  deploy:
    if: github.ref == 'refs/heads/main'
    needs: build
    runs-on: ubuntu-latest
    environment: production
    
    steps:
    - name: Deploy to production
      uses: azure/k8s-deploy@v1
      with:
        manifests: |
          k8s/deployment.yaml
        images: |
          ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:sha-${{ github.sha }}
```

### GitLab CI/CD Pipeline
```yaml
# .gitlab-ci.yml
stages:
  - test
  - build
  - deploy

variables:
  GO_VERSION: "1.21"
  DOCKER_DRIVER: overlay2

before_script:
  - export GOPATH=$CI_PROJECT_DIR/.go
  - mkdir -p $GOPATH

test:
  stage: test
  image: golang:$GO_VERSION
  services:
    - postgres:15
  variables:
    POSTGRES_DB: testdb
    POSTGRES_USER: testuser
    POSTGRES_PASSWORD: testpass
    DB_HOST: postgres
    DB_USER: testuser
    DB_PASSWORD: testpass
    DB_NAME: testdb
  script:
    - go mod download
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage: '/^total:\s+\(statements\)\s+(\d+\.\d+\%)/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml

lint:
  stage: test
  image: golangci/golangci-lint:latest
  script:
    - golangci-lint run

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - docker tag $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA $CI_REGISTRY_IMAGE:latest
    - docker push $CI_REGISTRY_IMAGE:latest
  only:
    - main

deploy:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl set image deployment/task-api task-api=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - kubectl rollout status deployment/task-api
  environment:
    name: production
    url: https://api.yourdomain.com
  only:
    - main
```

## Monitoring and Alerting

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'task-api'
    static_configs:
      - targets: ['task-api:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

rule_files:
  - "alert-rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Alert Rules
```yaml
# alert-rules.yml
groups:
- name: task-api-alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status_code=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }} for the last 5 minutes"

  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High latency detected"
      description: "95th percentile latency is {{ $value }}s"

  - alert: DatabaseDown
    expr: up{job="postgres"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Database is down"
      description: "PostgreSQL database is not responding"
```

### Grafana Dashboard
```json
{
  "dashboard": {
    "title": "Task API Dashboard",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status_code=~\"5..\"}[5m])",
            "legendFormat": "5xx errors"
          }
        ]
      }
    ]
  }
}
```

## Security in Production

### Security Headers Middleware
```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
```

### Rate Limiting
```go
func RateLimit(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(100), 10) // 100 requests per second, burst of 10
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## Best Practices Summary

### 1. Containerization
- Use multi-stage builds for smaller images
- Run as non-root user
- Use distroless base images when possible
- Set resource limits

### 2. Configuration
- Use environment variables for configuration
- Provide sensible defaults
- Validate configuration at startup
- Use secrets management

### 3. Health Checks
- Implement liveness and readiness probes
- Check dependencies in readiness probe
- Keep health checks lightweight
- Return appropriate status codes

### 4. Monitoring
- Implement structured logging
- Collect application metrics
- Set up alerting for critical issues
- Monitor business metrics

### 5. Security
- Use HTTPS everywhere
- Implement security headers
- Keep dependencies updated
- Regular security audits

## Next Steps and Course Completion

Congratulations! You've completed the REST API Design and Development course. You now have:

1. **Foundation Knowledge**: Understanding of REST principles and HTTP
2. **Design Skills**: API design patterns and best practices
3. **Implementation Experience**: Building APIs in Go
4. **Production Readiness**: Authentication, testing, and deployment

### Recommended Next Steps:
1. Build a complete project applying all concepts
2. Explore microservices architecture
3. Learn about GraphQL as an alternative to REST
4. Study distributed systems patterns
5. Contribute to open-source Go projects

## Key Takeaways

- Deployment requires careful consideration of containerization, configuration, and monitoring
- Health checks are essential for reliable deployments
- Observability (logging, metrics, tracing) is crucial for production systems
- CI/CD pipelines automate testing and deployment
- Security must be built into every layer
- Monitoring and alerting help maintain system reliability

## Practice Exercises

1. Containerize your task API with Docker
2. Set up a complete monitoring stack with Prometheus and Grafana
3. Create a CI/CD pipeline for automated deployment
4. Implement comprehensive health checks
5. Deploy your API to a cloud platform

This completes the comprehensive REST API course. You now have the knowledge and skills to build, test, secure, and deploy production-ready REST APIs using Go!