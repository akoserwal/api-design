package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// User with HATEOAS links
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Links Links  `json:"_links"`
}

// Order represents a user's order
type Order struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	Total  float64 `json:"total"`
	Status string `json:"status"`
	Links  Links  `json:"_links"`
}

// Links represents hypermedia links
type Links map[string]Link

// Link represents a single hypermedia link
type Link struct {
	Href   string `json:"href"`
	Method string `json:"method,omitempty"`
	Type   string `json:"type,omitempty"`
}

// CollectionResponse represents a collection with links
type CollectionResponse struct {
	Data  interface{} `json:"data"`
	Links Links       `json:"_links"`
	Meta  Meta        `json:"meta"`
}

// Meta contains collection metadata
type Meta struct {
	Total int `json:"total"`
	Count int `json:"count"`
}

// Sample data
var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com"},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
}

var orders = []Order{
	{ID: 1, UserID: 1, Total: 99.99, Status: "pending"},
	{ID: 2, UserID: 1, Total: 149.99, Status: "completed"},
	{ID: 3, UserID: 2, Total: 79.99, Status: "shipped"},
}

func addUserLinks(user User, baseURL string) User {
	user.Links = Links{
		"self": {
			Href:   fmt.Sprintf("%s/users/%d", baseURL, user.ID),
			Method: "GET",
		},
		"edit": {
			Href:   fmt.Sprintf("%s/users/%d", baseURL, user.ID),
			Method: "PUT",
			Type:   "application/json",
		},
		"delete": {
			Href:   fmt.Sprintf("%s/users/%d", baseURL, user.ID),
			Method: "DELETE",
		},
		"orders": {
			Href:   fmt.Sprintf("%s/users/%d/orders", baseURL, user.ID),
			Method: "GET",
		},
	}
	return user
}

func addOrderLinks(order Order, baseURL string) Order {
	order.Links = Links{
		"self": {
			Href:   fmt.Sprintf("%s/orders/%d", baseURL, order.ID),
			Method: "GET",
		},
		"user": {
			Href:   fmt.Sprintf("%s/users/%d", baseURL, order.UserID),
			Method: "GET",
		},
	}

	// State-dependent links based on order status
	switch order.Status {
	case "pending":
		order.Links["cancel"] = Link{
			Href:   fmt.Sprintf("%s/orders/%d/cancel", baseURL, order.ID),
			Method: "POST",
		}
		order.Links["pay"] = Link{
			Href:   fmt.Sprintf("%s/orders/%d/pay", baseURL, order.ID),
			Method: "POST",
		}
	case "completed":
		order.Links["ship"] = Link{
			Href:   fmt.Sprintf("%s/orders/%d/ship", baseURL, order.ID),
			Method: "POST",
		}
	case "shipped":
		order.Links["track"] = Link{
			Href:   fmt.Sprintf("%s/orders/%d/tracking", baseURL, order.ID),
			Method: "GET",
		}
	}

	return order
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// API Root - Level 3 HATEOAS Entry Point
func apiRootHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)
	
	root := map[string]interface{}{
		"message": "Welcome to the HATEOAS API Demo",
		"version": "1.0.0",
		"_links": Links{
			"self": {
				Href: baseURL + "/",
			},
			"users": {
				Href:   baseURL + "/users",
				Method: "GET",
			},
			"orders": {
				Href:   baseURL + "/orders",
				Method: "GET",
			},
			"documentation": {
				Href: baseURL + "/docs",
				Type: "text/html",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(root)
}

// Get all users with HATEOAS
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)
	
	// Add links to each user
	usersWithLinks := make([]User, len(users))
	for i, user := range users {
		usersWithLinks[i] = addUserLinks(user, baseURL)
	}

	response := CollectionResponse{
		Data: usersWithLinks,
		Links: Links{
			"self": {
				Href: baseURL + "/users",
			},
			"create": {
				Href:   baseURL + "/users",
				Method: "POST",
				Type:   "application/json",
			},
		},
		Meta: Meta{
			Total: len(users),
			Count: len(users),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get specific user with HATEOAS
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid user ID",
		})
		return
	}

	baseURL := getBaseURL(r)

	for _, user := range users {
		if user.ID == userID {
			userWithLinks := addUserLinks(user, baseURL)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userWithLinks)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "User not found",
		"_links": Links{
			"users": {
				Href: baseURL + "/users",
			},
		},
	})
}

