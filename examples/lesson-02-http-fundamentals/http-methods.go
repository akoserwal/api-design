package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Book represents a book resource
type Book struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	ISBN        string    `json:"isbn,omitempty"`
	Pages       int       `json:"pages,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// In-memory storage
var books = []Book{
	{
		ID:        1,
		Title:     "Go Programming",
		Author:    "John Doe",
		ISBN:      "978-0123456789",
		Pages:     300,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
	{
		ID:        2,
		Title:     "REST APIs",
		Author:    "Jane Smith",
		ISBN:      "978-0987654321",
		Pages:     250,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	},
}

var nextID = 3

// GET - Retrieve resources (Safe, Idempotent)
func getBooksHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[GET] %s - Safe: Yes, Idempotent: Yes\n", r.URL.Path)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	
	response := map[string]interface{}{
		"books": books,
		"count": len(books),
		"meta": map[string]interface{}{
			"method": "GET",
			"safe": true,
			"idempotent": true,
			"description": "GET is safe (doesn't modify server state) and idempotent (multiple calls have same effect)",
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

func getBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid book ID",
		})
		return
	}

	fmt.Printf("[GET] %s - Safe: Yes, Idempotent: Yes\n", r.URL.Path)

	for _, book := range books {
		if book.ID == id {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=600")
			w.Header().Set("ETag", fmt.Sprintf(`"book-%d-%d"`, book.ID, book.UpdatedAt.Unix()))
			
			response := map[string]interface{}{
				"book": book,
				"meta": map[string]interface{}{
					"method": "GET",
					"safe": true,
					"idempotent": true,
				},
			}
			
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Book not found",
	})
}

// POST - Create new resources (Not Safe, Not Idempotent)
func createBookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[POST] %s - Safe: No, Idempotent: No\n", r.URL.Path)
	
	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}

	// Validation
	if book.Title == "" || book.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Title and author are required",
		})
		return
	}

	// Set server-managed fields
	book.ID = nextID
	nextID++
	book.CreatedAt = time.Now()
	book.UpdatedAt = time.Now()

	books = append(books, book)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/books/%d", book.ID))
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"book": book,
		"meta": map[string]interface{}{
			"method": "POST",
			"safe": false,
			"idempotent": false,
			"description": "POST creates new resources. Not safe (modifies state) and not idempotent (multiple calls create multiple resources)",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// PUT - Replace entire resource (Not Safe, Idempotent)
func updateBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid book ID",
		})
		return
	}

	fmt.Printf("[PUT] %s - Safe: No, Idempotent: Yes\n", r.URL.Path)

	var updatedBook Book
	if err := json.NewDecoder(r.Body).Decode(&updatedBook); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}

	// Validation
	if updatedBook.Title == "" || updatedBook.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Title and author are required",
		})
		return
	}

	// Find and replace the book
	for i, book := range books {
		if book.ID == id {
			// Preserve server-managed fields
			updatedBook.ID = id
			updatedBook.CreatedAt = book.CreatedAt
			updatedBook.UpdatedAt = time.Now()
			
			books[i] = updatedBook

			w.Header().Set("Content-Type", "application/json")
			
			response := map[string]interface{}{
				"book": updatedBook,
				"meta": map[string]interface{}{
					"method": "PUT",
					"safe": false,
					"idempotent": true,
					"description": "PUT replaces entire resource. Not safe (modifies state) but idempotent (multiple calls have same effect)",
				},
			}
			
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Book not found",
	})
}

// PATCH - Partial update (Not Safe, May be Idempotent)
func patchBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid book ID",
		})
		return
	}

	fmt.Printf("[PATCH] %s - Safe: No, Idempotent: Depends on implementation\n", r.URL.Path)

	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}

	// Find and partially update the book
	for i, book := range books {
		if book.ID == id {
			// Apply partial updates
			if title, ok := patch["title"].(string); ok {
				book.Title = title
			}
			if author, ok := patch["author"].(string); ok {
				book.Author = author
			}
			if isbn, ok := patch["isbn"].(string); ok {
				book.ISBN = isbn
			}
			if pages, ok := patch["pages"].(float64); ok {
				book.Pages = int(pages)
			}
			
			book.UpdatedAt = time.Now()
			books[i] = book

			w.Header().Set("Content-Type", "application/json")
			
			response := map[string]interface{}{
				"book": book,
				"updated_fields": getUpdatedFields(patch),
				"meta": map[string]interface{}{
					"method": "PATCH",
					"safe": false,
					"idempotent": "depends on implementation",
					"description": "PATCH updates specific fields. Not safe, and idempotency depends on the operations performed",
				},
			}
			
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Book not found",
	})
}

// DELETE - Remove resource (Not Safe, Idempotent)
func deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid book ID",
		})
		return
	}

	fmt.Printf("[DELETE] %s - Safe: No, Idempotent: Yes\n", r.URL.Path)

	for i, book := range books {
		if book.ID == id {
			// Remove the book
			books = append(books[:i], books[i+1:]...)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // Using 200 instead of 204 to show response
			
			response := map[string]interface{}{
				"message": "Book deleted successfully",
				"deleted_book": book,
				"meta": map[string]interface{}{
					"method": "DELETE",
					"safe": false,
					"idempotent": true,
					"description": "DELETE removes resources. Not safe (modifies state) but idempotent (multiple calls have same effect)",
				},
			}
			
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// DELETE is idempotent - even if resource doesn't exist, we return success
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "Book not found",
		"meta": map[string]interface{}{
			"note": "DELETE is idempotent - the result is the same whether the resource exists or not",
		},
	})
}

// HEAD - Like GET but only headers (Safe, Idempotent)
func headBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("[HEAD] %s - Safe: Yes, Idempotent: Yes\n", r.URL.Path)

	for _, book := range books {
		if book.ID == id {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "0")
			w.Header().Set("ETag", fmt.Sprintf(`"book-%d-%d"`, book.ID, book.UpdatedAt.Unix()))
			w.Header().Set("Last-Modified", book.UpdatedAt.Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

// OPTIONS - Show allowed methods (Safe, Idempotent)
func optionsBookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[OPTIONS] %s - Safe: Yes, Idempotent: Yes\n", r.URL.Path)
	
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
}

// Demonstration endpoint
func methodsInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"title": "HTTP Methods Demonstration",
		"methods": map[string]interface{}{
			"GET": map[string]interface{}{
				"description": "Retrieve resources",
				"safe": true,
				"idempotent": true,
				"example": "GET /books",
			},
			"POST": map[string]interface{}{
				"description": "Create new resources",
				"safe": false,
				"idempotent": false,
				"example": "POST /books",
			},
			"PUT": map[string]interface{}{
				"description": "Replace entire resource",
				"safe": false,
				"idempotent": true,
				"example": "PUT /books/1",
			},
			"PATCH": map[string]interface{}{
				"description": "Partial update",
				"safe": false,
				"idempotent": "depends",
				"example": "PATCH /books/1",
			},
			"DELETE": map[string]interface{}{
				"description": "Remove resource",
				"safe": false,
				"idempotent": true,
				"example": "DELETE /books/1",
			},
			"HEAD": map[string]interface{}{
				"description": "Get headers only",
				"safe": true,
				"idempotent": true,
				"example": "HEAD /books/1",
			},
			"OPTIONS": map[string]interface{}{
				"description": "Get allowed methods",
				"safe": true,
				"idempotent": true,
				"example": "OPTIONS /books",
			},
		},
		"test_commands": []string{
			`curl http://localhost:8083/books`,
			`curl -X POST http://localhost:8083/books -d '{"title":"New Book","author":"Author"}' -H "Content-Type: application/json"`,
			`curl -X PUT http://localhost:8083/books/1 -d '{"title":"Updated Book","author":"Updated Author"}' -H "Content-Type: application/json"`,
			`curl -X PATCH http://localhost:8083/books/1 -d '{"title":"Patched Title"}' -H "Content-Type: application/json"`,
			`curl -X DELETE http://localhost:8083/books/1`,
			`curl -I http://localhost:8083/books/2`,
			`curl -X OPTIONS http://localhost:8083/books`,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func getUpdatedFields(patch map[string]interface{}) []string {
	var fields []string
	for key := range patch {
		fields = append(fields, key)
	}
	return fields
}

