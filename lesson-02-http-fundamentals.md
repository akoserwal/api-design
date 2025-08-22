# Lesson 2: HTTP Fundamentals for APIs

## Learning Objectives
By the end of this lesson, you will be able to:
- Understand HTTP request/response cycle
- Use appropriate HTTP methods for different operations
- Interpret and use HTTP status codes effectively
- Work with HTTP headers for API communication
- Structure URLs and handle query parameters
- Understand the differences between HTTP/1.1 and HTTP/2

## HTTP Overview

**HTTP (HyperText Transfer Protocol)** is the foundation of data communication on the web. For REST APIs, HTTP provides the communication protocol that defines how messages are formatted and transmitted.

### HTTP Request/Response Cycle

1. **Client** sends HTTP request to server
2. **Server** processes the request
3. **Server** sends HTTP response back to client
4. **Connection** may be closed or kept alive

## HTTP Methods (Verbs)

HTTP methods indicate the desired action for a resource.

### GET
- **Purpose**: Retrieve data from server
- **Safe**: Yes (doesn't modify server state)
- **Idempotent**: Yes (multiple identical requests have same effect)
- **Request Body**: Not recommended
- **Cacheable**: Yes

```http
GET /api/users/123 HTTP/1.1
Host: api.example.com
Accept: application/json
```

**Use Cases:**
- Fetch user profile
- Get list of products
- Retrieve search results

### POST
- **Purpose**: Create new resources or submit data
- **Safe**: No
- **Idempotent**: No
- **Request Body**: Yes
- **Cacheable**: Generally no

```http
POST /api/users HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com"
}
```

**Use Cases:**
- Create new user account
- Submit form data
- Upload files

### PUT
- **Purpose**: Create or completely replace a resource
- **Safe**: No
- **Idempotent**: Yes
- **Request Body**: Yes
- **Cacheable**: No

```http
PUT /api/users/123 HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "name": "John Smith",
  "email": "johnsmith@example.com"
}
```

**Use Cases:**
- Update entire user profile
- Replace document content
- Set configuration

### PATCH
- **Purpose**: Partially modify a resource
- **Safe**: No
- **Idempotent**: Depends on implementation
- **Request Body**: Yes
- **Cacheable**: No

```http
PATCH /api/users/123 HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "email": "newemail@example.com"
}
```

**Use Cases:**
- Update user email only
- Change password
- Modify specific fields

### DELETE
- **Purpose**: Remove a resource
- **Safe**: No
- **Idempotent**: Yes
- **Request Body**: Optional
- **Cacheable**: No

```http
DELETE /api/users/123 HTTP/1.1
Host: api.example.com
```

**Use Cases:**
- Delete user account
- Remove product from catalog
- Clear cache

### Other Methods

#### HEAD
- Like GET but returns only headers
- Used to check if resource exists
- Useful for caching validation

#### OPTIONS
- Returns allowed methods for a resource
- Used in CORS preflight requests

## HTTP Status Codes

Status codes indicate the outcome of an HTTP request.

### 1xx: Informational
- **100 Continue**: Server received initial part of request
- **101 Switching Protocols**: Server switching protocols

*Rarely used in REST APIs*

### 2xx: Success

#### 200 OK
- Request succeeded
- Most common success response
```http
HTTP/1.1 200 OK
Content-Type: application/json

{"id": 123, "name": "John Doe"}
```

#### 201 Created
- Resource successfully created
- Should include Location header
```http
HTTP/1.1 201 Created
Location: /api/users/124
Content-Type: application/json

{"id": 124, "name": "Jane Doe"}
```

#### 202 Accepted
- Request accepted for processing
- Processing not completed
- Used for asynchronous operations

#### 204 No Content
- Request succeeded
- No content to return
- Common for DELETE operations

### 3xx: Redirection

#### 301 Moved Permanently
- Resource permanently moved
- Update bookmarks/links

#### 302 Found
- Resource temporarily moved
- Keep using original URL

#### 304 Not Modified
- Resource not modified since last request
- Used with caching

### 4xx: Client Errors

#### 400 Bad Request
- Invalid request syntax
- Malformed JSON
- Missing required fields

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Invalid email format",
  "field": "email"
}
```

#### 401 Unauthorized
- Authentication required
- Invalid credentials

#### 403 Forbidden
- Server understood request
- Refused to authorize

#### 404 Not Found
- Resource doesn't exist
- Most common client error

#### 409 Conflict
- Request conflicts with current state
- Duplicate resource creation

#### 422 Unprocessable Entity
- Syntactically correct but semantically invalid
- Validation errors

### 5xx: Server Errors

#### 500 Internal Server Error
- Generic server error
- Something went wrong on server

#### 502 Bad Gateway
- Invalid response from upstream server

#### 503 Service Unavailable
- Server temporarily unavailable
- Maintenance or overload

## HTTP Headers

Headers provide metadata about the request or response.

### Request Headers

#### Accept
Specifies content types client can handle
```http
Accept: application/json
Accept: application/json, text/xml;q=0.9
```

#### Content-Type
Specifies format of request body
```http
Content-Type: application/json
Content-Type: application/x-www-form-urlencoded
```

#### Authorization
Contains authentication credentials
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Authorization: Basic dXNlcjpwYXNz
```

