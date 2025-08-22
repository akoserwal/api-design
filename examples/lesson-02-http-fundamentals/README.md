# Lesson 2: HTTP Fundamentals Validation Examples

This directory contains examples to validate your understanding of HTTP methods, status codes, headers, and URL structure for REST APIs.

## Examples Included

1. **http-methods.go** - Comprehensive HTTP methods demonstration
2. **status-codes.go** - HTTP status codes usage examples
3. **headers-demo.go** - HTTP headers and content negotiation
4. **url-structure.go** - URL design patterns and query parameters

## Learning Objectives Validation

- ✅ Use appropriate HTTP methods for different operations
- ✅ Return correct HTTP status codes for various scenarios
- ✅ Handle HTTP headers for content negotiation and caching
- ✅ Design clean and meaningful URL structures
- ✅ Implement query parameters for filtering and pagination

## Running the Examples

```bash
# Run HTTP methods demonstration
go run http-methods.go

# Run status codes examples
go run status-codes.go

# Run headers demonstration
go run headers-demo.go

# Run URL structure examples
go run url-structure.go
```

## Validation Exercises

### Exercise 1: HTTP Methods Usage
1. Test each HTTP method with appropriate data
2. Verify that GET requests don't modify data
3. Check that PUT is idempotent
4. Test PATCH for partial updates

```bash
# Test GET (safe, idempotent)
curl http://localhost:8083/books

# Test POST (not safe, not idempotent)
curl -X POST http://localhost:8083/books -d '{"title":"New Book","author":"Author"}' -H "Content-Type: application/json"

# Test PUT (not safe, idempotent)
curl -X PUT http://localhost:8083/books/1 -d '{"title":"Updated Book","author":"Updated Author"}' -H "Content-Type: application/json"

# Test PATCH (not safe, not always idempotent)
curl -X PATCH http://localhost:8083/books/1 -d '{"title":"Patched Title"}' -H "Content-Type: application/json"

# Test DELETE (not safe, idempotent)
curl -X DELETE http://localhost:8083/books/1
```

### Exercise 2: Status Codes
1. Trigger different status codes by sending various requests
2. Observe how error responses are structured
3. Test edge cases like invalid IDs and malformed JSON

```bash
# Test 200 OK
curl http://localhost:8084/api/test/200

# Test 201 Created
curl -X POST http://localhost:8084/api/test/201

# Test 400 Bad Request
curl http://localhost:8084/api/test/400

# Test 404 Not Found
curl http://localhost:8084/api/test/404

# Test 500 Internal Server Error
curl http://localhost:8084/api/test/500
```

### Exercise 3: Headers and Content Negotiation
1. Test different Accept headers to see content negotiation
2. Verify caching headers are set correctly
3. Test conditional requests with If-None-Match

```bash
# Test JSON response
curl -H "Accept: application/json" http://localhost:8085/data

# Test XML response
curl -H "Accept: application/xml" http://localhost:8085/data

# Test caching headers
curl -I http://localhost:8085/cached-data

# Test conditional request
curl -H "If-None-Match: \"data-123\"" http://localhost:8085/cached-data
```

### Exercise 4: URL Structure and Query Parameters
1. Test different URL patterns and hierarchies
2. Use query parameters for filtering, sorting, and pagination
3. Verify URL design follows REST conventions

```bash
# Test resource collections
curl http://localhost:8086/api/articles

# Test nested resources
curl http://localhost:8086/api/authors/1/articles

# Test filtering
curl "http://localhost:8086/api/articles?category=tech&status=published"

# Test sorting
curl "http://localhost:8086/api/articles?sort=-published_date,title"

# Test pagination
curl "http://localhost:8086/api/articles?page=2&limit=5"
```

## Key Concepts to Validate

### HTTP Methods
- **GET**: Retrieve data (safe, idempotent)
- **POST**: Create new resources (not safe, not idempotent)
- **PUT**: Replace entire resource (not safe, idempotent)
- **PATCH**: Partial update (not safe, may be idempotent)
- **DELETE**: Remove resource (not safe, idempotent)

### Status Codes
- **2xx**: Success responses
- **3xx**: Redirection messages
- **4xx**: Client error responses
- **5xx**: Server error responses

### Headers
- **Content-Type**: Specifies media type of response/request
- **Accept**: Client's preferred media types
- **Cache-Control**: Caching directives
- **ETag**: Resource version identifier

### URL Design
- Use nouns for resources
- Use plural forms for collections
- Implement proper nesting for relationships
- Use query parameters for optional features

## Testing Your Understanding

After running the examples, answer these questions:

1. Which HTTP methods are safe and idempotent?
2. When should you use 201 vs 200 status codes?
3. How do ETag headers help with caching?
4. What's the difference between /users/123/orders and /orders?filter=user:123?
5. How do you implement proper content negotiation?

## Expected Behaviors

The examples demonstrate:
- Proper HTTP method semantics
- Appropriate status code usage
- Content negotiation based on Accept headers
- Clean URL structure and meaningful hierarchies
- Query parameter handling for filtering and pagination