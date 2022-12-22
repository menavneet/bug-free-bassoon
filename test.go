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
	stmt, err := db.Prepare("DELETE FROM users WHERE id = $1")
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
	stmt, err := db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(id); err != nil {
		t.Error(err)
	}

}

