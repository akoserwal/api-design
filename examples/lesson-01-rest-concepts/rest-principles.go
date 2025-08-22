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

// Product represents a product in our catalog
type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	InStock     bool      `json:"in_stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Sample data
var products = []Product{
	{
		ID:          1,
		Name:        "Laptop",
		Description: "High-performance laptop",
		Price:       999.99,
		Category:    "Electronics",
		InStock:     true,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
	},
	{
		ID:          2,
		Name:        "Mouse",
		Description: "Wireless mouse",
		Price:       29.99,
		Category:    "Electronics",
		InStock:     true,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-2 * time.Hour),
	},
}

// Demonstration of REST Principle 1: Client-Server Architecture
// Server manages data and business logic, client handles presentation

func principlesHandler(w http.ResponseWriter, r *http.Request) {
	principles := map[string]interface{}{
		"title": "REST Architectural Principles Demonstration",
		"principles": map[string]interface{}{
			"1_client_server": "Clear separation between client and server responsibilities",
			"2_stateless": "Each request contains all information needed to process it",
			"3_cacheable": "Responses explicitly indicate if they can be cached",
			"4_uniform_interface": "Consistent way to interact with resources",
			"5_layered_system": "Architecture can have multiple layers",
			"6_code_on_demand": "Server can send executable code (optional)",
		},
		"examples": map[string]string{
			"stateless": "GET /products - no session state required",
			"cacheable": "GET /products/1 - includes cache headers",
			"uniform_interface": "Standard HTTP methods for all resources",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(principles)
}

// Demonstration of REST Principle 2: Statelessness
// Each request contains all information needed to process it
func getProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering (all state in request)
	category := r.URL.Query().Get("category")
	inStockParam := r.URL.Query().Get("in_stock")
	priceMinParam := r.URL.Query().Get("price_min")
	priceMaxParam := r.URL.Query().Get("price_max")

	var filteredProducts []Product

	for _, product := range products {
		// Apply filters based on request parameters
		if category != "" && product.Category != category {
			continue
		}

		if inStockParam != "" {
			inStock, _ := strconv.ParseBool(inStockParam)
			if product.InStock != inStock {
				continue
			}
		}

		if priceMinParam != "" {
			priceMin, _ := strconv.ParseFloat(priceMinParam, 64)
			if product.Price < priceMin {
				continue
			}
		}

		if priceMaxParam != "" {
			priceMax, _ := strconv.ParseFloat(priceMaxParam, 64)
			if product.Price > priceMax {
				continue
			}
		}

		filteredProducts = append(filteredProducts, product)
	}

	// Demonstration of REST Principle 3: Cacheability
	// Set cache headers to indicate this response can be cached
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	w.Header().Set("ETag", generateETag(filteredProducts))
	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))

	// Return filtered results
	response := map[string]interface{}{
		"products": filteredProducts,
		"count":    len(filteredProducts),
		"filters_applied": map[string]interface{}{
			"category":  category,
			"in_stock":  inStockParam,
			"price_min": priceMinParam,
			"price_max": priceMaxParam,
		},
		"demonstration": "This response demonstrates statelessness - all filtering logic is based on request parameters",
	}

	json.NewEncoder(w).Encode(response)
}

// Demonstration of conditional requests and caching
func getProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid product ID",
		})
		return
	}

	// Find product
	var product *Product
	for _, p := range products {
		if p.ID == productID {
			product = &p
			break
		}
	}

	if product == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Product not found",
		})
		return
	}

	// Generate ETag based on product data and last modified time
	etag := fmt.Sprintf(`"product-%d-%d"`, product.ID, product.UpdatedAt.Unix())
	lastModified := product.UpdatedAt.Format(http.TimeFormat)

	// Check If-None-Match header (ETag-based conditional request)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Check If-Modified-Since header (time-based conditional request)
	if ifModifiedSince := r.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
		if t, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
			if product.UpdatedAt.Before(t) || product.UpdatedAt.Equal(t) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// Set cache headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=600") // Cache for 10 minutes
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", lastModified)

	// Return product with caching demonstration info
	response := map[string]interface{}{
		"product": product,
		"cache_info": map[string]interface{}{
			"etag":          etag,
			"last_modified": lastModified,
			"cache_control": "public, max-age=600",
			"demonstration": "This response includes proper cache headers for client-side caching",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Demonstration of uniform interface
func createProductHandler(w http.ResponseWriter, r *http.Request) {
	var newProduct Product
	if err := json.NewDecoder(r.Body).Decode(&newProduct); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}

	// Validate required fields (business logic on server)
	if newProduct.Name == "" || newProduct.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Name and positive price are required",
		})
		return
	}

	// Set server-managed fields
	newProduct.ID = len(products) + 1
	newProduct.CreatedAt = time.Now()
	newProduct.UpdatedAt = time.Now()

	products = append(products, newProduct)

	// Return created resource with location header
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/products/%d", newProduct.ID))
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"product": newProduct,
		"demonstration": "Uniform interface - POST creates resource, returns 201 with Location header",
	}

	json.NewEncoder(w).Encode(response)
}

// Demonstration of layered system
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Add layer identifier header
		w.Header().Set("X-Layer", "Logging-Middleware")
		
		next.ServeHTTP(w, r)
		
		log.Printf("[%s] %s %s - %v", 
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method, 
			r.URL.Path, 
			time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers (another layer)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("X-Layer", "CORS-Middleware")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Utility function to generate ETag
func generateETag(products []Product) string {
	if len(products) == 0 {
		return `"empty"`
	}
	
	// Simple ETag generation based on product count and last update
	hash := len(products)
	for _, p := range products {
		hash += int(p.UpdatedAt.Unix())
	}
	
	return fmt.Sprintf(`"products-%d"`, hash)
}

// Demonstration endpoint showing all principles
func allPrinciplesHandler(w http.ResponseWriter, r *http.Request) {
	demo := map[string]interface{}{
		"demonstration": "All REST Principles in Action",
		"current_request": map[string]interface{}{
			"method": r.Method,
			"url": r.URL.String(),
			"headers": map[string]string{
				"user_agent": r.Header.Get("User-Agent"),
				"accept": r.Header.Get("Accept"),
			},
			"stateless": "This request contains all information needed to process it",
		},
		"response_headers": map[string]string{
			"content_type": "application/json",
			"cache_control": "public, max-age=60",
			"x_layer": "Multiple middleware layers processed this request",
		},
		"principles_demonstrated": map[string]interface{}{
			"client_server": "Server manages data, client handles presentation",
			"stateless": "No session state stored on server",
			"cacheable": "Response includes cache headers",
			"uniform_interface": "Standard HTTP methods and status codes",
			"layered_system": "Multiple middleware layers (logging, CORS)",
		},
		"try_these_requests": []string{
			"GET /products - see stateless filtering",
			"GET /products/1 - see caching headers",
			"POST /products - see uniform interface",
			"GET /products?category=Electronics - see stateless parameters",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(demo)
}

func main() {
	router := mux.NewRouter()

	// Apply middleware layers (demonstrating layered system)
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// REST principles demonstration endpoints
	router.HandleFunc("/", principlesHandler).Methods("GET")
	router.HandleFunc("/demo", allPrinciplesHandler).Methods("GET")
	router.HandleFunc("/products", getProductsHandler).Methods("GET")
	router.HandleFunc("/products", createProductHandler).Methods("POST")
	router.HandleFunc("/products/{id}", getProductHandler).Methods("GET")

	fmt.Println("REST Principles Demonstration Server")
	fmt.Println("===================================")
	fmt.Println("Server starting on :8082")
	fmt.Println("\nEndpoints demonstrating REST principles:")
	fmt.Println("GET  http://localhost:8082/        - Principles overview")
	fmt.Println("GET  http://localhost:8082/demo    - All principles demo")
	fmt.Println("GET  http://localhost:8082/products - Stateless filtering")
	fmt.Println("GET  http://localhost:8082/products/1 - Caching headers")
	fmt.Println("POST http://localhost:8082/products - Uniform interface")
	fmt.Println("\nTry these examples:")
	fmt.Println("curl http://localhost:8082/products")
	fmt.Println("curl http://localhost:8082/products?category=Electronics")
	fmt.Println("curl -I http://localhost:8082/products/1")
	fmt.Println(`curl -X POST http://localhost:8082/products -d '{"name":"Keyboard","price":49.99,"category":"Electronics"}' -H "Content-Type: application/json"`)

	log.Fatal(http.ListenAndServe(":8082", router))
}