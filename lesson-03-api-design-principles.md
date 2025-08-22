# Lesson 3: API Design Principles and Best Practices

## Learning Objectives
By the end of this lesson, you will be able to:
- Apply resource-oriented design principles
- Follow consistent naming conventions
- Implement proper error handling strategies
- Design APIs with discoverability in mind
- Create maintainable and extensible API architectures
- Understand API design patterns and anti-patterns

## Resource-Oriented Design

REST APIs should be designed around resources, not actions.

### What is a Resource?
A resource is any information that can be named and addressed. Resources are the key abstraction in REST.

**Examples:**
- Users
- Products
- Orders
- Blog posts
- Files

### Resource Identification
Each resource should have a unique identifier (URI).

```
Good:
/api/users/123
/api/products/abc-123
/api/orders/order-456

Bad:
/api/getUser?id=123
/api/getUserById/123
```

### Resource Relationships

#### One-to-Many
```
/api/users/123/orders          # Orders belonging to user 123
/api/categories/5/products     # Products in category 5
/api/posts/789/comments        # Comments on post 789
```

#### Many-to-Many
```
/api/users/123/roles           # Roles assigned to user 123
/api/products/456/tags         # Tags associated with product 456
```

#### Nested vs Flat Structure
```
# Nested (when relationship is important)
/api/users/123/orders/456

# Flat (when resource can stand alone)
/api/orders/456
```

## Naming Conventions

### URL Naming Rules

#### 1. Use Nouns, Not Verbs
```
✅ GET /api/users
✅ POST /api/users
✅ GET /api/products/123

❌ GET /api/getUsers
❌ POST /api/createUser
❌ GET /api/fetchProduct/123
```

#### 2. Use Plural Nouns
```
✅ /api/users
✅ /api/products
✅ /api/orders

❌ /api/user
❌ /api/product
❌ /api/order
```

#### 3. Use Lowercase and Hyphens
```
✅ /api/user-profiles
✅ /api/order-items
✅ /api/shipping-addresses

❌ /api/userProfiles
❌ /api/orderItems
❌ /api/shipping_addresses
```

#### 4. Avoid Deep Nesting
```
✅ /api/users/123/orders
✅ /api/orders/456/items

❌ /api/users/123/orders/456/items/789/reviews
```

### Field Naming in JSON

#### Use camelCase or snake_case Consistently
```json
// camelCase (common in JavaScript)
{
  "firstName": "John",
  "lastName": "Doe",
  "dateOfBirth": "1990-05-15"
}

// snake_case (common in Python)
{
  "first_name": "John",
  "last_name": "Doe",
  "date_of_birth": "1990-05-15"
}
```

#### Use Meaningful Names
```json
✅ Good:
{
  "userId": 123,
  "emailAddress": "john@example.com",
  "isActive": true
}

❌ Bad:
{
  "uid": 123,
  "email": "john@example.com",
  "active": 1
}
```

## API Design Patterns

### 1. Collection and Resource Pattern
```
Collection: /api/users
Resource:   /api/users/123
```

### 2. Sub-resource Pattern
```
/api/users/123/orders
/api/orders/456/items
```

### 3. Filter Pattern
```
/api/products?category=electronics
/api/users?status=active&role=admin
```

### 4. Search Pattern
```
/api/search/users?q=john
/api/products/search?q=laptop&category=electronics
```

### 5. Bulk Operations Pattern
```
POST /api/users/bulk
{
  "operations": [
    {"action": "create", "data": {...}},
    {"action": "update", "id": 123, "data": {...}},
    {"action": "delete", "id": 456}
  ]
}
```

## Pagination Strategies

### Offset-based Pagination
```
GET /api/products?offset=20&limit=10

Response:
{
  "data": [...],
  "meta": {
    "offset": 20,
    "limit": 10,
    "total": 150
  }
}
```

### Page-based Pagination
```
GET /api/products?page=3&per_page=10

Response:
{
  "data": [...],
  "meta": {
    "page": 3,
    "per_page": 10,
    "total_pages": 15,
    "total_count": 150
  }
}
```

### Cursor-based Pagination
```
GET /api/products?cursor=eyJpZCI6MTIz&limit=10

Response:
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJpZCI6MTMz",
    "has_more": true
  }
}
```

## Filtering, Sorting, and Searching

### Filtering
```
# Simple filters
/api/products?category=electronics&price_min=100

# Complex filters
/api/products?filter[category]=electronics&filter[price][gte]=100

# Multiple values
/api/products?status=active,published
```

### Sorting
```
# Single field
/api/products?sort=price

# Multiple fields
/api/products?sort=category,price

# Direction specification
/api/products?sort=-price,+name  # price desc, name asc
```

### Field Selection
```
# Include specific fields
/api/users?fields=id,name,email

# Exclude fields
/api/users?exclude=password,salt

# Nested field selection
/api/users?fields=id,name,profile.avatar
```

## Error Handling Design

### Consistent Error Structure
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request data is invalid",
    "details": [
      {
        "field": "email",
        "code": "INVALID_FORMAT",
        "message": "Email address is not valid"
      }
    ],
    "request_id": "req_12345",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Error Code Patterns