#### User-Agent
Identifies client application
```http
User-Agent: MyApp/1.0 (iOS; iPhone)
```

### Response Headers

#### Content-Type
Specifies format of response body
```http
Content-Type: application/json; charset=utf-8
```

#### Location
URL of newly created resource
```http
Location: /api/users/124
```

#### Cache-Control
Caching directives
```http
Cache-Control: no-cache
Cache-Control: max-age=3600
```

#### X-RateLimit-*
API rate limiting information
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## URL Structure

### Components
```
https://api.example.com:443/v1/users/123?include=profile&sort=name#section
│─────┘ │──────────────┘ │─┘ │─┘ │─────┘ │─┘ │─────────────────┘ │──────┘
scheme    hostname      port │   │       │   │         │          fragment
                           path │       │   │      query params
                            version    resource id
```

### Best Practices

#### Use Nouns for Resources
```
✅ /api/users
✅ /api/products
❌ /api/getUsers
❌ /api/createProduct
```

#### Use Plural Nouns
```
✅ /api/users
✅ /api/products
❌ /api/user
❌ /api/product
```

#### Hierarchical Relationships
```
✅ /api/users/123/orders
✅ /api/categories/5/products
❌ /api/user-orders?userId=123
```

### Query Parameters

#### Filtering
```
GET /api/products?category=electronics&price_min=100&price_max=500
```

#### Sorting
```
GET /api/users?sort=name
GET /api/users?sort=name,email
GET /api/users?sort=-created_at  // descending
```

#### Pagination
```
GET /api/products?page=2&limit=20
GET /api/products?offset=40&limit=20
```

#### Field Selection
```
GET /api/users?fields=name,email
GET /api/users/123?include=profile,orders
```

## Content Negotiation

### Accept Header
Client specifies preferred content type:
```http
Accept: application/json
Accept: application/xml
Accept: application/json, application/xml;q=0.8
```

### Content-Type Header
Server specifies actual content type:
```http
Content-Type: application/json; charset=utf-8
```

### Quality Values (q)
Indicates preference level:
```http
Accept: application/json;q=1.0, application/xml;q=0.8, text/plain;q=0.5
```

## HTTP/1.1 vs HTTP/2

### HTTP/1.1 Characteristics
- **Text Protocol**: Human-readable format
- **Persistent Connections**: Keep-alive connections
- **Pipelining**: Send multiple requests without waiting
- **Head-of-line Blocking**: One slow request blocks others

### HTTP/2 Improvements
- **Binary Protocol**: More efficient parsing
- **Multiplexing**: Multiple requests over single connection
- **Server Push**: Server can initiate data transfer
- **Header Compression**: Reduces overhead
- **Stream Prioritization**: Important requests first

### Impact on APIs
- **HTTP/1.1**: Still widely used, well-supported
- **HTTP/2**: Better performance, especially for multiple requests
- **Backward Compatibility**: HTTP/2 APIs work with HTTP/1.1 clients

## Best Practices

### Method Selection
- **GET**: Read operations only
- **POST**: Create new resources
- **PUT**: Replace entire resource
- **PATCH**: Partial updates
- **DELETE**: Remove resources

### Status Code Usage
- Use appropriate codes for different scenarios
- Be consistent across your API
- Include meaningful error messages

### Header Management
- Always set Content-Type
- Use appropriate caching headers
- Include CORS headers when needed
- Implement rate limiting headers

## Common Pitfalls

1. **Using GET for State Changes**: Never modify data with GET
2. **Wrong Status Codes**: Using 200 for all responses
3. **Missing Content-Type**: Client can't parse response
4. **Inconsistent URL Structure**: Makes API hard to learn
5. **Ignoring Caching**: Poor performance without proper caching

## Example: Complete HTTP Transaction

```http
# Request
POST /api/users HTTP/1.1
Host: api.example.com
Content-Type: application/json
Accept: application/json
Authorization: Bearer token123

{
  "name": "John Doe",
  "email": "john@example.com"
}

# Response
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Location: /api/users/124
Cache-Control: no-cache

{
  "id": 124,
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

## Key Takeaways

- HTTP methods have specific semantics - use them correctly
- Status codes communicate the outcome of operations
- Headers provide essential metadata for proper communication
- URL structure should be intuitive and consistent
- Query parameters enable filtering, sorting, and pagination
- HTTP/2 offers performance improvements over HTTP/1.1

## Next Steps

In the next lesson, we'll explore API design principles and best practices that will help you create well-structured, maintainable REST APIs.

## Practice Exercises

1. Design URLs for a blog API (posts, comments, authors)
2. Choose appropriate HTTP methods for each operation
3. Write example requests with proper headers
4. Define suitable status codes for different scenarios
5. Create query parameter structure for search functionality