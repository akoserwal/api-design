# Lesson 6A: Google API Design Standards

## Learning Objectives
By the end of this lesson, you will be able to:
- Apply Google's API design standards to REST APIs
- Implement Google-compliant resource naming conventions
- Use standard method patterns and field requirements
- Design APIs that integrate seamlessly with Google Cloud services
- Build enterprise-grade APIs following industry standards

## Why Google API Standards Matter

Google's API Improvement Proposals (AIPs) represent industry best practices learned from building APIs at massive scale. These standards are used by:
- Google Cloud Platform (200+ services)
- Major enterprise companies
- Open source projects
- API-first organizations

Following these standards ensures your APIs are:
- **Consistent** with industry expectations
- **Scalable** for enterprise use
- **Interoperable** with existing systems
- **Future-proof** and maintainable

## Google's Core API Design Principles

### 1. Resource-Oriented Design
Every API should be designed around resources, not actions.

```
✅ Good: /publishers/123/books/456
❌ Bad:  /getBooksByPublisher?publisherId=123
```

### 2. Standard Methods
Use standardized method names and patterns:
- **List**: Get multiple resources
- **Get**: Retrieve a single resource  
- **Create**: Generate new resources
- **Update**: Modify existing resources
- **Delete**: Remove resources

### 3. Consistent Naming
Follow strict naming conventions across all APIs.

## Resource Naming Standards

### Collection Identifiers
Collections must be:
- **Plural**: `books` not `book`
- **camelCase**: `userPreferences` not `user-preferences`
- **Concise**: `books` not `bookItems`
- **American English**: `colors` not `colours`

```go
// Google-compliant collection names
✅ /users/123/preferences
✅ /publishers/456/books  
✅ /projects/789/datasets

❌ /user/123/preference
❌ /publisher/456/book-list
❌ /project/789/data-sets
```

### Resource Names
Resources must have globally unique names following this pattern:

```
collection1/id1/collection2/id2/...
```

**Example Resource Names:**
```
publishers/o-reilly/books/learning-go
users/john-doe/preferences/notifications
projects/my-project/datasets/sales-data/tables/customers
```

### Resource Name Field Requirements

#### Primary Name Field
Every resource must have a `name` field as the first field:

```go
type Book struct {
    Name        string    `json:"name"`         // Required: unique resource name
    DisplayName string    `json:"display_name"` // Human-readable name
    Author      string    `json:"author"`
    ISBN        string    `json:"isbn"`
    CreatedAt   time.Time `json:"create_time"`  // Google uses create_time
    UpdatedAt   time.Time `json:"update_time"`  // Google uses update_time
}
```

#### Parent Field for Nested Resources
Nested resources must include a `parent` field:

```go
type CreateBookRequest struct {
    Parent string `json:"parent"` // Required: "publishers/o-reilly"
    BookId string `json:"book_id"` // Optional: user-specified ID
    Book   Book   `json:"book"`    // The resource to create
}
```

## Standard Methods Implementation

### 1. List Method
**Pattern**: `List{Resource}s`
**HTTP**: `GET /collection`

```go
type ListBooksRequest struct {
    Parent    string `json:"parent"`     // Required: publishers/123
    PageSize  int32  `json:"page_size"`  // Default: 50, Max: 1000
    PageToken string `json:"page_token"` // For pagination
    Filter    string `json:"filter"`     // Optional filtering
    OrderBy   string `json:"order_by"`   // Optional ordering
}

type ListBooksResponse struct {
    Books         []Book `json:"books"`
    NextPageToken string `json:"next_page_token"`
    TotalSize     int32  `json:"total_size,omitempty"`
}

func (h *BookHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
    req := ListBooksRequest{
        Parent:    r.URL.Query().Get("parent"),
        PageSize:  getIntParam(r, "page_size", 50),
        PageToken: r.URL.Query().Get("page_token"),
        Filter:    r.URL.Query().Get("filter"),
        OrderBy:   r.URL.Query().Get("order_by"),
    }
    
    // Validate parent
    if req.Parent == "" {
        respondWithError(w, http.StatusBadRequest, "parent field is required")
        return
    }
    
    // Validate page size
    if req.PageSize > 1000 {
        req.PageSize = 1000
    }
    
    // Implementation here...
}
```

