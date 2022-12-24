package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

// User represents a user resource
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// JWTMiddleware is a middleware that validates JWTs
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the JWT from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse the JWT
		token, err := jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Return the secret key
			return []byte("my-secret-key"), nil
		})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Validate the claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Set the claims in the context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	})
}

// ValidateAPIRequest is a middleware that validates API requests
// against the OpenAPI specification
// func ValidateAPIRequest(spec *spec.Swagger, next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Extract the operation from the specification
// 		op, ok := spec.Paths.Paths[spec.Paths.Paths.Paths(r.URL.Path)][r.Method]
// 		if !ok {
// 			w.WriteHeader(http.StatusNotFound)
// 			return
// 		}

// 		// Validate the request parameters and body
// 		res := validate.NewSpecValidator(spec, strfmt.Default).ValidateRequest(r)
// 		if res.IsValid() {
// 			// Set the validated request in the context
// 			ctx := context.WithValue(r.Context(), "request", r)
// 			next.ServeHTTP(w, r.WithContext(ctx))
// 		} else {
// 			// Return the validation errors
// 			w.WriteHeader(http.StatusBadRequest)
// 			json.NewEncoder(w).Encode(res.Errors)
// 		}
// 	})
// }

var db *sql.DB

func main() {
	// Connect to PostgreSQL database
	var err error
	connStr := "user=postgres password=postgres dbname=postgres sslmode=disable host=host.docker.internal"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// spec, err := spec.Load("./openapi/openapi.json")
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='users');").Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		fmt.Println("Users table exists")
	} else {
		fmt.Println("Users table does not exist")
		// Create the users table
		_, err = db.Exec(`
			CREATE TABLE users (
				id serial PRIMARY KEY,
				name text NOT NULL,
				email text UNIQUE NOT NULL,
				password text NOT NULL
			);
		`)
		if err != nil {
			panic(err)
		}

		fmt.Println("Users table created successfully")
	}

	// Initialize router
	router := mux.NewRouter()

	// Add the ValidateAPIRequest middleware to the router
	// router.Use(ValidateAPIRequest(spec, router))

	// Set up routes
	router.HandleFunc("/users", getUsers).Methods("GET")
	router.HandleFunc("/users", createUser).Methods("POST")
	router.Handle("/users/{id}", JWTMiddleware(http.HandlerFunc(getUser))).Methods("GET")
	router.HandleFunc("/users/{id}", updateUser).Methods("PUT")
	router.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")
	router.HandleFunc("/signup", signUp).Methods("POST")
	// Add the signin route
	router.HandleFunc("/signin", signIn).Methods("POST")

	// Start server
	fmt.Print("Starting Server")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func signIn(w http.ResponseWriter, r *http.Request) {
	// Extract the request parameters
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Check if the user exists
	var user User
	err := db.QueryRow("SELECT id, name, email, password FROM users WHERE email = $1", email).Scan(&user.ID, &user.Name, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Email or password is incorrect")
			return
		}
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Validate the password
	if user.Password != password {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Email or password is incorrect")
		return
	}

	// Generate a JWT and return it
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
	}).SignedString([]byte("my-secret-key"))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, token)
}

