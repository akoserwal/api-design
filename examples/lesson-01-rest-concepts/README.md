# Lesson 1: REST Concepts Validation Examples

This directory contains examples to validate your understanding of REST architectural principles and concepts.

## Examples Included

1. **rest-levels.go** - Richardson Maturity Model demonstration
2. **api-comparison.go** - REST vs SOAP vs GraphQL comparison
3. **hateoas-example.go** - HATEOAS implementation example
4. **rest-principles.go** - Core REST principles demonstration

## Learning Objectives Validation

- ✅ Understand REST architectural style
- ✅ Identify different maturity levels (Richardson Model)
- ✅ Compare REST with other API styles
- ✅ Implement HATEOAS (Hypermedia controls)
- ✅ Apply REST principles in practice

## Running the Examples

```bash
# Run Richardson Maturity Model demo
go run rest-levels.go

# Run API comparison examples
go run api-comparison.go

# Run HATEOAS implementation
go run hateoas-example.go

# Run REST principles demo
go run rest-principles.go
```

## Validation Exercises

### Exercise 1: Richardson Maturity Model
1. Run `rest-levels.go` and test each maturity level
2. Identify which level each endpoint represents
3. Modify Level 0 to become Level 1
4. Add proper HTTP methods to achieve Level 2

### Exercise 2: HATEOAS Implementation
1. Examine the HATEOAS response structure
2. Add new link relationships
3. Implement state-dependent links
4. Test navigation through hypermedia

### Exercise 3: REST Principles
1. Identify violations of REST principles in the examples
2. Fix any non-RESTful patterns you find
3. Add caching headers to appropriate responses
4. Implement stateless request handling

## Testing Your Understanding

Answer these questions after running the examples:

1. What makes an API RESTful?
2. What's the difference between Level 2 and Level 3 APIs?
3. How does HATEOAS improve API discoverability?
4. Why is statelessness important in REST?
5. When might you choose REST over GraphQL?

## Expected Outputs

The examples will demonstrate:
- Different API maturity levels in action
- Proper use of HTTP methods and status codes
- Hypermedia-driven state transitions
- Stateless request/response patterns