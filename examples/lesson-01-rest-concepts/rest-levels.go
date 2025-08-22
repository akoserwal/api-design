package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// User represents a user in our system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// In-memory storage
var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com"},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
}

// Richardson Maturity Model Level 0: The Swamp of POX
// Single endpoint, POST for everything, RPC-style
func level0Handler(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	action, ok := request["action"].(string)
	if !ok {
		http.Error(w, "Action required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch action {
	case "getUser":
		userID := int(request["userId"].(float64))
		for _, user := range users {
			if user.ID == userID {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"data":   user,
				})
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"message": "User not found",
		})

	case "getUsers":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   users,
		})

	case "createUser":
		name := request["name"].(string)
		email := request["email"].(string)
		newUser := User{
			ID:    len(users) + 1,
			Name:  name,
			Email: email,
		}
		users = append(users, newUser)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   newUser,
		})

	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"message": "Unknown action",
		})
	}
}

// Richardson Maturity Model Level 1: Resources
// Multiple endpoints, but still using POST for everything
func level1GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   users,
	})
}

func level1GetUser(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	json.NewDecoder(r.Body).Decode(&request)
	
	userID := int(request["userId"].(float64))
	
	w.Header().Set("Content-Type", "application/json")
	for _, user := range users {
		if user.ID == userID {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data":   user,
			})
			return
		}
	}
	
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"message": "User not found",
	})
}

func level1CreateUser(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	json.NewDecoder(r.Body).Decode(&request)
	
	name := request["name"].(string)
	email := request["email"].(string)
	
	newUser := User{
		ID:    len(users) + 1,
		Name:  name,
		Email: email,
	}
	users = append(users, newUser)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   newUser,
	})
}

// Richardson Maturity Model Level 2: HTTP Verbs
// Proper use of HTTP methods and status codes
func level2GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

func level2GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid user ID",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	for _, user := range users {
		if user.ID == userID {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "User not found",
	})
}

func level2CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON",
		})
		return
	}

	user.ID = len(users) + 1
	users = append(users, user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func level2UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid user ID",
		})
		return
	}

	var updatedUser User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	for i, user := range users {
		if user.ID == userID {
			updatedUser.ID = userID
			users[i] = updatedUser
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(updatedUser)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "User not found",
	})
}

func level2DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid user ID",
		})
		return
	}

	for i, user := range users {
		if user.ID == userID {
			users = append(users[:i], users[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "User not found",
	})
}

func main() {
	router := mux.NewRouter()

	// Level 0: Single endpoint, POST for everything
	fmt.Println("Richardson Maturity Model Demonstration")
	fmt.Println("=====================================")
	
	router.HandleFunc("/level0", level0Handler).Methods("POST")
	
	// Level 1: Multiple resources, still using POST
	router.HandleFunc("/level1/users", level1GetUsers).Methods("POST")
	router.HandleFunc("/level1/user", level1GetUser).Methods("POST")
	router.HandleFunc("/level1/user/create", level1CreateUser).Methods("POST")
	
	// Level 2: Proper HTTP verbs and status codes
	router.HandleFunc("/level2/users", level2GetUsers).Methods("GET")
	router.HandleFunc("/level2/users", level2CreateUser).Methods("POST")
	router.HandleFunc("/level2/users/{id}", level2GetUser).Methods("GET")
	router.HandleFunc("/level2/users/{id}", level2UpdateUser).Methods("PUT")
	router.HandleFunc("/level2/users/{id}", level2DeleteUser).Methods("DELETE")

	// Add demonstration endpoint
	router.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		demo := map[string]interface{}{
			"message": "Richardson Maturity Model Demo",
			"levels": map[string]interface{}{
				"level0": map[string]interface{}{
					"description": "Single endpoint, POST for everything",
					"endpoint": "/level0",
					"example": map[string]interface{}{
						"method": "POST",
						"body": map[string]interface{}{
							"action": "getUsers",
						},
					},
				},
				"level1": map[string]interface{}{
					"description": "Multiple resources, still using POST",
					"endpoints": []string{"/level1/users", "/level1/user", "/level1/user/create"},
				},
				"level2": map[string]interface{}{
					"description": "Proper HTTP verbs and status codes",
					"endpoints": map[string]string{
						"GET /level2/users": "Get all users",
						"POST /level2/users": "Create user",
						"GET /level2/users/{id}": "Get specific user",
						"PUT /level2/users/{id}": "Update user",
						"DELETE /level2/users/{id}": "Delete user",
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(demo)
	}).Methods("GET")

	fmt.Println("\nServer starting on :8080")
	fmt.Println("Visit http://localhost:8080/demo for API documentation")
	fmt.Println("\nTest Level 0 (POST /level0):")
	fmt.Println(`curl -X POST http://localhost:8080/level0 -d '{"action":"getUsers"}'`)
	fmt.Println("\nTest Level 2 (GET /level2/users):")
	fmt.Println(`curl http://localhost:8080/level2/users`)
	
	log.Fatal(http.ListenAndServe(":8080", router))
}