// Get user's orders with HATEOAS
func getUserOrdersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid user ID",
		})
		return
	}

	baseURL := getBaseURL(r)

	// Check if user exists
	userExists := false
	for _, user := range users {
		if user.ID == userID {
			userExists = true
			break
		}
	}

	if !userExists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	// Get user's orders
	userOrders := []Order{}
	for _, order := range orders {
		if order.UserID == userID {
			userOrders = append(userOrders, addOrderLinks(order, baseURL))
		}
	}

	response := CollectionResponse{
		Data: userOrders,
		Links: Links{
			"self": {
				Href: fmt.Sprintf("%s/users/%d/orders", baseURL, userID),
			},
			"user": {
				Href: fmt.Sprintf("%s/users/%d", baseURL, userID),
			},
			"create": {
				Href:   fmt.Sprintf("%s/users/%d/orders", baseURL, userID),
				Method: "POST",
				Type:   "application/json",
			},
		},
		Meta: Meta{
			Total: len(userOrders),
			Count: len(userOrders),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get all orders with HATEOAS
func getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)
	
	ordersWithLinks := make([]Order, len(orders))
	for i, order := range orders {
		ordersWithLinks[i] = addOrderLinks(order, baseURL)
	}

	response := CollectionResponse{
		Data: ordersWithLinks,
		Links: Links{
			"self": {
				Href: baseURL + "/orders",
			},
		},
		Meta: Meta{
			Total: len(orders),
			Count: len(orders),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get specific order with HATEOAS
func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid order ID",
		})
		return
	}

	baseURL := getBaseURL(r)

	for _, order := range orders {
		if order.ID == orderID {
			orderWithLinks := addOrderLinks(order, baseURL)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(orderWithLinks)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "Order not found",
		"_links": Links{
			"orders": {
				Href: baseURL + "/orders",
			},
		},
	})
}

// Order state transition examples
func cancelOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	baseURL := getBaseURL(r)

	for i, order := range orders {
		if order.ID == orderID {
			if order.Status != "pending" {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Order cannot be cancelled",
					"current_status": order.Status,
					"_links": Links{
						"order": {
							Href: fmt.Sprintf("%s/orders/%d", baseURL, orderID),
						},
					},
				})
				return
			}
			
			orders[i].Status = "cancelled"
			updatedOrder := addOrderLinks(orders[i], baseURL)
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(updatedOrder)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Order not found",
	})
}

// Documentation endpoint
func docsHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)
	
	docs := map[string]interface{}{
		"title": "HATEOAS API Documentation",
		"description": "This API demonstrates Level 3 REST (HATEOAS) principles",
		"features": []string{
			"Hypermedia controls for navigation",
			"State-dependent actions",
			"Self-descriptive messages",
			"Discoverable API structure",
		},
		"entry_point": baseURL + "/",
		"examples": map[string]interface{}{
			"start_here": "GET " + baseURL + "/",
			"browse_users": "GET " + baseURL + "/users",
			"user_orders": "GET " + baseURL + "/users/1/orders",
			"order_details": "GET " + baseURL + "/orders/1",
		},
		"hypermedia_features": map[string]interface{}{
			"navigation": "Follow _links to navigate the API",
			"state_transitions": "Available actions depend on resource state",
			"discoverability": "No need to hardcode URLs in clients",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func main() {
	router := mux.NewRouter()

	// Level 3 HATEOAS endpoints
	router.HandleFunc("/", apiRootHandler).Methods("GET")
	router.HandleFunc("/users", getUsersHandler).Methods("GET")
	router.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	router.HandleFunc("/users/{id}/orders", getUserOrdersHandler).Methods("GET")
	router.HandleFunc("/orders", getOrdersHandler).Methods("GET")
	router.HandleFunc("/orders/{id}", getOrderHandler).Methods("GET")
	router.HandleFunc("/orders/{id}/cancel", cancelOrderHandler).Methods("POST")
	router.HandleFunc("/docs", docsHandler).Methods("GET")

	fmt.Println("HATEOAS API Demo Server")
	fmt.Println("======================")
	fmt.Println("Server starting on :8081")
	fmt.Println("\nStart exploring at: http://localhost:8081/")
	fmt.Println("Documentation: http://localhost:8081/docs")
	fmt.Println("\nExample workflow:")
	fmt.Println("1. GET http://localhost:8081/")
	fmt.Println("2. Follow 'users' link")
	fmt.Println("3. Follow 'self' link for a specific user")
	fmt.Println("4. Follow 'orders' link to see user's orders")
	fmt.Println("5. Try cancelling a pending order")

	log.Fatal(http.ListenAndServe(":8081", router))
}