```
# Hierarchical error codes
USER_ERROR
├── USER_NOT_FOUND
├── USER_ALREADY_EXISTS
└── USER_VALIDATION_ERROR
    ├── USER_EMAIL_INVALID
    └── USER_PASSWORD_WEAK

# Simple error codes
INVALID_REQUEST
UNAUTHORIZED
FORBIDDEN
NOT_FOUND
VALIDATION_ERROR
INTERNAL_ERROR
```

## Versioning Strategy Design

### URL Versioning
```
/api/v1/users
/api/v2/users
```

### Header Versioning
```
GET /api/users
Accept: application/vnd.api+json;version=1
```

### Query Parameter Versioning
```
/api/users?version=1
```

## HATEOAS Implementation

### Basic Hypermedia Links
```json
{
  "id": 123,
  "name": "John Doe",
  "email": "john@example.com",
  "_links": {
    "self": {
      "href": "/api/users/123"
    },
    "orders": {
      "href": "/api/users/123/orders"
    },
    "edit": {
      "href": "/api/users/123",
      "method": "PUT"
    },
    "delete": {
      "href": "/api/users/123",
      "method": "DELETE"
    }
  }
}
```

### HAL (Hypertext Application Language)
```json
{
  "id": 123,
  "name": "John Doe",
  "_links": {
    "self": {"href": "/api/users/123"},
    "orders": {"href": "/api/users/123/orders"}
  },
  "_embedded": {
    "orders": [
      {
        "id": 456,
        "total": 99.99,
        "_links": {
          "self": {"href": "/api/orders/456"}
        }
      }
    ]
  }
}
```

## Content Type Strategy

### Standard Content Types
```
application/json          # Most common
application/xml           # XML format
text/csv                 # CSV exports
application/pdf          # PDF documents
multipart/form-data      # File uploads
```

### Custom Content Types
```
application/vnd.api+json              # JSON API
application/vnd.company.user+json     # Custom user format
application/vnd.company.v2+json       # Versioned format
```

## Rate Limiting Design

### Rate Limit Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
X-RateLimit-Window: 3600
```

### Rate Limiting Strategies
- **Fixed Window**: Reset at fixed intervals
- **Sliding Window**: Continuous time window
- **Token Bucket**: Allow bursts up to limit
- **Leaky Bucket**: Smooth out request rate

## Security Considerations

### Input Validation
- Validate all input data
- Use allow-lists over deny-lists
- Sanitize input to prevent injection attacks
- Validate file uploads

### Authentication Design
```http
# Bearer Token
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# API Key
X-API-Key: your-api-key-here

# Basic Auth (for internal APIs only)
Authorization: Basic dXNlcjpwYXNz
```

### HTTPS Everywhere
- Always use HTTPS in production
- Redirect HTTP to HTTPS
- Use HSTS headers

## Common Design Anti-Patterns

### 1. Chatty APIs
```
❌ Multiple round trips required:
GET /api/user/123
GET /api/user/123/profile
GET /api/user/123/preferences

✅ Single optimized endpoint:
GET /api/user/123?include=profile,preferences
```

### 2. Overly Generic Endpoints
```
❌ Too generic:
POST /api/execute
{
  "action": "createUser",
  "params": {...}
}

✅ Specific endpoints:
POST /api/users
```

### 3. Ignoring HTTP Methods
```
❌ Using POST for everything:
POST /api/getUser
POST /api/updateUser
POST /api/deleteUser

✅ Proper HTTP methods:
GET /api/users/123
PUT /api/users/123
DELETE /api/users/123
```

### 4. Poor Error Messages
```
❌ Unhelpful:
{
  "error": "Bad request"
}

✅ Descriptive:
{
  "error": {
    "message": "Email address is required",
    "field": "email",
    "code": "FIELD_REQUIRED"
  }
}
```

## API Evolution Best Practices

### Backward Compatibility
- Add new fields without breaking existing clients
- Use optional parameters for new features
- Deprecate gracefully with advance notice
- Support multiple versions during transition

### Extensibility
- Design for future growth
- Use consistent patterns
- Plan for new resource types
- Consider plugin architectures

## Documentation-Driven Development

### API-First Approach
1. Define API specification first
2. Review with stakeholders
3. Generate mock servers
4. Implement against specification
5. Test against specification

### OpenAPI Benefits
- Machine-readable documentation
- Code generation capabilities
- Interactive documentation
- Validation tools

## Performance Considerations

### Response Size Optimization
- Use field selection
- Implement compression
- Minimize nested data
- Use pagination effectively

### Caching Strategy
- Use appropriate cache headers
- Implement ETags for validation
- Consider edge caching
- Cache frequently accessed data

## Key Design Principles Summary

1. **Consistency**: Use consistent patterns throughout your API
2. **Simplicity**: Make common use cases easy
3. **Flexibility**: Allow for various client needs
4. **Discoverability**: Make your API self-documenting
5. **Reliability**: Handle errors gracefully
6. **Security**: Secure by default
7. **Performance**: Optimize for common use cases
8. **Evolvability**: Plan for future changes

## Next Steps

In the next lesson, we'll set up a Go development environment and start building our first REST API, applying these design principles in practice.

## Practice Exercise

Design a REST API for a simple e-commerce system with the following resources:
- Products
- Categories
- Users
- Orders
- Reviews

Include:
1. URL structure for all operations
2. Request/response examples
3. Error handling strategy
4. Pagination approach
5. Authentication method