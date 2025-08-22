# REST API Course - Code Examples

This directory contains practical code examples to validate your understanding of each lesson in the REST API Design and Development course.

## Directory Structure

```
examples/
├── lesson-01-rest-concepts/     # REST principles validation
├── lesson-02-http-fundamentals/ # HTTP methods and status codes
├── lesson-03-api-design/        # Design patterns and best practices
├── lesson-04-go-setup/          # Go environment and basic server
├── lesson-05-first-api/         # Complete CRUD API
├── lesson-09-auth/              # Authentication and authorization
├── lesson-12-testing/           # Testing strategies and examples
├── lesson-16-deployment/        # Deployment and monitoring
└── common/                      # Shared utilities and helpers
```

## Prerequisites

- Go 1.21 or later
- Docker (for containerization examples)
- Postman or curl (for API testing)

## How to Use These Examples

1. **Start with the basics**: Begin with lesson-01 and progress sequentially
2. **Run the code**: Each example includes instructions to run and test
3. **Modify and experiment**: Try changing the code to see different behaviors
4. **Complete the exercises**: Each example includes validation exercises
5. **Build upon**: Use earlier examples as foundation for later ones

## Running Examples

Each lesson directory contains:
- **README.md**: Specific instructions for that lesson
- **main.go**: Main application code
- **go.mod**: Go module definition
- **test files**: Unit and integration tests
- **docker files**: Containerization examples (where applicable)

### Quick Start

```bash
# Navigate to any lesson
cd lesson-05-first-api

# Install dependencies
go mod download

# Run the application
go run main.go

# Run tests
go test ./...
```

## Validation Exercises

Each example includes exercises to validate your understanding:

1. **Code Review**: Understand what each part does
2. **Modifications**: Make specific changes to see effects
3. **Testing**: Write additional tests
4. **Extension**: Add new features using learned concepts

## Learning Path

1. **Lesson 1**: Understand REST principles through simple examples
2. **Lesson 2**: Practice HTTP methods and status codes
3. **Lesson 3**: Apply design patterns to real scenarios
4. **Lesson 4**: Set up Go development environment
5. **Lesson 5**: Build complete CRUD API
6. **Lesson 9**: Implement authentication and authorization
7. **Lesson 12**: Write comprehensive tests
8. **Lesson 16**: Deploy and monitor your API

## Common Commands

```bash
# Install all dependencies
find . -name "go.mod" -execdir go mod download \;

# Run all tests
find . -name "*_test.go" -execdir go test ./... \;

# Format all code
find . -name "*.go" -exec go fmt {} \;

# Build all examples
find . -name "main.go" -execdir go build \;
```

## Troubleshooting

- **Port conflicts**: Change port numbers in examples if 8080 is in use
- **Module issues**: Ensure you're in the correct directory with go.mod
- **Dependencies**: Run `go mod tidy` to clean up dependencies
- **Docker issues**: Ensure Docker is running for containerization examples

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [REST API Best Practices](https://restfulapi.net/)
- [HTTP Status Codes](https://httpstatuses.com/)
- [JSON API Specification](https://jsonapi.org/)

Happy learning! 🚀