### 2. Get Method
**Pattern**: `Get{Resource}`
**HTTP**: `GET /collection/id`

```go
type GetBookRequest struct {
    Name string `json:"name"` // Required: publishers/123/books/456
}

func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
    name := extractResourceName(r) // Extract from URL path
    
    book, err := h.storage.GetByName(name)
    if err != nil {
        if isNotFound(err) {
            respondWithError(w, http.StatusNotFound, "Book not found")
            return
        }
        respondWithError(w, http.StatusInternalServerError, "Internal error")
        return
    }
    
    respondWithJSON(w, http.StatusOK, book)
}
```

### 3. Create Method
**Pattern**: `Create{Resource}`
**HTTP**: `POST /collection`

```go
type CreateBookRequest struct {
    Parent string `json:"parent"`  // Required: publishers/123
    BookId string `json:"book_id"` // Optional: user-specified ID
    Book   Book   `json:"book"`    // The resource to create
}

func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
    var req CreateBookRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    
    // Validate parent
    if req.Parent == "" {
        respondWithError(w, http.StatusBadRequest, "parent field is required")
        return
    }
    
    // Generate resource name
    bookId := req.BookId
    if bookId == "" {
        bookId = generateId() // System-generated ID
    }
    
    req.Book.Name = fmt.Sprintf("%s/books/%s", req.Parent, bookId)
    req.Book.CreateTime = time.Now()
    req.Book.UpdateTime = time.Now()
    
    // Create the resource
    createdBook, err := h.storage.Create(req.Book)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create book")
        return
    }
    
    w.Header().Set("Location", fmt.Sprintf("/v1/%s", createdBook.Name))
    respondWithJSON(w, http.StatusCreated, createdBook)
}
```

### 4. Update Method
**Pattern**: `Update{Resource}`
**HTTP**: `PATCH /collection/id`

```go
type UpdateBookRequest struct {
    Book       Book                     `json:"book"`        // Required
    UpdateMask *fieldmaskpb.FieldMask   `json:"update_mask"` // Optional
}

func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
    var req UpdateBookRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    
    // Extract resource name from URL
    req.Book.Name = extractResourceName(r)
    
    // Apply field mask if provided
    if req.UpdateMask != nil {
        // Only update specified fields
        updatedBook, err := h.storage.UpdateWithMask(req.Book, req.UpdateMask)
        if err != nil {
            respondWithError(w, http.StatusInternalServerError, "Update failed")
            return
        }
        respondWithJSON(w, http.StatusOK, updatedBook)
        return
    }
    
    // Update all fields
    req.Book.UpdateTime = time.Now()
    updatedBook, err := h.storage.Update(req.Book)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Update failed")
        return
    }
    
    respondWithJSON(w, http.StatusOK, updatedBook)
}
```

### 5. Delete Method
**Pattern**: `Delete{Resource}`
**HTTP**: `DELETE /collection/id`

```go
type DeleteBookRequest struct {
    Name         string `json:"name"`          // Required
    Force        bool   `json:"force"`         // Optional: force delete
    AllowMissing bool   `json:"allow_missing"` // Optional: succeed if not found
}

func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
    req := DeleteBookRequest{
        Name:         extractResourceName(r),
        Force:        getBoolParam(r, "force", false),
        AllowMissing: getBoolParam(r, "allow_missing", false),
    }
    
    err := h.storage.Delete(req.Name, req.Force)
    if err != nil {
        if isNotFound(err) && !req.AllowMissing {
            respondWithError(w, http.StatusNotFound, "Book not found")
            return
        }
        if isFailedPrecondition(err) && !req.Force {
            respondWithError(w, http.StatusFailedPrecondition, 
                "Book has dependencies. Use force=true to delete.")
            return
        }
        respondWithError(w, http.StatusInternalServerError, "Delete failed")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

## Google-Style Error Handling

### Error Response Structure

```go
type GoogleErrorResponse struct {
    Error GoogleErrorDetail `json:"error"`
}

type GoogleErrorDetail struct {
    Code    int32              `json:"code"`
    Message string             `json:"message"`
    Status  string             `json:"status"`
    Details []GoogleErrorInfo  `json:"details"`
}