// Add the signup route
func signUp(w http.ResponseWriter, r *http.Request) {

	// Extract the request parameters
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Check if the user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", email).Scan(&count)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if count > 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Email already exists")
		return
	}

	// Insert the new user into the database
	result, err := db.Exec("INSERT INTO users (name, email, password) VALUES ($1, $2, $3)", name, email, password)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get the ID of the inserted user
	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Generate a JWT and return it
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    id,
		"name":  name,
		"email": email,
	}).SignedString([]byte("my-secret-key"))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, token)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	// Query users from database
	fmt.Println("Getting getUsers")
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Scan rows into slice of users
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return users as JSON
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func createUser(w http.ResponseWriter, r *http.Request) {
	// Decode request body into user struct
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert user into database
	stmt, err := db.Prepare("INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	err = stmt.QueryRow(u.Name, u.Email, u.Password).Scan(&u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return created user as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL path
	vars := mux.Vars(r)
	id := vars["id"]

	// Query user from database
	var u User
	err := db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", id).Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return user as JSON
	if err := json.NewEncoder(w).Encode(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL path
	vars := mux.Vars(r)
	id := vars["id"]

	// Decode request body into user struct
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update user in database
	stmt, err := db.Prepare("UPDATE users SET name = $1, email = $2, password = $3 WHERE id = $4")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(u.Name, u.Email, u.Password, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated user as JSON
	if err := json.NewEncoder(w).Encode(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL path
	vars := mux.Vars(r)
	id := vars["id"]

	// Delete user from database
	stmt, err := db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	res, err := stmt.Exec(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if user was deleted
	count, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Return success message
	w.WriteHeader(http.StatusNoContent)
}

// TEST CASES

func TestCreateUser(t *testing.T) {
	// Set up test server
	router := mux.NewRouter()
	router.HandleFunc("/users", createUser).Methods("POST")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Send POST request to create user
	u := User{Name: "Test User", Email: "test@example.com", Password: "testpassword"}
	b, _ := json.Marshal(u)
	res, err := http.Post(ts.URL+"/users", "application/json", bytes.NewBuffer(b))
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d; got %d", http.StatusCreated, res.StatusCode)
	}

	// Check if user was created
	var created User
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Error(err)
	}
	if created.Name != u.Name || created.Email != u.Email || created.Password != u.Password {
		t.Errorf("expected %v; got %v", u, created)
	}

	// Clean up
	stmt, err := db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(created.ID); err != nil {
		t.Error(err)
	}
}

func TestGetUser(t *testing.T) {
	// Set up test server
	router := mux.NewRouter()
	router.HandleFunc("/users/{id}", getUser).Methods("GET")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create user to retrieve
	stmt, err := db.Prepare("INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	var id int
	if err := stmt.QueryRow("Test User", "test@example.com", "testpassword").Scan(&id); err != nil {
		t.Error(err)
	}

	// Send GET request to retrieve user
	res, err := http.Get(ts.URL + "/users/" + strconv.Itoa(id))
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %d; got %d", http.StatusOK, res.StatusCode)
	}

	// Check if user was retrieved
	var retrieved User
	if err := json.NewDecoder(res.Body).Decode(&retrieved); err != nil {
		t.Error(err)
	}
	if retrieved.ID != id || retrieved.Name != "Test User" || retrieved.Email != "test@example.com" || retrieved.Password != "testpassword" {
		t.Errorf("expected %v; got %v", User{ID: id, Name: "Test User", Email: "test@example.com", Password: "testpassword"}, retrieved)
	}

	// Clean up
	stmt, err = db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(id); err != nil {
		t.Error(err)
	}
}

func TestDeleteUser(t *testing.T) {
	// Set up test server
	router := mux.NewRouter()
	router.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create user to delete
	stmt, err := db.Prepare("INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	var id int
	if err := stmt.QueryRow("Test User", "test@example.com", "testpassword").Scan(&id); err != nil {
		t.Error(err)
	}

	// Send DELETE request to delete user
	req, err := http.NewRequest("DELETE", ts.URL+"/users/"+strconv.Itoa(id), nil)
	if err != nil {
		t.Error(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d; got %d", http.StatusNoContent, res.StatusCode)
	}

	// Check if user was deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", id).Scan(&count)
	if err != nil {
		t.Error(err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows; got %d", count)
	}
}

func TestUpdateUser(t *testing.T) {
	// Set up test server
	router := mux.NewRouter()
	router.HandleFunc("/users/{id}", updateUser).Methods("PUT")
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create user to update
	stmt, err := db.Prepare("INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	var id int
	if err := stmt.QueryRow("Test User", "test@example.com", "testpassword").Scan(&id); err != nil {
		t.Error(err)
	}

	// Send PUT request to update user
	u := User{Name: "Updated User", Email: "updated@example.com", Password: "updatedpassword"}
	b, _ := json.Marshal(u)
	req, err := http.NewRequest("PUT", ts.URL+"/users/"+strconv.Itoa(id), bytes.NewBuffer(b))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %d; got %d", http.StatusOK, res.StatusCode)
	}

	// Check if user was updated
	var updated User
	if err := json.NewDecoder(res.Body).Decode(&updated); err != nil {
		t.Error(err)
	}
	if updated.ID != id || updated.Name != u.Name || updated.Email != u.Email || updated.Password != u.Password {
		t.Errorf("expected %v; got %v", u, updated)
	}

	// Clean up
	stmt, err = db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(id); err != nil {
		t.Error(err)
	}

}
