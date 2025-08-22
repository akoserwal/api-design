# Lesson 1: Introduction to REST APIs

## Learning Objectives
By the end of this lesson, you will be able to:
- Define what an API is and explain its purpose
- Understand REST architectural principles
- Differentiate between REST, SOAP, and GraphQL
- Recognize REST APIs in real-world applications
- Understand the Richardson Maturity Model

## What is an API?

An **Application Programming Interface (API)** is a set of rules and protocols that allows different software applications to communicate with each other. Think of it as a waiter in a restaurant:

- You (client) don't go into the kitchen
- You tell the waiter (API) what you want
- The waiter communicates with the kitchen (server)
- The waiter brings you your food (response)

### Types of APIs
- **Library APIs**: Functions within programming libraries
- **Operating System APIs**: System calls
- **Web APIs**: HTTP-based APIs for web services
- **Database APIs**: Interfaces to database systems

## What is REST?

**REST (Representational State Transfer)** is an architectural style for designing networked applications, particularly web services. It was introduced by Roy Fielding in his 2000 doctoral dissertation.

### REST Principles

#### 1. Client-Server Architecture
- Clear separation between client and server
- Server manages data and business logic
- Client handles user interface and user experience
- Both can evolve independently

#### 2. Statelessness
- Each request contains all information needed to process it
- Server doesn't store client context between requests
- Improves scalability and reliability

#### 3. Cacheability
- Responses should indicate if they can be cached
- Improves performance and reduces server load
- Cache-Control headers specify caching behavior

#### 4. Uniform Interface
- Consistent way to interact with resources
- Uses standard HTTP methods
- Self-descriptive messages
- Hypermedia as the engine of application state (HATEOAS)

#### 5. Layered System
- Architecture can have multiple layers (proxies, gateways, load balancers)
- Each layer doesn't know about layers beyond the next one
- Improves scalability and security

#### 6. Code on Demand (Optional)
- Server can send executable code to client
- Rarely used in practice
- Examples: JavaScript, applets

## Richardson Maturity Model

Leonard Richardson developed a model to measure REST API maturity:

### Level 0: The Swamp of POX (Plain Old XML)
- Single URL endpoint
- Usually POST for everything
- Not RESTful

```
POST /api
{
  "action": "getUser",
  "userId": 123
}
```

### Level 1: Resources
- Multiple URL endpoints
- Different URLs for different resources
- Still using HTTP as transport only

```
POST /api/users
POST /api/orders
```

### Level 2: HTTP Verbs
- Proper use of HTTP methods
- Appropriate HTTP status codes
- Most APIs stop here

```
GET /api/users/123
POST /api/users
PUT /api/users/123
DELETE /api/users/123
```

### Level 3: Hypermedia Controls (HATEOAS)
- Responses include links to related actions
- True REST implementation
- Self-documenting API

```json
{
  "id": 123,
  "name": "John Doe",
  "links": {
    "self": "/api/users/123",
    "orders": "/api/users/123/orders",
    "edit": "/api/users/123"
  }
}
```

## REST vs Other API Styles

### REST vs SOAP

| REST | SOAP |
|------|------|
| Architectural style | Protocol |
| Uses HTTP methods | Uses POST only |
| JSON/XML format | XML only |
| Stateless | Can be stateful |
| Lightweight | Heavy with specifications |
| Better performance | More secure by default |

### REST vs GraphQL

| REST | GraphQL |
|------|---------|
| Multiple endpoints | Single endpoint |
| Over-fetching common | Precise data fetching |
| Simple caching | Complex caching |
| Mature ecosystem | Newer, growing ecosystem |
| Good for CRUD operations | Good for complex data requirements |

## Real-World REST API Examples

### Twitter API
```
GET /api/tweets - Get tweets
POST /api/tweets - Create tweet
DELETE /api/tweets/123 - Delete tweet
```

### GitHub API
```
GET /repos/owner/repo - Get repository info
GET /repos/owner/repo/issues - Get issues
POST /repos/owner/repo/issues - Create issue
```

### Stripe Payment API
```
GET /charges - List charges
POST /charges - Create charge
GET /charges/ch_123 - Retrieve charge
```

## Benefits of REST APIs

### For Developers
- **Simplicity**: Easy to understand and implement
- **Flexibility**: Can return different data formats
- **Scalability**: Stateless nature supports horizontal scaling
- **Reusability**: Same API can serve multiple clients

### For Businesses
- **Platform Independence**: Works across different systems
- **Cost Effective**: Leverage existing HTTP infrastructure
- **Innovation**: Enables third-party integrations
- **Mobile Ready**: Perfect for mobile applications

## Common REST API Use Cases

1. **Mobile Applications**: Backend services for mobile apps
2. **Single Page Applications**: Data for React/Vue/Angular apps
3. **Microservices**: Communication between services
4. **Third-party Integrations**: Payment processors, social media
5. **IoT Devices**: Lightweight communication protocol
6. **Public APIs**: Twitter, GitHub, Google Maps

## Key Takeaways

- REST is an architectural style, not a standard
- REST APIs use HTTP methods meaningfully
- Statelessness is crucial for scalability
- Most APIs achieve Level 2 of Richardson Maturity Model
- REST is ideal for CRUD operations and resource manipulation
- Choose REST when you need simplicity and broad compatibility

## Next Steps

In the next lesson, we'll dive deep into HTTP fundamentals that form the foundation of REST APIs, including methods, status codes, and headers.

## Practice Questions

1. What are the six principles of REST?
2. Which level of Richardson Maturity Model uses proper HTTP verbs?
3. How does REST differ from SOAP in terms of data format?
4. Why is statelessness important in REST APIs?
5. Give an example of a Level 3 (HATEOAS) response.