type GoogleErrorInfo struct {
    Type     string            `json:"@type"`
    Reason   string            `json:"reason"`
    Domain   string            `json:"domain"`
    Metadata map[string]string `json:"metadata"`
}

func respondWithGoogleError(w http.ResponseWriter, code int, reason, domain string, metadata map[string]string) {
    errorResponse := GoogleErrorResponse{
        Error: GoogleErrorDetail{
            Code:    int32(code),
            Message: getMessageForCode(code),
            Status:  getStatusName(code),
            Details: []GoogleErrorInfo{
                {
                    Type:     "type.googleapis.com/google.rpc.ErrorInfo",
                    Reason:   reason,
                    Domain:   domain,
                    Metadata: metadata,
                },
            },
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(errorResponse)
}

// Usage example
func handleValidationError(w http.ResponseWriter, field string) {
    metadata := map[string]string{
        "field": field,
        "constraint": "required",
    }
    
    respondWithGoogleError(w, http.StatusBadRequest, 
        "INVALID_ARGUMENT", "bookstore.googleapis.com", metadata)
}
```

## Advanced Pagination

### Cursor-Based Pagination
Google APIs use cursor-based pagination for consistency:

```go
type PaginationCursor struct {
    LastId    string    `json:"last_id"`
    Timestamp time.Time `json:"timestamp"`
}

func encodeCursor(cursor PaginationCursor) string {
    data, _ := json.Marshal(cursor)
    return base64.URLEncoding.EncodeToString(data)
}

func decodeCursor(token string) (PaginationCursor, error) {
    var cursor PaginationCursor
    data, err := base64.URLEncoding.DecodeString(token)
    if err != nil {
        return cursor, err
    }
    err = json.Unmarshal(data, &cursor)
    return cursor, err
}

func (h *BookHandler) paginateBooks(pageSize int32, pageToken string) ([]Book, string, error) {
    var books []Book
    var nextToken string
    
    // Decode cursor
    var cursor PaginationCursor
    if pageToken != "" {
        var err error
        cursor, err = decodeCursor(pageToken)
        if err != nil {
            return nil, "", fmt.Errorf("invalid page token")
        }
    }
    
    // Query with cursor
    books, err := h.storage.ListAfterCursor(cursor, pageSize+1)
    if err != nil {
        return nil, "", err
    }
    
    // Generate next token if more results exist
    if len(books) > int(pageSize) {
        lastBook := books[pageSize-1]
        nextCursor := PaginationCursor{
            LastId:    lastBook.Name,
            Timestamp: lastBook.UpdateTime,
        }
        nextToken = encodeCursor(nextCursor)
        books = books[:pageSize]
    }
    
    return books, nextToken, nil
}
```

## Filtering and Ordering

### Filter Syntax
Google APIs use a standardized filter syntax:

```go
// Example filters:
// filter=author="John Doe"
// filter=publishDate>"2023-01-01" AND category="fiction"
// filter=title:search_term

type FilterParser struct{}

func (fp *FilterParser) ParseFilter(filterStr string) (*FilterExpression, error) {
    // Implementation of Google's filter syntax
    // Supports: =, !=, >, <, >=, <=, :, AND, OR, NOT
    // Field names, operators, and parentheses
}

func (h *BookHandler) applyFilters(books []Book, filter string) ([]Book, error) {
    if filter == "" {
        return books, nil
    }
    
    filterExpr, err := h.filterParser.ParseFilter(filter)
    if err != nil {
        return nil, fmt.Errorf("invalid filter: %v", err)
    }
    
    var filtered []Book
    for _, book := range books {
        if filterExpr.Matches(book) {
            filtered = append(filtered, book)
        }
    }
    
    return filtered, nil
}
```

### Ordering
```go
// Example order_by values:
// order_by=publishDate desc,title asc
// order_by=author,createTime desc

func (h *BookHandler) applyOrdering(books []Book, orderBy string) ([]Book, error) {
    if orderBy == "" {
        return books, nil
    }
    
    orders := parseOrderBy(orderBy)
    
    sort.Slice(books, func(i, j int) bool {
        for _, order := range orders {
            result := compareBooks(books[i], books[j], order.Field)
            if result != 0 {
                if order.Descending {
                    return result > 0
                }
                return result < 0
            }
        }
        return false
    })
    
    return books, nil
}
```

## Complete Google-Compliant Example

### Book Store API Implementation

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
    
    "github.com/gorilla/mux"
)

// Google-compliant resource definition
type Book struct {
    Name        string            `json:"name"`         // Required: publishers/123/books/456
    DisplayName string            `json:"display_name"` // Human-readable title
    Author      string            `json:"author"`
    ISBN        string            `json:"isbn"`
    Category    string            `json:"category"`
    CreateTime  time.Time         `json:"create_time"`
    UpdateTime  time.Time         `json:"update_time"`
    Labels      map[string]string `json:"labels,omitempty"`
}

type Publisher struct {
    Name        string    `json:"name"`         // Required: publishers/123
    DisplayName string    `json:"display_name"` // Human-readable name
    CreateTime  time.Time `json:"create_time"`
    UpdateTime  time.Time `json:"update_time"`
}

// Google-compliant request/response messages
type ListBooksRequest struct {
    Parent    string `json:"parent"`
    PageSize  int32  `json:"page_size"`
    PageToken string `json:"page_token"`
    Filter    string `json:"filter"`
    OrderBy   string `json:"order_by"`
}

type ListBooksResponse struct {
    Books         []Book `json:"books"`
    NextPageToken string `json:"next_page_token"`
    TotalSize     int32  `json:"total_size,omitempty"`
}

type CreateBookRequest struct {
    Parent string `json:"parent"`
    BookId string `json:"book_id,omitempty"`
    Book   Book   `json:"book"`
}

// Google-compliant API handler
type BookstoreHandler struct {
    storage BookStorage
}

func (h *BookstoreHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
    parent := mux.Vars(r)["parent"]
    pageSize := getIntParam(r, "page_size", 50)
    pageToken := r.URL.Query().Get("page_token")
    filter := r.URL.Query().Get("filter")
    orderBy := r.URL.Query().Get("order_by")
    
    if pageSize > 1000 {
        pageSize = 1000
    }
    
    books, nextToken, totalSize, err := h.storage.ListBooks(
        parent, pageSize, pageToken, filter, orderBy)
    if err != nil {
        respondWithGoogleError(w, http.StatusInternalServerError, 
            "INTERNAL", "bookstore.googleapis.com", nil)
        return
    }
    
    response := ListBooksResponse{
        Books:         books,
        NextPageToken: nextToken,
        TotalSize:     totalSize,
    }
    
    respondWithJSON(w, http.StatusOK, response)
}

func (h *BookstoreHandler) GetBook(w http.ResponseWriter, r *http.Request) {
    name := fmt.Sprintf("publishers/%s/books/%s", 
        mux.Vars(r)["publisher"], mux.Vars(r)["book"])
    
    book, err := h.storage.GetBook(name)
    if err != nil {
        if isNotFound(err) {
            respondWithGoogleError(w, http.StatusNotFound, 
                "NOT_FOUND", "bookstore.googleapis.com", 
                map[string]string{"resource": name})
            return
        }
        respondWithGoogleError(w, http.StatusInternalServerError, 
            "INTERNAL", "bookstore.googleapis.com", nil)
        return
    }
    
    respondWithJSON(w, http.StatusOK, book)
}

func (h *BookstoreHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
    parent := fmt.Sprintf("publishers/%s", mux.Vars(r)["publisher"])
    
    var req CreateBookRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithGoogleError(w, http.StatusBadRequest, 
            "INVALID_ARGUMENT", "bookstore.googleapis.com", 
            map[string]string{"field": "body"})
        return
    }
    
    req.Parent = parent
    
    // Validate required fields
    if req.Book.DisplayName == "" {
        respondWithGoogleError(w, http.StatusBadRequest, 
            "INVALID_ARGUMENT", "bookstore.googleapis.com", 
            map[string]string{"field": "book.display_name"})
        return
    }
    
    // Generate resource name
    bookId := req.BookId
    if bookId == "" {
        bookId = generateBookId()
    }
    
    req.Book.Name = fmt.Sprintf("%s/books/%s", parent, bookId)
    req.Book.CreateTime = time.Now()
    req.Book.UpdateTime = time.Now()
    
    createdBook, err := h.storage.CreateBook(req.Book)
    if err != nil {
        if isAlreadyExists(err) {
            respondWithGoogleError(w, http.StatusConflict, 
                "ALREADY_EXISTS", "bookstore.googleapis.com", 
                map[string]string{"resource": req.Book.Name})
            return
        }
        respondWithGoogleError(w, http.StatusInternalServerError, 
            "INTERNAL", "bookstore.googleapis.com", nil)
        return
    }
    
    w.Header().Set("Location", fmt.Sprintf("/v1/%s", createdBook.Name))
    respondWithJSON(w, http.StatusCreated, createdBook)
}

// Router setup with Google-compliant URL patterns
func setupRoutes() *mux.Router {
    router := mux.NewRouter()
    handler := &BookstoreHandler{storage: NewBookStorage()}
    
    // Google-style URL patterns
    api := router.PathPrefix("/v1").Subrouter()
    
    // Publisher routes
    api.HandleFunc("/publishers", handler.ListPublishers).Methods("GET")
    api.HandleFunc("/publishers", handler.CreatePublisher).Methods("POST")
    api.HandleFunc("/publishers/{publisher}", handler.GetPublisher).Methods("GET")
    api.HandleFunc("/publishers/{publisher}", handler.UpdatePublisher).Methods("PATCH")
    api.HandleFunc("/publishers/{publisher}", handler.DeletePublisher).Methods("DELETE")
    
    // Book routes (nested under publishers)
    api.HandleFunc("/publishers/{publisher}/books", handler.ListBooks).Methods("GET")
    api.HandleFunc("/publishers/{publisher}/books", handler.CreateBook).Methods("POST")
    api.HandleFunc("/publishers/{publisher}/books/{book}", handler.GetBook).Methods("GET")
    api.HandleFunc("/publishers/{publisher}/books/{book}", handler.UpdateBook).Methods("PATCH")
    api.HandleFunc("/publishers/{publisher}/books/{book}", handler.DeleteBook).Methods("DELETE")
    
    return router
}

func main() {
    router := setupRoutes()
    
    fmt.Println("🚀 Google-Compliant Bookstore API")
    fmt.Println("Server starting on :8088")
    fmt.Println("\nGoogle API patterns implemented:")
    fmt.Println("✅ Resource-oriented design")
    fmt.Println("✅ Standard method naming")
    fmt.Println("✅ Proper resource names")
    fmt.Println("✅ Advanced pagination")
    fmt.Println("✅ Standardized errors")
    fmt.Println("✅ Filtering and ordering")
    
    log.Fatal(http.ListenAndServe(":8088", router))
}
```

## Testing Google Compliance

### Test Resource Names
```bash
# ✅ Correct: Google-style resource names
curl http://localhost:8088/v1/publishers/oreilly/books/learning-go

# ❌ Incorrect: Non-hierarchical names  
curl http://localhost:8088/v1/books/123
```

### Test Pagination
```bash
# Google-style pagination
curl "http://localhost:8088/v1/publishers/oreilly/books?page_size=10&page_token=abc123"
```

### Test Filtering
```bash
# Google-style filtering
curl "http://localhost:8088/v1/publishers/oreilly/books?filter=category=\"programming\"&order_by=createTime desc"
```

## Key Takeaways

1. **Resource Names**: Use hierarchical, globally unique resource names
2. **Standard Methods**: Follow Get, List, Create, Update, Delete patterns
3. **Consistent Naming**: Use camelCase for collections, proper field naming
4. **Advanced Features**: Implement pagination, filtering, and ordering
5. **Error Handling**: Use structured Google-style error responses
6. **Parent-Child Relationships**: Properly model resource hierarchies

## Next Steps

In the next lesson, we'll explore request handling and routing patterns that build upon these Google standards, implementing advanced middleware and validation techniques.

## Practice Exercises

1. Convert the task API from Lesson 5 to use Google standards
2. Implement proper resource naming for a blog API
3. Add Google-style pagination to any existing API
4. Create a hierarchical resource structure (users > projects > tasks)
5. Implement Google-compliant error handling