func main() {
	router := mux.NewRouter()

	// Demo endpoint
	router.HandleFunc("/", methodsInfoHandler).Methods("GET")

	// Book endpoints demonstrating different HTTP methods
	router.HandleFunc("/books", getBooksHandler).Methods("GET")
	router.HandleFunc("/books", createBookHandler).Methods("POST")
	router.HandleFunc("/books", optionsBookHandler).Methods("OPTIONS")
	
	router.HandleFunc("/books/{id}", getBookHandler).Methods("GET")
	router.HandleFunc("/books/{id}", updateBookHandler).Methods("PUT")
	router.HandleFunc("/books/{id}", patchBookHandler).Methods("PATCH")
	router.HandleFunc("/books/{id}", deleteBookHandler).Methods("DELETE")
	router.HandleFunc("/books/{id}", headBookHandler).Methods("HEAD")
	router.HandleFunc("/books/{id}", optionsBookHandler).Methods("OPTIONS")

	fmt.Println("HTTP Methods Demonstration Server")
	fmt.Println("================================")
	fmt.Println("Server starting on :8083")
	fmt.Println("\nHTTP Methods and their properties:")
	fmt.Println("GET    - Safe: ✓, Idempotent: ✓")
	fmt.Println("POST   - Safe: ✗, Idempotent: ✗")
	fmt.Println("PUT    - Safe: ✗, Idempotent: ✓")
	fmt.Println("PATCH  - Safe: ✗, Idempotent: depends")
	fmt.Println("DELETE - Safe: ✗, Idempotent: ✓")
	fmt.Println("HEAD   - Safe: ✓, Idempotent: ✓")
	fmt.Println("OPTIONS- Safe: ✓, Idempotent: ✓")
	fmt.Println("\nVisit http://localhost:8083/ for test commands")

	log.Fatal(http.ListenAndServe(":8083", router))
}