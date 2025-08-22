# REST API Course - Validation Guide

This guide helps you validate your understanding of each lesson through hands-on code examples and exercises.

## 🎯 Quick Start

```bash
cd examples
./run-all-examples.sh
```

The interactive script will guide you through all examples.

## 📚 Available Examples

### ✅ Completed Examples

| Lesson | Topic | Port | Status | Key Concepts |
|--------|-------|------|--------|--------------|
| 1 | REST Concepts | 8080-8082 | ✅ Complete | Richardson Maturity Model, HATEOAS, REST Principles |
| 2 | HTTP Fundamentals | 8083-8086 | ✅ Complete | HTTP Methods, Status Codes, Headers, URL Structure |
| 5 | First REST API | 8087 | ✅ Complete | Complete CRUD API, Testing, Error Handling |

## 🔧 Manual Setup

If you prefer to run examples manually:

```bash
# Install dependencies for all examples
find . -name "go.mod" -execdir go mod download \;

# Run a specific example
cd lesson-05-first-api
go run main.go

# Run tests
go test -v
```

## 📋 Validation Checklist

### Lesson 1: REST Concepts
- [ ] Understand Richardson Maturity Model levels 0-3
- [ ] Implement HATEOAS navigation
- [ ] Apply REST architectural principles
- [ ] Distinguish REST from SOAP and GraphQL

**Test Commands:**
```bash
# Level 0 (RPC-style)
curl -X POST http://localhost:8080/level0 -d '{"action":"getUsers"}' -H "Content-Type: application/json"

# Level 2 (HTTP verbs + status codes)
curl http://localhost:8080/level2/users

# HATEOAS Navigation
curl http://localhost:8081/ | jq
```

### Lesson 2: HTTP Fundamentals
- [ ] Use appropriate HTTP methods (GET, POST, PUT, PATCH, DELETE)
- [ ] Return correct status codes for different scenarios
- [ ] Implement content negotiation with headers
- [ ] Design clean URL structures

**Test Commands:**
```bash
# HTTP Methods
curl http://localhost:8083/books
curl -X POST http://localhost:8083/books -d '{"title":"Test","author":"Author"}' -H "Content-Type: application/json"

# Status Codes
curl http://localhost:8084/api/test/200
curl http://localhost:8084/api/test/404

# Headers
curl -H "Accept: application/json" http://localhost:8085/data
```

### Lesson 5: First REST API
- [ ] Build complete CRUD operations
- [ ] Implement proper error handling
- [ ] Add input validation
- [ ] Write comprehensive tests
- [ ] Handle JSON marshaling/unmarshaling

**Test Commands:**
```bash
# Health check
curl http://localhost:8087/health

# Create task
curl -X POST http://localhost:8087/api/tasks -H "Content-Type: application/json" -d '{"title":"Learn REST","description":"Complete course"}'

# Get all tasks
curl http://localhost:8087/api/tasks

# Filter tasks
curl http://localhost:8087/api/tasks?completed=false

# Run tests
go test -v
```

## 🧪 Testing Your Understanding

### Knowledge Check Questions

#### Lesson 1: REST Concepts
1. What are the 6 principles of REST?
2. What's the difference between Level 2 and Level 3 APIs?
3. How does HATEOAS improve API discoverability?
4. When would you choose REST over GraphQL?

#### Lesson 2: HTTP Fundamentals
1. Which HTTP methods are safe and idempotent?
2. When should you use 201 vs 200 status codes?
3. How do ETag headers help with caching?
4. What's the difference between PUT and PATCH?

#### Lesson 5: First REST API
1. How do you handle validation errors?
2. What makes an API idempotent?
3. How do you implement filtering with query parameters?
4. What's the difference between 400 and 422 status codes?

### Practical Exercises

#### Exercise 1: Extend the Task API
Add these features to the lesson-05 example:
- [ ] Add priority field (low, medium, high)
- [ ] Implement sorting by created_at
- [ ] Add search by title
- [ ] Implement task categories

#### Exercise 2: Error Handling
Test and fix these scenarios:
- [ ] Invalid JSON payload
- [ ] Missing required fields
- [ ] Invalid data types
- [ ] Business rule violations

#### Exercise 3: API Design
Design URLs for these resources:
- [ ] Blog posts with comments
- [ ] E-commerce products with reviews
- [ ] User profiles with orders
- [ ] Nested categories with products

## 📊 Progress Tracking

Track your validation progress:

- [ ] **Lesson 1 Complete**: All REST concept examples run successfully
- [ ] **Lesson 2 Complete**: All HTTP fundamental examples tested
- [ ] **Lesson 5 Complete**: Task API fully implemented and tested
- [ ] **Knowledge Check**: All questions answered correctly
- [ ] **Practical Exercises**: Additional features implemented
- [ ] **Code Review**: Code follows best practices

## 🐛 Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# Find and kill process using port
lsof -ti:8087 | xargs kill -9

# Or use different port
PORT=8088 go run main.go
```

#### Go Module Issues
```bash
# Clean and reinstall dependencies
go mod tidy
go clean -modcache
go mod download
```

#### Build Errors
```bash
# Check Go version (requires 1.21+)
go version

# Update dependencies
go get -u ./...
```

### Getting Help

1. **Check README files** in each lesson directory
2. **Review error messages** carefully
3. **Use Go's built-in help**: `go help <command>`
4. **Run tests** to identify issues: `go test -v`

## 🎓 Completion Certificate

Once you've completed all validations:

✅ **REST API Design and Development in Go - Validation Complete**

**Completed:**
- [x] REST Architectural Principles
- [x] HTTP Fundamentals
- [x] Complete CRUD API Implementation
- [x] Error Handling and Validation
- [x] Testing Strategies
- [x] Code Quality and Best Practices

**Skills Demonstrated:**
- Building production-ready REST APIs
- Proper HTTP method and status code usage
- Error handling and input validation
- JSON data processing
- Unit testing and test-driven development
- API design best practices

**Ready for Next Steps:**
- Advanced authentication and authorization
- Database integration
- API documentation with OpenAPI
- Performance optimization
- Deployment and monitoring

---

🎉 **Congratulations!** You've successfully validated your understanding of REST API development in Go. You're now ready to build production-ready APIs and tackle more advanced topics.